//! Decoded pages in memory, and turning them into the files callers get.
//!
//! Every page the engine draws or decodes becomes an opaque RGBA buffer, one
//! contiguous allocation. Resizing is `fast_image_resize` (SIMD convolution,
//! Lanczos3 for reductions), WebP is libwebp, JPEG is the encoder the capture
//! companion uses (`trenova-imaging-core`), and PNG is the `png` encoder in
//! `image` at its fast setting.

use std::io::Cursor;

use fast_image_resize::images::{Image as ResizeImage, ImageRef};
use fast_image_resize::{FilterType, PixelType, ResizeAlg, ResizeOptions, Resizer};
use trenova_imaging_core::page::encode_jpeg;
use trenova_imaging_core::{PixelFormat, Raster, Resolution};
use trenova_proto::imaging::v1::ImageFormat;

use super::error::{EngineError, EngineResult};

/// An opaque RGBA image, rows packed.
#[derive(Clone, Debug, PartialEq, Eq)]
pub struct Rgba {
    pub width: u32,
    pub height: u32,
    pub data: Vec<u8>,
}

impl Rgba {
    pub fn new(width: u32, height: u32, data: Vec<u8>) -> EngineResult<Self> {
        let expected = width as usize * height as usize * 4;
        if width == 0 || height == 0 || data.len() != expected {
            return Err(EngineError::internal(format!(
                "a {width} × {height} RGBA image needs {expected} bytes, not {}",
                data.len()
            )));
        }
        Ok(Self {
            width,
            height,
            data,
        })
    }

    /// Composites every pixel onto white and sets it opaque.
    pub fn flatten_on_white(&mut self) {
        for pixel in self.data.chunks_exact_mut(4) {
            let alpha = u32::from(pixel[3]);
            if alpha == 255 {
                continue;
            }
            for channel in &mut pixel[..3] {
                let blended = (u32::from(*channel) * alpha + 255 * (255 - alpha) + 127) / 255;
                *channel = u8::try_from(blended).unwrap_or(u8::MAX);
            }
            pixel[3] = 255;
        }
    }

    /// The image as RGB, dropping alpha (call after flattening).
    pub fn to_rgb(&self) -> Vec<u8> {
        let mut out = Vec::with_capacity(self.width as usize * self.height as usize * 3);
        for pixel in self.data.chunks_exact(4) {
            out.extend_from_slice(&pixel[..3]);
        }
        out
    }

    /// One byte of luma a pixel (BT.601, as everything in Trenova measures it).
    pub fn to_luma(&self) -> Vec<u8> {
        self.data
            .chunks_exact(4)
            .map(|p| trenova_imaging_core::luma(p[0], p[1], p[2]))
            .collect()
    }

    /// The image resized to `width` × `height`, Lanczos3. Returns a copy when
    /// the size is unchanged.
    pub fn resize(&self, width: u32, height: u32) -> EngineResult<Self> {
        if width == self.width && height == self.height {
            return Ok(self.clone());
        }
        let source = ImageRef::new(self.width, self.height, &self.data, PixelType::U8x4)
            .map_err(|err| EngineError::internal(format!("resize source: {err}")))?;
        let mut target = ResizeImage::new(width.max(1), height.max(1), PixelType::U8x4);
        let options = ResizeOptions::new()
            .resize_alg(ResizeAlg::Convolution(FilterType::Lanczos3))
            .use_alpha(false);
        Resizer::new()
            .resize(&source, &mut target, &options)
            .map_err(|err| EngineError::internal(format!("resize: {err}")))?;
        Self::new(width.max(1), height.max(1), target.into_vec())
    }
}

/// The largest size that fits `width` × `height` inside `max_width` ×
/// `max_height` (0 for no bound) keeping the aspect ratio, never enlarging.
pub fn fit_within(width: u32, height: u32, max_width: u32, max_height: u32) -> (u32, u32) {
    let scale_w = if max_width == 0 {
        f64::INFINITY
    } else {
        f64::from(max_width) / f64::from(width)
    };
    let scale_h = if max_height == 0 {
        f64::INFINITY
    } else {
        f64::from(max_height) / f64::from(height)
    };
    let scale = scale_w.min(scale_h).min(1.0);
    if !scale.is_finite() {
        return (width, height);
    }
    (scaled(width, scale), scaled(height, scale))
}

#[allow(clippy::cast_possible_truncation, clippy::cast_sign_loss)]
fn scaled(length: u32, scale: f64) -> u32 {
    ((f64::from(length) * scale).round() as u32).max(1)
}

/// An encoded image.
#[derive(Debug)]
pub struct Encoded {
    pub data: Vec<u8>,
    pub content_type: &'static str,
}

/// Encodes an opaque image as `format`.
pub fn encode(
    image: &Rgba,
    format: ImageFormat,
    quality: u32,
    lossless: bool,
) -> EngineResult<Encoded> {
    match format {
        ImageFormat::Webp | ImageFormat::Unspecified => Ok(Encoded {
            data: encode_webp(image, quality, lossless)?,
            content_type: "image/webp",
        }),
        ImageFormat::Jpeg => Ok(Encoded {
            data: encode_jpeg_rgb(image, quality)?,
            content_type: "image/jpeg",
        }),
        ImageFormat::Png => Ok(Encoded {
            data: encode_png(image)?,
            content_type: "image/png",
        }),
    }
}

fn quality_or(quality: u32, default: u32) -> u32 {
    if quality == 0 {
        default
    } else {
        quality.min(100)
    }
}

fn encode_webp(image: &Rgba, quality: u32, lossless: bool) -> EngineResult<Vec<u8>> {
    let stride = i32::try_from(image.width * 4)
        .map_err(|_| EngineError::pixels("the image is too wide for WebP"))?;
    let width = i32::try_from(image.width)
        .map_err(|_| EngineError::pixels("the image is too wide for WebP"))?;
    let height = i32::try_from(image.height)
        .map_err(|_| EngineError::pixels("the image is too tall for WebP"))?;
    if image.width > 16_383 || image.height > 16_383 {
        return Err(EngineError::pixels(
            "WebP holds at most 16383 pixels a side",
        ));
    }
    let mut output: *mut u8 = std::ptr::null_mut();
    #[allow(clippy::cast_precision_loss)]
    let quality = quality_or(quality, 80) as f32;
    // SAFETY: `image.data` holds `height` rows of `stride` bytes (checked by
    // `Rgba::new`), libwebp only reads it, and on success writes a buffer it
    // allocated to `output` and returns its length.
    let size = unsafe {
        if lossless {
            libwebp_sys::WebPEncodeLosslessRGBA(
                image.data.as_ptr(),
                width,
                height,
                stride,
                &raw mut output,
            )
        } else {
            libwebp_sys::WebPEncodeRGBA(
                image.data.as_ptr(),
                width,
                height,
                stride,
                quality,
                &raw mut output,
            )
        }
    };
    if size == 0 || output.is_null() {
        return Err(EngineError::internal("WebP encoding failed"));
    }
    // SAFETY: libwebp returned `size` bytes at `output`, which stay valid until
    // `WebPFree`; they are copied out before it.
    let encoded = unsafe { std::slice::from_raw_parts(output, size) }.to_vec();
    // SAFETY: `output` was allocated by libwebp and is freed exactly once.
    unsafe { libwebp_sys::WebPFree(output.cast()) };
    Ok(encoded)
}

fn encode_jpeg_rgb(image: &Rgba, quality: u32) -> EngineResult<Vec<u8>> {
    let rgb = image.to_rgb();
    let raster = Raster::new(
        image.width,
        image.height,
        image.width as usize * 3,
        PixelFormat::Rgb8,
        &rgb,
    )?;
    let quality = u8::try_from(quality_or(quality, 82).clamp(30, 95)).unwrap_or(82);
    Ok(encode_jpeg(
        &raster,
        jpeg_encoder::ColorType::Rgb,
        quality,
        Resolution::square(96),
    )?)
}

/// JPEG of a gray image, for thumbnails of scanned pages.
pub fn encode_jpeg_gray(
    luma: &[u8],
    width: u32,
    height: u32,
    quality: u32,
) -> EngineResult<Vec<u8>> {
    let raster = Raster::new(width, height, width as usize, PixelFormat::Gray8, luma)?;
    let quality = u8::try_from(quality_or(quality, 82).clamp(30, 95)).unwrap_or(82);
    Ok(encode_jpeg(
        &raster,
        jpeg_encoder::ColorType::Luma,
        quality,
        Resolution::square(96),
    )?)
}

fn encode_png(image: &Rgba) -> EngineResult<Vec<u8>> {
    use image::ImageEncoder;
    use image::codecs::png::{CompressionType, FilterType as PngFilter, PngEncoder};

    let rgb = image.to_rgb();
    let mut out = Cursor::new(Vec::with_capacity(rgb.len() / 4));
    PngEncoder::new_with_quality(&mut out, CompressionType::Fast, PngFilter::Adaptive)
        .write_image(
            &rgb,
            image.width,
            image.height,
            image::ExtendedColorType::Rgb8,
        )
        .map_err(|err| EngineError::internal(format!("PNG encoding failed: {err}")))?;
    Ok(out.into_inner())
}

#[cfg(test)]
mod tests {
    use super::*;

    fn checker(width: u32, height: u32) -> Rgba {
        let mut data = Vec::with_capacity((width * height * 4) as usize);
        for y in 0..height {
            for x in 0..width {
                let v = if (x / 8 + y / 8) % 2 == 0 { 20 } else { 235 };
                data.extend_from_slice(&[v, v, v, 255]);
            }
        }
        Rgba::new(width, height, data).expect("rgba")
    }

    #[test]
    fn fitting_never_enlarges_and_keeps_the_aspect_ratio() {
        assert_eq!(fit_within(1275, 1650, 1400, 1800), (1275, 1650));
        assert_eq!(fit_within(2550, 3300, 1400, 1800), (1391, 1800));
        assert_eq!(fit_within(2550, 3300, 240, 0), (240, 311));
        assert_eq!(fit_within(100, 50, 0, 0), (100, 50));
    }

    #[test]
    fn transparent_pixels_flatten_onto_white() {
        let mut image = Rgba::new(2, 1, vec![0, 0, 0, 0, 0, 0, 0, 128]).expect("rgba");
        image.flatten_on_white();
        assert_eq!(&image.data[..4], &[255, 255, 255, 255]);
        assert_eq!(image.data[7], 255);
        assert!(image.data[4] > 100 && image.data[4] < 160);
    }

    #[test]
    fn every_format_encodes_a_decodable_image() {
        let image = checker(64, 48);
        for (format, mime) in [
            (ImageFormat::Webp, "image/webp"),
            (ImageFormat::Jpeg, "image/jpeg"),
            (ImageFormat::Png, "image/png"),
        ] {
            let encoded = encode(&image, format, 80, false).expect("encodes");
            assert_eq!(encoded.content_type, mime);
            let decoded = image::load_from_memory(&encoded.data).expect("decodes");
            assert_eq!((decoded.width(), decoded.height()), (64, 48));
        }
        let lossless = encode(&image, ImageFormat::Webp, 0, true).expect("lossless");
        let decoded = image::load_from_memory(&lossless.data)
            .expect("decodes")
            .to_rgba8();
        assert_eq!(decoded.as_raw(), &image.data);
    }

    #[test]
    fn resizing_a_page_keeps_its_ink() {
        let image = checker(400, 300);
        let small = image.resize(100, 75).expect("resizes");
        assert_eq!((small.width, small.height), (100, 75));
        let mean: u32 = small.to_luma().iter().map(|&v| u32::from(v)).sum::<u32>() / (100 * 75);
        assert!((100..160).contains(&mean), "mean luma {mean}");
    }
}
