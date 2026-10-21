//! Assemble: new PDFs built from pages of the sources.
//!
//! Every output is a fresh document that PDFium's page import fills straight
//! from the sources, so a split into ranges copies each range once (with
//! only the resources its pages use) rather than splitting everything and
//! merging it back. Each range can be turned on top of its pages' own
//! rotation. An image source becomes a one-page PDF first, through the same
//! writer the capture companion uses: a JPEG goes in exactly as it is (its
//! EXIF rotation becomes the page's rotation), a black-and-white image as
//! CCITT G4, anything else as JPEG; the image is fitted onto a letter sheet.

use pdfium_render::prelude::PdfDocument as PdfiumDocument;
use trenova_imaging_core::{
    JpegColor, PageFit, PassthroughJpeg, PdfDocument, PixelFormat, Raster, Resolution, binarize_at,
};
use trenova_proto::imaging::v1::{
    ArtifactHeader, AssembleHeader, AssembleResponse, AssembleSummary, assemble_response::Part,
};

use super::decode::{DecodeOptions, ImageKind, decode, exif_orientation};
use super::error::{EngineError, EngineResult};
use super::raster::Rgba;
use super::{Engine, Input, Kind, Sink, emit_artifact, kind_of, pdf};
use crate::protocol::JobLimits;

/// The resolution an image page is written at; the page is fitted to a
/// letter sheet, so it only sets the image's proportions.
const IMAGE_DPI: u32 = 200;
const IMAGE_JPEG_QUALITY: u8 = 90;
const PRODUCER: &str = "Trenova";

struct Source<'a> {
    document: PdfiumDocument<'a>,
    /// Clockwise degrees an image's own orientation adds to its page.
    rotation: u32,
}

pub fn assemble(
    engine: &Engine,
    header: &AssembleHeader,
    inputs: &[Input<'_>],
    limits: JobLimits,
    sink: &mut dyn Sink,
) -> EngineResult<AssembleSummary> {
    let mut sources = Vec::with_capacity(inputs.len());
    for input in inputs {
        sources.push(open_source(engine, input, limits).map_err(|err| {
            EngineError::new(
                err.reason,
                format!("source {}: {}", input.index, err.message),
            )
        })?);
    }

    let mut counts = Vec::with_capacity(header.outputs.len());
    for (index, output) in header.outputs.iter().enumerate() {
        let mut document = engine.pdf().create()?;
        let mut total: u32 = 0;
        for range in &output.pages {
            let source = sources.get(range.source as usize).ok_or_else(|| {
                EngineError::request(format!(
                    "output {index} names source {}, which was not sent",
                    range.source
                ))
            })?;
            let count = pdf::page_count(&source.document);
            let last = if range.last == u32::MAX {
                count.saturating_sub(1)
            } else {
                range.last
            };
            if count == 0 || range.first > last || last >= count {
                return Err(EngineError::page(last.max(range.first), count));
            }
            if range.rotation % 90 != 0 {
                return Err(EngineError::request(format!(
                    "a rotation of {} is not a quarter turn",
                    range.rotation
                )));
            }
            let pages = last - range.first + 1;
            if total + pages > limits.max_pages {
                return Err(EngineError::request(format!(
                    "an output may hold at most {} pages",
                    limits.max_pages
                )));
            }
            let first = page_index(range.first)?;
            let last = page_index(last)?;
            let at = page_index(total)?;
            document
                .pages_mut()
                .copy_page_range_from_document(&source.document, first..=last, at)
                .map_err(|err| {
                    EngineError::invalid(format!(
                        "pages of source {} could not be copied: {err}",
                        range.source
                    ))
                })?;

            let turn = (source.rotation + range.rotation) % 360;
            if turn != 0 {
                for offset in 0..pages {
                    let mut page = pdf::page(&document, total + offset)?;
                    let current = pdf::rotation_degrees(&page);
                    page.set_rotation(pdf::rotation_from_degrees((current + turn) % 360));
                }
            }
            total += pages;
        }
        if total == 0 {
            return Err(EngineError::request(format!("output {index} has no pages")));
        }
        let bytes = document.save_to_bytes().map_err(|err| {
            EngineError::internal(format!("output {index} could not be written: {err}"))
        })?;
        emit_artifact(
            sink,
            ArtifactHeader {
                index: u32::try_from(index).unwrap_or(u32::MAX),
                content_type: "application/pdf".into(),
                width_px: 0,
                height_px: 0,
                size: bytes.len() as u64,
            },
            &bytes,
            limits.chunk_bytes,
            |output| AssembleResponse {
                part: Some(Part::Output(output)),
            },
            |chunk| AssembleResponse {
                part: Some(Part::Chunk(chunk)),
            },
        )?;
        counts.push(total);
    }

    Ok(AssembleSummary {
        output_page_counts: counts,
    })
}

fn page_index(index: u32) -> EngineResult<i32> {
    i32::try_from(index).map_err(|_| EngineError::request("a page index is too large"))
}

fn open_source<'a>(
    engine: &'a Engine,
    input: &Input<'a>,
    limits: JobLimits,
) -> EngineResult<Source<'a>> {
    match kind_of(input)? {
        Kind::Pdf => Ok(Source {
            document: engine.pdf().open(input.bytes)?,
            rotation: 0,
        }),
        Kind::Image(kind) => {
            let (bytes, rotation) = image_pdf(engine, input.bytes, kind, limits)?;
            Ok(Source {
                document: engine.pdf().open_owned(bytes)?,
                rotation,
            })
        }
    }
}

/// An image as a one-page PDF, and the rotation its orientation adds.
pub fn image_pdf(
    engine: &Engine,
    bytes: &[u8],
    kind: ImageKind,
    limits: JobLimits,
) -> EngineResult<(Vec<u8>, u32)> {
    let resolution = Resolution::square(IMAGE_DPI);
    let mut document = PdfDocument::new()
        .with_fit(PageFit::LETTER)
        .with_producer(PRODUCER);

    if kind == ImageKind::Jpeg
        && let Some((jpeg, rotation)) = passthrough(bytes, limits)?
    {
        document.push_jpeg(jpeg, resolution)?;
        return Ok((document.finish(), rotation));
    }

    let decoded = decode(
        bytes,
        DecodeOptions {
            max_pixels: limits.max_image_pixels,
            want_width: 0,
            want_height: 0,
        },
        engine.heif(),
    )?;
    push_raster(&mut document, &decoded.image, resolution)?;
    Ok((document.finish(), 0))
}

/// A JPEG that can go into the PDF untouched: its size, colour and the page
/// rotation its EXIF orientation means. `None` for a mirrored orientation,
/// which a page rotation cannot express.
fn passthrough(bytes: &[u8], limits: JobLimits) -> EngineResult<Option<(PassthroughJpeg, u32)>> {
    let mut decoder = jpeg_decoder::Decoder::new(std::io::Cursor::new(bytes));
    decoder
        .read_info()
        .map_err(|err| EngineError::invalid(format!("the JPEG could not be read: {err}")))?;
    let info = decoder
        .info()
        .ok_or_else(|| EngineError::invalid("the JPEG has no frame"))?;
    let (width, height) = (u32::from(info.width), u32::from(info.height));
    if u64::from(width) * u64::from(height) > limits.max_image_pixels {
        return Err(EngineError::pixels(format!(
            "a {width} × {height} image is over the pixel budget"
        )));
    }
    let color = match info.pixel_format {
        jpeg_decoder::PixelFormat::L8 | jpeg_decoder::PixelFormat::L16 => JpegColor::Gray,
        jpeg_decoder::PixelFormat::RGB24 => JpegColor::Rgb,
        jpeg_decoder::PixelFormat::CMYK32 => JpegColor::Cmyk { inverted: true },
    };
    let rotation = match decoder.exif_data().and_then(exif_orientation).unwrap_or(1) {
        1 => 0,
        3 => 180,
        6 => 90,
        8 => 270,
        _ => return Ok(None),
    };
    Ok(Some((
        PassthroughJpeg {
            data: bytes.to_vec(),
            width,
            height,
            color,
        },
        rotation,
    )))
}

/// A decoded image onto a page: G4 when it is black and white, JPEG
/// otherwise (gray when it has no colour).
fn push_raster(
    document: &mut PdfDocument,
    image: &Rgba,
    resolution: Resolution,
) -> EngineResult<()> {
    let gray = image
        .data
        .chunks_exact(4)
        .all(|p| p[0] == p[1] && p[1] == p[2]);
    if gray {
        let luma: Vec<u8> = image.data.chunks_exact(4).map(|p| p[0]).collect();
        let raster = Raster::new(
            image.width,
            image.height,
            image.width as usize,
            PixelFormat::Gray8,
            &luma,
        )?;
        if luma.iter().all(|v| !(16..=239).contains(v)) {
            let bilevel = binarize_at(&raster, 128);
            document.push(&bilevel.as_raster()?, resolution, IMAGE_JPEG_QUALITY)?;
        } else {
            document.push(&raster, resolution, IMAGE_JPEG_QUALITY)?;
        }
        return Ok(());
    }
    let rgb = image.to_rgb();
    let raster = Raster::new(
        image.width,
        image.height,
        image.width as usize * 3,
        PixelFormat::Rgb8,
        &rgb,
    )?;
    document.push(&raster, resolution, IMAGE_JPEG_QUALITY)?;
    Ok(())
}
