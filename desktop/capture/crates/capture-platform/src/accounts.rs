//! Windows accounts as SIDs, which name a person's print inbox and appear in
//! its ACL.

use windows::Win32::Foundation::{
    CloseHandle, ERROR_INSUFFICIENT_BUFFER, HANDLE, HLOCAL, LocalFree,
};
use windows::Win32::Security::Authorization::ConvertSidToStringSidW;
use windows::Win32::Security::{
    GetTokenInformation, LookupAccountNameW, PSID, SID_NAME_USE, TOKEN_QUERY, TOKEN_USER, TokenUser,
};
use windows::Win32::System::Threading::{GetCurrentProcess, OpenProcessToken};
use windows_core::{HSTRING, PCWSTR, PWSTR};

use crate::wide::from_pwstr;

/// A SID as a string (`S-1-5-21-…`).
///
/// # Safety
///
/// `sid` points at a valid SID for the duration of the call.
unsafe fn sid_string(sid: PSID) -> std::io::Result<String> {
    let mut text = PWSTR::null();
    // SAFETY: a valid SID, per the caller, and an out string.
    unsafe { ConvertSidToStringSidW(sid, &raw mut text) }.map_err(std::io::Error::other)?;
    // SAFETY: Windows allocated the string with LocalAlloc; freed once.
    Ok(unsafe {
        let value = from_pwstr(text);
        LocalFree(Some(HLOCAL(text.0.cast())));
        value
    })
}

/// A buffer aligned for the SID and token structures Windows writes into.
fn aligned(bytes: u32) -> Vec<u64> {
    vec![0u64; (bytes as usize).div_ceil(std::mem::size_of::<u64>())]
}

/// Closes a token handle when dropped.
struct Token(HANDLE);

impl Drop for Token {
    fn drop(&mut self) {
        // SAFETY: a handle this process opened, closed once.
        unsafe {
            let _ = CloseHandle(self.0);
        }
    }
}

/// The SID of the user this process runs as.
pub fn current_user_sid() -> std::io::Result<String> {
    let mut handle = HANDLE::default();
    // SAFETY: the pseudo-handle of this process, and an out handle.
    unsafe { OpenProcessToken(GetCurrentProcess(), TOKEN_QUERY, &raw mut handle) }
        .map_err(std::io::Error::other)?;
    let token = Token(handle);

    let mut size = 0u32;
    // SAFETY: asks for the size only; the expected failure is ignored.
    let _ = unsafe { GetTokenInformation(token.0, TokenUser, None, 0, &raw mut size) };
    if size == 0 {
        return Err(std::io::Error::last_os_error());
    }
    let mut buffer = aligned(size);
    // SAFETY: an 8-byte-aligned buffer of at least `size` bytes.
    unsafe {
        GetTokenInformation(
            token.0,
            TokenUser,
            Some(buffer.as_mut_ptr().cast()),
            size,
            &raw mut size,
        )
    }
    .map_err(std::io::Error::other)?;
    // SAFETY: the buffer now holds a TOKEN_USER, and is aligned for it.
    let user = unsafe { &*buffer.as_ptr().cast::<TOKEN_USER>() };
    // SAFETY: the SID lives inside `buffer`, which outlives the call.
    unsafe { sid_string(user.User.Sid) }
}

/// The SID of an account: `DOMAIN\user`, a bare `user` (resolved as Windows
/// resolves it, local accounts first), or a service's `NT SERVICE\<name>`.
pub fn account_sid(account: &str) -> std::io::Result<String> {
    let name = HSTRING::from(account);
    let mut sid_size = 0u32;
    let mut domain_size = 0u32;
    let mut kind = SID_NAME_USE::default();
    // SAFETY: asks for the sizes only.
    let sized = unsafe {
        LookupAccountNameW(
            PCWSTR::null(),
            &name,
            None,
            &raw mut sid_size,
            None,
            &raw mut domain_size,
            &raw mut kind,
        )
    };
    match sized {
        Err(err) if err.code() == ERROR_INSUFFICIENT_BUFFER.to_hresult() => {}
        Err(err) => return Err(std::io::Error::other(err)),
        Ok(()) => return Err(std::io::Error::other("the account lookup returned no size")),
    }
    let mut sid = aligned(sid_size);
    let mut domain = vec![0u16; domain_size as usize];
    // SAFETY: buffers of the sizes Windows asked for.
    unsafe {
        LookupAccountNameW(
            PCWSTR::null(),
            &name,
            Some(PSID(sid.as_mut_ptr().cast())),
            &raw mut sid_size,
            Some(PWSTR(domain.as_mut_ptr())),
            &raw mut domain_size,
            &raw mut kind,
        )
    }
    .map_err(std::io::Error::other)?;
    // SAFETY: the SID Windows just wrote into `sid`, which outlives the call.
    unsafe { sid_string(PSID(sid.as_mut_ptr().cast())) }
}
