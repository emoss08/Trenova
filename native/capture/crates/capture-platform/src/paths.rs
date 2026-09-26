//! Where the agent keeps its files.

use std::path::PathBuf;

use windows::Win32::System::Com::CoTaskMemFree;
use windows::Win32::UI::Shell::{FOLDERID_LocalAppData, KF_FLAG_CREATE, SHGetKnownFolderPath};

use crate::wide::from_pwstr;

/// `%LOCALAPPDATA%\Trenova\Capture`: per user, never roamed, which is right
/// for a spool of scanned pages.
pub fn data_dir() -> std::io::Result<PathBuf> {
    // SAFETY: Windows allocates the path; it is freed below.
    let path = unsafe { SHGetKnownFolderPath(&FOLDERID_LocalAppData, KF_FLAG_CREATE, None) }
        .map_err(std::io::Error::other)?;
    // SAFETY: a NUL-terminated path Windows returned; freed once.
    let local = unsafe {
        let local = from_pwstr(path);
        CoTaskMemFree(Some(path.0.cast()));
        local
    };
    Ok(PathBuf::from(local).join("Trenova").join("Capture"))
}
