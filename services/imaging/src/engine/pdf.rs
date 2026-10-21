//! PDFium, as the engine uses it: opening a document from mapped bytes,
//! describing it, drawing pages, and building new documents from pages of
//! others.
//!
//! The library is bound once per child, before the child hardens itself, and
//! every document is opened from the job's read-only mapping, so nothing is
//! copied on the way in.

use pdfium_render::prelude::*;
use trenova_proto::imaging::v1::{ErrorReason, PageInfo};

use super::error::{EngineError, EngineResult};
use super::raster::Rgba;

const POINTS_PER_INCH: f64 = 72.0;
const MAX_DPI: u32 = 1200;

/// The bound library.
pub struct Pdf {
    pdfium: Pdfium,
}

impl std::fmt::Debug for Pdf {
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        f.debug_struct("Pdf").finish_non_exhaustive()
    }
}

impl Pdf {
    /// Binds libpdfium at `library`.
    pub fn bind(library: &str) -> EngineResult<Self> {
        let bindings = Pdfium::bind_to_library(library).map_err(|err| {
            EngineError::internal(format!("load libpdfium from {library}: {err}"))
        })?;
        Ok(Self {
            pdfium: Pdfium::new(bindings),
        })
    }

    /// Opens a document. An encrypted document that needs a password is
    /// `ENCRYPTED`; anything PDFium cannot parse is `INVALID_DOCUMENT`.
    pub fn open<'a>(&'a self, bytes: &'a [u8]) -> EngineResult<PdfDocument<'a>> {
        if bytes.is_empty() {
            return Err(EngineError::invalid("the PDF is empty"));
        }
        self.pdfium
            .load_pdf_from_byte_slice(bytes, None)
            .map_err(open_error)
    }

    /// Opens a document this process wrote itself (an image's page).
    pub fn open_owned(&self, bytes: Vec<u8>) -> EngineResult<PdfDocument<'_>> {
        self.pdfium
            .load_pdf_from_byte_vec(bytes, None)
            .map_err(open_error)
    }

    pub fn create(&self) -> EngineResult<PdfDocument<'_>> {
        self.pdfium
            .create_new_pdf()
            .map_err(|err| EngineError::internal(format!("create a PDF: {err}")))
    }
}

fn open_error(err: PdfiumError) -> EngineError {
    match err {
        PdfiumError::PdfiumLibraryInternalError(PdfiumInternalError::PasswordError) => {
            EngineError::new(
                ErrorReason::Encrypted,
                "the PDF is encrypted and needs a password",
            )
        }
        PdfiumError::PdfiumLibraryInternalError(PdfiumInternalError::FormatError) => {
            EngineError::invalid("the file is not a readable PDF")
        }
        other => EngineError::invalid(format!("the PDF could not be opened: {other}")),
    }
}

pub fn page_count(document: &PdfDocument<'_>) -> u32 {
    u32::try_from(document.pages().len()).unwrap_or(0)
}

pub fn page<'a>(document: &PdfDocument<'a>, index: u32) -> EngineResult<PdfPage<'a>> {
    let count = page_count(document);
    let page_index = i32::try_from(index).map_err(|_| EngineError::page(index, count))?;
    if index >= count {
        return Err(EngineError::page(index, count));
    }
    document
        .pages()
        .get(page_index)
        .map_err(|err| EngineError::invalid(format!("page {index} could not be read: {err}")))
}

/// Whether the document carries a security handler (opened without a user
/// password, but encrypted all the same).
pub fn is_encrypted(document: &PdfDocument<'_>) -> bool {
    document
        .permissions()
        .security_handler_revision()
        .is_ok_and(|revision| !matches!(revision, PdfSecurityHandlerRevision::Unprotected))
}

pub fn version(document: &PdfDocument<'_>) -> String {
    let text = match document.version() {
        PdfDocumentVersion::Pdf1_0 => "1.0",
        PdfDocumentVersion::Pdf1_1 => "1.1",
        PdfDocumentVersion::Pdf1_2 => "1.2",
        PdfDocumentVersion::Pdf1_3 => "1.3",
        PdfDocumentVersion::Pdf1_4 => "1.4",
        PdfDocumentVersion::Pdf1_5 => "1.5",
        PdfDocumentVersion::Pdf1_6 => "1.6",
        PdfDocumentVersion::Pdf1_7 => "1.7",
        PdfDocumentVersion::Pdf2_0 => "2.0",
        PdfDocumentVersion::Unset => "",
        PdfDocumentVersion::Other(other) => return format!("{}.{}", other / 10, other % 10),
    };
    text.to_owned()
}

pub fn producer(document: &PdfDocument<'_>) -> String {
    let producer = document
        .metadata()
        .get(PdfDocumentMetadataTagType::Producer)
        .map(|tag| tag.value().to_owned())
        .unwrap_or_default();
    producer
        .chars()
        .filter(|c| !c.is_control())
        .take(256)
        .collect()
}

pub fn rotation_degrees(page: &PdfPage<'_>) -> u32 {
    match page.rotation() {
        Ok(PdfPageRenderRotation::Degrees90) => 90,
        Ok(PdfPageRenderRotation::Degrees180) => 180,
        Ok(PdfPageRenderRotation::Degrees270) => 270,
        _ => 0,
    }
}

pub fn rotation_from_degrees(degrees: u32) -> PdfPageRenderRotation {
    match degrees % 360 {
        90 => PdfPageRenderRotation::Degrees90,
        180 => PdfPageRenderRotation::Degrees180,
        270 => PdfPageRenderRotation::Degrees270,
        _ => PdfPageRenderRotation::None,
    }
}

/// A page's size and rotation, and, when asked, whether it carries text.
pub fn page_info(page: &PdfPage<'_>, index: u32, text_profile: bool) -> PageInfo {
    let mut info = PageInfo {
        index,
        width_points: page.width().value,
        height_points: page.height().value,
        rotation: rotation_degrees(page),
        has_text: false,
        char_count: 0,
    };
    if text_profile && let Ok(text) = page.text() {
        let all = text.all();
        info.has_text = !all.trim().is_empty();
        info.char_count =
            u32::try_from(all.chars().filter(|c| !c.is_whitespace()).count()).unwrap_or(u32::MAX);
    }
    info
}

/// The pixel size of a page drawn at `dpi`, lowered to stay inside
/// `max_pixels` (the TMS has always drawn an oversized page smaller rather
/// than refusing it).
pub fn size_at_dpi(
    width_pt: f32,
    height_pt: f32,
    dpi: u32,
    max_pixels: u64,
) -> EngineResult<(u32, u32)> {
    if dpi == 0 || dpi > MAX_DPI {
        return Err(EngineError::request(format!(
            "a dpi of {dpi} is outside 1 to {MAX_DPI}"
        )));
    }
    let scale = f64::from(dpi) / POINTS_PER_INCH;
    let (mut w, mut h) = (f64::from(width_pt) * scale, f64::from(height_pt) * scale);
    #[allow(clippy::cast_precision_loss)]
    let budget = max_pixels as f64;
    if max_pixels > 0 && w * h > budget {
        let shrink = (budget / (w * h)).sqrt();
        w *= shrink;
        h *= shrink;
    }
    pixel_size(w, h)
}

/// The pixel size of a page fitted inside the box (0 for no bound on a
/// side), never larger than the page at 300 DPI, and within `max_pixels`.
pub fn size_to_fit(
    width_pt: f32,
    height_pt: f32,
    max_width: u32,
    max_height: u32,
    max_pixels: u64,
) -> EngineResult<(u32, u32)> {
    if max_width == 0 && max_height == 0 {
        return Err(EngineError::request(
            "a fit box needs a width, a height or both",
        ));
    }
    let (w, h) = (f64::from(width_pt).max(1.0), f64::from(height_pt).max(1.0));
    let by_width = if max_width == 0 {
        f64::INFINITY
    } else {
        f64::from(max_width) / w
    };
    let by_height = if max_height == 0 {
        f64::INFINITY
    } else {
        f64::from(max_height) / h
    };
    let scale = by_width.min(by_height).min(300.0 / POINTS_PER_INCH);
    let (mut pw, mut ph) = (w * scale, h * scale);
    #[allow(clippy::cast_precision_loss)]
    let budget = max_pixels as f64;
    if max_pixels > 0 && pw * ph > budget {
        let shrink = (budget / (pw * ph)).sqrt();
        pw *= shrink;
        ph *= shrink;
    }
    pixel_size(pw, ph)
}

#[allow(clippy::cast_possible_truncation, clippy::cast_sign_loss)]
fn pixel_size(w: f64, h: f64) -> EngineResult<(u32, u32)> {
    let (w, h) = (w.round().max(1.0), h.round().max(1.0));
    if w > 65_535.0 || h > 65_535.0 {
        return Err(EngineError::pixels("the page is too large to draw"));
    }
    Ok((w as u32, h as u32))
}

/// Draws a page at exactly `width` × `height`, onto white.
pub fn render(page: &PdfPage<'_>, width: u32, height: u32) -> EngineResult<Rgba> {
    let w = i32::try_from(width).map_err(|_| EngineError::pixels("too wide"))?;
    let h = i32::try_from(height).map_err(|_| EngineError::pixels("too tall"))?;
    let config = PdfRenderConfig::new()
        .set_fixed_size(w, h)
        .set_format(PdfBitmapFormat::BGRA)
        .set_clear_color(PdfColor::WHITE)
        .render_form_data(true)
        .render_annotations(true);
    let bitmap = page
        .render_with_config(&config)
        .map_err(|err| EngineError::invalid(format!("the page could not be drawn: {err}")))?;
    let mut image = Rgba::new(width, height, bitmap.as_rgba_bytes())?;
    image.flatten_on_white();
    Ok(image)
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn an_oversized_page_is_drawn_smaller_rather_than_refused() {
        let (w, h) = size_at_dpi(14_400.0, 14_400.0, 150, 25_000_000).expect("bounded");
        assert!(u64::from(w) * u64::from(h) <= 25_100_000);
        assert_eq!(
            size_at_dpi(612.0, 792.0, 150, 25_000_000).expect("letter"),
            (1275, 1650)
        );
    }

    #[test]
    fn fitting_a_letter_page_to_the_viewer() {
        assert_eq!(
            size_to_fit(612.0, 792.0, 1400, 1800, 0).expect("fits"),
            (1391, 1800)
        );
        assert_eq!(
            size_to_fit(612.0, 792.0, 240, 0, 0).expect("fits"),
            (240, 311)
        );
        assert!(size_to_fit(612.0, 792.0, 0, 0, 0).is_err());
    }
}
