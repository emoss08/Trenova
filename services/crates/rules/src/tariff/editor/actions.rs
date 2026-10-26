//! `tariff.actions`: the refactorings available for a selection. Today that is **Promote to
//! input**, which turns a literal into an `input` with that default, and **Promote all**, which
//! does the same for every equal literal of the file.
//!
//! - **Where.** The selection is, or lies inside, one literal a contract could set
//!   ([`super::literals`]) in a tariff's or condition's file-level statements and the blocks
//!   inside them. Nothing is offered in a library, a function, a test or property, a `match`
//!   pattern, an input's default or constraint, a `const`, a type or a misplaced declaration.
//! - **The declaration.** `input name: Type = literal`, the type worked out from the literal
//!   and the literal written as the selected one is, inserted as a whole line after the file's
//!   last `input`; with no input, after its `use`s, or after the header and its doc, with a
//!   blank line before it and after it, as the formatter keeps a group.
//! - **The name** comes from where the literal is written: a charge's `rate:` or `quantity:`
//!   option is the label in camelCase plus `Rate` or `Quantity`, its amount the label plus
//!   `Amount`, a `let x = …` value `xDefault`, a named argument its parameter; anything else
//!   `value`. A name already taken (by a declaration, an import, a local anywhere in the file,
//!   a schema root or a keyword) gets `2`, `3`, ….
//! - **Promote all** is offered when other literals of the same type and value (`$2.35/mi` and
//!   `$2.350/mi`) are written where promotion is allowed, and replaces them all; the default
//!   keeps the selected literal's text.
//! - **`select`** is the new name where the selected literal was, in the edited source, ready
//!   to rename.

use super::cursor::{Located, Node, Scopes, item_statements, line_end, locate};
use super::describe::type_text;
use super::literals::{is_literal, literal_type, literal_value};
use super::rename::Edit;
use super::session::Session;
use crate::tariff::check::describe::{quote, snippet};
use crate::tariff::syntax::ast::{Expr, ExprKind, FileKind, ItemKind, NodeId, StmtKind};
use crate::tariff::syntax::lexer::{classify, is_keyword};
use crate::tariff::syntax::program::ModuleId;
use crate::tariff::syntax::token::TokenKind;
use crate::tariff::types::Type;
use std::collections::BTreeSet;
use trenova_module_kit::message;
use trenova_module_kit::message::{Arg, Message};
use trenova_module_kit::span::{Range, Span};

/// One refactoring.
#[derive(Debug, Clone, PartialEq)]
pub struct Action {
    /// `promote-input` or `promote-input-all`.
    pub id: &'static str,
    /// What the editor shows.
    pub title: Message,
    /// The edits, in source order, none overlapping.
    pub edits: Vec<Edit>,
    /// What to select afterwards, in UTF-16 units of the edited source.
    pub select: Option<Range>,
}

/// The names a test reads besides the record, which an input must not take.
const RESERVED: &[&str] = &[
    "input", "tables", "bill", "total", "lines", "reviews", "applies",
];

/// The most suffixes tried for a free name.
const MAX_SUFFIX: u32 = 1_000;

/// The refactorings for the byte range `selection` of the root file.
#[must_use]
pub fn actions(session: &Session<'_>, selection: Span) -> Vec<Action> {
    if !matches!(session.kind(), FileKind::Tariff | FileKind::Condition) {
        return Vec::new();
    }
    let source = session.source();
    let selection = trimmed(source, selection);
    let scopes = Scopes::of(source, session.file());
    session.tick(scopes.exprs.len());
    let Some(literal) = selected(&scopes, selection) else {
        return Vec::new();
    };
    let Some(ty) = literal_type(session, literal).filter(|ty| nameable(session, ty)) else {
        return Vec::new();
    };
    let allowed = Allowed::of(session, &scopes);
    if !allowed.holds(literal.span) {
        return Vec::new();
    }
    let located = locate(source, session.file(), literal.span.start);
    session.tick(located.visited);
    let name = free_name(session, &scopes, &base_name(&located, literal));
    let promotion = Promotion {
        session,
        literal,
        declaration: format!(
            "input {name}: {} = {}",
            type_text(&ty),
            written(source, literal)
        ),
        name: &name,
    };
    let written_literal = written(source, literal);
    let mut out = vec![Action {
        id: "promote-input",
        title: Message::with(
            message!("Promote {0} to an input"),
            [quote(&written_literal)],
        ),
        edits: promotion.edits(&[literal]),
        select: promotion.select(&[literal]),
    }];
    let equal = equal_literals(session, &scopes, &allowed, literal, &ty);
    if equal.len() > 1 {
        out.push(Action {
            id: "promote-input-all",
            title: Message::with(
                message!("Promote all {1} uses of {0} to an input"),
                [quote(&written_literal), Arg::from(equal.len())],
            ),
            edits: promotion.edits(&equal),
            select: promotion.select(&equal),
        });
    }
    out
}

/// The selection without the spaces at either end.
fn trimmed(source: &str, selection: Span) -> Span {
    let text = snippet(source, selection);
    let start = text.len() - text.trim_start().len();
    let end = text.trim_end().len().max(start);
    let at = |offset: usize| {
        selection
            .start
            .saturating_add(u32::try_from(offset).unwrap_or(0))
    };
    Span::new(at(start), at(end))
}

/// The innermost literal that holds the selection. A negated literal (`-$5`) is taken whole,
/// unless the selection starts after its `-`.
fn selected<'a>(scopes: &Scopes<'a>, selection: Span) -> Option<&'a Expr> {
    let holds = |expr: &Expr| expr.span.start <= selection.start && selection.end <= expr.span.end;
    let innermost = scopes.find_last(|expr| holds(expr) && is_literal(expr))?;
    let negation = scopes
        .find_last(|expr| {
            matches!(&expr.kind, ExprKind::Unary { operand, .. } if operand.id == innermost.id)
        })
        .filter(|negation| is_literal(negation));
    Some(match negation {
        Some(negation)
            if selection.start == selection.end || selection.start <= negation.span.start =>
        {
            negation
        }
        _ => innermost,
    })
}

/// Whether an input of type `ty` can be declared in the root file: an enum must be one the
/// file names bare (a schema enum, its own or an imported one).
fn nameable(session: &Session<'_>, ty: &Type) -> bool {
    match ty {
        Type::Enum(enumeration) => {
            let name = enumeration
                .name
                .rsplit('.')
                .next()
                .unwrap_or(&enumeration.name);
            session.schema.enumeration(name).is_some()
                || session.program.resolve(ModuleId::ROOT, name).is_some()
        }
        _ => true,
    }
}

/// Where a literal may become an input: the file-level statements, less their `match`
/// patterns and misplaced declarations. Worked out once per request, so checking every equal
/// literal of a long file is a search, not a walk.
struct Allowed {
    /// The file-level statements, in source order.
    items: Vec<Span>,
    /// The patterns and misplaced declarations inside them.
    excluded: Vec<Span>,
}

impl Allowed {
    fn of(session: &Session<'_>, scopes: &Scopes<'_>) -> Self {
        let file = session.file();
        let items: Vec<Span> = file
            .items
            .iter()
            .filter(|item| matches!(item.kind, ItemKind::Stmt(_)))
            .map(|item| item.span)
            .collect();
        let mut excluded = Vec::new();
        for item in file
            .items
            .iter()
            .filter(|item| matches!(item.kind, ItemKind::Stmt(_)))
        {
            item_statements(item, &mut |stmt| match &stmt.kind {
                StmtKind::Match(found) => {
                    excluded.extend(found.arms.iter().map(|arm| arm.pattern.span));
                }
                StmtKind::Item(misplaced) => excluded.push(misplaced.span),
                _ => {}
            });
        }
        for expr in &scopes.exprs {
            if let ExprKind::Match { arms, .. } = &expr.kind {
                excluded.extend(arms.iter().map(|arm| arm.pattern.span));
            }
        }
        session.tick(scopes.exprs.len() + excluded.len());
        Self { items, excluded }
    }

    /// Whether the literal at `span` may become an input.
    fn holds(&self, span: Span) -> bool {
        let within = |outer: &Span| outer.start <= span.start && span.end <= outer.end;
        let at = self.items.partition_point(|item| item.end < span.end);
        self.items.get(at).is_some_and(within) && !self.excluded.iter().any(within)
    }
}

/// The literal as written, without the parentheses around it.
fn written(source: &str, literal: &Expr) -> String {
    let mut text = snippet(source, literal.span).trim();
    for _ in 0..literal.parens {
        text = text
            .strip_prefix('(')
            .and_then(|inner| inner.strip_suffix(')'))
            .map_or(text, str::trim);
    }
    super::describe::flat(text)
}

/// The name the literal's place suggests, before collisions.
fn base_name(located: &Located<'_>, literal: &Expr) -> String {
    let parent = located
        .chain
        .iter()
        .rposition(|node| matches!(node, Node::Expr(expr) if expr.id == literal.id))
        .and_then(|at| at.checked_sub(1))
        .and_then(|at| located.chain.get(at));
    let charge = located.stmt().and_then(|stmt| match &stmt.kind {
        StmtKind::Charge(charge) => Some(charge),
        _ => None,
    });
    match parent {
        Some(Node::Arg(arg)) if arg.value.id == literal.id => {
            let Some(name) = &arg.name else {
                return String::from("value");
            };
            match (charge, name.text.as_str()) {
                (Some(charge), "quantity" | "rate" | "minimum" | "maximum")
                    if charge.options.iter().any(|option| option.span == arg.span) =>
                {
                    let suffix = match name.text.as_str() {
                        "quantity" => "Quantity",
                        "rate" => "Rate",
                        "minimum" => "Minimum",
                        _ => "Maximum",
                    };
                    labelled(&charge.label, suffix)
                }
                _ => name.text.clone(),
            }
        }
        Some(Node::Stmt(stmt)) => match &stmt.kind {
            StmtKind::Charge(charge)
                if charge
                    .amount
                    .as_ref()
                    .is_some_and(|amount| amount.id == literal.id) =>
            {
                labelled(&charge.label, "Amount")
            }
            StmtKind::Let(binding) | StmtKind::Var(binding)
                if binding.value.id == literal.id && !binding.name.text.is_empty() =>
            {
                format!("{}Default", binding.name.text)
            }
            _ => String::from("value"),
        },
        _ => String::from("value"),
    }
}

/// A charge label in camelCase with a suffix: `Flat rate` and `Amount` give `flatRateAmount`;
/// a label with no letters gives the suffix alone, lowercased.
fn labelled(label: &crate::tariff::syntax::ast::Text, suffix: &str) -> String {
    let text: String = label
        .parts
        .iter()
        .map(|part| match part {
            crate::tariff::syntax::ast::TextPart::Literal(text, _) => text.as_str(),
            crate::tariff::syntax::ast::TextPart::Expr(_) => " ",
        })
        .collect();
    let mut out = String::new();
    for word in text
        .split(|c: char| !c.is_ascii_alphanumeric())
        .filter(|word| !word.is_empty())
    {
        let lower = word.to_ascii_lowercase();
        if out.is_empty() {
            out.push_str(lower.trim_start_matches(|c: char| c.is_ascii_digit()));
        } else {
            let mut chars = lower.chars();
            if let Some(first) = chars.next() {
                out.push(first.to_ascii_uppercase());
                out.push_str(chars.as_str());
            }
        }
    }
    if out.is_empty() {
        return suffix.to_ascii_lowercase();
    }
    out.push_str(suffix);
    out
}

/// `base`, or `base2`, `base3`, … when it is taken.
fn free_name(session: &Session<'_>, scopes: &Scopes<'_>, base: &str) -> String {
    let taken = |name: &str| {
        is_keyword(name)
            || classify(name) != TokenKind::Name
            || RESERVED.contains(&name)
            || session.schema.root(name).is_some()
            || session.program.resolve(ModuleId::ROOT, name).is_some()
            || session.root().aliases.contains_key(name)
            || scopes.locals.iter().any(|local| local.name == name)
    };
    session.tick(scopes.locals.len());
    if !taken(base) {
        return base.to_owned();
    }
    (2..MAX_SUFFIX)
        .map(|suffix| format!("{base}{suffix}"))
        .find(|name| !taken(name))
        .unwrap_or_else(|| format!("{base}{MAX_SUFFIX}"))
}

/// Every literal of the file equal to `literal` (same type, same value) where promotion is
/// allowed, the selected one included, in source order.
fn equal_literals<'a>(
    session: &Session<'_>,
    scopes: &Scopes<'a>,
    allowed: &Allowed,
    literal: &'a Expr,
    ty: &Type,
) -> Vec<&'a Expr> {
    let Some(value) = literal_value(literal, ty) else {
        return vec![literal];
    };
    let negated: BTreeSet<NodeId> = scopes
        .exprs
        .iter()
        .filter(|expr| is_literal(expr))
        .filter_map(|expr| match &expr.kind {
            ExprKind::Unary { operand, .. } => Some(operand.id),
            _ => None,
        })
        .collect();
    let mut out: Vec<&Expr> = Vec::new();
    for candidate in &scopes.exprs {
        if !is_literal(candidate)
            || out.last().is_some_and(|last| {
                last.span.start <= candidate.span.start && candidate.span.end <= last.span.end
            })
        {
            continue;
        }
        if candidate.id == literal.id {
            out.push(candidate);
            continue;
        }
        let Some(other) = literal_type(session, candidate).filter(|other| other == ty) else {
            continue;
        };
        if literal_value(candidate, &other).as_ref() != Some(&value) {
            continue;
        }
        session.tick(allowed.excluded.len());
        if allowed.holds(candidate.span) && !negated.contains(&candidate.id) {
            out.push(candidate);
        }
    }
    out
}

/// The edits of one promotion.
struct Promotion<'s, 'c, 'e> {
    session: &'s Session<'c>,
    literal: &'e Expr,
    declaration: String,
    name: &'s str,
}

impl Promotion<'_, '_, '_> {
    /// The declaration's insertion and each literal's replacement, in source order.
    fn edits(&self, literals: &[&Expr]) -> Vec<Edit> {
        let mut edits: Vec<Edit> = literals
            .iter()
            .map(|literal| Edit {
                span: literal.span,
                insert: self.name.to_owned(),
            })
            .collect();
        edits.push(self.insertion());
        edits.sort();
        edits
    }

    /// The new name where the selected literal was, in UTF-16 units of the edited source.
    fn select(&self, literals: &[&Expr]) -> Option<Range> {
        let utf16 = &self.session.root().utf16;
        let source = self.session.source();
        let length = |text: &str| i64::try_from(text.encode_utf16().count()).unwrap_or(0);
        let mut shift: i64 = 0;
        for edit in self.edits(literals) {
            if edit.span.start >= self.literal.span.start
                && !(edit.span.start == self.literal.span.start && edit.span.end == edit.span.start)
            {
                break;
            }
            shift += length(&edit.insert) - length(snippet(source, edit.span));
        }
        let from = i64::from(utf16.to_utf16(self.literal.span.start)).checked_add(shift)?;
        let from = u32::try_from(from).ok()?;
        let to = from.checked_add(u32::try_from(length(self.name)).ok()?)?;
        Some(Range { from, to })
    }

    /// The declaration inserted as whole lines: after the last input, or (with blank lines
    /// around it) after the `use`s, or the header and its doc.
    fn insertion(&self) -> Edit {
        let file = self.session.file();
        let source = self.session.source();
        let last_input = file
            .items
            .iter()
            .filter(|item| matches!(item.kind, ItemKind::Input(_)))
            .map(|item| item.span.end)
            .max();
        let grouped = last_input.is_some();
        let anchor = last_input.or_else(|| {
            let uses = file
                .items
                .iter()
                .filter(|item| matches!(item.kind, ItemKind::Use(_)))
                .map(|item| item.span.end);
            let header = file.header.as_ref().map(|header| header.span.end);
            let docs = file.docs.as_ref().map(|docs| docs.span.end);
            uses.chain(header).chain(docs).max()
        });
        let Some(anchor) = anchor else {
            return Edit {
                span: Span::new(0, 0),
                insert: format!("{}\n\n", self.declaration),
            };
        };
        let end = line_end(source, anchor);
        let newline = source
            .get(end as usize..)
            .and_then(|rest| rest.find('\n'))
            .and_then(|at| u32::try_from(at).ok());
        let Some(newline) = newline else {
            let gap = if grouped { "\n" } else { "\n\n" };
            return Edit {
                span: Span::new(end, end),
                insert: format!("{gap}{}", self.declaration),
            };
        };
        let next = end.saturating_add(newline).saturating_add(1);
        let rest = source.get(next as usize..).unwrap_or_default();
        let next_blank = rest
            .split('\n')
            .next()
            .is_some_and(|line| line.trim().is_empty());
        let insert = if grouped {
            format!("{}\n", self.declaration)
        } else if next_blank {
            format!("\n{}\n", self.declaration)
        } else {
            format!("\n{}\n\n", self.declaration)
        };
        Edit {
            span: Span::new(next, next),
            insert,
        }
    }
}
