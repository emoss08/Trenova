//! UTF-16 strings for Win32.

use windows_core::PWSTR;

/// A NUL-terminated UTF-16 copy of `value`.
pub(crate) fn wide(value: &str) -> Vec<u16> {
    value.encode_utf16().chain(Some(0)).collect()
}

/// Reads a NUL-terminated UTF-16 string Windows returned.
///
/// # Safety
///
/// `value` is null or points at a NUL-terminated UTF-16 string.
pub(crate) unsafe fn from_pwstr(value: PWSTR) -> String {
    if value.is_null() {
        return String::new();
    }
    // SAFETY: per the caller.
    unsafe { value.to_string() }.unwrap_or_default()
}

/// The UTF-16 in `buffer` up to its first NUL.
pub(crate) fn from_buffer(buffer: &[u16]) -> String {
    let end = buffer.iter().position(|&c| c == 0).unwrap_or(buffer.len());
    String::from_utf16_lossy(&buffer[..end])
}
