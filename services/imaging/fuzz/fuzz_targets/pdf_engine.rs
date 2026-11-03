//! The PDF targets' shared setup: PDFium loaded once from `TRENOVA_PDFIUM_LIBRARY`, as the
//! worker's own tests load it, a sink that keeps what a job emits, and the limits a child runs
//! with.

// Each target is its own crate and uses only part of this setup.
#![allow(dead_code)]

use trenova_imaging::engine::error::EngineResult;
use trenova_imaging::engine::{Engine, Input, Sink};
use trenova_imaging::protocol::{ChildInit, JobLimits};

pub const LIMITS: JobLimits = JobLimits {
    max_pixels: 4_000_000,
    max_pages: 64,
    chunk_bytes: 64 << 10,
    max_image_pixels: 4_000_000,
};

thread_local! {
    static ENGINE: Engine = Engine::new(&ChildInit {
        pdfium_library: std::env::var("TRENOVA_PDFIUM_LIBRARY")
            .expect("set TRENOVA_PDFIUM_LIBRARY to a libpdfium to fuzz the PDF engine"),
        heif_library: String::new(),
        max_frame_bytes: 0,
    })
    .expect("PDFium loads");
}

pub fn with_engine<T>(run: impl FnOnce(&Engine) -> T) -> T {
    ENGINE.with(run)
}

pub fn pdf(bytes: &[u8]) -> Input<'_> {
    Input {
        index: 0,
        content_type: "application/pdf",
        bytes,
    }
}

#[derive(Default)]
pub struct Kept {
    pub messages: usize,
    pub bytes: usize,
}

impl Sink for Kept {
    fn emit(&mut self, message: Vec<u8>) -> EngineResult<()> {
        self.messages += 1;
        self.bytes += message.len();
        Ok(())
    }
}
