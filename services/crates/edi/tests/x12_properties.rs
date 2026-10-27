//! Properties of reading and writing that hold for any input.

use proptest::prelude::*;
use trenova_edi::x12::read::{ReadRequest, read};
use trenova_edi::x12::write::{Element, WriteRequest, write};

fn request(text: String, fragment: bool) -> ReadRequest {
    ReadRequest {
        text,
        separators: None,
        guides: Vec::new(),
        include_segments: true,
        fragment,
    }
}

fn segment_id() -> impl Strategy<Value = String> {
    prop::sample::select(vec![
        "N1", "N3", "N4", "L11", "G62", "S5", "AT7", "B10", "LX", "K1",
    ])
    .prop_map(str::to_owned)
}

fn value() -> impl Strategy<Value = String> {
    "[A-Za-z0-9 .,/#&()-]{0,12}"
}

proptest! {
    #![proptest_config(ProptestConfig::with_cases(256))]

    /// Whatever the writer writes reads back as the same segments with the same elements, apart
    /// from the trailing empty elements the writer leaves off.
    #[test]
    fn written_segments_read_back_unchanged(
        body in prop::collection::vec((segment_id(), prop::collection::vec(value(), 1..8)), 1..20),
    ) {
        let mut segments = vec![vec![Element::from("ST"), Element::from("214"), Element::from("0001")]];
        for (id, elements) in &body {
            let mut segment = vec![Element::from(id.as_str())];
            segment.extend(elements.iter().map(|value| Element::from(value.as_str())));
            segments.push(segment);
        }
        segments.push(vec![Element::from("SE"), Element::from(""), Element::from("0001")]);
        let written = write(&WriteRequest { separators: None, segments, counts: true })
            .expect("text");
        prop_assert!(written.findings.is_empty());

        let reading = read(&request(written.text, true)).expect("a reading");
        prop_assert_eq!(reading.segments.len(), body.len() + 2);
        for ((id, elements), segment) in body.iter().zip(&reading.segments[1..]) {
            prop_assert_eq!(&segment.id, id);
            let mut expected = elements.clone();
            while expected.last().is_some_and(String::is_empty) {
                expected.pop();
            }
            prop_assert_eq!(&segment.elements, &expected);
        }
        let transaction = &reading.interchanges[0].groups[0].transactions[0];
        prop_assert!(
            transaction.findings.iter().all(|finding| finding.kind != trenova_edi::x12::finding::FindingKind::SegmentCountMismatch),
            "the writer counts SE01"
        );
    }

    /// Any text reads without a panic, and every span stays inside it.
    #[test]
    fn any_text_reads_and_its_spans_stay_inside_it(text in "(ISA|GS|ST|SE|GE|IEA|N1|S5|[A-Z]{2,3})?[*~>^|\\n A-Z0-9]{0,200}") {
        for fragment in [false, true] {
            let reading = read(&request(text.clone(), fragment)).expect("a reading");
            for segment in &reading.segments {
                prop_assert!(segment.start <= segment.end && segment.end <= text.len());
            }
        }
    }
}
