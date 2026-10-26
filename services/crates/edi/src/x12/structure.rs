//! Places each segment of a transaction set in its guide's loop tree.
//!
//! X12 has no brackets: a loop is recognised by its first segment, and where a segment belongs
//! follows only from the order the guide gives. The walker keeps a stack of open loops, each
//! with a cursor (the child it last placed a segment at) and a count per child. For each segment
//! it looks forward from the cursor of the innermost loop for a segment use with that ID or a
//! child loop that segment starts; failing that, the enclosing loop's, which is where a loop's
//! next repetition is found; and so on up to the transaction set. Passing a child that was never
//! seen reports it missing when it is mandatory, and closing a loop reports what it still lacked.
//!
//! A segment found nowhere is unrecognised when its ID is not one X12 has, out of sequence when
//! the set uses it elsewhere, and not in the set otherwise. It is left where the walk was, so
//! one stray segment does not misplace everything after it.
//!
//! The walk is linear in the number of segments: each segment is compared with at most the
//! children of the loops open around it, and loops nest at most 16 deep.

use trenova_module_kit::message;
use trenova_module_kit::message::{Arg, Message};

use super::finding::FindingKind;
use super::guide::builtin;
use super::guide::compiled::{CompiledGuide, NodeKind};
use super::guide::model::Usage;

/// Where one segment was placed.
#[derive(Debug, Clone, PartialEq, Eq, Default)]
pub struct Placement {
    /// The structure node it was placed at, or `None` when it fits nowhere.
    pub node: Option<usize>,
    /// The loops around it, outermost first: `0300/0310`.
    pub loop_path: String,
    /// The innermost loop around it, empty at the top of the set.
    pub loop_id: String,
}

/// One thing the walk found wrong.
#[derive(Debug, Clone, PartialEq, Eq)]
pub struct StructureIssue {
    /// What is wrong.
    pub kind: FindingKind,
    /// The segment of the transaction set the issue was found at (its index in the list given
    /// to [`walk`]); for a missing segment, the one being read when the absence became certain.
    pub at: usize,
    /// The segment the issue is about: the segment itself, or the one that is missing.
    pub segment_id: String,
    /// The loop the segment is, or would be, in.
    pub loop_id: String,
    /// What is wrong, for people.
    pub message: Message,
}

/// The result of a walk.
#[derive(Debug, Clone, Default)]
pub struct Walk {
    /// One placement per segment, in order.
    pub placements: Vec<Placement>,
    /// What was wrong, in the order found.
    pub issues: Vec<StructureIssue>,
}

struct Frame {
    node: Option<usize>,
    children: Vec<usize>,
    cursor: Option<usize>,
    counts: Vec<u32>,
}

impl Frame {
    fn new(node: Option<usize>, children: &[usize]) -> Self {
        Self {
            node,
            children: children.to_vec(),
            cursor: None,
            counts: vec![0; children.len()],
        }
    }
}

struct Walker<'g> {
    guide: &'g CompiledGuide,
    stack: Vec<Frame>,
    issues: Vec<StructureIssue>,
}

/// Walks the segment IDs of one transaction set, ST first, through `guide`.
#[must_use]
pub fn walk(guide: &CompiledGuide, ids: &[&str]) -> Walk {
    let mut walker = Walker {
        guide,
        stack: vec![Frame::new(None, guide.root())],
        issues: Vec::new(),
    };
    let mut placements = Vec::with_capacity(ids.len());
    for (at, id) in ids.iter().enumerate() {
        placements.push(walker.place(at, id));
    }
    let last = ids.len().saturating_sub(1);
    while let Some(frame) = walker.stack.pop() {
        walker.close(&frame, last);
    }
    Walk {
        placements,
        issues: walker.issues,
    }
}

impl Walker<'_> {
    fn loop_id(&self, frame: &Frame) -> String {
        match frame.node.map(|node| &self.guide.node(node).kind) {
            Some(NodeKind::Loop { id, .. }) => id.clone(),
            _ => String::new(),
        }
    }

    fn placement(&self, node: Option<usize>) -> Placement {
        let loops: Vec<String> = self
            .stack
            .iter()
            .map(|frame| self.loop_id(frame))
            .filter(|id| !id.is_empty())
            .collect();
        Placement {
            node,
            loop_id: loops.last().cloned().unwrap_or_default(),
            loop_path: loops.join("/"),
        }
    }

    /// The child of the loop at `depth` that `id` belongs at, looking forward from its cursor.
    /// A loop's first segment is never matched inside the loop: meeting it again starts the
    /// next repetition, which the enclosing loop finds.
    fn find(&self, depth: usize, id: &str) -> Option<usize> {
        let frame = &self.stack[depth];
        let floor = usize::from(frame.node.is_some());
        let start = frame.cursor.unwrap_or(0).max(floor);
        (start..frame.children.len()).find(|position| {
            let node = self.guide.node(frame.children[*position]);
            node.segment_id() == id
        })
    }

    fn place(&mut self, at: usize, id: &str) -> Placement {
        for depth in (0..self.stack.len()).rev() {
            let Some(position) = self.find(depth, id) else {
                continue;
            };
            while self.stack.len() > depth + 1 {
                if let Some(frame) = self.stack.pop() {
                    self.close(&frame, at);
                }
            }
            let (node_index, count, passed) = {
                let frame = &mut self.stack[depth];
                let passed: Vec<usize> = match frame.cursor {
                    Some(cursor) if cursor == position => Vec::new(),
                    Some(cursor) => (cursor..position).collect(),
                    None => (0..position).collect(),
                };
                frame.cursor = Some(position);
                frame.counts[position] += 1;
                (frame.children[position], frame.counts[position], passed)
            };
            for child in passed {
                self.check_passed(depth, child, at);
            }
            let node = self.guide.node(node_index).clone();
            let loop_id = self.loop_id(&self.stack[depth]);
            if node.usage == Usage::NotUsed {
                self.issue(
                    FindingKind::NotUsedSegmentPresent,
                    at,
                    id,
                    &loop_id,
                    Message::with(
                        message!("{0} is not used by this partner, but is present"),
                        [Arg::from(id)],
                    ),
                );
            }
            if node.is_loop() {
                if count > node.max {
                    let NodeKind::Loop { id: inner, .. } = &node.kind else {
                        unreachable!("a loop node is a loop");
                    };
                    self.issue(
                        FindingKind::LoopOverMaximum,
                        at,
                        id,
                        inner,
                        Message::with(
                            message!("Loop {0} repeats {1} times, more than its maximum of {2}"),
                            [
                                Arg::from(inner.as_str()),
                                Arg::from(i64::from(count)),
                                Arg::from(i64::from(node.max)),
                            ],
                        ),
                    );
                }
                let mut frame = Frame::new(Some(node_index), &node.children);
                frame.cursor = Some(0);
                frame.counts[0] = 1;
                self.stack.push(frame);
                return self.placement(node.children.first().copied());
            }
            if count > node.max {
                self.issue(
                    FindingKind::SegmentOverMaximumUse,
                    at,
                    id,
                    &loop_id,
                    Message::with(
                        message!("{0} appears {1} times here, more than its maximum of {2}"),
                        [
                            Arg::from(id),
                            Arg::from(i64::from(count)),
                            Arg::from(i64::from(node.max)),
                        ],
                    ),
                );
            }
            return self.placement(Some(node_index));
        }

        self.unplaced(at, id)
    }

    /// Reports a segment that fits nowhere and leaves the walk where it was.
    fn unplaced(&mut self, at: usize, id: &str) -> Placement {
        let loop_id = self
            .stack
            .last()
            .map(|frame| self.loop_id(frame))
            .unwrap_or_default();
        let (kind, message) = if !is_segment_id(id) {
            (
                FindingKind::UnrecognizedSegment,
                Message::with(message!("{0} is not a segment ID"), [Arg::from(id)]),
            )
        } else if self.guide.uses_segment(id) {
            (
                FindingKind::SegmentOutOfSequence,
                Message::with(
                    message!("{0} belongs to this transaction set, but not here"),
                    [Arg::from(id)],
                ),
            )
        } else if builtin::is_known_segment(id) {
            (
                FindingKind::SegmentNotInSet,
                Message::with(
                    message!("{0} is not part of this transaction set"),
                    [Arg::from(id)],
                ),
            )
        } else {
            (
                FindingKind::UnrecognizedSegment,
                Message::with(
                    message!("{0} is not a segment this guide knows"),
                    [Arg::from(id)],
                ),
            )
        };
        self.issue(kind, at, id, &loop_id, message);
        self.placement(None)
    }

    /// Checks a child the walk moved past: missing when mandatory and never seen, short when a
    /// minimum was not met.
    fn check_passed(&mut self, depth: usize, child: usize, at: usize) {
        let frame = &self.stack[depth];
        let count = frame.counts[child];
        let node = self.guide.node(frame.children[child]).clone();
        let loop_id = self.loop_id(frame);
        self.check_count(&node, count, at, &loop_id);
    }

    fn close(&mut self, frame: &Frame, at: usize) {
        let loop_id = self.loop_id(frame);
        let start = frame.cursor.unwrap_or(0);
        for child in start..frame.children.len() {
            let node = self.guide.node(frame.children[child]).clone();
            self.check_count(&node, frame.counts[child], at, &loop_id);
        }
    }

    fn check_count(
        &mut self,
        node: &super::guide::compiled::CompiledNode,
        count: u32,
        at: usize,
        loop_id: &str,
    ) {
        let id = node.segment_id().to_owned();
        let own_loop = match &node.kind {
            NodeKind::Loop { id, .. } => id.clone(),
            NodeKind::Segment(_) => loop_id.to_owned(),
        };
        if count == 0 && node.usage == Usage::Mandatory {
            let message = if node.is_loop() {
                Message::with(
                    message!("Loop {0} is mandatory, but its first segment {1} is missing"),
                    [Arg::from(own_loop.as_str()), Arg::from(id.as_str())],
                )
            } else {
                Message::with(
                    message!("{0} is mandatory here, but is missing"),
                    [Arg::from(id.as_str())],
                )
            };
            self.issue(
                FindingKind::MandatorySegmentMissing,
                at,
                &id,
                &own_loop,
                message,
            );
        } else if count > 0 && count < node.min {
            let (kind, message) = if node.is_loop() {
                (
                    FindingKind::LoopUnderMinimum,
                    Message::with(
                        message!(
                            "Loop {0} repeats {1} times, fewer than the partner's minimum of {2}"
                        ),
                        [
                            Arg::from(own_loop.as_str()),
                            Arg::from(i64::from(count)),
                            Arg::from(i64::from(node.min)),
                        ],
                    ),
                )
            } else {
                (
                    FindingKind::SegmentUnderMinimumUse,
                    Message::with(
                        message!("{0} appears {1} times, fewer than the partner's minimum of {2}"),
                        [
                            Arg::from(id.as_str()),
                            Arg::from(i64::from(count)),
                            Arg::from(i64::from(node.min)),
                        ],
                    ),
                )
            };
            self.issue(kind, at, &id, &own_loop, message);
        }
    }

    fn issue(&mut self, kind: FindingKind, at: usize, id: &str, loop_id: &str, message: Message) {
        self.issues.push(StructureIssue {
            kind,
            at,
            segment_id: id.to_owned(),
            loop_id: loop_id.to_owned(),
            message,
        });
    }
}

fn is_segment_id(id: &str) -> bool {
    let bytes = id.as_bytes();
    (2..=3).contains(&bytes.len())
        && bytes[0].is_ascii_uppercase()
        && bytes
            .iter()
            .all(|byte| byte.is_ascii_uppercase() || byte.is_ascii_digit())
}

#[cfg(test)]
mod tests {
    use super::walk;
    use crate::x12::finding::FindingKind;
    use crate::x12::guide::builtin;

    fn run(set: &str, ids: &str) -> (Vec<String>, Vec<(FindingKind, usize, String)>) {
        let guide = builtin::by_id(set).expect("a guide");
        let ids: Vec<&str> = ids.split_whitespace().collect();
        let result = walk(guide, &ids);
        let paths = result
            .placements
            .iter()
            .map(|placement| placement.loop_path.clone())
            .collect();
        let issues = result
            .issues
            .into_iter()
            .map(|issue| (issue.kind, issue.at, issue.segment_id))
            .collect();
        (paths, issues)
    }

    #[test]
    fn places_a_tender_in_its_loops() {
        let (paths, issues) = run(
            "x12.004010.204",
            "ST B2 B2A L11 S5 G62 N1 N3 N4 S5 G62 N1 N3 N4 L5 L3 SE",
        );
        assert!(issues.is_empty(), "{issues:?}");
        assert_eq!(
            paths,
            [
                "",
                "",
                "",
                "",
                "0300",
                "0300",
                "0300/0310",
                "0300/0310",
                "0300/0310",
                "0300",
                "0300",
                "0300/0310",
                "0300/0310",
                "0300/0310",
                "0300/0320",
                "",
                ""
            ]
        );
    }

    #[test]
    fn the_flat_renderers_tender_puts_stop_parties_in_the_heading() {
        let (paths, issues) = run(
            "x12.004010.204",
            "ST B2 B2A L11 G62 G62 N1 N1 N4 N4 G61 S5 S5 SE",
        );
        assert_eq!(
            issues,
            [
                (FindingKind::SegmentOverMaximumUse, 5, String::from("G62")),
                (FindingKind::SegmentOverMaximumUse, 9, String::from("N4")),
            ]
        );
        assert_eq!(paths[6], "0100", "the stop's N1 reads as a heading party");
        assert_eq!(paths[11], "0300");
    }

    #[test]
    fn reports_missing_mandatory_segments_and_loops() {
        let (_, issues) = run("x12.004010.204", "ST B2 L11 SE");
        assert_eq!(
            issues,
            [
                (FindingKind::MandatorySegmentMissing, 2, String::from("B2A")),
                (FindingKind::MandatorySegmentMissing, 3, String::from("S5")),
            ]
        );
        let (_, issues) = run("x12.004010.990", "ST B1");
        assert_eq!(
            issues,
            [(FindingKind::MandatorySegmentMissing, 1, String::from("SE"))]
        );
    }

    #[test]
    fn counts_uses_and_loop_repetitions() {
        let (_, issues) = run("x12.004010.990", "ST B1 N9 N9 SE");
        assert_eq!(
            issues,
            [(FindingKind::SegmentOverMaximumUse, 3, String::from("N9"))]
        );
        let ids = format!("ST B2 B2A {} S5 SE", "N1 ".repeat(6));
        let (_, issues) = run("x12.004010.204", &ids);
        assert_eq!(
            issues,
            [(FindingKind::LoopOverMaximum, 8, String::from("N1"))]
        );
    }

    #[test]
    fn classifies_segments_that_fit_nowhere() {
        let (paths, issues) = run("x12.004010.990", "ST B1 AT7 ZZZ b1 SE");
        assert_eq!(
            issues,
            [
                (FindingKind::SegmentNotInSet, 2, String::from("AT7")),
                (FindingKind::UnrecognizedSegment, 3, String::from("ZZZ")),
                (FindingKind::UnrecognizedSegment, 4, String::from("b1")),
            ]
        );
        assert_eq!(paths.len(), 6);
        let (_, issues) = run("x12.004010.204", "ST B2 B2A S5 B2 SE");
        assert_eq!(
            issues,
            [(FindingKind::SegmentOutOfSequence, 4, String::from("B2"))]
        );
    }

    #[test]
    fn reads_every_status_loop_of_a_214() {
        let (paths, issues) = run(
            "x12.004010.214",
            "ST B10 L11 LX AT7 MS1 MS2 AT7 L11 LX AT7 MS1 L11 LX AT7 N1 N4 SE",
        );
        assert!(issues.is_empty(), "{issues:?}");
        assert_eq!(paths[4], "0205/0210");
        assert_eq!(paths[7], "0205/0210");
        assert_eq!(paths[8], "0205");
        assert_eq!(paths[15], "0205/0250");
        let (_, issues) = run("x12.004010.214", "ST B10 LX AT7 L11 AT7 SE");
        assert_eq!(
            issues,
            [(FindingKind::SegmentOutOfSequence, 5, String::from("AT7"))],
            "a status after the loop's references belongs to a new LX"
        );
    }

    #[test]
    fn nests_acknowledgment_errors_under_their_transaction() {
        let (paths, issues) = run(
            "x12.004010.997",
            "ST AK1 AK2 AK3 AK4 AK4 AK3 AK5 AK2 AK5 AK9 SE",
        );
        assert!(issues.is_empty(), "{issues:?}");
        assert_eq!(paths[4], "AK2/AK3");
        assert_eq!(paths[7], "AK2");
        assert_eq!(paths[10], "");
    }
}
