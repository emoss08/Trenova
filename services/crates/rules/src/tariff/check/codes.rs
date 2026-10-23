//! Charge codes the accessorial charge catalog doesn't have (`unknown-code`), judged once a
//! module is checked so that each code is reported once: at its first use, with every other use
//! as a related place. A code is used by `charge … as CODE`, `charge(CODE)` in a test and
//! `shipment.requested(CODE)`.
//!
//! The finding suggests a catalog code spelled nearly the same, or else one whose description
//! matches a charge's label ("`FUEL` is "Fuel surcharge""), with a fix that writes it at every
//! use. When every use is a `charge … as CODE`, a second fix bills those lines as freight by
//! removing `as CODE`. Codes are not judged when the host has not loaded the catalog.

use super::ModuleAnalysis;
use super::calls::{REQUESTED, is_code};
use super::describe::quote;
use super::fixes::changes;
use crate::message;
use crate::message::{Arg, Message};
use crate::span::Span;
use crate::tariff::context::Context;
use crate::tariff::diagnostic::{Change, Diagnostic, nearest};
use crate::tariff::schema::ChargeCode;
use crate::tariff::syntax::ast::{Expr, ExprKind, Stmt, StmtKind};
use crate::tariff::syntax::program::Module;
use crate::tariff::syntax::walk::{self, Visit};

/// One use of a code the catalog doesn't have.
struct Use {
    code: String,
    span: Span,
    /// For a `charge … as CODE`: the charge's label, and the span of ` as CODE`.
    charge: Option<(Option<String>, Span)>,
}

/// Reports every code the module bills or reads that the catalog doesn't have.
pub fn check(module: &Module, context: &Context, out: &mut ModuleAnalysis) {
    let Some(catalog) = &context.codes else {
        return;
    };
    let mut uses = Uses {
        catalog,
        found: Vec::new(),
    };
    walk::nodes(&module.file, &mut uses);
    let mut codes: Vec<&str> = Vec::new();
    for found in &uses.found {
        if !codes.contains(&found.code.as_str()) {
            codes.push(&found.code);
        }
    }
    let active: Vec<&ChargeCode> = catalog.iter().filter(|known| known.active).collect();
    for code in codes {
        let all: Vec<&Use> = uses
            .found
            .iter()
            .filter(|found| found.code == code)
            .collect();
        out.diagnostics.push(report(code, &all, &active));
    }
}

/// The walk that finds the uses.
struct Uses<'a> {
    catalog: &'a [ChargeCode],
    found: Vec<Use>,
}

impl Uses<'_> {
    fn unknown(&self, code: &str) -> bool {
        !self.catalog.iter().any(|known| known.code == code)
    }
}

impl Visit for Uses<'_> {
    fn expr(&mut self, expr: &Expr) {
        let ExprKind::Call { callee, args } = &expr.kind else {
            return;
        };
        let [arg] = args.as_slice() else {
            return;
        };
        let read = match &callee.kind {
            ExprKind::Name(name) => name.text == "charge",
            ExprKind::Field { name, .. } => name.text == REQUESTED,
            _ => false,
        };
        if !read {
            return;
        }
        if let ExprKind::Name(code) = &arg.value.kind
            && is_code(&code.text)
            && self.unknown(&code.text)
        {
            self.found.push(Use {
                code: code.text.clone(),
                span: code.span,
                charge: None,
            });
        }
    }

    fn stmt(&mut self, stmt: &Stmt) {
        let StmtKind::Charge(charge) = &stmt.kind else {
            return;
        };
        let Some(code) = &charge.code else {
            return;
        };
        if code.text.is_empty() || !self.unknown(&code.text) {
            return;
        }
        self.found.push(Use {
            code: code.text.clone(),
            span: code.span,
            charge: Some((
                charge.label.literal(),
                Span::new(charge.label.span.end, code.span.end),
            )),
        });
    }
}

/// The finding for one unknown code and all its uses.
fn report(code: &str, uses: &[&Use], active: &[&ChargeCode]) -> Diagnostic {
    let first = uses.first().map_or_else(Span::default, |found| found.span);
    let missing = Message::with(
        message!("{0} isn't in the accessorial charge catalog"),
        [quote(code)],
    );
    let spelled = nearest(code, active.iter().map(|known| known.code.as_str()));
    let meant = || {
        uses.iter()
            .filter_map(|found| found.charge.as_ref()?.0.as_deref())
            .find_map(|label| {
                active
                    .iter()
                    .find(|known| describes(&known.description, label))
            })
    };
    let (message, suggestion) = match (spelled, spelled.is_none().then(meant).flatten()) {
        (Some(close), _) => (
            Message::with(
                message!("{0}. Did you mean {1}?"),
                [Arg::from(missing), quote(close)],
            ),
            Some(close.to_owned()),
        ),
        (None, Some(known)) => (
            Message::with(
                message!("{0}. {1} is \"{2}\"; did you mean {1}?"),
                [
                    Arg::from(missing),
                    quote(&known.code),
                    Arg::from(known.description.trim()),
                ],
            ),
            Some(known.code.clone()),
        ),
        (None, None) => (missing, None),
    };
    let mut diagnostic = Diagnostic::error("unknown-code", first, message);
    for found in uses.iter().skip(1) {
        diagnostic = diagnostic.with_related(
            found.span,
            Message::with(message!("{0} is also used here"), [quote(code)]),
        );
    }
    if let Some(suggestion) = suggestion {
        let edits = uses
            .iter()
            .map(|found| Change {
                span: found.span,
                insert: suggestion.clone(),
            })
            .collect();
        let label = if uses.len() > 1 {
            Message::with(message!("Change every use to {0}"), [quote(&suggestion)])
        } else {
            Message::with(message!("Change to {0}"), [quote(&suggestion)])
        };
        diagnostic = diagnostic.with_fix(changes(label, edits));
    }
    if uses.iter().all(|found| found.charge.is_some()) {
        let edits = uses
            .iter()
            .filter_map(|found| found.charge.as_ref())
            .map(|(_, span)| Change {
                span: *span,
                insert: String::new(),
            })
            .collect();
        let label = Message::with(
            message!("Bill as a freight line instead: remove `as {0}`"),
            [Arg::from(code)],
        );
        diagnostic = diagnostic.with_fix(changes(label, edits));
    }
    diagnostic
}

/// Whether a catalog code's description names what a charge's label says: the same words, or
/// one's words within the other's, ignoring case and punctuation.
fn describes(description: &str, label: &str) -> bool {
    let words = |text: &str| -> Vec<String> {
        text.split(|c: char| !c.is_alphanumeric())
            .filter(|word| !word.is_empty())
            .map(str::to_lowercase)
            .collect()
    };
    let description = words(description);
    let label = words(label);
    if description.is_empty() || label.is_empty() {
        return false;
    }
    let within =
        |short: &[String], long: &[String]| long.windows(short.len()).any(|window| window == short);
    if description.len() <= label.len() {
        within(&description, &label)
    } else {
        within(&label, &description)
    }
}
