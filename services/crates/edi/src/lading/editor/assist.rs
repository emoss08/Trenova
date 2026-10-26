//! Inlay hints and refactorings.

use trenova_module_kit::message;
use trenova_module_kit::message::Message;
use trenova_module_kit::span::Span;

use super::hover::{RunValue, last_step};
use super::navigate::file_exprs;
use super::session::Session;
use crate::lading::facts::Target;
use crate::lading::syntax::ast::{ExprKind, Header, Item, Stmt, TextPart};
use crate::lading::types::Type;

/// One inlay hint.
#[derive(Debug, Clone, PartialEq, Eq)]
pub struct Hint {
    /// Where it is shown, in bytes.
    pub offset: u32,
    /// What it says.
    pub label: String,
    /// `type` or `value`.
    pub kind: &'static str,
}

/// The hints in a byte range: types of `let`s, and what each assignment produced in the last
/// run.
#[must_use]
pub fn inlays(session: &Session, range: Span, values: &[RunValue]) -> Vec<Hint> {
    let program = &session.program;
    let mut hints = Vec::new();
    let mut lets = Vec::new();
    for item in program.units[session.unit].items() {
        match item {
            Item::Stmt(stmt) => collect_lets(stmt, &mut lets),
            Item::Fn(decl) => lets.extend(
                decl.body
                    .lets
                    .iter()
                    .map(|binding| (binding.name.span, binding.value.span)),
            ),
            _ => {}
        }
    }
    for (name, value) in lets {
        if name.start < range.start || name.end > range.end {
            continue;
        }
        if let Some(typed) = program
            .facts
            .types
            .iter()
            .find(|typed| typed.unit == session.unit && typed.span == value)
        {
            hints.push(Hint {
                offset: name.end,
                label: format!(": {}", typed.ty),
                kind: "type",
            });
        }
    }
    if !values.is_empty() {
        for occurrence in program
            .facts
            .occurrences
            .iter()
            .filter(|item| item.unit == session.unit && item.write)
        {
            if occurrence.span.start < range.start || occurrence.span.end > range.end {
                continue;
            }
            let wanted = match &occurrence.target {
                Target::Field { field, .. } => (*field).to_owned(),
                Target::Element { node, position } => format!(
                    "{}{position:02}",
                    crate::lading::check::x12::segment_id(
                        &program.guides[node.guide].compiled,
                        node.node
                    )
                ),
                _ => continue,
            };
            let Some(found) = values.iter().find(|value| last_step(&value.path) == wanted) else {
                continue;
            };
            let line_end = session.source()[occurrence.span.end as usize..]
                .find('\n')
                .map_or(session.source().len(), |at| {
                    occurrence.span.end as usize + at
                });
            let shown: String = found.value.chars().take(60).collect();
            hints.push(Hint {
                offset: u32::try_from(line_end).unwrap_or(occurrence.span.end),
                label: format!("= {shown}"),
                kind: "value",
            });
        }
    }
    hints.sort_by_key(|hint| hint.offset);
    hints.dedup_by(|a, b| a.offset == b.offset && a.kind == b.kind);
    hints
}

fn collect_lets(stmt: &Stmt, out: &mut Vec<(Span, Span)>) {
    match stmt {
        Stmt::Let(binding) => out.push((binding.name.span, binding.value.span)),
        Stmt::When { body, .. }
        | Stmt::Loop { body, .. }
        | Stmt::Each { body, .. }
        | Stmt::For { body, .. } => {
            for inner in &body.stmts {
                collect_lets(inner, out);
            }
        }
        Stmt::Assign { value, .. } => {
            if let ExprKind::Each { body, .. } = &value.kind {
                for inner in &body.stmts {
                    collect_lets(inner, out);
                }
            }
        }
        _ => {}
    }
}

/// One edit of a refactoring: a file, a span in it (empty to insert), the new text.
#[derive(Debug, Clone, PartialEq, Eq)]
pub struct Edit {
    /// The file's path; a path not in the project creates the file.
    pub path: String,
    /// What it replaces.
    pub span: Span,
    /// The text.
    pub insert: String,
}

/// A refactoring.
#[derive(Debug, Clone, PartialEq, Eq)]
pub struct Action {
    /// A stable identifier.
    pub id: &'static str,
    /// What it does.
    pub title: Message,
    /// Its edits.
    pub edits: Vec<Edit>,
}

/// The refactorings for a selection.
#[must_use]
pub fn actions(session: &Session, selection: Span) -> Vec<Action> {
    let mut found = Vec::new();
    if let Some(action) = promote_to_setting(session, selection) {
        found.push(action);
    }
    if let Some(action) = extract_function(session, selection) {
        found.push(action);
    }
    if let Some(action) = extract_crosswalk(session, selection) {
        found.push(action);
    }
    if let Some(action) = move_to_library(session, selection) {
        found.push(action);
    }
    found
}

/// The expression exactly at a selection (spaces around it ignored).
fn selected(session: &Session, selection: Span) -> Option<&crate::lading::syntax::ast::Expr> {
    let source = session.source();
    let text = source.get(selection.start as usize..selection.end as usize)?;
    let leading = u32::try_from(text.len() - text.trim_start().len()).ok()?;
    let trailing = u32::try_from(text.len() - text.trim_end().len()).ok()?;
    let wanted = Span::new(selection.start + leading, selection.end - trailing);
    let mut best = None;
    for expr in file_exprs(session.program.units[session.unit].items()) {
        super::navigate::visit(expr, &mut |candidate| {
            if candidate.span == wanted {
                best = Some(candidate);
            }
        });
    }
    best
}

fn end_insert(session: &Session, text: &str) -> Edit {
    let source = session.source();
    let end = u32::try_from(source.len()).unwrap_or(0);
    let separator = if source.ends_with("\n\n") {
        ""
    } else if source.ends_with('\n') {
        "\n"
    } else {
        "\n\n"
    };
    Edit {
        path: session.program.units[session.unit].path.clone(),
        span: Span::new(end, end),
        insert: format!("{separator}{text}\n"),
    }
}

fn promote_to_setting(session: &Session, selection: Span) -> Option<Action> {
    let expr = selected(session, selection)?;
    let ty = match &expr.kind {
        ExprKind::Number(text) if text.contains('.') => "Decimal".to_owned(),
        ExprKind::Number(_) => "Int".to_owned(),
        ExprKind::Quantity(_, unit) => match crate::lading::eval::value::Unit::named(unit) {
            Some(found) => found.dim.name().to_owned(),
            None if matches!(unit.as_str(), "s" | "min" | "h" | "d") => "Duration".to_owned(),
            None => return None,
        },
        ExprKind::Money(_) => "Money".to_owned(),
        ExprKind::Percent(_) => "Percent".to_owned(),
        ExprKind::Bool(_) => "Bool".to_owned(),
        ExprKind::Text(parts)
            if parts
                .iter()
                .all(|part| matches!(part, TextPart::Literal(_))) =>
        {
            "Text".to_owned()
        }
        _ => return None,
    };
    let program = &session.program;
    let partner = program.partner.as_ref()?;
    let partner_path = program.units[partner.unit].path.clone();
    let partner_source = &program.units[partner.unit].source;
    let mut name = "newSetting".to_owned();
    let mut counter = 2;
    while partner.settings.iter().any(|setting| setting.name == name) {
        name = format!("newSetting{counter}");
        counter += 1;
    }
    let literal = session.text(expr.span).to_owned();
    let end = u32::try_from(partner_source.len()).unwrap_or(0);
    let prefix = if partner_source.ends_with('\n') {
        ""
    } else {
        "\n"
    };
    Some(Action {
        id: "promote-to-setting",
        title: Message::new(message!("Promote to a partner setting")),
        edits: vec![
            Edit {
                path: session.program.units[session.unit].path.clone(),
                span: expr.span,
                insert: format!("partner.{name}"),
            },
            Edit {
                path: partner_path,
                span: Span::new(end, end),
                insert: format!(
                    "{prefix}\n/// What this value is for.\nsetting {name}: {ty} = {literal}\n"
                ),
            },
        ],
    })
}

fn extract_function(session: &Session, selection: Span) -> Option<Action> {
    let expr = selected(session, selection)?;
    if matches!(
        expr.kind,
        ExprKind::Name(_)
            | ExprKind::Number(_)
            | ExprKind::Text(_)
            | ExprKind::Error
            | ExprKind::Lambda { .. }
            | ExprKind::Each { .. }
    ) {
        return None;
    }
    let program = &session.program;
    let ty = program
        .facts
        .types
        .iter()
        .find(|typed| typed.unit == session.unit && typed.span == expr.span)
        .map(|typed| typed.ty.clone())?;
    if matches!(ty, Type::Any)
        || program.facts.occurrences.iter().any(|item| {
            item.unit == session.unit
                && item.span.start >= expr.span.start
                && item.span.end <= expr.span.end
                && matches!(
                    item.target,
                    Target::Segment(_)
                        | Target::Element { .. }
                        | Target::Loop(_)
                        | Target::Field { .. }
                )
                && !item.write
                && program.units[session.unit].header().kind() == "map"
        })
    {
        return None;
    }
    let mut params: Vec<(String, Type)> = Vec::new();
    for occurrence in program.facts.occurrences.iter().filter(|item| {
        item.unit == session.unit
            && item.span.start >= expr.span.start
            && item.span.end <= expr.span.end
            && !item.declaration
    }) {
        if let Target::Local { span, .. } = occurrence.target
            && (span.start < expr.span.start || span.end > expr.span.end)
        {
            let name = session.text(occurrence.span).to_owned();
            let local_ty = program
                .facts
                .types
                .iter()
                .find(|typed| typed.unit == session.unit && typed.span == occurrence.span)
                .map_or(Type::Any, |typed| typed.ty.clone());
            if !params.iter().any(|(existing, _)| *existing == name) {
                params.push((name, local_ty));
            }
        }
    }
    let mut name = "extracted".to_owned();
    let mut counter = 2;
    while program.units[session.unit].decls.contains_key(&name) {
        name = format!("extracted{counter}");
        counter += 1;
    }
    let signature: Vec<String> = params
        .iter()
        .map(|(param, param_ty)| format!("{param}: {param_ty}"))
        .collect();
    let arguments: Vec<&str> = params.iter().map(|(param, _)| param.as_str()).collect();
    let body = session.text(expr.span).to_owned();
    Some(Action {
        id: "extract-function",
        title: Message::new(message!("Extract a function")),
        edits: vec![
            Edit {
                path: program.units[session.unit].path.clone(),
                span: expr.span,
                insert: format!("{name}({})", arguments.join(", ")),
            },
            end_insert(
                session,
                &format!("fn {name}({}) -> {ty} = {body}", signature.join(", ")),
            ),
        ],
    })
}

fn extract_crosswalk(session: &Session, selection: Span) -> Option<Action> {
    let expr = selected(session, selection)?;
    let ExprKind::Match {
        value,
        arms,
        otherwise,
    } = &expr.kind
    else {
        return None;
    };
    let program = &session.program;
    let type_of = |span: Span| {
        program
            .facts
            .types
            .iter()
            .find(|typed| typed.unit == session.unit && typed.span == span)
            .map(|typed| typed.ty.unwrapped().clone())
    };
    let left = type_of(value.span).unwrap_or(Type::Text);
    let right = arms.first().and_then(|arm| match &arm.value.kind {
        ExprKind::Type(variant) => {
            crate::lading::schema::enum_of_variant(variant).map(|owner| Type::Enum(owner.name))
        }
        ExprKind::Text(_) => Some(Type::Text),
        ExprKind::Number(_) => Some(Type::Int),
        _ => None,
    })?;
    let mut lines = Vec::new();
    for arm in arms {
        let patterns: Vec<&str> = arm
            .patterns
            .iter()
            .map(|pattern| session.text(pattern.span))
            .collect();
        lines.push(format!(
            "  {} -> {}",
            patterns.join(", "),
            session.text(arm.value.span)
        ));
    }
    if let Some(otherwise) = otherwise {
        lines.push(format!("  else -> {}", session.text(otherwise.span)));
    }
    let mut name = "extractedCodes".to_owned();
    let mut counter = 2;
    while program.units[session.unit].decls.contains_key(&name) {
        name = format!("extractedCodes{counter}");
        counter += 1;
    }
    Some(Action {
        id: "extract-crosswalk",
        title: Message::new(message!("Extract a crosswalk")),
        edits: vec![
            Edit {
                path: program.units[session.unit].path.clone(),
                span: expr.span,
                insert: format!("{} via {name}", session.text(value.span)),
            },
            end_insert(
                session,
                &format!(
                    "crosswalk {name}: {left} <-> {right} {{\n{}\n}}",
                    lines.join("\n")
                ),
            ),
        ],
    })
}

fn move_to_library(session: &Session, selection: Span) -> Option<Action> {
    let program = &session.program;
    let header = program.units[session.unit].header();
    if matches!(
        header,
        Header::Library { .. } | Header::Partner { .. } | Header::Guide { .. }
    ) {
        return None;
    }
    let item = program.units[session.unit].items().iter().find(|item| {
        let (name, span) = match item {
            Item::Fn(decl) => (decl.name.span, decl.span),
            Item::Crosswalk(decl) => (decl.name.span, decl.span),
            Item::Const(decl) => (decl.name.span, decl.span),
            _ => return false,
        };
        (name.start <= selection.start && selection.end <= name.end)
            || (span.start == selection.start && span.end == selection.end)
    })?;
    let span = item.span();
    let source = session.source();
    let start = source[..span.start as usize]
        .rfind('\n')
        .map_or(0, |at| at + 1);
    let mut doc_start = start;
    while doc_start > 0 {
        let previous = source[..doc_start - 1].rfind('\n').map_or(0, |at| at + 1);
        if source[previous..doc_start].trim_start().starts_with("///") {
            doc_start = previous;
        } else {
            break;
        }
    }
    let text = source[doc_start..span.end as usize].to_owned();
    let (library_name, library_unit) = match program.libraries.iter().next() {
        Some((name, unit)) => (name.clone(), Some(*unit)),
        None => ("common".to_owned(), None),
    };
    let mut edits = vec![Edit {
        path: program.units[session.unit].path.clone(),
        span: Span::new(u32::try_from(doc_start).unwrap_or(0), span.end),
        insert: String::new(),
    }];
    match library_unit {
        Some(unit) => {
            let library_source = &program.units[unit].source;
            let end = u32::try_from(library_source.len()).unwrap_or(0);
            let prefix = if library_source.ends_with('\n') {
                "\n"
            } else {
                "\n\n"
            };
            edits.push(Edit {
                path: program.units[unit].path.clone(),
                span: Span::new(end, end),
                insert: format!("{prefix}{text}\n"),
            });
        }
        None => edits.push(Edit {
            path: "lib/common.lading".to_owned(),
            span: Span::new(0, 0),
            insert: format!("library common\n\n{text}\n"),
        }),
    }
    if !header
        .uses()
        .iter()
        .any(|library| library.name == library_name)
    {
        let name_end = header.name().map_or(0, |name| name.span.end);
        let header_line_end = source[name_end as usize..]
            .find("\n\n")
            .map_or(source.len(), |at| name_end as usize + at);
        let insert = if header.uses().is_empty() {
            format!("\n  use  {library_name}")
        } else {
            format!(", {library_name}")
        };
        edits.push(Edit {
            path: program.units[session.unit].path.clone(),
            span: Span::new(
                u32::try_from(header_line_end).unwrap_or(0),
                u32::try_from(header_line_end).unwrap_or(0),
            ),
            insert,
        });
    }
    Some(Action {
        id: "move-to-library",
        title: Message::new(message!("Move to a library")),
        edits,
    })
}
