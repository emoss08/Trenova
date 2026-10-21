//! Probe and Render through the real worker.

mod common;

use std::time::Duration;

use common::{chunked, fixture, reason, start};
use tonic::Code;
use trenova_proto::imaging::v1::{
    DocumentHeader, DocumentKind, FitBox, ImageFormat, ProbeHeader, ProbeRequest, ProbeResponse,
    RenderHeader, RenderRequest, RenderSize, probe_request, render_request, render_response,
    render_size,
};

async fn probe(
    worker: &mut common::Worker,
    content_type: &str,
    bytes: &[u8],
    text: bool,
) -> Result<ProbeResponse, tonic::Status> {
    let mut messages = vec![ProbeRequest {
        part: Some(probe_request::Part::Header(ProbeHeader {
            document: Some(DocumentHeader {
                content_type: content_type.into(),
                declared_size: bytes.len() as u64,
            }),
            text_profile: text,
        })),
    }];
    messages.extend(chunked(bytes, |c| ProbeRequest {
        part: Some(probe_request::Part::Chunk(c)),
    }));
    worker
        .client
        .probe(tokio_stream::iter(messages))
        .await
        .map(tonic::Response::into_inner)
}

#[derive(Debug)]
struct Page {
    index: u32,
    width: u32,
    height: u32,
    content_type: String,
    data: Vec<u8>,
}

fn render_messages(
    content_type: &str,
    bytes: &[u8],
    size: render_size::Size,
    format: ImageFormat,
) -> Vec<RenderRequest> {
    let mut messages = vec![RenderRequest {
        part: Some(render_request::Part::Header(RenderHeader {
            document: Some(DocumentHeader {
                content_type: content_type.into(),
                declared_size: bytes.len() as u64,
            }),
            size: Some(RenderSize { size: Some(size) }),
            format: format as i32,
            quality: 80,
            ..RenderHeader::default()
        })),
    }];
    messages.extend(chunked(bytes, |c| RenderRequest {
        part: Some(render_request::Part::Chunk(c)),
    }));
    messages
}

async fn render(
    worker: &mut common::Worker,
    content_type: &str,
    bytes: &[u8],
    size: render_size::Size,
    format: ImageFormat,
) -> Result<(Vec<Page>, u32), tonic::Status> {
    let messages = render_messages(content_type, bytes, size, format);
    let mut stream = worker
        .client
        .render(tokio_stream::iter(messages))
        .await?
        .into_inner();
    let mut pages: Vec<Page> = Vec::new();
    let mut errors = 0;
    while let Some(message) = stream.message().await? {
        match message.part {
            Some(render_response::Part::Page(header)) => pages.push(Page {
                index: header.index,
                width: header.width_px,
                height: header.height_px,
                content_type: header.content_type,
                data: Vec::with_capacity(usize::try_from(header.size).unwrap_or(0)),
            }),
            Some(render_response::Part::Chunk(chunk)) => {
                pages.last_mut().expect("a page").data.extend(chunk.data);
            }
            Some(render_response::Part::PageError(_)) => errors += 1,
            Some(render_response::Part::Summary(summary)) => {
                assert_eq!(summary.rendered as usize, pages.len());
            }
            None => {}
        }
    }
    Ok((pages, errors))
}

#[tokio::test]
async fn a_native_pdf_is_described_with_its_text_profile() {
    let Some(mut worker) = start(&[]).await else {
        return;
    };
    let described = probe(
        &mut worker,
        "application/pdf",
        &fixture("rate-confirmation.pdf"),
        true,
    )
    .await
    .expect("probes");
    assert_eq!(described.kind, DocumentKind::Pdf as i32);
    assert_eq!(described.page_count, 2);
    assert!(!described.encrypted);
    assert!(
        described
            .pages
            .iter()
            .all(|p| p.has_text && p.char_count > 50)
    );
    assert!((described.pages[0].width_points - 612.0).abs() < 0.5);
    assert!(described.pdf_version.starts_with("1."));
    assert!(
        described
            .producer
            .to_ascii_lowercase()
            .contains("reportlab")
    );

    let scanned = probe(
        &mut worker,
        "application/pdf",
        &fixture("capture-blank.pdf"),
        true,
    )
    .await
    .expect("probes");
    assert!(scanned.pages.iter().all(|p| !p.has_text));
}

#[tokio::test]
async fn an_encrypted_pdf_is_reported_not_failed() {
    let Some(mut worker) = start(&[]).await else {
        return;
    };
    let described = probe(
        &mut worker,
        "application/pdf",
        &fixture("hostile-encrypted.pdf"),
        false,
    )
    .await
    .expect("probes");
    assert!(described.encrypted);
}

#[tokio::test]
async fn bytes_that_are_not_what_they_claim_are_invalid() {
    let Some(mut worker) = start(&[]).await else {
        return;
    };
    let err = probe(
        &mut worker,
        "application/pdf",
        &fixture("hostile-garbage.pdf"),
        false,
    )
    .await
    .expect_err("refused");
    assert_eq!(err.code(), Code::InvalidArgument);
    assert_eq!(reason(&err), "INVALID_DOCUMENT");

    let err = probe(&mut worker, "image/png", &fixture("native-1p.pdf"), false)
        .await
        .expect_err("refused");
    assert_eq!(reason(&err), "INVALID_DOCUMENT");

    let err = probe(&mut worker, "text/html", b"<html>", false)
        .await
        .expect_err("refused");
    assert_eq!(reason(&err), "UNSUPPORTED_FORMAT");
}

#[tokio::test]
async fn a_stream_that_does_not_match_its_declared_size_is_refused() {
    let Some(mut worker) = start(&[]).await else {
        return;
    };
    let bytes = fixture("native-1p.pdf");
    let messages = vec![
        ProbeRequest {
            part: Some(probe_request::Part::Header(ProbeHeader {
                document: Some(DocumentHeader {
                    content_type: "application/pdf".into(),
                    declared_size: bytes.len() as u64 + 10,
                }),
                text_profile: false,
            })),
        },
        ProbeRequest {
            part: Some(probe_request::Part::Chunk(bytes.clone())),
        },
    ];
    let err = worker
        .client
        .probe(tokio_stream::iter(messages))
        .await
        .expect_err("refused");
    assert_eq!(reason(&err), "INVALID_DOCUMENT");

    let messages = vec![
        ProbeRequest {
            part: Some(probe_request::Part::Header(ProbeHeader {
                document: Some(DocumentHeader {
                    content_type: "application/pdf".into(),
                    declared_size: 10,
                }),
                text_profile: false,
            })),
        },
        ProbeRequest {
            part: Some(probe_request::Part::Chunk(bytes)),
        },
    ];
    let err = worker
        .client
        .probe(tokio_stream::iter(messages))
        .await
        .expect_err("refused");
    assert_eq!(reason(&err), "TOO_LARGE");
}

#[tokio::test]
async fn page_views_and_thumbnails_render_at_their_boxes() {
    let Some(mut worker) = start(&[]).await else {
        return;
    };
    let bytes = fixture("rate-confirmation.pdf");
    let (pages, errors) = render(
        &mut worker,
        "application/pdf",
        &bytes,
        render_size::Size::Fit(FitBox {
            max_width: 1400,
            max_height: 1800,
        }),
        ImageFormat::Webp,
    )
    .await
    .expect("renders");
    assert_eq!(errors, 0);
    assert_eq!(pages.len(), 2);
    for (i, page) in pages.iter().enumerate() {
        assert_eq!(page.index as usize, i);
        assert_eq!((page.width, page.height), (1391, 1800));
        assert_eq!(page.content_type, "image/webp");
        let decoded = image::load_from_memory(&page.data)
            .expect("a WebP")
            .to_luma8();
        assert_eq!(decoded.dimensions(), (1391, 1800));
        let mean =
            decoded.pixels().map(|p| u64::from(p.0[0])).sum::<u64>() / u64::from(1391 * 1800u32);
        assert!(
            mean > 200,
            "the page is mostly white paper: mean luma {mean}"
        );
    }

    let (thumbs, _) = render(
        &mut worker,
        "application/pdf",
        &bytes,
        render_size::Size::Fit(FitBox {
            max_width: 240,
            max_height: 0,
        }),
        ImageFormat::Jpeg,
    )
    .await
    .expect("renders");
    assert_eq!((thumbs[0].width, thumbs[0].height), (240, 311));
    assert_eq!(thumbs[0].content_type, "image/jpeg");
}

#[tokio::test]
async fn a_giant_page_is_drawn_within_the_pixel_budget() {
    let Some(mut worker) = start(&[("LIMITS__MAX_RENDER_PIXELS", "4000000")]).await else {
        return;
    };
    let (pages, _) = render(
        &mut worker,
        "application/pdf",
        &fixture("hostile-giant-page.pdf"),
        render_size::Size::Dpi(150),
        ImageFormat::Png,
    )
    .await
    .expect("renders");
    let page = &pages[0];
    assert!(
        u64::from(page.width) * u64::from(page.height) <= 4_010_000,
        "{} × {}",
        page.width,
        page.height
    );
}

#[tokio::test]
async fn a_phone_photo_comes_out_upright_and_reduced() {
    let Some(mut worker) = start(&[]).await else {
        return;
    };
    let photo = sideways_jpeg(1200, 900);
    let (pages, _) = render(
        &mut worker,
        "image/jpeg",
        &photo,
        render_size::Size::Fit(FitBox {
            max_width: 600,
            max_height: 600,
        }),
        ImageFormat::Png,
    )
    .await
    .expect("renders");
    let page = &pages[0];
    assert_eq!(
        (page.width, page.height),
        (450, 600),
        "the stored 1200 × 900 is shown 900 × 1200"
    );
}

#[tokio::test]
async fn hostile_pdfs_fail_alone_and_the_pool_carries_on() {
    let Some(mut worker) = start(&[]).await else {
        return;
    };
    for name in [
        "hostile-truncated.pdf",
        "hostile-page-tree-loop.pdf",
        "hostile-deep-nesting.pdf",
        "hostile-broken-xref.pdf",
    ] {
        match render(
            &mut worker,
            "application/pdf",
            &fixture(name),
            render_size::Size::Fit(FitBox {
                max_width: 240,
                max_height: 0,
            }),
            ImageFormat::Webp,
        )
        .await
        {
            Ok(_) => {}
            Err(err) => assert!(
                matches!(
                    err.code(),
                    Code::InvalidArgument | Code::Internal | Code::OutOfRange
                ),
                "{name}: {err}"
            ),
        }
    }
    let err = render(
        &mut worker,
        "image/jpeg",
        &fixture("hostile-pixel-bomb.jpg"),
        render_size::Size::Fit(FitBox {
            max_width: 240,
            max_height: 0,
        }),
        ImageFormat::Webp,
    )
    .await
    .expect_err("refused");
    assert_eq!(reason(&err), "PIXEL_BUDGET");
    let described = probe(
        &mut worker,
        "application/pdf",
        &fixture("native-1p.pdf"),
        false,
    )
    .await
    .expect("the pool still works");
    assert_eq!(described.page_count, 1);
    assert!(worker.pool.healthy());
}

#[tokio::test]
async fn a_render_its_caller_abandons_gives_its_slot_back() {
    let Some(mut worker) = start(&[]).await else {
        return;
    };
    let pdf = fixture("native-10p.pdf");
    let abandoned = tokio::time::timeout(Duration::from_secs(30), async {
        for _ in 0..=worker.pool.size() {
            let messages = render_messages(
                "application/pdf",
                &pdf,
                render_size::Size::Fit(FitBox {
                    max_width: 2400,
                    max_height: 0,
                }),
                ImageFormat::Png,
            );
            let mut stream = worker
                .client
                .render(tokio_stream::iter(messages))
                .await
                .expect("the render starts")
                .into_inner();
            stream
                .message()
                .await
                .expect("the first frame arrives")
                .expect("the render has output");
            drop(stream);
        }
        while worker.pool.available() < worker.pool.size() {
            tokio::time::sleep(Duration::from_millis(20)).await;
        }
    })
    .await;
    assert!(
        abandoned.is_ok(),
        "abandoned renders kept {} of {} slots",
        worker.pool.size() - worker.pool.available(),
        worker.pool.size()
    );

    let described = probe(&mut worker, "application/pdf", &pdf, false)
        .await
        .expect("a following job runs");
    assert_eq!(described.page_count, 10);
    assert!(worker.pool.healthy());
}

/// A JPEG stored landscape with EXIF orientation 6 (shown turned a quarter
/// clockwise), as phones write them.
fn sideways_jpeg(width: u32, height: u32) -> Vec<u8> {
    let mut rgb = Vec::with_capacity((width * height * 3) as usize);
    for y in 0..height {
        for x in 0..width {
            let v = u8::try_from((x + y) % 256).unwrap_or(0);
            rgb.extend_from_slice(&[v, 255 - v, 128]);
        }
    }
    let mut out = Vec::new();
    let mut encoder = jpeg_encoder::Encoder::new(&mut out, 85);
    let mut exif = b"MM\0\x2A\0\0\0\x08\0\x01".to_vec();
    exif.extend_from_slice(&[0x01, 0x12, 0, 3, 0, 0, 0, 1, 0, 6, 0, 0, 0, 0, 0, 0]);
    let mut segment = b"Exif\0\0".to_vec();
    segment.extend_from_slice(&exif);
    encoder.add_app_segment(1, segment).expect("exif");
    encoder
        .encode(
            &rgb,
            u16::try_from(width).unwrap_or(1),
            u16::try_from(height).unwrap_or(1),
            jpeg_encoder::ColorType::Rgb,
        )
        .expect("encodes");
    out
}
