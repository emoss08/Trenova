//! The Windows print queue and signed-in sessions, for attribution.

use std::io;

use capture_platform::accounts::account_sid;
use trenova_capture_svc::PRINTER_NAME;
use trenova_capture_svc::attribution::{PrintSystem, QueuedJob, SignedIn};
use windows::Win32::Foundation::ERROR_INSUFFICIENT_BUFFER;
use windows::Win32::Graphics::Printing::{
    ClosePrinter, EnumJobsW, JOB_INFO_2W, OpenPrinterW, PRINTER_HANDLE,
};
use windows::Win32::System::RemoteDesktop::{
    WTS_INFO_CLASS, WTS_SESSION_INFOW, WTSDomainName, WTSEnumerateSessionsW, WTSFreeMemory,
    WTSQuerySessionInformationW, WTSUserName,
};
use windows_core::{HSTRING, PWSTR};

/// The most jobs read from the queue at once; the Trenova queue holds a
/// handful at a time.
const MAX_JOBS: u32 = 1024;
/// Tries at reading the queue while it changes between the size query and
/// the read.
const ENUM_ATTEMPTS: usize = 4;

/// Closes a printer handle when dropped.
struct Printer(PRINTER_HANDLE);

impl Drop for Printer {
    fn drop(&mut self) {
        // SAFETY: a handle OpenPrinterW returned, closed once.
        unsafe {
            let _ = ClosePrinter(self.0);
        }
    }
}

/// Reads a string Windows returned, or an empty one for null.
///
/// # Safety
///
/// `value` is null or points at a NUL-terminated UTF-16 string.
unsafe fn read(value: PWSTR) -> String {
    if value.is_null() {
        return String::new();
    }
    // SAFETY: per the caller.
    unsafe { value.to_string() }.unwrap_or_default()
}

#[derive(Debug, Default)]
pub struct WindowsPrintSystem;

impl WindowsPrintSystem {
    fn open() -> io::Result<Printer> {
        let name = HSTRING::from(PRINTER_NAME);
        let mut handle = PRINTER_HANDLE::default();
        // SAFETY: a NUL-terminated name, an out handle, and default access,
        // which includes enumerating jobs.
        unsafe { OpenPrinterW(&name, &raw mut handle, None) }.map_err(io::Error::other)?;
        Ok(Printer(handle))
    }

    fn session_string(session: u32, class: WTS_INFO_CLASS) -> String {
        let mut buffer = PWSTR::null();
        let mut bytes = 0u32;
        // SAFETY: an out buffer Windows allocates, freed below.
        if unsafe {
            WTSQuerySessionInformationW(None, session, class, &raw mut buffer, &raw mut bytes)
        }
        .is_err()
        {
            return String::new();
        }
        // SAFETY: a NUL-terminated string Windows allocated; freed once.
        unsafe {
            let value = read(buffer);
            WTSFreeMemory(buffer.0.cast());
            value
        }
    }
}

impl PrintSystem for WindowsPrintSystem {
    fn queued_jobs(&self) -> io::Result<Vec<QueuedJob>> {
        let printer = Self::open()?;
        let mut needed = 0u32;
        let mut count = 0u32;
        for _ in 0..ENUM_ATTEMPTS {
            let words = (needed as usize)
                .div_ceil(std::mem::size_of::<u64>())
                .max(1);
            let mut buffer = vec![0u64; words];
            let bytes_len = if needed == 0 {
                0
            } else {
                words * std::mem::size_of::<u64>()
            };
            // SAFETY: the u64 buffer is at least `bytes_len` bytes, and
            // aligned for JOB_INFO_2W.
            let bytes = unsafe {
                std::slice::from_raw_parts_mut(buffer.as_mut_ptr().cast::<u8>(), bytes_len)
            };
            let pjob = if bytes_len == 0 { None } else { Some(bytes) };
            // SAFETY: an open printer, a buffer of the size given, and out
            // counts.
            let listed = unsafe {
                EnumJobsW(
                    printer.0,
                    0,
                    MAX_JOBS,
                    2,
                    pjob,
                    &raw mut needed,
                    &raw mut count,
                )
            };
            match listed {
                Ok(()) => {
                    // SAFETY: Windows wrote `count` JOB_INFO_2W records at the
                    // start of the buffer, with their strings after them.
                    let jobs = unsafe {
                        std::slice::from_raw_parts(
                            buffer.as_ptr().cast::<JOB_INFO_2W>(),
                            count as usize,
                        )
                    };
                    return Ok(jobs
                        .iter()
                        .map(|job| QueuedJob {
                            id: job.JobId,
                            // SAFETY: strings inside the buffer, alive here.
                            document: unsafe { read(job.pDocument) },
                            // SAFETY: as above.
                            user: unsafe { read(job.pUserName) },
                        })
                        .collect());
                }
                Err(err) if err.code() == ERROR_INSUFFICIENT_BUFFER.to_hresult() && needed > 0 => {}
                Err(err) => return Err(io::Error::other(err)),
            }
        }
        Err(io::Error::other(
            "the print queue kept changing while it was read",
        ))
    }

    fn signed_in(&self) -> io::Result<Vec<SignedIn>> {
        let mut sessions: *mut WTS_SESSION_INFOW = std::ptr::null_mut();
        let mut count = 0u32;
        // SAFETY: the local server, and out parameters Windows allocates.
        unsafe { WTSEnumerateSessionsW(None, 0, 1, &raw mut sessions, &raw mut count) }
            .map_err(io::Error::other)?;
        // SAFETY: Windows returned `count` records; they are copied out
        // before the block is freed.
        let ids: Vec<u32> = unsafe { std::slice::from_raw_parts(sessions, count as usize) }
            .iter()
            .map(|session| session.SessionId)
            .collect();
        // SAFETY: the block WTSEnumerateSessionsW allocated, freed once.
        unsafe { WTSFreeMemory(sessions.cast()) };

        Ok(ids
            .into_iter()
            .filter_map(|id| {
                let user = Self::session_string(id, WTSUserName);
                (!user.is_empty()).then(|| SignedIn {
                    domain: Self::session_string(id, WTSDomainName),
                    user,
                })
            })
            .collect())
    }

    fn sid_of(&self, domain: &str, user: &str) -> io::Result<String> {
        if domain.is_empty() {
            account_sid(user)
        } else {
            account_sid(&format!("{domain}\\{user}"))
        }
    }
}
