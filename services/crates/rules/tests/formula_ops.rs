//! The five static formula operations through the module's request envelope: the wire shapes,
//! UTF-16 positions, and what signature help and the hover card get.

use serde_json::Value;
use serde_json::json;
use trenova_rules::formula::check;
use trenova_rules::ops::dispatch;

const DESCRIBE: &str = include_str!(
    "../../../tms/internal/modules/rating/app/formula/testdata/shipment.describe.json"
);

fn call(op: &str, input: &Value) -> Value {
    let request = json!({"op": op, "input": input}).to_string();
    let response: Value =
        serde_json::from_slice(&dispatch(request.as_bytes())).expect("the response is JSON");
    response
        .get("ok")
        .cloned()
        .unwrap_or_else(|| panic!("{op} failed: {response}"))
}

fn schema(inputs: &Value) -> Value {
    let mut schema: Value = serde_json::from_str(DESCRIBE).expect("the schema parses");
    schema["inputs"] = inputs.clone();
    schema
}

fn checked(source: &str) -> Value {
    call(
        "formula.check",
        &json!({"source": source, "expect": "amount", "schema": schema(&json!([{"name": "fuelPct", "type": "Number"}]))}),
    )
}

fn utf16_slice(source: &str, from: u64, to: u64) -> String {
    let units: Vec<u16> = source.encode_utf16().collect();
    let from = usize::try_from(from).expect("a small offset");
    let to = usize::try_from(to).expect("a small offset");
    String::from_utf16(&units[from..to]).expect("a slice on a boundary")
}

#[test]
fn the_catalog_lists_functions_builtins_disabled_and_reserved_names() {
    let catalog = call("formula.catalog", &json!({}));
    let functions = catalog["functions"].as_array().expect("functions");
    let builtins = catalog["builtins"].as_array().expect("builtins");
    let find = |list: &[Value], name: &str| {
        list.iter()
            .find(|entry| entry["name"] == name)
            .cloned()
            .unwrap_or_else(|| panic!("{name} is listed"))
    };
    let round = find(functions, "round");
    assert_eq!(round["minArgs"], 1);
    assert_eq!(round["maxArgs"], 2);
    assert_eq!(round["returns"], "Number");
    assert_eq!(round["numericArgs"], true);
    assert_eq!(round["category"], "rounding");
    assert_eq!(
        round["description"],
        json!({"message": "Rounds a number to the given decimal places (default 0, between -12 and 12)."})
    );
    assert_eq!(
        round["example"],
        json!({"message": "round(3.14159, 2) // 3.14"})
    );
    assert_eq!(find(functions, "min")["maxArgs"], -1);
    assert_eq!(find(functions, "min")["signature"], "min(...values)");
    assert_eq!(find(functions, "max")["signature"], "max(...values)");
    assert_eq!(find(builtins, "filter")["predicate"], true);
    assert_eq!(find(functions, "sum")["predicate"], false);
    for name in ["reverse", "toPairs", "fromPairs", "bitnot"] {
        find(builtins, name);
        assert!(
            !functions.iter().any(|entry| entry["name"] == name),
            "{name} is a builtin, not a documented function"
        );
    }
    for name in [
        "startsWith",
        "endsWith",
        "contains",
        "matches",
        "[start:end]",
    ] {
        let entry = find(functions, name);
        assert_eq!(entry["operator"], true, "{name}");
        assert_eq!(entry["category"], "string");
    }
    for name in ["upper", "lower", "trim", "len", "indexOf", "replace"] {
        let entry = find(functions, name);
        assert_eq!(entry["operator"], false, "{name}");
        assert_eq!(entry["category"], "string");
    }
    for name in ["lookup", "lookupOr", "lookup2", "lookup2Or"] {
        assert_eq!(find(functions, name)["category"], "rateTable");
    }
    for name in ["toBase64", "fromBase64", "timezone"] {
        assert!(
            !builtins.iter().any(|entry| entry["name"] == name),
            "{name}"
        );
    }
    let disabled = catalog["disabled"].as_array().expect("disabled");
    let names: Vec<&str> = disabled.iter().filter_map(|d| d["name"].as_str()).collect();
    assert_eq!(names, ["fromJSON", "now", "repeat", "toJSON"]);
    let now = &disabled[1];
    assert_eq!(now["reason"], "nondeterministic");
    assert_eq!(now["replacement"], "ratingTime");
    assert!(disabled[0].get("replacement").is_none());
    let reserved: Vec<&str> = catalog["reserved"]
        .as_array()
        .expect("reserved")
        .iter()
        .filter_map(Value::as_str)
        .collect();
    for function in functions.iter().chain(builtins) {
        let name = function["name"].as_str().expect("a name");
        if !name.starts_with('[') {
            assert!(reserved.contains(&name), "{name} is reserved");
        }
    }
    assert!(reserved.contains(&"date"));
    assert!(reserved.contains(&"let"));
    assert!(!reserved.contains(&"baseRate"));
}

#[test]
fn every_catalog_entry_carries_complete_documentation() {
    let catalog = call("formula.catalog", &json!({}));
    let mut seen = std::collections::HashSet::new();
    for entry in catalog["functions"]
        .as_array()
        .expect("functions")
        .iter()
        .chain(catalog["builtins"].as_array().expect("builtins"))
    {
        let name = entry["name"].as_str().expect("a name");
        assert!(seen.insert(name.to_owned()), "{name} is listed twice");
        for field in ["signature", "category", "returns"] {
            assert!(
                entry[field].as_str().is_some_and(|text| !text.is_empty()),
                "{name}.{field}"
            );
        }
        for field in ["description", "example"] {
            assert!(
                entry[field]["message"]
                    .as_str()
                    .is_some_and(|text| !text.is_empty() && !text.contains(['{', '}'])),
                "{name}.{field} is a message without braces"
            );
        }
        let max = entry["maxArgs"].as_i64().expect("maxArgs");
        assert!(
            max == -1 || max >= entry["minArgs"].as_i64().expect("minArgs"),
            "{name}"
        );
    }
}

#[test]
fn check_reports_diagnostics_with_utf16_positions_messages_and_fixes() {
    let source = "let tarifa_año = 1; tarifa_año + bar";
    let output = checked(source);
    let diagnostic = &output["diagnostics"][0];
    assert_eq!(diagnostic["from"], 33);
    assert_eq!(diagnostic["to"], 36);
    assert_eq!(diagnostic["severity"], "error");
    assert_eq!(diagnostic["code"], "unknown-variable");
    assert_eq!(
        diagnostic["message"],
        "{0} isn't a shipment field or one of this template's inputs. Declare it as an input, or pick a field from the reference."
    );
    assert_eq!(diagnostic["args"], json!(["`bar`"]));
    assert_eq!(diagnostic["name"], "bar");
    assert_eq!(
        diagnostic["fixes"],
        json!([{"kind": "declare", "label": {"message": "Declare {0} as a {1} input", "args": ["bar", "Number"]}, "name": "bar", "type": "Number"}])
    );
}

#[test]
fn check_returns_the_studio_analysis() {
    let output = checked("let fuel = fuelPct;\nbaseRate * totalDistance + fuel + round(1)");
    assert_eq!(output["empty"], false);
    assert_eq!(output["pending"], false);
    assert_eq!(output["resultType"], "Number");
    assert_eq!(
        output["reads"],
        json!(["fuelPct", "baseRate", "totalDistance"])
    );
    assert_eq!(output["schemaReads"], json!(["baseRate", "totalDistance"]));
    assert_eq!(output["inputReads"], json!(["fuelPct"]));
    assert_eq!(output["functionCalls"], json!(["round"]));
    assert_eq!(
        output["bindings"],
        json!([{"name": "fuel", "type": "Number", "reads": 1, "from": 0, "to": 18, "nameFrom": 4, "nameTo": 8}])
    );
    assert_eq!(output["lookupTables"], json!([]));
}

#[test]
fn check_with_no_schema_is_pending() {
    let output = call(
        "formula.check",
        &json!({"source": "baseRate * 2", "schema": null}),
    );
    assert_eq!(output["pending"], true);
    assert_eq!(output["diagnostics"], json!([]));
}

#[test]
fn check_lists_the_rate_tables_a_formula_reads() {
    let source = "lookup(\"fuel\", 1) + lookup2(\"grid\", \"a\", 2)";
    let output = call(
        "formula.check",
        &json!({"source": source, "schema": schema(&json!([])), "tables": [{"name": "fuel", "axes": 1, "banded": false}, {"name": "grid", "axes": 2}]}),
    );
    assert_eq!(output["diagnostics"], json!([]));
    assert_eq!(
        output["lookupTables"],
        json!([{"name": "fuel", "function": "lookup", "from": 7, "to": 13}, {"name": "grid", "function": "lookup2", "from": 28, "to": 34}])
    );
}

#[test]
fn statements_are_split_into_lets_and_the_result() {
    let source = "let base = rate * miles;\nlet fuel = base * 0.1;\nbase + fuel";
    let statements = checked(source)["statements"].clone();
    let statements = statements.as_array().expect("statements");
    assert_eq!(statements.len(), 3);
    assert_eq!(statements[0]["kind"], "let");
    assert_eq!(statements[0]["name"], "base");
    assert_eq!(statements[0]["code"], "let base = rate * miles");
    assert_eq!(statements[0]["index"], 0);
    assert_eq!(statements[1]["name"], "fuel");
    assert_eq!(statements[1]["code"], "let fuel = base * 0.1");
    assert_eq!(statements[2]["kind"], "result");
    assert!(statements[2].get("name").is_none());
    assert_eq!(statements[2]["code"], "base + fuel");
    assert_eq!(statements[2]["index"], 2);
    for statement in statements {
        assert_eq!(
            statement["code"].as_str().expect("code"),
            utf16_slice(
                source,
                statement["from"].as_u64().expect("from"),
                statement["to"].as_u64().expect("to")
            )
        );
    }
}

#[test]
fn statements_skip_semicolons_inside_strings_and_parentheses() {
    let source =
        "let label = \"a;b\"; let inner = (let x = 2; x * 3); label == \"a;b\" ? inner : 0";
    let output = checked(source);
    let codes: Vec<&str> = output["statements"]
        .as_array()
        .expect("statements")
        .iter()
        .filter_map(|statement| statement["code"].as_str())
        .collect();
    assert_eq!(
        codes,
        [
            "let label = \"a;b\"",
            "let inner = (let x = 2; x * 3)",
            "label == \"a;b\" ? inner : 0"
        ]
    );
}

#[test]
fn statements_count_positions_in_utf16() {
    let source = "let tag = \"🚚 é\"; // ✓ comment\nlet miles = 10; miles * 2;";
    let output = checked(source);
    let statements = output["statements"].as_array().expect("statements");
    assert_eq!(statements.len(), 3);
    assert_eq!(statements[0]["from"], 0);
    let slice = |index: usize| {
        utf16_slice(
            source,
            statements[index]["from"].as_u64().expect("from"),
            statements[index]["to"].as_u64().expect("to"),
        )
    };
    assert_eq!(slice(0), "let tag = \"🚚 é\"");
    assert_eq!(statements[1]["code"], "let miles = 10");
    assert_eq!(slice(1), "let miles = 10");
    assert_eq!(statements[2]["code"], "miles * 2");
    assert_eq!(slice(2), "miles * 2");
}

#[test]
fn a_single_expression_is_one_statement_without_its_spaces() {
    let output = checked("  totalDistance * 2  ");
    assert_eq!(
        output["statements"],
        json!([{"index": 0, "kind": "result", "code": "totalDistance * 2", "from": 2, "to": 19}])
    );
}

#[test]
fn a_formula_that_does_not_parse_still_has_its_statements_and_an_error() {
    let output = checked("let a = ; a");
    assert_eq!(
        output["statements"].as_array().expect("statements").len(),
        2
    );
    assert_eq!(output["diagnostics"][0]["code"], "syntax");
}

#[test]
fn format_returns_the_text_or_why_not() {
    assert_eq!(
        call("formula.format", &json!({"source": "a+b"})),
        json!({"ok": true, "text": "a + b", "changed": true})
    );
    assert_eq!(
        call("formula.format", &json!({"source": "a + b", "width": 10})),
        json!({"ok": true, "text": "a + b", "changed": false})
    );
    assert_eq!(
        call("formula.format", &json!({"source": "a +"})),
        json!({"ok": false, "reason": "syntax"})
    );
    assert_eq!(
        call("formula.format", &json!({"source": ""})),
        json!({"ok": false, "reason": "empty"})
    );
    assert_eq!(
        call("formula.format", &json!({"source": "a // b"})),
        json!({"ok": false, "reason": "comments"})
    );
    assert_eq!(
        call(
            "formula.format",
            &json!({"source": "a soundsLike\"b\"", "operators": ["soundsLike"]})
        ),
        json!({"ok": true, "text": "a soundsLike \"b\"", "changed": true})
    );
}

fn signature(source: &str, position: usize) -> Value {
    call(
        "formula.signature",
        &json!({"source": source, "position": position}),
    )
}

#[test]
fn signature_help_finds_the_call_and_the_argument() {
    let source = "round(baseRate, 2)";
    assert_eq!(
        signature(source, 6),
        json!({"name": "round", "active": 0, "from": 5, "to": 18})
    );
    assert_eq!(signature(source, 16)["active"], 1);
    assert_eq!(signature(source, 5), Value::Null);
    assert_eq!(signature(source, 18), Value::Null);
    assert_eq!(signature("round(baseRate, ", 16)["active"], 1);
    assert_eq!(signature("round(", 6)["name"], "round");
}

#[test]
fn signature_help_takes_the_innermost_call() {
    let source = "max(round(totalWeight), 2)";
    assert_eq!(signature(source, 12)["name"], "round");
    assert_eq!(signature(source, 25)["name"], "max");
    assert_eq!(signature(source, 25)["active"], 1);
    assert_eq!(signature("filter(stops, .type == 1)", 15)["name"], "filter");
    assert_eq!(signature("pickupDate.Year()", 16), Value::Null);
    assert_eq!(signature("nope(1)", 5), Value::Null);
}

#[test]
fn signature_help_counts_positions_in_utf16() {
    assert_eq!(
        signature("\"🚚\" + round(1, 2)", 15),
        json!({"name": "round", "active": 1, "from": 12, "to": 18})
    );
}

fn identifier(source: &str, position: usize, with_schema: bool) -> Value {
    let mut input = json!({"source": source, "position": position});
    if with_schema {
        input["schema"] = schema(&json!([{"name": "fuel", "type": "Number"}]));
    }
    call("formula.identifier", &input)
}

#[test]
fn hover_finds_the_identifier_under_the_cursor() {
    let doc = "coalesce(weight, 0) * customer.rate + fuel";
    let weight = doc.find("weight").expect("weight");
    assert_eq!(
        identifier(doc, weight + 2, false),
        json!({"name": "weight", "kind": "variable", "from": weight, "to": weight + 6})
    );
    let customer = doc.find("customer.rate").expect("customer");
    assert_eq!(
        identifier(doc, customer + 10, false)["name"],
        "customer.rate"
    );
    assert_eq!(identifier(doc, 2, false)["name"], "coalesce");
    assert_eq!(identifier(doc, 2, false)["kind"], "function");
    assert_eq!(
        identifier(doc, doc.find('*').expect("*"), false),
        Value::Null
    );
    assert_eq!(
        identifier(doc, doc.find(" +").expect("+"), false),
        Value::Null
    );
}

#[test]
fn hover_says_what_a_name_is() {
    let doc = "let base = fuel; base + weight + nope + #index + pickupDate.Year() + .weight";
    let kind = |needle: &str| {
        let at = doc.find(needle).expect("in the doc");
        identifier(doc, at + 1, true)["kind"].clone()
    };
    assert_eq!(kind("base +"), "let");
    assert_eq!(kind("fuel"), "input");
    assert_eq!(kind("weight +"), "variable");
    assert_eq!(kind("nope"), "unknown");
    assert_eq!(kind("#index"), "pointer");
    assert_eq!(kind("pickupDate"), "method");
    let field = doc.rfind(".weight").expect("a field") + 2;
    assert_eq!(identifier(doc, field, true)["kind"], "field");
}

#[test]
fn identifier_tokens_join_dotted_paths_and_mark_calls() {
    let tokens = check::identifier_tokens("baseRate + customer.name");
    let names: Vec<&str> = tokens.iter().map(|token| token.name.as_str()).collect();
    assert_eq!(names, ["baseRate", "customer.name"]);
    assert_eq!((tokens[0].span.start, tokens[0].span.end), (0, 8));
    let calls = check::identifier_tokens("round(baseRate, 2)");
    assert!(calls[0].is_call);
    assert_eq!(calls[0].name, "round");
    assert!(!calls[1].is_call);
    assert!(check::identifier_tokens("round (1.5)")[0].is_call);
    let names: Vec<String> = check::identifier_tokens(
        "\"notAVariable\" + baseRate // trailing bogusName\n/* bogusBlock */ + weight",
    )
    .into_iter()
    .map(|token| token.name)
    .collect();
    assert_eq!(names, ["baseRate", "weight"]);
}

#[test]
fn malformed_input_is_a_request_error() {
    let response: Value =
        serde_json::from_slice(&dispatch(br#"{"op":"formula.check","input":{"source":1}}"#))
            .expect("JSON");
    assert_eq!(response["error"]["code"], "invalid_input");
    let response: Value = serde_json::from_slice(&dispatch(
        br#"{"op":"formula.signature","input":{"source":"a"}}"#,
    ))
    .expect("JSON");
    assert_eq!(response["error"]["code"], "invalid_input");
}
