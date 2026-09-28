//! Pages encoded and wrapped as PDF: one scanned page at a time, or a whole
//! printed document.
//!
//! Black and white pages are CCITT Group 4, which is what fax machines and
//! every document scanner speak and what keeps a letter page at 300 DPI to a
//! few tens of kilobytes. Grayscale and colour pages are baseline JPEG at the
//! profile's quality. Either way the image is stored exactly as encoded, as
//! the page's only content, with the page sized so the image is at its
//! scanned resolution: the server's thumbnailer, text extraction and OCR all
//! take PDF, and reading the image back loses nothing.

use jpeg_encoder::{ColorType, Encoder as JpegEncoder, PixelDensity, PixelDensityUnit};
use pdf_writer::{Content, Filter, Finish, Name, Pdf, Rect, Ref, TextStr};

use crate::ImagingError;
use crate::raster::{PixelFormat, Raster};

/// The resolutions a page may claim. Anything outside is a driver reporting
/// nonsense, and would produce a page the size of a postage stamp or a wall.
pub const DPI_RANGE: std::ops::RangeInclusive<u32> = 50..=1200;
/// The profile's bounds (`capture.CaptureProfile.Validate`).
pub const JPEG_QUALITY_RANGE: std::ops::RangeInclusive<u8> = 30..=95;

const PRODUCER: &str = "Trenova Capture";
const IMAGE_NAME: Name<'static> = Name(b"Im0");

/// The resolution of a page, which a scanner may report differently for each
/// axis.
#[derive(Clone, Copy, Debug, PartialEq, Eq)]
pub struct Resolution {
    pub x: u32,
    pub y: u32,
}

impl Resolution {
    pub fn square(dpi: u32) -> Self {
        Self { x: dpi, y: dpi }
    }
}

/// A page ready to upload.
#[derive(Clone, Debug, PartialEq, Eq)]
pub struct EncodedPage {
    /// The one-page PDF.
    pub pdf: Vec<u8>,
    pub width_px: u32,
    pub height_px: u32,
    pub resolution: Resolution,
    /// Whether the page went out as bilevel G4 rather than JPEG.
    pub bilevel: bool,
}

/// Encodes a page and wraps it in a PDF.
pub fn encode_page(
    raster: &Raster<'_>,
    resolution: Resolution,
    jpeg_quality: u8,
) -> Result<EncodedPage, ImagingError> {
    if !DPI_RANGE.contains(&resolution.x) || !DPI_RANGE.contains(&resolution.y) {
        return Err(ImagingError::Resolution {
            x: resolution.x,
            y: resolution.y,
        });
    }

    let image = PageImage::encode(raster, resolution, jpeg_quality)?;
    let bilevel = image.bilevel();
    Ok(EncodedPage {
        pdf: write_pdf(std::slice::from_ref(&image)),
        width_px: raster.width,
        height_px: raster.height,
        resolution,
        bilevel,
    })
}

/// A document assembled a page at a time, as a printed job is. Each page is
/// encoded as it is pushed, so only the compressed pages are held.
#[derive(Debug, Default)]
pub struct PdfDocument {
    pages: Vec<PageImage>,
    encoded_bytes: usize,
}

impl PdfDocument {
    pub fn new() -> Self {
        Self::default()
    }

    /// Encodes a page and appends it.
    pub fn push(
        &mut self,
        raster: &Raster<'_>,
        resolution: Resolution,
        jpeg_quality: u8,
    ) -> Result<(), ImagingError> {
        let image = PageImage::encode(raster, resolution, jpeg_quality)?;
        self.encoded_bytes += image.data().len();
        self.pages.push(image);
        Ok(())
    }

    pub fn len(&self) -> usize {
        self.pages.len()
    }

    pub fn is_empty(&self) -> bool {
        self.pages.is_empty()
    }

    /// The size of the encoded images so far, which the PDF exceeds only by
    /// a few hundred bytes a page.
    pub fn encoded_bytes(&self) -> usize {
        self.encoded_bytes
    }

    pub fn finish(self) -> Vec<u8> {
        write_pdf(&self.pages)
    }
}

#[derive(Debug)]
struct PageImage {
    image: EncodedImage,
    width: u32,
    height: u32,
    resolution: Resolution,
}

impl PageImage {
    fn encode(
        raster: &Raster<'_>,
        resolution: Resolution,
        jpeg_quality: u8,
    ) -> Result<Self, ImagingError> {
        if !DPI_RANGE.contains(&resolution.x) || !DPI_RANGE.contains(&resolution.y) {
            return Err(ImagingError::Resolution {
                x: resolution.x,
                y: resolution.y,
            });
        }
        let image = match raster.format {
            PixelFormat::Bilevel { zero_is_black } => {
                EncodedImage::G4(encode_g4(raster, zero_is_black))
            }
            PixelFormat::Gray8 => EncodedImage::Jpeg {
                data: encode_jpeg(raster, ColorType::Luma, jpeg_quality, resolution)?,
                gray: true,
            },
            PixelFormat::Rgb8 => EncodedImage::Jpeg {
                data: encode_jpeg(raster, ColorType::Rgb, jpeg_quality, resolution)?,
                gray: false,
            },
            PixelFormat::Bgr8 => EncodedImage::Jpeg {
                data: encode_jpeg(raster, ColorType::Bgr, jpeg_quality, resolution)?,
                gray: false,
            },
        };
        Ok(Self {
            image,
            width: raster.width,
            height: raster.height,
            resolution,
        })
    }

    fn bilevel(&self) -> bool {
        matches!(self.image, EncodedImage::G4(_))
    }

    fn data(&self) -> &[u8] {
        match &self.image {
            EncodedImage::G4(data) | EncodedImage::Jpeg { data, .. } => data,
        }
    }
}

#[derive(Debug)]
enum EncodedImage {
    G4(Vec<u8>),
    Jpeg { data: Vec<u8>, gray: bool },
}

/// ITU-T T.6, as PDF's `CCITTFaxDecode` reads it with `K -1`.
fn encode_g4(raster: &Raster<'_>, zero_is_black: bool) -> Vec<u8> {
    use fax::{Color, VecWriter, encoder::Encoder};

    let pixels = raster.width as usize * raster.height as usize;
    let mut encoder = Encoder::new(VecWriter::with_capacity(pixels / 8));
    for y in 0..raster.height {
        let row = raster.row(y);
        let pels = (0..raster.width as usize).map(|x| {
            let bit = row[x / 8] >> (7 - (x % 8)) & 1;
            if (bit == 1) == zero_is_black {
                Color::White
            } else {
                Color::Black
            }
        });
        let Ok(()) = encoder.encode_line(pels, raster.width);
    }
    let Ok(writer) = encoder.finish();
    writer.finish()
}

fn encode_jpeg(
    raster: &Raster<'_>,
    color: ColorType,
    quality: u8,
    resolution: Resolution,
) -> Result<Vec<u8>, ImagingError> {
    if !JPEG_QUALITY_RANGE.contains(&quality) {
        return Err(ImagingError::Quality(quality));
    }
    let width = u16::try_from(raster.width).map_err(|_| ImagingError::Dimensions {
        width: raster.width,
        height: raster.height,
    })?;
    let height = u16::try_from(raster.height).map_err(|_| ImagingError::Dimensions {
        width: raster.width,
        height: raster.height,
    })?;
    let density = |dpi: u32| u16::try_from(dpi).unwrap_or(u16::MAX);

    let packed = raster.packed();
    let mut out = Vec::with_capacity(packed.len() / 8);
    let mut encoder = JpegEncoder::new(&mut out, quality);
    encoder.set_density(PixelDensity {
        density: (density(resolution.x), density(resolution.y)),
        unit: PixelDensityUnit::Inches,
    });
    encoder
        .encode(&packed, width, height, color)
        .map_err(|err| ImagingError::Jpeg(err.to_string()))?;
    Ok(out)
}

#[allow(clippy::cast_precision_loss)]
fn points(pixels: u32, dpi: u32) -> f32 {
    pixels as f32 * 72.0 / dpi as f32
}

/// Each page's image stored as encoded, as the page's only content, with
/// the page sized so the image is at its resolution.
fn write_pdf(pages: &[PageImage]) -> Vec<u8> {
    const CATALOG: Ref = Ref::new(1);
    const PAGE_TREE: Ref = Ref::new(2);
    const INFO: Ref = Ref::new(3);
    const FIRST_PAGE: i32 = 4;

    let object = |page: usize, offset: i32| {
        let page = i32::try_from(page).unwrap_or(i32::MAX / 4);
        Ref::new(FIRST_PAGE + page * 3 + offset)
    };

    let mut pdf = Pdf::new();
    pdf.catalog(CATALOG).pages(PAGE_TREE);
    pdf.pages(PAGE_TREE)
        .kids((0..pages.len()).map(|i| object(i, 0)))
        .count(i32::try_from(pages.len()).unwrap_or(i32::MAX));

    for (i, page) in pages.iter().enumerate() {
        let (page_id, image_id, content_id) = (object(i, 0), object(i, 1), object(i, 2));
        let width_pt = points(page.width, page.resolution.x);
        let height_pt = points(page.height, page.resolution.y);
        let columns = i32::try_from(page.width).unwrap_or(i32::MAX);
        let rows = i32::try_from(page.height).unwrap_or(i32::MAX);

        let mut writer = pdf.page(page_id);
        writer.media_box(Rect::new(0.0, 0.0, width_pt, height_pt));
        writer.parent(PAGE_TREE);
        writer.contents(content_id);
        writer.resources().x_objects().pair(IMAGE_NAME, image_id);
        writer.finish();

        let mut xobject = pdf.image_xobject(image_id, page.data());
        xobject.width(columns);
        xobject.height(rows);
        match &page.image {
            EncodedImage::G4(_) => {
                xobject.filter(Filter::CcittFaxDecode);
                xobject.color_space().device_gray();
                xobject.bits_per_component(1);
                xobject
                    .decode_parms()
                    .k(-1)
                    .columns(columns)
                    .rows(rows)
                    .black_is_1(false);
            }
            EncodedImage::Jpeg { gray, .. } => {
                xobject.filter(Filter::DctDecode);
                if *gray {
                    xobject.color_space().device_gray();
                } else {
                    xobject.color_space().device_rgb();
                }
                xobject.bits_per_component(8);
            }
        }
        xobject.finish();

        let mut content = Content::new();
        content.save_state();
        content.transform([width_pt, 0.0, 0.0, height_pt, 0.0, 0.0]);
        content.x_object(IMAGE_NAME);
        content.restore_state();
        pdf.stream(content_id, &content.finish());
    }

    pdf.document_info(INFO).producer(TextStr(PRODUCER));
    pdf.finish()
}

#[cfg(test)]
mod tests {
    use super::*;

    /// A bilevel page with a black frame and a diagonal, chocolate or vanilla.
    fn drawing(width: u32, height: u32, zero_is_black: bool) -> Vec<u8> {
        let stride = PixelFormat::Bilevel { zero_is_black }.row_bytes(width);
        let mut data = vec![0u8; stride * height as usize];
        for y in 0..height {
            for x in 0..width {
                let black = x < 3 || y < 3 || x >= width - 3 || y >= height - 3 || x == y;
                let bit = if zero_is_black { !black } else { black };
                if bit {
                    data[stride * y as usize + x as usize / 8] |= 0x80 >> (x % 8);
                }
            }
        }
        data
    }

    fn decode(g4: &[u8], width: u32, height: u32) -> Vec<Vec<bool>> {
        let mut rows = Vec::new();
        fax::decoder::decode_g4(g4.iter().copied(), width, Some(height), |transitions| {
            let mut row = vec![false; width as usize];
            let mut black = false;
            let mut start = 0;
            for &edge in transitions.iter().chain(std::iter::once(&width)) {
                for pixel in &mut row[start as usize..edge as usize] {
                    *pixel = black;
                }
                black = !black;
                start = edge;
            }
            rows.push(row);
        })
        .expect("decodes");
        rows
    }

    #[test]
    fn g4_round_trips_both_bit_meanings() {
        let (width, height) = (61, 40);
        for zero_is_black in [true, false] {
            let data = drawing(width, height, zero_is_black);
            let stride = PixelFormat::Bilevel { zero_is_black }.row_bytes(width);
            let raster = Raster::new(
                width,
                height,
                stride,
                PixelFormat::Bilevel { zero_is_black },
                &data,
            )
            .expect("raster");
            let rows = decode(&encode_g4(&raster, zero_is_black), width, height);
            assert_eq!(rows.len(), height as usize);
            for (y, row) in rows.iter().enumerate() {
                for (x, &black) in row.iter().enumerate() {
                    let expected = x < 3
                        || y < 3
                        || x >= width as usize - 3
                        || y >= height as usize - 3
                        || x == y;
                    assert_eq!(
                        black, expected,
                        "pixel ({x}, {y}), zero_is_black={zero_is_black}"
                    );
                }
            }
        }
    }

    #[test]
    fn a_bilevel_page_is_a_one_page_g4_pdf_sized_to_its_resolution() {
        let (width, height) = (2550, 3300);
        let data = drawing(width, height, true);
        let stride = PixelFormat::Bilevel {
            zero_is_black: true,
        }
        .row_bytes(width);
        let raster = Raster::new(
            width,
            height,
            stride,
            PixelFormat::Bilevel {
                zero_is_black: true,
            },
            &data,
        )
        .expect("raster");
        let page = encode_page(&raster, Resolution::square(300), 80).expect("page");
        assert!(page.bilevel);
        assert!(page.pdf.starts_with(b"%PDF-"));
        let text = String::from_utf8_lossy(&page.pdf);
        assert!(text.contains("/CCITTFaxDecode"));
        assert!(text.contains("/K -1"));
        assert!(text.contains("/Columns 2550"));
        assert!(text.contains("/MediaBox [0 0 612 792]"));
        assert!(text.contains("/Count 1"));
        assert!(
            page.pdf.len() < 40_000,
            "a mostly white letter page is small: {}",
            page.pdf.len()
        );
    }

    #[test]
    fn gray_and_colour_pages_are_jpeg_at_the_profile_quality() {
        let (width, height) = (40u32, 30u32);
        let gray: Vec<u8> = (0..width * height)
            .map(|i| u8::try_from(i % 256).expect("byte"))
            .collect();
        let raster =
            Raster::new(width, height, width as usize, PixelFormat::Gray8, &gray).expect("raster");
        let page = encode_page(&raster, Resolution::square(200), 80).expect("page");
        assert!(!page.bilevel);
        let text = String::from_utf8_lossy(&page.pdf);
        assert!(text.contains("/DCTDecode"));
        assert!(text.contains("/DeviceGray"));
        assert!(text.contains("/MediaBox [0 0 14.4 10.8]"));

        let rgb = vec![200u8; (width * height * 3) as usize];
        let raster = Raster::new(width, height, width as usize * 3, PixelFormat::Rgb8, &rgb)
            .expect("raster");
        let page = encode_page(&raster, Resolution { x: 300, y: 150 }, 95).expect("page");
        let text = String::from_utf8_lossy(&page.pdf);
        assert!(text.contains("/DeviceRGB"));
        assert!(text.contains("/MediaBox [0 0 9.6 14.4]"));
    }

    #[test]
    fn nonsense_from_a_driver_is_refused() {
        let gray = [0u8; 4];
        let raster = Raster::new(2, 2, 2, PixelFormat::Gray8, &gray).expect("raster");
        assert!(matches!(
            encode_page(&raster, Resolution::square(0), 80),
            Err(ImagingError::Resolution { .. })
        ));
        assert!(matches!(
            encode_page(&raster, Resolution::square(300), 100),
            Err(ImagingError::Quality(100))
        ));
    }
}
