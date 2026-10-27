//! Finding one's way around a guide's structure: the segments and loops under a node, an
//! element's type and code list, and how a written reference (`N104`, `0310`) names them.

use crate::lading::types::{NodeRef, Place, Type};
use crate::x12::guide::compiled::{CompiledGuide, NodeKind};
use crate::x12::guide::model::{ElementDef, ElementType, SegmentDef, Usage};

/// What a written X12 reference names.
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum Reference<'a> {
    /// A segment (`N1`).
    Segment(&'a str),
    /// A segment's element (`N104`).
    Element(&'a str, u32),
}

/// Reads `N1`, `N104`, `B2A01`, `G6202`.
#[must_use]
pub fn reference(text: &str) -> Option<Reference<'_>> {
    let bytes = text.as_bytes();
    if !bytes.first().is_some_and(u8::is_ascii_uppercase)
        || !bytes
            .iter()
            .all(|byte| byte.is_ascii_uppercase() || byte.is_ascii_digit())
    {
        return None;
    }
    match bytes.len() {
        2 | 3 => Some(Reference::Segment(text)),
        4 | 5 => {
            let (segment, position) = text.split_at(text.len() - 2);
            let position: u32 = position.parse().ok()?;
            (position > 0).then_some(Reference::Element(segment, position))
        }
        _ => None,
    }
}

/// The children of a node, or the top of the structure for none.
#[must_use]
pub fn children(guide: &CompiledGuide, node: Option<usize>) -> &[usize] {
    match node {
        Some(index) => &guide.node(index).children,
        None => guide.root(),
    }
}

/// The segment use with an ID among a node's children.
#[must_use]
pub fn child_segment(guide: &CompiledGuide, node: Option<usize>, id: &str) -> Option<usize> {
    children(guide, node).iter().copied().find(
        |index| matches!(&guide.node(*index).kind, NodeKind::Segment(segment) if segment == id),
    )
}

/// The loop with an ID among a node's children.
#[must_use]
pub fn child_loop(guide: &CompiledGuide, node: Option<usize>, id: &str) -> Option<usize> {
    children(guide, node)
        .iter()
        .copied()
        .find(|index| matches!(&guide.node(*index).kind, NodeKind::Loop { id: loop_id, .. } if loop_id == id))
}

/// Where a segment ID is used anywhere in the structure: the loop path of its first use.
#[must_use]
pub fn find_segment_anywhere(guide: &CompiledGuide, id: &str) -> Option<Vec<String>> {
    fn search(guide: &CompiledGuide, nodes: &[usize], id: &str, path: &mut Vec<String>) -> bool {
        for index in nodes {
            match &guide.node(*index).kind {
                NodeKind::Segment(segment) if segment == id => return true,
                NodeKind::Loop { id: loop_id, .. } => {
                    path.push(loop_id.clone());
                    if search(guide, &guide.node(*index).children, id, path) {
                        return true;
                    }
                    path.pop();
                }
                NodeKind::Segment(_) => {}
            }
        }
        false
    }
    let mut path = Vec::new();
    search(guide, guide.root(), id, &mut path).then_some(path)
}

/// The definition of the segment at a node.
#[must_use]
pub fn definition(guide: &CompiledGuide, node: usize) -> Option<&SegmentDef> {
    match &guide.node(node).kind {
        NodeKind::Segment(id) => guide.definition(Some(node), id),
        NodeKind::Loop { .. } => None,
    }
}

/// The segment ID of a node.
#[must_use]
pub fn segment_id(guide: &CompiledGuide, node: usize) -> &str {
    guide.node(node).segment_id()
}

/// The type of one element.
#[must_use]
pub fn element_type(segment: &str, position: u32, element: &ElementDef, place: Place) -> Type {
    let base = match element.kind {
        ElementType::Alphanumeric | ElementType::Binary => Type::Text,
        ElementType::Identifier => Type::Code(format!("{segment}{position:02}"), place),
        ElementType::Date => Type::Date,
        ElementType::Time => Type::Time,
        ElementType::Numeric(0) => Type::Int,
        ElementType::Numeric(_) | ElementType::Decimal => Type::Decimal,
        ElementType::Composite => Type::Struct(
            element
                .components
                .as_deref()
                .unwrap_or_default()
                .iter()
                .enumerate()
                .map(|(index, component)| {
                    let position = u32::try_from(index + 1).unwrap_or(1);
                    let ty = element_type(segment, position, component, place);
                    (
                        format!("{position:02}"),
                        if component.usage == Usage::Mandatory {
                            ty
                        } else {
                            ty.optional()
                        },
                    )
                })
                .collect(),
        ),
    };
    if element.repeat > 1 {
        base.list()
    } else {
        base
    }
}

/// The type of reading element `position` of the segment at `reference`.
#[must_use]
pub fn read_element(guide: &CompiledGuide, reference: NodeRef, position: u32) -> Option<Type> {
    let node = reference.node;
    let definition = definition(guide, node)?;
    let element = definition.elements.get(position.checked_sub(1)? as usize)?;
    let ty = element_type(
        segment_id(guide, node),
        position,
        element,
        Place(Some(reference)),
    );
    Some(if element.usage == Usage::Mandatory {
        ty
    } else {
        ty.optional()
    })
}

/// The type of reading a child (segment or loop) of a node.
#[must_use]
pub fn child_type(guide: &CompiledGuide, guide_index: usize, child: usize) -> Type {
    let node = guide.node(child);
    let reference = NodeRef {
        guide: guide_index,
        node: child,
    };
    let single = match node.kind {
        NodeKind::Segment(_) => Type::Segment(reference),
        NodeKind::Loop { .. } => Type::Loop(reference),
    };
    if node.max > 1 {
        single.list()
    } else if node.usage == Usage::Mandatory {
        single
    } else {
        single.optional()
    }
}

/// The codes an element may hold under a guide: at `node` when it is known, else at the first
/// use of its segment, or the dictionary's.
#[must_use]
pub fn codes(
    guide: &CompiledGuide,
    node: Option<usize>,
    segment: &str,
    position: u32,
) -> Option<Vec<(String, String)>> {
    let node = node.or_else(|| first_use(guide, segment));
    let definition = guide.definition(node, segment)?;
    let element = definition.elements.get(position.checked_sub(1)? as usize)?;
    element.codes.as_ref().map(|codes| {
        codes
            .iter()
            .map(|code| (code.code.clone(), code.description.clone()))
            .collect()
    })
}

/// The first node using a segment.
#[must_use]
pub fn first_use(guide: &CompiledGuide, segment: &str) -> Option<usize> {
    (0..guide.node_count())
        .find(|index| matches!(&guide.node(*index).kind, NodeKind::Segment(id) if id == segment))
}

/// A loop node's ID.
#[must_use]
pub fn loop_id(guide: &CompiledGuide, node: usize) -> Option<&str> {
    match &guide.node(node).kind {
        NodeKind::Loop { id, .. } => Some(id),
        NodeKind::Segment(_) => None,
    }
}

#[cfg(test)]
mod tests {
    use super::{Reference, reference};

    #[test]
    fn references_split_segment_and_position() {
        assert_eq!(reference("N1"), Some(Reference::Segment("N1")));
        assert_eq!(reference("AT7"), Some(Reference::Segment("AT7")));
        assert_eq!(reference("N104"), Some(Reference::Element("N1", 4)));
        assert_eq!(reference("B2A01"), Some(Reference::Element("B2A", 1)));
        assert_eq!(reference("G6202"), Some(Reference::Element("G62", 2)));
        assert_eq!(reference("ZZ"), Some(Reference::Segment("ZZ")));
        assert_eq!(reference("N100"), None);
    }
}
