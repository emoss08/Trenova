//! PWG Raster (PWG 5102.4), which Windows' IPP Class Driver sends when a
//! printer offers it, decoded a page at a time and turned into a PDF.
//!
//! A stream is the sync word `RaS2` followed by pages, each a 1796-byte
//! big-endian header and the page's lines, run-length compressed: a byte
//! repeating the line, then packets that either repeat one pixel or copy a
//! run of them. Everything in a header comes from whoever printed, so each
//! field is checked before it sizes anything.

use crate::ImagingError;
use crate::page::{PdfDocument, Resolution};
use crate::raster::{MAX_SIDE, OwnedRaster, PixelFormat};

const SYNC: &[u8; 4] = b"RaS2";
const HEADER_BYTES: usize = 1796;

const HW_RESOLUTION: usize = 276;
const WIDTH: usize = 372;
const HEIGHT: usize = 376;
const BITS_PER_COLOR: usize = 384;
const BITS_PER_PIXEL: usize = 388;
const BYTES_PER_LINE: usize = 392;
const COLOR_ORDER: usize = 396;
const COLOR_SPACE: usize = 400;
const NUM_COLORS: usize = 420;

const CHUNKY: u32 = 0;
const SPACE_RGB: u32 = 1;
const SPACE_BLACK: u32 = 3;
const SPACE_SGRAY: u32 = 18;
const SPACE_SRGB: u32 = 19;

/// The most raw bytes one page may decode to: a legal page in 8-bit colour at
/// 600 DPI is 129 MB, and nothing Windows sends is larger.
pub const MAX_PAGE_RASTER_BYTES: usize = 160 << 20;

#[derive(Debug, thiserror::Error, PartialEq, Eq)]
pub enum PwgError {
    #[error("the document is not PWG raster")]
    NotPwg,
    #[error("page {page} ends early")]
    Truncated { page: u32 },
    #[error("page {page}: {reason}")]
    Header { page: u32, reason: String },
    #[error("page {page}, line {line}: the compressed data overruns the line")]
    Overrun { page: u32, line: u32 },
    #[error("the document has more than {max} pages")]
    TooManyPages { max: u32 },
    #[error("the document has no pages")]
    Empty,
    #[error("the PDF would be larger than {max} bytes")]
    TooLarge { max: usize },
    #[error(transparent)]
    Imaging(#[from] ImagingError),
}

/// How a page's pixels are stored in the stream.
#[derive(Clone, Copy, Debug, PartialEq, Eq)]
enum Layout {
    /// One bit a pixel, where 1 is ink.
    Black1,
    /// One byte a pixel, where 255 is ink.
    Black8,
    /// One bit a pixel, where 1 is white.
    Gray1,
    Gray8,
    Rgb8,
}

impl Layout {
    fn of(space: u32, bits_per_color: u32, colors: u32) -> Option<Self> {
        match (space, bits_per_color, colors) {
            (SPACE_BLACK, 1, 1) => Some(Self::Black1),
            (SPACE_BLACK, 8, 1) => Some(Self::Black8),
            (SPACE_SGRAY, 1, 1) => Some(Self::Gray1),
            (SPACE_SGRAY, 8, 1) => Some(Self::Gray8),
            (SPACE_SRGB | SPACE_RGB, 8, 3) => Some(Self::Rgb8),
            _ => None,
        }
    }

    fn bits_per_pixel(self) -> u32 {
        match self {
            Self::Black1 | Self::Gray1 => 1,
            Self::Black8 | Self::Gray8 => 8,
            Self::Rgb8 => 24,
        }
    }

    /// The unit a compression packet counts in: a pixel, or a byte of eight
    /// pixels when they are smaller than a byte.
    fn unit(self) -> usize {
        match self {
            Self::Black1 | Self::Gray1 | Self::Black8 | Self::Gray8 => 1,
            Self::Rgb8 => 3,
        }
    }

    /// The byte that fills the rest of a line with white.
    fn white(self) -> u8 {
        match self {
            Self::Black1 | Self::Black8 => 0x00,
            Self::Gray1 | Self::Gray8 | Self::Rgb8 => 0xFF,
        }
    }

    /// The format the decoded page is handed on as. Black8 is inverted as it
    /// is decoded, so it reads as gray.
    fn format(self) -> PixelFormat {
        match self {
            Self::Black1 => PixelFormat::Bilevel {
                zero_is_black: false,
            },
            Self::Gray1 => PixelFormat::Bilevel {
                zero_is_black: true,
            },
            Self::Black8 | Self::Gray8 => PixelFormat::Gray8,
            Self::Rgb8 => PixelFormat::Rgb8,
        }
    }
}

/// One decoded page.
#[derive(Clone, Debug, PartialEq, Eq)]
pub struct PwgPage {
    pub raster: OwnedRaster,
    pub resolution: Resolution,
}

/// Reads a PWG raster stream a page at a time.
#[derive(Debug)]
pub struct PwgReader<'a> {
    data: &'a [u8],
    offset: usize,
    page: u32,
}

fn be_u32(header: &[u8], offset: usize) -> u32 {
    u32::from_be_bytes([
        header[offset],
        header[offset + 1],
        header[offset + 2],
        header[offset + 3],
    ])
}

impl<'a> PwgReader<'a> {
    pub fn new(data: &'a [u8]) -> Result<Self, PwgError> {
        if !data.starts_with(SYNC) {
            return Err(PwgError::NotPwg);
        }
        Ok(Self {
            data,
            offset: SYNC.len(),
            page: 0,
        })
    }

    fn header_error(&self, reason: impl Into<String>) -> PwgError {
        PwgError::Header {
            page: self.page,
            reason: reason.into(),
        }
    }

    /// The next page, or `None` at the end of the stream.
    pub fn next_page(&mut self) -> Result<Option<PwgPage>, PwgError> {
        if self.offset == self.data.len() {
            return Ok(None);
        }
        self.page += 1;
        let header = self
            .data
            .get(self.offset..self.offset + HEADER_BYTES)
            .ok_or(PwgError::Truncated { page: self.page })?;
        self.offset += HEADER_BYTES;

        let width = be_u32(header, WIDTH);
        let height = be_u32(header, HEIGHT);
        if width == 0 || height == 0 || width > MAX_SIDE || height > MAX_SIDE {
            return Err(self.header_error(format!("{width} × {height} is not a page")));
        }
        let resolution = Resolution {
            x: be_u32(header, HW_RESOLUTION),
            y: be_u32(header, HW_RESOLUTION + 4),
        };
        if be_u32(header, COLOR_ORDER) != CHUNKY {
            return Err(self.header_error("only chunky pixels are supported"));
        }
        let space = be_u32(header, COLOR_SPACE);
        let bits_per_color = be_u32(header, BITS_PER_COLOR);
        let colors = be_u32(header, NUM_COLORS);
        let layout = Layout::of(space, bits_per_color, colors).ok_or_else(|| {
            self.header_error(format!(
                "colour space {space} at {bits_per_color} bits and {colors} colours is not supported"
            ))
        })?;
        if be_u32(header, BITS_PER_PIXEL) != layout.bits_per_pixel() {
            return Err(self.header_error("bits per pixel disagrees with the colour space"));
        }
        let row_bytes = layout.format().row_bytes(width);
        if usize::try_from(be_u32(header, BYTES_PER_LINE)).ok() != Some(row_bytes) {
            return Err(self.header_error("bytes per line disagrees with the width"));
        }
        let total = row_bytes
            .checked_mul(height as usize)
            .filter(|&t| t <= MAX_PAGE_RASTER_BYTES)
            .ok_or_else(|| self.header_error("the page is too large"))?;

        let mut pixels = vec![0u8; total];
        self.decode_lines(layout, row_bytes, height, &mut pixels)?;
        if layout == Layout::Black8 {
            for byte in &mut pixels {
                *byte = !*byte;
            }
        }

        Ok(Some(PwgPage {
            raster: OwnedRaster {
                width,
                height,
                stride: row_bytes,
                format: layout.format(),
                data: pixels,
            },
            resolution,
        }))
    }

    fn byte(&mut self) -> Result<u8, PwgError> {
        let byte = *self
            .data
            .get(self.offset)
            .ok_or(PwgError::Truncated { page: self.page })?;
        self.offset += 1;
        Ok(byte)
    }

    fn take(&mut self, len: usize) -> Result<&'a [u8], PwgError> {
        let bytes = self
            .data
            .get(self.offset..self.offset + len)
            .ok_or(PwgError::Truncated { page: self.page })?;
        self.offset += len;
        Ok(bytes)
    }

    fn decode_lines(
        &mut self,
        layout: Layout,
        row_bytes: usize,
        height: u32,
        pixels: &mut [u8],
    ) -> Result<(), PwgError> {
        let unit = layout.unit();
        let mut line = 0u32;
        while line < height {
            let repeat = u32::from(self.byte()?) + 1;
            if repeat > height - line {
                return Err(PwgError::Overrun {
                    page: self.page,
                    line,
                });
            }
            let start = line as usize * row_bytes;
            let row = &mut pixels[start..start + row_bytes];
            let mut filled = 0;
            while filled < row_bytes {
                let control = self.byte()?;
                let overrun = PwgError::Overrun {
                    page: self.page,
                    line,
                };
                match control {
                    0..=127 => {
                        let len = (usize::from(control) + 1) * unit;
                        let pixel = self.take(unit)?;
                        let run = row.get_mut(filled..filled + len).ok_or(overrun)?;
                        for chunk in run.chunks_exact_mut(unit) {
                            chunk.copy_from_slice(pixel);
                        }
                        filled += len;
                    }
                    128 => {
                        row[filled..].fill(layout.white());
                        filled = row_bytes;
                    }
                    129..=255 => {
                        let len = (257 - usize::from(control)) * unit;
                        let literal = self.take(len)?;
                        row.get_mut(filled..filled + len)
                            .ok_or(overrun)?
                            .copy_from_slice(literal);
                        filled += len;
                    }
                }
            }
            for copy in 1..repeat {
                let target = start + copy as usize * row_bytes;
                pixels.copy_within(start..start + row_bytes, target);
            }
            line += repeat;
        }
        Ok(())
    }
}

/// Whether every pixel of a colour page is a shade of gray, as a black and
/// white document printed in colour mode is.
fn is_gray(rgb: &[u8]) -> bool {
    rgb.chunks_exact(3).all(|p| p[0] == p[1] && p[1] == p[2])
}

fn to_gray(raster: &OwnedRaster) -> OwnedRaster {
    OwnedRaster {
        width: raster.width,
        height: raster.height,
        stride: raster.width as usize,
        format: PixelFormat::Gray8,
        data: raster.data.chunks_exact(3).map(|p| p[0]).collect(),
    }
}

/// A printed PWG raster document as a PDF.
#[derive(Debug, PartialEq, Eq)]
pub struct PrintedDocument {
    pub pdf: Vec<u8>,
    pub pages: u32,
}

/// Limits on a converted document.
#[derive(Clone, Copy, Debug)]
pub struct ConvertLimits {
    pub max_pages: u32,
    pub max_bytes: usize,
    pub jpeg_quality: u8,
}

/// Decodes every page and writes them as one PDF: bilevel pages as G4, the
/// rest as JPEG, and a colour page with no colour in it as gray.
pub fn pwg_to_pdf(data: &[u8], limits: ConvertLimits) -> Result<PrintedDocument, PwgError> {
    let mut reader = PwgReader::new(data)?;
    let mut document = PdfDocument::new();
    while let Some(page) = reader.next_page()? {
        if document.len() >= limits.max_pages as usize {
            return Err(PwgError::TooManyPages {
                max: limits.max_pages,
            });
        }
        let raster = if page.raster.format == PixelFormat::Rgb8 && is_gray(&page.raster.data) {
            to_gray(&page.raster)
        } else {
            page.raster
        };
        document.push(&raster.as_raster()?, page.resolution, limits.jpeg_quality)?;
        if document.encoded_bytes() > limits.max_bytes {
            return Err(PwgError::TooLarge {
                max: limits.max_bytes,
            });
        }
    }
    if document.is_empty() {
        return Err(PwgError::Empty);
    }
    let pages = u32::try_from(document.len()).unwrap_or(u32::MAX);
    let pdf = document.finish();
    if pdf.len() > limits.max_bytes {
        return Err(PwgError::TooLarge {
            max: limits.max_bytes,
        });
    }
    Ok(PrintedDocument { pdf, pages })
}

/// Writes a PWG raster stream, as the class driver does, for tests and the
/// sample generator. Each line is compressed the way the format intends:
/// repeated lines folded, runs of a pixel repeated, the rest copied.
#[doc(hidden)]
pub mod encode {
    use super::{
        BITS_PER_COLOR, BITS_PER_PIXEL, BYTES_PER_LINE, COLOR_SPACE, HEADER_BYTES, HEIGHT,
        HW_RESOLUTION, NUM_COLORS, SPACE_BLACK, SPACE_SGRAY, SPACE_SRGB, SYNC, WIDTH,
    };

    /// How a test page is stored.
    #[derive(Clone, Copy, Debug)]
    pub enum Kind {
        Black1,
        Black8,
        Gray1,
        Gray8,
        Srgb8,
    }

    impl Kind {
        fn fields(self) -> (u32, u32, u32, usize) {
            match self {
                Self::Black1 => (SPACE_BLACK, 1, 1, 1),
                Self::Black8 => (SPACE_BLACK, 8, 1, 1),
                Self::Gray1 => (SPACE_SGRAY, 1, 1, 1),
                Self::Gray8 => (SPACE_SGRAY, 8, 1, 1),
                Self::Srgb8 => (SPACE_SRGB, 8, 3, 3),
            }
        }
    }

    #[derive(Clone, Copy, Debug)]
    pub struct Page<'a> {
        pub kind: Kind,
        pub width: u32,
        pub height: u32,
        pub dpi: u32,
        /// Rows packed end to end.
        pub pixels: &'a [u8],
    }

    fn put(header: &mut [u8], offset: usize, value: u32) {
        header[offset..offset + 4].copy_from_slice(&value.to_be_bytes());
    }

    fn compress_line(line: &[u8], unit: usize, out: &mut Vec<u8>) {
        let units: Vec<&[u8]> = line.chunks_exact(unit).collect();
        let mut i = 0;
        while i < units.len() {
            let mut run = 1;
            while i + run < units.len() && run < 128 && units[i + run] == units[i] {
                run += 1;
            }
            if run > 1 {
                out.push(u8::try_from(run - 1).unwrap_or(127));
                out.extend_from_slice(units[i]);
                i += run;
                continue;
            }
            let mut literal = 1;
            while i + literal < units.len()
                && literal < 128
                && (i + literal + 1 >= units.len() || units[i + literal] != units[i + literal + 1])
            {
                literal += 1;
            }
            if literal == 1 {
                out.push(0);
                out.extend_from_slice(units[i]);
                i += 1;
                continue;
            }
            out.push(u8::try_from(257 - literal).unwrap_or(129));
            for unit in &units[i..i + literal] {
                out.extend_from_slice(unit);
            }
            i += literal;
        }
    }

    pub fn stream(pages: &[Page<'_>]) -> Vec<u8> {
        let mut out = SYNC.to_vec();
        for page in pages {
            let (space, bits, colors, unit) = page.kind.fields();
            let bits_per_pixel = bits * colors;
            let row_bytes = (page.width * bits_per_pixel).div_ceil(8) as usize;
            let mut header = vec![0u8; HEADER_BYTES];
            put(&mut header, HW_RESOLUTION, page.dpi);
            put(&mut header, HW_RESOLUTION + 4, page.dpi);
            put(&mut header, WIDTH, page.width);
            put(&mut header, HEIGHT, page.height);
            put(&mut header, BITS_PER_COLOR, bits);
            put(&mut header, BITS_PER_PIXEL, bits_per_pixel);
            put(
                &mut header,
                BYTES_PER_LINE,
                u32::try_from(row_bytes).unwrap_or(0),
            );
            put(&mut header, COLOR_SPACE, space);
            put(&mut header, NUM_COLORS, colors);
            out.extend_from_slice(&header);

            let rows: Vec<&[u8]> = page.pixels.chunks_exact(row_bytes).collect();
            let mut y = 0;
            while y < rows.len() {
                let mut repeat = 1;
                while y + repeat < rows.len() && repeat < 256 && rows[y + repeat] == rows[y] {
                    repeat += 1;
                }
                out.push(u8::try_from(repeat - 1).unwrap_or(255));
                compress_line(rows[y], unit, &mut out);
                y += repeat;
            }
        }
        out
    }
}

#[cfg(test)]
mod tests {
    use super::encode::{Kind, Page, stream};
    use super::*;

    fn limits() -> ConvertLimits {
        ConvertLimits {
            max_pages: 10,
            max_bytes: 1 << 20,
            jpeg_quality: 80,
        }
    }

    /// A bilevel page with a frame and a diagonal; `ink` is the bit that
    /// means black.
    fn bilevel(width: u32, height: u32, ink: bool) -> Vec<u8> {
        let stride = width.div_ceil(8) as usize;
        let mut data = vec![if ink { 0x00 } else { 0xFF }; stride * height as usize];
        for y in 0..height {
            for x in 0..width {
                let black = x == 0 || y == 0 || x == width - 1 || y == height - 1 || x == y;
                let offset = stride * y as usize + x as usize / 8;
                let mask = 0x80 >> (x % 8);
                if black == ink {
                    data[offset] |= mask;
                } else {
                    data[offset] &= !mask;
                }
            }
        }
        data
    }

    #[test]
    fn every_supported_layout_decodes_to_what_was_printed() {
        let (width, height) = (37u32, 21u32);
        let gradient: Vec<u8> = (0..width * height)
            .map(|i| u8::try_from((i * 7) % 256).expect("byte"))
            .collect();
        let rgb: Vec<u8> = (0..width * height * 3)
            .map(|i| u8::try_from((i * 13) % 256).expect("byte"))
            .collect();
        let black1 = bilevel(width, height, true);
        let gray1 = bilevel(width, height, false);
        let black8: Vec<u8> = gradient.iter().map(|g| !g).collect();
        let pages = [
            Page {
                kind: Kind::Black1,
                width,
                height,
                dpi: 300,
                pixels: &black1,
            },
            Page {
                kind: Kind::Gray1,
                width,
                height,
                dpi: 300,
                pixels: &gray1,
            },
            Page {
                kind: Kind::Gray8,
                width,
                height,
                dpi: 300,
                pixels: &gradient,
            },
            Page {
                kind: Kind::Black8,
                width,
                height,
                dpi: 600,
                pixels: &black8,
            },
            Page {
                kind: Kind::Srgb8,
                width,
                height,
                dpi: 300,
                pixels: &rgb,
            },
        ];
        let data = stream(&pages);
        let mut reader = PwgReader::new(&data).expect("reader");
        let mut decoded = Vec::new();
        while let Some(page) = reader.next_page().expect("page") {
            decoded.push(page);
        }
        assert_eq!(decoded.len(), 5);
        assert_eq!(decoded[0].raster.data, black1);
        assert_eq!(
            decoded[0].raster.format,
            PixelFormat::Bilevel {
                zero_is_black: false
            }
        );
        assert_eq!(decoded[1].raster.data, gray1);
        assert_eq!(
            decoded[1].raster.format,
            PixelFormat::Bilevel {
                zero_is_black: true
            }
        );
        assert_eq!(decoded[2].raster.data, gradient);
        assert_eq!(decoded[3].raster.data, gradient, "black8 reads as gray");
        assert_eq!(decoded[3].resolution, Resolution::square(600));
        assert_eq!(decoded[4].raster.data, rgb);
        assert_eq!(decoded[4].raster.format, PixelFormat::Rgb8);
    }

    #[test]
    fn repeated_lines_and_the_white_fill_packet_expand() {
        let mut data = SYNC.to_vec();
        let mut header = vec![0u8; HEADER_BYTES];
        for (offset, value) in [
            (HW_RESOLUTION, 300),
            (HW_RESOLUTION + 4, 300),
            (WIDTH, 4),
            (HEIGHT, 3),
            (BITS_PER_COLOR, 8),
            (BITS_PER_PIXEL, 8),
            (BYTES_PER_LINE, 4),
            (COLOR_SPACE, SPACE_SGRAY),
            (NUM_COLORS, 1),
        ] {
            header[offset..offset + 4].copy_from_slice(&u32::to_be_bytes(value));
        }
        data.extend_from_slice(&header);
        data.extend_from_slice(&[1, 0, 0x10, 128]);
        data.extend_from_slice(&[0, 0xFE, 1, 2, 3]);
        data.extend_from_slice(&[0, 3]);
        let page = PwgReader::new(&data)
            .expect("reader")
            .next_page()
            .expect("decodes")
            .expect("a page");
        assert_eq!(
            page.raster.data,
            [0x10, 0xFF, 0xFF, 0xFF, 0x10, 0xFF, 0xFF, 0xFF, 1, 2, 3, 3]
        );
    }

    #[test]
    fn hostile_streams_are_refused_without_panicking() {
        assert_eq!(
            PwgReader::new(b"%PDF-1.7").expect_err("a PDF is not PWG"),
            PwgError::NotPwg
        );

        let pixels = vec![0u8; 16];
        let good = stream(&[Page {
            kind: Kind::Gray8,
            width: 4,
            height: 4,
            dpi: 300,
            pixels: &pixels,
        }]);
        for cut in [5, HEADER_BYTES, good.len() - 1] {
            let mut reader = PwgReader::new(&good[..cut]).expect("reader");
            assert!(
                matches!(reader.next_page(), Err(PwgError::Truncated { page: 1 })),
                "cut at {cut}"
            );
        }

        let mut lying = good.clone();
        lying[4 + BYTES_PER_LINE + 3] = 9;
        assert!(matches!(
            PwgReader::new(&lying).expect("reader").next_page(),
            Err(PwgError::Header { .. })
        ));

        let mut huge = good.clone();
        huge[4 + WIDTH..4 + WIDTH + 4].copy_from_slice(&60_000u32.to_be_bytes());
        huge[4 + HEIGHT..4 + HEIGHT + 4].copy_from_slice(&60_000u32.to_be_bytes());
        huge[4 + BYTES_PER_LINE..4 + BYTES_PER_LINE + 4].copy_from_slice(&60_000u32.to_be_bytes());
        assert!(matches!(
            PwgReader::new(&huge).expect("reader").next_page(),
            Err(PwgError::Header { .. })
        ));

        let mut header_only = good[..4 + HEADER_BYTES].to_vec();
        header_only.extend_from_slice(&[0, 9, 0]);
        assert!(matches!(
            PwgReader::new(&header_only).expect("reader").next_page(),
            Err(PwgError::Overrun { page: 1, line: 0 })
        ));
        let mut too_many_lines = good[..4 + HEADER_BYTES].to_vec();
        too_many_lines.extend_from_slice(&[4, 128]);
        assert!(matches!(
            PwgReader::new(&too_many_lines).expect("reader").next_page(),
            Err(PwgError::Overrun { page: 1, line: 0 })
        ));
    }

    #[test]
    fn a_printed_job_becomes_one_pdf_and_gray_colour_pages_become_gray() {
        let (width, height) = (850u32, 1100u32);
        let text = bilevel(width, height, true);
        let gray_rgb = vec![128u8; (width * height * 3) as usize];
        let data = stream(&[
            Page {
                kind: Kind::Black1,
                width,
                height,
                dpi: 100,
                pixels: &text,
            },
            Page {
                kind: Kind::Srgb8,
                width,
                height,
                dpi: 100,
                pixels: &gray_rgb,
            },
        ]);
        let document = pwg_to_pdf(&data, limits()).expect("converts");
        assert_eq!(document.pages, 2);
        let pdf = String::from_utf8_lossy(&document.pdf);
        assert!(pdf.contains("/Count 2"));
        assert!(pdf.contains("/CCITTFaxDecode"));
        assert!(pdf.contains("/DeviceGray"));
        assert!(!pdf.contains("/DeviceRGB"));
        assert!(pdf.contains("/MediaBox [0 0 612 792]"));
    }

    #[test]
    fn a_job_over_the_limits_is_refused() {
        let pixels = vec![0u8; 16];
        let page = Page {
            kind: Kind::Gray8,
            width: 4,
            height: 4,
            dpi: 300,
            pixels: &pixels,
        };
        let three = stream(&[
            Page { ..page },
            Page {
                pixels: &pixels,
                ..page
            },
            Page {
                pixels: &pixels,
                ..page
            },
        ]);
        let tight = ConvertLimits {
            max_pages: 2,
            ..limits()
        };
        assert_eq!(
            pwg_to_pdf(&three, tight),
            Err(PwgError::TooManyPages { max: 2 })
        );
        let small = ConvertLimits {
            max_bytes: 100,
            ..limits()
        };
        assert_eq!(
            pwg_to_pdf(&three, small),
            Err(PwgError::TooLarge { max: 100 })
        );
        assert_eq!(pwg_to_pdf(SYNC, limits()), Err(PwgError::Empty));
    }
}
