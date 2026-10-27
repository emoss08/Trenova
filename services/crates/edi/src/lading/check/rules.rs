//! Checks rules files: each rule's statements against the document its map produces or
//! writes, which outcomes the direction allows, and rules that can never fail.

use trenova_module_kit::diagnostic::{Diagnostic, nearest};
use trenova_module_kit::message;
use trenova_module_kit::message::{Arg, Message};

use super::env::Env;
use super::{Checker, Program};
use crate::lading::facts::Target;
use crate::lading::schema;
use crate::lading::syntax::ast::{Block, Expr, ExprKind, Ident, Item, LValue, Stmt};
use crate::lading::types::Type;

impl Checker<'_> {
    /// Checks a rules file for `map`.
    pub(crate) fn check_rules_file(&mut self, unit: usize, map: &Ident) {
        let program: &Program = self.program;
        let Some(info) = program.maps.get(&map.name) else {
            let names: Vec<&str> = program.maps.keys().map(String::as_str).collect();
            let mut diagnostic = Diagnostic::error(
                "unknown-map",
                map.span,
                Message::with(
                    message!("There is no map named {0}"),
                    [Arg::from(map.name.as_str())],
                ),
            );
            if let Some(close) = nearest(&map.name, names) {
                diagnostic = diagnostic.with_fix(super::replace_fix(map.span, close));
            }
            self.report(unit, diagnostic);
            return;
        };
        self.occurrence(unit, map.span, Target::File(map.name.clone()), false, false);
        let mut env = Env::new(unit);
        env.document = info.document;
        env.inbound = info.inbound();
        for item in program.units[unit].items() {
            match item {
                Item::Rule(rule) => {
                    env.push();
                    let can_fail = self.rule_block(&mut env, &rule.body);
                    if !can_fail && !rule.body.stmts.is_empty() {
                        self.report(
                            unit,
                            Diagnostic::warning(
                                "rule-always-passes",
                                rule.title.span,
                                Message::new(message!(
                                    "This rule can never fail: what it checks is never none, or it has no outcome"
                                )),
                            ),
                        );
                    }
                    self.scope_fact(&env, rule.body.span, None);
                    env.pop();
                }
                Item::Stmt(stmt) => self.report(
                    unit,
                    Diagnostic::error(
                        "stmt-outside-rule",
                        stmt.span(),
                        Message::new(message!(
                            "A rules file holds rules: rule \"title\" '{' ... '}'"
                        )),
                    ),
                ),
                _ => {}
            }
        }
    }

    /// Checks a rule's statements; true when an outcome can happen.
    fn rule_block(&mut self, env: &mut Env, block: &Block) -> bool {
        let mut can_fail = false;
        for stmt in &block.stmts {
            can_fail |= self.rule_stmt(env, stmt);
        }
        can_fail
    }

    #[allow(clippy::too_many_lines)]
    fn rule_stmt(&mut self, env: &mut Env, stmt: &Stmt) -> bool {
        let inbound = env.inbound;
        match stmt {
            Stmt::Require { cond, outcome, .. } => {
                self.condition(env, cond);
                self.outcome(env, outcome, inbound);
                !always_true(cond)
            }
            Stmt::OutcomeWhen { outcome, cond, .. } => {
                self.outcome(env, outcome, inbound);
                self.condition(env, cond);
                true
            }
            Stmt::For {
                binder,
                list,
                filter,
                body,
                ..
            } => {
                let list_ty = self.expr(env, list, None);
                let item = match list_ty.unwrapped() {
                    Type::List(item) => item.as_ref().clone(),
                    Type::Map(value) => Type::Struct(vec![
                        ("key".into(), Type::Text),
                        ("value".into(), value.as_ref().clone()),
                    ]),
                    Type::Json => Type::Json,
                    Type::Any => Type::Any,
                    other => {
                        self.report(
                            env.unit,
                            Diagnostic::error(
                                "not-a-list",
                                list.span,
                                Message::with(
                                    message!("for needs a list, and this is {0}"),
                                    [Arg::from(other.to_string())],
                                ),
                            ),
                        );
                        Type::Any
                    }
                };
                env.push();
                self.bind_local(env, binder, item);
                env.bind(
                    "index",
                    Type::Int,
                    trenova_module_kit::span::Span::default(),
                );
                if let Some(filter) = filter {
                    self.condition(env, filter);
                }
                let can_fail = self.rule_block(env, body);
                self.scope_fact(env, body.span, None);
                env.pop();
                can_fail
            }
            Stmt::Fix { target, value, .. } => {
                let target_ty = self.fix_target(env, target);
                let value_ty = self.expr(env, value, Some(&target_ty));
                if !value_ty.unwrapped().fits(target_ty.unwrapped()) && !value_ty.is_any() {
                    self.report(
                        env.unit,
                        super::expr::mismatch(value.span, target_ty.unwrapped(), &value_ty),
                    );
                }
                true
            }
            Stmt::Let(binding) => {
                let ty = self.expr(env, &binding.value, None);
                self.bind_local(env, &binding.name, ty);
                false
            }
            Stmt::When { cond, body, .. } => {
                self.condition(env, cond);
                env.push();
                let can_fail = self.rule_block(env, body);
                env.pop();
                can_fail
            }
            Stmt::Assign { span, .. } => {
                self.report(
                    env.unit,
                    Diagnostic::error(
                        "assign-in-rule",
                        *span,
                        Message::new(message!(
                            "A rule changes the document with fix: fix field <- value"
                        )),
                    ),
                );
                false
            }
            Stmt::Segment { span, .. } | Stmt::Loop { span, .. } | Stmt::Each { span, .. } => {
                self.report(
                    env.unit,
                    Diagnostic::error(
                        "map-in-rule",
                        *span,
                        Message::new(message!(
                            "A rule checks the document; segments and each belong in a map"
                        )),
                    ),
                );
                false
            }
            Stmt::Error(_) => false,
        }
    }

    /// The type a `fix` target holds: a document field, or a field of a `for` item.
    fn fix_target(&mut self, env: &mut Env, target: &LValue) -> Type {
        let root = &target.root;
        if let Some(local) = env.lookup(&root.name) {
            let ty = local.ty.clone();
            let declared = local.span;
            self.occurrence(
                env.unit,
                root.span,
                Target::Local {
                    unit: env.unit,
                    span: declared,
                },
                false,
                false,
            );
            if target.steps.is_empty() {
                self.report(
                    env.unit,
                    Diagnostic::error(
                        "fix-local",
                        root.span,
                        Message::with(
                            message!("{0} is a name for a value; fix a field of the document, such as {0}.address.postal"),
                            [Arg::from(root.name.as_str())],
                        ),
                    ),
                );
                return Type::Any;
            }
            return self.walk_steps(env, ty, &root.name, &target.steps).0;
        }
        let Some(document) = env.document.and_then(schema::document) else {
            return Type::Any;
        };
        let Some(field) = document.field(&root.name) else {
            let mut diagnostic = Diagnostic::error(
                "unknown-field",
                root.span,
                Message::with(
                    message!("{0} has no field {1}"),
                    [Arg::from(document.name), Arg::from(root.name.as_str())],
                ),
            );
            if let Some(close) = nearest(&root.name, document.fields.iter().map(|field| field.name))
            {
                diagnostic = diagnostic.with_fix(super::replace_fix(root.span, close));
            }
            self.report(env.unit, diagnostic);
            return Type::Any;
        };
        self.occurrence(
            env.unit,
            root.span,
            Target::Field {
                record: document.name,
                field: field.name,
            },
            false,
            true,
        );
        let ty = self.program.field_type(field);
        self.walk_steps(env, ty, field.name, &target.steps).0
    }
}

/// Whether a condition is the literal `true`.
fn always_true(cond: &Expr) -> bool {
    matches!(cond.kind, ExprKind::Bool(true))
}
