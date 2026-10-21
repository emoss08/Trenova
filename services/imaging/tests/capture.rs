//! Capture batches through the real worker.

mod common;

use common::{chunked, fixture, start};
use trenova_proto::imaging::v1::{
    CapturePageResult, CapturePageStart, CaptureSummary, DocumentHeader, ErrorReason,
    InspectCaptureHeader, InspectCaptureRequest, inspect_capture_request::Part,
    inspect_capture_response,
};

async fn inspect(
    worker: &mut common::Worker,
    pages: &[(&str, Vec<u8>)],
) -> (Vec<CapturePageResult>, CaptureSummary) {
    let mut messages = vec![InspectCaptureRequest {
        part: Some(Part::Header(InspectCaptureHeader {
            page_count: u32::try_from(pages.len()).expect("count"),
            ..InspectCaptureHeader::default()
        })),
    }];
    for (index, (content_type, bytes)) in pages.iter().enumerate() {
        messages.push(InspectCaptureRequest {
            part: Some(Part::Page(CapturePageStart {
                index: u32::try_from(index).expect("index"),
                document: Some(DocumentHeader {
                    content_type: (*content_type).into(),
                    declared_size: bytes.len() as u64,
                }),
            })),
        });
        messages.extend(chunked(bytes, |c| InspectCaptureRequest {
            part: Some(Part::Chunk(c)),
        }));
    }
    let mut stream = worker
        .client
        .inspect_capture(tokio_stream::iter(messages))
        .await
        .expect("starts")
        .into_inner();
    let mut results = Vec::new();
    let mut summary = None;
    while let Some(message) = stream.message().await.expect("streams") {
        match message.part {
            Some(inspect_capture_response::Part::Page(page)) => results.push(page),
            Some(inspect_capture_response::Part::Summary(s)) => summary = Some(s),
            None => {}
        }
    }
    results.sort_by_key(|page| page.index);
    (results, summary.expect("a summary"))
}

#[tokio::test]
async fn a_batch_is_read_page_by_page_in_one_pass_each() {
    let Some(mut worker) = start(&[]).await else {
        return;
    };
    let pages = vec![
        ("application/pdf", fixture("capture-cover-2qr.pdf")),
        ("application/pdf", fixture("capture-blank.pdf")),
        ("application/pdf", fixture("capture-text-0.pdf")),
        ("application/pdf", fixture("hostile-garbage.pdf")),
        ("application/pdf", Vec::new()),
    ];
    let (results, summary) = inspect(&mut worker, &pages).await;
    assert_eq!(results.len(), 5);
    assert_eq!(
        (summary.inspected, summary.failed),
        (3, 2),
        "{:?}",
        results.iter().map(|r| r.error.clone()).collect::<Vec<_>>()
    );

    let cover = &results[0];
    let mut codes: Vec<&str> = cover.codes.iter().map(|c| c.text.as_str()).collect();
    codes.sort_unstable();
    assert_eq!(
        codes,
        [
            "trenova:cs:01JD3Q5V8Z2K7W4M6N9P0R1T2X",
            "trenova:cs:01JD3Q5V8Z2K7W4M6N9P0R1T3Y"
        ]
    );
    assert!(cover.codes.iter().all(|c| c.format == "qr_code"));
    assert_eq!((cover.width_px, cover.height_px), (1275, 1650));
    assert_eq!(cover.thumbnail_content_type, "image/jpeg");
    let thumb = image::load_from_memory(&cover.thumbnail).expect("a JPEG");
    assert_eq!(thumb.width(), 240);

    assert!(
        results[1].ink_coverage < 0.005,
        "blank sheet: {}",
        results[1].ink_coverage
    );
    assert!(results[1].codes.is_empty());
    assert!(
        results[2].ink_coverage > 0.01,
        "text page: {}",
        results[2].ink_coverage
    );
    assert!(results[3].error.is_some());
    let empty = results[4]
        .error
        .as_ref()
        .expect("an empty page fails in band");
    assert_eq!(empty.page_index, 4);
}

#[tokio::test]
async fn a_long_batch_runs_in_sub_batches_and_answers_every_page() {
    let Some(mut worker) = start(&[
        ("LIMITS__CAPTURE_BATCH", "3"),
        ("LIMITS__CAPTURE_PARALLELISM", "2"),
    ])
    .await
    else {
        return;
    };
    let text = fixture("capture-text-0.pdf");
    let pages: Vec<(&str, Vec<u8>)> = (0..20).map(|_| ("application/pdf", text.clone())).collect();
    let (results, summary) = inspect(&mut worker, &pages).await;
    assert_eq!(summary.inspected, 20);
    assert_eq!(
        results.iter().map(|p| p.index).collect::<Vec<_>>(),
        (0..20).collect::<Vec<_>>()
    );
    assert!(
        results
            .iter()
            .all(|p| p.error.is_none() && p.ink_coverage > 0.01)
    );
}

#[tokio::test]
async fn a_batch_larger_than_one_input_is_inspected_page_by_page() {
    let Some(mut worker) = start(&[
        ("LIMITS__MAX_INPUT_BYTES", "\"1MiB\""),
        ("LIMITS__MAX_CAPTURE_PAGE_BYTES", "\"256KiB\""),
        ("LIMITS__CAPTURE_PARALLELISM", "2"),
    ])
    .await
    else {
        return;
    };
    let text = fixture("capture-text-0.pdf");
    let mut pages: Vec<(&str, Vec<u8>)> =
        (0..14).map(|_| ("application/pdf", text.clone())).collect();
    let total: usize = pages.iter().map(|(_, bytes)| bytes.len()).sum();
    assert!(total > 2 << 20, "the batch is over twice one input");
    let mut oversized = text.clone();
    oversized.resize(300 << 10, b'\n');
    pages.push(("application/pdf", oversized));

    let (results, summary) = inspect(&mut worker, &pages).await;
    assert_eq!(summary.inspected, 14);
    assert_eq!(results.len(), 15);
    assert!(
        results[..14]
            .iter()
            .all(|p| p.error.is_none() && p.ink_coverage > 0.01),
        "every page under the page limit is inspected"
    );
    let refused = results[14]
        .error
        .as_ref()
        .expect("the oversized page fails in band");
    assert_eq!(refused.reason, ErrorReason::TooLarge as i32);
}
