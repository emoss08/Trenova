//! Windows device-independent bitmaps, which is what WIA hands over.
//!
//! WIA 2.0 is asked for `WiaImgFmt_BMP`, the one format every WIA driver must
//! produce. The stream is a BMP file: a 14-byte file header, a
//! `BITMAPINFOHEADER` (or one of its longer successors), an optional colour
//! table, and the rows, bottom-up unless the height is negative. This reads
//! exactly the layouts scanners produce: 1, 4 and 8 bits through a palette,
//! and 24 or 32 bits direct, uncompressed. Anything else is refused rather
//! than guessed at.

use crate::ImagingError;
use crate::raster::{OwnedRaster, PixelFormat};

const FILE_HEADER_LEN: usize = 14;
const INFO_HEADER_MIN: usize = 40;
const BI_RGB: u32 = 0;
const BI_BITFIELDS: u32 = 3;
/// The biggest page a scanner makes: a legal page at 1200 DPI in colour is
/// about 240 MB, so this bounds allocation without refusing a real scan.
const MAX_PIXELS: u64 = 120_000_000;

/// A decoded bitmap and the resolution it says it was scanned at, if any.
#[derive(Debug, Clone, PartialEq, Eq)]
pub struct Bitmap {
    pub raster: OwnedRaster,
    pub dpi: Option<(u32, u32)>,
}

fn u16_at(bytes: &[u8], at: usize) -> Result<u16, ImagingError> {
    bytes
        .get(at..at + 2)
        .map(|b| u16::from_le_bytes([b[0], b[1]]))
        .ok_or(ImagingError::Dib("the header is cut short"))
}

fn u32_at(bytes: &[u8], at: usize) -> Result<u32, ImagingError> {
    bytes
        .get(at..at + 4)
        .map(|b| u32::from_le_bytes([b[0], b[1], b[2], b[3]]))
        .ok_or(ImagingError::Dib("the header is cut short"))
}

fn i32_at(bytes: &[u8], at: usize) -> Result<i32, ImagingError> {
    u32_at(bytes, at).map(|v| i32::from_le_bytes(v.to_le_bytes()))
}

/// Pixels per metre to dots per inch, dropping values no scanner produces.
fn dpi_from_ppm(ppm: i32) -> Option<u32> {
    let ppm = u32::try_from(ppm).ok().filter(|&p| p > 0)?;
    let dpi = (u64::from(ppm) * 254 + 5000) / 10000;
    u32::try_from(dpi).ok().filter(|d| (50..=1200).contains(d))
}

/// Decodes a BMP file, or a bare DIB starting at its info header.
#[allow(clippy::too_many_lines)]
pub fn decode_bmp(bytes: &[u8]) -> Result<Bitmap, ImagingError> {
    let (info_at, pixels_at_from_header) = if bytes.starts_with(b"BM") {
        let offset = u32_at(bytes, 10)? as usize;
        (FILE_HEADER_LEN, Some(offset))
    } else {
        (0, None)
    };

    let header_len = u32_at(bytes, info_at)? as usize;
    if header_len < INFO_HEADER_MIN {
        return Err(ImagingError::Dib(
            "the info header is older than BITMAPINFOHEADER",
        ));
    }
    let width = i32_at(bytes, info_at + 4)?;
    let raw_height = i32_at(bytes, info_at + 8)?;
    let bits = u16_at(bytes, info_at + 14)?;
    let compression = u32_at(bytes, info_at + 16)?;
    let x_ppm = i32_at(bytes, info_at + 24)?;
    let y_ppm = i32_at(bytes, info_at + 28)?;
    let colors_used = u32_at(bytes, info_at + 32)?;

    let width = u32::try_from(width)
        .ok()
        .filter(|&w| w > 0)
        .ok_or(ImagingError::Dib("the width is not positive"))?;
    let top_down = raw_height < 0;
    let height = raw_height.unsigned_abs();
    if height == 0 {
        return Err(ImagingError::Dib("the height is zero"));
    }
    if u64::from(width) * u64::from(height) > MAX_PIXELS {
        return Err(ImagingError::Dimensions { width, height });
    }

    let direct = match (bits, compression) {
        (1 | 4 | 8, BI_RGB) => false,
        (24, BI_RGB) | (32, BI_RGB | BI_BITFIELDS) => true,
        _ => {
            return Err(ImagingError::Dib(
                "only uncompressed 1, 4, 8, 24 and 32 bit bitmaps are read",
            ));
        }
    };
    if bits == 32 && compression == BI_BITFIELDS {
        let masks = (
            u32_at(bytes, info_at + 40)?,
            u32_at(bytes, info_at + 44)?,
            u32_at(bytes, info_at + 48)?,
        );
        if masks != (0x00FF_0000, 0x0000_FF00, 0x0000_00FF) {
            return Err(ImagingError::Dib(
                "32-bit bitfields other than BGRX are not read",
            ));
        }
    }

    let masks_len = if header_len == INFO_HEADER_MIN && compression == BI_BITFIELDS {
        12
    } else {
        0
    };
    let palette_at = info_at + header_len + masks_len;
    let palette_len = if direct {
        0
    } else if colors_used == 0 {
        1usize << bits
    } else {
        (colors_used as usize).min(1 << bits)
    };
    let palette: Vec<[u8; 3]> = (0..palette_len)
        .map(|i| {
            bytes
                .get(palette_at + i * 4..palette_at + i * 4 + 3)
                .map(|bgr| [bgr[2], bgr[1], bgr[0]])
                .ok_or(ImagingError::Dib("the colour table is cut short"))
        })
        .collect::<Result<_, _>>()?;

    let pixels_at = pixels_at_from_header.unwrap_or(palette_at + palette_len * 4);
    let source_stride = ((width as usize * bits as usize).div_ceil(32)) * 4;
    let needed = source_stride
        .checked_mul(height as usize)
        .and_then(|len| len.checked_add(pixels_at))
        .ok_or(ImagingError::Dib("the bitmap is too large"))?;
    if bytes.len() < needed {
        return Err(ImagingError::ShortBuffer {
            needed,
            actual: bytes.len(),
        });
    }
    let source_row = |y: u32| {
        let stored = if top_down { y } else { height - 1 - y };
        let start = pixels_at + source_stride * stored as usize;
        &bytes[start..start + source_stride]
    };

    let gray_palette = palette.iter().all(|[r, g, b]| r == g && g == b);
    let raster = if bits == 1 && gray_palette && palette.len() == 2 {
        let zero_is_black = palette[0][0] < palette[1][0];
        let format = PixelFormat::Bilevel { zero_is_black };
        let stride = format.row_bytes(width);
        let mut data = Vec::with_capacity(stride * height as usize);
        for y in 0..height {
            data.extend_from_slice(&source_row(y)[..stride]);
        }
        OwnedRaster {
            width,
            height,
            stride,
            format,
            data,
        }
    } else if direct {
        let stride = PixelFormat::Bgr8.row_bytes(width);
        let mut data = Vec::with_capacity(stride * height as usize);
        let step = usize::from(bits / 8);
        for y in 0..height {
            let row = source_row(y);
            for x in 0..width as usize {
                data.extend_from_slice(&row[x * step..x * step + 3]);
            }
        }
        OwnedRaster {
            width,
            height,
            stride,
            format: PixelFormat::Bgr8,
            data,
        }
    } else {
        let format = if gray_palette {
            PixelFormat::Gray8
        } else {
            PixelFormat::Rgb8
        };
        let stride = format.row_bytes(width);
        let mut data = Vec::with_capacity(stride * height as usize);
        let per_byte = 8 / usize::from(bits);
        let mask = u8::try_from((1u16 << bits) - 1).unwrap_or(u8::MAX);
        for y in 0..height {
            let row = source_row(y);
            for x in 0..width as usize {
                let byte = row[x / per_byte];
                let shift = 8 - usize::from(bits) * (x % per_byte + 1);
                let index = usize::from(byte >> shift & mask);
                let color = palette.get(index).copied().unwrap_or([0, 0, 0]);
                if gray_palette {
                    data.push(color[0]);
                } else {
                    data.extend_from_slice(&color);
                }
            }
        }
        OwnedRaster {
            width,
            height,
            stride,
            format,
            data,
        }
    };

    let dpi = dpi_from_ppm(x_ppm).zip(dpi_from_ppm(y_ppm));
    Ok(Bitmap { raster, dpi })
}

#[cfg(test)]
mod tests {
    use super::*;

    /// Builds a BMP file the way Windows writes one.
    fn bmp(width: i32, height: i32, bits: u16, palette: &[[u8; 3]], rows: &[Vec<u8>]) -> Vec<u8> {
        let stride = ((width.unsigned_abs() as usize * bits as usize).div_ceil(32)) * 4;
        let pixels_at = 14 + 40 + palette.len() * 4;
        let mut out = Vec::new();
        out.extend_from_slice(b"BM");
        out.extend_from_slice(
            &u32::try_from(pixels_at + stride * rows.len())
                .expect("len")
                .to_le_bytes(),
        );
        out.extend_from_slice(&[0; 4]);
        out.extend_from_slice(&u32::try_from(pixels_at).expect("offset").to_le_bytes());
        out.extend_from_slice(&40u32.to_le_bytes());
        out.extend_from_slice(&width.to_le_bytes());
        out.extend_from_slice(&height.to_le_bytes());
        out.extend_from_slice(&1u16.to_le_bytes());
        out.extend_from_slice(&bits.to_le_bytes());
        out.extend_from_slice(&0u32.to_le_bytes());
        out.extend_from_slice(&0u32.to_le_bytes());
        out.extend_from_slice(&11811i32.to_le_bytes());
        out.extend_from_slice(&11811i32.to_le_bytes());
        out.extend_from_slice(&u32::try_from(palette.len()).expect("colours").to_le_bytes());
        out.extend_from_slice(&0u32.to_le_bytes());
        for [r, g, b] in palette {
            out.extend_from_slice(&[*b, *g, *r, 0]);
        }
        for row in rows {
            let mut padded = row.clone();
            padded.resize(stride, 0);
            out.extend_from_slice(&padded);
        }
        out
    }

    #[test]
    fn a_bottom_up_bilevel_bitmap_keeps_its_palette_meaning() {
        let file = bmp(
            3,
            2,
            1,
            &[[255, 255, 255], [0, 0, 0]],
            &[vec![0b1000_0000], vec![0b0100_0000]],
        );
        let bitmap = decode_bmp(&file).expect("bitmap");
        assert_eq!(bitmap.dpi, Some((300, 300)));
        assert_eq!(
            bitmap.raster.format,
            PixelFormat::Bilevel {
                zero_is_black: false
            }
        );
        assert_eq!(bitmap.raster.data, vec![0b0100_0000, 0b1000_0000]);
    }

    #[test]
    fn a_top_down_24_bit_bitmap_reads_in_order() {
        let rows = vec![vec![1, 2, 3, 4, 5, 6], vec![7, 8, 9, 10, 11, 12]];
        let bitmap = decode_bmp(&bmp(2, -2, 24, &[], &rows)).expect("bitmap");
        assert_eq!(bitmap.raster.format, PixelFormat::Bgr8);
        assert_eq!(
            bitmap.raster.data,
            vec![1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12]
        );
    }

    #[test]
    fn an_8_bit_gray_palette_becomes_gray_and_a_colour_one_rgb() {
        let gray: Vec<[u8; 3]> = (0..=255u8).map(|v| [v, v, v]).collect();
        let bitmap = decode_bmp(&bmp(2, 1, 8, &gray, &[vec![10, 250]])).expect("bitmap");
        assert_eq!(bitmap.raster.format, PixelFormat::Gray8);
        assert_eq!(bitmap.raster.data, vec![10, 250]);

        let colour = [[255, 0, 0], [0, 0, 255]];
        let bitmap = decode_bmp(&bmp(3, 1, 4, &colour, &[vec![0x01, 0x00]])).expect("bitmap");
        assert_eq!(bitmap.raster.format, PixelFormat::Rgb8);
        assert_eq!(bitmap.raster.data, vec![255, 0, 0, 0, 0, 255, 255, 0, 0]);
    }

    #[test]
    fn a_truncated_or_compressed_bitmap_is_refused() {
        let file = bmp(4, 4, 24, &[], &[vec![0; 12]]);
        assert!(matches!(
            decode_bmp(&file),
            Err(ImagingError::ShortBuffer { .. })
        ));

        let mut rle = bmp(2, 1, 8, &[[0, 0, 0], [1, 1, 1]], &[vec![0, 1]]);
        rle[14 + 16] = 1;
        assert!(matches!(decode_bmp(&rle), Err(ImagingError::Dib(_))));

        assert!(matches!(decode_bmp(b"BM"), Err(ImagingError::Dib(_))));
    }
}
