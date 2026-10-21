//! NormalizeImage: any image the worker reads, as a plain upright PNG or
//! JPEG. Used where a browser or a PDF renderer needs a format it is sure to
//! take (the template asset inliner turns WebP and HEIC into PNG). The image
//! is decoded at a reduced size when its codec can, turned upright, fitted
//! into the box if one is given, and re-encoded, so none of the source's
//! metadata (EXIF, GPS, colour profiles, comments) survives.

use trenova_proto::imaging::v1::{
    ArtifactHeader, ImageFormat, NormalizeImageHeader, NormalizeImageResponse, NormalizeSummary,
    normalize_image_response::Part,
};

use super::decode::{DecodeOptions, decode};
use super::error::{EngineError, EngineResult};
use super::raster::{encode, fit_within};
use super::{Engine, Input, Kind, Sink, emit_artifact, kind_of};
use crate::protocol::JobLimits;

const DEFAULT_JPEG_QUALITY: u32 = 90;

pub fn normalize(
    engine: &Engine,
    header: &NormalizeImageHeader,
    input: Input<'_>,
    limits: JobLimits,
    sink: &mut dyn Sink,
) -> EngineResult<NormalizeSummary> {
    let format = match ImageFormat::try_from(header.format) {
        Ok(ImageFormat::Unspecified | ImageFormat::Png) | Err(_) => ImageFormat::Png,
        Ok(ImageFormat::Jpeg) => ImageFormat::Jpeg,
        Ok(ImageFormat::Webp) => {
            return Err(EngineError::request(
                "an image is normalized to PNG or JPEG",
            ));
        }
    };
    if kind_of(&input)? == Kind::Pdf {
        return Err(EngineError::unsupported(
            "a PDF is not an image; render it instead",
        ));
    }
    let decoded = decode(
        input.bytes,
        DecodeOptions {
            max_pixels: limits.max_image_pixels,
            want_width: header.max_width,
            want_height: header.max_height,
        },
        engine.heif(),
    )?;
    let (width, height) = fit_within(
        decoded.image.width,
        decoded.image.height,
        header.max_width,
        header.max_height,
    );
    let image = if (width, height) == (decoded.image.width, decoded.image.height) {
        decoded.image
    } else {
        decoded.image.resize(width, height)?
    };
    let quality = match header.quality {
        0 => DEFAULT_JPEG_QUALITY,
        q => q,
    };
    let encoded = encode(&image, format, quality, false)?;
    emit_artifact(
        sink,
        ArtifactHeader {
            index: 0,
            content_type: encoded.content_type.to_owned(),
            width_px: image.width,
            height_px: image.height,
            size: encoded.data.len() as u64,
        },
        &encoded.data,
        limits.chunk_bytes,
        |image| NormalizeImageResponse {
            part: Some(Part::Image(image)),
        },
        |chunk| NormalizeImageResponse {
            part: Some(Part::Chunk(chunk)),
        },
    )?;
    Ok(NormalizeSummary {
        source_format: decoded.kind.name().to_owned(),
        source_width: decoded.source_width,
        source_height: decoded.source_height,
        orientation_applied: u32::from(decoded.orientation),
    })
}
