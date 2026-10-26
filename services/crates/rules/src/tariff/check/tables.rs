//! Rate tables: `tables.Code[k, …]`, `.cell(…)`, `.band(x)` and `.interpolate(x)`, typed from
//! the shapes the context describes.
//!
//! Each key is checked against its axis's key type ([`Axis::key_type`]); an exact or zip-range
//! axis also takes a list of keys (`lane.origin.zones`), which the evaluator tries in order. A
//! zip-range axis takes `Text`, and is not banded, so `.band(x)` and `.interpolate(x)` refuse
//! it. A lookup gives the table's value type, optional, since a miss is `none`. Every table a
//! module reads is recorded ([`ModuleAnalysis::tables`](super::ModuleAnalysis::tables)), and
//! `tariff.check` reports them with their versions so the host can pin them.
//!
//! A test's `given tables.Code[k, …] = value` is held to the same shape ([`Checker::given_cell`]):
//! one key per axis (never a list), and a value of the table's value type or a whole cell. After
//! a module is checked, [`live`] warns (`live-table`) at each test or property that reads a
//! table without giving any of its cells.

use super::Callee;
use super::ModuleAnalysis;
use super::checker::{Checker, Want};
use super::describe::{quote, words};
use super::fixes::replace;
use trenova_module_kit::span::Span;
use crate::tariff::context::Context;
use trenova_module_kit::diagnostic::Diagnostic;
use crate::tariff::schema::{Axis, TableShape};
use crate::tariff::syntax::ast::{Arg as CallArg, Expr, ExprKind, ItemKind, Name, StmtKind};
use crate::tariff::syntax::program::Module;
use crate::tariff::types::{FieldDef, RecordType, Type};
use std::sync::Arc;
use trenova_module_kit::message;
use trenova_module_kit::message::{Arg, Message};

/// The methods a table has besides indexing.
pub const TABLE_METHODS: &[(&str, &str)] = &[
    (
        "cell",
        message!(
            "The matching cell: its `value`, its `minimum` charge (`Money?`) and its `deficit` flag, or `none`."
        ),
    ),
    (
        "band",
        message!(
            "On a one-axis table of bands, the band a value falls in (its `min`, `max` and `value`), or `none`."
        ),
    ),
    (
        "interpolate",
        message!(
            "On a one-axis table of bands, the value interpolated between the bands around a value."
        ),
    ),
];

impl Checker<'_> {
    /// The table a code names, recorded as read, reporting an unknown one (with a fix to a code
    /// that differs only in case, or a close one).
    fn table(&mut self, code: &Name) -> Option<TableShape> {
        if let Some(shape) = self
            .context
            .tables
            .iter()
            .find(|shape| shape.code == code.text)
        {
            self.out.tables.insert(shape.code.clone());
            return Some(shape.clone());
        }
        let lower = code.text.to_lowercase();
        if let Some(shape) = self
            .context
            .tables
            .iter()
            .find(|shape| shape.code.to_lowercase() == lower)
        {
            let stored = shape.code.clone();
            self.report(
                Diagnostic::error(
                    "unknown-table",
                    code.span,
                    Message::with(
                        message!("No rate table has the code {0}; its code is written {1}"),
                        [quote(&code.text), quote(&stored)],
                    ),
                )
                .with_fix(replace(
                    Message::with(message!("Change to {0}"), [quote(&stored)]),
                    code.span,
                    stored.clone(),
                )),
            );
            return None;
        }
        let candidates: Vec<String> = self
            .context
            .tables
            .iter()
            .map(|shape| shape.code.clone())
            .collect();
        let diagnostic = Self::did_you_mean(
            "unknown-table",
            code.span,
            &code.text,
            &candidates,
            Message::with(
                message!("No rate table has the code {0}"),
                [quote(&code.text)],
            ),
            Message::with(
                message!("No rate table has the code {0}; only active rate matrices can be read"),
                [quote(&code.text)],
            ),
        );
        self.report(diagnostic);
        None
    }

    /// The type a table's cells hold.
    fn table_value(&mut self, shape: &TableShape, code: &Name) -> Type {
        shape.value().unwrap_or_else(|| {
            self.report(Diagnostic::error(
                "table-value",
                code.span,
                Message::with(
                    message!(
                        "The rate table {0} holds {1}, which isn't a value type a table can have"
                    ),
                    [quote(&shape.code), quote(&shape.value_type)],
                ),
            ));
            Type::Error
        })
    }

    /// Checks a lookup's keys (or, with `cell`, a given cell's keys) against the table's axes.
    fn table_keys(&mut self, at: Span, shape: &TableShape, keys: &[&Expr], cell: bool) {
        if keys.len() != shape.axes.len() {
            let labels: Vec<String> = shape
                .axes
                .iter()
                .map(|axis| {
                    if axis.label.is_empty() {
                        axis.kind.clone()
                    } else {
                        axis.label.clone()
                    }
                })
                .collect();
            self.report(Diagnostic::error(
                "table-keys",
                at,
                Message::with(
                    message!("{0} takes {1, plural, one {# key} other {# keys}} ({2}), but {3, plural, one {# is} other {# are}} given"),
                    [quote(&shape.code), Arg::from(shape.axes.len()), Arg::Text(labels.join(", ")), Arg::from(keys.len())],
                ),
            ));
        }
        for (index, key) in keys.iter().enumerate() {
            let Some(axis) = shape.axes.get(index) else {
                self.expr(key, None);
                continue;
            };
            self.table_key(key, axis, shape, cell);
        }
    }

    fn table_key(&mut self, key: &Expr, axis: &Axis, shape: &TableShape, cell: bool) {
        let expected = axis.key_type(self.schema);
        let actual = self.expr(key, Some(&expected));
        if axis.takes_key_list()
            && !cell
            && let Type::List(item) = &actual
            && expected.accepts(item)
        {
            return;
        }
        let label = if axis.label.is_empty() {
            axis.kind.clone()
        } else {
            axis.label.clone()
        };
        self.fits(
            key,
            &actual,
            &expected,
            &Want::Argument {
                param: label,
                callee: format!("tables.{}", shape.code),
            },
        );
    }

    /// `given tables.Code[k, …] = value` in a test or property: one key per axis of the axis's
    /// key type (never a list), and the table's value type or a whole cell
    /// (`{ value, minimum, deficit }`).
    pub fn given_cell(&mut self, target: &[Name], keys: Option<&[Expr]>, value: &Expr) {
        let skip = |checker: &mut Self| {
            for key in keys.into_iter().flatten() {
                checker.expr(key, None);
            }
            checker.expr(value, None);
        };
        let [head, code] = target else {
            let span = target.first().map_or(value.span, |first| {
                first
                    .span
                    .cover(target.last().map_or(first.span, |last| last.span))
            });
            self.report(Diagnostic::error(
                "given-target",
                span,
                Message::new(message!(
                    "Set one cell of a rate table by its code and keys, as in `given tables.Zones[\"A\", \"B\"] = $20/cwt`"
                )),
            ));
            skip(self);
            return;
        };
        if let Some(symbol) = self.root_symbol(&head.text) {
            self.reference(head.span, symbol);
        }
        let Some(keys) = keys else {
            self.report(Diagnostic::error(
                "given-target",
                code.span,
                Message::with(
                    message!("A `given` sets one cell of {0}, with its keys: {1}"),
                    [
                        quote(&code.text),
                        quote(&format!("given tables.{}[…] = …", code.text)),
                    ],
                ),
            ));
            skip(self);
            return;
        };
        let Some(shape) = self.table(code) else {
            skip(self);
            return;
        };
        let at = keys.first().map_or(code.span, |first| {
            first
                .span
                .cover(keys.last().map_or(first.span, |last| last.span))
        });
        let refs: Vec<&Expr> = keys.iter().collect();
        self.table_keys(at, &shape, &refs, true);
        let cell = self.table_value(&shape, code);
        let path = format!("tables.{}[…]", shape.code);
        if matches!(value.kind, ExprKind::Record(_)) {
            self.expect(value, &cell_type(cell), &Want::Given { path });
        } else {
            self.expect(value, &cell, &Want::Given { path });
        }
    }

    /// `tables.Code[k, …]`.
    pub fn table_index(&mut self, expr: &Expr, code: &Name, keys: &[Expr]) -> Type {
        let Some(shape) = self.table(code) else {
            for key in keys {
                self.expr(key, None);
            }
            return Type::Error;
        };
        let keys: Vec<&Expr> = keys.iter().collect();
        self.table_keys(expr.span, &shape, &keys, false);
        let value = self.table_value(&shape, code);
        Type::optional(value)
    }

    /// `tables.Code.cell(…)`, `.band(x)`, `.interpolate(x)`.
    pub fn table_method(
        &mut self,
        expr: &Expr,
        code: &Name,
        method: &Name,
        args: &[CallArg],
    ) -> Type {
        let known = TABLE_METHODS
            .iter()
            .find(|(name, _)| *name == method.text)
            .map(|(name, _)| *name);
        let Some(known) = known else {
            let candidates: Vec<String> = TABLE_METHODS
                .iter()
                .map(|(name, _)| (*name).to_owned())
                .collect();
            let found = Message::with(
                message!("A rate table has no method {0}"),
                [quote(&method.text)],
            );
            let missing = Message::with(
                message!(
                    "A rate table has no method {0}: read it with `[…]`, `.cell(…)`, `.band(x)` or `.interpolate(x)`"
                ),
                [quote(&method.text)],
            );
            let diagnostic = Self::did_you_mean(
                "unknown-field",
                method.span,
                &method.text,
                &candidates,
                found,
                missing,
            );
            self.report(diagnostic);
            return self.skip_args(args);
        };
        let Some(shape) = self.table(code) else {
            return self.skip_args(args);
        };
        self.callee(
            expr,
            Callee::Table {
                code: shape.code.clone(),
                method: known,
            },
        );
        for arg in args {
            if let Some(name) = &arg.name {
                self.report(Diagnostic::error(
                    "argument-order",
                    name.span,
                    Message::new(message!(
                        "A table's keys are given in axis order, without names"
                    )),
                ));
            }
        }
        let keys: Vec<&Expr> = args.iter().map(|arg| &arg.value).collect();
        let value = self.table_value(&shape, code);
        if known == "cell" {
            self.table_keys(expr.span, &shape, &keys, false);
            return Type::optional(cell_type(value));
        }
        self.banded(expr, &shape, method, known, &keys, value)
    }

    /// `.band(x)` and `.interpolate(x)` on a one-axis table of bands.
    fn banded(
        &mut self,
        expr: &Expr,
        shape: &TableShape,
        method: &Name,
        known: &'static str,
        keys: &[&Expr],
        value: Type,
    ) -> Type {
        let banded = shape.axes.len() == 1 && shape.axes.first().is_some_and(Axis::is_banded);
        if !banded {
            self.report(Diagnostic::error(
                "table-shape",
                method.span,
                Message::with(
                    message!("{0} works only on a table with one axis of bands, and {1} isn't one"),
                    [quote(known), quote(&shape.code)],
                ),
            ));
            for key in keys {
                self.expr(key, None);
            }
            return Type::Error;
        }
        self.table_keys(expr.span, shape, keys, false);
        let key = shape
            .axes
            .first()
            .map_or(Type::Error, |axis| axis.key_type(self.schema));
        if known == "interpolate" {
            if !matches!(
                value,
                Type::Quantity(_) | Type::Number | Type::Int | Type::Percent | Type::Error
            ) {
                self.report(Diagnostic::error(
                    "table-shape",
                    method.span,
                    Message::with(
                        message!("{0} holds {1}, which can't be interpolated"),
                        [quote(&shape.code), Arg::from(words(&value))],
                    ),
                ));
            }
            return Type::optional(value);
        }
        Type::optional(record(&[
            ("min", key.clone()),
            ("max", Type::optional(key)),
            ("value", value),
        ]))
    }
}

/// What `.cell(…)` gives, and what a test may give a cell as: `{ value, minimum, deficit }`.
#[must_use]
pub fn cell_type(value: Type) -> Type {
    record(&[
        ("value", value),
        ("minimum", Type::optional(Type::money())),
        ("deficit", Type::Bool),
    ])
}

fn record(fields: &[(&str, Type)]) -> Type {
    Type::Record(Arc::new(RecordType {
        name: None,
        fields: fields
            .iter()
            .map(|(name, ty)| FieldDef {
                name: (*name).to_owned(),
                ty: ty.clone(),
                doc: None,
            })
            .collect(),
    }))
}

/// Each table a module's code reads, by code, where it is first read, in order.
fn reads(on: impl FnOnce(&mut dyn FnMut(&Expr))) -> Vec<(String, Span)> {
    let mut found: Vec<(String, Span)> = Vec::new();
    on(&mut |expr: &Expr| {
        if let ExprKind::Field { target, name, .. } = &expr.kind
            && matches!(&target.kind, ExprKind::Name(root) if root.text == "tables")
            && !found.iter().any(|(code, _)| *code == name.text)
        {
            found.push((name.text.clone(), expr.span));
        }
    });
    found
}

/// Warns (`live-table`) at each test and property that reads a rate table, in the file's body or
/// its own, without giving any of its cells: it passes or fails with the matrix's live data.
pub fn live(module: &Module, context: &Context, out: &mut ModuleAnalysis) {
    let known = |code: &str| context.tables.iter().any(|shape| shape.code == code);
    let file = reads(|visit| {
        for item in &module.file.items {
            if let ItemKind::Stmt(stmt) = &item.kind {
                super::visit::stmt(stmt, &mut |_| {}, &mut |expr| visit(expr));
            }
        }
    });
    for item in &module.file.items {
        let (name, body, property) = match &item.kind {
            ItemKind::Test(test) => (&test.name, &test.body, false),
            ItemKind::Property(property) => (&property.name, &property.body, true),
            _ => continue,
        };
        let own = reads(|visit| super::visit::block(body, &mut |_| {}, &mut |expr| visit(expr)));
        let given: Vec<&str> = body
            .stmts
            .iter()
            .filter_map(|stmt| match &stmt.kind {
                StmtKind::Given {
                    target,
                    keys: Some(_),
                    ..
                } => match target.as_slice() {
                    [head, code] if head.text == "tables" => Some(code.text.as_str()),
                    _ => None,
                },
                _ => None,
            })
            .collect();
        let mut warned: Vec<&str> = Vec::new();
        for (code, at) in file.iter().chain(&own) {
            if !known(code) || given.contains(&code.as_str()) || warned.contains(&code.as_str()) {
                continue;
            }
            warned.push(code);
            let template = if property {
                message!(
                    "This property reads the rate table {0} without giving any of its cells, so it passes or fails with the matrix's live data: give the cells it reads, as in {1}"
                )
            } else {
                message!(
                    "This test reads the rate table {0} without giving any of its cells, so it passes or fails with the matrix's live data: give the cells it reads, as in {1}"
                )
            };
            out.diagnostics.push(
                Diagnostic::warning(
                    "live-table",
                    name.span,
                    Message::with(
                        template,
                        [quote(code), quote(&format!("given tables.{code}[…] = …"))],
                    ),
                )
                .with_related(*at, Message::new(message!("It is read here"))),
            );
        }
    }
}
