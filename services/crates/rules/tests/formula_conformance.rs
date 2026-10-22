//! The conformance corpus: the checker saves exactly what the corpus says, and for every refused
//! formula reports the diagnostic the corpus names where it names it.

use serde::Deserialize;
use trenova_rules::formula::check;
use trenova_rules::formula::check::Expect;
use trenova_rules::formula::check::Schema;
use trenova_rules::formula::check::SchemaInput;
use trenova_rules::formula::check::Severity;
use trenova_rules::span::Utf16Index;

const DESCRIBE: &str = include_str!(
    "../../../tms/internal/modules/rating/app/formula/testdata/shipment.describe.json"
);
const CORPUS: &str = include_str!("../testdata/conformance.json");

#[derive(Deserialize)]
struct Input {
    name: String,
    #[serde(rename = "type")]
    kind: String,
}

#[derive(Deserialize)]
struct Case {
    id: String,
    expression: String,
    accepted: bool,
    code: Option<String>,
    from: Option<u32>,
}

#[derive(Deserialize)]
struct Corpus {
    inputs: Vec<Input>,
    cases: Vec<Case>,
}

#[test]
fn the_checker_saves_exactly_what_the_corpus_says() {
    let corpus: Corpus = serde_json::from_str(CORPUS).expect("the corpus parses");
    let mut schema: Schema = serde_json::from_str(DESCRIBE).expect("the schema parses");
    schema.inputs = corpus
        .inputs
        .iter()
        .map(|input| SchemaInput {
            name: input.name.clone(),
            kind: Some(input.kind.clone()),
        })
        .collect();
    let mut ids = std::collections::HashSet::new();
    let mut failures = Vec::new();
    for case in &corpus.cases {
        assert!(ids.insert(case.id.clone()), "{} is listed twice", case.id);
        let analysis = check::check(&case.expression, Expect::Amount, Some(&schema), None);
        let index = Utf16Index::new(&case.expression);
        let errors: Vec<(String, u32, String)> = analysis
            .diagnostics
            .iter()
            .filter(|diagnostic| diagnostic.severity == Severity::Error)
            .map(|diagnostic| {
                (
                    diagnostic.code.as_str().to_owned(),
                    index.to_utf16(diagnostic.span.start),
                    diagnostic.message.english(),
                )
            })
            .collect();
        if case.accepted {
            if !errors.is_empty() {
                failures.push(format!("{}: accepted, but {errors:?}", case.id));
            }
            continue;
        }
        let (Some(code), Some(from)) = (&case.code, case.from) else {
            failures.push(format!(
                "{}: a refused case names its code and from",
                case.id
            ));
            continue;
        };
        if !errors
            .iter()
            .any(|(found, at, _)| found == code && *at == from)
        {
            failures.push(format!(
                "{}: wanted {code} at {from}, got {errors:?}",
                case.id
            ));
        }
    }
    assert!(failures.is_empty(), "{}", failures.join("\n"));
}
