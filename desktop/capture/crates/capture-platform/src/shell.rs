//! Opening pages in the person's browser, and the one thing the agent does
//! with administrator rights.

use std::path::Path;

use windows::Win32::Foundation::{CloseHandle, ERROR_CANCELLED, WAIT_OBJECT_0};
use windows::Win32::System::Threading::{GetExitCodeProcess, INFINITE, WaitForSingleObject};
use windows::Win32::UI::Shell::{
    SEE_MASK_NOASYNC, SEE_MASK_NOCLOSEPROCESS, SHELLEXECUTEINFOW, ShellExecuteExW, ShellExecuteW,
};
use windows::Win32::UI::WindowsAndMessaging::{SW_HIDE, SW_SHOWNORMAL};
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

/// How a program run with administrator rights ended.
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum Elevated {
    /// It ran, and exited with this code.
    Exited(u32),
    /// The person said no at the Windows prompt.
    Declined,
}

/// Runs a program beside this one with administrator rights, after the
/// Windows prompt, and waits for it. Only a program in this program's own
/// folder is run: the caller names the file, never a path.
pub fn run_elevated(file_name: &str, arguments: &str) -> std::io::Result<Elevated> {
    if file_name.contains(['\\', '/']) {
        return Err(std::io::Error::other(
            "only a program beside this one is run",
        ));
    }
    let exe = std::env::current_exe()?;
    let dir = exe
        .parent()
        .ok_or_else(|| std::io::Error::other("this program has no folder"))?;
    let program = dir.join(file_name);
    if !program.is_file() {
        return Err(std::io::Error::other(format!(
            "{} is missing",
            program.display()
        )));
    }
    run_as_administrator(&program, arguments)
}

fn run_as_administrator(program: &Path, arguments: &str) -> std::io::Result<Elevated> {
    let file = wide(&program.to_string_lossy());
    let parameters = wide(arguments);
    let mut info = SHELLEXECUTEINFOW {
        cbSize: u32::try_from(std::mem::size_of::<SHELLEXECUTEINFOW>()).unwrap_or(0),
        fMask: SEE_MASK_NOCLOSEPROCESS | SEE_MASK_NOASYNC,
        lpVerb: w!("runas"),
        lpFile: PCWSTR(file.as_ptr()),
        lpParameters: PCWSTR(parameters.as_ptr()),
        nShow: SW_HIDE.0,
        ..Default::default()
    };
    // SAFETY: the structure and the strings it points at outlive the call.
    if let Err(err) = unsafe { ShellExecuteExW(&raw mut info) } {
        if err.code() == ERROR_CANCELLED.to_hresult() {
            return Ok(Elevated::Declined);
        }
        return Err(std::io::Error::other(err));
    }
    let process = info.hProcess;
    if process.is_invalid() {
        return Err(std::io::Error::other("Windows did not return the process"));
    }
    // SAFETY: a process handle this call owns, waited on and closed once.
    let outcome = unsafe {
        let waited = WaitForSingleObject(process, INFINITE);
        let mut code = 0u32;
        let read = GetExitCodeProcess(process, &raw mut code);
        let _ = CloseHandle(process);
        if waited == WAIT_OBJECT_0 {
            read.map(|()| code).map_err(std::io::Error::other)
        } else {
            Err(std::io::Error::other("could not wait for the program"))
        }
    };
    outcome.map(Elevated::Exited)
}
