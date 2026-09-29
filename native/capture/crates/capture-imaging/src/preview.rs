//! Pictures of a page for a person to look at before it is sent: a small one
//! for a strip of pages and a larger one to read it by.
//!
//! They are made from the raster where it exists, in the scan helper and the
//! print service, because the page itself travels as a PDF whose G4 image a
//! browser cannot draw. Both are baseline JPEG, which the window shows as a
//! data URI; black and white pages become gray, since a reduced bilevel page
//! reads better with the in-between shades an average gives it.

use jpeg_encoder::ColorType;

use crate::ImagingError;
use crate::page::{Resolution, encode_jpeg};
use crate::raster::{OwnedRaster, PixelFormat, Raster};

/// The longest side of the small picture, in pixels: two columns of it fit
/// the window at a high-DPI scale.
pub const THUMB_SIDE: u32 = 240;
/// The longest side of the large picture: a letter page at this size reads
/// comfortably on a laptop screen.
pub const VIEW_SIDE: u32 = 1100;
const QUALITY: u8 = 70;
/// JPEG density is meaningless for a picture on screen; this is the
/// conventional one.
const SCREEN: Resolution = Resolution { x: 96, y: 96 };

/// A page's two pictures, as JPEG.
#[derive(Clone, Debug, PartialEq, Eq)]
pub struct PagePreview {
    pub thumb: Vec<u8>,
    pub view: Vec<u8>,
}

/// Makes both pictures of a page.
pub fn preview(raster: &Raster<'_>) -> Result<PagePreview, ImagingError> {
    let view = reduce(raster, VIEW_SIDE);
    let view_raster = view.as_raster()?;
    let thumb = reduce(&view_raster, THUMB_SIDE);
    Ok(PagePreview {
        thumb: encode(&thumb)?,
        view: encode(&view)?,
    })
}

fn encode(raster: &OwnedRaster) -> Result<Vec<u8>, ImagingError> {
    let color = match raster.format {
        PixelFormat::Rgb8 => ColorType::Rgb,
        _ => ColorType::Luma,
    };
    encode_jpeg(&raster.as_raster()?, color, QUALITY, SCREEN)
}

/// The size a page is reduced to so its longest side is at most `side`;
/// never enlarged.
fn target(width: u32, height: u32, side: u32) -> (u32, u32) {
    let longest = width.max(height);
    if longest <= side {
        return (width, height);
    }
    let scale = |length: u32| {
        let scaled = u64::from(length) * u64::from(side) / u64::from(longest);
        u32::try_from(scaled).unwrap_or(side).max(1)
    };
    (scale(width), scale(height))
}

/// Reduces a page by averaging each block of source pixels into one, in one
/// pass over the source: gray for bilevel and gray pages, RGB for colour.
fn reduce(raster: &Raster<'_>, side: u32) -> OwnedRaster {
    let (width, height) = target(raster.width, raster.height, side);
    let channels: usize = match raster.format {
        PixelFormat::Rgb8 | PixelFormat::Bgr8 => 3,
        PixelFormat::Bilevel { .. } | PixelFormat::Gray8 => 1,
    };
    let out_width = width as usize;
    let column_of: Vec<usize> = (0..raster.width)
        .map(|x| {
            usize::try_from(u64::from(x) * u64::from(width) / u64::from(raster.width)).unwrap_or(0)
        })
        .collect();
    let mut column_count = vec![0u32; out_width];
    for &column in &column_of {
        column_count[column] += 1;
    }

    let mut data = Vec::with_capacity(out_width * channels * height as usize);
    let mut sums = vec![0u32; out_width * channels];
    let mut rows_in_block = 0u32;
    let mut block = 0u32;
    for y in 0..raster.height {
        let out_row = u32::try_from(u64::from(y) * u64::from(height) / u64::from(raster.height))
            .unwrap_or(height - 1);
        if out_row != block {
            flush(&mut data, &mut sums, &column_count, rows_in_block, channels);
            rows_in_block = 0;
            block = out_row;
        }
        accumulate(raster.row(y), raster.format, &column_of, &mut sums);
        rows_in_block += 1;
    }
    flush(&mut data, &mut sums, &column_count, rows_in_block, channels);

    OwnedRaster {
        width,
        height,
        stride: out_width * channels,
        format: if channels == 3 {
            PixelFormat::Rgb8
        } else {
            PixelFormat::Gray8
        },
        data,
    }
}

/// Adds one source row into the sums of the output columns it falls in.
fn accumulate(row: &[u8], format: PixelFormat, column_of: &[usize], sums: &mut [u32]) {
    match format {
        PixelFormat::Bilevel { zero_is_black } => {
            for (x, &column) in column_of.iter().enumerate() {
                let bit = row[x / 8] >> (7 - (x % 8)) & 1;
                let white = (bit == 1) == zero_is_black;
                if white {
                    sums[column] += 255;
                }
            }
        }
        PixelFormat::Gray8 => {
            for (&value, &column) in row.iter().zip(column_of) {
                sums[column] += u32::from(value);
            }
        }
        PixelFormat::Rgb8 | PixelFormat::Bgr8 => {
            let swap = format == PixelFormat::Bgr8;
            for (pixel, &column) in row.chunks_exact(3).zip(column_of) {
                let (r, g, b) = if swap {
                    (pixel[2], pixel[1], pixel[0])
                } else {
                    (pixel[0], pixel[1], pixel[2])
                };
                let at = column * 3;
                sums[at] += u32::from(r);
                sums[at + 1] += u32::from(g);
                sums[at + 2] += u32::from(b);
            }
        }
    }
}

/// Writes one output row, the average of its block, and clears the sums.
fn flush(data: &mut Vec<u8>, sums: &mut [u32], column_count: &[u32], rows: u32, channels: usize) {
    if rows == 0 {
        return;
    }
    for (index, sum) in sums.iter_mut().enumerate() {
        let count = column_count[index / channels] * rows;
        let average = if count == 0 {
            255
        } else {
            (*sum + count / 2) / count
        };
        data.push(u8::try_from(average).unwrap_or(u8::MAX));
        *sum = 0;
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    fn is_jpeg(bytes: &[u8]) -> bool {
        bytes.starts_with(&[0xFF, 0xD8]) && bytes.ends_with(&[0xFF, 0xD9])
    }

    #[test]
    fn a_page_is_reduced_to_the_longest_side_and_never_enlarged() {
        assert_eq!(target(2550, 3300, VIEW_SIDE), (850, 1100));
        assert_eq!(target(3300, 2550, THUMB_SIDE), (240, 185));
        assert_eq!(target(200, 100, THUMB_SIDE), (200, 100));
        assert_eq!(target(10_000, 3, THUMB_SIDE), (240, 1));
    }

    #[test]
    fn bilevel_averages_to_gray_either_way_round() {
        let data = [0b1010_1010u8, 0b0101_0101];
        for zero_is_black in [true, false] {
            let raster = Raster::new(8, 2, 1, PixelFormat::Bilevel { zero_is_black }, &data)
                .expect("raster");
            let reduced = reduce(&raster, 1);
            assert_eq!((reduced.width, reduced.height), (1, 1));
            assert_eq!(reduced.format, PixelFormat::Gray8);
            assert_eq!(reduced.data, [128], "half black, half white");
        }
    }

    #[test]
    fn a_white_page_stays_white_and_black_stays_black() {
        let white = [0xFFu8; 4 * 4];
        let raster = Raster::new(
            32,
            4,
            4,
            PixelFormat::Bilevel {
                zero_is_black: true,
            },
            &white,
        )
        .expect("raster");
        assert!(reduce(&raster, 8).data.iter().all(|&v| v == 255));
        let raster = Raster::new(
            32,
            4,
            4,
            PixelFormat::Bilevel {
                zero_is_black: false,
            },
            &white,
        )
        .expect("raster");
        assert!(reduce(&raster, 8).data.iter().all(|&v| v == 0));
    }

    #[test]
    fn bgr_is_read_as_bgr() {
        let data = [0u8, 0, 255, 0, 0, 255];
        let raster = Raster::new(2, 1, 6, PixelFormat::Bgr8, &data).expect("raster");
        let reduced = reduce(&raster, 1);
        assert_eq!(reduced.format, PixelFormat::Rgb8);
        assert_eq!(reduced.data, [255, 0, 0], "a red page stays red");
    }

    #[test]
    fn padded_rows_ignore_their_padding() {
        let data = [10u8, 20, 99, 99, 30, 40, 99, 99];
        let raster = Raster::new(2, 2, 4, PixelFormat::Gray8, &data).expect("raster");
        assert_eq!(reduce(&raster, 1).data, [25]);
    }

    #[test]
    fn a_letter_page_makes_two_jpegs() {
        let (width, height) = (2550u32, 3300u32);
        let data: Vec<u8> = (0..width * height)
            .map(|i| u8::try_from(i % 251).unwrap_or(0))
            .collect();
        let raster =
            Raster::new(width, height, width as usize, PixelFormat::Gray8, &data).expect("raster");
        let made = preview(&raster).expect("preview");
        assert!(is_jpeg(&made.thumb));
        assert!(is_jpeg(&made.view));
        assert!(made.thumb.len() < made.view.len());
    }
}
