//! One agent per Windows session.

use windows::Win32::Foundation::{CloseHandle, ERROR_ALREADY_EXISTS, GetLastError, HANDLE};
use windows::Win32::System::Threading::CreateMutexW;
use windows_core::w;

/// Held for the agent's lifetime. The mutex lives in the session's own
/// namespace, so two people signed in to one computer each get their own
/// agent.
#[derive(Debug)]
pub struct SingleInstance {
    handle: HANDLE,
}

impl SingleInstance {
    /// `None` when another agent already runs in this session.
    pub fn acquire() -> std::io::Result<Option<Self>> {
        // SAFETY: creating or opening a named mutex; the handle is closed on
        // drop.
        let handle = unsafe { CreateMutexW(None, false, w!("Local\\TrenovaCapture.Agent")) }
            .map_err(std::io::Error::other)?;
        // SAFETY: reads this thread's last error, set by the call above.
        if unsafe { GetLastError() } == ERROR_ALREADY_EXISTS {
            // SAFETY: the handle was just opened.
            unsafe {
                let _ = CloseHandle(handle);
            }
            return Ok(None);
        }
        Ok(Some(Self { handle }))
    }
}

impl Drop for SingleInstance {
    fn drop(&mut self) {
        // SAFETY: opened in `acquire`, closed once.
        unsafe {
            let _ = CloseHandle(self.handle);
        }
    }
}
