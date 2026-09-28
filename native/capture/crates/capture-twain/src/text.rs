//! TWAIN's fixed-size strings.
//!
//! Sources write their names in the system's ANSI code page, so on Windows
//! they are decoded through it; elsewhere (the unit tests) they are read as
//! UTF-8. The same decoding is used for listing and for matching a name the
//! web app sends back, so a name round-trips whatever its characters.

use crate::sys::TW_STR32;

/// Reads a NUL-terminated string from a TWAIN string field.
pub fn decode_str(field: &[i8]) -> String {
    let bytes: Vec<u8> = field
        .iter()
        .map(|&c| u8::from_ne_bytes(c.to_ne_bytes()))
        .take_while(|&b| b != 0)
        .collect();
    decode_ansi(&bytes).trim().to_owned()
}

#[cfg(windows)]
fn decode_ansi(bytes: &[u8]) -> String {
    use windows::Win32::Globalization::{
        CP_ACP, MULTI_BYTE_TO_WIDE_CHAR_FLAGS, MultiByteToWideChar,
    };

    if bytes.is_empty() {
        return String::new();
    }
    let mut wide = vec![0u16; bytes.len()];
    // SAFETY: both buffers are valid for their lengths; ANSI never expands
    // past one UTF-16 unit per byte.
    let len = unsafe {
        MultiByteToWideChar(
            CP_ACP,
            MULTI_BYTE_TO_WIDE_CHAR_FLAGS(0),
            bytes,
            Some(&mut wide),
        )
    };
    match usize::try_from(len) {
        Ok(len) if len > 0 => String::from_utf16_lossy(&wide[..len]),
        _ => String::from_utf8_lossy(bytes).into_owned(),
    }
}

#[cfg(not(windows))]
fn decode_ansi(bytes: &[u8]) -> String {
    String::from_utf8_lossy(bytes).into_owned()
}

/// Writes an ASCII string into a `TW_STR32`, truncated to leave its NUL.
/// Only this application's own identity is written this way.
pub fn encode_str32(value: &str) -> TW_STR32 {
    let mut field: TW_STR32 = [0; 34];
    for (slot, byte) in field
        .iter_mut()
        .zip(value.bytes().filter(u8::is_ascii).take(33))
    {
        *slot = i8::from_ne_bytes([byte]);
    }
    field
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn strings_round_trip_and_stop_at_nul() {
        let field = encode_str32("Trenova Capture");
        assert_eq!(decode_str(&field), "Trenova Capture");
        let long = encode_str32(&"x".repeat(40));
        assert_eq!(long[33], 0);
        assert_eq!(decode_str(&long).len(), 33);
    }
}
