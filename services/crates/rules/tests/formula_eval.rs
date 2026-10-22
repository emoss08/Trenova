//! The evaluator against `testdata/evaluation.json`: every evaluation-semantics case of the Go
//! engine's tests, ported, plus the exact or ordered results where Go's were float64 or hash
//! order. Each case names the Go test it comes from.

use rust_decimal::Decimal;
use serde::Deserialize;
use serde_json::value::RawValue;
use std::collections::BTreeMap;
use std::str::FromStr;
use trenova_rules::formula::eval::EvalError;
use trenova_rules::formula::eval::Limits;
use trenova_rules::formula::eval::Tables;
use trenova_rules::formula::eval::Value;
use trenova_rules::formula::eval::interp;
use trenova_rules::formula::eval::interp::LookupRecord;
use trenova_rules::formula::eval::interp::LookupSource;
use trenova_rules::formula::eval::interp::Setup;
use trenova_rules::formula::eval::parse_value;
use trenova_rules::formula::eval::tables::TableWire;
use trenova_rules::formula::eval::write_value;
use trenova_rules::formula::syntax;

#[derive(Deserialize)]
#[serde(deny_unknown_fields)]
struct Corpus {
    #[serde(rename = "about")]
    _about: String,
    environments: BTreeMap<String, Box<RawValue>>,
    tables: BTreeMap<String, Vec<TableWire>>,
    cases: Vec<Case>,
}

#[derive(Deserialize)]
#[serde(deny_unknown_fields)]
struct Case {
    name: String,
    from: String,
    source: String,
    #[serde(default)]
    env: Option<Box<RawValue>>,
    #[serde(default)]
    environment: Option<String>,
    #[serde(default)]
    lookups: Option<String>,
    #[serde(default)]
    limits: Option<LimitsInput>,
    #[serde(default, deserialize_with = "present")]
    value: Option<Box<RawValue>>,
    #[serde(default)]
    near: Option<Near>,
    #[serde(default)]
    error: Option<ErrorExpectation>,
    #[serde(default)]
    recorded: Option<Vec<Recorded>>,
}

#[derive(Deserialize)]
#[serde(deny_unknown_fields, rename_all = "camelCase")]
struct LimitsInput {
    steps: Option<u64>,
    text_bytes: Option<u64>,
    items: Option<u64>,
}

#[derive(Deserialize)]
#[serde(deny_unknown_fields)]
struct Near {
    value: String,
    within: String,
}

#[derive(Deserialize)]
#[serde(deny_unknown_fields)]
struct ErrorExpectation {
    code: String,
    #[serde(default)]
    message: Option<String>,
    #[serde(default)]
    english: Option<String>,
    #[serde(default)]
    fields: Option<Vec<String>>,
    #[serde(default)]
    guards: Option<Vec<String>>,
}

#[derive(Deserialize)]
#[serde(deny_unknown_fields, rename_all = "camelCase")]
struct Recorded {
    function: String,
    table: String,
    #[serde(default)]
    value: Option<String>,
    #[serde(default)]
    matched_key: Option<String>,
    #[serde(default)]
    band_min: Option<String>,
    #[serde(default)]
    band_max: Option<String>,
    #[serde(default)]
    adjusted: bool,
    #[serde(default)]
    error: Option<String>,
}

/// A present `value`, even `null`: an expected nil is still an expectation.
fn present<'de, D: serde::Deserializer<'de>>(
    deserializer: D,
) -> Result<Option<Box<RawValue>>, D::Error> {
    Box::<RawValue>::deserialize(deserializer).map(Some)
}

fn corpus() -> Corpus {
    let text = include_str!("../testdata/evaluation.json");
    serde_json::from_str(text).expect("testdata/evaluation.json reads")
}

fn decimal(text: &str) -> Decimal {
    Decimal::from_str(text).expect("a decimal")
}

fn environment(case: &Case, corpus: &Corpus) -> Result<Value, String> {
    let env_text = match (&case.env, &case.environment) {
        (Some(env), _) => env.get().to_owned(),
        (None, Some(name)) => corpus
            .environments
            .get(name)
            .ok_or_else(|| format!("no environment {name}"))?
            .get()
            .to_owned(),
        (None, None) => String::from("{}"),
    };
    parse_value(&env_text).map_err(|error| format!("env: {error}"))
}

fn check(case: &Case, corpus: &Corpus, built: &BTreeMap<String, Tables>) -> Result<(), String> {
    let env = environment(case, corpus)?;
    let lookups = match case.lookups.as_deref() {
        None => LookupSource::Unavailable,
        Some("stub") => LookupSource::Stub,
        Some(name) => LookupSource::Tables(
            built
                .get(name)
                .ok_or_else(|| format!("no table set {name}"))?,
        ),
    };
    let mut limits = Limits::DEFAULT;
    if let Some(input) = &case.limits {
        limits.steps = input.steps.unwrap_or(limits.steps);
        limits.text_bytes = input.text_bytes.unwrap_or(limits.text_bytes);
        limits.items = input.items.unwrap_or(limits.items);
    }
    let program = syntax::parse(&case.source, &[]);
    if let Some(diagnostic) = program.diagnostics.first() {
        return Err(format!("does not parse: {}", diagnostic.message.english()));
    }
    let evaluation = interp::evaluate(
        &program,
        Setup {
            env: &env,
            lookups,
            limits,
            trace_statements: false,
            record_lookups: case.recorded.is_some(),
        },
    );
    match (&evaluation.result, &case.error) {
        (Ok(got), None) => check_value(case, got)?,
        (Ok(got), Some(want)) => {
            return Err(format!(
                "got {}, want a {} error",
                write_value(got),
                want.code
            ));
        }
        (Err(error), None) => {
            return Err(format!(
                "failed with {}: {}",
                error.code.as_str(),
                error.message.english()
            ));
        }
        (Err(error), Some(want)) => check_error(error, want)?,
    }
    match &case.recorded {
        Some(recorded) => check_recorded(&evaluation.lookups, recorded),
        None => Ok(()),
    }
}

fn check_value(case: &Case, got: &Value) -> Result<(), String> {
    if let Some(want) = &case.value {
        let want = parse_value(want.get()).map_err(|error| format!("value: {error}"))?;
        if *got != want {
            return Err(format!(
                "got {}, want {}",
                write_value(got),
                write_value(&want)
            ));
        }
    }
    if let Some(near) = &case.near {
        let Value::Number(number) = got else {
            return Err(format!("got {}, want a number", write_value(got)));
        };
        if (*number - decimal(&near.value)).abs() > decimal(&near.within) {
            return Err(format!(
                "got {number}, want {} ± {}",
                near.value, near.within
            ));
        }
    }
    if case.value.is_none() && case.near.is_none() {
        return Err(String::from("the case expects nothing"));
    }
    Ok(())
}

fn check_error(error: &EvalError, want: &ErrorExpectation) -> Result<(), String> {
    if error.code.as_str() != want.code {
        return Err(format!(
            "failed with {} ({}), want {}",
            error.code.as_str(),
            error.message.english(),
            want.code
        ));
    }
    if let Some(message) = &want.message
        && !error.message.message.contains(message.as_str())
    {
        return Err(format!(
            "message {:?} does not contain {message:?}",
            error.message.message
        ));
    }
    if let Some(english) = &want.english
        && !error.message.english().contains(english.as_str())
    {
        return Err(format!(
            "message {:?} does not contain {english:?}",
            error.message.english()
        ));
    }
    if let Some(fields) = &want.fields
        && &error.fields != fields
    {
        return Err(format!("fields {:?}, want {fields:?}", error.fields));
    }
    if let Some(guards) = &want.guards
        && &error.guards != guards
    {
        return Err(format!("guards {:?}, want {guards:?}", error.guards));
    }
    Ok(())
}

fn check_recorded(records: &[LookupRecord], recorded: &[Recorded]) -> Result<(), String> {
    if recorded.len() != records.len() {
        return Err(format!(
            "{} lookups recorded, want {}",
            records.len(),
            recorded.len()
        ));
    }
    let text = |value: Option<Decimal>| value.map(|value| value.normalize().to_string());
    for (got, want) in records.iter().zip(recorded) {
        let matched = got.matched.clone().unwrap_or_default();
        let same = got.function == want.function
            && got.table == want.table
            && text(got.value) == want.value
            && matched.key == want.matched_key
            && text(matched.band_min) == want.band_min
            && text(matched.band_max) == want.band_max
            && matched.adjusted == want.adjusted
            && got
                .error
                .as_ref()
                .map(|error| error.code.as_str().to_owned())
                == want.error;
        if !same {
            return Err(format!("recorded {got:?}"));
        }
    }
    Ok(())
}

#[test]
fn the_harness_notices_a_wrong_expectation() {
    let corpus: Corpus = serde_json::from_str(
        r#"{"about": "", "environments": {}, "tables": {}, "cases": [
            {"name": "wrong value", "from": "self", "source": "1 + 1", "value": 3},
            {"name": "wrong code", "from": "self", "source": "1 / 0", "error": {"code": "overflow"}},
            {"name": "missing error", "from": "self", "source": "1", "error": {"code": "type"}},
            {"name": "no expectation", "from": "self", "source": "1"},
            {"name": "wrong kind", "from": "self", "source": "\"2\"", "value": 2}
        ]}"#,
    )
    .expect("corpus");
    for case in &corpus.cases {
        assert!(
            check(case, &corpus, &BTreeMap::new()).is_err(),
            "{} passed",
            case.name
        );
    }
}

#[test]
fn every_case_in_the_corpus_holds() {
    let corpus = corpus();
    let mut built = BTreeMap::new();
    for (name, tables) in &corpus.tables {
        built.insert(
            name.clone(),
            Tables::build(tables).unwrap_or_else(|error| panic!("table set {name}: {error}")),
        );
    }
    let mut names = std::collections::BTreeSet::new();
    let mut failures = Vec::new();
    for case in &corpus.cases {
        assert!(
            names.insert(case.name.clone()),
            "{} is listed twice",
            case.name
        );
        assert!(!case.from.is_empty(), "{} names no Go test", case.name);
        if let Err(problem) = check(case, &corpus, &built) {
            failures.push(format!(
                "{} ({}): {}\n    {}",
                case.name, case.from, case.source, problem
            ));
        }
    }
    assert!(
        failures.is_empty(),
        "{} of {} cases fail:\n{}",
        failures.len(),
        corpus.cases.len(),
        failures.join("\n")
    );
}
