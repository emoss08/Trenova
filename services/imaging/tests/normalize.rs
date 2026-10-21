//! Image normalization through the real worker.

mod common;

use common::{chunked, fixture, reason, start};
use trenova_proto::imaging::v1::{
    DocumentHeader, ImageFormat, NormalizeImageHeader, NormalizeImageRequest, NormalizeSummary,
    normalize_image_request, normalize_image_response,
};

async fn normalize(
    worker: &mut common::Worker,
    content_type: &str,
    bytes: &[u8],
    format: ImageFormat,
    max: u32,
) -> Result<(Vec<u8>, String, NormalizeSummary), tonic::Status> {
    let mut messages = vec![NormalizeImageRequest {
        part: Some(normalize_image_request::Part::Header(
            NormalizeImageHeader {
                document: Some(DocumentHeader {
                    content_type: content_type.into(),
                    declared_size: bytes.len() as u64,
                }),
                format: format as i32,
                quality: 0,
                max_width: max,
                max_height: max,
            },
        )),
    }];
    messages.extend(chunked(bytes, |c| NormalizeImageRequest {
        part: Some(normalize_image_request::Part::Chunk(c)),
    }));
    let mut stream = worker
        .client
        .normalize_image(tokio_stream::iter(messages))
        .await?
        .into_inner();
    let (mut data, mut content_type, mut summary) = (Vec::new(), String::new(), None);
    while let Some(message) = stream.message().await? {
        match message.part {
            Some(normalize_image_response::Part::Image(header)) => {
                content_type = header.content_type;
            }
            Some(normalize_image_response::Part::Chunk(chunk)) => data.extend(chunk.data),
            Some(normalize_image_response::Part::Summary(s)) => summary = Some(s),
            None => {}
        }
    }
    Ok((data, content_type, summary.expect("a summary")))
}

#[tokio::test]
async fn webp_becomes_png_for_the_asset_inliner() {
    let Some(mut worker) = start(&[]).await else {
        return;
    };
    let (png, content_type, summary) = normalize(
        &mut worker,
        "image/webp",
        &fixture("photo.webp"),
        ImageFormat::Png,
        0,
    )
    .await
    .expect("normalizes");
    assert_eq!(content_type, "image/png");
    assert_eq!(summary.source_format, "webp");
    let decoded =
        image::load_from_memory_with_format(&png, image::ImageFormat::Png).expect("a PNG");
    assert_eq!((decoded.width(), decoded.height()), (320, 240));
}

#[tokio::test]
async fn heic_from_a_phone_becomes_jpeg() {
    if std::env::var("TRENOVA_HEIF_LIBRARY").is_err() {
        return;
    }
    let Some(mut worker) = start(&[]).await else {
        return;
    };
    let (jpeg, content_type, summary) = normalize(
        &mut worker,
        "image/heic",
        &fixture("photo.heic"),
        ImageFormat::Jpeg,
        160,
    )
    .await
    .expect("normalizes");
    assert_eq!(content_type, "image/jpeg");
    assert_eq!(summary.source_format, "heic");
    let decoded = image::load_from_memory(&jpeg).expect("a JPEG");
    assert_eq!((decoded.width(), decoded.height()), (160, 120));
}

#[tokio::test]
async fn a_pdf_is_not_an_image() {
    let Some(mut worker) = start(&[]).await else {
        return;
    };
    let err = normalize(
        &mut worker,
        "application/pdf",
        &fixture("native-1p.pdf"),
        ImageFormat::Png,
        0,
    )
    .await
    .expect_err("refused");
    assert_eq!(reason(&err), "UNSUPPORTED_FORMAT");
}
