//! Turns what a scanner hands over into what the server takes: one PDF per
//! page, bilevel pages as CCITT Group 4 and the rest as JPEG.
//!
//! It runs inside the scan helper, so raw bitmaps are encoded where they are
//! produced and never cross a process boundary.

pub mod dib;
pub mod page;
pub mod raster;
pub mod scan;

pub use dib::{Bitmap, decode_bmp};
pub use page::{EncodedPage, Resolution, encode_page};
pub use raster::{OwnedRaster, PixelFormat, Raster, binarize};

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
    #[error("unreadable bitmap: {0}")]
    Dib(&'static str),
}
