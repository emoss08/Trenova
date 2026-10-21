//! Capture inspection: each scanned page read once for everything the
//! intake queue needs.
//!
//! A page (a one-page PDF from the capture companion, or an image) is drawn
//! or decoded once. From that one raster come the thumbnail the queue shows,
//! the share of the page carrying ink (the companion's own measure from
//! `trenova-imaging-core`, with the server's threshold, margins and stride),
//! and every QR code and barcode on it, cover sheets included. A page that
//! cannot be read is reported with its error; the batch carries on.

use std::collections::{BTreeSet, HashSet};

use rxing::{BarcodeFormat, DecodeHints};
use trenova_imaging_core::{InkParams, PixelFormat, Raster, ink_coverage};
use trenova_proto::imaging::v1::{
    Barcode, CapturePageResult, CaptureSummary, ImageFormat, InspectCaptureHeader,
    InspectCaptureResponse, PageError, inspect_capture_response::Part,
};

use super::decode::{DecodeOptions, decode};
use super::error::{EngineError, EngineResult};
use super::raster::{Rgba, encode, fit_within};
use super::{Engine, Input, Kind, Sink, kind_of, pdf};
use crate::protocol::JobLimits;

const DEFAULT_DPI: u32 = 150;
const DEFAULT_THUMBNAIL_WIDTH: u32 = 240;
const DEFAULT_THUMBNAIL_QUALITY: u32 = 72;
/// An image page is read at no more than a letter page at the inspection
/// DPI would be: phone photos are decoded smaller, which is all the measures
/// need.
const LETTER_INCHES: (u32, u32) = (17, 22);

/// The symbologies a page is searched for.
const FORMATS: [BarcodeFormat; 10] = [
    BarcodeFormat::QR_CODE,
    BarcodeFormat::DATA_MATRIX,
    BarcodeFormat::PDF_417,
    BarcodeFormat::AZTEC,
    BarcodeFormat::CODE_128,
    BarcodeFormat::CODE_39,
    BarcodeFormat::CODE_93,
    BarcodeFormat::ITF,
    BarcodeFormat::EAN_13,
    BarcodeFormat::UPC_A,
];

pub fn inspect(
    engine: &Engine,
    header: &InspectCaptureHeader,
    inputs: &[Input<'_>],
    limits: JobLimits,
    sink: &mut dyn Sink,
) -> EngineResult<CaptureSummary> {
    let mut failed = 0;
    for input in inputs {
        let result = inspect_page(engine, header, input, limits).unwrap_or_else(|err| {
            failed += 1;
            CapturePageResult {
                index: input.index,
                error: Some(PageError {
                    page_index: input.index,
                    reason: err.reason as i32,
                    message: err.message,
                }),
                ..CapturePageResult::default()
            }
        });
        sink.emit(prost::Message::encode_to_vec(&InspectCaptureResponse {
            part: Some(Part::Page(result)),
        }))?;
    }
    Ok(CaptureSummary {
        inspected: u32::try_from(inputs.len()).unwrap_or(u32::MAX) - failed,
        failed,
    })
}

fn inspect_page(
    engine: &Engine,
    header: &InspectCaptureHeader,
    input: &Input<'_>,
    limits: JobLimits,
) -> EngineResult<CapturePageResult> {
    let dpi = if header.render_dpi == 0 {
        DEFAULT_DPI
    } else {
        header.render_dpi.clamp(72, 300)
    };
    let page = match kind_of(input)? {
        Kind::Pdf => {
            let document = engine.pdf().open(input.bytes)?;
            if pdf::page_count(&document) == 0 {
                return Err(EngineError::invalid("the PDF has no pages"));
            }
            let page = pdf::page(&document, 0)?;
            let (width, height) = pdf::size_at_dpi(
                page.width().value,
                page.height().value,
                dpi,
                limits.max_pixels,
            )?;
            pdf::render(&page, width, height)?
        }
        Kind::Image(_) => {
            let (want_w, want_h) = (LETTER_INCHES.0 * dpi / 2, LETTER_INCHES.1 * dpi / 2);
            let decoded = decode(
                input.bytes,
                DecodeOptions {
                    max_pixels: limits.max_image_pixels,
                    want_width: want_w,
                    want_height: want_h,
                },
                engine.heif(),
            )?;
            let (w, h) = fit_within(decoded.image.width, decoded.image.height, want_w, want_h);
            decoded.image.resize(w, h)?
        }
    };
    let luma = page.to_luma();
    let params = ink_params(header);
    let gray = Raster::new(
        page.width,
        page.height,
        page.width as usize,
        PixelFormat::Gray8,
        &luma,
    )?;
    let ink = ink_coverage(&gray, params);

    let thumbnail = thumbnail(&page, header)?;
    let codes = if header.skip_barcodes {
        Vec::new()
    } else {
        barcodes(luma, page.width, page.height)
    };

    Ok(CapturePageResult {
        index: input.index,
        width_px: page.width,
        height_px: page.height,
        thumbnail: thumbnail.data,
        thumbnail_content_type: thumbnail.content_type.to_owned(),
        ink_coverage: ink,
        codes,
        error: None,
    })
}

/// The server's measure unless the call overrides it.
pub fn ink_params(header: &InspectCaptureHeader) -> InkParams {
    let server = InkParams::SERVER;
    InkParams {
        threshold: match header.ink_threshold {
            0 => server.threshold,
            t => u8::try_from(t.min(255)).unwrap_or(u8::MAX),
        },
        margin_fraction: if header.ink_margin > 0.0 && header.ink_margin < 0.5 {
            header.ink_margin
        } else {
            server.margin_fraction
        },
        stride: match header.ink_stride {
            0 => server.stride,
            s => s.min(16),
        },
    }
}

fn thumbnail(page: &Rgba, header: &InspectCaptureHeader) -> EngineResult<super::raster::Encoded> {
    let width = match header.thumbnail_width {
        0 => DEFAULT_THUMBNAIL_WIDTH,
        w => w.min(1200),
    };
    let (w, h) = fit_within(page.width, page.height, width, 0);
    let small = page.resize(w, h)?;
    let format = match ImageFormat::try_from(header.thumbnail_format) {
        Ok(ImageFormat::Unspecified) | Err(_) => ImageFormat::Jpeg,
        Ok(format) => format,
    };
    let quality = match header.thumbnail_quality {
        0 => DEFAULT_THUMBNAIL_QUALITY,
        q => q,
    };
    encode(&small, format, quality, false)
}

/// Every QR code and barcode on the page. Nothing found, or a code too
/// damaged to read, is an empty list: most pages carry no code, and an
/// unreadable one is a page to route by hand.
pub fn barcodes(luma: Vec<u8>, width: u32, height: u32) -> Vec<Barcode> {
    let mut hints = DecodeHints {
        TryHarder: Some(true),
        PossibleFormats: Some(FORMATS.iter().copied().collect::<HashSet<_>>()),
        ..DecodeHints::default()
    };
    let Ok(results) =
        rxing::helpers::detect_multiple_in_luma_with_hints(luma, width, height, &mut hints)
    else {
        return Vec::new();
    };
    let mut seen = BTreeSet::new();
    results
        .into_iter()
        .filter_map(|result| {
            let text = result.getText().to_owned();
            let format = format_name(*result.getBarcodeFormat());
            (!text.is_empty() && seen.insert((format.clone(), text.clone())))
                .then_some(Barcode { format, text })
        })
        .collect()
}

fn format_name(format: BarcodeFormat) -> String {
    format!("{format:?}").to_ascii_lowercase()
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn a_header_without_parameters_measures_as_the_server_always_has() {
        assert_eq!(
            ink_params(&InspectCaptureHeader::default()),
            InkParams::SERVER
        );
        let custom = ink_params(&InspectCaptureHeader {
            ink_threshold: 128,
            ink_margin: 0.1,
            ink_stride: 4,
            ..InspectCaptureHeader::default()
        });
        assert_eq!((custom.threshold, custom.stride), (128, 4));
        assert!((custom.margin_fraction - 0.1).abs() < f32::EPSILON);
    }

    #[test]
    fn a_blank_page_has_no_codes() {
        assert!(barcodes(vec![255; 400 * 400], 400, 400).is_empty());
    }

    #[test]
    fn format_names_are_snake_case() {
        assert_eq!(format_name(BarcodeFormat::QR_CODE), "qr_code");
        assert_eq!(format_name(BarcodeFormat::CODE_128), "code_128");
    }
}
