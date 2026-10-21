//! Assembly through the real worker.

mod common;

use common::{chunked, fixture, reason, start};
use trenova_proto::imaging::v1::{
    AssembleHeader, AssembleOutput, AssembleRequest, AssembleSourceStart, DocumentHeader,
    PageRange, ProbeHeader, ProbeRequest, ProbeResponse, assemble_request, assemble_response,
    probe_request,
};

async fn assemble(
    worker: &mut common::Worker,
    sources: &[(&str, Vec<u8>)],
    outputs: Vec<Vec<PageRange>>,
) -> Result<(Vec<Vec<u8>>, Vec<u32>), tonic::Status> {
    let total: u64 = sources.iter().map(|(_, b)| b.len() as u64).sum();
    let mut messages = vec![AssembleRequest {
        part: Some(assemble_request::Part::Header(AssembleHeader {
            source_count: u32::try_from(sources.len()).expect("count"),
            outputs: outputs
                .into_iter()
                .map(|pages| AssembleOutput { pages })
                .collect(),
            total_size: total,
        })),
    }];
    for (index, (content_type, bytes)) in sources.iter().enumerate() {
        messages.push(AssembleRequest {
            part: Some(assemble_request::Part::Source(AssembleSourceStart {
                index: u32::try_from(index).expect("index"),
                document: Some(DocumentHeader {
                    content_type: (*content_type).into(),
                    declared_size: bytes.len() as u64,
                }),
            })),
        });
        messages.extend(chunked(bytes, |c| AssembleRequest {
            part: Some(assemble_request::Part::Chunk(c)),
        }));
    }
    let mut stream = worker
        .client
        .assemble(tokio_stream::iter(messages))
        .await?
        .into_inner();
    let mut files: Vec<Vec<u8>> = Vec::new();
    let mut counts = Vec::new();
    while let Some(message) = stream.message().await? {
        match message.part {
            Some(assemble_response::Part::Output(header)) => {
                assert_eq!(header.index as usize, files.len());
                files.push(Vec::new());
            }
            Some(assemble_response::Part::Chunk(chunk)) => {
                files.last_mut().expect("an output").extend(chunk.data);
            }
            Some(assemble_response::Part::Summary(summary)) => counts = summary.output_page_counts,
            None => {}
        }
    }
    Ok((files, counts))
}

async fn describe(worker: &mut common::Worker, pdf: &[u8]) -> ProbeResponse {
    let mut messages = vec![ProbeRequest {
        part: Some(probe_request::Part::Header(ProbeHeader {
            document: Some(DocumentHeader {
                content_type: "application/pdf".into(),
                declared_size: pdf.len() as u64,
            }),
            text_profile: true,
        })),
    }];
    messages.extend(chunked(pdf, |c| ProbeRequest {
        part: Some(probe_request::Part::Chunk(c)),
    }));
    worker
        .client
        .probe(tokio_stream::iter(messages))
        .await
        .expect("probes")
        .into_inner()
}

fn all(source: u32, pages: u32) -> PageRange {
    PageRange {
        source,
        first: 0,
        last: pages - 1,
        rotation: 0,
    }
}

fn photo_jpeg(orientation: u16) -> Vec<u8> {
    let (width, height) = (400u16, 300u16);
    let rgb: Vec<u8> = (0..u32::from(width) * u32::from(height))
        .flat_map(|i| [u8::try_from(i % 251).unwrap_or(0), 90, 160])
        .collect();
    let mut out = Vec::new();
    let mut encoder = jpeg_encoder::Encoder::new(&mut out, 85);
    let mut segment = b"Exif\0\0MM\0\x2A\0\0\0\x08\0\x01\x01\x12\0\x03\0\0\0\x01".to_vec();
    segment.extend_from_slice(&orientation.to_be_bytes());
    segment.extend_from_slice(&[0, 0, 0, 0, 0, 0]);
    encoder.add_app_segment(1, segment).expect("exif");
    encoder
        .encode(&rgb, width, height, jpeg_encoder::ColorType::Rgb)
        .expect("encodes");
    out
}

#[tokio::test]
async fn pdfs_and_a_photo_merge_into_one_document() {
    let Some(mut worker) = start(&[]).await else {
        return;
    };
    let photo = photo_jpeg(6);
    let sources = vec![
        ("application/pdf", fixture("native-1p.pdf")),
        ("application/pdf", fixture("rate-confirmation.pdf")),
        ("image/jpeg", photo.clone()),
    ];
    let (files, counts) = assemble(
        &mut worker,
        &sources,
        vec![vec![all(0, 1), all(1, 2), all(2, 1)]],
    )
    .await
    .expect("merges");
    assert_eq!(counts, vec![4]);
    let merged = &files[0];
    assert!(
        merged.windows(photo.len()).any(|w| w == photo.as_slice()),
        "the JPEG went in byte for byte"
    );
    let described = describe(&mut worker, merged).await;
    assert_eq!(described.page_count, 4);
    assert!(described.pages[..3].iter().all(|p| p.has_text));
    assert_eq!(
        described.pages[3].rotation, 90,
        "EXIF orientation 6 is the page's rotation"
    );
}

#[tokio::test]
async fn a_split_copies_each_range_straight_from_the_source() {
    let Some(mut worker) = start(&[]).await else {
        return;
    };
    let source = fixture("native-10p.pdf");
    let ranges = vec![
        vec![PageRange {
            source: 0,
            first: 0,
            last: 2,
            rotation: 0,
        }],
        vec![PageRange {
            source: 0,
            first: 3,
            last: 9,
            rotation: 0,
        }],
        vec![PageRange {
            source: 0,
            first: 5,
            last: 5,
            rotation: 270,
        }],
    ];
    let (files, counts) = assemble(&mut worker, &[("application/pdf", source.clone())], ranges)
        .await
        .expect("splits");
    assert_eq!(counts, vec![3, 7, 1]);
    assert!(files[0].len() < source.len());
    let extracted = describe(&mut worker, &files[2]).await;
    assert_eq!(extracted.pages[0].rotation, 270);
}

#[tokio::test]
async fn an_open_ended_range_runs_to_the_last_page() {
    let Some(mut worker) = start(&[]).await else {
        return;
    };
    let ranges = vec![vec![PageRange {
        source: 0,
        first: 4,
        last: u32::MAX,
        rotation: 0,
    }]];
    let (_, counts) = assemble(
        &mut worker,
        &[("application/pdf", fixture("native-10p.pdf"))],
        ranges,
    )
    .await
    .expect("assembles");
    assert_eq!(counts, vec![6]);
}

#[tokio::test]
async fn a_range_past_the_end_is_refused() {
    let Some(mut worker) = start(&[]).await else {
        return;
    };
    let err = assemble(
        &mut worker,
        &[("application/pdf", fixture("native-1p.pdf"))],
        vec![vec![PageRange {
            source: 0,
            first: 0,
            last: 4,
            rotation: 0,
        }]],
    )
    .await
    .expect_err("refused");
    assert_eq!(reason(&err), "PAGE_OUT_OF_RANGE");
}

#[tokio::test]
async fn a_black_and_white_scan_becomes_a_g4_page() {
    let Some(mut worker) = start(&[]).await else {
        return;
    };
    let mut png = Vec::new();
    let mut bilevel = image::GrayImage::from_pixel(850, 1100, image::Luma([255]));
    for y in 100..140 {
        for x in 100..700 {
            bilevel.put_pixel(x, y, image::Luma([0]));
        }
    }
    bilevel
        .write_to(&mut std::io::Cursor::new(&mut png), image::ImageFormat::Png)
        .expect("png");
    let (files, counts) = assemble(&mut worker, &[("image/png", png)], vec![vec![all(0, 1)]])
        .await
        .expect("converts");
    assert_eq!(counts, vec![1]);
    assert!(String::from_utf8_lossy(&files[0]).contains("CCITTFaxDecode"));
}
