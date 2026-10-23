//! A `chosen("Label")` compared (`==`, `!=`, `in [...]`, `not in [...]`) with a label that is no
//! option of that choice is an `unknown-option` error, with the nearest option's label as a fix.
//! A choice the file doesn't have is `unknown-choice`, reported where `chosen` is checked.

use super::ModuleAnalysis;
use super::describe::quote;
use super::fixes::replace;
use crate::message;
use crate::message::Message;
use crate::span::Span;
use crate::tariff::diagnostic::{Diagnostic, nearest};
use crate::tariff::syntax::ast::{BinaryOp, Choose, Expr, ExprKind, Stmt, StmtKind};
use crate::tariff::syntax::program::Module;
use crate::tariff::syntax::walk::{self, Visit};
use std::collections::BTreeMap;

/// The function a test reads a choice's kept option with.
const CHOSEN: &str = "chosen";

/// Checks every comparison of `chosen(…)` with option labels.
pub fn check(module: &Module, out: &mut ModuleAnalysis) {
    let mut choices = Choices::default();
    walk::nodes(&module.file, &mut choices);
    if choices.options.is_empty() {
        return;
    }
    let mut labels = Labels {
        options: &choices.options,
        found: Vec::new(),
    };
    walk::nodes(&module.file, &mut labels);
    out.diagnostics.extend(labels.found);
}

/// Each choice's option labels, by the choice's label.
#[derive(Default)]
struct Choices {
    options: BTreeMap<String, Vec<String>>,
}

impl Choices {
    fn add(&mut self, choose: &Choose) {
        self.options
            .entry(choose.label.text.clone())
            .or_default()
            .extend(
                choose
                    .options
                    .iter()
                    .map(|option| option.label.text.clone()),
            );
    }
}

impl Visit for Choices {
    fn expr(&mut self, expr: &Expr) {
        if let ExprKind::Choose(choose) = &expr.kind {
            self.add(choose);
        }
    }

    fn stmt(&mut self, stmt: &Stmt) {
        if let StmtKind::Choose(choose) = &stmt.kind {
            self.add(choose);
        }
    }
}

/// The walk that judges the comparisons.
struct Labels<'a> {
    options: &'a BTreeMap<String, Vec<String>>,
    found: Vec<Diagnostic>,
}

impl Visit for Labels<'_> {
    fn expr(&mut self, expr: &Expr) {
        let ExprKind::Binary {
            op, left, right, ..
        } = &expr.kind
        else {
            return;
        };
        if !matches!(
            op,
            BinaryOp::Eq | BinaryOp::Ne | BinaryOp::In | BinaryOp::NotIn
        ) {
            return;
        }
        let (choice, other) = match (chosen(left), chosen(right)) {
            (Some(choice), _) => (choice, right),
            (None, Some(choice)) if matches!(op, BinaryOp::Eq | BinaryOp::Ne) => (choice, left),
            _ => return,
        };
        let Some(options) = self.options.get(&choice) else {
            return;
        };
        let written: Vec<(String, Span)> = match (&other.kind, op) {
            (ExprKind::List(items), BinaryOp::In | BinaryOp::NotIn) => {
                items.iter().filter_map(text).collect()
            }
            (_, BinaryOp::Eq | BinaryOp::Ne) => text(other).into_iter().collect(),
            _ => return,
        };
        for (label, span) in written {
            if options.contains(&label) {
                continue;
            }
            self.found.push(unknown(&choice, options, &label, span));
        }
    }
}

/// The choice label a `chosen("Label")` call reads.
fn chosen(expr: &Expr) -> Option<String> {
    let ExprKind::Call { callee, args } = &expr.kind else {
        return None;
    };
    let ExprKind::Name(name) = &callee.kind else {
        return None;
    };
    if name.text != CHOSEN {
        return None;
    }
    let [arg] = args.as_slice() else {
        return None;
    };
    text(&arg.value).map(|(label, _)| label)
}

/// A text literal's value and span.
fn text(expr: &Expr) -> Option<(String, Span)> {
    let ExprKind::Text(text) = &expr.kind else {
        return None;
    };
    Some((text.literal()?, expr.span))
}

fn unknown(choice: &str, options: &[String], label: &str, span: Span) -> Diagnostic {
    let listed: Vec<String> = options
        .iter()
        .map(|option| format!("\"{option}\""))
        .collect();
    match nearest(label, options.iter().map(String::as_str)) {
        Some(close) => {
            let written = format!("\"{close}\"");
            Diagnostic::error(
                "unknown-option",
                span,
                Message::with(
                    message!("The choice {0} has no option labelled {1}. Did you mean {2}?"),
                    [quote(choice), quote(label), quote(close)],
                ),
            )
            .with_fix(replace(
                Message::with(message!("Change to {0}"), [quote(&written)]),
                span,
                written,
            ))
        }
        None => Diagnostic::error(
            "unknown-option",
            span,
            Message::with(
                message!("The choice {0} has no option labelled {1}: its options are {2}"),
                [quote(choice), quote(label), quote(&listed.join(", "))],
            ),
        ),
    }
}
