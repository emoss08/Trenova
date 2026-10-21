//! The page formats only the capture companion meets: Windows DIBs from WIA,
//! PWG raster from the Trenova printer, the scan helper's page hand-off and
//! the test scanner's sample pages.
//!
//! The imaging itself (rasters, binarizing, previews, G4 and JPEG encoding
//! and the PDFs pages travel as) is `trenova-imaging-core`, shared with the
//! server's imaging worker so both read a page the same way.

pub mod dib;
pub mod pwg;
pub mod sample;
pub mod scan;

pub use dib::{Bitmap, DibError, decode_bmp};
pub use pwg::{ConvertLimits, PrintedDocument, PwgError, PwgPage, PwgReader, pwg_to_pdf};
