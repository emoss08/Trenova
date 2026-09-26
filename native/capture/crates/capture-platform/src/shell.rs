//! Opening pages in the person's browser.

use windows::Win32::UI::Shell::ShellExecuteW;
use windows::Win32::UI::WindowsAndMessaging::SW_SHOWNORMAL;
use windows_core::{PCWSTR, w};

use crate::wide::wide;

/// Opens a web address in the default browser. Anything but an http(s)
/// address is refused: this is only ever handed links to Trenova, and
/// `ShellExecute` would run a file path just as happily.
pub fn open_url(url: &str) -> std::io::Result<()> {
    let lower = url.trim().to_ascii_lowercase();
    if !(lower.starts_with("https://") || lower.starts_with("http://")) {
        return Err(std::io::Error::other("only web addresses are opened"));
    }
    let target = wide(url.trim());
    // SAFETY: NUL-terminated strings; no window owns the browser.
    let result = unsafe {
        ShellExecuteW(
            None,
            w!("open"),
            PCWSTR(target.as_ptr()),
            PCWSTR::null(),
            PCWSTR::null(),
            SW_SHOWNORMAL,
        )
    };
    // ShellExecute reports success as a value above 32.
    if result.0 as usize > 32 {
        Ok(())
    } else {
        Err(std::io::Error::other("the browser could not be opened"))
    }
}

/// Opens one of the agent's own folders in Explorer. Only an existing
/// directory is opened, never a file.
pub fn open_folder(path: &std::path::Path) -> std::io::Result<()> {
    if !path.is_dir() {
        return Err(std::io::Error::other("not a folder"));
    }
    let target = wide(&path.to_string_lossy());
    // SAFETY: NUL-terminated strings; Explorer opens the folder.
    let result = unsafe {
        ShellExecuteW(
            None,
            w!("explore"),
            PCWSTR(target.as_ptr()),
            PCWSTR::null(),
            PCWSTR::null(),
            SW_SHOWNORMAL,
        )
    };
    if result.0 as usize > 32 {
        Ok(())
    } else {
        Err(std::io::Error::other("the folder could not be opened"))
    }
}
