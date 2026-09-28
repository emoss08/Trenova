//! The Trenova mark, from the web app's `logo.ico`.
//!
//! An `.ico` file is a directory of images, each a PNG or a bitmap. The tray
//! wants the one nearest its small-icon size, which Windows scales with the
//! display, so the image is picked here and handed to Windows as is.

/// The icon file, compiled in from the one copy the web app ships.
pub const LOGO: &[u8] = include_bytes!("../../../../../client/apps/web/public/logo.ico");

/// One image in an icon file.
#[derive(Clone, Copy, Debug, PartialEq, Eq)]
pub struct IconImage<'a> {
    pub size: u32,
    pub bytes: &'a [u8],
}

/// Every image in an `.ico` file, or `None` if it is not one.
pub fn images(ico: &[u8]) -> Option<Vec<IconImage<'_>>> {
    let header = ico.get(..6)?;
    if header[..4] != [0, 0, 1, 0] {
        return None;
    }
    let count = usize::from(u16::from_le_bytes([header[4], header[5]]));
    let mut found = Vec::with_capacity(count);
    for i in 0..count {
        let entry = ico.get(6 + i * 16..22 + i * 16)?;
        let size = if entry[0] == 0 {
            256
        } else {
            u32::from(entry[0])
        };
        let length = u32::from_le_bytes([entry[8], entry[9], entry[10], entry[11]]) as usize;
        let offset = u32::from_le_bytes([entry[12], entry[13], entry[14], entry[15]]) as usize;
        let bytes = ico.get(offset..offset.checked_add(length)?)?;
        found.push(IconImage { size, bytes });
    }
    Some(found)
}

/// The image to draw at `size` pixels: the smallest at least that big, else
/// the biggest there is.
pub fn best_for(ico: &[u8], size: u32) -> Option<IconImage<'_>> {
    let images = images(ico)?;
    images
        .iter()
        .filter(|i| i.size >= size)
        .min_by_key(|i| i.size)
        .or_else(|| images.iter().max_by_key(|i| i.size))
        .copied()
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn the_logo_carries_every_tray_size() {
        let sizes: Vec<u32> = images(LOGO)
            .expect("an icon")
            .iter()
            .map(|i| i.size)
            .collect();
        for size in [16, 24, 32] {
            assert!(sizes.contains(&size), "missing {size}px");
        }
        assert_eq!(best_for(LOGO, 20).map(|i| i.size), Some(24));
        assert_eq!(best_for(LOGO, 512).map(|i| i.size), Some(256));
    }

    #[test]
    fn a_file_that_is_not_an_icon_is_refused() {
        assert_eq!(images(b"%PDF-1.7"), None);
        let mut truncated = LOGO[..40].to_vec();
        truncated.truncate(30);
        assert_eq!(images(&truncated), None);
    }
}
