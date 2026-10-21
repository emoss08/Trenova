//! How much of a page carries ink, which is how a blank sheet in a batch is
//! told from a page with a signature on it.
//!
//! The measure samples the page inside its margins, because scanners leave
//! shadows, punch holes and the edge of the platen at the borders of an
//! otherwise blank sheet, and counts the share of samples darker than a
//! threshold that scanner noise and tinted paper sit well above.

use crate::raster::{PixelFormat, Raster, luma};

/// How a page is sampled for ink.
#[derive(Clone, Copy, Debug, PartialEq)]
pub struct InkParams {
    /// The luma (0 black to 255 white) below which a sample counts as ink.
    pub threshold: u8,
    /// The share of the width and of the height left out on every side.
    pub margin_fraction: f32,
    /// Every `stride`-th pixel of every `stride`-th row is sampled. A blank
    /// page is blank at every other pixel too, and a signature still leaves
    /// hundreds of dark samples.
    pub stride: u32,
}

impl InkParams {
    /// What the server has always measured a captured page with: a page
    /// rendered at 150 DPI, sampled at every other pixel inside 6% margins.
    pub const SERVER: Self = Self {
        threshold: 160,
        margin_fraction: 0.06,
        stride: 2,
    };
}

impl Default for InkParams {
    fn default() -> Self {
        Self::SERVER
    }
}

/// The share of sampled pixels inside the margins that are dark enough to be
/// content, from 0 (nothing) to 1 (solid ink). A page too small to have an
/// inside measures 0.
#[allow(
    clippy::cast_possible_truncation,
    clippy::cast_sign_loss,
    clippy::cast_precision_loss
)]
pub fn ink_coverage(raster: &Raster<'_>, params: InkParams) -> f64 {
    let fraction = params.margin_fraction.clamp(0.0, 0.49);
    let margin_x = (raster.width as f32 * fraction) as u32;
    let margin_y = (raster.height as f32 * fraction) as u32;
    let (min_x, max_x) = (margin_x, raster.width.saturating_sub(margin_x));
    let (min_y, max_y) = (margin_y, raster.height.saturating_sub(margin_y));
    if min_x >= max_x || min_y >= max_y {
        return 0.0;
    }
    let step = params.stride.max(1) as usize;

    let (mut sampled, mut dark) = (0u64, 0u64);
    for y in (min_y..max_y).step_by(step) {
        let row = raster.row(y);
        let columns = (min_x as usize..max_x as usize).step_by(step);
        match raster.format {
            PixelFormat::Gray8 => {
                for x in columns {
                    dark += u64::from(row[x] < params.threshold);
                    sampled += 1;
                }
            }
            PixelFormat::Rgb8 => {
                for x in columns {
                    let at = x * 3;
                    dark += u64::from(luma(row[at], row[at + 1], row[at + 2]) < params.threshold);
                    sampled += 1;
                }
            }
            PixelFormat::Bgr8 => {
                for x in columns {
                    let at = x * 3;
                    dark += u64::from(luma(row[at + 2], row[at + 1], row[at]) < params.threshold);
                    sampled += 1;
                }
            }
            PixelFormat::Bilevel { zero_is_black } => {
                for x in columns {
                    let bit = row[x / 8] >> (7 - (x % 8)) & 1;
                    dark += u64::from((bit == 1) != zero_is_black);
                    sampled += 1;
                }
            }
        }
    }

    if sampled == 0 {
        return 0.0;
    }
    dark as f64 / sampled as f64
}

#[cfg(test)]
mod tests {
    use super::*;
    use proptest::prelude::*;

    fn page(width: u32, height: u32, fill: u8) -> Vec<u8> {
        vec![fill; (width * height) as usize]
    }

    #[test]
    fn a_white_page_has_no_ink_and_a_black_one_is_all_ink() {
        let white = page(200, 260, 250);
        let raster = Raster::new(200, 260, 200, PixelFormat::Gray8, &white).expect("raster");
        assert!(ink_coverage(&raster, InkParams::SERVER).abs() < f64::EPSILON);

        let black = page(200, 260, 10);
        let raster = Raster::new(200, 260, 200, PixelFormat::Gray8, &black).expect("raster");
        assert!((ink_coverage(&raster, InkParams::SERVER) - 1.0).abs() < f64::EPSILON);
    }

    #[test]
    fn ink_in_the_margins_is_not_counted() {
        let (width, height) = (200u32, 200u32);
        let mut data = page(width, height, 255);
        for y in 0..height {
            for x in 0..8 {
                data[(y * width + x) as usize] = 0;
            }
        }
        let raster =
            Raster::new(width, height, width as usize, PixelFormat::Gray8, &data).expect("raster");
        assert!(ink_coverage(&raster, InkParams::SERVER).abs() < f64::EPSILON);
    }

    #[test]
    fn bilevel_and_colour_pages_read_like_their_gray_equivalent() {
        let rgb = [0u8, 0, 0].repeat(100 * 100);
        let raster = Raster::new(100, 100, 300, PixelFormat::Rgb8, &rgb).expect("raster");
        assert!((ink_coverage(&raster, InkParams::SERVER) - 1.0).abs() < f64::EPSILON);

        let chocolate = vec![0u8; 13 * 100];
        let raster = Raster::new(
            100,
            100,
            13,
            PixelFormat::Bilevel {
                zero_is_black: true,
            },
            &chocolate,
        )
        .expect("raster");
        assert!((ink_coverage(&raster, InkParams::SERVER) - 1.0).abs() < f64::EPSILON);
    }

    proptest! {
        #[test]
        fn coverage_is_always_a_share(
            width in 1u32..64,
            height in 1u32..64,
            seed in any::<u64>(),
            threshold in any::<u8>(),
            stride in 0u32..5,
        ) {
            let mut state = seed;
            let data: Vec<u8> = (0..width * height)
                .map(|_| {
                    state = state.wrapping_mul(6_364_136_223_846_793_005).wrapping_add(1);
                    (state >> 56) as u8
                })
                .collect();
            let raster = Raster::new(width, height, width as usize, PixelFormat::Gray8, &data)
                .expect("raster");
            let params = InkParams { threshold, margin_fraction: 0.06, stride };
            let coverage = ink_coverage(&raster, params);
            prop_assert!((0.0..=1.0).contains(&coverage));
        }
    }
}
