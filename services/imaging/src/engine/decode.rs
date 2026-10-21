//! Decoding uploaded images: JPEG, PNG, WebP, TIFF, GIF and HEIC/HEIF.
//!
//! The format is read from the file's own signature, never from the
//! declared content type, and the image's dimensions are checked against
//! the pixel budget before a single pixel is decoded, so a header that
//! claims 65,000 × 65,000 costs nothing. JPEG decodes at a reduced IDCT
//! size and WebP with libwebp's scaling when the caller only needs a smaller
//! picture. EXIF orientation is applied, so a phone photo comes out upright,
//! and nothing of the source's metadata survives into what is produced.

use std::io::Cursor;

use super::error::{EngineError, EngineResult};
use super::heif::Heif;
use super::raster::Rgba;

/// What an image's signature says it is.
#[derive(Clone, Copy, Debug, PartialEq, Eq)]
pub enum ImageKind {
    Jpeg,
    Png,
    Webp,
    Tiff,
    Gif,
    Heif,
}

impl ImageKind {
    pub fn name(self) -> &'static str {
        match self {
            Self::Jpeg => "jpeg",
            Self::Png => "png",
            Self::Webp => "webp",
            Self::Tiff => "tiff",
            Self::Gif => "gif",
            Self::Heif => "heic",
        }
    }
}

/// Reads the signature.
pub fn sniff(bytes: &[u8]) -> Option<ImageKind> {
    match bytes {
        [0xFF, 0xD8, 0xFF, ..] => Some(ImageKind::Jpeg),
        [0x89, b'P', b'N', b'G', 0x0D, 0x0A, 0x1A, 0x0A, ..] => Some(ImageKind::Png),
        [
            b'R',
            b'I',
            b'F',
            b'F',
            _,
            _,
            _,
            _,
            b'W',
            b'E',
            b'B',
            b'P',
            ..,
        ] => Some(ImageKind::Webp),
        [b'I', b'I', 0x2A, 0x00, ..] | [b'M', b'M', 0x00, 0x2A, ..] => Some(ImageKind::Tiff),
        [b'G', b'I', b'F', b'8', b'7' | b'9', b'a', ..] => Some(ImageKind::Gif),
        [_, _, _, _, b'f', b't', b'y', b'p', brand @ ..] if is_heif_brand(brand) => {
            Some(ImageKind::Heif)
        }
        _ => None,
    }
}

fn is_heif_brand(rest: &[u8]) -> bool {
    let Some(brand) = rest.get(..4) else {
        return false;
    };
    matches!(
        brand,
        b"heic" | b"heix" | b"hevc" | b"hevx" | b"heim" | b"heis" | b"mif1" | b"msf1" | b"avif"
    )
}

/// Whether the bytes are a PDF.
pub fn is_pdf(bytes: &[u8]) -> bool {
    let head = &bytes[..bytes.len().min(1024)];
    head.windows(5).any(|window| window == b"%PDF-")
}

/// A decoded image, upright.
#[derive(Debug)]
pub struct Decoded {
    pub image: Rgba,
    pub kind: ImageKind,
    /// The EXIF orientation applied (1 to 8), or 0 when there was none.
    pub orientation: u8,
    /// The size as stored, before orientation and any reduced decoding.
    pub source_width: u32,
    pub source_height: u32,
}

/// How an image is decoded.
#[derive(Clone, Copy, Debug)]
pub struct DecodeOptions {
    /// Refuse images with more stored pixels than this.
    pub max_pixels: u64,
    /// The caller needs at most this size (0 for full size); codecs that can
    /// decode smaller do, never below it.
    pub want_width: u32,
    pub want_height: u32,
}

/// Decodes `bytes`.
pub fn decode(bytes: &[u8], options: DecodeOptions, heif: Option<&Heif>) -> EngineResult<Decoded> {
    let kind = sniff(bytes)
        .ok_or_else(|| EngineError::unsupported("the file is not an image this worker reads"))?;
    match kind {
        ImageKind::Jpeg => decode_jpeg(bytes, options),
        ImageKind::Webp => decode_webp(bytes, options),
        ImageKind::Heif => {
            let heif = heif.ok_or_else(|| {
                EngineError::unsupported("HEIC and HEIF are not available on this worker")
            })?;
            let (image, width, height) = heif.decode(bytes, options.max_pixels)?;
            Ok(Decoded {
                image,
                kind,
                orientation: 0,
                source_width: width,
                source_height: height,
            })
        }
        ImageKind::Png | ImageKind::Tiff | ImageKind::Gif => decode_generic(bytes, kind, options),
    }
}

fn check_budget(width: u32, height: u32, max_pixels: u64) -> EngineResult<()> {
    if width == 0 || height == 0 {
        return Err(EngineError::invalid("the image has no pixels"));
    }
    let pixels = u64::from(width) * u64::from(height);
    if max_pixels > 0 && pixels > max_pixels {
        return Err(EngineError::pixels(format!(
            "a {width} × {height} image is over the {max_pixels}-pixel budget"
        )));
    }
    Ok(())
}

fn decode_jpeg(bytes: &[u8], options: DecodeOptions) -> EngineResult<Decoded> {
    let mut decoder = jpeg_decoder::Decoder::new(Cursor::new(bytes));
    decoder
        .read_info()
        .map_err(|err| EngineError::invalid(format!("the JPEG could not be read: {err}")))?;
    let info = decoder
        .info()
        .ok_or_else(|| EngineError::invalid("the JPEG has no frame"))?;
    let (source_width, source_height) = (u32::from(info.width), u32::from(info.height));
    check_budget(source_width, source_height, options.max_pixels)?;
    let budget = usize::try_from(options.max_pixels.saturating_mul(4)).unwrap_or(usize::MAX);
    decoder.set_max_decoding_buffer_size(budget);
    let orientation = decoder.exif_data().and_then(exif_orientation).unwrap_or(0);

    if options.want_width > 0 || options.want_height > 0 {
        let (want_w, want_h) = oriented_want(options, orientation);
        let request_w = u16::try_from(want_w.max(1)).unwrap_or(u16::MAX);
        let request_h = u16::try_from(want_h.max(1)).unwrap_or(u16::MAX);
        decoder
            .scale(request_w, request_h)
            .map_err(|err| EngineError::invalid(format!("the JPEG could not be scaled: {err}")))?;
    }
    let pixels = decoder
        .decode()
        .map_err(|err| EngineError::invalid(format!("the JPEG could not be decoded: {err}")))?;
    let info = decoder
        .info()
        .ok_or_else(|| EngineError::invalid("the JPEG has no frame"))?;
    let (width, height) = (u32::from(info.width), u32::from(info.height));
    let rgba = match info.pixel_format {
        jpeg_decoder::PixelFormat::L8 => pixels.iter().flat_map(|&v| [v, v, v, 255]).collect(),
        jpeg_decoder::PixelFormat::L16 => pixels
            .chunks_exact(2)
            .flat_map(|p| [p[0], p[0], p[0], 255])
            .collect(),
        jpeg_decoder::PixelFormat::RGB24 => pixels
            .chunks_exact(3)
            .flat_map(|p| [p[0], p[1], p[2], 255])
            .collect(),
        jpeg_decoder::PixelFormat::CMYK32 => {
            pixels.chunks_exact(4).flat_map(cmyk_to_rgba).collect()
        }
    };
    let image = orient(Rgba::new(width, height, rgba)?, orientation);
    Ok(Decoded {
        image,
        kind: ImageKind::Jpeg,
        orientation,
        source_width,
        source_height,
    })
}

/// The size wanted, in the stored image's axes.
fn oriented_want(options: DecodeOptions, orientation: u8) -> (u32, u32) {
    if (5..=8).contains(&orientation) {
        (options.want_height, options.want_width)
    } else {
        (options.want_width, options.want_height)
    }
}

/// jpeg-decoder hands back Adobe-inverted CMYK as most encoders write it.
fn cmyk_to_rgba(p: &[u8]) -> [u8; 4] {
    let k = u32::from(p[3]);
    let channel = |c: u8| u8::try_from(u32::from(c) * k / 255).unwrap_or(u8::MAX);
    [channel(p[0]), channel(p[1]), channel(p[2]), 255]
}

fn decode_webp(bytes: &[u8], options: DecodeOptions) -> EngineResult<Decoded> {
    use libwebp_sys::{
        VP8StatusCode, WEBP_CSP_MODE, WebPDecode, WebPDecoderConfig, WebPFreeDecBuffer,
        WebPGetFeatures, WebPInitDecoderConfig,
    };

    let mut config = std::mem::MaybeUninit::<WebPDecoderConfig>::uninit();
    // SAFETY: `WebPInitDecoderConfig` fully initialises the struct it is given
    // and reports whether the library version matches the header.
    if !unsafe { WebPInitDecoderConfig(config.as_mut_ptr()) } {
        return Err(EngineError::internal(
            "libwebp's decoder could not be initialised",
        ));
    }
    // SAFETY: initialised by the call above.
    let mut config = unsafe { config.assume_init() };
    // SAFETY: `bytes` is a valid slice for the call; libwebp writes only into
    // `config.input`.
    let status = unsafe { WebPGetFeatures(bytes.as_ptr(), bytes.len(), &raw mut config.input) };
    if status != VP8StatusCode::VP8_STATUS_OK {
        return Err(EngineError::invalid("the WebP image could not be read"));
    }
    let source_width = u32::try_from(config.input.width).unwrap_or(0);
    let source_height = u32::try_from(config.input.height).unwrap_or(0);
    check_budget(source_width, source_height, options.max_pixels)?;

    let (mut width, mut height) = (source_width, source_height);
    if options.want_width > 0 || options.want_height > 0 {
        (width, height) = super::raster::fit_within(
            source_width,
            source_height,
            options.want_width,
            options.want_height,
        );
        if (width, height) != (source_width, source_height) {
            config.options.use_scaling = 1;
            config.options.scaled_width = i32::try_from(width).unwrap_or(i32::MAX);
            config.options.scaled_height = i32::try_from(height).unwrap_or(i32::MAX);
        }
    }
    config.output.colorspace = WEBP_CSP_MODE::MODE_RGBA;
    // SAFETY: `config` was initialised and filled in above; libwebp allocates
    // the output buffer it describes in `config.output`.
    let status = unsafe { WebPDecode(bytes.as_ptr(), bytes.len(), &raw mut config) };
    if status != VP8StatusCode::VP8_STATUS_OK {
        // SAFETY: freeing a decoder buffer libwebp may have partly allocated.
        unsafe { WebPFreeDecBuffer(&raw mut config.output) };
        return Err(EngineError::invalid("the WebP image could not be decoded"));
    }
    // SAFETY: after a successful RGBA decode, `u.RGBA` describes a buffer of
    // `size` bytes owned by libwebp until `WebPFreeDecBuffer`.
    let data = unsafe {
        let rgba = config.output.u.RGBA;
        std::slice::from_raw_parts(rgba.rgba, rgba.size).to_vec()
    };
    // SAFETY: the buffer was allocated by `WebPDecode` and is freed once.
    unsafe { WebPFreeDecBuffer(&raw mut config.output) };
    let mut image = Rgba::new(width, height, data)?;
    image.flatten_on_white();
    Ok(Decoded {
        image,
        kind: ImageKind::Webp,
        orientation: 0,
        source_width,
        source_height,
    })
}

fn decode_generic(bytes: &[u8], kind: ImageKind, options: DecodeOptions) -> EngineResult<Decoded> {
    use image::{ImageDecoder, ImageReader, Limits};

    let mut reader = ImageReader::new(Cursor::new(bytes))
        .with_guessed_format()
        .map_err(|err| EngineError::invalid(format!("the image could not be read: {err}")))?;
    let mut limits = Limits::default();
    let side = u32::try_from(options.max_pixels.min(u64::from(u32::MAX))).unwrap_or(u32::MAX);
    limits.max_image_width = Some(side.max(1));
    limits.max_image_height = Some(side.max(1));
    limits.max_alloc = Some(options.max_pixels.saturating_mul(8).max(1 << 20));
    reader.limits(limits);
    let mut decoder = reader
        .into_decoder()
        .map_err(|err| EngineError::invalid(format!("the image could not be read: {err}")))?;
    let (source_width, source_height) = decoder.dimensions();
    check_budget(source_width, source_height, options.max_pixels)?;
    let orientation = decoder.orientation().map_or(0, orientation_code);
    let pixels = image::DynamicImage::from_decoder(decoder).map_err(|err| {
        if matches!(err, image::ImageError::Limits(_)) {
            EngineError::pixels(format!("the image is over the budget: {err}"))
        } else {
            EngineError::invalid(format!("the image could not be decoded: {err}"))
        }
    })?;
    let rgba = pixels.to_rgba8();
    let (width, height) = rgba.dimensions();
    let mut image = Rgba::new(width, height, rgba.into_raw())?;
    image.flatten_on_white();
    Ok(Decoded {
        image: orient(image, orientation),
        kind,
        orientation,
        source_width,
        source_height,
    })
}

fn orientation_code(orientation: image::metadata::Orientation) -> u8 {
    use image::metadata::Orientation;
    match orientation {
        Orientation::NoTransforms => 1,
        Orientation::FlipHorizontal => 2,
        Orientation::Rotate180 => 3,
        Orientation::FlipVertical => 4,
        Orientation::Rotate90FlipH => 5,
        Orientation::Rotate90 => 6,
        Orientation::Rotate270FlipH => 7,
        Orientation::Rotate270 => 8,
    }
}

/// Reads the orientation tag (0x0112) from an EXIF block (the TIFF structure
/// after the `Exif\0\0` marker), or `None` if there is none or the block is
/// malformed.
pub fn exif_orientation(exif: &[u8]) -> Option<u8> {
    let tiff = exif.strip_prefix(b"Exif\0\0").unwrap_or(exif);
    let little = match tiff.get(..4)? {
        [b'I', b'I', 0x2A, 0x00] => true,
        [b'M', b'M', 0x00, 0x2A] => false,
        _ => return None,
    };
    let u16_at = |at: usize| -> Option<u16> {
        let b = tiff.get(at..at + 2)?;
        Some(if little {
            u16::from_le_bytes([b[0], b[1]])
        } else {
            u16::from_be_bytes([b[0], b[1]])
        })
    };
    let u32_at = |at: usize| -> Option<u32> {
        let b = tiff.get(at..at + 4)?;
        let array = [b[0], b[1], b[2], b[3]];
        Some(if little {
            u32::from_le_bytes(array)
        } else {
            u32::from_be_bytes(array)
        })
    };
    let ifd = usize::try_from(u32_at(4)?).ok()?;
    let entries = usize::from(u16_at(ifd)?);
    for i in 0..entries.min(512) {
        let entry = ifd.checked_add(2 + i * 12)?;
        if u16_at(entry)? == 0x0112 {
            let value = u16_at(entry + 8)?;
            return u8::try_from(value).ok().filter(|v| (1..=8).contains(v));
        }
    }
    None
}

/// Turns an image upright for its EXIF orientation.
pub fn orient(image: Rgba, orientation: u8) -> Rgba {
    if orientation <= 1 || orientation > 8 {
        return image;
    }
    let (w, h) = (image.width as usize, image.height as usize);
    let swap = orientation >= 5;
    let (out_w, out_h) = if swap { (h, w) } else { (w, h) };
    let mut data = vec![0u8; image.data.len()];
    for y in 0..h {
        for x in 0..w {
            let (tx, ty) = match orientation {
                2 => (w - 1 - x, y),
                3 => (w - 1 - x, h - 1 - y),
                4 => (x, h - 1 - y),
                5 => (y, x),
                6 => (h - 1 - y, x),
                7 => (h - 1 - y, w - 1 - x),
                _ => (y, w - 1 - x),
            };
            let from = (y * w + x) * 4;
            let to = (ty * out_w + tx) * 4;
            data[to..to + 4].copy_from_slice(&image.data[from..from + 4]);
        }
    }
    Rgba {
        width: u32::try_from(out_w).unwrap_or(u32::MAX),
        height: u32::try_from(out_h).unwrap_or(u32::MAX),
        data,
    }
}

#[cfg(test)]
mod tests {
    use proptest::prelude::*;

    use super::*;

    fn options() -> DecodeOptions {
        DecodeOptions {
            max_pixels: 64_000_000,
            want_width: 0,
            want_height: 0,
        }
    }

    fn jpeg(width: u32, height: u32) -> Vec<u8> {
        let gray = vec![200u8; (width * height) as usize];
        super::super::raster::encode_jpeg_gray(&gray, width, height, 80).expect("jpeg")
    }

    #[test]
    fn signatures_decide_the_format() {
        assert_eq!(sniff(&jpeg(8, 8)), Some(ImageKind::Jpeg));
        assert_eq!(sniff(b"\x89PNG\r\n\x1a\n...."), Some(ImageKind::Png));
        assert_eq!(sniff(b"RIFF\0\0\0\0WEBPVP8 "), Some(ImageKind::Webp));
        assert_eq!(sniff(b"\0\0\0\x18ftypheic\0\0"), Some(ImageKind::Heif));
        assert_eq!(sniff(b"%PDF-1.7"), None);
        assert!(is_pdf(b"%PDF-1.7\n"));
    }

    #[test]
    fn a_jpeg_decodes_at_a_reduced_size_when_asked() {
        let bytes = jpeg(800, 600);
        let full = decode(&bytes, options(), None).expect("full");
        assert_eq!((full.image.width, full.image.height), (800, 600));
        let small = decode(
            &bytes,
            DecodeOptions {
                want_width: 200,
                want_height: 200,
                ..options()
            },
            None,
        )
        .expect("scaled");
        assert!(
            small.image.width >= 200 && small.image.width < 800,
            "{}",
            small.image.width
        );
    }

    #[test]
    fn a_header_claiming_too_many_pixels_is_refused_before_decoding() {
        let mut bytes = jpeg(64, 64);
        let sof = bytes
            .windows(2)
            .position(|w| w == [0xFF, 0xC0])
            .expect("sof");
        bytes[sof + 5..sof + 9].copy_from_slice(&[0xFD, 0xE8, 0xFD, 0xE8]);
        let err = decode(&bytes, options(), None).expect_err("refused");
        assert_eq!(
            err.reason,
            trenova_proto::imaging::v1::ErrorReason::PixelBudget
        );
    }

    #[test]
    fn orientation_six_turns_the_image_a_quarter_clockwise() {
        let image = Rgba::new(2, 1, vec![1, 1, 1, 255, 2, 2, 2, 255]).expect("rgba");
        let upright = orient(image, 6);
        assert_eq!((upright.width, upright.height), (1, 2));
        assert_eq!(upright.data[0], 1);
        assert_eq!(upright.data[4], 2);
    }

    #[test]
    fn the_orientation_tag_is_read_in_both_byte_orders() {
        let mut le = b"Exif\0\0II\x2A\0\x08\0\0\0\x01\0".to_vec();
        le.extend_from_slice(&[0x12, 0x01, 3, 0, 1, 0, 0, 0, 6, 0, 0, 0]);
        assert_eq!(exif_orientation(&le), Some(6));
        let mut be = b"MM\0\x2A\0\0\0\x08\0\x01".to_vec();
        be.extend_from_slice(&[0x01, 0x12, 0, 3, 0, 0, 0, 1, 0, 8, 0, 0]);
        assert_eq!(exif_orientation(&be), Some(8));
    }

    proptest! {
        #[test]
        fn malformed_exif_never_panics(bytes in proptest::collection::vec(any::<u8>(), 0..128)) {
            let _ = exif_orientation(&bytes);
        }

        #[test]
        fn arbitrary_bytes_decode_or_fail_cleanly(bytes in proptest::collection::vec(any::<u8>(), 0..512)) {
            let _ = decode(&bytes, options(), None);
        }
    }
}
