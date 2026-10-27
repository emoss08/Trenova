//! `lading.replay`: two versions of a project run one map on the same inputs, compared: which
//! read the same, which fields differ and how, and what is newly held, rejected or declined.
//! Shadow runs use the same comparison, one live message at a time.

use std::collections::BTreeMap;

use serde::{Deserialize, Serialize};
use serde_json::Value as Json;
use trenova_module_kit::message::Message;

use super::check::Program;
use super::eval::interp::{Outcome, OutcomeKind2};
use super::eval::run::{LookupAnswer, RunInput, RunRequest, RunResult, run};

/// One input to replay.
#[derive(Debug, Clone, Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct ReplayInput {
    /// The host's identifier for it (a message's ID).
    pub id: String,
    /// The input.
    pub input: RunInput,
    /// Lookup answers for it.
    #[serde(default)]
    pub lookups: Vec<LookupAnswer>,
    /// When it originally ran, RFC 3339.
    #[serde(default)]
    pub now: Option<String>,
}

/// What to replay.
#[derive(Debug, Clone, Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct ReplayRequest {
    /// The map, in both versions.
    pub map: String,
    /// The inputs, at most 500.
    pub inputs: Vec<ReplayInput>,
    /// Setting values both versions run with.
    #[serde(default)]
    pub settings: BTreeMap<String, Json>,
}

/// The most inputs one replay call takes; the host pages through more.
pub const MAX_INPUTS: usize = 500;

/// One field that differs.
#[derive(Debug, Clone, PartialEq, Serialize)]
#[serde(rename_all = "camelCase")]
pub struct Difference {
    /// Its path: `stops[1].address.city`, or `transaction` for X12.
    pub path: String,
    /// The active version's value.
    pub before: Json,
    /// The candidate's.
    pub after: Json,
}

/// An outcome only the candidate has.
#[derive(Debug, Clone, PartialEq, Serialize)]
#[serde(rename_all = "camelCase")]
pub struct NewOutcome {
    /// `reject`, `decline`, `hold` or `warn`.
    pub kind: OutcomeKind2,
    /// Its message.
    pub message: String,
    /// The rule, for a rule.
    pub rule: String,
}

/// One input compared.
#[derive(Debug, Clone, PartialEq, Serialize)]
#[serde(rename_all = "camelCase")]
pub struct ReplayItem {
    /// The input's identifier.
    pub id: String,
    /// Whether both versions produced the same.
    pub same: bool,
    /// The fields that differ.
    pub differences: Vec<Difference>,
    /// Outcomes only the candidate has.
    pub new_outcomes: Vec<NewOutcome>,
    /// Outcomes only the active version had.
    pub gone_outcomes: Vec<NewOutcome>,
    /// Why the active version could not run it.
    #[serde(skip_serializing_if = "Option::is_none")]
    pub before_error: Option<Message>,
    /// Why the candidate could not.
    #[serde(skip_serializing_if = "Option::is_none")]
    pub after_error: Option<Message>,
}

/// The counts over a replay.
#[derive(Debug, Clone, Default, PartialEq, Serialize)]
#[serde(rename_all = "camelCase")]
pub struct ReplaySummary {
    /// Inputs compared.
    pub total: usize,
    /// Inputs that read the same.
    pub same: usize,
    /// Inputs that differ.
    pub differ: usize,
    /// Inputs newly rejected.
    pub newly_rejected: usize,
    /// Inputs newly held.
    pub newly_held: usize,
    /// Inputs newly declined.
    pub newly_declined: usize,
    /// Inputs the candidate could not run.
    pub failed: usize,
    /// How many inputs each field differs in, with list positions folded: `stops[].window.start`.
    pub fields: BTreeMap<String, usize>,
}

/// A replay's result.
#[derive(Debug, Clone, Default, PartialEq, Serialize)]
#[serde(rename_all = "camelCase")]
pub struct ReplayReport {
    /// Every input.
    pub items: Vec<ReplayItem>,
    /// The counts.
    pub summary: ReplaySummary,
}

/// Replays a map in two versions.
#[must_use]
pub fn replay(before: &Program, after: &Program, request: &ReplayRequest) -> ReplayReport {
    let mut report = ReplayReport::default();
    for input in request.inputs.iter().take(MAX_INPUTS) {
        let run_request = RunRequest {
            map: request.map.clone(),
            input: input.input.clone(),
            settings: request.settings.clone(),
            lookups: input.lookups.clone(),
            now: input.now.clone(),
            control: super::eval::run::Control::default(),
            rules: true,
        };
        let old = run(before, &run_request);
        let new = run(after, &run_request);
        let item = compare(&input.id, &old, &new);
        let summary = &mut report.summary;
        summary.total += 1;
        if item.same {
            summary.same += 1;
        } else {
            summary.differ += 1;
        }
        if item.after_error.is_some() {
            summary.failed += 1;
        }
        let newly =
            |kind: OutcomeKind2| item.new_outcomes.iter().any(|outcome| outcome.kind == kind);
        summary.newly_rejected += usize::from(newly(OutcomeKind2::Reject));
        summary.newly_held += usize::from(newly(OutcomeKind2::Hold));
        summary.newly_declined += usize::from(newly(OutcomeKind2::Decline));
        for difference in &item.differences {
            *summary.fields.entry(fold(&difference.path)).or_default() += 1;
        }
        report.items.push(item);
    }
    report
}

/// A path with its list positions folded: `stops[1].name` is `stops[].name`.
fn fold(path: &str) -> String {
    let mut out = String::with_capacity(path.len());
    let mut inside = false;
    for c in path.chars() {
        match c {
            '[' => {
                inside = true;
                out.push('[');
            }
            ']' => {
                inside = false;
                out.push(']');
            }
            _ if inside => {}
            other => out.push(other),
        }
    }
    out
}

/// Compares two runs of one input.
#[must_use]
pub fn compare(id: &str, before: &RunResult, after: &RunResult) -> ReplayItem {
    let mut differences = Vec::new();
    let (old_outcomes, new_outcomes) = (outcomes(before), outcomes(after));
    if let (Some(old), Some(new)) = (&before.output, &after.output) {
        if old.transaction != new.transaction {
            differences.push(Difference {
                path: "transaction".to_owned(),
                before: Json::String(old.transaction.clone()),
                after: Json::String(new.transaction.clone()),
            });
        }
    } else {
        let count = before.transactions.len().max(after.transactions.len());
        for index in 0..count {
            let old = before
                .transactions
                .get(index)
                .map_or(&Json::Null, |transaction| &transaction.document);
            let new = after
                .transactions
                .get(index)
                .map_or(&Json::Null, |transaction| &transaction.document);
            let prefix = if count > 1 {
                format!("[{index}]")
            } else {
                String::new()
            };
            diff(&prefix, old, new, &mut differences);
        }
    }
    let key = |outcome: &Outcome| {
        (
            format!("{:?}", outcome.kind),
            outcome.message.clone(),
            outcome.rule.clone(),
        )
    };
    let added: Vec<NewOutcome> = new_outcomes
        .iter()
        .filter(|outcome| !old_outcomes.iter().any(|old| key(old) == key(outcome)))
        .map(new_outcome)
        .collect();
    let gone: Vec<NewOutcome> = old_outcomes
        .iter()
        .filter(|outcome| !new_outcomes.iter().any(|new| key(new) == key(outcome)))
        .map(new_outcome)
        .collect();
    ReplayItem {
        id: id.to_owned(),
        same: differences.is_empty()
            && added.is_empty()
            && gone.is_empty()
            && before.error.is_none() == after.error.is_none(),
        differences,
        new_outcomes: added,
        gone_outcomes: gone,
        before_error: before.error.clone(),
        after_error: after.error.clone(),
    }
}

fn new_outcome(outcome: &Outcome) -> NewOutcome {
    NewOutcome {
        kind: outcome.kind,
        message: outcome.message.clone(),
        rule: outcome.rule.clone(),
    }
}

fn outcomes(result: &RunResult) -> Vec<Outcome> {
    let mut all: Vec<Outcome> = result
        .transactions
        .iter()
        .flat_map(|transaction| transaction.outcomes.iter().cloned())
        .collect();
    if let Some(output) = &result.output {
        all.extend(output.outcomes.iter().cloned());
    }
    all
}

/// Every path where two JSON values differ, at most 200.
fn diff(path: &str, before: &Json, after: &Json, out: &mut Vec<Difference>) {
    if before == after || out.len() >= 200 {
        return;
    }
    match (before, after) {
        (Json::Object(old), Json::Object(new)) => {
            let keys: std::collections::BTreeSet<&String> = old.keys().chain(new.keys()).collect();
            for key in keys {
                let child = if path.is_empty() {
                    key.clone()
                } else {
                    format!("{path}.{key}")
                };
                diff(
                    &child,
                    old.get(key).unwrap_or(&Json::Null),
                    new.get(key).unwrap_or(&Json::Null),
                    out,
                );
            }
        }
        (Json::Array(old), Json::Array(new)) => {
            for index in 0..old.len().max(new.len()) {
                diff(
                    &format!("{path}[{index}]"),
                    old.get(index).unwrap_or(&Json::Null),
                    new.get(index).unwrap_or(&Json::Null),
                    out,
                );
            }
        }
        _ => out.push(Difference {
            path: path.to_owned(),
            before: before.clone(),
            after: after.clone(),
        }),
    }
}

#[cfg(test)]
mod tests {
    use super::{ReplayInput, ReplayRequest, fold, replay};
    use crate::lading::check::check;
    use crate::lading::check::corpus::{acme, context};
    use crate::lading::eval::run::RunInput;

    #[test]
    fn a_changed_map_shows_its_differences() {
        let project = acme();
        let before = check(&project, &context());
        let mut changed = project.clone();
        for file in &mut changed.files {
            if file.path == "maps/tender-in.lading" {
                file.source = file.source.replace(
                    "name     <- party.N102.titleCase()",
                    "name     <- party.N102",
                );
            }
        }
        let after = check(&changed, &context());
        let sample = project
            .sample("samples/two-stops.x12")
            .expect("sample")
            .content
            .clone();
        let report = replay(
            &before,
            &after,
            &ReplayRequest {
                map: "tenderIn".into(),
                inputs: vec![ReplayInput {
                    id: "m-1".into(),
                    input: RunInput::X12(sample),
                    lookups: Vec::new(),
                    now: None,
                }],
                settings: std::collections::BTreeMap::new(),
            },
        );
        assert_eq!(report.summary.differ, 1);
        assert_eq!(report.summary.fields.get("stops[].name"), Some(&2));
        let difference = &report.items[0].differences[0];
        assert_eq!(difference.before, "Acme Dc");
        assert_eq!(difference.after, "ACME DC");
        assert_eq!(fold("a[1].b[22].c"), "a[].b[].c");
    }
}
