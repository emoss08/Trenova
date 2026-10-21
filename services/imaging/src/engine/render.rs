//! Render: pages drawn to WebP, JPEG or PNG, at a DPI or fitted into a box.
//!
//! A PDF page is drawn by PDFium straight at the size it is wanted, onto
//! white, rather than drawn large and reduced. An image is decoded (at a
//! reduced size when its codec can), turned upright, and reduced to fit. A
//! page that cannot be drawn is reported in band and the rest carry on.

use trenova_proto::imaging::v1::{
    ArtifactHeader, ErrorReason, ImageFormat, PageError, RenderHeader, RenderResponse,
    RenderSummary, render_response::Part, render_size::Size,
};

use super::decode::{DecodeOptions, decode};
use super::error::{EngineError, EngineResult};
use super::raster::{Rgba, encode, fit_within};
use super::{Engine, Input, Kind, Sink, emit_artifact, kind_of, pdf};
use crate::protocol::JobLimits;

pub fn render(
    engine: &Engine,
    header: &RenderHeader,
    input: Input<'_>,
    limits: JobLimits,
    sink: &mut dyn Sink,
) -> EngineResult<RenderSummary> {
    let size = header
        .size
        .as_ref()
        .and_then(|size| size.size)
        .ok_or_else(|| EngineError::request("a render needs a size"))?;
    let max_pixels = match header.max_pixels {
        0 => limits.max_pixels,
        asked => asked.min(limits.max_pixels),
    };
    let format = ImageFormat::try_from(header.format).unwrap_or(ImageFormat::Webp);

    match kind_of(&input)? {
        Kind::Pdf => render_pdf(
            engine,
            header,
            input.bytes,
            size,
            max_pixels,
            format,
            limits,
            sink,
        ),
        Kind::Image(_) => {
            if header.pages.iter().any(|&page| page != 0) {
                return Err(EngineError::page(
                    header.pages.iter().copied().max().unwrap_or(0),
                    1,
                ));
            }
            let (want_width, want_height) = match size {
                Size::Fit(fit) => (fit.max_width, fit.max_height),
                Size::Dpi(_) => (0, 0),
            };
            let decoded = decode(
                input.bytes,
                DecodeOptions {
                    max_pixels: limits.max_image_pixels,
                    want_width,
                    want_height,
                },
                engine.heif(),
            )?;
            let image = fit_image(decoded.image, want_width, want_height, max_pixels)?;
            emit_page(sink, 0, &image, header, format, limits.chunk_bytes)?;
            Ok(RenderSummary {
                page_count: 1,
                rendered: 1,
            })
        }
    }
}

#[allow(clippy::too_many_arguments)]
fn render_pdf(
    engine: &Engine,
    header: &RenderHeader,
    bytes: &[u8],
    size: Size,
    max_pixels: u64,
    format: ImageFormat,
    limits: JobLimits,
    sink: &mut dyn Sink,
) -> EngineResult<RenderSummary> {
    let document = engine.pdf().open(bytes)?;
    let count = pdf::page_count(&document);
    let wanted: Vec<u32> = if header.pages.is_empty() {
        let cap = match header.max_pages {
            0 => limits.max_pages,
            asked => asked.min(limits.max_pages),
        };
        (0..count.min(cap)).collect()
    } else {
        if header.pages.len() > limits.max_pages as usize {
            return Err(EngineError::request(format!(
                "a render may ask for at most {} pages",
                limits.max_pages
            )));
        }
        header.pages.clone()
    };

    let mut rendered = 0;
    for index in wanted {
        let result = pdf::page(&document, index).and_then(|page| {
            let (width, height) = match size {
                Size::Dpi(dpi) => {
                    pdf::size_at_dpi(page.width().value, page.height().value, dpi, max_pixels)?
                }
                Size::Fit(fit) => pdf::size_to_fit(
                    page.width().value,
                    page.height().value,
                    fit.max_width,
                    fit.max_height,
                    max_pixels,
                )?,
            };
            pdf::render(&page, width, height)
        });
        match result {
            Ok(image) => {
                emit_page(sink, index, &image, header, format, limits.chunk_bytes)?;
                rendered += 1;
            }
            Err(err) => sink.emit(prost::Message::encode_to_vec(&RenderResponse {
                part: Some(Part::PageError(PageError {
                    page_index: index,
                    reason: err.reason as i32,
                    message: err.message,
                })),
            }))?,
        }
    }
    Ok(RenderSummary {
        page_count: count,
        rendered,
    })
}

fn fit_image(image: Rgba, max_width: u32, max_height: u32, max_pixels: u64) -> EngineResult<Rgba> {
    let (mut width, mut height) = fit_within(image.width, image.height, max_width, max_height);
    let pixels = u64::from(width) * u64::from(height);
    if max_pixels > 0 && pixels > max_pixels {
        #[allow(clippy::cast_precision_loss)]
        let shrink = (max_pixels as f64 / pixels as f64).sqrt();
        #[allow(clippy::cast_possible_truncation, clippy::cast_sign_loss)]
        {
            width = ((f64::from(width) * shrink) as u32).max(1);
            height = ((f64::from(height) * shrink) as u32).max(1);
        }
    }
    if (width, height) == (image.width, image.height) {
        return Ok(image);
    }
    image.resize(width, height)
}

fn emit_page(
    sink: &mut dyn Sink,
    index: u32,
    image: &Rgba,
    header: &RenderHeader,
    format: ImageFormat,
    chunk_bytes: u32,
) -> EngineResult<()> {
    let encoded = encode(image, format, header.quality, header.lossless).map_err(|err| {
        if err.reason == ErrorReason::PixelBudget {
            err
        } else {
            EngineError::internal(format!(
                "page {index} could not be encoded: {}",
                err.message
            ))
        }
    })?;
    emit_artifact(
        sink,
        ArtifactHeader {
            index,
            content_type: encoded.content_type.to_owned(),
            width_px: image.width,
            height_px: image.height,
            size: encoded.data.len() as u64,
        },
        &encoded.data,
        chunk_bytes,
        |page| RenderResponse {
            part: Some(Part::Page(page)),
        },
        |chunk| RenderResponse {
            part: Some(Part::Chunk(chunk)),
        },
    )
}
