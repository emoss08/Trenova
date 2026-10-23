//! The bill so far, in a tariff's own code: `bill.freight`, `bill.accessorials` and
//! `bill.total` (`Money`), and `bill.charge("Label")` or `bill.charge(CODE)` (`Money?`).
//!
//! - `bill` is read in a tariff's top level and the blocks inside it, options included. A
//!   function is closed over its parameters, a test reads the finished bill (`total`, `lines`,
//!   `charge(…)`, `reviews`), a constant or default is worked out before rating, and a condition
//!   or a library has no bill: each is refused with its reason. `bill` alone is not a value.
//! - Its members are the fields of [`bill_type`] and `charge(…)`, which takes one charge code or
//!   label in quotes, like a test's `charge(…)`.
//! - A literal label or a code the tariff never bills is `not-billed`. One whose every charge is
//!   billed after the read is `bill-order`, naming the first of them. A charge reaches a read
//!   when it is written before it, or inside a `for` loop around it (an earlier iteration bills
//!   first). A charge inside an option of a choice that is around the read reaches it only from
//!   inside that same option; once a choice is over, any of its options may have been kept.
//!
//! [`sites`] collects every `charge` of the root file with the loops and options around it for
//! that judgement; the checker keeps them in [`super::checker::Billed::sites`].

use super::Callee;
use super::checker::{Body, Checker};
use super::describe::quote;
use super::fixes::replace;
use crate::message;
use crate::message::{Arg, Message};
use crate::span::Span;
use crate::tariff::diagnostic::Diagnostic;
use crate::tariff::schema::bill_type;
use crate::tariff::syntax::ast::{
    Arg as CallArg, ArmBody, Block, Choose, Expr, ExprKind, FileKind, Name, Stmt, StmtKind,
};
use crate::tariff::types::Type;

/// A `charge` of the root file and where it is written.
#[derive(Debug, Clone)]
pub struct ChargeSite {
    /// The label, when it does not interpolate.
    pub label: Option<String>,
    /// The code, when written.
    pub code: Option<String>,
    /// The `charge` statement.
    pub span: Span,
    /// The `for` statements around it.
    pub loops: Vec<Span>,
    /// The options around it, each with its choice: `(option, choice)`.
    pub options: Vec<(Span, Span)>,
}

impl ChargeSite {
    /// Whether this charge may have billed before code at `read` runs: it ends before the read,
    /// or a loop around both bills it on an earlier iteration, and it is not in another option
    /// of a choice around the read.
    fn reaches(&self, read: Span) -> bool {
        !self.in_sibling(read)
            && (self.span.end <= read.start || self.loops.iter().any(|at| contains(*at, read)))
    }

    /// Whether this charge is in an option of a choice around `read` that does not hold it.
    fn in_sibling(&self, read: Span) -> bool {
        self.options
            .iter()
            .any(|(option, choice)| contains(*choice, read) && !contains(*option, read))
    }
}

fn contains(outer: Span, inner: Span) -> bool {
    outer.start <= inner.start && inner.end <= outer.end
}

/// Every `charge` of a file's top-level statements, in source order, with the loops and
/// options around it.
#[must_use]
pub fn sites(stmts: &[&Stmt]) -> Vec<ChargeSite> {
    let mut walk = Sites::default();
    for stmt in stmts {
        walk.stmt(stmt);
    }
    walk.out
}

#[derive(Default)]
struct Sites {
    loops: Vec<Span>,
    options: Vec<(Span, Span)>,
    out: Vec<ChargeSite>,
}

impl Sites {
    fn block(&mut self, block: &Block) {
        for stmt in &block.stmts {
            self.stmt(stmt);
        }
    }

    fn stmt(&mut self, stmt: &Stmt) {
        match &stmt.kind {
            StmtKind::Charge(charge) => self.out.push(ChargeSite {
                label: charge.label.literal(),
                code: charge.code.as_ref().map(|code| code.text.clone()),
                span: stmt.span,
                loops: self.loops.clone(),
                options: self.options.clone(),
            }),
            StmtKind::If(statement) => {
                for (_, block) in &statement.branches {
                    self.block(block);
                }
                if let Some(block) = &statement.otherwise {
                    self.block(block);
                }
            }
            StmtKind::Match(statement) => {
                for arm in &statement.arms {
                    if let ArmBody::Block(block) = &arm.body {
                        self.block(block);
                    }
                }
            }
            StmtKind::For(statement) => {
                self.loops.push(stmt.span);
                self.block(&statement.body);
                self.loops.pop();
            }
            StmtKind::Choose(choose) => self.choose(choose),
            StmtKind::Let(binding) | StmtKind::Var(binding) => self.value(&binding.value),
            StmtKind::Assign { value, .. } => self.value(value),
            _ => {}
        }
    }

    fn value(&mut self, value: &Expr) {
        if let ExprKind::Choose(choose) = &value.kind {
            self.choose(choose);
        }
    }

    fn choose(&mut self, choose: &Choose) {
        for option in &choose.options {
            self.options.push((option.span, choose.span));
            self.block(&option.body);
            self.options.pop();
        }
    }
}

impl Checker<'_> {
    /// Whether an expression names the bill so far: `bill`, when nothing in scope or declared
    /// takes the name.
    pub fn is_bill(&self, target: &Expr) -> bool {
        matches!(&target.kind, ExprKind::Name(name) if name.text == "bill")
            && self.scopes.find("bill").is_none()
            && !self.globals.values.contains_key("bill")
    }

    /// Whether `bill` may be read where the checker is: a tariff's own code, reporting why not
    /// at `name` otherwise.
    fn bill_visible(&mut self, name: &Name) -> bool {
        let what = Message::new(message!("the bill being built"));
        match (&self.body, self.kind) {
            (Body::Top, FileKind::Tariff) => return true,
            (Body::Function { .. } | Body::Test | Body::Constant, _) => self.closed(name, what),
            (Body::Top, kind) => {
                let reason = if kind == FileKind::Library {
                    Message::new(message!("a library has no bill"))
                } else {
                    Message::new(message!(
                        "a condition decides true or false and bills nothing"
                    ))
                };
                self.report(Diagnostic::error(
                    "misplaced",
                    name.span,
                    Message::with(
                        message!(
                            "`bill` reads the bill being built, so it can't be read here: {0}"
                        ),
                        [Arg::from(reason)],
                    ),
                ));
            }
        }
        false
    }

    /// The `bill` name itself: recorded with its type for the editor, and visible only in a
    /// tariff's own code.
    fn bill_root(&mut self, target: &Expr, name: &Name) -> bool {
        self.record(target, Type::Record(bill_type()));
        if let Some(symbol) = self.root_symbol("bill") {
            self.reference(name.span, symbol);
        }
        self.bill_visible(name)
    }

    /// `bill` read on its own, which is not a value.
    fn bill_alone(&mut self, expr: &Expr, name: &Name) -> Type {
        if self.bill_root(expr, name) {
            self.report(Diagnostic::error(
                "not-a-value",
                name.span,
                Message::new(message!(
                    "`bill` is the bill so far: read a part of it, as in `bill.freight`, `bill.total` or `bill.charge(\"Linehaul\")`"
                )),
            ));
        }
        Type::Error
    }

    /// `bill.member`: a field of the bill so far.
    pub fn bill_field(&mut self, target: &Expr, member: &Name) -> Type {
        let ExprKind::Name(root) = &target.kind else {
            return Type::Error;
        };
        if !self.bill_root(target, root) {
            return Type::Error;
        }
        if member.text.is_empty() {
            return Type::Error;
        }
        if let Some(field) = bill_type().field(&member.text) {
            return field.ty.clone();
        }
        if member.text == "charge" {
            self.report(
                Diagnostic::error(
                    "method-call",
                    member.span,
                    Message::new(message!(
                        "`bill.charge` is a method: call it with a label or code, as in `bill.charge(\"Linehaul\")` or `bill.charge(FSC)`"
                    )),
                ),
            );
            return Type::Error;
        }
        self.unknown_bill_member(member);
        Type::Error
    }

    fn unknown_bill_member(&mut self, member: &Name) {
        let candidates: Vec<String> = bill_type()
            .fields
            .iter()
            .map(|field| field.name.clone())
            .chain(std::iter::once(String::from("charge")))
            .collect();
        let diagnostic = Self::did_you_mean(
            "unknown-member",
            member.span,
            &member.text,
            &candidates,
            Message::with(message!("The bill has no {0}"), [quote(&member.text)]),
            Message::with(
                message!(
                    "The bill has no {0}: read `bill.freight`, `bill.accessorials`, `bill.total` or `bill.charge(…)`"
                ),
                [quote(&member.text)],
            ),
        );
        self.report(diagnostic);
    }

    /// `bill.member(args)`: `bill.charge(…)`, or a field called by mistake.
    pub fn bill_call(
        &mut self,
        expr: &Expr,
        target: &Expr,
        member: &Name,
        args: &[CallArg],
    ) -> Type {
        let ExprKind::Name(root) = &target.kind else {
            return self.skip_args(args);
        };
        if !self.bill_root(target, root) {
            return self.skip_args(args);
        }
        if member.text == "charge" {
            return self.bill_charge(expr, args);
        }
        if bill_type().field(&member.text).is_some() {
            let span = self.parens_span(expr);
            self.report(
                Diagnostic::error(
                    "property-call",
                    span,
                    Message::with(
                        message!("{0} is a property: write it without parentheses"),
                        [quote(&member.text)],
                    ),
                )
                .with_fix(replace(
                    Message::new(message!("Remove the parentheses")),
                    span,
                    "",
                )),
            );
            self.skip_args(args);
            return Type::money();
        }
        if !member.text.is_empty() {
            self.unknown_bill_member(member);
        }
        self.skip_args(args)
    }

    /// `bill.charge(CODE)` or `bill.charge("Label")`: what the lines so far with that code or
    /// label add up to, or `none`.
    fn bill_charge(&mut self, expr: &Expr, args: &[CallArg]) -> Type {
        self.callee(expr, Callee::BillCharge);
        let result = Type::optional(Type::money());
        let [arg] = args else {
            self.report(Diagnostic::error(
                "arity",
                self.parens_span(expr),
                Message::new(message!("`bill.charge(…)` takes one code or label, as in `bill.charge(FSC)` or `bill.charge(\"Linehaul\")`")),
            ));
            self.skip_args(args);
            return result;
        };
        match &arg.value.kind {
            ExprKind::Name(code) if super::calls::is_code(&code.text) && arg.name.is_none() => {
                self.record(&arg.value, Type::Text);
                if self.code_known(code) {
                    self.bill_order(expr.span, arg.value.span, Read::Code(&code.text));
                }
            }
            ExprKind::Text(text) if arg.name.is_none() => {
                self.expr(&arg.value, None);
                if let Some(label) = text.literal() {
                    self.bill_order(expr.span, arg.value.span, Read::Label(&label));
                }
            }
            _ => {
                let ty = self.expr(&arg.value, None);
                if !matches!(ty, Type::Text | Type::Error) {
                    self.report(Diagnostic::error(
                        "type-mismatch",
                        arg.value.span,
                        Message::new(message!(
                            "`bill.charge(…)` takes a charge code such as `FSC` or a label in quotes"
                        )),
                    ));
                }
            }
        }
        result
    }

    /// Warns when what `bill.charge(…)` at `read` names is never billed, or billed only later.
    fn bill_order(&mut self, read: Span, at: Span, wanted: Read<'_>) {
        let matching: Vec<&ChargeSite> = self
            .billed
            .sites
            .iter()
            .filter(|site| match wanted {
                Read::Code(code) => site.code.as_deref() == Some(code),
                Read::Label(label) => site.label.as_deref().is_none_or(|own| own == label),
            })
            .collect();
        if matching.iter().any(|site| site.reaches(read)) {
            return;
        }
        let (what, never) = match wanted {
            Read::Code(code) => (
                quote(code),
                message!("This tariff never bills {0}, so this is always `none`"),
            ),
            Read::Label(label) => (
                quote(label),
                message!("This tariff never bills a line labelled {0}, so this is always `none`"),
            ),
        };
        let Some(first) = matching.first().map(|site| site.span) else {
            let mut diagnostic =
                Diagnostic::warning("not-billed", at, Message::with(never, [what]));
            if let Read::Label(label) = wanted
                && let Some(close) = crate::tariff::diagnostic::nearest(
                    label,
                    self.billed.labels.iter().map(String::as_str),
                )
            {
                let written = format!("\"{close}\"");
                diagnostic = diagnostic.with_fix(replace(
                    Message::with(message!("Change to {0}"), [quote(&written)]),
                    at,
                    written,
                ));
            }
            self.report(diagnostic);
            return;
        };
        let later = matching
            .iter()
            .find(|site| !site.in_sibling(read) && site.span.end > read.start)
            .map(|site| site.span);
        let first = matching
            .iter()
            .find(|site| site.in_sibling(read))
            .map_or(first, |site| site.span);
        let label = matches!(wanted, Read::Label(_));
        let (template, related) = match (later, label) {
            (Some(later), false) => (
                message!("{0} is billed after this line, so this reads `none`"),
                later,
            ),
            (Some(later), true) => (
                message!("The line labelled {0} is billed after this line, so this reads `none`"),
                later,
            ),
            (None, false) => (
                message!(
                    "{0} is billed only in another option of this choice, so this reads `none`: each option reads the bill outside the choice and its own lines"
                ),
                first,
            ),
            (None, true) => (
                message!(
                    "The line labelled {0} is billed only in another option of this choice, so this reads `none`: each option reads the bill outside the choice and its own lines"
                ),
                first,
            ),
        };
        self.report(
            Diagnostic::warning("bill-order", at, Message::with(template, [what]))
                .with_related(related, Message::new(message!("It is billed here"))),
        );
    }

    /// `bill` read as a name on its own, when no local or declaration takes it; `None` when the
    /// name is not the bill.
    pub fn bill_read(&mut self, expr: &Expr, name: &Name) -> Option<Type> {
        if name.text != "bill" || self.globals.values.contains_key("bill") {
            return None;
        }
        Some(self.bill_alone(expr, name))
    }
}

/// What `bill.charge(…)` names.
#[derive(Debug, Clone, Copy)]
enum Read<'r> {
    /// A charge code.
    Code(&'r str),
    /// A label in quotes.
    Label(&'r str),
}
