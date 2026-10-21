//! What a call is expected to cost, for admission, worked out from its header
//! before any bytes arrive.
//!
//! A unit is one child for the length of a job: the pool runs one job per
//! child, so units bound how much work queues for a child. Memory is the
//! input held in its memfd plus the largest page the job will hold decoded
//! at once (RGBA, with room for a resized copy and the encoder's buffers).

use trenova_service_kit::Cost;

/// Bytes per pixel a job holds for the page it is working on.
const WORKING_BYTES_PER_PIXEL: u64 = 12;
/// What a child's libraries hold besides the page.
const BASE_BYTES: u64 = 48 << 20;
/// A letter page in square points.
const LETTER_POINTS: u64 = 612 * 792;

pub fn job(input_bytes: u64, page_pixels: u64) -> Cost {
    Cost {
        units: 1,
        memory_bytes: BASE_BYTES
            + input_bytes
            + page_pixels.saturating_mul(WORKING_BYTES_PER_PIXEL),
    }
}

/// The pixels of a letter page drawn at `dpi`, the size a page is assumed to
/// be before it is opened.
pub fn letter_pixels(dpi: u32) -> u64 {
    let dpi = u64::from(dpi);
    LETTER_POINTS * dpi * dpi / (72 * 72)
}

/// The pixels of a fit box, or a letter page at 300 DPI when a side is open.
pub fn fit_pixels(max_width: u32, max_height: u32) -> u64 {
    match (max_width, max_height) {
        (0, 0) => letter_pixels(300),
        (w, 0) => u64::from(w) * u64::from(w) * 11 / 8,
        (0, h) => u64::from(h) * u64::from(h) * 8 / 11,
        (w, h) => u64::from(w) * u64::from(h),
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn a_page_view_costs_one_child_and_its_page() {
        let cost = job(1 << 20, fit_pixels(1400, 1800));
        assert_eq!(cost.units, 1);
        assert_eq!(cost.memory_bytes, (48 << 20) + (1 << 20) + 1400 * 1800 * 12);
        assert_eq!(letter_pixels(72), LETTER_POINTS);
    }
}
