//! Checks maps that turn X12, CSV or JSON into a document: what each assignment fills, that
//! every required field is filled on every path, and the records `each` builds.

use std::collections::BTreeSet;

use trenova_module_kit::diagnostic::{Change, Diagnostic, Fix, nearest};
use trenova_module_kit::message;
use trenova_module_kit::message::{Arg, Message};
use trenova_module_kit::span::Span;

use super::env::Env;
use super::expr::mismatch;
use super::{Checker, MapInfo, Source};
use crate::lading::facts::{ScopeFact, Target};
use crate::lading::schema::{self, Record, Spec};
use crate::lading::syntax::ast::{
    Block, Else, Expr, ExprKind, Ident, Item, LValue, LvStep, OutcomeKind, Stmt,
};
use crate::lading::types::Type;

/// What a block has filled of the record it builds, by dotted path.
#[derive(Debug, Clone, Default)]
pub(crate) struct Filled {
    /// Paths filled on every way through the block with a value that is never none.
    certain: BTreeSet<String>,
    /// Paths filled at all.
    any: BTreeSet<String>,
}

impl Filled {
    /// The required fields of `record` under `prefix` that are not certainly filled.
    fn missing(&self, record: &Record, prefix: &str, out: &mut Vec<String>) {
        for field in record.fields.iter().filter(|field| field.required) {
            let path = format!("{prefix}{}", field.name);
            if self.certain.contains(&path) {
                continue;
            }
            let nested = format!("{path}.");
            if let Spec::Record(sub) = field.spec
                && self
                    .certain
                    .iter()
                    .any(|filled| filled.starts_with(&nested))
                && let Some(sub_record) = schema::record(sub)
            {
                self.missing(sub_record, &nested, out);
                continue;
            }
            out.push(path);
        }
    }
}

impl Checker<'_> {
    /// Checks an inbound map.
    pub(crate) fn check_inbound(&mut self, unit: usize, info: &MapInfo) {
        let program = self.program;
        let Some(document) = info.document.and_then(schema::document) else {
            return;
        };
        let mut env = Env::new(unit);
        env.inbound = true;
        match &info.source {
            Source::X12(guide) => env.x12_root = Some(*guide),
            Source::Csv(columns) => {
                let row = Type::Struct(columns.clone());
                env.bind("rows", row.list(), Span::default());
            }
            Source::Json => env.bind("input", Type::Json, Span::default()),
            Source::Document(_) => return,
        }
        let stmts: Vec<&Stmt> = program.units[unit]
            .items()
            .iter()
            .filter_map(|item| match item {
                Item::Stmt(stmt) => Some(stmt),
                _ => None,
            })
            .collect();
        let mut filled = Filled::default();
        for stmt in &stmts {
            self.inbound_stmt(&mut env, stmt, document, &mut filled, true);
        }
        let header_span = program.units[unit]
            .header()
            .name()
            .map_or_else(Span::default, |name| name.span);
        self.report_missing(&env, document, &filled, header_span);
        let file_span = program.units[unit].parsed.file.span;
        self.scope_fact(&env, file_span, Some(Type::Record(document.name)));
    }

    pub(crate) fn scope_fact(&mut self, env: &Env, span: Span, builds: Option<Type>) {
        self.facts.scopes.push(ScopeFact {
            unit: env.unit,
            span,
            locals: env.visible(),
            builds,
            x12: env.node,
            document: env.document,
            guide: env.x12_root.or(env.write_guide),
        });
    }

    fn report_missing(&mut self, env: &Env, record: &Record, filled: &Filled, span: Span) {
        let mut missing = Vec::new();
        filled.missing(record, "", &mut missing);
        if missing.is_empty() {
            return;
        }
        let maybe: Vec<&String> = missing
            .iter()
            .filter(|path| filled.any.contains(*path))
            .collect();
        let message = if maybe.len() == missing.len() {
            Message::with(
                message!(
                    "{0} is not filled on every path: fill it outside when, or give it an else"
                ),
                [Arg::from(missing.join(", "))],
            )
        } else {
            Message::with(
                message!("{0} needs {1}"),
                [Arg::from(record.name), Arg::from(missing.join(", "))],
            )
        };
        self.report(
            env.unit,
            Diagnostic::error("field-unassigned", span, message),
        );
    }

    fn inbound_block(
        &mut self,
        env: &mut Env,
        block: &Block,
        record: &'static Record,
        filled: &mut Filled,
        certain: bool,
    ) {
        env.push();
        for stmt in &block.stmts {
            self.inbound_stmt(env, stmt, record, filled, certain);
        }
        self.scope_fact(env, block.span, Some(Type::Record(record.name)));
        env.pop();
    }

    #[allow(clippy::too_many_lines)]
    fn inbound_stmt(
        &mut self,
        env: &mut Env,
        stmt: &Stmt,
        record: &'static Record,
        filled: &mut Filled,
        certain: bool,
    ) {
        match stmt {
            Stmt::Assign {
                target,
                value,
                otherwise,
                span,
            } => self.inbound_assign(env, target, value, otherwise.as_ref(), *span, record, filled, certain),
            Stmt::Let(binding) => {
                let ty = self.expr(env, &binding.value, None);
                self.bind_local(env, &binding.name, ty);
            }
            Stmt::When { cond, body, .. } => {
                self.condition(env, cond);
                self.inbound_block(env, body, record, filled, false);
            }
            Stmt::Each { span, .. } => self.report(
                env.unit,
                Diagnostic::error(
                    "each-statement",
                    *span,
                    Message::new(message!(
                        "In a map to a document, each builds a field: stops <- each 0300 as stop '{' ... '}'"
                    )),
                ),
            ),
            Stmt::Segment { span, .. } | Stmt::Loop { span, .. } => self.report(
                env.unit,
                Diagnostic::error(
                    "segment-in-inbound",
                    *span,
                    Message::new(message!(
                        "Segments are written by a map from a document to X12; this map fills a document"
                    )),
                ),
            ),
            Stmt::Require { span, .. } | Stmt::OutcomeWhen { span, .. } | Stmt::For { span, .. } | Stmt::Fix { span, .. } => {
                self.report(
                    env.unit,
                    Diagnostic::error(
                        "rule-in-map",
                        *span,
                        Message::new(message!(
                            "require, for, fix and outcomes belong in a rules file; a map uses else reject on an assignment"
                        )),
                    ),
                );
            }
            Stmt::Error(_) => {}
        }
    }

    /// Binds a `let` and records its declaration.
    pub(crate) fn bind_local(&mut self, env: &mut Env, name: &Ident, ty: Type) {
        if name.name.is_empty() {
            return;
        }
        if env.lookup(&name.name).is_some() {
            self.report(
                env.unit,
                Diagnostic::warning(
                    "shadowed",
                    name.span,
                    Message::with(
                        message!("{0} hides an earlier name"),
                        [Arg::from(name.name.as_str())],
                    ),
                ),
            );
        }
        env.bind(&name.name, ty, name.span);
        self.occurrence(
            env.unit,
            name.span,
            Target::Local {
                unit: env.unit,
                span: name.span,
            },
            true,
            false,
        );
    }

    #[allow(clippy::too_many_arguments, clippy::too_many_lines)]
    fn inbound_assign(
        &mut self,
        env: &mut Env,
        target: &LValue,
        value: &Expr,
        otherwise: Option<&Else>,
        span: Span,
        record: &'static Record,
        filled: &mut Filled,
        certain: bool,
    ) {
        let Some(field) = record.field(&target.root.name) else {
            let mut diagnostic = if env.lookup(&target.root.name).is_some() {
                Diagnostic::error(
                    "assign-local",
                    target.root.span,
                    Message::with(
                        message!("{0} is a let; an assignment fills a field of {1}"),
                        [Arg::from(target.root.name.as_str()), Arg::from(record.name)],
                    ),
                )
            } else {
                Diagnostic::error(
                    "unknown-field",
                    target.root.span,
                    Message::with(
                        message!("{0} has no field {1}"),
                        [Arg::from(record.name), Arg::from(target.root.name.as_str())],
                    ),
                )
            };
            if let Some(close) = nearest(
                &target.root.name,
                record.fields.iter().map(|field| field.name),
            ) {
                diagnostic = diagnostic.with_fix(super::replace_fix(target.root.span, close));
            }
            self.report(env.unit, diagnostic);
            self.expr(env, value, None);
            return;
        };
        self.occurrence(
            env.unit,
            target.root.span,
            Target::Field {
                record: record.name,
                field: field.name,
            },
            false,
            true,
        );
        let root_ty = self.program.field_type(field);
        let (target_ty, path) = self.walk_steps(env, root_ty, field.name, &target.steps);
        let value_ty = if let ExprKind::Each { list, binder, body } = &value.kind {
            self.each_value(env, &target_ty, list, binder, body, value.span)
        } else {
            let ty = self.expr(env, value, Some(&target_ty));
            self.else_clause(env, ty, otherwise, true)
        };
        if matches!(value.kind, ExprKind::Each { .. }) && otherwise.is_some() {
            self.report(
                env.unit,
                Diagnostic::warning(
                    "else-never-used",
                    span,
                    Message::new(message!(
                        "each always builds a list, so this else is never used"
                    )),
                ),
            );
        }
        if !value_ty.unwrapped().fits(target_ty.unwrapped()) && !value_ty.is_any() {
            let mut diagnostic = mismatch(value.span, target_ty.unwrapped(), &value_ty);
            if let (Type::Enum(name), Type::Code(..) | Type::Text) =
                (target_ty.unwrapped(), value_ty.unwrapped())
            {
                diagnostic = Diagnostic::error(
                    "type-mismatch",
                    value.span,
                    Message::with(
                        message!(
                            "{0} is a code, and the field is a {1}: translate it with via and a crosswalk"
                        ),
                        [
                            Arg::from(value_ty.unwrapped().to_string()),
                            Arg::from(*name),
                        ],
                    ),
                );
            } else if let (Type::Ref(kind), Type::Code(..) | Type::Text) =
                (target_ty.unwrapped(), value_ty.unwrapped())
            {
                diagnostic = Diagnostic::error(
                    "type-mismatch",
                    value.span,
                    Message::with(
                        message!("The field is a {0}: find it with lookup {0}(code)"),
                        [Arg::from(*kind)],
                    ),
                )
                .with_fix(Fix {
                    label: Message::with(message!("Look up a {0}"), [Arg::from(*kind)]),
                    changes: vec![
                        Change {
                            span: Span::new(value.span.start, value.span.start),
                            insert: format!("lookup {kind}("),
                        },
                        Change {
                            span: Span::new(value.span.end, value.span.end),
                            insert: ")".into(),
                        },
                    ],
                });
            }
            self.report(env.unit, diagnostic);
        }
        let required = field.required && target.steps.is_empty();
        let sure = !value_ty.is_optional();
        if required && !sure && !value_ty.is_any() {
            let end = span.end;
            let field_name = field.name;
            self.report(
                env.unit,
                Diagnostic::error(
                    "optional-unhandled",
                    value.span,
                    Message::with(
                        message!("{0} is required and this may be none: give a default or say what happens"),
                        [Arg::from(field_name)],
                    ),
                )
                .with_fix(Fix {
                    label: Message::new(message!("Reject the message when it is none")),
                    changes: vec![Change {
                        span: Span::new(end, end),
                        insert: format!(" else reject \"{field_name} is missing\""),
                    }],
                })
                .with_fix(Fix {
                    label: Message::new(message!("Hold the document when it is none")),
                    changes: vec![Change {
                        span: Span::new(end, end),
                        insert: format!(" else hold \"{field_name} is missing\""),
                    }],
                }),
            );
        }
        let handled = sure
            || otherwise.is_some_and(|clause| {
                matches!(clause, Else::Outcome(outcome) if matches!(outcome.kind, OutcomeKind::Reject(_) | OutcomeKind::Decline(_)))
            });
        if certain && handled {
            if filled.certain.contains(&path) && target.steps.is_empty() {
                self.report(
                    env.unit,
                    Diagnostic::warning(
                        "assignment-overwritten",
                        target.span,
                        Message::with(
                            message!("{0} is already filled with a value that is never none; this overwrites it"),
                            [Arg::from(path.as_str())],
                        ),
                    ),
                );
            }
            filled.certain.insert(path.clone());
        }
        filled.any.insert(path);
        if target.steps.is_empty() {
            let readable = if filled.certain.contains(field.name) {
                target_ty.unwrapped().clone()
            } else {
                target_ty.unwrapped().clone().optional()
            };
            env.bind(field.name, readable, target.root.span);
        }
    }

    /// The type a value assigned through `steps` must have, and the dotted path filled.
    #[allow(clippy::too_many_lines)]
    pub(crate) fn walk_steps(
        &mut self,
        env: &mut Env,
        root: Type,
        root_name: &str,
        steps: &[LvStep],
    ) -> (Type, String) {
        let mut ty = root;
        let mut path = root_name.to_owned();
        for step in steps {
            match step {
                LvStep::Field(name) => {
                    let next = match ty.unwrapped() {
                        Type::Record(record_name) => {
                            schema::record(record_name).and_then(|owner| {
                                let found = owner.field(&name.name)?;
                                Some((owner, found))
                            })
                        }
                        _ => None,
                    };
                    if let Some((owner, field)) = next {
                        self.occurrence(
                            env.unit,
                            name.span,
                            Target::Field {
                                record: owner.name,
                                field: field.name,
                            },
                            false,
                            true,
                        );
                        ty = self.program.field_type(field);
                    } else if let Type::Struct(fields) = ty.unwrapped() {
                        if let Some((_, field_ty)) =
                            fields.iter().find(|(field, _)| *field == name.name)
                        {
                            ty = field_ty.unwrapped().clone();
                        } else {
                            let names: Vec<&str> =
                                fields.iter().map(|(field, _)| field.as_str()).collect();
                            let message = unknown_field(&path, &name.name, &names);
                            let mut diagnostic =
                                Diagnostic::error("unknown-field", name.span, message);
                            if let Some(close) = nearest(&name.name, names) {
                                diagnostic =
                                    diagnostic.with_fix(super::replace_fix(name.span, close));
                            }
                            self.report(env.unit, diagnostic);
                            ty = Type::Any;
                        }
                    } else if !ty.is_any() {
                        let candidates: Vec<&str> = match ty.unwrapped() {
                            Type::Record(record_name) => schema::record(record_name)
                                .map(|owner| owner.fields.iter().map(|field| field.name).collect())
                                .unwrap_or_default(),
                            _ => Vec::new(),
                        };
                        let mut diagnostic = Diagnostic::error(
                            "unknown-field",
                            name.span,
                            Message::with(
                                message!("{0} has no field {1}"),
                                [
                                    Arg::from(ty.unwrapped().to_string()),
                                    Arg::from(name.name.as_str()),
                                ],
                            ),
                        );
                        if let Some(close) = nearest(&name.name, candidates) {
                            diagnostic = diagnostic.with_fix(super::replace_fix(name.span, close));
                        }
                        self.report(env.unit, diagnostic);
                        ty = Type::Any;
                    }
                    path.push('.');
                    path.push_str(&name.name);
                }
                LvStep::Index(index) => {
                    match ty.unwrapped().clone() {
                        Type::Map(value) => {
                            self.expect_type(env, index, &Type::Text);
                            ty = value.as_ref().clone();
                        }
                        Type::List(item) => {
                            self.expect_type(env, index, &Type::Int);
                            ty = item.as_ref().clone();
                        }
                        Type::Any => {
                            self.expr(env, index, None);
                        }
                        other => {
                            self.report(
                                env.unit,
                                Diagnostic::error(
                                    "bad-index",
                                    index.span,
                                    Message::with(
                                        message!("{0} cannot be indexed"),
                                        [Arg::from(other.to_string())],
                                    ),
                                ),
                            );
                            ty = Type::Any;
                        }
                    }
                    path.push_str("[]");
                }
            }
        }
        (ty, path)
    }

    /// Types `each list as binder { ... }` building the records of a list field.
    fn each_value(
        &mut self,
        env: &mut Env,
        target: &Type,
        list: &Expr,
        binder: &Ident,
        body: &Block,
        span: Span,
    ) -> Type {
        let list_ty = self.expr(env, list, None);
        let item = match list_ty.unwrapped() {
            Type::List(item) => item.as_ref().clone(),
            Type::Json => Type::Json,
            Type::Any => Type::Any,
            other => {
                self.report(
                    env.unit,
                    Diagnostic::error(
                        "not-a-list",
                        list.span,
                        Message::with(
                            message!("each needs a list, and this is {0}"),
                            [Arg::from(other.to_string())],
                        ),
                    ),
                );
                Type::Any
            }
        };
        let Some(record) = (match target.unwrapped() {
            Type::List(inner) => match inner.as_ref() {
                Type::Record(name) => schema::record(name),
                _ => None,
            },
            _ => None,
        }) else {
            if !target.is_any() {
                self.report(
                    env.unit,
                    Diagnostic::error(
                        "each-target",
                        span,
                        Message::with(
                            message!("each builds a list of records, and this field is {0}"),
                            [Arg::from(target.to_string())],
                        ),
                    ),
                );
            }
            return Type::Any;
        };
        env.push();
        self.bind_local(env, binder, item);
        env.bind("index", Type::Int, Span::default());
        let mut filled = Filled::default();
        self.inbound_block(env, body, record, &mut filled, true);
        env.pop();
        self.report_missing(env, record, &filled, binder.span);
        Type::Record(record.name).list()
    }
}

/// How many custom fields an unknown one's message names before it names only the closest.
const NAMED_CUSTOM_FIELDS: usize = 5;

/// What a field path that names no field says: a custom field's points at Settings, or names
/// the organization's custom fields (the closest first, past a handful).
fn unknown_field(path: &str, name: &str, names: &[&str]) -> Message {
    if path != "custom" && !path.ends_with(".custom") {
        return Message::with(message!("There is no field {0} here"), [Arg::from(name)]);
    }
    if names.is_empty() {
        return Message::with(
            message!(
                "The organization has no custom field {0}: add it in Settings, under Custom fields"
            ),
            [Arg::from(name)],
        );
    }
    if names.len() <= NAMED_CUSTOM_FIELDS {
        return Message::with(
            message!("There is no custom field {0}; the organization's are {1}"),
            [Arg::from(name), Arg::from(names.join(", "))],
        );
    }
    let mut named: Vec<&str> = nearest(name, names.iter().copied()).into_iter().collect();
    named.extend(
        names
            .iter()
            .copied()
            .filter(|candidate| !named.contains(candidate))
            .take(NAMED_CUSTOM_FIELDS - named.len()),
    );
    Message::with(
        message!(
            "There is no custom field {0}; the organization has {1, plural, one {# custom field} other {# custom fields}}, such as {2}"
        ),
        [
            Arg::from(name),
            Arg::from(names.len()),
            Arg::from(named.join(", ")),
        ],
    )
}
