//! Uncompressed page images as scanners hand them over.

use crate::ImagingError;

/// The largest side the encoders take: JPEG stores dimensions in 16 bits,
/// and a letter page at 1200 DPI is 13 200 pixels, well inside it.
pub const MAX_SIDE: u32 = u16::MAX as u32;

/// How the bytes of each row are laid out.
#[derive(Clone, Copy, Debug, PartialEq, Eq)]
pub enum PixelFormat {
    /// One bit per pixel, most significant bit first. TWAIN calls the two
    /// meanings of a zero bit "chocolate" (zero is black) and "vanilla" (zero
    /// is white); both occur in the wild.
    Bilevel { zero_is_black: bool },
    /// One byte per pixel, zero black.
    Gray8,
    /// Red, green, blue; one byte each.
    Rgb8,
    /// Blue, green, red; one byte each, as a Windows DIB stores it.
    Bgr8,
}

impl PixelFormat {
    /// The bytes a row needs before any padding.
    pub fn row_bytes(self, width: u32) -> usize {
        let width = width as usize;
        match self {
            Self::Bilevel { .. } => width.div_ceil(8),
            Self::Gray8 => width,
            Self::Rgb8 | Self::Bgr8 => width * 3,
        }
    }
}

/// A borrowed page image.
#[derive(Clone, Copy, Debug)]
pub struct Raster<'a> {
    pub width: u32,
    pub height: u32,
    /// Bytes from the start of one row to the start of the next; at least
    /// [`PixelFormat::row_bytes`].
    pub stride: usize,
    pub format: PixelFormat,
    pub data: &'a [u8],
}

impl<'a> Raster<'a> {
    /// Checks the dimensions against the buffer, so nothing after this can
    /// index out of it.
    pub fn new(
        width: u32,
        height: u32,
        stride: usize,
        format: PixelFormat,
        data: &'a [u8],
    ) -> Result<Self, ImagingError> {
        if width == 0 || height == 0 || width > MAX_SIDE || height > MAX_SIDE {
            return Err(ImagingError::Dimensions { width, height });
        }
        let row_bytes = format.row_bytes(width);
        if stride < row_bytes {
            return Err(ImagingError::Stride { stride, row_bytes });
        }
        let needed = stride * (height as usize - 1) + row_bytes;
        if data.len() < needed {
            return Err(ImagingError::ShortBuffer {
                needed,
                actual: data.len(),
            });
        }
        Ok(Self {
            width,
            height,
            stride,
            format,
            data,
        })
    }

    /// Row `y`, without its padding.
    pub fn row(&self, y: u32) -> &'a [u8] {
        let start = self.stride * y as usize;
        &self.data[start..start + self.format.row_bytes(self.width)]
    }

    /// The rows packed end to end, borrowing when they already are.
    pub(crate) fn packed(&self) -> std::borrow::Cow<'a, [u8]> {
        let row_bytes = self.format.row_bytes(self.width);
        let total = row_bytes * self.height as usize;
        if self.stride == row_bytes {
            return std::borrow::Cow::Borrowed(&self.data[..total]);
        }
        let mut packed = Vec::with_capacity(total);
        for y in 0..self.height {
            packed.extend_from_slice(self.row(y));
        }
        std::borrow::Cow::Owned(packed)
    }
}

/// A page image that owns its bytes, as a decoded DIB or a binarized page is.
#[derive(Clone, Debug, PartialEq, Eq)]
pub struct OwnedRaster {
    pub width: u32,
    pub height: u32,
    pub stride: usize,
    pub format: PixelFormat,
    pub data: Vec<u8>,
}

impl OwnedRaster {
    pub fn as_raster(&self) -> Result<Raster<'_>, ImagingError> {
        Raster::new(
            self.width,
            self.height,
            self.stride,
            self.format,
            &self.data,
        )
    }
}

/// The luminance a pixel reads as, 0 black to 255 white (ITU-R BT.601).
fn luma(r: u8, g: u8, b: u8) -> u8 {
    let weighted = 299 * u32::from(r) + 587 * u32::from(g) + 114 * u32::from(b);
    u8::try_from((weighted + 500) / 1000).unwrap_or(u8::MAX)
}

/// Reduces a page to black and white at a fixed midpoint, for a source that
/// cannot scan bilevel itself when the profile asks for it. The result is
/// packed chocolate (zero is black), which is what the G4 encoder reads.
pub fn binarize(raster: &Raster<'_>) -> OwnedRaster {
    const THRESHOLD: u8 = 128;

    let row_bytes = PixelFormat::Bilevel {
        zero_is_black: true,
    }
    .row_bytes(raster.width);
    let mut data = vec![0u8; row_bytes * raster.height as usize];
    for y in 0..raster.height {
        let source = raster.row(y);
        let target = &mut data[row_bytes * y as usize..row_bytes * (y as usize + 1)];
        for x in 0..raster.width as usize {
            let white = match raster.format {
                PixelFormat::Bilevel { zero_is_black } => {
                    let bit = source[x / 8] >> (7 - (x % 8)) & 1;
                    (bit == 1) == zero_is_black
                }
                PixelFormat::Gray8 => source[x] >= THRESHOLD,
                PixelFormat::Rgb8 => {
                    luma(source[x * 3], source[x * 3 + 1], source[x * 3 + 2]) >= THRESHOLD
                }
                PixelFormat::Bgr8 => {
                    luma(source[x * 3 + 2], source[x * 3 + 1], source[x * 3]) >= THRESHOLD
                }
            };
            if white {
                target[x / 8] |= 0x80 >> (x % 8);
            }
        }
    }

    OwnedRaster {
        width: raster.width,
        height: raster.height,
        stride: row_bytes,
        format: PixelFormat::Bilevel {
            zero_is_black: true,
        },
        data,
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn a_raster_refuses_a_buffer_too_short_for_its_rows() {
        let data = [0u8; 10];
        assert!(matches!(
            Raster::new(4, 4, 3, PixelFormat::Gray8, &data),
            Err(ImagingError::Stride { .. })
        ));
        assert!(matches!(
            Raster::new(4, 4, 4, PixelFormat::Gray8, &data),
            Err(ImagingError::ShortBuffer {
                needed: 16,
                actual: 10
            })
        ));
        assert!(matches!(
            Raster::new(0, 4, 4, PixelFormat::Gray8, &data),
            Err(ImagingError::Dimensions { .. })
        ));
        let last_row_unpadded = [0u8; 19];
        assert!(Raster::new(3, 2, 10, PixelFormat::Rgb8, &last_row_unpadded).is_ok());
    }

    #[test]
    fn padded_rows_are_packed_and_packed_rows_borrowed() {
        let data = [1, 2, 0xEE, 3, 4, 0xEE];
        let raster = Raster::new(2, 2, 3, PixelFormat::Gray8, &data).expect("raster");
        assert_eq!(&*raster.packed(), &[1, 2, 3, 4]);
        let tight = [1, 2, 3, 4];
        let raster = Raster::new(2, 2, 2, PixelFormat::Gray8, &tight).expect("raster");
        assert!(matches!(raster.packed(), std::borrow::Cow::Borrowed(_)));
    }

    #[test]
    fn binarizing_keeps_dark_pixels_black_whatever_the_input() {
        let gray = [0, 200, 127, 0xEE, 128, 255, 10, 0xEE];
        let raster = Raster::new(3, 2, 4, PixelFormat::Gray8, &gray).expect("raster");
        let bw = binarize(&raster);
        assert_eq!(bw.data, vec![0b0100_0000, 0b1100_0000]);

        let bgr = [0, 0, 255, 255, 255, 255];
        let raster = Raster::new(2, 1, 6, PixelFormat::Bgr8, &bgr).expect("raster");
        assert_eq!(binarize(&raster).data, vec![0b0100_0000]);

        let vanilla = [0b1000_0000];
        let raster = Raster::new(
            2,
            1,
            1,
            PixelFormat::Bilevel {
                zero_is_black: false,
            },
            &vanilla,
        )
        .expect("raster");
        assert_eq!(binarize(&raster).data, vec![0b0100_0000]);
    }
}
