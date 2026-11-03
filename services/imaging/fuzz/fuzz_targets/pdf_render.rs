//! Arbitrary bytes drawn as a PDF's first pages, small: each page is drawn or refused on its
//! own, within the pixel budget, and the document never panics the engine.

#![no_main]

#[path = "pdf_engine.rs"]
mod pdf_engine;

use libfuzzer_sys::fuzz_target;
use pdf_engine::{Kept, LIMITS, pdf, with_engine};
use trenova_imaging::engine::render::render;
use trenova_proto::imaging::v1::{ImageFormat, RenderHeader, RenderSize, render_size};

fuzz_target!(|data: &[u8]| {
    with_engine(|engine| {
        let header = RenderHeader {
            document: None,
            pages: vec![0, 1, 2],
            size: Some(RenderSize {
                size: Some(render_size::Size::Dpi(24)),
            }),
            format: ImageFormat::Png as i32,
            quality: 0,
            lossless: false,
            max_pixels: 0,
            max_pages: 3,
        };
        let mut kept = Kept::default();
        if let Ok(summary) = render(engine, &header, pdf(data), LIMITS, &mut kept) {
            assert!(summary.rendered <= 3);
            assert!(summary.rendered <= summary.page_count);
        }
    });
});
