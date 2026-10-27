//! Signature help, go to definition, find references and rename.

use trenova_module_kit::message;
use trenova_module_kit::message::{Arg, Message};
use trenova_module_kit::span::Span;

use super::describe;
use super::session::{Doc, Session};
use crate::lading::check::Decl;
use crate::lading::facts::Target;
use crate::lading::stdlib;
use crate::lading::syntax::ast::{Expr, ExprKind, GuideDecl, Item, MemberKind, Stmt};
use crate::lading::syntax::token::is_reserved;

/// One signature.
#[derive(Debug, Clone, PartialEq, Eq)]
pub struct Signature {
    /// The whole label.
    pub label: String,
    /// Its doc.
    pub doc: Option<Doc>,
    /// Each parameter's range in the label, in bytes.
    pub parameters: Vec<(u32, u32)>,
    /// The parameter the cursor is at.
    pub active: Option<usize>,
}

/// The call around an offset and the argument it is at.
fn call_at<'e>(stmts_or_exprs: &[&'e Expr], offset: u32) -> Option<(&'e Expr, usize)> {
    let mut best: Option<(&Expr, usize)> = None;
    for expr in stmts_or_exprs {
        visit(expr, &mut |candidate| {
            if let ExprKind::Call { callee, args } = &candidate.kind
                && callee.span.end <= offset
                && offset <= candidate.span.end
            {
                let active = args
                    .iter()
                    .filter(|arg| arg.value.span.end < offset)
                    .count();
                let narrower = best.is_none_or(|(current, _)| {
                    candidate.span.end - candidate.span.start
                        < current.span.end - current.span.start
                });
                if narrower {
                    best = Some((candidate, active));
                }
            }
        });
    }
    best
}

/// Calls `found` on an expression and every expression inside it.
pub fn visit<'e>(expr: &'e Expr, found: &mut impl FnMut(&'e Expr)) {
    found(expr);
    match &expr.kind {
        ExprKind::Member { target, .. } | ExprKind::Exists(target) | ExprKind::Unary(_, target) => {
            visit(target, found);
        }
        ExprKind::Index { target, index } => {
            visit(target, found);
            visit(index, found);
        }
        ExprKind::Call { callee, args } => {
            visit(callee, found);
            for arg in args {
                visit(&arg.value, found);
            }
        }
        ExprKind::Binary(_, lhs, rhs) => {
            visit(lhs, found);
            visit(rhs, found);
        }
        ExprKind::Via { value, .. } => visit(value, found),
        ExprKind::Lookup { code, learn, .. } => {
            visit(code, found);
            if let Some(learn) = learn {
                visit(learn, found);
            }
        }
        ExprKind::Match {
            value,
            arms,
            otherwise,
        } => {
            visit(value, found);
            for arm in arms {
                for pattern in &arm.patterns {
                    visit(pattern, found);
                }
                visit(&arm.value, found);
            }
            if let Some(otherwise) = otherwise {
                visit(otherwise, found);
            }
        }
        ExprKind::If {
            cond,
            then,
            otherwise,
        } => {
            visit(cond, found);
            visit(then, found);
            visit(otherwise, found);
        }
        ExprKind::Lambda { body, .. } => visit(body, found),
        ExprKind::Each { list, body, .. } => {
            visit(list, found);
            for stmt in &body.stmts {
                stmt_exprs(stmt, found);
            }
        }
        ExprKind::List(items) => {
            for item in items {
                visit(item, found);
            }
        }
        ExprKind::Map(entries) => {
            for (key, value) in entries {
                visit(key, found);
                visit(value, found);
            }
        }
        ExprKind::Record { fields, .. } => {
            for field in fields {
                visit(&field.value, found);
            }
        }
        ExprKind::Text(parts) => {
            for part in parts {
                if let crate::lading::syntax::ast::TextPart::Expr(inner) = part {
                    visit(inner, found);
                }
            }
        }
        _ => {}
    }
}

/// Calls `found` on every expression of a statement.
pub fn stmt_exprs<'e>(stmt: &'e Stmt, found: &mut impl FnMut(&'e Expr)) {
    match stmt {
        Stmt::Assign {
            value, otherwise, ..
        } => {
            visit(value, found);
            if let Some(crate::lading::syntax::ast::Else::Value(default)) = otherwise {
                visit(default, found);
            }
        }
        Stmt::Let(binding) => visit(&binding.value, found),
        Stmt::When { cond, body, .. } => {
            visit(cond, found);
            for inner in &body.stmts {
                stmt_exprs(inner, found);
            }
        }
        Stmt::Segment { cond, elements, .. } => {
            if let Some(cond) = cond {
                visit(cond, found);
            }
            for element in elements {
                visit(&element.value, found);
            }
        }
        Stmt::Loop { each, body, .. } => {
            if let Some((list, _)) = each {
                visit(list, found);
            }
            for inner in &body.stmts {
                stmt_exprs(inner, found);
            }
        }
        Stmt::Each { list, body, .. } | Stmt::For { list, body, .. } => {
            visit(list, found);
            for inner in &body.stmts {
                stmt_exprs(inner, found);
            }
        }
        Stmt::Require { cond, outcome, .. } | Stmt::OutcomeWhen { cond, outcome, .. } => {
            visit(cond, found);
            visit(&outcome.message, found);
        }
        Stmt::Fix { value, .. } => visit(value, found),
        Stmt::Error(_) => {}
    }
}

/// Every top-level expression of a file.
#[must_use]
pub fn file_exprs(items: &[Item]) -> Vec<&Expr> {
    let mut out = Vec::new();
    for item in items {
        match item {
            Item::Fn(decl) => {
                for binding in &decl.body.lets {
                    out.push(&binding.value);
                }
                out.push(&decl.body.value);
            }
            Item::Const(decl) => out.push(&decl.value),
            Item::Stmt(stmt) => stmt_exprs(stmt, &mut |expr| out.push(expr)),
            Item::Rule(rule) => {
                for stmt in &rule.body.stmts {
                    stmt_exprs(stmt, &mut |expr| out.push(expr));
                }
            }
            Item::Test(test) => {
                for (step, _) in &test.steps {
                    if let crate::lading::syntax::ast::TestStep::Expect(expr) = step {
                        out.push(expr);
                    }
                }
            }
            _ => {}
        }
    }
    out
}

/// The signature help at a byte offset.
#[must_use]
pub fn signature(session: &Session, offset: u32) -> Option<Signature> {
    let program = &session.program;
    let exprs = file_exprs(program.units[session.unit].items());
    let (call, active) = call_at(&exprs, offset)?;
    let ExprKind::Call { callee, .. } = &call.kind else {
        return None;
    };
    let (name, method) = match &callee.kind {
        ExprKind::Name(name) => (name.clone(), false),
        ExprKind::Member {
            member,
            kind: MemberKind::Name,
            ..
        } => (member.name.clone(), true),
        _ => return None,
    };
    let shift = usize::from(method);
    if let Some((unit, Decl::Fn(_))) = program.decl(session.unit, &name) {
        let sig = program.fns.get(&(unit, name.clone()))?;
        let label = describe::signature(&name, sig);
        let doc = describe::decl(program, unit, &name).and_then(|(_, doc)| doc);
        return Some(Signature {
            parameters: parameter_ranges(&label),
            label,
            doc,
            active: Some(active + shift),
        });
    }
    let function = stdlib::named(&name).next()?;
    let label = function.signature.to_owned();
    Some(Signature {
        parameters: parameter_ranges(&label),
        label,
        doc: Some(Doc::message(function.doc)),
        active: Some(active + shift),
    })
}

/// The byte ranges of the parameters between the parentheses of a signature label.
fn parameter_ranges(label: &str) -> Vec<(u32, u32)> {
    let Some(open) = label.find('(') else {
        return Vec::new();
    };
    let mut ranges = Vec::new();
    let mut depth = 0;
    let mut start = open + 1;
    for (index, c) in label.char_indices().skip(open + 1) {
        match c {
            '(' | '[' | '{' | '<' => depth += 1,
            ')' if depth == 0 => {
                if index > start {
                    ranges.push((
                        u32::try_from(start).unwrap_or(0),
                        u32::try_from(index).unwrap_or(0),
                    ));
                }
                break;
            }
            ')' | ']' | '}' | '>' => depth -= 1,
            ',' if depth == 0 => {
                ranges.push((
                    u32::try_from(start).unwrap_or(0),
                    u32::try_from(index).unwrap_or(0),
                ));
                start = index + 2;
            }
            _ => {}
        }
    }
    ranges
}

/// Where a definition is: a file and a span in it.
#[derive(Debug, Clone, PartialEq, Eq)]
pub struct Location {
    /// The file.
    pub unit: usize,
    /// The declaration.
    pub span: Span,
}

/// Where what is under the cursor is declared.
#[must_use]
pub fn definition(session: &Session, offset: u32) -> Vec<Location> {
    let program = &session.program;
    let Some(occurrence) = program.facts.occurrence_at(session.unit, offset) else {
        return Vec::new();
    };
    let declared = |target: &Target| {
        program
            .facts
            .occurrences
            .iter()
            .filter(|item| item.declaration && &item.target == target)
            .map(|item| Location {
                unit: item.unit,
                span: item.span,
            })
            .collect::<Vec<_>>()
    };
    match &occurrence.target {
        Target::Local { unit, span } => vec![Location {
            unit: *unit,
            span: *span,
        }],
        Target::Decl { .. } | Target::File(_) | Target::Setting(_) => declared(&occurrence.target),
        Target::Segment(reference)
        | Target::Loop(reference)
        | Target::Element {
            node: reference, ..
        } => {
            let Some(unit) = program.guides[reference.guide].unit else {
                return Vec::new();
            };
            let path = describe::node_path(program, *reference);
            program.units[unit]
                .items()
                .iter()
                .filter_map(|item| match item {
                    Item::Guide(
                        GuideDecl::Segment {
                            path: written,
                            span,
                            ..
                        }
                        | GuideDecl::Loop {
                            path: written,
                            span,
                            ..
                        },
                    ) => {
                        let joined: Vec<&str> =
                            written.iter().map(|ident| ident.name.as_str()).collect();
                        (joined.join(".") == path).then_some(Location { unit, span: *span })
                    }
                    _ => None,
                })
                .collect()
        }
        _ => Vec::new(),
    }
}

/// One reference.
#[derive(Debug, Clone, PartialEq, Eq)]
pub struct Reference {
    /// The file.
    pub unit: usize,
    /// Where.
    pub span: Span,
    /// `declaration`, `read` or `write`.
    pub kind: &'static str,
}

/// Every use of what is under the cursor, across the project.
#[must_use]
pub fn references(session: &Session, offset: u32, include_declaration: bool) -> Vec<Reference> {
    let program = &session.program;
    let Some(occurrence) = program.facts.occurrence_at(session.unit, offset) else {
        return Vec::new();
    };
    let mut found: Vec<Reference> = program
        .facts
        .occurrences_of(&occurrence.target)
        .filter(|item| include_declaration || !item.declaration)
        .map(|item| Reference {
            unit: item.unit,
            span: item.span,
            kind: if item.declaration {
                "declaration"
            } else if item.write {
                "write"
            } else {
                "read"
            },
        })
        .collect();
    found.sort_by_key(|item| (item.unit, item.span.start));
    found.dedup();
    found
}

/// What rename would change, or why it cannot.
#[derive(Debug, Clone, PartialEq, Eq)]
pub enum Renamed {
    /// The edits: file, span, new text.
    Edits(Vec<(usize, Span, String)>),
    /// What can be renamed here, and the name now.
    Prepared(Span, String),
    /// Why not.
    Refused(Message),
}

/// Renames what is under the cursor; without a new name, says what would be renamed.
#[must_use]
pub fn rename(session: &Session, offset: u32, new_name: Option<&str>) -> Renamed {
    let program = &session.program;
    let Some(occurrence) = program.facts.occurrence_at(session.unit, offset) else {
        return Renamed::Refused(Message::new(message!("There is nothing to rename here")));
    };
    let renamable = matches!(
        occurrence.target,
        Target::Local { .. } | Target::Decl { .. } | Target::File(_) | Target::Setting(_)
    );
    if !renamable {
        return Renamed::Refused(Message::new(message!(
            "Only names the project declares can be renamed: files, functions, crosswalks, constants, settings and locals"
        )));
    }
    let current = session.text(occurrence.span).to_owned();
    let Some(new_name) = new_name else {
        return Renamed::Prepared(occurrence.span, current);
    };
    let valid = new_name
        .chars()
        .next()
        .is_some_and(|c| c.is_ascii_lowercase())
        && new_name.chars().all(|c| c.is_ascii_alphanumeric())
        && !is_reserved(new_name);
    if !valid {
        return Renamed::Refused(Message::with(
            message!(
                "{0} is not a name: start with a small letter, use letters and digits, and avoid keywords"
            ),
            [Arg::from(new_name)],
        ));
    }
    let taken = match &occurrence.target {
        Target::File(_) => program.unit_named(new_name).is_some(),
        Target::Decl { unit, .. } => program.units[*unit].decls.contains_key(new_name),
        Target::Setting(_) => program.partner.as_ref().is_some_and(|partner| {
            partner
                .settings
                .iter()
                .any(|setting| setting.name == new_name)
        }),
        _ => false,
    };
    if taken {
        return Renamed::Refused(Message::with(
            message!("{0} is already a name here"),
            [Arg::from(new_name)],
        ));
    }
    let mut edits: Vec<(usize, Span, String)> = program
        .facts
        .occurrences_of(&occurrence.target)
        .map(|item| (item.unit, item.span, new_name.to_owned()))
        .collect();
    if let Target::Setting(setting) = &occurrence.target {
        for (unit, source_unit) in program.units.iter().enumerate() {
            let pattern = format!("partner.{setting}");
            let mut start = 0;
            while let Some(found) = source_unit.source[start..].find(&pattern) {
                let at = start + found + "partner.".len();
                let end = at + setting.len();
                let boundary = source_unit.source[end..]
                    .chars()
                    .next()
                    .is_none_or(|c| !c.is_alphanumeric());
                if boundary {
                    let span = Span::new(
                        u32::try_from(at).unwrap_or(0),
                        u32::try_from(end).unwrap_or(0),
                    );
                    if !edits
                        .iter()
                        .any(|(edit_unit, edit_span, _)| *edit_unit == unit && *edit_span == span)
                    {
                        edits.push((unit, span, new_name.to_owned()));
                    }
                }
                start = end;
            }
        }
    }
    edits.sort_by_key(|(unit, span, _)| (*unit, span.start));
    edits.dedup();
    Renamed::Edits(edits)
}
