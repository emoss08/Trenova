//! What a scan is asked for and what it hands back, whichever protocol ran
//! it. TWAIN and WIA both produce these, so the helper treats them alike.

use capture_protocol::api::PixelType;
use capture_protocol::helper::ScanCondition;

use crate::page::Resolution;
use crate::raster::OwnedRaster;

/// What a scan was asked for.
#[derive(Clone, Copy, Debug, PartialEq, Eq)]
pub struct ScanSettings {
    pub dpi: u32,
    pub pixel_type: PixelType,
    pub duplex: bool,
    pub use_feeder: bool,
    pub discard_blank_pages: bool,
    pub show_ui: bool,
    pub detect_patch_codes: bool,
    pub detect_barcodes: bool,
}

/// One page, as transferred.
#[derive(Clone, Debug, PartialEq, Eq)]
pub struct ScannedPage {
    pub raster: OwnedRaster,
    pub resolution: Resolution,
    pub patch_code: Option<&'static str>,
    pub barcodes: Vec<String>,
}

/// How a scan ended.
#[derive(Clone, Copy, Debug, PartialEq, Eq)]
pub enum ScanEnd {
    Finished {
        pages: u32,
    },
    /// The source stopped early. The pages delivered before stand.
    Stopped {
        condition: ScanCondition,
        pages: u32,
    },
    /// The agent asked to stop.
    Canceled {
        pages: u32,
    },
}
