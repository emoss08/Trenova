//! Writes one page of each kind, and a printed two-page PWG raster job
//! converted to PDF, to a directory, so the output can be read back by an
//! independent PDF reader: `cargo run -p capture-imaging --example
//! sample_pages -- <dir>`.

use std::path::PathBuf;

use capture_imaging::pwg::encode::{Kind, Page, stream};
use capture_imaging::{ConvertLimits, PixelFormat, Raster, Resolution, encode_page, pwg_to_pdf};

fn main() -> Result<(), Box<dyn std::error::Error>> {
    let dir = PathBuf::from(std::env::args().nth(1).ok_or("usage: sample_pages <dir>")?);
    std::fs::create_dir_all(&dir)?;

    let (width, height) = (850u32, 1100u32);
    let format = PixelFormat::Bilevel {
        zero_is_black: true,
    };
    let stride = format.row_bytes(width);
    let mut bilevel = vec![0xFFu8; stride * height as usize];
    for y in 0..height {
        for x in 0..width {
            if x < 20 || y < 20 || x == y || (x / 50 + y / 50) % 7 == 0 {
                bilevel[stride * y as usize + x as usize / 8] &= !(0x80 >> (x % 8));
            }
        }
    }
    let page = encode_page(
        &Raster::new(width, height, stride, format, &bilevel)?,
        Resolution::square(100),
        80,
    )?;
    std::fs::write(dir.join("bilevel.pdf"), &page.pdf)?;

    let mut rgb = Vec::with_capacity(width as usize * height as usize * 3);
    for y in 0..height {
        for x in 0..width {
            rgb.extend_from_slice(&[u8::try_from(x % 256)?, u8::try_from(y % 256)?, 128]);
        }
    }
    let page = encode_page(
        &Raster::new(width, height, width as usize * 3, PixelFormat::Rgb8, &rgb)?,
        Resolution::square(100),
        90,
    )?;
    std::fs::write(dir.join("colour.pdf"), &page.pdf)?;

    let ink: Vec<u8> = bilevel.iter().map(|byte| !byte).collect();
    let job = stream(&[
        Page {
            kind: Kind::Black1,
            width,
            height,
            dpi: 100,
            pixels: &ink,
        },
        Page {
            kind: Kind::Srgb8,
            width,
            height,
            dpi: 100,
            pixels: &rgb,
        },
    ]);
    let printed = pwg_to_pdf(
        &job,
        ConvertLimits {
            max_pages: 10,
            max_bytes: 10 << 20,
            jpeg_quality: 90,
        },
    )?;
    std::fs::write(dir.join("printed.pdf"), &printed.pdf)?;
    Ok(())
}
