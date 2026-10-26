//! The compact text form the built-in base guides are written in.
//!
//! JSON is the wire shape of a guide, but a base dictionary is hundreds of element definitions,
//! and one per line is what keeps it reviewable against a published guide. Two kinds of file:
//!
//! A **dictionary** defines segments and code lists:
//!
//! ```text
//! segment N1 | Name
//!   98 ID 2/3 M | Entity Identifier Code
//!   93 AN 1/60 X | Name
//!   rules R0203 P0304
//! segment AK4 | Data Element Note
//!   C030 composite M | Position in Segment
//!     - 722 N0 1/2 M | Element Position in Segment
//! codes 717 | Transaction Set Acknowledgment Code
//!   A | Accepted
//! ```
//!
//! An element line is `<ref> <type> <min>/<max> <usage> [r<repeat>] | <name>`, a component line
//! the same after `- `. A code list attaches to every element with its reference number.
//!
//! A **guide** names its set and lays out the structure:
//!
//! ```text
//! guide x12.004010.990 | Response to a Load Tender
//! set 990 GF 004010
//! ST M 1
//! B1 M 1
//! loop 0100 O 10 | Name
//!   N1 M 1
//! end
//! SE M 1
//! ```
//!
//! A segment line is `<id> <usage> <max> [min<n>]`; a loop line is
//! `loop <id> <usage> <repeat> [min<n>] | <name>`, closed by `end`. Blank lines and `#` comments
//! are ignored.

use std::collections::BTreeMap;

use super::model::{
    Code, ElementDef, ElementType, Guide, LoopDef, Node, SegmentDef, SegmentUse, Usage,
};

/// A parsed dictionary: segments by ID and code lists by element reference.
#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub struct Dictionary {
    /// Segment definitions, code lists already attached.
    pub segments: BTreeMap<String, SegmentDef>,
}

impl Dictionary {
    /// Adds every segment of `other` this one does not define.
    pub fn extend(&mut self, other: &Self) {
        for (id, segment) in &other.segments {
            self.segments
                .entry(id.clone())
                .or_insert_with(|| segment.clone());
        }
    }
}

/// Parses a dictionary file. The error names the line.
pub fn parse_dictionary(text: &str) -> Result<Dictionary, String> {
    let mut segments: BTreeMap<String, SegmentDef> = BTreeMap::new();
    let mut codes: BTreeMap<String, Vec<Code>> = BTreeMap::new();
    let mut current_segment: Option<String> = None;
    let mut current_codes: Option<String> = None;

    for (number, line) in lines(text) {
        let fail = |reason: &str| format!("line {number}: {reason}: {line:?}");
        let (head, name) = split_name(line);
        let words: Vec<&str> = head.split_whitespace().collect();
        match words.first().copied() {
            Some("segment") => {
                let [_, id] = words[..] else {
                    return Err(fail("a segment line is `segment <id> | <name>`"));
                };
                if segments.contains_key(id) {
                    return Err(fail("the segment is defined twice"));
                }
                segments.insert(
                    id.to_owned(),
                    SegmentDef {
                        name: name.to_owned(),
                        elements: Vec::new(),
                        rules: Vec::new(),
                        notes: String::new(),
                    },
                );
                current_segment = Some(id.to_owned());
                current_codes = None;
            }
            Some("codes") => {
                let [_, reference] = words[..] else {
                    return Err(fail("a code list line is `codes <ref> | <name>`"));
                };
                codes.entry(reference.to_owned()).or_default();
                current_codes = Some(reference.to_owned());
                current_segment = None;
            }
            Some("rules") => {
                let segment = current_segment
                    .as_ref()
                    .and_then(|id| segments.get_mut(id))
                    .ok_or_else(|| fail("rules outside a segment"))?;
                segment
                    .rules
                    .extend(words[1..].iter().map(|rule| (*rule).to_owned()));
            }
            Some("-") => {
                let segment = current_segment
                    .as_ref()
                    .and_then(|id| segments.get_mut(id))
                    .ok_or_else(|| fail("a component outside a segment"))?;
                let component = element(&words[1..], name).map_err(|reason| fail(&reason))?;
                let composite = segment
                    .elements
                    .last_mut()
                    .filter(|element| element.kind == ElementType::Composite)
                    .ok_or_else(|| fail("a component must follow a composite"))?;
                composite
                    .components
                    .get_or_insert_with(Vec::new)
                    .push(component);
            }
            Some(_) if current_codes.is_some() => {
                let [code] = words[..] else {
                    return Err(fail("a code line is `<code> | <description>`"));
                };
                if let Some(list) = current_codes
                    .as_ref()
                    .and_then(|reference| codes.get_mut(reference))
                {
                    list.push(Code {
                        code: code.to_owned(),
                        description: name.to_owned(),
                    });
                }
            }
            Some(_) => {
                let segment = current_segment
                    .as_ref()
                    .and_then(|id| segments.get_mut(id))
                    .ok_or_else(|| fail("an element outside a segment"))?;
                let element = element(&words, name).map_err(|reason| fail(&reason))?;
                segment.elements.push(element);
            }
            None => {}
        }
    }

    for segment in segments.values_mut() {
        for element in &mut segment.elements {
            attach_codes(element, &codes);
        }
    }
    Ok(Dictionary { segments })
}

fn attach_codes(element: &mut ElementDef, codes: &BTreeMap<String, Vec<Code>>) {
    if let Some(list) = codes.get(&element.reference) {
        element.codes = Some(list.clone());
    }
    if let Some(components) = element.components.as_mut() {
        for component in components {
            attach_codes(component, codes);
        }
    }
}

fn element(words: &[&str], name: &str) -> Result<ElementDef, String> {
    let (reference, kind, lengths, usage, rest) = match words {
        [reference, kind, lengths, usage, rest @ ..] => (*reference, *kind, *lengths, *usage, rest),
        [reference, kind, usage] if *kind == "composite" => {
            (*reference, *kind, "0/0", *usage, &[][..])
        }
        _ => {
            return Err(String::from(
                "an element line is `<ref> <type> <min>/<max> <usage> | <name>`",
            ));
        }
    };
    let kind = ElementType::from_code(kind).ok_or("an unknown data type")?;
    let (min, max) = lengths
        .split_once('/')
        .and_then(|(min, max)| Some((min.parse().ok()?, max.parse().ok()?)))
        .ok_or("lengths are `<min>/<max>`")?;
    let usage = Usage::from_letter(usage).ok_or("usage is M, O, X or N")?;
    let mut repeat = 1;
    for modifier in rest {
        match modifier
            .strip_prefix('r')
            .and_then(|count| count.parse().ok())
        {
            Some(count) => repeat = count,
            None => return Err(format!("an unknown modifier {modifier:?}")),
        }
    }
    Ok(ElementDef {
        reference: reference.to_owned(),
        name: name.to_owned(),
        kind,
        min,
        max,
        usage,
        repeat,
        codes: None,
        fixed: None,
        components: None,
        notes: String::new(),
    })
}

/// A parsed guide file: everything but the segment dictionary, which the caller fills in.
#[derive(Debug, Clone, PartialEq, Eq)]
pub struct Layout {
    /// The guide's identifier.
    pub id: String,
    /// Its name.
    pub name: String,
    /// ST01.
    pub transaction_set: String,
    /// GS01.
    pub functional_group: String,
    /// GS08.
    pub version: String,
    /// The structure.
    pub structure: Vec<Node>,
}

impl Layout {
    /// The guide, its dictionary taken from `dictionary` for every segment the structure uses.
    /// The error names the first segment the dictionary does not define.
    pub fn into_guide(self, dictionary: &Dictionary) -> Result<Guide, String> {
        let mut ids = Vec::new();
        collect_segments(&self.structure, &mut ids);
        let mut segments = BTreeMap::new();
        for id in ids {
            let definition = dictionary
                .segments
                .get(&id)
                .ok_or_else(|| format!("{}: the dictionary does not define {id}", self.id))?;
            segments.insert(id, definition.clone());
        }
        Ok(Guide {
            id: self.id,
            name: self.name,
            description: String::new(),
            transaction_set: self.transaction_set,
            functional_group: self.functional_group,
            version: self.version,
            derived_from: None,
            segments,
            structure: self.structure,
            severities: BTreeMap::new(),
        })
    }
}

fn collect_segments(nodes: &[Node], ids: &mut Vec<String>) {
    for node in nodes {
        match node {
            Node::Segment(segment) => {
                if !ids.contains(&segment.segment) {
                    ids.push(segment.segment.clone());
                }
            }
            Node::Loop(definition) => collect_segments(&definition.children, ids),
        }
    }
}

/// Parses a guide file. The error names the line.
pub fn parse_layout(text: &str) -> Result<Layout, String> {
    let mut id = None;
    let mut name = String::new();
    let mut set = None;
    let mut stack: Vec<(LoopDef, usize)> = Vec::new();
    let mut root: Vec<Node> = Vec::new();

    for (number, line) in lines(text) {
        let fail = |reason: &str| format!("line {number}: {reason}: {line:?}");
        let (head, line_name) = split_name(line);
        let words: Vec<&str> = head.split_whitespace().collect();
        match words.first().copied() {
            Some("guide") => {
                let [_, guide_id] = words[..] else {
                    return Err(fail("a guide line is `guide <id> | <name>`"));
                };
                id = Some(guide_id.to_owned());
                line_name.clone_into(&mut name);
            }
            Some("set") => {
                let [_, transaction_set, functional_group, version] = words[..] else {
                    return Err(fail("a set line is `set <ST01> <GS01> <GS08>`"));
                };
                set = Some((
                    transaction_set.to_owned(),
                    functional_group.to_owned(),
                    version.to_owned(),
                ));
            }
            Some("loop") => {
                let (usage, repeat, min) = counts(&words[2..]).map_err(|reason| fail(&reason))?;
                let loop_id = words.get(1).ok_or_else(|| fail("a loop needs an id"))?;
                stack.push((
                    LoopDef {
                        id: (*loop_id).to_owned(),
                        name: line_name.to_owned(),
                        usage,
                        repeat,
                        min,
                        children: Vec::new(),
                        notes: String::new(),
                    },
                    number,
                ));
            }
            Some("end") => {
                let (finished, _) = stack.pop().ok_or_else(|| fail("`end` without a loop"))?;
                push_node(&mut stack, &mut root, Node::Loop(finished));
            }
            Some(segment) => {
                let (usage, max, min) = counts(&words[1..]).map_err(|reason| fail(&reason))?;
                push_node(
                    &mut stack,
                    &mut root,
                    Node::Segment(SegmentUse {
                        segment: segment.to_owned(),
                        usage,
                        max,
                        min,
                        notes: String::new(),
                        definition: None,
                    }),
                );
            }
            None => {}
        }
    }
    if let Some((open, number)) = stack.last() {
        return Err(format!("line {number}: loop {} is never closed", open.id));
    }
    let id = id.ok_or("a guide file starts with `guide <id> | <name>`")?;
    let (transaction_set, functional_group, version) =
        set.ok_or("a guide file has a `set <ST01> <GS01> <GS08>` line")?;
    Ok(Layout {
        id,
        name,
        transaction_set,
        functional_group,
        version,
        structure: root,
    })
}

fn push_node(stack: &mut [(LoopDef, usize)], root: &mut Vec<Node>, node: Node) {
    match stack.last_mut() {
        Some((open, _)) => open.children.push(node),
        None => root.push(node),
    }
}

fn counts(words: &[&str]) -> Result<(Usage, u32, u32), String> {
    let (usage, max, rest) = match words {
        [usage, max, rest @ ..] => (*usage, *max, rest),
        _ => return Err(String::from("expected `<usage> <max>`")),
    };
    let usage = Usage::from_letter(usage).ok_or("usage is M, O, X or N")?;
    let max = max
        .parse()
        .map_err(|_| String::from("the maximum is a number"))?;
    let mut min = 0;
    for modifier in rest {
        match modifier
            .strip_prefix("min")
            .and_then(|count| count.parse().ok())
        {
            Some(count) => min = count,
            None => return Err(format!("an unknown modifier {modifier:?}")),
        }
    }
    Ok((usage, max, min))
}

fn lines(text: &str) -> impl Iterator<Item = (usize, &str)> {
    text.lines().enumerate().filter_map(|(index, line)| {
        let line = line
            .split_once('#')
            .map_or(line, |(before, _)| before)
            .trim();
        (!line.is_empty()).then_some((index + 1, line))
    })
}

fn split_name(line: &str) -> (&str, &str) {
    match line.split_once('|') {
        Some((head, name)) => (head.trim(), name.trim()),
        None => (line, ""),
    }
}

#[cfg(test)]
mod tests {
    use super::{parse_dictionary, parse_layout};
    use crate::x12::guide::model::{ElementType, Node, Usage};

    const DICTIONARY: &str = "
        # a comment
        segment N1 | Name
          98 ID 2/3 M | Entity Identifier Code
          93 AN 1/60 X | Name
          rules R0203 P0304
        segment AK4 | Data Element Note
          C030 composite M | Position in Segment
            - 722 N0 1/2 M | Element Position in Segment
            - 1528 N0 1/2 O | Component Data Element Position in Composite
          725 N0 1/4 O r3 | Data Element Reference Number
        codes 98 | Entity Identifier Code
          SH | Shipper
          CN | Consignee
    ";

    #[test]
    fn reads_segments_elements_components_rules_and_codes() {
        let dictionary = parse_dictionary(DICTIONARY).expect("a dictionary");
        let n1 = &dictionary.segments["N1"];
        assert_eq!(n1.name, "Name");
        assert_eq!(n1.rules, ["R0203", "P0304"]);
        assert_eq!(n1.elements[0].reference, "98");
        assert_eq!(n1.elements[0].kind, ElementType::Identifier);
        assert_eq!((n1.elements[0].min, n1.elements[0].max), (2, 3));
        assert_eq!(n1.elements[1].usage, Usage::Conditional);
        let codes = n1.elements[0].codes.as_ref().expect("codes");
        assert_eq!(codes[1].code, "CN");
        let ak4 = &dictionary.segments["AK4"];
        assert_eq!(ak4.elements[0].components.as_ref().map(Vec::len), Some(2));
        assert_eq!(ak4.elements[1].repeat, 3);
    }

    #[test]
    fn names_the_line_of_a_mistake() {
        let error =
            parse_dictionary("segment N1 | Name\n  98 ZZ 2/3 M | Code").expect_err("an error");
        assert!(error.starts_with("line 2: an unknown data type"), "{error}");
        let error = parse_dictionary("  98 ID 2/3 M | Code").expect_err("an error");
        assert!(error.contains("outside a segment"), "{error}");
    }

    #[test]
    fn reads_a_layout_with_nested_loops() {
        let layout = parse_layout(
            "guide x | Test\nset 204 SM 004010\nST M 1\nloop 0300 M 999 | Stop\n  S5 M 1\n  loop 0310 O 1\n    N1 O 1\n  end\nend\nSE M 1",
        )
        .expect("a layout");
        assert_eq!(layout.transaction_set, "204");
        assert_eq!(layout.structure.len(), 3);
        let Node::Loop(stop) = &layout.structure[1] else {
            panic!("a loop");
        };
        assert_eq!((stop.id.as_str(), stop.repeat), ("0300", 999));
        assert!(matches!(&stop.children[1], Node::Loop(inner) if inner.id == "0310"));
        let error = parse_layout("guide x\nset 204 SM 004010\nloop 0300 M 1\nS5 M 1")
            .expect_err("unclosed");
        assert!(error.contains("never closed"), "{error}");
    }
}
