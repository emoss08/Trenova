//! Runs a rules file on a document: requirements and outcomes, `for` over the document's
//! lists, and fixes, each recorded with its rule, the old value and the new.

use serde::Serialize;

use super::inbound::{Step, get, set, steps_of};
use super::interp::{Eval, Frame, Interp, Lineage};
use super::json::to_json;
use super::value::{Value, display};
use crate::lading::schema;
use crate::lading::syntax::ast::{Block, Item, LValue, LvStep, Stmt};

/// A change a rule made.
#[derive(Debug, Clone, PartialEq, Serialize)]
#[serde(rename_all = "camelCase")]
pub struct Fixed {
    /// The rule.
    pub rule: String,
    /// The document path.
    pub path: String,
    /// The value before.
    pub old: serde_json::Value,
    /// The value after.
    pub new: serde_json::Value,
}

impl Interp<'_> {
    /// Runs every rule of a rules file on a document, giving the document after its fixes.
    pub fn run_rules(&mut self, unit: usize, document: Value, fixes: &mut Vec<Fixed>) -> Eval {
        let program = self.program;
        let mut frame = Frame::new(unit);
        frame.document = Some(document);
        for item in program.units[unit].items() {
            let Item::Rule(rule) = item else {
                continue;
            };
            self.cover(unit, rule.span.start);
            frame.push();
            self.rule_block(&mut frame, &rule.body, &rule.title.value, fixes)?;
            frame.pop();
        }
        Ok(frame.document.unwrap_or(Value::None))
    }

    fn rule_block(
        &mut self,
        frame: &mut Frame,
        block: &Block,
        rule: &str,
        fixes: &mut Vec<Fixed>,
    ) -> Eval<()> {
        for stmt in &block.stmts {
            self.rule_stmt(frame, stmt, rule, fixes)?;
        }
        Ok(())
    }

    fn rule_stmt(
        &mut self,
        frame: &mut Frame,
        stmt: &Stmt,
        rule: &str,
        fixes: &mut Vec<Fixed>,
    ) -> Eval<()> {
        match stmt {
            Stmt::Require {
                cond,
                outcome,
                span,
            } => {
                self.cover(frame.unit, span.start);
                if !self.eval(frame, cond)?.truthy() {
                    let path = Self::doc_path(frame, cond).unwrap_or_default();
                    self.outcome(frame, outcome, rule, &path)?;
                }
            }
            Stmt::OutcomeWhen {
                outcome,
                cond,
                span,
            } => {
                self.cover(frame.unit, span.start);
                if self.eval(frame, cond)?.truthy() {
                    self.outcome(frame, outcome, rule, "")?;
                }
            }
            Stmt::For {
                binder,
                list,
                filter,
                body,
                span,
            } => {
                self.cover(frame.unit, span.start);
                let items = self.eval(frame, list)?;
                let origin = Self::doc_path(frame, list);
                let entries: Vec<(String, Value)> = match &items {
                    Value::Map(fields) => fields
                        .iter()
                        .map(|(key, value)| {
                            let mut pair = std::collections::BTreeMap::new();
                            pair.insert("key".to_owned(), Value::text(key));
                            pair.insert("value".to_owned(), value.clone());
                            (key.clone(), Value::Struct(std::rc::Rc::new(pair)))
                        })
                        .collect(),
                    other => other
                        .items()
                        .iter()
                        .enumerate()
                        .map(|(index, item)| (index.to_string(), item.clone()))
                        .collect(),
                };
                for (key, item) in entries {
                    frame.push();
                    frame.bind(&binder.name, item);
                    if let Some(origin) = &origin {
                        frame
                            .origins
                            .push((binder.name.clone(), format!("{origin}[{key}]")));
                    }
                    let keep = match filter {
                        Some(filter) => self.eval(frame, filter)?.truthy(),
                        None => true,
                    };
                    let result = if keep {
                        self.rule_block(frame, body, rule, fixes)
                    } else {
                        Ok(())
                    };
                    if origin.is_some() {
                        frame.origins.pop();
                    }
                    frame.pop();
                    result?;
                }
            }
            Stmt::Fix {
                target,
                value,
                span,
            } => {
                self.cover(frame.unit, span.start);
                self.fix(frame, target, value, rule, fixes)?;
            }
            Stmt::Let(binding) => {
                let value = self.eval(frame, &binding.value)?;
                frame.bind(&binding.name.name, value);
            }
            Stmt::When { cond, body, span } => {
                self.cover(frame.unit, span.start);
                if self.eval(frame, cond)?.truthy() {
                    frame.push();
                    let result = self.rule_block(frame, body, rule, fixes);
                    frame.pop();
                    result?;
                }
            }
            _ => {}
        }
        Ok(())
    }

    fn fix(
        &mut self,
        frame: &mut Frame,
        target: &LValue,
        value: &crate::lading::syntax::ast::Expr,
        rule: &str,
        fixes: &mut Vec<Fixed>,
    ) -> Eval<()> {
        let root = &target.root.name;
        let base = match frame.origin(root) {
            Some(origin) => origin.to_owned(),
            None => root.clone(),
        };
        let mut steps = steps_of(&base);
        let mut path = base;
        for step in &target.steps {
            match step {
                LvStep::Field(name) => {
                    path = format!("{path}.{}", name.name);
                    steps.push(Step::Field(name.name.clone()));
                }
                LvStep::Index(index) => {
                    let key = self.eval(frame, index)?;
                    let key_text = display(&key);
                    path = format!("{path}[{key_text}]");
                    steps.push(match key {
                        Value::Int(position) => {
                            Step::Index(usize::try_from(position).unwrap_or(usize::MAX))
                        }
                        _ => Step::Key(key_text),
                    });
                }
            }
        }
        let new = self.eval(frame, value)?;
        let Some(document) = frame.document.take() else {
            return Ok(());
        };
        let old = get(&document, &steps);
        let new = match &document {
            Value::Record(name, _) => self.field_value(name, &steps, new),
            _ => new,
        };
        let updated = set(document, &steps, new.clone());
        fixes.push(Fixed {
            rule: rule.to_owned(),
            path: path.clone(),
            old: to_json(&old),
            new: to_json(&new),
        });
        self.lineage.push(Lineage {
            path,
            sources: Vec::new(),
            steps: vec![format!("fix: {rule}")],
            value: display(&new),
        });
        if let Some(origin) = frame.origin(root).map(str::to_owned) {
            let rebound = get(&updated, &steps_of(&origin));
            frame.bind(root, rebound);
        }
        frame.document = Some(updated);
        Ok(())
    }

    /// A value made to fit the field at `steps` of a record.
    fn field_value(&self, record: &'static str, steps: &[Step], value: Value) -> Value {
        let mut owner = schema::record(record);
        let mut ty = None;
        for step in steps {
            match step {
                Step::Field(name) => {
                    let field = owner.and_then(|current| current.field(name));
                    ty = field.map(|field| self.program.field_type(field));
                    owner = match field.map(|field| field.spec) {
                        Some(schema::Spec::Record(name) | schema::Spec::Records(name)) => {
                            schema::record(name)
                        }
                        _ => None,
                    };
                }
                Step::Index(_) | Step::Key(_) => {
                    ty = ty.and_then(|ty| {
                        ty.item().cloned().or(match ty {
                            crate::lading::types::Type::Map(value) => Some(*value),
                            _ => None,
                        })
                    });
                }
            }
        }
        match ty {
            Some(ty) => self.coerce(value, &ty),
            None => value,
        }
    }
}
