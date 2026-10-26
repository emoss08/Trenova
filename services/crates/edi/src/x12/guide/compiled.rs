//! A guide made ready for reading: its structure flattened into an arena of nodes.
//!
//! Reading a transaction set walks the structure once per segment, so the tree is kept as a
//! flat list of nodes with child indexes, and each loop knows the segment that starts it. A
//! guide is compiled only after [`super::check::check_guide`] finds nothing that would make the
//! walk ambiguous or unbounded.

use std::collections::BTreeSet;

use super::check::{GuideProblem, check_guide};
use super::model::{Guide, Node, SegmentDef, Usage};
use crate::x12::finding::{FindingKind, Severity};

/// What a node of the compiled structure is.
#[derive(Debug, Clone, PartialEq, Eq)]
pub enum NodeKind {
    /// A segment use.
    Segment(String),
    /// A loop and the segment that starts each repetition.
    Loop {
        /// The loop identifier.
        id: String,
        /// The ID of its first segment.
        trigger: String,
    },
}

/// One node of the compiled structure.
#[derive(Debug, Clone, PartialEq, Eq)]
pub struct CompiledNode {
    /// What it is.
    pub kind: NodeKind,
    /// Whether it must appear.
    pub usage: Usage,
    /// The most times it may appear (or a loop repeat).
    pub max: u32,
    /// The fewest times it must appear once it appears; 0 when the standard decides.
    pub min: u32,
    /// Its children, for a loop.
    pub children: Vec<usize>,
}

impl CompiledNode {
    /// The segment ID it is about: a use's own, or a loop's first segment.
    #[must_use]
    pub fn segment_id(&self) -> &str {
        match &self.kind {
            NodeKind::Segment(id) | NodeKind::Loop { trigger: id, .. } => id,
        }
    }

    /// Whether it is a loop.
    #[must_use]
    pub fn is_loop(&self) -> bool {
        matches!(self.kind, NodeKind::Loop { .. })
    }
}

/// A guide ready for reading.
#[derive(Debug, Clone)]
pub struct CompiledGuide {
    guide: Guide,
    nodes: Vec<CompiledNode>,
    root: Vec<usize>,
    in_structure: BTreeSet<String>,
}

impl CompiledGuide {
    /// Compiles a guide, or returns what is wrong with it.
    pub fn compile(guide: Guide) -> Result<Self, Vec<GuideProblem>> {
        let problems = check_guide(&guide, None);
        if !problems.is_empty() {
            return Err(problems);
        }
        let mut compiled = Self {
            nodes: Vec::new(),
            root: Vec::new(),
            in_structure: BTreeSet::new(),
            guide: Guide {
                structure: Vec::new(),
                ..guide.clone()
            },
        };
        compiled.root = compiled.add_nodes(&guide.structure);
        compiled.guide = guide;
        Ok(compiled)
    }

    fn add_nodes(&mut self, nodes: &[Node]) -> Vec<usize> {
        let mut indexes = Vec::with_capacity(nodes.len());
        for node in nodes {
            let index = self.nodes.len();
            match node {
                Node::Segment(segment) => {
                    self.in_structure.insert(segment.segment.clone());
                    self.nodes.push(CompiledNode {
                        kind: NodeKind::Segment(segment.segment.clone()),
                        usage: segment.usage,
                        max: segment.max,
                        min: segment.min,
                        children: Vec::new(),
                    });
                }
                Node::Loop(definition) => {
                    let trigger = match definition.children.first() {
                        Some(Node::Segment(first)) => first.segment.clone(),
                        _ => String::new(),
                    };
                    self.nodes.push(CompiledNode {
                        kind: NodeKind::Loop {
                            id: definition.id.clone(),
                            trigger,
                        },
                        usage: definition.usage,
                        max: definition.repeat,
                        min: definition.min,
                        children: Vec::new(),
                    });
                    let children = self.add_nodes(&definition.children);
                    self.nodes[index].children = children;
                }
            }
            indexes.push(index);
        }
        indexes
    }

    /// The guide as written.
    #[must_use]
    pub fn guide(&self) -> &Guide {
        &self.guide
    }

    /// A node by index.
    #[must_use]
    pub fn node(&self, index: usize) -> &CompiledNode {
        &self.nodes[index]
    }

    /// The top level of the structure, ST first.
    #[must_use]
    pub fn root(&self) -> &[usize] {
        &self.root
    }

    /// A segment's definition.
    #[must_use]
    pub fn segment(&self, id: &str) -> Option<&SegmentDef> {
        self.guide.segments.get(id)
    }

    /// Whether the structure uses the segment anywhere.
    #[must_use]
    pub fn uses_segment(&self, id: &str) -> bool {
        self.in_structure.contains(id)
    }

    /// A finding kind's severity under this guide.
    #[must_use]
    pub fn severity(&self, kind: FindingKind) -> Severity {
        self.guide
            .severities
            .get(&kind)
            .copied()
            .unwrap_or_else(|| kind.default_severity())
    }
}

#[cfg(test)]
mod tests {
    use super::{CompiledGuide, NodeKind};
    use crate::x12::guide::builtin;

    #[test]
    fn flattens_loops_and_records_their_trigger() {
        let guide = builtin::by_id("x12.004010.204").expect("the 204");
        let stop = guide
            .root()
            .iter()
            .map(|index| guide.node(*index))
            .find(|node| matches!(&node.kind, NodeKind::Loop { id, .. } if id == "0300"))
            .expect("the stop loop");
        assert_eq!(stop.segment_id(), "S5");
        assert_eq!(stop.max, 999);
        assert!(guide.uses_segment("LH1"));
        assert!(!guide.uses_segment("AT7"));
    }

    #[test]
    fn refuses_a_guide_with_problems() {
        let mut guide = builtin::by_id("x12.004010.990")
            .expect("the 990")
            .guide()
            .clone();
        guide.segments.remove("B1");
        assert!(CompiledGuide::compile(guide).is_err());
    }
}
