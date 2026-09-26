//! Other people's processes: the agents to close before an install, and the
//! agents to start again for everyone signed in once it is done.

use std::path::Path;

use trenova_capture_update::UpdaterError;
use windows::Win32::Foundation::{CloseHandle, HANDLE};
use windows::Win32::System::Diagnostics::ToolHelp::{
    CreateToolhelp32Snapshot, PROCESSENTRY32W, Process32FirstW, Process32NextW, TH32CS_SNAPPROCESS,
};
use windows::Win32::System::Environment::{CreateEnvironmentBlock, DestroyEnvironmentBlock};
use windows::Win32::System::RemoteDesktop::{
    WTS_SESSION_INFOW, WTSActive, WTSEnumerateSessionsW, WTSFreeMemory, WTSQueryUserToken,
};
use windows::Win32::System::Threading::{
    CREATE_NO_WINDOW, CREATE_UNICODE_ENVIRONMENT, CreateProcessAsUserW, OpenProcess,
    PROCESS_INFORMATION, PROCESS_TERMINATE, STARTUPINFOW, TerminateProcess,
};
use windows_core::{HSTRING, PWSTR};

/// Closes a handle when dropped.
struct Owned(HANDLE);

impl Drop for Owned {
    fn drop(&mut self) {
        if !self.0.is_invalid() {
            // SAFETY: a handle this process owns, closed once.
            unsafe {
                let _ = CloseHandle(self.0);
            }
        }
    }
}

fn name_of(entry: &PROCESSENTRY32W) -> String {
    let end = entry
        .szExeFile
        .iter()
        .position(|&c| c == 0)
        .unwrap_or(entry.szExeFile.len());
    String::from_utf16_lossy(&entry.szExeFile[..end])
}

/// Ends every process with one of `names`, in every session. Their pages are
/// on disk already; what a killed scan loses is what the scanner had not yet
/// handed over, and the agent does not start an update while scanning.
pub fn close_processes(names: &[&str]) {
    // SAFETY: a snapshot of all processes; closed by `Owned`.
    let snapshot = unsafe { CreateToolhelp32Snapshot(TH32CS_SNAPPROCESS, 0) };
    let Ok(snapshot) = snapshot else {
        return;
    };
    let snapshot = Owned(snapshot);
    let mut entry = PROCESSENTRY32W {
        dwSize: u32::try_from(std::mem::size_of::<PROCESSENTRY32W>()).unwrap_or(0),
        ..PROCESSENTRY32W::default()
    };
    // SAFETY: a valid snapshot and a sized entry.
    let mut next = unsafe { Process32FirstW(snapshot.0, &raw mut entry) };
    while next.is_ok() {
        let name = name_of(&entry);
        if names.iter().any(|n| n.eq_ignore_ascii_case(&name)) {
            // SAFETY: opens by id with the one right needed; the handle is
            // closed by `Owned`.
            let opened = unsafe { OpenProcess(PROCESS_TERMINATE, false, entry.th32ProcessID) };
            if let Ok(process) = opened {
                let process = Owned(process);
                // SAFETY: a handle with terminate rights.
                if unsafe { TerminateProcess(process.0, 1) }.is_ok() {
                    tracing::info!(name, pid = entry.th32ProcessID, "closed before installing");
                }
            }
        }
        // SAFETY: as above.
        next = unsafe { Process32NextW(snapshot.0, &raw mut entry) };
    }
}

/// The ids of sessions somebody is signed in to and using.
fn active_sessions() -> Result<Vec<u32>, UpdaterError> {
    let mut sessions: *mut WTS_SESSION_INFOW = std::ptr::null_mut();
    let mut count = 0u32;
    // SAFETY: the local server and out parameters Windows allocates.
    unsafe { WTSEnumerateSessionsW(None, 0, 1, &raw mut sessions, &raw mut count) }
        .map_err(std::io::Error::other)?;
    // SAFETY: `count` records Windows returned; copied out before the free.
    let ids: Vec<u32> = unsafe { std::slice::from_raw_parts(sessions, count as usize) }
        .iter()
        .filter(|s| s.State == WTSActive)
        .map(|s| s.SessionId)
        .collect();
    // SAFETY: the block WTSEnumerateSessionsW allocated, freed once.
    unsafe { WTSFreeMemory(sessions.cast()) };
    Ok(ids)
}

/// Starts `exe` as the user of `session`, on their desktop, with their
/// environment. The agent keeps one instance per session itself.
fn start_in_session(exe: &Path, session: u32) -> Result<(), UpdaterError> {
    let mut token = HANDLE::default();
    // SAFETY: an out handle; the caller is the system, which may ask.
    unsafe { WTSQueryUserToken(session, &raw mut token) }.map_err(std::io::Error::other)?;
    let token = Owned(token);

    let mut environment: *mut core::ffi::c_void = std::ptr::null_mut();
    // SAFETY: the user's token and an out pointer; destroyed below.
    unsafe { CreateEnvironmentBlock(&raw mut environment, Some(token.0), false) }
        .map_err(std::io::Error::other)?;

    let mut desktop: Vec<u16> = "winsta0\\default".encode_utf16().chain(Some(0)).collect();
    let startup = STARTUPINFOW {
        cb: u32::try_from(std::mem::size_of::<STARTUPINFOW>()).unwrap_or(0),
        lpDesktop: PWSTR(desktop.as_mut_ptr()),
        ..STARTUPINFOW::default()
    };
    let mut command: Vec<u16> = format!("\"{}\"", exe.display())
        .encode_utf16()
        .chain(Some(0))
        .collect();
    let mut information = PROCESS_INFORMATION::default();
    let directory = HSTRING::from(exe.parent().unwrap_or(exe).as_os_str());
    // SAFETY: the user's token, a NUL-terminated mutable command line, a
    // Unicode environment block and startup information of the right size.
    let started = unsafe {
        CreateProcessAsUserW(
            Some(token.0),
            &HSTRING::from(exe.as_os_str()),
            Some(PWSTR(command.as_mut_ptr())),
            None,
            None,
            false,
            CREATE_UNICODE_ENVIRONMENT | CREATE_NO_WINDOW,
            Some(environment),
            &directory,
            &raw const startup,
            &raw mut information,
        )
    };
    // SAFETY: the block CreateEnvironmentBlock made, destroyed once.
    unsafe {
        let _ = DestroyEnvironmentBlock(environment);
    }
    started.map_err(std::io::Error::other)?;
    drop(Owned(information.hProcess));
    drop(Owned(information.hThread));
    Ok(())
}

/// Starts `exe` for everyone signed in. One session failing does not stop
/// the others; the first failure is returned once all were tried.
pub fn start_in_every_session(exe: &Path) -> Result<(), UpdaterError> {
    let mut first_error = None;
    for session in active_sessions()? {
        match start_in_session(exe, session) {
            Ok(()) => tracing::info!(session, "started Trenova Capture"),
            Err(err) => {
                tracing::warn!(session, error = %err, "could not start Trenova Capture");
                first_error.get_or_insert(err);
            }
        }
    }
    first_error.map_or(Ok(()), Err)
}
