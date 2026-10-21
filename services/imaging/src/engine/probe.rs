//! Probe: is this a readable document, and what is in it.
//!
//! For a PDF: the page count, every page's size and rotation and (when asked)
//! whether it carries native text, whether it is encrypted, its version and
//! producer. An encrypted PDF that needs a password is reported as encrypted,
//! not as an error, so the caller decides what to do with it. For an image:
//! its format, upright size and EXIF orientation.

use trenova_proto::imaging::v1::{DocumentKind, ErrorReason, PageInfo, ProbeHeader, ProbeResponse};

use super::decode::{DecodeOptions, decode};
use super::error::EngineResult;
use super::{Engine, Input, Kind, kind_of, pdf};
use crate::protocol::JobLimits;

pub fn probe(
    engine: &Engine,
    header: &ProbeHeader,
    input: Input<'_>,
    limits: JobLimits,
) -> EngineResult<ProbeResponse> {
    match kind_of(&input)? {
        Kind::Pdf => probe_pdf(engine, header, input.bytes),
        Kind::Image(_) => {
            let decoded = decode(
                input.bytes,
                DecodeOptions {
                    max_pixels: limits.max_image_pixels,
                    want_width: 64,
                    want_height: 64,
                },
                engine.heif(),
            )?;
            let (width, height) = if (5..=8).contains(&decoded.orientation) {
                (decoded.source_height, decoded.source_width)
            } else {
                (decoded.source_width, decoded.source_height)
            };
            #[allow(clippy::cast_precision_loss)]
            Ok(ProbeResponse {
                kind: DocumentKind::Image as i32,
                page_count: 1,
                pages: vec![PageInfo {
                    index: 0,
                    width_points: width as f32,
                    height_points: height as f32,
                    rotation: 0,
                    has_text: false,
                    char_count: 0,
                }],
                encrypted: false,
                pdf_version: String::new(),
                producer: String::new(),
                image_format: decoded.kind.name().to_owned(),
                orientation: u32::from(decoded.orientation),
            })
        }
    }
}

fn probe_pdf(engine: &Engine, header: &ProbeHeader, bytes: &[u8]) -> EngineResult<ProbeResponse> {
    let document = match engine.pdf().open(bytes) {
        Ok(document) => document,
        Err(err) if err.reason == ErrorReason::Encrypted => {
            return Ok(ProbeResponse {
                kind: DocumentKind::Pdf as i32,
                encrypted: true,
                ..ProbeResponse::default()
            });
        }
        Err(err) => return Err(err),
    };
    let count = pdf::page_count(&document);
    let mut pages = Vec::with_capacity(count as usize);
    for index in 0..count {
        let page = pdf::page(&document, index)?;
        pages.push(pdf::page_info(&page, index, header.text_profile));
    }
    Ok(ProbeResponse {
        kind: DocumentKind::Pdf as i32,
        page_count: count,
        pages,
        encrypted: pdf::is_encrypted(&document),
        pdf_version: pdf::version(&document),
        producer: pdf::producer(&document),
        image_format: String::new(),
        orientation: 0,
    })
}
