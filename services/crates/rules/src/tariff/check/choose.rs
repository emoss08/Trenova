//! `best` and `worst`: choosing between whole billing methods, and `chosen("Label")` in tests.
//!
//! A choice is checked where it is written:
//!
//! - it may appear only where a `charge` may (a tariff's top level and the blocks inside it,
//!   another option included), so the billing rule of `stmt` refuses it in functions, tests,
//!   libraries and conditions;
//! - as a value (`let linehaul = best "Linehaul" { … }`) it gives the kept option's subtotal,
//!   `Money`, and only as the whole value of a `let`, `var` or assignment: anywhere else inside
//!   an expression it is `misplaced`, since it bills lines and must stand where a statement
//!   does;
//! - it has a non-empty label, unique among the file's choices, and at least two options, each
//!   with a non-empty label unique within the choice (the parser already refuses an `option`
//!   written outside a choice, so that one gets no second finding);
//! - each option is a scope of its own; it may not assign a `var` declared outside the choice
//!   (so a losing option leaves no trace), nor leave it with `break` or `continue`;
//! - an option with no `charge` on any path bills `$0`, which is a warning.
//!
//! `chosen("Label")` is a test-only function, like `charge(…)`: the label of the option the
//! last evaluation of that choice kept, as `Text?`. Its argument is a choice label in quotes,
//! one the file declares ("did you mean" otherwise).

use super::checker::{Body, Checker};
use super::describe::quote;
use super::fixes::replace;
use trenova_module_kit::span::Span;
use trenova_module_kit::diagnostic::Diagnostic;
use crate::tariff::syntax::ast::{
    Arg as CallArg, ArmBody, Block, ChoiceOption, Choose, ChooseKind, Expr, ExprKind, FileKind,
    Name, NodeId, StmtKind,
};
use crate::tariff::types::Type;
use trenova_module_kit::message;
use trenova_module_kit::message::Message;

/// What the checker tracks about the choices around the statement being checked.
#[derive(Debug, Default)]
pub struct ChoiceState {
    /// For each enclosing option, innermost last: how many locals were declared when it began
    /// (a local with a lower id is declared outside it) and its choice's kind.
    floors: Vec<(usize, ChooseKind)>,
    /// The labels of the choices checked so far in this body, with where each is written.
    labels: Vec<(String, Span)>,
    /// The choice written as the whole value of the `let`, `var` or assignment being checked,
    /// where a choice may give a value.
    pub value_at: Option<NodeId>,
}

impl Checker<'_> {
    /// A `best` or `worst` written as a value: the kept option's subtotal, when it is the whole
    /// value of a `let`, `var` or assignment.
    pub fn choose_value(&mut self, expr: &Expr, choose: &Choose) -> Type {
        let allowed = self.choice.value_at.take() == Some(expr.id);
        if !allowed {
            self.report(Diagnostic::error(
                "misplaced",
                Span::new(choose.span.start, choose.label.span.end.max(choose.span.start)),
                Message::with(
                    message!("{0} bills lines, so it gives a value only as the whole value of a `let`, `var` or assignment: bind it first, as in {1}, and use the name"),
                    [
                        quote(choose.kind.keyword()),
                        quote(&format!("let subtotal = {} \"{}\" …", choose.kind.keyword(), choose.label.text)),
                    ],
                ),
            ));
        }
        self.choose(choose.span, choose);
        Type::money()
    }

    /// A `best` or `worst`, as a statement or a value written at `span`.
    pub fn choose(&mut self, span: Span, choose: &Choose) {
        if is_misplaced_option(choose) {
            for option in &choose.options {
                self.block(&option.body);
            }
            return;
        }
        let what = match choose.kind {
            ChooseKind::Best => Message::new(message!("A `best`")),
            ChooseKind::Worst => Message::new(message!("A `worst`")),
        };
        self.billing(span, what);
        self.choice_label(choose);
        if choose.options.len() < 2 {
            self.report(Diagnostic::error(
                "choice",
                Span::new(choose.span.start, choose.label.span.end.max(choose.span.start)),
                Message::with(
                    message!("{0} needs at least two options to choose between: one `option \"Label\"` block for each method"),
                    [quote(choose.kind.keyword())],
                ),
            ));
        }
        for (index, option) in choose.options.iter().enumerate() {
            self.option_label(choose, index, option);
            self.option(choose.kind, option);
        }
    }

    /// The choice's label: written, and not used by another choice of the file.
    fn choice_label(&mut self, choose: &Choose) {
        let label = &choose.label;
        if label.text.trim().is_empty() {
            if label.span.end > label.span.start {
                self.report(Diagnostic::error(
                    "choice",
                    label.span,
                    Message::with(
                        message!("{0} needs a label, as in {1}: it names the choice on the trace and in `chosen(…)`"),
                        [
                            quote(choose.kind.keyword()),
                            quote(&format!("{} \"Linehaul\"", choose.kind.keyword())),
                        ],
                    ),
                ));
            }
            return;
        }
        if let Some((_, first)) = self
            .choice
            .labels
            .iter()
            .find(|(seen, _)| *seen == label.text)
        {
            let first = *first;
            self.report(
                Diagnostic::error(
                    "duplicate",
                    label.span,
                    Message::with(
                        message!("Another choice is already labelled {0}: each `best` and `worst` needs its own label, so the trace and `chosen(…)` can tell them apart"),
                        [quote(&label.text)],
                    ),
                )
                .with_related(first, Message::new(message!("First labelled here"))),
            );
            return;
        }
        self.choice.labels.push((label.text.clone(), label.span));
    }

    /// An option's label: written, and not used by an earlier option of the same choice.
    fn option_label(&mut self, choose: &Choose, index: usize, option: &ChoiceOption) {
        let label = &option.label;
        if label.text.trim().is_empty() {
            if label.span.end > label.span.start {
                self.report(Diagnostic::error(
                    "choice",
                    label.span,
                    Message::new(message!(
                        "An option needs a label, as in `option \"Per mile\"`: it names the method on the trace"
                    )),
                ));
            }
            return;
        }
        if let Some(first) = choose
            .options
            .iter()
            .take(index)
            .find(|earlier| earlier.label.text == label.text)
        {
            self.report(
                Diagnostic::error(
                    "duplicate",
                    label.span,
                    Message::with(
                        message!("This {0} already has an option labelled {1}"),
                        [quote(choose.kind.keyword()), quote(&label.text)],
                    ),
                )
                .with_related(
                    first.label.span,
                    Message::new(message!("First labelled here")),
                ),
            );
        }
    }

    /// One option's body: a scope of its own, which may not reach outside the choice.
    fn option(&mut self, kind: ChooseKind, option: &ChoiceOption) {
        if self.loops > 0 {
            self.leaving(kind, &option.body);
        }
        self.choice.floors.push((self.scopes.locals.len(), kind));
        self.block(&option.body);
        self.choice.floors.pop();
        if !bills(&option.body) && !self.block_exits(&option.body) {
            let message = match kind {
                ChooseKind::Best => Message::with(
                    message!(
                        "The option {0} bills no charge on any path, so it is always $0 and `best` always picks it"
                    ),
                    [quote(&option.label.text)],
                ),
                ChooseKind::Worst => Message::with(
                    message!(
                        "The option {0} bills no charge on any path, so it is always $0 and `worst` picks it only when every other option bills $0 or less"
                    ),
                    [quote(&option.label.text)],
                ),
            };
            self.report(Diagnostic::warning("no-charge", option.label.span, message));
        }
    }

    /// Refuses `break` and `continue` that would leave an option for a loop around its choice.
    fn leaving(&mut self, kind: ChooseKind, block: &Block) {
        let mut found = Vec::new();
        exits_of(block, &mut found);
        for (span, word) in found {
            self.report(Diagnostic::error(
                "misplaced",
                span,
                Message::with(
                    message!("{0} can't leave an option of a {1}: each option runs to its end or refuses, and the choice compares what they billed"),
                    [quote(word), quote(kind.keyword())],
                ),
            ));
        }
    }

    /// Refuses assigning a `var` declared outside the innermost enclosing choice; `id` is the
    /// `var`'s local.
    pub fn assign_in_choice(&mut self, target: &Name, id: usize) {
        let Some((floor, kind)) = self.choice.floors.last().copied() else {
            return;
        };
        if id >= floor {
            return;
        }
        let mut diagnostic = Diagnostic::error(
            "assign",
            target.span,
            Message::with(
                message!(
                    "An option can't change {0}, which is declared outside its {1}: a losing option must leave nothing behind. Declare the `var` inside the option, or bill the amount with `charge`"
                ),
                [quote(&target.text), quote(kind.keyword())],
            ),
        );
        if let Some(local) = self.scopes.get(id) {
            diagnostic =
                diagnostic.with_related(local.span, Message::new(message!("It is declared here")));
        }
        self.report(diagnostic);
    }

    /// `chosen("Label")` in a tariff's test: the label of the option the choice kept, or `none`.
    pub fn chosen_lookup(&mut self, expr: &Expr, args: &[CallArg]) -> Type {
        if !matches!(self.body, Body::Test) || self.kind != FileKind::Tariff {
            self.report(Diagnostic::error(
                "test-only",
                expr.span,
                Message::new(message!("`chosen(…)` reads which option a choice kept, so it can only be used inside a tariff's `test` block")),
            ));
            return self.skip_args(args);
        }
        let result = Type::optional(Type::Text);
        let [arg] = args else {
            self.report(Diagnostic::error(
                "arity",
                expr.span,
                Message::new(message!(
                    "`chosen(…)` takes one choice label in quotes, as in `chosen(\"Linehaul\")`"
                )),
            ));
            self.skip_args(args);
            return result;
        };
        self.expr(&arg.value, None);
        let label = match (&arg.name, &arg.value.kind) {
            (None, ExprKind::Text(text)) => text.literal(),
            _ => None,
        };
        let Some(label) = label else {
            self.report(Diagnostic::error(
                "type-mismatch",
                arg.value.span,
                Message::new(message!(
                    "`chosen(…)` takes a choice's label in quotes, as in `chosen(\"Linehaul\")`"
                )),
            ));
            return result;
        };
        if !self.billed.choices.contains(&label) {
            self.unknown_choice(arg.value.span, &label);
        }
        result
    }

    fn unknown_choice(&mut self, span: Span, label: &str) {
        let close = trenova_module_kit::diagnostic::nearest(
            label,
            self.billed.choices.iter().map(String::as_str),
        )
        .map(str::to_owned);
        let diagnostic = match close {
            Some(close) => {
                let written = format!("\"{close}\"");
                Diagnostic::error(
                    "unknown-choice",
                    span,
                    Message::with(
                        message!("This tariff has no choice labelled {0}. Did you mean {1}?"),
                        [quote(label), quote(&close)],
                    ),
                )
                .with_fix(replace(
                    Message::with(message!("Change to {0}"), [quote(&written)]),
                    span,
                    written,
                ))
            }
            None => Diagnostic::error(
                "unknown-choice",
                span,
                Message::with(
                    message!(
                        "This tariff has no choice labelled {0}: `chosen(…)` reads a `best` or `worst` by its label"
                    ),
                    [quote(label)],
                ),
            ),
        };
        self.report(diagnostic);
    }
}

/// Whether the parser made this choice for an `option` written outside one (and reported it).
fn is_misplaced_option(choose: &Choose) -> bool {
    matches!(choose.options.as_slice(), [only] if only.span == choose.span)
        && choose.label.text.is_empty()
}

/// Whether a block bills a charge on some path: a `charge` anywhere in it, nested blocks and
/// choices included.
fn bills(block: &Block) -> bool {
    let mut found = false;
    super::visit::block(
        block,
        &mut |stmt| found |= matches!(stmt.kind, StmtKind::Charge(_)),
        &mut |_| {},
    );
    found
}

/// The `break`s and `continue`s of a block that are not inside a `for` of their own (nor in a
/// nested choice, which reports its own).
fn exits_of(block: &Block, out: &mut Vec<(Span, &'static str)>) {
    for stmt in &block.stmts {
        match &stmt.kind {
            StmtKind::Break => out.push((stmt.span, "break")),
            StmtKind::Continue => out.push((stmt.span, "continue")),
            StmtKind::If(statement) => {
                for (_, body) in &statement.branches {
                    exits_of(body, out);
                }
                if let Some(otherwise) = &statement.otherwise {
                    exits_of(otherwise, out);
                }
            }
            StmtKind::Match(statement) => {
                for arm in &statement.arms {
                    if let ArmBody::Block(body) = &arm.body {
                        exits_of(body, out);
                    }
                }
            }
            _ => {}
        }
    }
}
