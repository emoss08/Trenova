//! Checks maps that write X12 from a document: that every segment block is where the guide
//! has it, every value converts to its element, every literal code is one the guide allows, and
//! every mandatory segment and element is written.

use std::collections::{BTreeMap, BTreeSet};

use trenova_module_kit::diagnostic::{Change, Diagnostic, Fix, nearest};
use trenova_module_kit::message;
use trenova_module_kit::message::{Arg, Message};
use trenova_module_kit::span::Span;

use super::env::Env;
use super::x12 as nav;
use super::{Checker, MapInfo, Program};
use crate::lading::facts::Target;
use crate::lading::syntax::ast::{Block, ElementAssign, Expr, ExprKind, Ident, Item, Stmt};
use crate::lading::types::{NodeRef, Type};
use crate::x12::guide::compiled::NodeKind;
use crate::x12::guide::model::Usage;

/// How surely a node is written in a scope.
#[derive(Debug, Clone, Copy, PartialEq, Eq, PartialOrd, Ord)]
enum Written {
    /// Only under a condition.
    Maybe,
    /// Whenever the scope is.
    Always,
}

/// The children of one node a scope writes.
#[derive(Debug, Default)]
struct Writes {
    nodes: BTreeMap<usize, Written>,
}

impl Writes {
    fn mark(&mut self, node: usize, how: Written) {
        let entry = self.nodes.entry(node).or_insert(how);
        if how > *entry {
            *entry = how;
        }
    }
}

/// Whether a value of `value` can be written into an element of type `element`.
fn writable(value: &Type, element: &Type) -> bool {
    let value = value.unwrapped();
    match (value, element) {
        (Type::List(item), Type::List(element_item)) => writable(item, element_item),
        (_, Type::List(item)) => writable(value, item),
        (Type::Any, _)
        | (_, Type::Any)
        | (Type::Text | Type::Code(_), Type::Text | Type::Code(_))
        | (Type::Int | Type::Decimal, Type::Text)
        | (Type::Date | Type::DateTime, Type::Date)
        | (Type::Time | Type::DateTime, Type::Time)
        | (Type::Int, Type::Int)
        | (Type::Int | Type::Decimal | Type::Money | Type::Percent, Type::Decimal) => true,
        _ => false,
    }
}

impl Checker<'_> {
    /// Checks an outbound map.
    pub(crate) fn check_outbound(&mut self, unit: usize, info: &MapInfo) {
        let program: &Program = self.program;
        let Some(guide) = info.target_guide else {
            return;
        };
        let mut env = Env::new(unit);
        env.document = info.document;
        env.write_guide = Some(guide);
        let stmts: Vec<Stmt> = program.units[unit]
            .items()
            .iter()
            .filter_map(|item| match item {
                Item::Stmt(stmt) => Some(stmt.clone()),
                _ => None,
            })
            .collect();
        let mut writes = Writes::default();
        for stmt in &stmts {
            self.outbound_stmt(&mut env, guide, stmt, &mut writes, Written::Always);
        }
        let header = program.units[unit]
            .header()
            .name()
            .map_or_else(Span::default, |name| name.span);
        self.report_unwritten(&env, guide, None, &writes, header);
        self.scope_fact(&env, program.units[unit].parsed.file.span, None);
    }

    fn outbound_block(
        &mut self,
        env: &mut Env,
        guide: usize,
        block: &Block,
        writes: &mut Writes,
        how: Written,
    ) {
        env.push();
        for stmt in &block.stmts {
            self.outbound_stmt(env, guide, stmt, writes, how);
        }
        self.scope_fact(env, block.span, None);
        env.pop();
    }

    #[allow(clippy::too_many_lines)]
    fn outbound_stmt(
        &mut self,
        env: &mut Env,
        guide: usize,
        stmt: &Stmt,
        writes: &mut Writes,
        how: Written,
    ) {
        match stmt {
            Stmt::Segment { id, cond, elements, .. } => {
                let mut how = how;
                if let Some(cond) = cond {
                    self.condition(env, cond);
                    how = Written::Maybe;
                }
                if let Some(node) = self.locate(env, guide, id, false) {
                    writes.mark(node, how);
                    self.segment_elements(env, guide, node, id, elements);
                } else {
                    for element in elements {
                        self.expr(env, &element.value, None);
                    }
                }
            }
            Stmt::Loop { id, each, body, .. } => {
                let Some(node) = self.locate(env, guide, id, true) else {
                    env.push();
                    if let Some((list, binder)) = each {
                        self.expr(env, list, None);
                        env.bind(&binder.name, Type::Any, binder.span);
                    }
                    self.outbound_block(env, guide, body, &mut Writes::default(), Written::Maybe);
                    env.pop();
                    return;
                };
                writes.mark(node, how);
                let saved = env.node;
                env.node = Some(NodeRef { guide, node });
                env.push();
                if let Some((list, binder)) = each {
                    self.bind_each(env, list, binder);
                }
                let mut inner = Writes::default();
                self.outbound_block(env, guide, body, &mut inner, Written::Always);
                env.pop();
                env.node = saved;
                self.report_unwritten(env, guide, Some(node), &inner, id.span);
            }
            Stmt::Each { list, binder, body, .. } => {
                env.push();
                self.bind_each(env, list, binder);
                self.outbound_block(env, guide, body, writes, Written::Maybe);
                env.pop();
            }
            Stmt::When { cond, body, .. } => {
                self.condition(env, cond);
                self.outbound_block(env, guide, body, writes, Written::Maybe);
            }
            Stmt::Let(binding) => {
                let ty = self.expr(env, &binding.value, None);
                self.bind_local(env, &binding.name, ty);
            }
            Stmt::Assign { span, .. } => self.report(
                env.unit,
                Diagnostic::error(
                    "assign-in-outbound",
                    *span,
                    Message::new(message!(
                        "A map to X12 writes segments: SEG {{ 01 <- value }}, or LOOP <- each list as item {{ ... }}"
                    )),
                ),
            ),
            Stmt::Require { span, .. } | Stmt::OutcomeWhen { span, .. } | Stmt::For { span, .. } | Stmt::Fix { span, .. } => {
                self.report(
                    env.unit,
                    Diagnostic::error(
                        "rule-in-map",
                        *span,
                        Message::new(message!("require, for, fix and outcomes belong in a rules file")),
                    ),
                );
            }
            Stmt::Error(_) => {}
        }
    }

    fn bind_each(&mut self, env: &mut Env, list: &Expr, binder: &Ident) {
        let list_ty = self.expr(env, list, None);
        let item = match list_ty.unwrapped() {
            Type::List(item) => item.as_ref().clone(),
            Type::Map(value) => Type::Struct(vec![
                ("key".into(), Type::Text),
                ("value".into(), value.as_ref().clone()),
            ]),
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
        self.bind_local(env, binder, item);
        env.bind("index", Type::Int, Span::default());
    }

    /// The node a segment or loop block writes under the current loop.
    fn locate(&mut self, env: &Env, guide: usize, id: &Ident, is_loop: bool) -> Option<usize> {
        let compiled = &self.program.guides[guide].compiled;
        let parent = env.node.map(|node| node.node);
        let found = if is_loop {
            nav::child_loop(compiled, parent, &id.name)
        } else {
            nav::child_segment(compiled, parent, &id.name)
        };
        if let Some(node) = found {
            let reference = NodeRef { guide, node };
            let target = if is_loop {
                Target::Loop(reference)
            } else {
                Target::Segment(reference)
            };
            self.occurrence(env.unit, id.span, target, false, true);
            if compiled.node(node).usage == Usage::NotUsed {
                self.report(
                    env.unit,
                    Diagnostic::warning(
                        "x12-unused",
                        id.span,
                        Message::with(
                            message!("The partner's guide does not use {0}"),
                            [Arg::from(id.name.as_str())],
                        ),
                    ),
                );
            }
            if !is_loop && matches!(id.name.as_str(), "ST" | "SE" | "ISA" | "IEA" | "GS" | "GE") {
                self.report(
                    env.unit,
                    Diagnostic::error(
                        "envelope-written",
                        id.span,
                        Message::new(message!(
                            "Trenova writes the envelope (ISA, GS, ST, SE, GE, IEA) itself"
                        )),
                    ),
                );
                return None;
            }
            return Some(node);
        }
        let message = if is_loop {
            Message::with(
                message!("There is no loop {0} here"),
                [Arg::from(id.name.as_str())],
            )
        } else {
            match nav::find_segment_anywhere(compiled, &id.name) {
                Some(path) if !path.is_empty() => Message::with(
                    message!("{0} belongs in loop {1}: write it inside {1} {{ ... }}"),
                    [Arg::from(id.name.as_str()), Arg::from(path.join(" > "))],
                ),
                Some(_) => Message::with(
                    message!("{0} is at the top of the transaction, not in this loop"),
                    [Arg::from(id.name.as_str())],
                ),
                None => Message::with(
                    message!("The guide has no {0} segment"),
                    [Arg::from(id.name.as_str())],
                ),
            }
        };
        let candidates: Vec<String> = nav::children(compiled, parent)
            .iter()
            .filter_map(|index| match &compiled.node(*index).kind {
                NodeKind::Segment(segment) if !is_loop => Some(segment.clone()),
                NodeKind::Loop { id, .. } if is_loop => Some(id.clone()),
                _ => None,
            })
            .collect();
        let mut diagnostic = Diagnostic::error("x12-unknown", id.span, message);
        if let Some(close) = nearest(&id.name, candidates.iter().map(String::as_str)) {
            diagnostic = diagnostic.with_fix(super::replace_fix(id.span, close));
        }
        self.report(env.unit, diagnostic);
        None
    }

    #[allow(clippy::too_many_lines)]
    fn segment_elements(
        &mut self,
        env: &mut Env,
        guide: usize,
        node: usize,
        id: &Ident,
        elements: &[ElementAssign],
    ) {
        let program: &Program = self.program;
        let compiled = &program.guides[guide].compiled;
        let Some(definition) = nav::definition(compiled, node) else {
            return;
        };
        let reference = NodeRef { guide, node };
        let mut seen = BTreeSet::new();
        for element in elements {
            let (position, component) = element.position;
            let Some(def) = position
                .checked_sub(1)
                .and_then(|index| definition.elements.get(index as usize))
            else {
                self.report(
                    env.unit,
                    Diagnostic::error(
                        "x12-unknown",
                        element.position_span,
                        Message::with(
                            message!("{0} has {1} elements"),
                            [
                                Arg::from(id.name.as_str()),
                                Arg::from(definition.elements.len().to_string()),
                            ],
                        ),
                    ),
                );
                self.expr(env, &element.value, None);
                continue;
            };
            self.occurrence(
                env.unit,
                element.position_span,
                Target::Element {
                    node: reference,
                    position,
                },
                false,
                true,
            );
            if !seen.insert((position, component)) {
                self.report(
                    env.unit,
                    Diagnostic::error(
                        "duplicate-element",
                        element.position_span,
                        Message::new(message!("This element is already written in this block")),
                    ),
                );
            }
            if def.usage == Usage::NotUsed {
                self.report(
                    env.unit,
                    Diagnostic::warning(
                        "x12-unused",
                        element.position_span,
                        Message::with(
                            message!("The partner's guide does not use {0}{1}"),
                            [
                                Arg::from(id.name.as_str()),
                                Arg::from(format!("{position:02}")),
                            ],
                        ),
                    ),
                );
            }
            let element_ty = nav::element_type(&id.name, position, def);
            let target_ty = match (component, &element_ty) {
                (Some(component), Type::Struct(fields)) => {
                    let key = format!("{component:02}");
                    if let Some((_, ty)) = fields.iter().find(|(name, _)| *name == key) {
                        ty.unwrapped().clone()
                    } else {
                        self.report(
                            env.unit,
                            Diagnostic::error(
                                "x12-unknown",
                                element.position_span,
                                Message::with(
                                    message!("There is no component {0}"),
                                    [Arg::from(key)],
                                ),
                            ),
                        );
                        Type::Any
                    }
                }
                (Some(_), _) => {
                    self.report(
                        env.unit,
                        Diagnostic::error(
                            "x12-unknown",
                            element.position_span,
                            Message::new(message!(
                                "This element is not a composite, so it has no components"
                            )),
                        ),
                    );
                    Type::Any
                }
                (None, Type::Struct(_)) => {
                    self.report(
                        env.unit,
                        Diagnostic::error(
                            "composite-whole",
                            element.position_span,
                            Message::with(
                                message!("{0}{1} is a composite: write its components, as {1}.01 <- value"),
                                [Arg::from(id.name.as_str()), Arg::from(format!("{position:02}"))],
                            ),
                        ),
                    );
                    Type::Any
                }
                (None, ty) => ty.clone(),
            };
            let value_ty = self.expr(env, &element.value, Some(&target_ty));
            if !writable(&value_ty, &target_ty) {
                self.report(
                    env.unit,
                    unwritable(element.value.span, &value_ty, &target_ty),
                );
            }
            self.code_literal(env, &target_ty, &element.value);
            self.length_check(env, &element.value, def.max);
        }
        let missing: Vec<String> = definition
            .elements
            .iter()
            .enumerate()
            .filter(|(index, def)| {
                let position = u32::try_from(index + 1).unwrap_or(0);
                def.usage == Usage::Mandatory
                    && def.fixed.is_none()
                    && !seen.iter().any(|(written, _)| *written == position)
            })
            .map(|(index, _)| format!("{}{:02}", id.name, index + 1))
            .collect();
        if !missing.is_empty() {
            let insert_at = elements
                .last()
                .map_or(id.span.end, |element| element.span.end);
            let lines = missing.iter().fold(String::new(), |mut lines, element| {
                lines.push_str(", ");
                lines.push_str(&element[element.len() - 2..]);
                lines.push_str(" <- ");
                lines
            });
            self.report(
                env.unit,
                Diagnostic::error(
                    "x12-mandatory-unwritten",
                    id.span,
                    Message::with(
                        message!("{0} must be written"),
                        [Arg::from(missing.join(", "))],
                    ),
                )
                .with_fix(Fix {
                    label: Message::new(message!("Add the mandatory elements")),
                    changes: vec![Change {
                        span: Span::new(insert_at, insert_at),
                        insert: lines,
                    }],
                }),
            );
        }
    }

    /// Warns when a document field may hold more characters than its element.
    fn length_check(&mut self, env: &Env, value: &Expr, max: u32) {
        if !matches!(value.kind, ExprKind::Name(_) | ExprKind::Member { .. }) {
            return;
        }
        let end = match &value.kind {
            ExprKind::Member { member, .. } => member.span,
            _ => value.span,
        };
        let field =
            self.facts
                .occurrences
                .iter()
                .rev()
                .find_map(|occurrence| match occurrence.target {
                    Target::Field { record, field }
                        if occurrence.unit == env.unit && occurrence.span == end =>
                    {
                        Some((record, field))
                    }
                    _ => None,
                });
        let Some((record, field)) = field else {
            return;
        };
        let Some(limit) = crate::lading::schema::record(record)
            .and_then(|owner| owner.field(field))
            .and_then(|field| field.max_length)
        else {
            return;
        };
        if limit > max {
            self.report(
                env.unit,
                Diagnostic::warning(
                    "may-exceed-length",
                    value.span,
                    Message::with(
                        message!("{0} may hold {1} characters, and the element takes {2}"),
                        [
                            Arg::from(field),
                            Arg::from(limit.to_string()),
                            Arg::from(max.to_string()),
                        ],
                    ),
                )
                .with_fix(Fix {
                    label: Message::with(
                        message!("Cut it to {0} characters"),
                        [Arg::from(max.to_string())],
                    ),
                    changes: vec![Change {
                        span: Span::new(value.span.end, value.span.end),
                        insert: format!(".truncate({max})"),
                    }],
                }),
            );
        }
    }

    /// Reports the mandatory children of `node` (the top for none) a scope never writes.
    fn report_unwritten(
        &mut self,
        env: &Env,
        guide: usize,
        node: Option<usize>,
        writes: &Writes,
        span: Span,
    ) {
        let compiled = &self.program.guides[guide].compiled;
        let mut never = Vec::new();
        let mut sometimes = Vec::new();
        for child in nav::children(compiled, node) {
            let child_node = compiled.node(*child);
            if child_node.usage != Usage::Mandatory {
                continue;
            }
            let name = match &child_node.kind {
                NodeKind::Segment(id) if matches!(id.as_str(), "ST" | "SE") => continue,
                NodeKind::Segment(id) => id.clone(),
                NodeKind::Loop { id, .. } => format!("loop {id}"),
            };
            match writes.nodes.get(child) {
                Some(Written::Always) => {}
                Some(Written::Maybe) => sometimes.push(name),
                None => never.push(name),
            }
        }
        if !never.is_empty() {
            self.report(
                env.unit,
                Diagnostic::error(
                    "x12-mandatory-unwritten",
                    span,
                    Message::with(
                        message!("The guide requires {0} here, and nothing writes it"),
                        [Arg::from(never.join(", "))],
                    ),
                ),
            );
        }
        if !sometimes.is_empty() {
            self.report(
                env.unit,
                Diagnostic::warning(
                    "x12-mandatory-conditional",
                    span,
                    Message::with(
                        message!(
                            "The guide requires {0} here, and it is written only under a condition"
                        ),
                        [Arg::from(sometimes.join(", "))],
                    ),
                ),
            );
        }
    }
}

fn unwritable(span: Span, value: &Type, element: &Type) -> Diagnostic {
    let hint = match (value.unwrapped(), element) {
        (Type::Measure(_), _) => Some(message!(
            "Write a measure as a number in a unit: weight.value(lb)"
        )),
        (Type::Enum(_), _) => Some(message!(
            "Write an enum through a crosswalk: kind via statusCodes"
        )),
        (Type::Ref(_), _) => Some(message!(
            "Write a reference by its code or label: customer.code"
        )),
        (Type::Decimal, Type::Int) => Some(message!(
            "The element takes a whole number: round it with .round()"
        )),
        (Type::Bool, _) => Some(message!(
            "Write a Bool as a code: if flag then \"Y\" else \"N\""
        )),
        _ => None,
    };
    match hint {
        Some(hint) => Diagnostic::error("type-mismatch", span, Message::new(hint)),
        None => super::expr::mismatch(span, element, value),
    }
}
