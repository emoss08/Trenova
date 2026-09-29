//! Sample pages for the test scanner: freight paperwork drawn at any
//! resolution, in black and white, gray or colour, so every part of
//! scanning can be tried on a computer without a scanner.
//!
//! The pages are drawn, not loaded, with a 5 × 7 bitmap font, so nothing
//! needs shipping beside the helper. Every page says in its footer that it
//! is a sample.

use capture_protocol::api::PixelType;

use crate::raster::{OwnedRaster, PixelFormat};

/// US letter, in inches × 10.
const LETTER_WIDTH_TENTHS: u32 = 85;
const LETTER_HEIGHT_TENTHS: u32 = 110;

/// What a sample page shows.
#[derive(Clone, Debug, PartialEq, Eq)]
pub enum SampleKind {
    BillOfLading,
    DeliveryReceipt,
    /// The back of a bill of lading: its terms.
    Terms,
    /// A patch sheet, which a scanner that reads patch codes reports and a
    /// scan to intake splits documents on.
    PatchT,
    /// A blank side, which blank-page removal drops.
    Blank,
}

/// One sample page.
#[derive(Clone, Debug, PartialEq, Eq)]
pub struct SamplePage {
    pub kind: SampleKind,
    /// The PRO number printed on it, and carried as its barcode.
    pub pro: String,
    /// Its place in its document, from 1, and the document's length.
    pub page: u32,
    pub of: u32,
}

impl SamplePage {
    /// The barcode printed on the page, when it has one.
    pub fn barcode(&self) -> Option<String> {
        match self.kind {
            SampleKind::BillOfLading | SampleKind::DeliveryReceipt if self.page == 1 => {
                Some(format!("PRO {}", self.pro))
            }
            _ => None,
        }
    }

    pub fn is_blank(&self) -> bool {
        self.kind == SampleKind::Blank
    }
}

/// A gray canvas drawn on in page units, with an optional colour layer for
/// the few coloured marks.
struct Canvas {
    width: u32,
    height: u32,
    /// Pixels per tenth of an inch, as the resolution gives it.
    scale: f32,
    gray: Vec<u8>,
    /// Red, green and blue where a mark is coloured.
    color: Option<Vec<u8>>,
}

#[derive(Clone, Copy)]
enum Ink {
    Black,
    Gray,
    Blue,
    Red,
}

impl Ink {
    fn gray(self) -> u8 {
        match self {
            Self::Black => 20,
            Self::Gray => 150,
            Self::Blue => 70,
            Self::Red => 95,
        }
    }

    fn rgb(self) -> [u8; 3] {
        match self {
            Self::Black => [20, 20, 24],
            Self::Gray => [150, 150, 155],
            Self::Blue => [30, 70, 170],
            Self::Red => [200, 40, 40],
        }
    }
}

impl Canvas {
    fn new(dpi: u32, colour: bool) -> Self {
        let width = LETTER_WIDTH_TENTHS * dpi / 10;
        let height = LETTER_HEIGHT_TENTHS * dpi / 10;
        let pixels = width as usize * height as usize;
        #[allow(clippy::cast_precision_loss)]
        let scale = dpi as f32 / 10.0;
        Self {
            width,
            height,
            scale,
            gray: vec![255; pixels],
            color: colour.then(|| vec![255; pixels * 3]),
        }
    }

    #[allow(
        clippy::cast_possible_truncation,
        clippy::cast_sign_loss,
        clippy::cast_precision_loss
    )]
    fn px(&self, tenths: f32) -> u32 {
        (tenths * self.scale).max(0.0).round() as u32
    }

    /// Fills a rectangle given in tenths of an inch.
    fn fill(&mut self, x: f32, y: f32, w: f32, h: f32, ink: Ink) {
        let (x0, y0) = (self.px(x), self.px(y));
        let x1 = self.px(x + w).max(x0 + 1).min(self.width);
        let y1 = self.px(y + h).max(y0 + 1).min(self.height);
        self.fill_px(x0, y0, x1, y1, ink);
    }

    fn fill_px(&mut self, x0: u32, y0: u32, x1: u32, y1: u32, ink: Ink) {
        let (gray, rgb) = (ink.gray(), ink.rgb());
        for y in y0.min(self.height)..y1.min(self.height) {
            let row = y as usize * self.width as usize;
            for x in x0.min(self.width)..x1.min(self.width) {
                let at = row + x as usize;
                self.gray[at] = gray;
                if let Some(color) = &mut self.color {
                    color[at * 3..at * 3 + 3].copy_from_slice(&rgb);
                }
            }
        }
    }

    /// A rectangle's outline, `weight` tenths thick.
    fn frame(&mut self, x: f32, y: f32, w: f32, h: f32, weight: f32, ink: Ink) {
        self.fill(x, y, w, weight, ink);
        self.fill(x, y + h - weight, w, weight, ink);
        self.fill(x, y, weight, h, ink);
        self.fill(x + w - weight, y, weight, h, ink);
    }

    /// Text in capitals, `size` tenths of an inch tall.
    fn text(&mut self, x: f32, y: f32, size: f32, ink: Ink, text: &str) {
        let dot = (self.px(size) / 7).max(1);
        let mut left = self.px(x);
        let top = self.px(y);
        for c in text.chars() {
            let rows = glyph(c.to_ascii_uppercase());
            for (row, bits) in rows.iter().enumerate() {
                for col in 0..5u32 {
                    if bits >> (4 - col) & 1 == 1 {
                        let px = left + col * dot;
                        let py = top + u32::try_from(row).unwrap_or(0) * dot;
                        self.fill_px(px, py, px + dot, py + dot, ink);
                    }
                }
            }
            left += dot * 6;
        }
    }

    /// Bars that look like a barcode for `value`; it is drawn for the eye,
    /// and the scanner reports the value itself.
    fn barcode(&mut self, x: f32, y: f32, h: f32, value: &str) {
        let mut at = x;
        for (i, byte) in value.bytes().enumerate() {
            for bit in 0..6 {
                let wide = (byte >> bit) & 1 == 1;
                let width = if wide { 0.3 } else { 0.12 };
                if (i + bit) % 2 == 0 {
                    self.fill(at, y, width, h, Ink::Black);
                }
                at += width + 0.1;
            }
        }
    }

    fn into_raster(self, pixel_type: PixelType) -> OwnedRaster {
        let width = self.width;
        let height = self.height;
        match (pixel_type, self.color) {
            (PixelType::Color, Some(color)) => OwnedRaster {
                width,
                height,
                stride: width as usize * 3,
                format: PixelFormat::Rgb8,
                data: color,
            },
            (PixelType::BlackWhite, _) => {
                let row_bytes = (width as usize).div_ceil(8);
                let mut data = vec![0u8; row_bytes * height as usize];
                for y in 0..height as usize {
                    for x in 0..width as usize {
                        if self.gray[y * width as usize + x] >= 128 {
                            data[y * row_bytes + x / 8] |= 0x80 >> (x % 8);
                        }
                    }
                }
                OwnedRaster {
                    width,
                    height,
                    stride: row_bytes,
                    format: PixelFormat::Bilevel {
                        zero_is_black: true,
                    },
                    data,
                }
            }
            _ => OwnedRaster {
                width,
                height,
                stride: width as usize,
                format: PixelFormat::Gray8,
                data: self.gray,
            },
        }
    }
}

/// Draws a sample page at `dpi` (clamped to 75–300, which is plenty to
/// read one) in `pixel_type`.
pub fn draw(page: &SamplePage, dpi: u32, pixel_type: PixelType) -> OwnedRaster {
    let dpi = dpi.clamp(75, 300);
    let mut canvas = Canvas::new(dpi, pixel_type == PixelType::Color);
    match page.kind {
        SampleKind::Blank => {}
        SampleKind::PatchT => draw_patch(&mut canvas),
        SampleKind::Terms => draw_terms(&mut canvas, page),
        SampleKind::BillOfLading => draw_document(&mut canvas, page, "BILL OF LADING"),
        SampleKind::DeliveryReceipt => draw_document(&mut canvas, page, "DELIVERY RECEIPT"),
    }
    if !page.is_blank() {
        canvas.text(
            4.0,
            105.0,
            1.0,
            Ink::Gray,
            "TRENOVA TEST SCANNER - SAMPLE PAGE, NOT A REAL DOCUMENT",
        );
    }
    canvas.into_raster(pixel_type)
}

fn draw_document(canvas: &mut Canvas, page: &SamplePage, title: &str) {
    canvas.fill(4.0, 4.0, 77.0, 7.0, Ink::Blue);
    canvas.text(6.0, 5.5, 4.0, Ink::Black, title);
    canvas.text(56.0, 13.0, 1.6, Ink::Black, &format!("PRO {}", page.pro));
    canvas.text(
        56.0,
        16.0,
        1.4,
        Ink::Black,
        &format!("PAGE {} OF {}", page.page, page.of),
    );
    if page.page == 1 {
        canvas.barcode(6.0, 13.0, 4.0, &page.pro);
    }

    let parties = [
        (
            "SHIPPER",
            "ACME FREIGHT CO",
            "1200 INDUSTRIAL WAY",
            "DALLAS TX 75201",
        ),
        (
            "CONSIGNEE",
            "NORTHWIND FOODS",
            "88 HARBOR ROAD",
            "TACOMA WA 98421",
        ),
    ];
    for (column, (label, name, street, city)) in parties.iter().enumerate() {
        #[allow(clippy::cast_precision_loss)]
        let x = 4.0 + 39.0 * column as f32;
        canvas.frame(x, 20.0, 38.0, 13.0, 0.15, Ink::Black);
        canvas.text(x + 1.0, 21.0, 1.2, Ink::Gray, label);
        canvas.text(x + 1.0, 24.0, 1.5, Ink::Black, name);
        canvas.text(x + 1.0, 27.0, 1.3, Ink::Black, street);
        canvas.text(x + 1.0, 29.5, 1.3, Ink::Black, city);
    }

    canvas.frame(4.0, 36.0, 77.0, 40.0, 0.15, Ink::Black);
    let heads = ["QTY", "DESCRIPTION", "WEIGHT", "CLASS"];
    let columns = [5.0, 15.0, 55.0, 70.0];
    for (head, x) in heads.iter().zip(columns) {
        canvas.text(x, 37.5, 1.3, Ink::Black, head);
    }
    canvas.fill(4.0, 40.0, 77.0, 0.15, Ink::Black);
    let items = [
        ("12", "PALLETS CANNED GOODS", "14400 LB", "55"),
        ("4", "PALLETS PAPER PRODUCTS", "2600 LB", "70"),
        ("1", "CRATE DISPLAY FIXTURES", "380 LB", "85"),
    ];
    let offset = page.page.saturating_sub(1) as usize;
    for (row, item) in items.iter().cycle().skip(offset).take(6).enumerate() {
        #[allow(clippy::cast_precision_loss)]
        let y = 42.0 + 5.5 * row as f32;
        for (value, x) in [item.0, item.1, item.2, item.3].iter().zip(columns) {
            canvas.text(x, y, 1.4, Ink::Black, value);
        }
        canvas.fill(4.0, y + 3.8, 77.0, 0.05, Ink::Gray);
    }

    canvas.frame(4.0, 80.0, 38.0, 16.0, 0.15, Ink::Black);
    canvas.text(5.0, 81.0, 1.2, Ink::Gray, "RECEIVED BY");
    let mut x = 7.0;
    for step in 0..24 {
        #[allow(clippy::cast_precision_loss)]
        let lift = ((step * 7) % 5) as f32 * 0.6;
        canvas.fill(x, 89.0 - lift, 1.3, 0.35, Ink::Black);
        x += 1.2;
    }
    canvas.text(44.0, 81.0, 1.2, Ink::Gray, "DATE");
    canvas.text(44.0, 84.0, 1.5, Ink::Black, "09/29/2026");
    if page.kind == SampleKind::DeliveryReceipt {
        canvas.frame(55.0, 86.0, 24.0, 8.0, 0.4, Ink::Red);
        canvas.text(57.5, 88.5, 2.4, Ink::Red, "RECEIVED");
    }
}

fn draw_terms(canvas: &mut Canvas, page: &SamplePage) {
    canvas.text(4.0, 5.0, 2.5, Ink::Black, "TERMS AND CONDITIONS");
    canvas.text(56.0, 5.5, 1.4, Ink::Black, &format!("PRO {}", page.pro));
    for line in 0..40u16 {
        let width = 60.0 + f32::from((line * 13) % 17);
        canvas.fill(4.0, 11.0 + 2.2 * f32::from(line), width, 0.6, Ink::Gray);
    }
}

fn draw_patch(canvas: &mut Canvas) {
    for bar in 0..4u8 {
        canvas.fill(10.0 + 14.0 * f32::from(bar), 10.0, 5.0, 60.0, Ink::Black);
    }
    canvas.text(10.0, 76.0, 6.0, Ink::Black, "PATCH T");
    canvas.text(
        10.0,
        86.0,
        1.4,
        Ink::Black,
        "THE PAGES BEFORE AND AFTER THIS SHEET ARE SEPARATE DOCUMENTS",
    );
}

/// The 5 × 7 font: each row's five bits, most significant on the left.
fn glyph(c: char) -> [u8; 7] {
    match c {
        'A' => [14, 17, 17, 31, 17, 17, 17],
        'B' => [30, 17, 17, 30, 17, 17, 30],
        'C' => [14, 17, 16, 16, 16, 17, 14],
        'D' => [30, 17, 17, 17, 17, 17, 30],
        'E' => [31, 16, 16, 30, 16, 16, 31],
        'F' => [31, 16, 16, 30, 16, 16, 16],
        'G' => [14, 17, 16, 23, 17, 17, 15],
        'H' => [17, 17, 17, 31, 17, 17, 17],
        'I' => [14, 4, 4, 4, 4, 4, 14],
        'J' => [7, 2, 2, 2, 2, 18, 12],
        'K' => [17, 18, 20, 24, 20, 18, 17],
        'L' => [16, 16, 16, 16, 16, 16, 31],
        'M' => [17, 27, 21, 21, 17, 17, 17],
        'N' => [17, 17, 25, 21, 19, 17, 17],
        'O' => [14, 17, 17, 17, 17, 17, 14],
        'P' => [30, 17, 17, 30, 16, 16, 16],
        'Q' => [14, 17, 17, 17, 21, 18, 13],
        'R' => [30, 17, 17, 30, 20, 18, 17],
        'S' => [15, 16, 16, 14, 1, 1, 30],
        'T' => [31, 4, 4, 4, 4, 4, 4],
        'U' => [17, 17, 17, 17, 17, 17, 14],
        'V' => [17, 17, 17, 17, 17, 10, 4],
        'W' => [17, 17, 17, 21, 21, 21, 10],
        'X' => [17, 17, 10, 4, 10, 17, 17],
        'Y' => [17, 17, 10, 4, 4, 4, 4],
        'Z' => [31, 1, 2, 4, 8, 16, 31],
        '0' => [14, 17, 19, 21, 25, 17, 14],
        '1' => [4, 12, 4, 4, 4, 4, 14],
        '2' => [14, 17, 1, 2, 4, 8, 31],
        '3' => [31, 2, 4, 2, 1, 17, 14],
        '4' => [2, 6, 10, 18, 31, 2, 2],
        '5' => [31, 16, 30, 1, 1, 17, 14],
        '6' => [6, 8, 16, 30, 17, 17, 14],
        '7' => [31, 1, 2, 4, 8, 8, 8],
        '8' => [14, 17, 17, 14, 17, 17, 14],
        '9' => [14, 17, 17, 15, 1, 2, 12],
        '-' => [0, 0, 0, 31, 0, 0, 0],
        ':' => [0, 12, 12, 0, 12, 12, 0],
        '/' => [1, 1, 2, 4, 8, 16, 16],
        '.' => [0, 0, 0, 0, 0, 12, 12],
        ',' => [0, 0, 0, 0, 12, 4, 8],
        '#' => [10, 10, 31, 10, 31, 10, 10],
        '&' => [12, 18, 20, 8, 21, 18, 13],
        _ => [0; 7],
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    fn bol(page: u32) -> SamplePage {
        SamplePage {
            kind: SampleKind::BillOfLading,
            pro: "1042".into(),
            page,
            of: 2,
        }
    }

    #[test]
    fn a_page_is_letter_sized_at_its_resolution_in_each_pixel_type() {
        let gray = draw(&bol(1), 100, PixelType::Grayscale);
        assert_eq!((gray.width, gray.height), (850, 1100));
        assert_eq!(gray.format, PixelFormat::Gray8);
        assert!(gray.data.contains(&20), "something is printed on it");

        let colour = draw(&bol(1), 100, PixelType::Color);
        assert_eq!(colour.format, PixelFormat::Rgb8);
        assert!(
            colour.data.chunks_exact(3).any(|p| p == Ink::Blue.rgb()),
            "the header is blue"
        );

        let bilevel = draw(&bol(1), 100, PixelType::BlackWhite);
        assert_eq!(
            bilevel.format,
            PixelFormat::Bilevel {
                zero_is_black: true
            }
        );
        assert!(bilevel.as_raster().is_ok());
    }

    #[test]
    fn resolution_is_kept_to_what_reads_well() {
        assert_eq!(draw(&bol(1), 1200, PixelType::Grayscale).width, 2550);
        assert_eq!(draw(&bol(1), 10, PixelType::Grayscale).width, 637);
    }

    #[test]
    fn a_blank_page_is_white_and_only_first_pages_carry_a_barcode() {
        let blank = SamplePage {
            kind: SampleKind::Blank,
            pro: "1042".into(),
            page: 2,
            of: 2,
        };
        assert!(
            draw(&blank, 75, PixelType::Grayscale)
                .data
                .iter()
                .all(|&v| v == 255)
        );
        assert_eq!(bol(1).barcode().as_deref(), Some("PRO 1042"));
        assert_eq!(bol(2).barcode(), None);
    }

    #[test]
    fn the_font_covers_what_the_pages_say() {
        for c in "ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789-:/.,#&".chars() {
            assert_ne!(glyph(c), [0; 7], "{c} has a glyph");
        }
    }
}
