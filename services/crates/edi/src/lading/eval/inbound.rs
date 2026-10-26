//! Runs a map that fills a document: each assignment in order, `each` building the records of
//! a list, `when` and `let`, with lineage for every field filled.

use std::collections::BTreeMap;
use std::rc::Rc;

use super::interp::{Eval, Frame, Interp, join_path};
use super::value::Value;
use crate::lading::schema;
use crate::lading::syntax::ast::{Block, Else, Expr, ExprKind, Ident, LValue, LvStep, Stmt};
use crate::lading::types::Type;

/// A record being filled.
struct Builder {
    record: &'static str,
    fields: BTreeMap<String, Value>,
}

impl Interp<'_> {
    /// Runs an inbound map's statements, giving the document.
    pub fn fill_document(
        &mut self,
        frame: &mut Frame,
        stmts: &[&Stmt],
        document: &'static str,
    ) -> Eval {
        let mut builder = Builder {
            record: document,
            fields: BTreeMap::new(),
        };
        for stmt in stmts {
            self.fill_stmt(frame, stmt, &mut builder)?;
        }
        Ok(self.finish(builder))
    }

    fn finish(&self, builder: Builder) -> Value {
        let Some(record) = schema::record(builder.record) else {
            return Value::Struct(Rc::new(builder.fields));
        };
        let mut fields = BTreeMap::new();
        for (key, value) in builder.fields {
            let converted = match record.field(&key) {
                Some(field) => self.coerce(value, &self.program.field_type(field)),
                None => value,
            };
            if !converted.is_none() {
                fields.insert(key, converted);
            }
        }
        Value::Record(record.name, Rc::new(fields))
    }

    fn fill_block(&mut self, frame: &mut Frame, block: &Block, builder: &mut Builder) -> Eval<()> {
        frame.push();
        for stmt in &block.stmts {
            self.fill_stmt(frame, stmt, builder)?;
        }
        frame.pop();
        Ok(())
    }

    fn fill_stmt(&mut self, frame: &mut Frame, stmt: &Stmt, builder: &mut Builder) -> Eval<()> {
        match stmt {
            Stmt::Assign {
                target,
                value,
                otherwise,
                span,
            } => {
                self.cover(frame.unit, span.start);
                self.fill_assign(frame, target, value, otherwise.as_ref(), builder)
            }
            Stmt::Let(binding) => {
                self.cover(frame.unit, binding.span.start);
                let value = self.eval(frame, &binding.value)?;
                frame.bind(&binding.name.name, value);
                Ok(())
            }
            Stmt::When { cond, body, span } => {
                self.cover(frame.unit, span.start);
                if self.eval(frame, cond)?.truthy() {
                    self.fill_block(frame, body, builder)?;
                }
                Ok(())
            }
            _ => Ok(()),
        }
    }

    fn fill_assign(
        &mut self,
        frame: &mut Frame,
        target: &LValue,
        value: &Expr,
        otherwise: Option<&Else>,
        builder: &mut Builder,
    ) -> Eval<()> {
        let Some(field) =
            schema::record(builder.record).and_then(|record| record.field(&target.root.name))
        else {
            return Ok(());
        };
        let mut steps = Vec::with_capacity(target.steps.len());
        let mut path = join_path(&frame.path, field.name);
        for step in &target.steps {
            match step {
                LvStep::Field(name) => {
                    path = join_path(&path, &name.name);
                    steps.push(Step::Field(name.name.clone()));
                }
                LvStep::Index(index) => {
                    let key = self.eval(frame, index)?;
                    let key_text = super::value::display(&key);
                    path = format!("{path}[{key_text}]");
                    steps.push(Step::Key(key_text));
                }
            }
        }
        self.begin_trace();
        let mut result = if let ExprKind::Each { list, binder, body } = &value.kind {
            let item_record = match self.program.field_type(field) {
                Type::List(item) => match *item {
                    Type::Record(name) => Some(name),
                    _ => None,
                },
                _ => None,
            };
            match item_record {
                Some(record) => self.each_records(frame, list, binder, body, record, &path)?,
                None => Value::None,
            }
        } else {
            let asking = frame.asking.replace(path.clone());
            let value = self.eval(frame, value);
            frame.asking = asking;
            value?
        };
        if result.is_none() {
            match otherwise {
                Some(Else::Value(default)) => result = self.eval(frame, default)?,
                Some(Else::Outcome(outcome)) => self.outcome(frame, outcome, "", &path)?,
                None => {}
            }
        }
        let traced = if matches!(value.kind, ExprKind::Each { .. }) {
            Value::None
        } else {
            result.clone()
        };
        self.end_trace(&path, &traced);
        if result.is_none() {
            return Ok(());
        }
        let current = builder.fields.remove(field.name).unwrap_or(Value::None);
        let updated = set(current, &steps, result);
        let field_value = self.coerce(updated, &self.program.field_type(field));
        builder
            .fields
            .insert(field.name.to_owned(), field_value.clone());
        frame.bind(field.name, field_value);
        Ok(())
    }

    fn each_records(
        &mut self,
        frame: &mut Frame,
        list: &Expr,
        binder: &Ident,
        body: &Block,
        record: &'static str,
        path: &str,
    ) -> Eval {
        let items = self.eval(frame, list)?;
        let saved_path = std::mem::take(&mut frame.path);
        let mut records = Vec::new();
        for (index, item) in items.items().iter().enumerate() {
            frame.path = format!("{path}[{index}]");
            frame.push();
            frame.bind(&binder.name, item.clone());
            frame.bind("index", Value::Int(i64::try_from(index).unwrap_or(0)));
            let mut builder = Builder {
                record,
                fields: BTreeMap::new(),
            };
            let outcome = (|| {
                for stmt in &body.stmts {
                    self.fill_stmt(frame, stmt, &mut builder)?;
                }
                Ok(())
            })();
            frame.pop();
            if let Err(halt) = outcome {
                frame.path = saved_path;
                return Err(halt);
            }
            records.push(self.finish(builder));
        }
        frame.path = saved_path;
        Ok(Value::list(records))
    }
}

/// One step into a field.
#[derive(Debug, Clone)]
pub enum Step {
    /// A record's field.
    Field(String),
    /// A map's key.
    Key(String),
    /// A list's item.
    Index(usize),
}

/// `value` with what is at `steps` replaced.
#[must_use]
pub fn set(value: Value, steps: &[Step], new: Value) -> Value {
    let Some((first, rest)) = steps.split_first() else {
        return new;
    };
    if let (Value::Record(..) | Value::Struct(_) | Value::Map(_), Step::Index(index)) =
        (&value, first)
    {
        let mut keyed = vec![Step::Key(index.to_string())];
        keyed.extend_from_slice(rest);
        return set(value, &keyed, new);
    }
    match (value, first) {
        (Value::Record(name, mut fields), Step::Field(key) | Step::Key(key)) => {
            let map = Rc::make_mut(&mut fields);
            let current = map.remove(key).unwrap_or(Value::None);
            map.insert(key.clone(), set(current, rest, new));
            Value::Record(name, fields)
        }
        (Value::Struct(mut fields), Step::Field(key) | Step::Key(key)) => {
            let map = Rc::make_mut(&mut fields);
            let current = map.remove(key).unwrap_or(Value::None);
            map.insert(key.clone(), set(current, rest, new));
            Value::Struct(fields)
        }
        (Value::Map(mut fields), Step::Field(key) | Step::Key(key)) => {
            let map = Rc::make_mut(&mut fields);
            let current = map.remove(key).unwrap_or(Value::None);
            map.insert(key.clone(), set(current, rest, new));
            Value::Map(fields)
        }
        (Value::List(mut items), Step::Index(index)) => {
            let list = Rc::make_mut(&mut items);
            if let Some(slot) = list.get_mut(*index) {
                let current = std::mem::replace(slot, Value::None);
                *slot = set(current, rest, new);
            }
            Value::List(items)
        }
        (Value::None, Step::Field(key)) => {
            let mut map = BTreeMap::new();
            map.insert(key.clone(), set(Value::None, rest, new));
            Value::Struct(Rc::new(map))
        }
        (Value::None, Step::Key(key)) => {
            let mut map = BTreeMap::new();
            map.insert(key.clone(), set(Value::None, rest, new));
            Value::Map(Rc::new(map))
        }
        (other, _) => other,
    }
}

/// The value at `steps`.
#[must_use]
pub fn get(value: &Value, steps: &[Step]) -> Value {
    let Some((first, rest)) = steps.split_first() else {
        return value.clone();
    };
    let next = match (value, first) {
        (
            Value::Record(_, fields) | Value::Struct(fields) | Value::Map(fields),
            Step::Field(key) | Step::Key(key),
        ) => fields.get(key).cloned().unwrap_or(Value::None),
        (Value::List(items), Step::Index(index)) => {
            items.get(*index).cloned().unwrap_or(Value::None)
        }
        (
            Value::Record(_, fields) | Value::Struct(fields) | Value::Map(fields),
            Step::Index(index),
        ) => fields
            .get(&index.to_string())
            .cloned()
            .unwrap_or(Value::None),
        _ => Value::None,
    };
    get(&next, rest)
}

/// Reads a document path (`stops[1].address.postal`) as steps.
#[must_use]
pub fn steps_of(path: &str) -> Vec<Step> {
    let mut steps = Vec::new();
    for part in path.split('.') {
        let mut rest = part;
        if let Some(open) = rest.find('[') {
            let name = &rest[..open];
            if !name.is_empty() {
                steps.push(Step::Field(name.to_owned()));
            }
            rest = &rest[open..];
            while let Some(stripped) = rest.strip_prefix('[') {
                let Some(close) = stripped.find(']') else {
                    break;
                };
                let inner = &stripped[..close];
                match inner.parse::<usize>() {
                    Ok(index) => steps.push(Step::Index(index)),
                    Err(_) => steps.push(Step::Key(inner.to_owned())),
                }
                rest = &stripped[close + 1..];
            }
        } else if !rest.is_empty() {
            steps.push(Step::Field(rest.to_owned()));
        }
    }
    steps
}
