//! Where the agent and the print service keep their files.

use std::path::PathBuf;

use windows::Win32::System::Com::CoTaskMemFree;
use windows::Win32::UI::Shell::{
    FOLDERID_LocalAppData, FOLDERID_ProgramData, KF_FLAG_CREATE, KNOWN_FOLDER_FLAG,
    SHGetKnownFolderPath,
};
use windows_core::GUID;

use crate::wide::from_pwstr;

fn known_folder(id: &GUID, flags: KNOWN_FOLDER_FLAG) -> std::io::Result<PathBuf> {
    // SAFETY: Windows allocates the path; it is freed below.
    let path = unsafe { SHGetKnownFolderPath(id, flags, None) }.map_err(std::io::Error::other)?;
    // SAFETY: a NUL-terminated path Windows returned; freed once.
    let folder = unsafe {
        let folder = from_pwstr(path);
        CoTaskMemFree(Some(path.0.cast()));
        folder
    };
    Ok(PathBuf::from(folder).join("Trenova").join("Capture"))
}

/// `%LOCALAPPDATA%\Trenova\Capture`: per user, never roamed, which is right
/// for a spool of scanned pages.
pub fn data_dir() -> std::io::Result<PathBuf> {
    known_folder(&FOLDERID_LocalAppData, KF_FLAG_CREATE)
}

/// `%ProgramData%\Trenova\Capture`: the print service's, created by the
/// installer.
pub fn shared_dir() -> std::io::Result<PathBuf> {
    known_folder(&FOLDERID_ProgramData, KNOWN_FOLDER_FLAG(0))
}

/// `%ProgramData%\Trenova\Capture\spool`: one inbox per Windows user, named
/// by SID, where the print service leaves what that user printed.
pub fn print_spool_dir() -> std::io::Result<PathBuf> {
    Ok(shared_dir()?.join("spool"))
}
