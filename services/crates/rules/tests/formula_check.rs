//! The checker against the shipment schema the server publishes: every case the browser's
//! language service and the old engine held it to, and the language's deliberate changes.

use trenova_rules::formula::check;
use trenova_rules::formula::check::Analysis;
use trenova_rules::formula::check::Code;
use trenova_rules::formula::check::Diagnostic;
use trenova_rules::formula::check::Expect;
use trenova_rules::formula::check::Fix;
use trenova_rules::formula::check::InputType;
use trenova_rules::formula::check::RateTable;
use trenova_rules::formula::check::Schema;
use trenova_rules::formula::check::SchemaInput;
use trenova_rules::formula::check::Severity;
use trenova_rules::formula::check::Type;

const DESCRIBE: &str = include_str!(
    "../../../tms/internal/modules/rating/app/formula/testdata/shipment.describe.json"
);

fn schema(inputs: &[(&str, &str)]) -> Schema {
    let mut schema: Schema = serde_json::from_str(DESCRIBE).expect("the schema parses");
    schema.inputs = inputs
        .iter()
        .map(|(name, kind)| SchemaInput {
            name: (*name).to_owned(),
            kind: Some((*kind).to_owned()),
        })
        .collect();
    schema
}

fn default_schema() -> Schema {
    schema(&[("fuelPct", "Number"), ("hazmatFee", "Number")])
}

fn analyse(source: &str) -> Analysis {
    check::check(source, Expect::Amount, Some(&default_schema()), None)
}

fn codes(source: &str) -> Vec<String> {
    analyse(source)
        .diagnostics
        .iter()
        .map(|diagnostic| {
            format!(
                "{}:{}",
                diagnostic.severity.as_str(),
                diagnostic.code.as_str()
            )
        })
        .collect()
}

fn messages(source: &str) -> Vec<String> {
    analyse(source)
        .diagnostics
        .iter()
        .map(|diagnostic| diagnostic.message.english())
        .collect()
}

fn inserts(diagnostic: &Diagnostic) -> Vec<String> {
    diagnostic
        .fixes
        .iter()
        .flat_map(|fix| match fix {
            Fix::Replace { changes, .. } => changes.iter().map(|c| c.insert.clone()).collect(),
            Fix::Declare { .. } => Vec::new(),
        })
        .collect()
}

fn fixes_for(source: &str, code: Code) -> Vec<String> {
    analyse(source)
        .diagnostics
        .iter()
        .filter(|diagnostic| diagnostic.code == code)
        .flat_map(inserts)
        .collect()
}

#[test]
fn accepts_a_sound_formula() {
    assert!(
        codes(
            "let fuel = baseRate * totalDistance * (fuelPct / 100);\nbaseRate * totalDistance + fuel"
        )
        .is_empty()
    );
}

#[test]
fn an_unknown_name_gets_a_suggestion_and_a_declare_fix() {
    let analysis = analyse("baseRaet * weekendFee");
    let found: Vec<Code> = analysis.diagnostics.iter().map(|d| d.code).collect();
    assert_eq!(found, [Code::UnknownVariable, Code::UnknownVariable]);
    assert_eq!(inserts(&analysis.diagnostics[0]), ["baseRate"]);
    assert!(matches!(
        analysis.diagnostics[1].fixes.last(),
        Some(Fix::Declare { name, input_type: InputType::Number, .. }) if name == "weekendFee"
    ));
    assert_eq!(
        analysis.diagnostics[0].message.english(),
        "`baseRaet` isn't a shipment field or one of this template's inputs. Did you mean `baseRate`?"
    );
    assert_eq!(
        analysis.diagnostics[1].message.english(),
        "`weekendFee` isn't a shipment field or one of this template's inputs. Declare it as an input, or pick a field from the reference."
    );
}

#[test]
fn now_is_refused_and_points_at_rating_time_with_a_fix() {
    let analysis = analyse("now().Year() == 2026 ? 10 : 0");
    assert_eq!(analysis.diagnostics.len(), 1);
    let diagnostic = &analysis.diagnostics[0];
    assert_eq!(diagnostic.code, Code::DisabledFunction);
    assert_eq!(diagnostic.severity, Severity::Error);
    assert_eq!(diagnostic.span.start, 0);
    assert_eq!(
        diagnostic.message.english(),
        "`now()` isn't available in formulas: a formula must price a shipment the same way every time it runs. Use `ratingTime`, the date the contract terms are read at"
    );
    let Some(Fix::Replace { label, changes }) = diagnostic.fixes.first() else {
        panic!("a replace fix");
    };
    assert_eq!(label.english(), "Change to ratingTime");
    assert_eq!(
        (
            changes[0].span.start,
            changes[0].span.end,
            changes[0].insert.as_str()
        ),
        (0, 5, "ratingTime")
    );
}

#[test]
fn the_functions_formulas_leave_out_are_refused_without_a_suggestion() {
    for source in [
        "len(toJSON(totalStops))",
        "fromJSON(\"1\")",
        "len(repeat(\"x\", 3))",
    ] {
        let analysis = analyse(source);
        let found: Vec<Code> = analysis.diagnostics.iter().map(|d| d.code).collect();
        assert_eq!(found, [Code::DisabledFunction], "{source}");
        assert!(
            analysis.diagnostics[0]
                .message
                .english()
                .contains("isn't available in formulas")
        );
        assert!(analysis.diagnostics[0].fixes.is_empty());
    }
}

#[test]
fn a_flag_name_is_declared_as_a_boolean() {
    let analysis = analyse("isTeam ? 1 : 0");
    assert!(matches!(
        analysis.diagnostics[0].fixes.last(),
        Some(Fix::Declare {
            input_type: InputType::Boolean,
            ..
        })
    ));
}

#[test]
fn an_unknown_field_under_a_known_root_gets_a_suggestion() {
    let analysis = analyse("customer.nmae");
    assert_eq!(analysis.diagnostics[0].code, Code::UnknownField);
    assert!(
        analysis.diagnostics[0]
            .message
            .english()
            .contains("customer.name")
    );
    assert_eq!(inserts(&analysis.diagnostics[0]), ["customer.name"]);
    assert_eq!(
        analysis.diagnostics[0].message.english(),
        "`customer` has no field `nmae`. Did you mean `customer.name`?"
    );
}

#[test]
fn an_unknown_field_is_reported_at_the_first_part_that_is_unknown() {
    let analysis = analyse("len(customer.nmae.first)");
    assert_eq!(analysis.diagnostics.len(), 1);
    assert_eq!(analysis.diagnostics[0].span.start, 13);
}

#[test]
fn arity_comes_from_the_catalog() {
    assert_eq!(codes("pow(2)"), ["error:arity"]);
    assert!(codes("max(1, 2, 3)").is_empty());
    assert_eq!(codes("clamp(1, 2)"), ["error:arity"]);
    let arity = |source: &str| -> Vec<String> {
        analyse(source)
            .diagnostics
            .iter()
            .filter(|d| d.code == Code::Arity)
            .map(|d| d.message.english())
            .collect()
    };
    assert_eq!(
        arity("round(1, 2, 3)"),
        ["round(x, decimals?) takes 1 to 2 arguments; 3 were given"]
    );
    assert_eq!(
        arity("max()"),
        ["max(...values) takes at least 1 argument; 0 were given"]
    );
    assert!(arity("count([1, 2])").is_empty());
    assert_eq!(
        arity("reverse([1], [2])"),
        ["reverse(list) takes 1 argument; 2 were given"]
    );
}

#[test]
fn arithmetic_on_a_flag_suggests_the_ternary() {
    let analysis = analyse("hasHazmat * hazmatFee");
    assert_eq!(analysis.diagnostics[0].code, Code::Type);
    assert!(
        analysis.diagnostics[0]
            .message
            .english()
            .contains("(hasHazmat ? amount : 0)")
    );
    assert_eq!(inserts(&analysis.diagnostics[0]), ["(hasHazmat ? 1 : 0)"]);
}

#[test]
fn a_number_is_not_a_test() {
    assert!(messages("totalStops ? 1 : 0")[0].contains("totalStops > 0"));
}

#[test]
fn a_nullable_field_read_without_a_guard_warns_once_with_the_coalesce_guard() {
    let analysis = analyse("weight * 2 + weight");
    let found: Vec<Code> = analysis.diagnostics.iter().map(|d| d.code).collect();
    assert_eq!(found, [Code::Nullable]);
    assert_eq!(analysis.diagnostics[0].severity, Severity::Warning);
    assert_eq!(inserts(&analysis.diagnostics[0]), ["coalesce(weight, 0)"]);
    assert_eq!(
        analysis.diagnostics[0].message.english(),
        "`weight` is empty on some shipments, and the formula would fail to rate them. Give it a fallback: coalesce(weight, 0)"
    );
}

#[test]
fn a_nullable_field_under_an_object_is_guarded_with_optional_chaining() {
    let analysis = analyse("shipmentType.code == \"LTL\" ? 1 : 0");
    assert_eq!(analysis.diagnostics[0].code, Code::Nullable);
    assert_eq!(
        inserts(&analysis.diagnostics[0]),
        ["(shipmentType?.code ?? \"\")"]
    );
    assert!(codes("shipmentType?.code == \"LTL\" ? 1 : 0").is_empty());
}

#[test]
fn a_nullable_date_falls_back_to_the_rating_time() {
    let analysis = analyse("pickupDate.Hour()");
    assert_eq!(
        inserts(&analysis.diagnostics[0]),
        ["coalesce(pickupDate, ratingTime)"]
    );
}

#[test]
fn guards_are_coalesce_a_nil_test_and_optional_chaining() {
    assert!(codes("(weight ?? 0) * 2").is_empty());
    assert!(codes("coalesce(weight, 0) * 2").is_empty());
    assert!(codes("weight == nil ? 0 : weight * 2").is_empty());
    assert!(codes("weight != nil && weight > 10 ? 1 : 0").is_empty());
    assert!(codes("if weight == nil { 0 } else { weight * 2 }").is_empty());
    assert_eq!(codes("weight * 2 + (weight ?? 0)"), ["warning:nullable"]);
}

#[test]
fn the_old_engines_nullable_cases_hold() {
    let warned = |source: &str| -> Vec<String> {
        analyse(source)
            .diagnostics
            .iter()
            .filter(|d| d.code == Code::Nullable)
            .filter_map(|d| d.name.clone())
            .collect()
    };
    assert_eq!(warned("weight * 2"), ["weight"]);
    assert_eq!(warned("weight > 100 ? 50 : 0"), ["weight"]);
    assert!(warned("coalesce(weight, 0) * 2").is_empty());
    assert!(warned("1 + 1").is_empty());
    assert!(warned("1 +* 2").is_empty());
}

#[test]
fn let_problems_are_reported() {
    assert!(codes("let a = 1;\nlet a = 2;\na").contains(&String::from("error:let-duplicate")));
    assert!(codes("let a = 1;\nbaseRate").contains(&String::from("warning:let-unused")));
    assert!(codes("let a = 1;\nlet b = a;").contains(&String::from("error:let-last")));
    assert!(codes("b + 1;\nlet b = 2;\nb").contains(&String::from("error:let-before-definition")));
}

#[test]
fn ending_on_a_let_offers_its_value_after_the_semicolon() {
    let analysis = analyse("let a = 1;\nlet b = a;");
    let last = analysis
        .diagnostics
        .iter()
        .find(|d| d.code == Code::LetLast)
        .expect("let-last");
    let Some(Fix::Replace { changes, .. }) = last.fixes.first() else {
        panic!("a fix");
    };
    assert_eq!(changes[0].span.start, 21);
    assert_eq!(changes[0].insert, "\nb");
}

#[test]
fn a_let_may_not_hide_a_field_an_input_or_a_function() {
    let shadows = |source: &str| -> Vec<String> {
        analyse(source)
            .diagnostics
            .iter()
            .filter(|d| d.code == Code::LetShadows)
            .map(|d| d.message.english())
            .collect()
    };
    assert_eq!(
        shadows("let totalWeight = 1; totalWeight"),
        [
            "`totalWeight` is already a shipment field, and a 'let' can't redeclare it. Use another name"
        ]
    );
    assert_eq!(
        shadows("let fuelPct = 1; fuelPct"),
        [
            "`fuelPct` is one of this template's inputs, and a 'let' can't redeclare it. Use another name"
        ]
    );
    assert_eq!(
        shadows("let count = 1; count"),
        ["`count` is the name of a formula function, and a 'let' can't take it. Use another name"]
    );
    assert_eq!(shadows("let customer = 1; customer").len(), 1);
}

#[test]
fn scopes_reuse_names_side_by_side_but_not_inside() {
    assert!(codes("(let a = 2; a * 3) + (let a = 4; a)").is_empty());
    assert_eq!(codes("let x = 1; (let x = 2; x)"), ["error:let-duplicate"]);
    assert_eq!(codes("(let y = 1; 2)"), ["warning:let-unused"]);
}

#[test]
fn the_formula_must_come_to_an_amount() {
    assert_eq!(codes("hasHazmat"), ["error:result-type"]);
    assert_eq!(codes("customer.name"), ["error:result-type"]);
    assert_eq!(codes("nil"), ["error:result-type"]);
    assert_eq!(codes("ratingTime - ratingTime"), ["error:result-type"]);
    assert_eq!(
        messages("hasHazmat"),
        [
            "The formula must come to an amount, but this comes to true/false. Write hasHazmat ? amount : 0"
        ]
    );
    assert_eq!(
        messages("[1]"),
        ["The formula must come to an amount, but this comes to a list"]
    );
}

#[test]
fn a_yes_no_rule_may_come_to_true_false_or_a_number() {
    let rule = |source: &str| {
        check::check(source, Expect::Predicate, Some(&default_schema()), None)
            .diagnostics
            .iter()
            .map(|d| d.code.as_str())
            .collect::<Vec<_>>()
    };
    assert!(rule("hasHazmat").is_empty());
    assert!(rule("totalWeight > 1000").is_empty());
    assert!(rule("totalStops").is_empty());
    assert_eq!(rule("customer.name"), ["result-type"]);
}

#[test]
fn a_value_line_before_the_last_is_thrown_away() {
    assert_eq!(codes("1;\n2"), ["warning:discarded"]);
}

#[test]
fn the_analysis_lists_what_the_formula_reads() {
    let analysis = analyse("let fuel = fuelPct;\nbaseRate * totalDistance + fuel + round(1)");
    assert_eq!(analysis.reads, ["fuelPct", "baseRate", "totalDistance"]);
    assert_eq!(analysis.function_calls, ["round"]);
    assert_eq!(analysis.result_type, Type::Number);
}

#[test]
fn builtins_and_the_pipe_are_known() {
    assert!(codes("len(customer.name) | abs()").is_empty());
    assert!(codes("len(customer.name)").is_empty());
}

#[test]
fn while_the_schema_loads_no_name_is_unknown() {
    let analysis = check::check(
        "baseRate * totalDistance + lookup(\"zones\", 1)",
        Expect::Amount,
        None,
        None,
    );
    assert!(analysis.pending);
    assert!(analysis.diagnostics.is_empty());
    let broken = check::check("baseRate *", Expect::Amount, None, None);
    let found: Vec<Code> = broken.diagnostics.iter().map(|d| d.code).collect();
    assert_eq!(found, [Code::Syntax]);
}

#[test]
fn while_the_schema_loads_calls_are_still_checked() {
    let analysis = check::check("pow(2) + rond(1)", Expect::Amount, None, None);
    let found: Vec<Code> = analysis.diagnostics.iter().map(|d| d.code).collect();
    assert_eq!(found, [Code::Arity, Code::UnknownFunction]);
}

#[test]
fn a_predicate_given_to_a_sum_maps_the_list() {
    assert_eq!(
        fixes_for("sum(stops, .weight)", Code::Type),
        ["sum(map(stops, .weight))"]
    );
    assert_eq!(codes("sum([1, 2], # * 2)"), ["error:type"]);
    assert!(codes("sum(map([1, 2], # * 2))").is_empty());
    assert!(codes("count(stops, max(.a, .b) > 1) * 2").is_empty());
}

#[test]
fn modulo_takes_whole_numbers() {
    assert_eq!(
        fixes_for("totalDistance % 100", Code::Type),
        ["int(totalDistance)"]
    );
    assert_eq!(codes("baseRate % 2"), ["error:type"]);
    assert_eq!(codes("10 / 2 % 3"), ["error:type"]);
    assert_eq!(codes("5.5 % 2"), ["error:type"]);
    assert!(codes("totalStops % 2").is_empty());
    assert!(codes("(10 + 2) * 3 % 5").is_empty());
    assert!(codes("int(totalWeight) % 7").is_empty());
    assert!(codes("let x = 10; x % 3").is_empty());
    assert!(codes("len([1, 2, 3]) % 2").is_empty());
    assert!(codes("-totalStops % 2").is_empty());
}

#[test]
fn bits_ranges_and_take_need_whole_numbers() {
    assert_eq!(codes("bitand(1.5, 1)"), ["error:type"]);
    assert_eq!(
        messages("bitand(baseRate, 1)"),
        [
            "bitand takes whole numbers, but `baseRate` can hold a fraction. Use int(baseRate) to take its whole part"
        ]
    );
    assert_eq!(codes("len(1.5..3)"), ["error:type"]);
    assert_eq!(codes("len(take([1, 2], 1.5))"), ["error:type"]);
    assert!(codes("bitnot(1) + 3").is_empty());
    assert!(codes("len(1..totalStops)").is_empty());
}

#[test]
fn equality_between_kinds_is_refused_wherever_the_value_came_from() {
    for source in [
        "status == 1 ? 1 : 0",
        "totalWeight == \"heavy\" ? 1 : 0",
        "hasHazmat == 1 ? 1 : 0",
        "\"a\" == 1 ? 1 : 0",
        "origin.zip == 75201 ? 1 : 0",
    ] {
        assert_eq!(codes(source), ["error:type"], "{source}");
    }
    assert_eq!(
        messages("origin.zip == 75201 ? 1 : 0"),
        [
            "`origin.zip` is text and this part is a number; the engine compares only values of the same kind"
        ]
    );
    assert!(codes("totalStops == 2.0 ? 1 : 0").is_empty());
}

#[test]
fn comparisons_take_values_of_one_kind() {
    assert_eq!(
        messages("(1 < 2) < 3 ? 1 : 0"),
        ["'<' compares numbers, but this part is true/false. Use it as the test itself."]
    );
    assert!(codes("0 < totalWeight <= 5000 ? 1 : 0").is_empty());
    assert!(codes("0 < totalWeight < 2000 <= 3000 ? 1 : 0").is_empty());
    assert_eq!(
        codes("0 < customer.name < 5 ? 1 : 0"),
        ["error:type", "error:type"]
    );
    assert!(codes("pickupDate < ratingTime ? 1 : 0").len() == 1);
}

#[test]
fn text_and_numbers_do_not_add() {
    assert_eq!(codes("\"a\" + 1"), ["error:type"]);
    assert_eq!(
        codes("laneCode + \"x\""),
        ["error:unknown-variable", "error:result-type"]
    );
    assert_eq!(
        codes("customer.name + customer.code"),
        ["error:result-type"]
    );
}

#[test]
fn dates_move_by_durations() {
    assert_eq!(codes("ratingTime + 1 > ratingTime ? 1 : 0"), ["error:type"]);
    assert_eq!(codes("ratingTime - 1 > ratingTime ? 1 : 0"), ["error:type"]);
    assert!(codes("ratingTime + duration(\"1h\") > ratingTime ? 1 : 0").is_empty());
    assert!(codes("ratingTime - ratingTime > duration(\"1h\") ? 1 : 0").is_empty());
    assert!(codes("(ratingTime - ratingTime) / duration(\"1h\")").is_empty());
    assert!(codes("(ratingTime - ratingTime).Hours()").is_empty());
    assert_eq!(
        messages("ratingTime + 1 > ratingTime ? 1 : 0"),
        ["A date moves by a duration, not a number: write ratingTime + duration(\"24h\")"]
    );
    assert_eq!(
        codes("duration(\"1h\") + 2 > duration(\"1h\") ? 1 : 0"),
        ["error:type"]
    );
    assert_eq!(
        codes("ratingTime + ratingTime > ratingTime ? 1 : 0"),
        ["error:type"]
    );
}

#[test]
fn methods_exist_on_dates_and_durations_only() {
    assert!(codes("ratingTime.Year() == 2026 ? 1 : 0").is_empty());
    assert!(codes("ratingTime.Year() % 4").is_empty());
    assert_eq!(codes("ratingTime.Yaer()"), ["error:unknown-function"]);
    assert_eq!(
        fixes_for("ratingTime.Yaer()", Code::UnknownFunction),
        ["Year"]
    );
    assert_eq!(codes("ratingTime.Hours()"), ["error:unknown-function"]);
    assert_eq!(codes("totalWeight.Year()"), ["error:type"]);
    assert_eq!(codes("ratingTime.Year(1)"), ["error:arity"]);
}

#[test]
fn fields_are_not_read_from_values_that_have_none() {
    assert_eq!(codes("len(customer.name.first)"), ["error:type"]);
    assert_eq!(
        messages("len(customer.name.first)"),
        ["`customer.name` is text; it has no field `first`"]
    );
    assert_eq!(codes("totalWeight[0]"), ["error:type"]);
    assert!(codes("len(origin.zip[0:3])").is_empty());
}

#[test]
fn functions_take_values_of_the_kinds_they_need() {
    assert_eq!(codes("len(upper(5))"), ["error:type"]);
    assert_eq!(codes("len(5)"), ["error:type"]);
    assert_eq!(codes("round(customer.name)"), ["error:type"]);
    assert_eq!(codes("len(keys([1]))"), ["error:type"]);
    assert!(codes("max([1, 2], 3)").is_empty());
    assert!(codes("float(\"1.5\") + int(\"2\")").is_empty());
}

#[test]
fn a_name_that_is_a_value_is_not_called() {
    assert_eq!(
        messages("totalWeight(1)"),
        ["`totalWeight` is a value, not a function; it can't be called with ( )"]
    );
    assert_eq!(
        messages("rond(baseRate, 2)"),
        ["`rond` isn't a formula function. Did you mean `round`?"]
    );
    assert_eq!(codes("toBase64(\"a\")"), ["error:unknown-function"]);
    assert_eq!(codes("len(timezone(\"UTC\"))"), ["error:unknown-function"]);
}

#[test]
fn pointers_only_work_inside_predicates() {
    assert_eq!(codes("len(.weight)"), ["error:syntax"]);
    assert_eq!(
        messages("len(.weight)"),
        [
            "`.weight` reads a field of the current item, so it only works inside filter(), map() and the like"
        ]
    );
    assert_eq!(codes("# + 1"), ["error:syntax"]);
    assert_eq!(codes("len(map([1], #acc))"), ["error:syntax"]);
    assert_eq!(codes("len(map([1], #foo))"), ["error:syntax"]);
    assert!(codes("reduce([1, 2, 3], #acc + #, 0)").is_empty());
    assert!(codes("len(map([1, 2], #index))").is_empty());
    assert!(codes("len(filter([1, 2, 3], { let x = #; x > 1 }))").is_empty());
}

#[test]
fn rate_tables_are_named_in_quotes() {
    assert_eq!(
        codes("lookup(laneCode, 1)"),
        ["error:unknown-variable", "error:rate-table"]
    );
    let analysis = check::check(
        "lookup(laneCode, 1)",
        Expect::Amount,
        Some(&schema(&[("laneCode", "String")])),
        None,
    );
    assert_eq!(analysis.diagnostics[0].code, Code::RateTable);
    assert_eq!(
        analysis.diagnostics[0].message.english(),
        "lookup names its rate table in quotes: lookup(\"zone_rates\", …)"
    );
}

#[test]
fn rate_tables_are_checked_against_the_ones_that_exist() {
    let tables = [
        RateTable {
            name: String::from("fuel"),
            axes: 1,
            banded: false,
        },
        RateTable {
            name: String::from("grid"),
            axes: 2,
            banded: false,
        },
        RateTable {
            name: String::from("curve"),
            axes: 1,
            banded: true,
        },
    ];
    let table_messages = |source: &str| -> Vec<String> {
        check::check(
            source,
            Expect::Amount,
            Some(&default_schema()),
            Some(&tables),
        )
        .diagnostics
        .iter()
        .filter(|d| d.code == Code::RateTable)
        .map(|d| d.message.english())
        .collect()
    };
    assert!(table_messages("lookup(\"fuel\", 1) + lookup2(\"grid\", 1, 2)").is_empty());
    assert_eq!(
        table_messages("lookup(\"fule\", 1)"),
        ["There is no rate table named `fule`. Did you mean `fuel`?"]
    );
    assert_eq!(
        table_messages("lookup(\"grid\", 1)"),
        ["`grid` has two axes; read it with lookup2() or lookup2Or()"]
    );
    assert_eq!(
        table_messages("lookup2Or(\"fuel\", 1, 2, 0)"),
        ["`fuel` has one axis; read it with lookup() or lookupOr()"]
    );
    assert_eq!(
        table_messages("lookupInterp(\"fuel\", 1)"),
        ["`fuel` has no bands, so lookupInterp() can't read it"]
    );
    assert!(table_messages("deficitWeight(\"curve\", 1) + lookupInterp(\"curve\", 2)").is_empty());
    let analysis = check::check(
        "lookup(\"fule\", 1)",
        Expect::Amount,
        Some(&default_schema()),
        Some(&tables),
    );
    assert_eq!(
        (
            analysis.diagnostics[0].span.start,
            analysis.diagnostics[0].span.end
        ),
        (7, 13)
    );
    assert_eq!(inserts(&analysis.diagnostics[0]), ["\"fuel\""]);
}

#[test]
fn the_rate_tables_a_formula_reads_are_listed() {
    let tables = |source: &str| -> Vec<(String, &'static str)> {
        analyse(source)
            .lookup_tables
            .iter()
            .map(|table| (table.name.clone(), table.function))
            .collect()
    };
    assert_eq!(
        tables(
            "lookup(\"fuel\", 1) + lookupOr(\"lane\", 2, 0) + lookup2(\"class_rates\", 1, 2) + lookup2Or(\"zone_weight\", 1, 2, 0)"
        ),
        [
            (String::from("fuel"), "lookup"),
            (String::from("lane"), "lookupOr"),
            (String::from("class_rates"), "lookup2"),
            (String::from("zone_weight"), "lookup2Or"),
        ]
    );
    assert_eq!(
        tables("sum(lookup(\"fsc\", fuelPct) * 2, 25)"),
        [(String::from("fsc"), "lookup")]
    );
    assert_eq!(
        tables("lookupInterp(\"fuel_curve\", 1) + deficitWeight(\"cwt\", 2) / 100"),
        [
            (String::from("fuel_curve"), "lookupInterp"),
            (String::from("cwt"), "deficitWeight"),
        ]
    );
}

#[test]
fn item_fields_type_predicates_when_the_schema_publishes_them() {
    let mut with_items = default_schema();
    with_items.variables.push(
        serde_json::from_value(serde_json::json!({
            "name": "stops[].weight", "type": "number", "nullable": true
        }))
        .expect("a variable"),
    );
    with_items.variables.push(
        serde_json::from_value(serde_json::json!({"name": "stops[].type", "type": "string"}))
            .expect("a variable"),
    );
    let item = |source: &str| -> Vec<String> {
        check::check(source, Expect::Amount, Some(&with_items), None)
            .diagnostics
            .iter()
            .map(|d| d.message.english())
            .collect()
    };
    assert!(item("count(stops, .type == \"Delivery\")").is_empty());
    assert_eq!(
        item("count(stops, .tyep == \"Delivery\")"),
        ["An item of `stops` has no field `tyep`. Did you mean `type`?"]
    );
    assert_eq!(
        item("count(stops, .type > 1)"),
        ["'>' compares values of the same kind, but `.type` is text and this part is a number"]
    );
    assert_eq!(
        item("sum(map(stops, .weight))"),
        [
            "`.weight` is empty on some shipments, and the formula would fail to rate them. Give it a fallback: (.weight ?? 0)"
        ]
    );
    assert!(item("sum(map(stops, .weight ?? 0))").is_empty());
    assert!(item("sum(map(filter(stops, .type == \"Pickup\"), #.weight ?? 0))").is_empty());
    assert_eq!(item("first(stops).wieght ?? 0").len(), 1);
    assert!(codes("count(stops, .anything == 1)").is_empty());
}

#[test]
fn an_empty_formula_has_one_empty_diagnostic() {
    let analysis = analyse("   ");
    assert!(analysis.empty);
    let found: Vec<Code> = analysis.diagnostics.iter().map(|d| d.code).collect();
    assert_eq!(found, [Code::Empty]);
    assert!(!analysis.accepted());
}

#[test]
fn the_word_operators_are_not_unknown_names() {
    assert!(
        codes("(origin.state startsWith \"T\" and customer.name contains \"Acme\" and origin.zip matches \"^7\" and destination.state endsWith \"X\") ? 1 : 0")
            .is_empty()
    );
    assert!(codes("(hasHazmat and true) ? 1 : 0").is_empty());
}

#[test]
fn operators_the_schema_publishes_parse_and_check() {
    let schema: Schema = serde_json::from_value(serde_json::json!({
        "variables": [{"name": "code", "type": "string", "description": "", "category": "shipment",
                       "nullable": false, "computed": false}],
        "operators": ["soundsLike"],
    }))
    .expect("a schema");
    let analysis = check::check(
        "code soundsLike \"abc\" ? 1 : 0",
        Expect::Amount,
        Some(&schema),
        None,
    );
    assert!(analysis.diagnostics.is_empty());
}

#[test]
fn a_server_schema_knows_only_its_own_variables() {
    let schema: Schema = serde_json::from_value(serde_json::json!({
        "variables": [{"name": "newServerVariable", "type": "number"}],
    }))
    .expect("a schema");
    let analysis = check::check(
        "newServerVariable + totalDistance",
        Expect::Amount,
        Some(&schema),
        None,
    );
    let found: Vec<Code> = analysis.diagnostics.iter().map(|d| d.code).collect();
    assert_eq!(found, [Code::UnknownVariable]);
}

#[test]
fn let_bindings_are_known_wherever_they_are_read() {
    let none = schema(&[]);
    let lint = |source: &str| -> Vec<String> {
        check::check(source, Expect::Amount, Some(&none), None)
            .diagnostics
            .iter()
            .map(|d| d.message.english())
            .collect()
    };
    assert!(
        lint(
            "let linehaul = baseRate * totalDistance;\nlet fuel = linehaul * 0.18;\nlinehaul + fuel"
        )
        .is_empty()
    );
    let found = lint("let linehaul = baseRate;\nlinehaul + weekendFee");
    assert_eq!(found.len(), 1);
    assert!(found[0].contains("weekendFee"));
}

#[test]
fn the_studio_analysis_separates_reads_inputs_functions_and_lets() {
    let analysis = check::check(
        "let linehaul = baseRate * totalDistance;\nmax(linehaul * (1 + fuelSurchargePercent / 100), 250)",
        Expect::Amount,
        Some(&schema(&[("fuelSurchargePercent", "Number")])),
        None,
    );
    assert!(analysis.accepted());
    assert_eq!(analysis.schema_reads, ["baseRate", "totalDistance"]);
    assert_eq!(analysis.input_reads, ["fuelSurchargePercent"]);
    assert_eq!(analysis.function_calls, ["max"]);
    assert_eq!(
        analysis.reads,
        ["fuelSurchargePercent", "baseRate", "totalDistance"]
    );
    let names: Vec<&str> = analysis.bindings.iter().map(|b| b.name.as_str()).collect();
    assert_eq!(names, ["linehaul"]);
    assert_eq!(analysis.bindings[0].reads, 1);
    assert_eq!(analysis.bindings[0].ty, Type::Number);
}

#[test]
fn an_unknown_name_is_a_problem_the_studio_turns_into_an_input() {
    let analysis = check::check(
        "baseRate * weekendFee",
        Expect::Amount,
        Some(&schema(&[("fuelSurchargePercent", "Number")])),
        None,
    );
    assert_eq!(analysis.diagnostics.len(), 1);
    assert_eq!(analysis.diagnostics[0].code, Code::UnknownVariable);
    assert_eq!(analysis.diagnostics[0].name.as_deref(), Some("weekendFee"));
}

#[test]
fn flag_names_start_as_booleans_and_others_as_numbers() {
    for (name, expected) in [
        ("isWeekend", InputType::Boolean),
        ("hasLiftgate", InputType::Boolean),
        ("requiresEscort", InputType::Boolean),
        ("usesTolls", InputType::Boolean),
        ("weekendFee", InputType::Number),
        ("island", InputType::Number),
    ] {
        assert_eq!(check::infer_input_type(name), expected, "{name}");
    }
}

#[test]
fn suggestions_are_within_two_edits_or_a_case_change() {
    assert_eq!(
        check::nearest_known("baseRaet", ["totalWeight", "baseRate"]),
        Some("baseRate")
    );
    assert_eq!(
        check::nearest_known("baserate", ["BaseRate"]),
        Some("BaseRate")
    );
    assert_eq!(check::nearest_known("xyz", ["baseRate"]), None);
    assert_eq!(check::nearest_known("baseRate", ["baseRate"]), None);
}

#[test]
fn a_divide_by_a_literal_zero_is_a_warning() {
    assert_eq!(codes("totalWeight / 0"), ["warning:divide-by-zero"]);
    assert!(analyse("totalWeight / 0").accepted());
}

#[test]
fn text_operators_take_text() {
    assert_eq!(
        codes("totalWeight startsWith \"1\" ? 1 : 0"),
        ["error:type"]
    );
    assert_eq!(
        codes("[1, 2] contains 1 ? 1 : 0"),
        ["error:type", "error:type"]
    );
    assert_eq!(codes("1 in 5 ? 1 : 0"), ["error:type"]);
}

#[test]
fn logic_takes_true_false_tests() {
    assert_eq!(
        messages("(totalStops and hasHazmat) ? 1 : 0"),
        [
            "'and' joins two true/false tests, but `totalStops` is a number. Compare it, such as totalStops > 0"
        ]
    );
    assert_eq!(codes("!totalStops ? 1 : 0"), ["error:type"]);
    assert_eq!(codes("-customer.name"), ["error:type"]);
}

#[test]
fn checking_a_thousand_node_formula_is_fast() {
    let terms: Vec<String> = (0..90)
        .map(|i| format!("(totalWeight * {i} + coalesce(weight, {i})) * baseRate"))
        .collect();
    let source = format!("max({})", terms.join(", "));
    let start = std::time::Instant::now();
    let analysis = analyse(&source);
    let elapsed = start.elapsed();
    let mut nodes = 0;
    analysis.program.walk(&mut |_| nodes += 1);
    assert!(nodes > 800, "{nodes}");
    assert!(analysis.accepted(), "{:?}", analysis.diagnostics);
    assert!(elapsed.as_millis() < 250, "{elapsed:?}");
}
