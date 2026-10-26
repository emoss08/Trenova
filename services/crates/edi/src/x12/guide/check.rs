//! Checks a guide before it is used, and a partner's guide against the base it narrows.
//!
//! A guide comes from a person, an SEF import or the onboarding agent, so nothing about it is
//! trusted: the structure must start with ST and end with SE, every loop must start with a
//! segment, every segment it uses must be defined, lengths and counts must make sense, syntax
//! notes must name elements that exist, and loops nest at most [`MAX_DEPTH`] deep so a walk
//! is bounded. A guide with a problem is never compiled.
//!
//! A partner's guide (one with `derivedFrom`) is also compared with its base: it may make
//! things stricter, never looser, so a partner guide cannot accept what the standard rejects.
//! Each widening is reported with the path it is at.

use std::collections::BTreeMap;

use serde::Serialize;
use trenova_module_kit::message;
use trenova_module_kit::message::{Arg, Message};

use super::model::{ElementDef, ElementType, Guide, Node, SegmentDef, Usage};
use super::syntax::SyntaxNote;

/// The deepest loops may nest.
pub const MAX_DEPTH: usize = 16;

/// The longest element a guide may define.
pub const MAX_ELEMENT_LENGTH: u32 = 1_000_000;

/// One problem with a guide.
#[derive(Debug, Clone, PartialEq, Eq, Serialize)]
#[serde(rename_all = "camelCase")]
pub struct GuideProblem {
    /// Where it is: `structure/0300/S5`, `segments/N1/02`, `transactionSet`.
    pub path: String,
    /// Whether it makes the guide unusable (`error`) or only widens its base (`widening`).
    pub kind: ProblemKind,
    /// What is wrong, for people.
    pub message: Message,
}

/// Whether a problem blocks the guide.
#[derive(Debug, Clone, Copy, PartialEq, Eq, Serialize)]
#[serde(rename_all = "camelCase")]
pub enum ProblemKind {
    /// The guide cannot be used.
    Error,
    /// The guide accepts something its base does not.
    Widening,
}

struct Problems(Vec<GuideProblem>);

impl Problems {
    fn error(&mut self, path: impl Into<String>, message: Message) {
        self.0.push(GuideProblem {
            path: path.into(),
            kind: ProblemKind::Error,
            message,
        });
    }

    fn widening(&mut self, path: impl Into<String>, message: Message) {
        self.0.push(GuideProblem {
            path: path.into(),
            kind: ProblemKind::Widening,
            message,
        });
    }
}

/// Every problem with `guide`, and every way it widens `base` when one is given. Empty when
/// the guide is sound.
#[must_use]
pub fn check_guide(guide: &Guide, base: Option<&Guide>) -> Vec<GuideProblem> {
    let mut problems = Problems(Vec::new());
    check_header(guide, &mut problems);
    check_structure(guide, &mut problems);
    for (id, segment) in &guide.segments {
        check_segment(id, segment, &mut problems);
    }
    if let Some(base) = base {
        check_narrowing(guide, base, &mut problems);
    }
    problems.0
}

fn check_header(guide: &Guide, problems: &mut Problems) {
    if guide.id.trim().is_empty() {
        problems.error("id", Message::new(message!("A guide needs an identifier")));
    }
    let set = guide.transaction_set.as_bytes();
    if set.len() != 3 || !set.iter().all(u8::is_ascii_digit) {
        problems.error(
            "transactionSet",
            Message::with(
                message!("The transaction set {0} is not three digits"),
                [Arg::from(guide.transaction_set.as_str())],
            ),
        );
    }
    let group = guide.functional_group.as_bytes();
    if group.len() != 2 || !group.iter().all(u8::is_ascii_uppercase) {
        problems.error(
            "functionalGroup",
            Message::with(
                message!("The functional group {0} is not two capital letters"),
                [Arg::from(guide.functional_group.as_str())],
            ),
        );
    }
    let version = guide.version.as_bytes();
    if version.len() < 6 || version.len() > 12 || !version[..6].iter().all(u8::is_ascii_digit) {
        problems.error(
            "version",
            Message::with(
                message!("The version {0} does not start with six digits"),
                [Arg::from(guide.version.as_str())],
            ),
        );
    }
}

fn check_structure(guide: &Guide, problems: &mut Problems) {
    let first = guide.structure.first();
    if !matches!(first, Some(Node::Segment(segment)) if segment.segment == "ST") {
        problems.error(
            "structure",
            Message::new(message!("The structure must start with the ST segment")),
        );
    }
    let last = guide.structure.last();
    if !matches!(last, Some(Node::Segment(segment)) if segment.segment == "SE") {
        problems.error(
            "structure",
            Message::new(message!("The structure must end with the SE segment")),
        );
    }
    check_nodes(guide, &guide.structure, "structure", 0, problems);
}

fn check_nodes(guide: &Guide, nodes: &[Node], path: &str, depth: usize, problems: &mut Problems) {
    for node in nodes {
        match node {
            Node::Segment(segment) => {
                let here = format!("{path}/{}", segment.segment);
                if !guide.segments.contains_key(&segment.segment) {
                    problems.error(
                        here.as_str(),
                        Message::with(
                            message!("The structure uses {0}, which the guide does not define"),
                            [Arg::from(segment.segment.as_str())],
                        ),
                    );
                }
                check_counts(&here, segment.max, segment.min, problems);
            }
            Node::Loop(definition) => {
                let here = format!("{path}/{}", definition.id);
                if definition.id.trim().is_empty() {
                    problems.error(
                        here.as_str(),
                        Message::new(message!("A loop needs an identifier")),
                    );
                }
                if depth + 1 > MAX_DEPTH {
                    problems.error(
                        here.as_str(),
                        Message::with(
                            message!("Loops nest more than {0} deep"),
                            [Arg::from(MAX_DEPTH)],
                        ),
                    );
                    continue;
                }
                match definition.children.first() {
                    Some(Node::Segment(first)) if first.usage == Usage::NotUsed => problems.error(
                        here.as_str(),
                        Message::with(
                            message!("Loop {0} starts with a segment that is not used"),
                            [Arg::from(definition.id.as_str())],
                        ),
                    ),
                    Some(Node::Segment(_)) => {}
                    _ => problems.error(
                        here.as_str(),
                        Message::with(
                            message!("Loop {0} must start with a segment"),
                            [Arg::from(definition.id.as_str())],
                        ),
                    ),
                }
                check_counts(&here, definition.repeat, definition.min, problems);
                check_nodes(guide, &definition.children, &here, depth + 1, problems);
            }
        }
    }
}

fn check_counts(path: &str, max: u32, min: u32, problems: &mut Problems) {
    if max == 0 {
        problems.error(
            path,
            Message::new(message!("The maximum must be at least 1")),
        );
    }
    if min > max {
        problems.error(
            path,
            Message::with(
                message!("The minimum {0} is above the maximum {1}"),
                [Arg::from(i64::from(min)), Arg::from(i64::from(max))],
            ),
        );
    }
}

fn check_segment(id: &str, segment: &SegmentDef, problems: &mut Problems) {
    let path = format!("segments/{id}");
    let bytes = id.as_bytes();
    let valid_id = (2..=3).contains(&bytes.len())
        && bytes[0].is_ascii_uppercase()
        && bytes
            .iter()
            .all(|byte| byte.is_ascii_uppercase() || byte.is_ascii_digit());
    if !valid_id {
        problems.error(
            path.as_str(),
            Message::with(
                message!("{0} is not a segment ID: two or three capital letters and digits"),
                [Arg::from(id)],
            ),
        );
    }
    for (index, element) in segment.elements.iter().enumerate() {
        let element_path = format!("{path}/{:02}", index + 1);
        check_element(&element_path, element, false, problems);
    }
    for rule in &segment.rules {
        match SyntaxNote::parse(rule) {
            None => problems.error(
                path.as_str(),
                Message::with(
                    message!("{0} is not a syntax note"),
                    [Arg::from(rule.as_str())],
                ),
            ),
            Some(note) => {
                let count = u32::try_from(segment.elements.len()).unwrap_or(u32::MAX);
                if note.positions.iter().any(|position| *position > count) {
                    problems.error(
                        path.as_str(),
                        Message::with(
                            message!(
                                "The syntax note {0} names an element the segment does not have"
                            ),
                            [Arg::from(rule.as_str())],
                        ),
                    );
                }
            }
        }
    }
}

fn check_element(path: &str, element: &ElementDef, in_composite: bool, problems: &mut Problems) {
    if element.kind == ElementType::Composite {
        if in_composite {
            problems.error(
                path,
                Message::new(message!("A composite cannot contain a composite")),
            );
        }
        let components = element.components.as_deref().unwrap_or_default();
        if components.is_empty() {
            problems.error(
                path,
                Message::new(message!("A composite needs its components")),
            );
        }
        for (index, component) in components.iter().enumerate() {
            check_element(
                &format!("{path}-{:02}", index + 1),
                component,
                true,
                problems,
            );
        }
        if element.codes.is_some() || element.fixed.is_some() {
            problems.error(
                path,
                Message::new(message!("A composite's codes belong to its components")),
            );
        }
        return;
    }
    if element.components.is_some() {
        problems.error(
            path,
            Message::new(message!("Only a composite has components")),
        );
    }
    if element.max == 0 || element.max > MAX_ELEMENT_LENGTH {
        problems.error(
            path,
            Message::with(
                message!("The maximum length must be between 1 and {0}"),
                [Arg::from(i64::from(MAX_ELEMENT_LENGTH))],
            ),
        );
    }
    if element.min > element.max {
        problems.error(
            path,
            Message::with(
                message!("The minimum length {0} is above the maximum {1}"),
                [
                    Arg::from(i64::from(element.min)),
                    Arg::from(i64::from(element.max)),
                ],
            ),
        );
    }
    if element.repeat == 0 {
        problems.error(
            path,
            Message::new(message!("An element repeats at least once")),
        );
    }
    if let Some(codes) = &element.codes
        && codes.is_empty()
    {
        problems.error(
            path,
            Message::new(message!("A code list needs at least one code")),
        );
    }
    if let Some(fixed) = &element.fixed {
        let length = u32::try_from(fixed.chars().count()).unwrap_or(u32::MAX);
        let fits = length >= element.min.max(1) && length <= element.max;
        let listed = element
            .codes
            .as_ref()
            .is_none_or(|codes| codes.iter().any(|code| &code.code == fixed));
        if !fits || !listed {
            problems.error(
                path,
                Message::with(
                    message!("The fixed value {0} is not a value the element allows"),
                    [Arg::from(fixed.as_str())],
                ),
            );
        }
    }
}

fn check_narrowing(guide: &Guide, base: &Guide, problems: &mut Problems) {
    if guide.transaction_set != base.transaction_set {
        problems.widening(
            "transactionSet",
            Message::with(
                message!("The guide is for {0}, but its base is for {1}"),
                [
                    Arg::from(guide.transaction_set.as_str()),
                    Arg::from(base.transaction_set.as_str()),
                ],
            ),
        );
        return;
    }
    narrow_segments(guide, base, problems);
    narrow_structure(guide, base, problems);
}

fn narrow_segments(guide: &Guide, base: &Guide, problems: &mut Problems) {
    for (id, segment) in &guide.segments {
        let path = format!("segments/{id}");
        let Some(base_segment) = base.segments.get(id) else {
            problems.widening(
                path.as_str(),
                Message::with(
                    message!("{0} is not a segment of the base guide"),
                    [Arg::from(id.as_str())],
                ),
            );
            continue;
        };
        if segment.elements.len() > base_segment.elements.len() {
            problems.widening(
                path.as_str(),
                Message::with(
                    message!("{0} has more elements than the base defines"),
                    [Arg::from(id.as_str())],
                ),
            );
        }
        for (index, (element, base_element)) in segment
            .elements
            .iter()
            .zip(&base_segment.elements)
            .enumerate()
        {
            narrow_element(
                &format!("{path}/{:02}", index + 1),
                element,
                base_element,
                problems,
            );
        }
        for rule in &base_segment.rules {
            if !segment.rules.contains(rule) {
                problems.widening(
                    path.as_str(),
                    Message::with(
                        message!("The base's syntax note {0} is missing"),
                        [Arg::from(rule.as_str())],
                    ),
                );
            }
        }
    }
}

fn narrow_structure(guide: &Guide, base: &Guide, problems: &mut Problems) {
    let mut base_nodes = BTreeMap::new();
    index_nodes(&base.structure, "structure", &mut base_nodes);
    let mut nodes = BTreeMap::new();
    index_nodes(&guide.structure, "structure", &mut nodes);
    for (path, node) in &nodes {
        let Some(base_node) = base_nodes.get(path) else {
            problems.widening(
                path.as_str(),
                Message::new(message!(
                    "This part of the structure is not in the base guide"
                )),
            );
            continue;
        };
        if !node.usage.is_narrowing_of(base_node.usage) {
            problems.widening(
                path.as_str(),
                Message::new(message!(
                    "The base requires this, so a partner cannot make it optional or not used"
                )),
            );
        }
        if node.max > base_node.max {
            problems.widening(
                path.as_str(),
                Message::with(
                    message!("The maximum {0} is above the base's {1}"),
                    [
                        Arg::from(i64::from(node.max)),
                        Arg::from(i64::from(base_node.max)),
                    ],
                ),
            );
        }
    }
    let mut siblings: BTreeMap<&str, Vec<(usize, usize, &str)>> = BTreeMap::new();
    for (path, node) in &nodes {
        if let Some(base_node) = base_nodes.get(path) {
            siblings.entry(node.parent.as_str()).or_default().push((
                node.order,
                base_node.order,
                path.as_str(),
            ));
        }
    }
    for group in siblings.values_mut() {
        group.sort_unstable();
        for pair in group.windows(2) {
            if pair[1].1 < pair[0].1 {
                problems.widening(
                    pair[1].2,
                    Message::new(message!(
                        "This is in a different order than in the base guide"
                    )),
                );
            }
        }
    }
    for (path, base_node) in &base_nodes {
        let parent_kept = base_node.parent == "structure" || nodes.contains_key(&base_node.parent);
        if base_node.usage == Usage::Mandatory && !nodes.contains_key(path) && parent_kept {
            problems.widening(
                path.as_str(),
                Message::new(message!(
                    "The base requires this, but the guide leaves it out"
                )),
            );
        }
    }
}

fn narrow_element(path: &str, element: &ElementDef, base: &ElementDef, problems: &mut Problems) {
    if element.kind != base.kind || element.reference != base.reference {
        problems.widening(
            path,
            Message::with(
                message!("The element is {0} {1} in the base guide"),
                [
                    Arg::from(base.reference.as_str()),
                    Arg::from(base.kind.code()),
                ],
            ),
        );
        return;
    }
    if !element.usage.is_narrowing_of(base.usage) {
        problems.widening(
            path,
            Message::new(message!(
                "The base requires this element, so a partner cannot make it optional or not used"
            )),
        );
    }
    if element.min < base.min || element.max > base.max {
        problems.widening(
            path,
            Message::with(
                message!("The lengths {0} to {1} are wider than the base's {2} to {3}"),
                [
                    Arg::from(i64::from(element.min)),
                    Arg::from(i64::from(element.max)),
                    Arg::from(i64::from(base.min)),
                    Arg::from(i64::from(base.max)),
                ],
            ),
        );
    }
    if element.repeat > base.repeat {
        problems.widening(
            path,
            Message::with(
                message!("The element repeats up to {0} times, more than the base's {1}"),
                [
                    Arg::from(i64::from(element.repeat)),
                    Arg::from(i64::from(base.repeat)),
                ],
            ),
        );
    }
    if let Some(base_codes) = &base.codes {
        match &element.codes {
            None => problems.widening(
                path,
                Message::new(message!(
                    "The base restricts this element's codes, but the guide does not"
                )),
            ),
            Some(codes) => {
                for code in codes {
                    if !base_codes
                        .iter()
                        .any(|base_code| base_code.code == code.code)
                    {
                        problems.widening(
                            path,
                            Message::with(
                                message!("The code {0} is not in the base's list"),
                                [Arg::from(code.code.as_str())],
                            ),
                        );
                    }
                }
            }
        }
    }
    if let (Some(components), Some(base_components)) = (&element.components, &base.components) {
        for (index, (component, base_component)) in
            components.iter().zip(base_components).enumerate()
        {
            narrow_element(
                &format!("{path}-{:02}", index + 1),
                component,
                base_component,
                problems,
            );
        }
    }
}

struct IndexedNode {
    parent: String,
    order: usize,
    usage: Usage,
    max: u32,
}

fn index_nodes(nodes: &[Node], parent: &str, index: &mut BTreeMap<String, IndexedNode>) {
    let mut seen: BTreeMap<String, usize> = BTreeMap::new();
    for (order, node) in nodes.iter().enumerate() {
        let (name, usage, max) = match node {
            Node::Segment(segment) => (segment.segment.as_str(), segment.usage, segment.max),
            Node::Loop(definition) => (definition.id.as_str(), definition.usage, definition.repeat),
        };
        let occurrence = seen.entry(name.to_owned()).or_insert(0);
        *occurrence += 1;
        let path = if *occurrence == 1 {
            format!("{parent}/{name}")
        } else {
            format!("{parent}/{name}#{occurrence}")
        };
        if let Node::Loop(definition) = node {
            index_nodes(&definition.children, &path, index);
        }
        index.insert(
            path,
            IndexedNode {
                parent: parent.to_owned(),
                order,
                usage,
                max,
            },
        );
    }
}

#[cfg(test)]
mod tests {
    use super::{ProblemKind, check_guide};
    use crate::x12::guide::builtin;
    use crate::x12::guide::model::{Code, Node, Usage};

    fn base(id: &str) -> crate::x12::guide::model::Guide {
        builtin::by_id(id).expect("a base guide").guide().clone()
    }

    #[test]
    fn reports_a_broken_structure() {
        let mut guide = base("x12.004010.990");
        guide.structure.remove(0);
        guide.transaction_set = String::from("99");
        let problems = check_guide(&guide, None);
        let paths: Vec<_> = problems
            .iter()
            .map(|problem| problem.path.as_str())
            .collect();
        assert!(paths.contains(&"structure"), "{paths:?}");
        assert!(paths.contains(&"transactionSet"), "{paths:?}");
        assert!(
            problems
                .iter()
                .all(|problem| problem.kind == ProblemKind::Error)
        );
    }

    #[test]
    fn reports_a_bad_syntax_note_and_bad_lengths() {
        let mut guide = base("x12.004010.990");
        let b1 = guide.segments.get_mut("B1").expect("B1");
        b1.rules.push(String::from("P0309"));
        b1.elements[0].min = 9;
        let messages: Vec<_> = check_guide(&guide, None)
            .into_iter()
            .map(|problem| problem.message.english())
            .collect();
        assert!(
            messages.iter().any(|message| message.contains("P0309")),
            "{messages:?}"
        );
        assert!(
            messages
                .iter()
                .any(|message| message.contains("minimum length 9")),
            "{messages:?}"
        );
    }

    #[test]
    fn a_partner_guide_may_narrow_but_not_widen() {
        let base_guide = base("x12.004010.204");
        let mut partner = base_guide.clone();
        partner.derived_from = Some(base_guide.id.clone());
        let n1 = partner.segments.get_mut("N1").expect("N1");
        n1.elements[0].codes = Some(vec![Code {
            code: String::from("SH"),
            description: String::new(),
        }]);
        n1.elements[1].usage = Usage::Mandatory;
        if let Some(Node::Segment(l11)) = partner.structure.get_mut(3) {
            l11.max = 5;
        }
        assert!(check_guide(&partner, Some(&base_guide)).is_empty());

        if let Some(Node::Segment(b2)) = partner.structure.get_mut(1) {
            b2.usage = Usage::Optional;
        }
        if let Some(Node::Segment(l11)) = partner.structure.get_mut(3) {
            l11.max = 99;
        }
        partner.segments.get_mut("N4").expect("N4").elements[0].max = 99;
        let problems = check_guide(&partner, Some(&base_guide));
        assert!(
            problems
                .iter()
                .all(|problem| problem.kind == ProblemKind::Widening)
        );
        let paths: Vec<_> = problems
            .iter()
            .map(|problem| problem.path.as_str())
            .collect();
        assert!(paths.contains(&"structure/B2"), "{paths:?}");
        assert!(paths.contains(&"structure/L11"), "{paths:?}");
        assert!(paths.contains(&"segments/N4/01"), "{paths:?}");
    }

    #[test]
    fn leaving_out_a_mandatory_segment_widens() {
        let base_guide = base("x12.004010.204");
        let mut partner = base_guide.clone();
        partner
            .structure
            .retain(|node| !matches!(node, Node::Segment(segment) if segment.segment == "B2A"));
        let problems = check_guide(&partner, Some(&base_guide));
        assert!(
            problems
                .iter()
                .any(|problem| problem.path == "structure/B2A"),
            "{problems:?}"
        );
    }
}
