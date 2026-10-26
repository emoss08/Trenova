//! Hover: what the name under the cursor is, its type, its doc, and the last run's value and
//! lineage when the studio passes them.

use serde::{Deserialize, Serialize};
use trenova_module_kit::span::Span;

use super::describe;
use super::session::{Doc, Session};
use crate::lading::facts::Target;
use crate::lading::schema;
use crate::lading::stdlib;

/// One value of the last run, as the studio passes it back: a lineage entry of `lading.run`.
#[derive(Debug, Clone, Deserialize, Serialize, PartialEq, Eq)]
#[serde(rename_all = "camelCase")]
pub struct RunValue {
    /// The output path.
    pub path: String,
    /// The value, as text.
    pub value: String,
    /// Where it came from, as paths.
    #[serde(default)]
    pub sources: Vec<String>,
    /// What it passed through.
    #[serde(default)]
    pub steps: Vec<String>,
}

/// What hover shows.
#[derive(Debug, Clone, PartialEq, Eq)]
pub struct Hovered {
    /// The name hovered.
    pub range: Span,
    /// Its declaration or label: `N104`, `fn window(...)`, `stops: [Stop]`.
    pub signature: String,
    /// Its type.
    pub ty: String,
    /// Its doc.
    pub doc: Option<Doc>,
    /// The last run's value and lineage.
    pub value: Option<RunValue>,
}

/// What hover shows at a byte offset.
#[must_use]
#[allow(clippy::too_many_lines)]
pub fn hover(session: &Session, offset: u32, values: &[RunValue]) -> Option<Hovered> {
    let program = &session.program;
    let occurrence = program.facts.occurrence_at(session.unit, offset);
    let typed = program.facts.type_at(session.unit, offset);
    let ty = typed.map(|typed| typed.ty.to_string()).unwrap_or_default();
    let Some(occurrence) = occurrence else {
        let typed = typed?;
        return Some(Hovered {
            range: typed.span,
            signature: session.text(typed.span).to_owned(),
            ty,
            doc: None,
            value: None,
        });
    };
    let name = session.text(occurrence.span).to_owned();
    let (signature, doc) = match &occurrence.target {
        Target::Field { record, field } => {
            let field_def = schema::record(record).and_then(|owner| owner.field(field));
            match field_def {
                Some(found) => {
                    let (detail, doc) = describe::field(program, found);
                    (format!("{record}.{detail}"), Some(doc))
                }
                None => (name.clone(), None),
            }
        }
        Target::Segment(reference) | Target::Loop(reference) => {
            let (detail, doc) = describe::node(program, *reference);
            (detail, Some(doc))
        }
        Target::Element { node, position } => match describe::element(program, *node, *position) {
            Some((label, detail, doc)) => (format!("{label} · {detail}"), Some(doc)),
            None => (name.clone(), None),
        },
        Target::Stdlib(function) => {
            let found = stdlib::named(function).next();
            (
                found.map_or_else(
                    || (*function).to_owned(),
                    |found| found.signature.to_owned(),
                ),
                found.map(|found| Doc::message(found.doc)),
            )
        }
        Target::Decl { unit, name } => describe::decl(program, *unit, name).unwrap_or_default(),
        Target::Local { .. } => (format!("{name}: {ty}"), None),
        Target::File(file) => (
            describe::file(program, file).unwrap_or_else(|| file.clone()),
            None,
        ),
        Target::Setting(setting) => describe::setting(program, setting).unwrap_or_default(),
        Target::Variant {
            owner,
            name: variant,
        } => {
            let doc = schema::enumeration(owner)
                .and_then(|item| {
                    item.variants
                        .iter()
                        .find(|(candidate, _)| candidate == variant)
                })
                .map(|(_, doc)| Doc::message(doc));
            (format!("{owner}.{variant}"), doc)
        }
        Target::Type(type_name) => {
            let doc = schema::record(type_name)
                .map(|record| record.doc)
                .or_else(|| schema::enumeration(type_name).map(|item| item.doc))
                .or_else(|| schema::lookup(type_name).map(|item| item.doc))
                .map(Doc::message);
            (type_name.clone(), doc)
        }
        Target::Event(event) => {
            let found = schema::event(event);
            (
                format!("{event} → {}", found.map_or("", |found| found.document)),
                found.map(|found| Doc::message(found.doc)),
            )
        }
        Target::Builtin(builtin) => (builtin.clone(), None),
    };
    let value = value_for(session, occurrence.span, &occurrence.target, values);
    Some(Hovered {
        range: occurrence.span,
        signature,
        ty,
        doc,
        value,
    })
}

/// The last run's value for an assigned field or written element.
fn value_for(
    session: &Session,
    span: Span,
    target: &Target,
    values: &[RunValue],
) -> Option<RunValue> {
    if values.is_empty() {
        return None;
    }
    let program = &session.program;
    let occurrence = program
        .facts
        .occurrences
        .iter()
        .find(|item| item.unit == session.unit && item.span == span)?;
    if !occurrence.write {
        return None;
    }
    let wanted = match target {
        Target::Field { field, .. } => (*field).to_owned(),
        Target::Element { node, position } => format!(
            "{}{position:02}",
            crate::lading::check::x12::segment_id(&program.guides[node.guide].compiled, node.node)
        ),
        _ => return None,
    };
    values
        .iter()
        .find(|value| last_step(&value.path) == wanted)
        .cloned()
}

/// The last part of a path, without list positions: `stops[1].name` is `name`, `0205[1].AT701`
/// is `AT701`.
#[must_use]
pub fn last_step(path: &str) -> String {
    let last = path.rsplit('.').next().unwrap_or(path);
    let mut out = String::new();
    let mut inside = false;
    for c in last.chars() {
        match c {
            '[' => inside = true,
            ']' => inside = false,
            _ if !inside => out.push(c),
            _ => {}
        }
    }
    out
}
