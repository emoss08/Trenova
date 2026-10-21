//! The page imaging Trenova runs wherever a page is: rasters and their luma,
//! thresholds and ink coverage, the previews a person looks at, JPEG and CCITT
//! Group 4 encoding, and PDFs whose pages are those images.
//!
//! It is shared by the capture companion (`desktop/capture`), which encodes
//! scanned and printed pages where they are produced, and the imaging worker
//! (`services/imaging`), which inspects captured pages and turns images into
//! PDFs on the server. One implementation means the companion and the server
//! measure a page the same way. It has no platform code and no native
//! dependencies, so it builds for Windows, Linux and WebAssembly alike.

pub mod ink;
pub mod page;
pub mod preview;
pub mod raster;

pub use ink::{InkParams, ink_coverage};
pub use page::{
    EncodedPage, JpegColor, PageFit, PassthroughJpeg, PdfDocument, Resolution, encode_page,
};
pub use preview::{PagePreview, preview};
pub use raster::{OwnedRaster, PixelFormat, Raster, binarize, binarize_at, luma, to_gray};

#[derive(Debug, thiserror::Error, PartialEq, Eq)]
pub enum ImagingError {
    #[error("a {width} × {height} page is not a page this can encode")]
    Dimensions { width: u32, height: u32 },
    #[error("a row stride of {stride} bytes is shorter than a row of {row_bytes}")]
    Stride { stride: usize, row_bytes: usize },
    #[error("the image needs {needed} bytes but has {actual}")]
    ShortBuffer { needed: usize, actual: usize },
    #[error("a resolution of {x} × {y} DPI is outside what a scanner produces")]
    Resolution { x: u32, y: u32 },
    #[error("JPEG quality {0} is outside 30 to 95")]
    Quality(u8),
    #[error("JPEG encoding failed: {0}")]
    Jpeg(String),
    #[error("a page of {width} × {height} points is not a page")]
    PageSize { width: u32, height: u32 },
}
