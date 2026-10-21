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
//!
//! A JPEG that already exists (a phone photo) is placed as it is, without
//! decoding and encoding it again, and a photo's page can be a letter sheet
//! the image is fitted onto rather than the size of its pixels.

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
/// US letter, in points.
const LETTER: (f32, f32) = (612.0, 792.0);
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
        pdf: write_pdf(std::slice::from_ref(&image), PageFit::Native, PRODUCER),
        width_px: raster.width,
        height_px: raster.height,
        resolution,
        bilevel,
    })
}

/// A document assembled a page at a time, as a printed job is. Each page is
/// encoded as it is pushed, so only the compressed pages are held.
#[derive(Debug)]
pub struct PdfDocument {
    pages: Vec<PageImage>,
    encoded_bytes: usize,
    fit: PageFit,
    producer: &'static str,
}

impl Default for PdfDocument {
    fn default() -> Self {
        Self {
            pages: Vec::new(),
            encoded_bytes: 0,
            fit: PageFit::Native,
            producer: PRODUCER,
        }
    }
}

/// How a page is sized around its image.
#[derive(Clone, Copy, Debug, PartialEq)]
pub enum PageFit {
    /// The page is the image at its resolution, as a scanned page is.
    Native,
    /// The page is `width` × `height` points, turned to the image's
    /// orientation, with the image centred and scaled to `scale` of it.
    Fit { width: f32, height: f32, scale: f32 },
}

impl PageFit {
    /// A letter sheet with the image at 95% of it: how a photographed
    /// document is put into a PDF so it prints and merges like paper.
    pub const LETTER: Self = Self::Fit {
        width: LETTER.0,
        height: LETTER.1,
        scale: 0.95,
    };
}

/// How the samples of a JPEG are coloured.
#[derive(Clone, Copy, Debug, PartialEq, Eq)]
pub enum JpegColor {
    Gray,
    Rgb,
    /// Four components. Adobe's encoders store CMYK inverted, which the
    /// image's `Decode` array undoes.
    Cmyk {
        inverted: bool,
    },
}

/// A JPEG to place as it is. The caller has read its header: the
/// dimensions and colour here are what a PDF reader is told.
#[derive(Clone, Debug, PartialEq, Eq)]
pub struct PassthroughJpeg {
    pub data: Vec<u8>,
    pub width: u32,
    pub height: u32,
    pub color: JpegColor,
}

impl PdfDocument {
    pub fn new() -> Self {
        Self::default()
    }

    /// Sizes every page with `fit` rather than at the image's resolution.
    #[must_use]
    pub fn with_fit(mut self, fit: PageFit) -> Self {
        self.fit = fit;
        self
    }

    /// Names the producer written into the document information.
    #[must_use]
    pub fn with_producer(mut self, producer: &'static str) -> Self {
        self.producer = producer;
        self
    }

    /// Appends an already encoded JPEG without decoding it.
    pub fn push_jpeg(
        &mut self,
        jpeg: PassthroughJpeg,
        resolution: Resolution,
    ) -> Result<(), ImagingError> {
        if !DPI_RANGE.contains(&resolution.x) || !DPI_RANGE.contains(&resolution.y) {
            return Err(ImagingError::Resolution {
                x: resolution.x,
                y: resolution.y,
            });
        }
        if jpeg.width == 0
            || jpeg.height == 0
            || jpeg.width > crate::raster::MAX_SIDE
            || jpeg.height > crate::raster::MAX_SIDE
        {
            return Err(ImagingError::Dimensions {
                width: jpeg.width,
                height: jpeg.height,
            });
        }
        self.encoded_bytes += jpeg.data.len();
        self.pages.push(PageImage {
            image: EncodedImage::Jpeg {
                data: jpeg.data,
                color: jpeg.color,
            },
            width: jpeg.width,
            height: jpeg.height,
            resolution,
        });
        Ok(())
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
        write_pdf(&self.pages, self.fit, self.producer)
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
                color: JpegColor::Gray,
            },
            PixelFormat::Rgb8 => EncodedImage::Jpeg {
                data: encode_jpeg(raster, ColorType::Rgb, jpeg_quality, resolution)?,
                color: JpegColor::Rgb,
            },
            PixelFormat::Bgr8 => EncodedImage::Jpeg {
                data: encode_jpeg(raster, ColorType::Bgr, jpeg_quality, resolution)?,
                color: JpegColor::Rgb,
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
    Jpeg { data: Vec<u8>, color: JpegColor },
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

/// Encodes a raster as baseline JPEG at `quality` (30 to 95).
pub fn encode_jpeg(
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

/// Where a page's image sits: the media box, and the image's origin and size,
/// all in points.
fn placement(page: &PageImage, fit: PageFit) -> ([f32; 2], [f32; 4]) {
    let width_pt = points(page.width, page.resolution.x);
    let height_pt = points(page.height, page.resolution.y);
    match fit {
        PageFit::Native => ([width_pt, height_pt], [0.0, 0.0, width_pt, height_pt]),
        PageFit::Fit {
            width,
            height,
            scale,
        } => {
            let (short, long) = (width.min(height), width.max(height));
            let (page_w, page_h) = if width_pt > height_pt {
                (long, short)
            } else {
                (short, long)
            };
            let scale = scale.clamp(0.1, 1.0);
            let factor = (page_w * scale / width_pt).min(page_h * scale / height_pt);
            let (image_w, image_h) = (width_pt * factor, height_pt * factor);
            (
                [page_w, page_h],
                [
                    (page_w - image_w) / 2.0,
                    (page_h - image_h) / 2.0,
                    image_w,
                    image_h,
                ],
            )
        }
    }
}

/// Each page's image stored as encoded, as the page's only content, with
/// the page sized by `fit`.
fn write_pdf(pages: &[PageImage], fit: PageFit, producer: &str) -> Vec<u8> {
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
        let ([page_w, page_h], [x, y, image_w, image_h]) = placement(page, fit);
        let columns = i32::try_from(page.width).unwrap_or(i32::MAX);
        let rows = i32::try_from(page.height).unwrap_or(i32::MAX);

        let mut writer = pdf.page(page_id);
        writer.media_box(Rect::new(0.0, 0.0, page_w, page_h));
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
            EncodedImage::Jpeg { color, .. } => {
                xobject.filter(Filter::DctDecode);
                match color {
                    JpegColor::Gray => xobject.color_space().device_gray(),
                    JpegColor::Rgb => xobject.color_space().device_rgb(),
                    JpegColor::Cmyk { inverted } => {
                        xobject.color_space().device_cmyk();
                        if *inverted {
                            xobject.decode([1.0, 0.0, 1.0, 0.0, 1.0, 0.0, 1.0, 0.0]);
                        }
                    }
                }
                xobject.bits_per_component(8);
            }
        }
        xobject.finish();

        let mut content = Content::new();
        content.save_state();
        content.transform([image_w, 0.0, 0.0, image_h, x, y]);
        content.x_object(IMAGE_NAME);
        content.restore_state();
        pdf.stream(content_id, &content.finish());
    }

    pdf.document_info(INFO).producer(TextStr(producer));
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

    #[test]
    fn a_passed_through_jpeg_is_stored_byte_for_byte_on_a_fitted_letter_page() {
        let gray = vec![128u8; 300 * 200];
        let raster = Raster::new(300, 200, 300, PixelFormat::Gray8, &gray).expect("raster");
        let jpeg = encode_jpeg(&raster, ColorType::Luma, 80, Resolution::square(72)).expect("jpeg");

        let mut doc = PdfDocument::new()
            .with_fit(PageFit::LETTER)
            .with_producer("Trenova");
        doc.push_jpeg(
            PassthroughJpeg {
                data: jpeg.clone(),
                width: 300,
                height: 200,
                color: JpegColor::Gray,
            },
            Resolution::square(72),
        )
        .expect("push");
        let pdf = doc.finish();

        assert!(
            pdf.windows(jpeg.len())
                .any(|window| window == jpeg.as_slice())
        );
        let text = String::from_utf8_lossy(&pdf);
        assert!(
            text.contains("/MediaBox [0 0 792 612]"),
            "a landscape photo gets a landscape sheet"
        );
        assert!(text.contains("(Trenova)"));
    }

    #[test]
    fn a_passed_through_jpeg_with_impossible_dimensions_is_refused() {
        let mut doc = PdfDocument::new();
        let jpeg = PassthroughJpeg {
            data: vec![0xFF, 0xD8],
            width: 0,
            height: 10,
            color: JpegColor::Rgb,
        };
        assert!(matches!(
            doc.push_jpeg(jpeg, Resolution::square(72)),
            Err(ImagingError::Dimensions { .. })
        ));
    }

    #[test]
    fn fitting_keeps_the_aspect_ratio_inside_the_sheet() {
        let page = PageImage {
            image: EncodedImage::G4(Vec::new()),
            width: 1000,
            height: 4000,
            resolution: Resolution::square(100),
        };
        let ([w, h], [x, y, iw, ih]) = placement(&page, PageFit::LETTER);
        assert!((w - 612.0).abs() < 0.01 && (h - 792.0).abs() < 0.01);
        assert!((iw / ih - 0.25).abs() < 0.001);
        assert!(x >= 0.0 && y >= 0.0 && x + iw <= w && y + ih <= h);
    }
}
