//! Guide files: a partner's narrowing of a base guide, compiled to the X12 guide the reader
//! and the writer use.

use std::collections::BTreeSet;

use trenova_module_kit::diagnostic::{Diagnostic, nearest};
use trenova_module_kit::message;
use trenova_module_kit::message::{Arg, Message};
use trenova_module_kit::span::Span;

use super::{GuideEntry, Program};
use crate::lading::syntax::ast::{
    ElementRule, GuideDecl, GuideElement, Header, Ident, Item, StructureRule,
};
use crate::x12::finding::{FindingKind, Severity};
use crate::x12::guide::builtin;
use crate::x12::guide::check::{ProblemKind, check_guide};
use crate::x12::guide::compiled::CompiledGuide;
use crate::x12::guide::model::{Code, ElementDef, Guide, Node, SegmentDef, Usage};

/// Compiles every guide file of the project, bases first.
pub fn compile_guides(program: &mut Program) {
    let guide_units: Vec<usize> = program
        .units
        .iter()
        .enumerate()
        .filter(|(_, unit)| matches!(unit.header(), Header::Guide { .. }))
        .map(|(index, _)| index)
        .collect();
    let mut visiting = BTreeSet::new();
    for unit in guide_units {
        compile_unit(program, unit, &mut visiting);
    }
}

/// The index of a guide by name, loading a base guide on first use.
pub fn guide_index(
    program: &mut Program,
    name: &Ident,
    diagnostics: &mut Vec<Diagnostic>,
) -> Option<usize> {
    if let Some(index) = program.guide_names.get(&name.name) {
        return Some(*index);
    }
    if let Some(compiled) = builtin::by_id(&name.name) {
        return Some(add(program, name.name.clone(), compiled.clone(), None));
    }
    let mut diagnostic = Diagnostic::error(
        "unknown-guide",
        name.span,
        Message::with(
            message!("There is no guide named {0}"),
            [Arg::from(name.name.as_str())],
        ),
    );
    let names: Vec<String> = builtin::all()
        .iter()
        .map(|guide| guide.guide().id.clone())
        .chain(program.guide_names.keys().cloned())
        .collect();
    if let Some(close) = nearest(&name.name, names.iter().map(String::as_str)) {
        diagnostic = diagnostic.with_fix(super::replace_fix(name.span, close));
    }
    diagnostics.push(diagnostic);
    None
}

fn add(program: &mut Program, name: String, compiled: CompiledGuide, unit: Option<usize>) -> usize {
    let index = program.guides.len();
    program.guide_names.insert(name.clone(), index);
    program.guides.push(GuideEntry {
        name,
        compiled,
        unit,
    });
    index
}

fn compile_unit(
    program: &mut Program,
    unit: usize,
    visiting: &mut BTreeSet<usize>,
) -> Option<usize> {
    let Header::Guide { name, base, title } = program.units[unit].header().clone() else {
        return None;
    };
    if let Some(index) = program.guide_names.get(&name.name) {
        return Some(*index);
    }
    if !visiting.insert(unit) {
        program.units[unit].diagnostics.push(Diagnostic::error(
            "guide-cycle",
            base.span,
            Message::new(message!("This guide narrows itself through its bases")),
        ));
        return None;
    }
    let base_guide = if let Some(base_unit) = program
        .unit_named(&base.name)
        .filter(|other| matches!(program.units[*other].header(), Header::Guide { .. }))
    {
        let index = compile_unit(program, base_unit, visiting)?;
        program.guides[index].compiled.guide().clone()
    } else if let Some(compiled) = builtin::by_id(&base.name) {
        compiled.guide().clone()
    } else {
        let mut diagnostics = Vec::new();
        guide_index(program, &base, &mut diagnostics);
        program.units[unit].diagnostics.extend(diagnostics);
        return None;
    };
    let mut guide = base_guide.clone();
    guide.id.clone_from(&name.name);
    guide.derived_from = Some(base_guide.id.clone());
    if let Some(title) = &title {
        guide.name.clone_from(&title.value);
    }
    let mut spans: Vec<(String, Span)> = Vec::new();
    let mut diagnostics = Vec::new();
    let items = program.units[unit].items().to_vec();
    for item in &items {
        let Item::Guide(decl) = item else {
            if !matches!(item, Item::Test(_) | Item::Property(_) | Item::Error(_)) {
                diagnostics.push(Diagnostic::error(
                    "guide-item",
                    item.span(),
                    Message::new(message!("A guide file holds guide lines only")),
                ));
            }
            continue;
        };
        apply(&mut guide, decl, &mut spans, &mut diagnostics);
    }
    let header_span = name.span.cover(base.span);
    for problem in check_guide(&guide, Some(&base_guide)) {
        let span = spans
            .iter()
            .filter(|(path, _)| problem.path.starts_with(path.as_str()))
            .max_by_key(|(path, _)| path.len())
            .map_or(header_span, |(_, span)| *span);
        let code = match problem.kind {
            ProblemKind::Error => "guide-invalid",
            ProblemKind::Widening => "guide-widens",
        };
        diagnostics.push(Diagnostic::error(code, span, problem.message));
    }
    visiting.remove(&unit);
    program.units[unit].diagnostics.extend(diagnostics);
    let compiled = match CompiledGuide::compile(guide) {
        Ok(compiled) => compiled,
        Err(_) => CompiledGuide::compile(base_guide).ok()?,
    };
    Some(add(program, name.name.clone(), compiled, Some(unit)))
}

/// Where a guide line points in the structure.
fn locate<'g>(nodes: &'g mut [Node], path: &[Ident]) -> Option<&'g mut Node> {
    let (first, rest) = path.split_first()?;
    let found = nodes.iter_mut().find(|node| match node {
        Node::Segment(segment) => segment.segment == first.name && rest.is_empty(),
        Node::Loop(definition) => definition.id == first.name,
    })?;
    if rest.is_empty() {
        return Some(found);
    }
    match found {
        Node::Loop(definition) => locate(&mut definition.children, rest),
        Node::Segment(_) => None,
    }
}

fn structure_path(path: &[Ident]) -> String {
    let mut text = String::from("structure");
    for part in path {
        text.push('/');
        text.push_str(&part.name);
    }
    text
}

#[allow(clippy::too_many_lines)]
fn apply(
    guide: &mut Guide,
    decl: &GuideDecl,
    spans: &mut Vec<(String, Span)>,
    diagnostics: &mut Vec<Diagnostic>,
) {
    match decl {
        GuideDecl::Severity { kind, level, span } => {
            let camel = kebab_to_camel(&kind.name);
            let parsed: Option<FindingKind> =
                serde_json::from_value(serde_json::Value::String(camel)).ok();
            let Some(parsed) = parsed else {
                diagnostics.push(Diagnostic::error(
                    "unknown-finding",
                    kind.span,
                    Message::with(
                        message!("{0} is not a finding kind"),
                        [Arg::from(kind.name.as_str())],
                    ),
                ));
                return;
            };
            let severity = match level.name.as_str() {
                "error" => Severity::Error,
                "warning" => Severity::Warning,
                "info" => Severity::Info,
                _ => {
                    diagnostics.push(Diagnostic::error(
                        "unknown-severity",
                        level.span,
                        Message::new(message!("Expected error, warning or info")),
                    ));
                    return;
                }
            };
            guide.severities.insert(parsed, severity);
            spans.push((String::from("severities"), *span));
        }
        GuideDecl::Loop {
            path,
            rules,
            doc,
            span,
            ..
        } => {
            let at = structure_path(path);
            spans.push((at, *span));
            let Some(Node::Loop(definition)) = locate(&mut guide.structure, path) else {
                diagnostics.push(not_found(path, *span));
                return;
            };
            if let Some(doc) = doc {
                definition.notes.clone_from(doc);
            }
            for (rule, rule_span) in rules {
                match rule {
                    StructureRule::Required => definition.usage = Usage::Mandatory,
                    StructureRule::Unused => definition.usage = Usage::NotUsed,
                    StructureRule::Max(max) => definition.repeat = *max,
                    StructureRule::Min(min) => definition.min = *min,
                    StructureRule::Syntax(_) => diagnostics.push(Diagnostic::error(
                        "guide-syntax-on-loop",
                        *rule_span,
                        Message::new(message!("Syntax notes belong to segments")),
                    )),
                }
            }
        }
        GuideDecl::Segment {
            path,
            rules,
            elements,
            note,
            doc,
            span,
        } => {
            let at = structure_path(path);
            spans.push((at.clone(), *span));
            let segment_id = path
                .last()
                .map(|ident| ident.name.clone())
                .unwrap_or_default();
            let dictionary = guide.segments.get(&segment_id).cloned();
            let Some(Node::Segment(segment)) = locate(&mut guide.structure, path) else {
                diagnostics.push(not_found(path, *span));
                return;
            };
            if let Some(text) = doc.as_ref().or(note.as_ref().map(|note| &note.value)) {
                segment.notes.clone_from(text);
            }
            let mut definition = segment.definition.clone().or(dictionary);
            for (rule, _) in rules {
                match rule {
                    StructureRule::Required => segment.usage = Usage::Mandatory,
                    StructureRule::Unused => segment.usage = Usage::NotUsed,
                    StructureRule::Max(max) => segment.max = *max,
                    StructureRule::Min(min) => segment.min = *min,
                    StructureRule::Syntax(notes) => {
                        if let Some(definition) = definition.as_mut() {
                            for note in notes {
                                if !definition.rules.contains(&note.name) {
                                    definition.rules.push(note.name.clone());
                                }
                            }
                        }
                    }
                }
            }
            for element in elements {
                spans.push((format!("{at}/{:02}", element.position), element.span));
                if let Some(definition) = definition.as_mut() {
                    apply_element(definition, element, diagnostics);
                }
            }
            if !elements.is_empty()
                || rules
                    .iter()
                    .any(|(rule, _)| matches!(rule, StructureRule::Syntax(_)))
            {
                segment.definition = definition;
            }
        }
    }
}

fn not_found(path: &[Ident], span: Span) -> Diagnostic {
    let written: Vec<&str> = path.iter().map(|ident| ident.name.as_str()).collect();
    Diagnostic::error(
        "guide-unknown-path",
        span,
        Message::with(
            message!("The base guide has no {0}"),
            [Arg::from(written.join("."))],
        ),
    )
}

fn apply_element(
    definition: &mut SegmentDef,
    element: &GuideElement,
    diagnostics: &mut Vec<Diagnostic>,
) {
    let Some(target) = element
        .position
        .checked_sub(1)
        .and_then(|index| definition.elements.get_mut(index as usize))
    else {
        diagnostics.push(Diagnostic::error(
            "guide-unknown-element",
            element.span,
            Message::with(
                message!("The segment has no element {0}"),
                [Arg::from(format!("{:02}", element.position))],
            ),
        ));
        return;
    };
    let target: &mut ElementDef = match element.component {
        None => target,
        Some(component) => {
            let found = target.components.as_mut().and_then(|components| {
                component
                    .checked_sub(1)
                    .and_then(|index| components.get_mut(index as usize))
            });
            let Some(found) = found else {
                diagnostics.push(Diagnostic::error(
                    "guide-unknown-element",
                    element.span,
                    Message::new(message!("The element has no such component")),
                ));
                return;
            };
            found
        }
    };
    if let Some(note) = &element.note {
        target.notes.clone_from(&note.value);
    }
    for (rule, _) in &element.rules {
        match rule {
            ElementRule::Required => target.usage = Usage::Mandatory,
            ElementRule::Unused => target.usage = Usage::NotUsed,
            ElementRule::Codes(codes) => {
                let known = target.codes.clone().unwrap_or_default();
                target.codes = Some(
                    codes
                        .iter()
                        .map(|code| Code {
                            code: code.value.clone(),
                            description: known
                                .iter()
                                .find(|existing| existing.code == code.value)
                                .map(|existing| existing.description.clone())
                                .unwrap_or_default(),
                        })
                        .collect(),
                );
            }
            ElementRule::Length(min, max) => {
                target.min = *min;
                target.max = *max;
            }
            ElementRule::Repeat(repeat) => target.repeat = *repeat,
            ElementRule::Fixed(value) => target.fixed = Some(value.value.clone()),
        }
    }
}

fn kebab_to_camel(text: &str) -> String {
    let mut out = String::with_capacity(text.len());
    let mut upper = false;
    for character in text.chars() {
        if character == '-' {
            upper = true;
        } else if upper {
            out.extend(character.to_uppercase());
            upper = false;
        } else {
            out.push(character);
        }
    }
    out
}

#[cfg(test)]
mod tests {
    use super::kebab_to_camel;

    #[test]
    fn finding_names_read_in_kebab_case() {
        assert_eq!(kebab_to_camel("code-not-in-list"), "codeNotInList");
    }
}
