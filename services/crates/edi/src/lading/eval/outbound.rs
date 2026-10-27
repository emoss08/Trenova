//! Runs a map that writes X12 from a document: each segment block becomes a segment of the
//! loop instance it is written in, values convert by their element's type, and the segments
//! are laid out in the guide's order inside the envelope Trenova writes.

use rust_decimal::{Decimal, RoundingStrategy};
use serde::Serialize;
use trenova_module_kit::message;
use trenova_module_kit::message::{Arg, Message};

use super::interp::{Eval, Frame, Interp, Lineage};
use super::value::{Value, decimal_text, display, local};
use crate::lading::check::x12 as nav;
use crate::lading::syntax::ast::{Block, ElementAssign, Stmt};
use crate::lading::tz::civil_from_days;
use crate::x12::guide::compiled::{CompiledGuide, NodeKind};
use crate::x12::guide::model::{ElementDef, ElementType};
use crate::x12::write::Element;

/// Something wrong with what a run produced.
#[derive(Debug, Clone, PartialEq, Eq, Serialize)]
#[serde(rename_all = "camelCase")]
pub struct RunFinding {
    /// `error` or `warning`.
    pub severity: &'static str,
    /// A stable code.
    pub code: &'static str,
    /// What is wrong.
    pub message: Message,
    /// Where: an output path such as `0205[1].AT701`.
    #[serde(skip_serializing_if = "String::is_empty")]
    pub path: String,
}

/// A segment written.
#[derive(Debug, Clone)]
struct Written {
    node: usize,
    elements: Vec<Element>,
}

/// A loop instance written.
#[derive(Debug, Clone, Default)]
struct OutInstance {
    node: Option<usize>,
    segments: Vec<Written>,
    loops: Vec<usize>,
}

/// What an outbound map wrote, before it is laid out.
#[derive(Debug, Default)]
pub struct Output {
    instances: Vec<OutInstance>,
    /// What was wrong with the values written.
    pub findings: Vec<RunFinding>,
}

impl Interp<'_> {
    /// Runs an outbound map's statements on a document.
    pub fn write_document(
        &mut self,
        frame: &mut Frame,
        guide: usize,
        stmts: &[&Stmt],
    ) -> Eval<Output> {
        let mut output = Output {
            instances: vec![OutInstance::default()],
            findings: Vec::new(),
        };
        for stmt in stmts {
            self.write_stmt(frame, guide, stmt, &mut output, 0)?;
        }
        Ok(output)
    }

    fn write_block(
        &mut self,
        frame: &mut Frame,
        guide: usize,
        block: &Block,
        output: &mut Output,
        instance: usize,
    ) -> Eval<()> {
        frame.push();
        for stmt in &block.stmts {
            let result = self.write_stmt(frame, guide, stmt, output, instance);
            if result.is_err() {
                frame.pop();
                return result;
            }
        }
        frame.pop();
        Ok(())
    }

    #[allow(clippy::too_many_lines)]
    fn write_stmt(
        &mut self,
        frame: &mut Frame,
        guide: usize,
        stmt: &Stmt,
        output: &mut Output,
        instance: usize,
    ) -> Eval<()> {
        let program = self.program;
        let compiled = &program.guides[guide].compiled;
        let parent = output.instances[instance].node;
        match stmt {
            Stmt::Segment {
                id,
                cond,
                elements,
                span,
            } => {
                self.cover(frame.unit, span.start);
                if let Some(cond) = cond
                    && !self.eval(frame, cond)?.truthy()
                {
                    return Ok(());
                }
                let Some(node) = nav::child_segment(compiled, parent, &id.name) else {
                    return Ok(());
                };
                let ordinal = output.instances[instance]
                    .segments
                    .iter()
                    .filter(|written| written.node == node)
                    .count()
                    + 1;
                let base = instance_path(output, compiled, instance);
                let segment_path = format!(
                    "{base}{}{}",
                    if base.is_empty() { "" } else { "." },
                    segment_name(compiled, node, &id.name, ordinal)
                );
                let written = self.write_elements(
                    frame,
                    compiled,
                    node,
                    &id.name,
                    elements,
                    &segment_path,
                    &mut output.findings,
                )?;
                output.instances[instance].segments.push(Written {
                    node,
                    elements: written,
                });
            }
            Stmt::Loop {
                id,
                each,
                body,
                span,
            } => {
                self.cover(frame.unit, span.start);
                let Some(node) = nav::child_loop(compiled, parent, &id.name) else {
                    return Ok(());
                };
                if let Some((list, binder)) = each {
                    let items = self.eval(frame, list)?;
                    for (index, item) in items.items().iter().enumerate() {
                        let child = new_instance(output, instance, node);
                        frame.push();
                        frame.bind(&binder.name, item.clone());
                        frame.bind("index", Value::Int(i64::try_from(index).unwrap_or(0)));
                        let result = self.write_block(frame, guide, body, output, child);
                        frame.pop();
                        result?;
                    }
                } else {
                    let child = new_instance(output, instance, node);
                    self.write_block(frame, guide, body, output, child)?;
                }
            }
            Stmt::Each {
                list,
                binder,
                body,
                span,
            } => {
                self.cover(frame.unit, span.start);
                let items = self.eval(frame, list)?;
                let entries: Vec<Value> = match &items {
                    Value::Map(fields) => fields
                        .iter()
                        .map(|(key, value)| {
                            let mut pair = std::collections::BTreeMap::new();
                            pair.insert("key".to_owned(), Value::text(key));
                            pair.insert("value".to_owned(), value.clone());
                            Value::Struct(std::rc::Rc::new(pair))
                        })
                        .collect(),
                    other => other.items().to_vec(),
                };
                for (index, item) in entries.into_iter().enumerate() {
                    frame.push();
                    frame.bind(&binder.name, item);
                    frame.bind("index", Value::Int(i64::try_from(index).unwrap_or(0)));
                    let result = self.write_block(frame, guide, body, output, instance);
                    frame.pop();
                    result?;
                }
            }
            Stmt::When { cond, body, span } => {
                self.cover(frame.unit, span.start);
                if self.eval(frame, cond)?.truthy() {
                    self.write_block(frame, guide, body, output, instance)?;
                }
            }
            Stmt::Let(binding) => {
                let value = self.eval(frame, &binding.value)?;
                frame.bind(&binding.name.name, value);
            }
            _ => {}
        }
        Ok(())
    }

    #[allow(clippy::too_many_arguments)]
    fn write_elements(
        &mut self,
        frame: &mut Frame,
        compiled: &CompiledGuide,
        node: usize,
        id: &str,
        elements: &[ElementAssign],
        path: &str,
        findings: &mut Vec<RunFinding>,
    ) -> Eval<Vec<Element>> {
        let definition = nav::definition(compiled, node);
        let count = definition.map_or(0, |definition| definition.elements.len());
        let mut written: Vec<Element> = vec![Element::Text(String::new()); count];
        if let Some(definition) = definition {
            for (offset, element) in definition.elements.iter().enumerate() {
                if let Some(fixed) = &element.fixed {
                    written[offset] = Element::Text(fixed.clone());
                }
            }
        }
        for assign in elements {
            let (position, component) = assign.position;
            let Some(offset) = position.checked_sub(1).map(|offset| offset as usize) else {
                continue;
            };
            let Some(def) = definition.and_then(|definition| definition.elements.get(offset))
            else {
                continue;
            };
            let element_path = format!(
                "{path}{position:02}{}",
                component.map_or_else(String::new, |component| format!("-{component:02}"))
            );
            self.begin_trace();
            let value = self.eval(frame, &assign.value)?;
            let (target, label) = match component {
                Some(component) => {
                    if let Some(component_def) = def.components.as_deref().and_then(|components| {
                        components.get((component as usize).saturating_sub(1))
                    }) {
                        (component_def, element_path.clone())
                    } else {
                        self.end_trace("", &Value::None);
                        continue;
                    }
                }
                None => (def, element_path.clone()),
            };
            let text = element_text(
                &value,
                target,
                self.program.partner_separators().get(3).copied(),
            );
            self.end_trace(&element_path, &value);
            let Some(text) = text else {
                continue;
            };
            check_element(target, &text, &label, findings);
            match component {
                Some(component) => {
                    let index = (component as usize).saturating_sub(1);
                    let mut parts =
                        match std::mem::replace(&mut written[offset], Element::Text(String::new()))
                        {
                            Element::Components(parts) => parts,
                            Element::Text(text) if text.is_empty() => Vec::new(),
                            Element::Text(text) => vec![text],
                        };
                    if parts.len() <= index {
                        parts.resize(index + 1, String::new());
                    }
                    parts[index] = text;
                    written[offset] = Element::Components(parts);
                }
                None => written[offset] = Element::Text(text),
            }
        }
        let _ = id;
        Ok(written)
    }

    /// Lays out what an outbound map wrote in the guide's order: the transaction's segments
    /// between ST and SE, each its ID first.
    #[must_use]
    pub fn lay_out(&self, guide: usize, output: &Output) -> Vec<Vec<Element>> {
        let compiled = &self.program.guides[guide].compiled;
        let mut segments = Vec::new();
        lay_out_instance(compiled, output, 0, &mut segments);
        segments
    }

    /// Records the lineage of a written element.
    pub fn note_written(&mut self, path: &str, value: &Value) {
        self.lineage.push(Lineage {
            path: path.to_owned(),
            sources: Vec::new(),
            steps: Vec::new(),
            value: display(value),
        });
    }
}

fn new_instance(output: &mut Output, parent: usize, node: usize) -> usize {
    let index = output.instances.len();
    output.instances.push(OutInstance {
        node: Some(node),
        segments: Vec::new(),
        loops: Vec::new(),
    });
    output.instances[parent].loops.push(index);
    index
}

fn segment_name(compiled: &CompiledGuide, node: usize, id: &str, ordinal: usize) -> String {
    if compiled.node(node).max > 1 {
        format!("{id}[{ordinal}]")
    } else {
        id.to_owned()
    }
}

/// How lineage names a written loop instance: `0205[1].0210`.
fn instance_path(output: &Output, compiled: &CompiledGuide, instance: usize) -> String {
    let mut chain = Vec::new();
    let mut current = instance;
    'outer: while current != 0 {
        for (index, candidate) in output.instances.iter().enumerate() {
            if let Some(position) = candidate.loops.iter().position(|child| *child == current) {
                let node = output.instances[current].node.unwrap_or(0);
                let ordinal = candidate.loops[..=position]
                    .iter()
                    .filter(|child| output.instances[**child].node == Some(node))
                    .count();
                let id = match &compiled.node(node).kind {
                    NodeKind::Loop { id, .. } | NodeKind::Segment(id) => id.clone(),
                };
                chain.push(if compiled.node(node).max > 1 {
                    format!("{id}[{ordinal}]")
                } else {
                    id
                });
                current = index;
                continue 'outer;
            }
        }
        break;
    }
    chain.reverse();
    chain.join(".")
}

fn lay_out_instance(
    compiled: &CompiledGuide,
    output: &Output,
    instance: usize,
    segments: &mut Vec<Vec<Element>>,
) {
    let current = &output.instances[instance];
    let children = nav::children(compiled, current.node);
    for child in children {
        match &compiled.node(*child).kind {
            NodeKind::Segment(id) => {
                if current.node.is_none() && matches!(id.as_str(), "ST" | "SE") {
                    continue;
                }
                for written in current
                    .segments
                    .iter()
                    .filter(|written| written.node == *child)
                {
                    let mut segment = vec![Element::Text(id.clone())];
                    segment.extend(written.elements.iter().cloned());
                    while matches!(segment.last(), Some(Element::Text(text)) if text.is_empty())
                        && segment.len() > 1
                    {
                        segment.pop();
                    }
                    segments.push(segment);
                }
            }
            NodeKind::Loop { .. } => {
                for loop_instance in current
                    .loops
                    .iter()
                    .filter(|loop_instance| output.instances[**loop_instance].node == Some(*child))
                {
                    lay_out_instance(compiled, output, *loop_instance, segments);
                }
            }
        }
    }
}

/// A value written as an element of a type; none when the value is none.
#[must_use]
pub fn element_text(value: &Value, def: &ElementDef, repetition: Option<char>) -> Option<String> {
    if let Value::List(items) = value {
        let parts: Vec<String> = items
            .iter()
            .filter_map(|item| element_text(item, def, repetition))
            .collect();
        if parts.is_empty() {
            return None;
        }
        return Some(parts.join(&repetition.unwrap_or('^').to_string()));
    }
    let text = match (value, def.kind) {
        (Value::None, _) => return None,
        (Value::Date(days), ElementType::Date) => date_element(*days, def.max),
        (Value::DateTime(instant, zone), ElementType::Date) => {
            date_element(local(*instant, *zone).0, def.max)
        }
        (Value::DateTime(instant, zone), ElementType::Time) => {
            time_element(local(*instant, *zone).1, def.max)
        }
        (Value::Time(seconds), ElementType::Time) => time_element(*seconds, def.max),
        (number, ElementType::Numeric(places)) => {
            let amount = match number {
                Value::Money(amount, _) => *amount,
                other => other.as_decimal()?,
            };
            let shifted = (amount * Decimal::from(10_i64.pow(u32::from(places))))
                .round_dp_with_strategy(0, RoundingStrategy::MidpointAwayFromZero);
            let mut digits = decimal_text(shifted.abs());
            let minimum = def.min as usize;
            if digits.len() < minimum {
                digits = format!("{}{digits}", "0".repeat(minimum - digits.len()));
            }
            if shifted.is_sign_negative() && !shifted.is_zero() {
                format!("-{digits}")
            } else {
                digits
            }
        }
        (number, ElementType::Decimal) => {
            let amount = match number {
                Value::Money(amount, _) => *amount,
                other => other.as_decimal()?,
            };
            decimal_text(amount)
        }
        (Value::Bool(flag), _) => (if *flag { "Y" } else { "N" }).to_owned(),
        (other, _) => display(other),
    };
    Some(text)
}

fn date_element(days: i64, max: u32) -> String {
    let (year, month, day) = civil_from_days(days);
    if max == 6 {
        format!("{:02}{month:02}{day:02}", year.rem_euclid(100))
    } else {
        format!("{year:04}{month:02}{day:02}")
    }
}

fn time_element(seconds: u32, max: u32) -> String {
    let (hours, minutes, rest) = (seconds / 3600, seconds / 60 % 60, seconds % 60);
    if rest != 0 && max >= 6 {
        format!("{hours:02}{minutes:02}{rest:02}")
    } else {
        format!("{hours:02}{minutes:02}")
    }
}

/// Reports a written value that does not fit its element.
fn check_element(def: &ElementDef, text: &str, path: &str, findings: &mut Vec<RunFinding>) {
    let length = text.trim_start_matches('-').chars().count();
    if length > def.max as usize {
        findings.push(RunFinding {
            severity: "error",
            code: "element-too-long",
            message: Message::with(
                message!("{0} is {1} characters, and the element takes at most {2}"),
                [
                    Arg::from(text),
                    Arg::from(length),
                    Arg::from(def.max as usize),
                ],
            ),
            path: path.to_owned(),
        });
    }
    if let Some(codes) = &def.codes
        && def.kind == ElementType::Identifier
        && !codes.iter().any(|code| code.code == text)
    {
        findings.push(RunFinding {
            severity: "error",
            code: "code-not-in-list",
            message: Message::with(
                message!("{0} is not one of the element's codes"),
                [Arg::from(text)],
            ),
            path: path.to_owned(),
        });
    }
}

#[cfg(test)]
mod tests {
    use rust_decimal::Decimal;

    use super::element_text;
    use crate::lading::eval::value::Value;
    use crate::x12::guide::model::{ElementDef, ElementType, Usage};

    fn def(kind: ElementType, min: u32, max: u32) -> ElementDef {
        ElementDef {
            reference: String::new(),
            name: String::new(),
            kind,
            min,
            max,
            usage: Usage::Optional,
            repeat: 1,
            codes: None,
            fixed: None,
            components: None,
            notes: String::new(),
        }
    }

    #[test]
    fn values_convert_by_element_type() {
        let money = Value::Money(Decimal::new(125_050, 2), "USD".into());
        assert_eq!(
            element_text(&money, &def(ElementType::Numeric(2), 1, 12), None).as_deref(),
            Some("125050")
        );
        assert_eq!(
            element_text(
                &Value::Decimal(Decimal::new(125_050, 2)),
                &def(ElementType::Decimal, 1, 10),
                None
            )
            .as_deref(),
            Some("1250.5")
        );
        assert_eq!(
            element_text(&Value::Date(20_463), &def(ElementType::Date, 6, 6), None).as_deref(),
            Some("260110")
        );
        assert_eq!(
            element_text(&Value::Int(7), &def(ElementType::Numeric(0), 3, 6), None).as_deref(),
            Some("007")
        );
    }
}
