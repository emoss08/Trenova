//! A transaction set read with its guide, as loop instances and segments a map walks: which
//! segments and loops are under each instance, each element decoded by its type, and where
//! every element sits, for lineage, element notes and unread data.

use std::cell::RefCell;
use std::collections::BTreeSet;
use std::rc::Rc;

use rust_decimal::Decimal;

use super::value::Value;
use crate::lading::tz::{days_from_civil, valid_date};
use crate::x12::guide::compiled::{CompiledGuide, NodeKind};
use crate::x12::guide::model::{ElementDef, ElementType};
use crate::x12::structure::walk;

/// A loop instance's index in its tree; 0 is the transaction's top.
pub type Node = usize;

/// One segment of the transaction.
#[derive(Debug, Clone)]
pub struct Segment {
    /// Its ID.
    pub id: String,
    /// Where the guide placed it.
    pub node: Option<usize>,
    /// Its elements as written, element 01 first.
    pub elements: Vec<String>,
    /// Its position in the transaction set, ST being 1.
    pub position: u32,
    /// The loop instance it is in.
    pub instance: Node,
    /// How lineage names it: `0300[2].0310.N1`.
    pub path: String,
}

/// One loop instance.
#[derive(Debug, Clone, Default)]
pub struct Instance {
    /// The guide's loop node; none for the top.
    pub node: Option<usize>,
    /// Its segments, in order.
    pub segments: Vec<usize>,
    /// Its loop instances, in order.
    pub loops: Vec<Node>,
    /// How lineage names it: `0300[2]`.
    pub path: String,
}

/// A transaction read with a guide.
#[derive(Debug)]
pub struct Tree {
    /// The guide's index in the program.
    pub guide: usize,
    /// The guide.
    pub compiled: CompiledGuide,
    /// The segments, ST first.
    pub segments: Vec<Segment>,
    /// The loop instances, the top first.
    pub instances: Vec<Instance>,
    /// The component separator.
    pub component: char,
    /// The repetition separator.
    pub repetition: Option<char>,
    /// The elements a map read: segment index and element position.
    pub read: RefCell<BTreeSet<(usize, u32)>>,
}

/// One segment to place: its ID, its elements and its position in the set.
#[derive(Debug)]
pub struct Raw {
    /// The ID.
    pub id: String,
    /// The elements as written.
    pub elements: Vec<String>,
    /// The position, ST being 1.
    pub position: u32,
}

fn parents(guide: &CompiledGuide) -> Vec<Option<usize>> {
    let mut parents = vec![None; guide.node_count()];
    for index in 0..guide.node_count() {
        for child in &guide.node(index).children {
            parents[*child] = Some(index);
        }
    }
    parents
}

impl Tree {
    /// Places a transaction's segments in loop instances.
    #[allow(clippy::too_many_lines)]
    #[must_use]
    pub fn build(
        guide_index: usize,
        compiled: CompiledGuide,
        raw: Vec<Raw>,
        component: char,
        repetition: Option<char>,
    ) -> Self {
        let ids: Vec<&str> = raw.iter().map(|segment| segment.id.as_str()).collect();
        let placements = walk(&compiled, &ids).placements;
        let parent_of = parents(&compiled);
        let mut instances = vec![Instance::default()];
        let mut segments = Vec::with_capacity(raw.len());
        let mut stack: Vec<(usize, Node)> = Vec::new();
        for (raw_segment, placement) in raw.into_iter().zip(placements) {
            let node = placement.node;
            let mut ancestors = Vec::new();
            let mut cursor = node.and_then(|node| parent_of[node]);
            while let Some(loop_node) = cursor {
                ancestors.push(loop_node);
                cursor = parent_of[loop_node];
            }
            ancestors.reverse();
            let starts = node.is_some_and(|node| {
                parent_of[node]
                    .is_some_and(|parent| compiled.node(parent).children.first() == Some(&node))
            });
            let mut keep = stack
                .iter()
                .zip(&ancestors)
                .take_while(|((open, _), wanted)| open == *wanted)
                .count();
            if starts && keep == ancestors.len() {
                keep = keep.saturating_sub(1);
            }
            if node.is_none() {
                keep = stack.len();
            }
            stack.truncate(keep);
            for loop_node in ancestors.iter().skip(keep) {
                let parent = stack.last().map_or(0, |(_, instance)| *instance);
                let ordinal = instances[parent]
                    .loops
                    .iter()
                    .filter(|child| instances[**child].node == Some(*loop_node))
                    .count()
                    + 1;
                let id = match &compiled.node(*loop_node).kind {
                    NodeKind::Loop { id, .. } | NodeKind::Segment(id) => id.clone(),
                };
                let name = if compiled.node(*loop_node).max > 1 {
                    format!("{id}[{ordinal}]")
                } else {
                    id
                };
                let parent_path = &instances[parent].path;
                let path = if parent_path.is_empty() {
                    name
                } else {
                    format!("{parent_path}.{name}")
                };
                let index = instances.len();
                instances.push(Instance {
                    node: Some(*loop_node),
                    segments: Vec::new(),
                    loops: Vec::new(),
                    path,
                });
                instances[parent].loops.push(index);
                stack.push((*loop_node, index));
            }
            let instance = stack.last().map_or(0, |(_, instance)| *instance);
            let ordinal = instances[instance]
                .segments
                .iter()
                .filter(|other| {
                    segments
                        .get(**other)
                        .is_some_and(|other: &Segment| other.id == raw_segment.id)
                })
                .count()
                + 1;
            let repeats = node.is_some_and(|node| compiled.node(node).max > 1);
            let name = if repeats {
                format!("{}[{ordinal}]", raw_segment.id)
            } else {
                raw_segment.id.clone()
            };
            let parent_path = &instances[instance].path;
            let path = if parent_path.is_empty() {
                name
            } else {
                format!("{parent_path}.{name}")
            };
            let index = segments.len();
            instances[instance].segments.push(index);
            segments.push(Segment {
                id: raw_segment.id,
                node,
                elements: raw_segment.elements,
                position: raw_segment.position,
                instance,
                path,
            });
        }
        Self {
            guide: guide_index,
            compiled,
            segments,
            instances,
            component,
            repetition,
            read: RefCell::new(BTreeSet::new()),
        }
    }

    /// The segments with an ID directly in an instance.
    #[must_use]
    pub fn segments_in(&self, instance: Node, id: &str) -> Vec<usize> {
        self.instances[instance]
            .segments
            .iter()
            .copied()
            .filter(|index| self.segments[*index].id == id)
            .collect()
    }

    /// The instances of a loop directly in an instance.
    #[must_use]
    pub fn loops_in(&self, instance: Node, id: &str) -> Vec<Node> {
        self.instances[instance]
            .loops
            .iter()
            .copied()
            .filter(|child| {
                self.instances[*child]
                    .node
                    .is_some_and(|node| matches!(&self.compiled.node(node).kind, NodeKind::Loop { id: loop_id, .. } if loop_id == id))
            })
            .collect()
    }

    /// A segment as text: `N1*SH*ACME DC`.
    #[must_use]
    pub fn segment_text(&self, index: usize) -> String {
        let segment = &self.segments[index];
        let mut text = segment.id.clone();
        for element in &segment.elements {
            text.push('*');
            text.push_str(element);
        }
        text
    }

    /// The definition of an element of a segment.
    fn definition(&self, index: usize, position: u32) -> Option<&ElementDef> {
        let segment = &self.segments[index];
        let definition = self.compiled.definition(segment.node, &segment.id)?;
        definition
            .elements
            .get(usize::try_from(position.checked_sub(1)?).ok()?)
    }

    /// Element `position` of a segment, decoded by its type; marks it read.
    #[must_use]
    pub fn element(&self, index: usize, position: u32) -> Value {
        self.read.borrow_mut().insert((index, position));
        let Some(raw) = position
            .checked_sub(1)
            .and_then(|offset| self.segments[index].elements.get(offset as usize))
        else {
            return Value::None;
        };
        let Some(definition) = self.definition(index, position) else {
            return if raw.is_empty() {
                Value::None
            } else {
                Value::text(raw.trim_end())
            };
        };
        if definition.repeat > 1
            && let Some(separator) = self.repetition
        {
            let items: Vec<Value> = raw
                .split(separator)
                .map(|part| self.decode(definition, part))
                .filter(|value| !value.is_none())
                .collect();
            return if items.is_empty() {
                Value::None
            } else {
                Value::list(items)
            };
        }
        self.decode(definition, raw)
    }

    fn decode(&self, definition: &ElementDef, raw: &str) -> Value {
        if definition.kind == ElementType::Composite {
            let components = definition.components.as_deref().unwrap_or_default();
            let mut fields = std::collections::BTreeMap::new();
            for (offset, part) in raw.split(self.component).enumerate() {
                let key = format!("{:02}", offset + 1);
                let value = components.get(offset).map_or_else(
                    || {
                        if part.is_empty() {
                            Value::None
                        } else {
                            Value::text(part)
                        }
                    },
                    |component| decode_simple(component.kind, part),
                );
                fields.insert(key, value);
            }
            for offset in fields.len()..components.len() {
                fields.insert(format!("{:02}", offset + 1), Value::None);
            }
            return if raw.is_empty() {
                Value::None
            } else {
                Value::Struct(Rc::new(fields))
            };
        }
        decode_simple(definition.kind, raw)
    }

    /// Every element present in the transaction: segment index and position.
    #[must_use]
    pub fn present(&self) -> Vec<(usize, u32)> {
        let mut present = Vec::new();
        for (index, segment) in self.segments.iter().enumerate() {
            if matches!(segment.id.as_str(), "ST" | "SE") {
                continue;
            }
            for (offset, element) in segment.elements.iter().enumerate() {
                if !element.is_empty() {
                    present.push((index, u32::try_from(offset + 1).unwrap_or(0)));
                }
            }
        }
        present
    }
}

/// A simple element decoded by its type; none when empty or not of its type.
#[must_use]
pub fn decode_simple(kind: ElementType, raw: &str) -> Value {
    let raw = raw.trim_end();
    if raw.is_empty() {
        return Value::None;
    }
    match kind {
        ElementType::Alphanumeric
        | ElementType::Binary
        | ElementType::Identifier
        | ElementType::Composite => Value::text(raw),
        ElementType::Date => read_date(raw).map_or(Value::None, Value::Date),
        ElementType::Time => read_time(raw).map_or(Value::None, Value::Time),
        ElementType::Numeric(places) => {
            let negative = raw.starts_with('-');
            let digits = raw.trim_start_matches(['-', '+']);
            if digits.is_empty() || !digits.bytes().all(|byte| byte.is_ascii_digit()) {
                return Value::None;
            }
            let Ok(number) = digits.parse::<i128>() else {
                return Value::None;
            };
            let number = if negative { -number } else { number };
            if places == 0 {
                i64::try_from(number).map_or(Value::None, Value::Int)
            } else {
                Decimal::try_from_i128_with_scale(number, u32::from(places))
                    .map_or(Value::None, Value::Decimal)
            }
        }
        ElementType::Decimal => raw.parse::<Decimal>().map_or(Value::None, Value::Decimal),
    }
}

/// `CCYYMMDD` or `YYMMDD`.
#[must_use]
pub fn read_date(raw: &str) -> Option<i64> {
    if !raw.bytes().all(|byte| byte.is_ascii_digit()) {
        return None;
    }
    let (year, rest) = match raw.len() {
        8 => (raw[..4].parse::<i64>().ok()?, &raw[4..]),
        6 => {
            let short: i64 = raw[..2].parse().ok()?;
            (
                if short < 50 {
                    2000 + short
                } else {
                    1900 + short
                },
                &raw[2..],
            )
        }
        _ => return None,
    };
    let month: u32 = rest[..2].parse().ok()?;
    let day: u32 = rest[2..4].parse().ok()?;
    valid_date(year, month, day).then(|| days_from_civil(year, month, day))
}

/// `HHMM`, `HHMMSS` or `HHMMSSd..`.
#[must_use]
pub fn read_time(raw: &str) -> Option<u32> {
    if raw.len() < 4 || !raw.bytes().all(|byte| byte.is_ascii_digit()) {
        return None;
    }
    let hours: u32 = raw[..2].parse().ok()?;
    let minutes: u32 = raw[2..4].parse().ok()?;
    let seconds: u32 = if raw.len() >= 6 {
        raw[4..6].parse().ok()?
    } else {
        0
    };
    (hours < 24 && minutes < 60 && seconds < 60).then_some(hours * 3600 + minutes * 60 + seconds)
}

#[cfg(test)]
mod tests {
    use rust_decimal::Decimal;

    use super::{decode_simple, read_date, read_time};
    use crate::lading::eval::value::{Value, equal};
    use crate::x12::guide::model::ElementType;

    #[test]
    fn elements_decode_by_type() {
        assert!(equal(
            &decode_simple(ElementType::Numeric(2), "185000"),
            &Value::Decimal(Decimal::new(185_000, 2))
        ));
        assert!(equal(
            &decode_simple(ElementType::Numeric(0), "12"),
            &Value::Int(12)
        ));
        assert!(equal(
            &decode_simple(ElementType::Decimal, "42000.5"),
            &Value::Decimal(Decimal::new(420_005, 1))
        ));
        assert!(decode_simple(ElementType::Date, "20260230").is_none());
        assert_eq!(read_date("260110"), read_date("20260110"));
        assert_eq!(read_time("0830"), Some(30_600));
        assert_eq!(read_time("2460"), None);
    }
}
