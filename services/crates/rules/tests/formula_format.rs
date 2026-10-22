//! The formatter, held to every case the editor's `expr-format.ts` was.

use trenova_rules::formula::syntax;
use trenova_rules::formula::syntax::DEFAULT_WIDTH;
use trenova_rules::formula::syntax::FormatOutcome;
use trenova_rules::formula::syntax::FormatRefusal;

const LONG_LETS: &str = "let linehaul = mileRate * totalDistance; let weightCharge = max(0, totalWeight - weightFreeLbs) / 100 * weightRatePerCwt; let stopCharge = max(0, totalStops - stopsIncluded) * stopRate; let hazmatCharge = hasHazmat ? hazmatFee : 0; linehaul + weightCharge + stopCharge + hazmatCharge";

fn format(source: &str, width: usize) -> FormatOutcome {
    syntax::format(source, width, &[])
}

fn formatted(source: &str, width: usize) -> String {
    match format(source, width) {
        FormatOutcome::Formatted { text, .. } => text,
        FormatOutcome::Refused(reason) => panic!("{source}: refused: {reason:?}"),
    }
}

fn lines(lines: &[&str]) -> String {
    lines.join("\n")
}

#[test]
fn every_statement_goes_on_its_own_line() {
    assert_eq!(
        formatted(LONG_LETS, DEFAULT_WIDTH),
        lines(&[
            "let linehaul = mileRate * totalDistance;",
            "let weightCharge = max(0, totalWeight - weightFreeLbs) / 100 * weightRatePerCwt;",
            "let stopCharge = max(0, totalStops - stopsIncluded) * stopRate;",
            "let hazmatCharge = hasHazmat ? hazmatFee : 0;",
            "linehaul + weightCharge + stopCharge + hazmatCharge",
        ])
    );
}

#[test]
fn spacing_is_normalised_and_parentheses_and_literals_are_kept() {
    assert_eq!(
        formatted(
            "(baseRate*totalDistance)+(hasHazmat?hazmatFee:0)*1.50",
            DEFAULT_WIDTH
        ),
        "(baseRate * totalDistance) + (hasHazmat ? hazmatFee : 0) * 1.50"
    );
    assert_eq!(
        formatted(
            "serviceType.code==\"EXP\"  and  not   isWeekendPickup",
            DEFAULT_WIDTH
        ),
        "serviceType.code == \"EXP\" and not isWeekendPickup"
    );
}

#[test]
fn a_long_sum_breaks_one_term_per_line_with_the_operator_leading() {
    let source = "(mileRate * totalDistance) + (max(0, totalStops - stopsIncluded) * stopRate) + (hasHazmat ? hazmatFee : 0) + (isWeekendPickup ? weekendFee : 0)";
    assert_eq!(
        formatted(source, DEFAULT_WIDTH),
        lines(&[
            "(mileRate * totalDistance)",
            "  + (max(0, totalStops - stopsIncluded) * stopRate)",
            "  + (hasHazmat ? hazmatFee : 0)",
            "  + (isWeekendPickup ? weekendFee : 0)",
        ])
    );
}

#[test]
fn a_let_breaks_after_its_equals_when_the_value_then_fits() {
    let source = "let fuelSurcharge = max(0, coalesce(fuelPrice, fuelBaseline) - fuelBaseline) * fuelMultiplier; fuelSurcharge";
    assert_eq!(
        formatted(source, 80),
        lines(&[
            "let fuelSurcharge =",
            "  max(0, coalesce(fuelPrice, fuelBaseline) - fuelBaseline) * fuelMultiplier;",
            "fuelSurcharge",
        ])
    );
}

#[test]
fn a_ternary_chain_breaks_one_case_per_line() {
    let source = "totalWeight < breakOne ? rateOne * totalWeight : totalWeight < breakTwo ? rateTwo * totalWeight : rateThree * totalWeight";
    assert_eq!(
        formatted(source, 60),
        lines(&[
            "totalWeight < breakOne ? rateOne * totalWeight",
            "  : totalWeight < breakTwo ? rateTwo * totalWeight",
            "  : rateThree * totalWeight",
        ])
    );
}

#[test]
fn a_call_breaks_one_argument_per_line_when_nothing_shorter_fits() {
    let source = "lookup(accessorialTable, customer.code, serviceType.code, equipmentType.code, origin.state)";
    assert_eq!(
        formatted(source, 50),
        lines(&[
            "lookup(",
            "  accessorialTable,",
            "  customer.code,",
            "  serviceType.code,",
            "  equipmentType.code,",
            "  origin.state",
            ")",
        ])
    );
}

#[test]
fn a_pipe_an_infix_not_and_a_predicate_stay_as_written() {
    assert_eq!(
        formatted("totalWeight|round(2)", DEFAULT_WIDTH),
        "totalWeight | round(2)"
    );
    assert_eq!(
        formatted("serviceType.code not in [\"EXP\",\"TEAM\"]", DEFAULT_WIDTH),
        "serviceType.code not in [\"EXP\", \"TEAM\"]"
    );
    assert_eq!(
        formatted("count(commodities,{.hazmat})>0?hazmatFee:0", DEFAULT_WIDTH),
        "count(commodities, {.hazmat}) > 0 ? hazmatFee : 0"
    );
    assert_eq!(
        formatted("filter(stops, #.type == 'Delivery')", DEFAULT_WIDTH),
        "filter(stops, #.type == 'Delivery')"
    );
}

#[test]
fn a_blank_line_between_statements_is_kept() {
    assert_eq!(
        formatted("let a = 1;\n\n\nlet b = 2;\na + b", DEFAULT_WIDTH),
        "let a = 1;\n\nlet b = 2;\na + b"
    );
}

#[test]
fn if_else_goes_on_one_line_or_one_branch_per_line() {
    assert_eq!(
        formatted("if hasHazmat {hazmatFee} else {0}", DEFAULT_WIDTH),
        "if hasHazmat { hazmatFee } else { 0 }"
    );
    assert_eq!(
        formatted(
            "if requiresTemperatureControl { reeferRate * totalDistance } else if hasHazmat { hazmatFee } else { 0 }",
            40
        ),
        lines(&[
            "if requiresTemperatureControl {",
            "  reeferRate * totalDistance",
            "} else if hasHazmat {",
            "  hazmatFee",
            "} else {",
            "  0",
            "}",
        ])
    );
}

#[test]
fn it_says_when_nothing_changed() {
    assert_eq!(
        format("baseRate * totalDistance", DEFAULT_WIDTH),
        FormatOutcome::Formatted {
            text: String::from("baseRate * totalDistance"),
            changed: false
        }
    );
}

#[test]
fn an_empty_a_broken_or_a_commented_formula_is_refused() {
    assert_eq!(
        format("   ", DEFAULT_WIDTH),
        FormatOutcome::Refused(FormatRefusal::Empty)
    );
    assert_eq!(
        format("baseRate * (totalDistance", DEFAULT_WIDTH),
        FormatOutcome::Refused(FormatRefusal::Syntax)
    );
    assert_eq!(
        format("baseRate // per mile\n* totalDistance", DEFAULT_WIDTH),
        FormatOutcome::Refused(FormatRefusal::Comments)
    );
    assert_eq!(
        format("if a {} else { 1 }", DEFAULT_WIDTH),
        FormatOutcome::Refused(FormatRefusal::Syntax)
    );
}

#[test]
fn formatting_its_own_output_changes_nothing() {
    let once = formatted(LONG_LETS, 60);
    assert_eq!(
        format(&once, 60),
        FormatOutcome::Formatted {
            text: once.clone(),
            changed: false
        }
    );
}

#[test]
fn the_newer_grammar_is_kept_as_written() {
    for source in [
        "(let a = 2; a * 3)",
        "reduce([1, 2, 3], #acc + #, 0)",
        "stops?.[0] == nil ? 1 : 0",
        "len(filter([1, 2, 3], {let x = #; x > 1}))",
        "if isRush { let a = 2; a } else { 0 }",
        "0x1F + 10_000 + 1.",
        "0 < totalWeight <= 5000 ? 1 : 0",
        "a ?: b",
        "pickupDate.Year()",
        "origin.zip[0:3]",
        "a?.[1:2]",
        "{a: 1, \"b c\": 2}",
        "- -x",
        "!x || !y",
        "2 ^ 3 ** 2",
        "x not contains \"a\"",
        "1..3",
    ] {
        assert_eq!(
            format(source, DEFAULT_WIDTH),
            FormatOutcome::Formatted {
                text: source.to_owned(),
                changed: false
            },
            "{source}"
        );
    }
}

#[test]
fn a_value_line_before_the_last_keeps_its_semicolon() {
    assert_eq!(formatted("1;2", DEFAULT_WIDTH), "1;\n2");
}

#[test]
fn a_long_chained_comparison_breaks_before_its_last_operator() {
    assert_eq!(
        formatted(
            "minimumBillableWeight < totalWeight <= maximumBillableWeight ? 1 : 0",
            40
        ),
        lines(&[
            "minimumBillableWeight < totalWeight",
            "    <= maximumBillableWeight ? 1",
            "  : 0",
        ])
    );
}

#[test]
fn a_broken_elvis_keeps_one_colon() {
    let source = "let v0 = ({k: (if a { 2.50 } else { 2.50 }), \"other key\": totalWeight} ?: (a ? a : `d``e`));";
    let text = formatted(&format!("{source} v0"), 82);
    assert_eq!(
        text,
        lines(&[
            "let v0 = ({k: (if a { 2.50 } else { 2.50 }), \"other key\": totalWeight} ?:",
            "  (a ? a : `d``e`));",
            "v0",
        ])
    );
    assert_eq!(
        formatted(
            "customer.name == \"a very long name indeed\" ?: totalWeight > 10",
            60
        ),
        lines(&[
            "customer.name == \"a very long name indeed\" ?:",
            "  totalWeight > 10",
        ])
    );
}

#[test]
fn a_range_from_a_trailing_dot_number_keeps_its_spaces() {
    assert_eq!(formatted("1. .. a", DEFAULT_WIDTH), "1. .. a");
    assert_eq!(formatted("1 .. 3", DEFAULT_WIDTH), "1..3");
}

#[test]
fn a_member_of_a_number_keeps_the_space_that_keeps_its_dot_apart() {
    let text = formatted("g..27\n.g4\n..54\n..5.", 71);
    assert_eq!(text, "g..27 .g4..54..5.");
    assert!(syntax::parse(&text, &[]).diagnostics.is_empty());
    assert_eq!(
        format(&text, 71),
        FormatOutcome::Formatted {
            text: text.clone(),
            changed: false
        }
    );
}

#[test]
fn a_dot_after_a_number_is_spaced_only_when_the_number_would_take_it() {
    for (source, expected) in [
        ("27 .g4", "27 .g4"),
        ("27\n.e5", "27 .e5"),
        ("27 .E1", "27 .E1"),
        ("1_ .x", "1_ .x"),
        ("5. .x", "5. .x"),
        ("27 .abs()", "27 .abs()"),
        ("27 ?.g4", "27?.g4"),
        ("1.5 .x", "1.5.x"),
        (".5 .x", ".5.x"),
        ("1e5 .x", "1e5.x"),
        ("0x1F .x", "0x1F.x"),
        ("(27) .g4", "(27).g4"),
        ("27 .g4 .. 5.", "27 .g4..5."),
        ("5. .. .5", "5. .. .5"),
    ] {
        assert_eq!(formatted(source, DEFAULT_WIDTH), expected, "{source:?}");
    }
}
