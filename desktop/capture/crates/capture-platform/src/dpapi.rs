//! DPAPI, scoped to the signed-in user, for pages in the spool.

use std::io;

use capture_client::Protector;
use windows::Win32::Foundation::{HLOCAL, LocalFree};
use windows::Win32::Security::Cryptography::{
    CRYPT_INTEGER_BLOB, CRYPTPROTECT_UI_FORBIDDEN, CryptProtectData, CryptUnprotectData,
};
use windows_core::PCWSTR;

/// Mixed into every blob, so a page sealed by this application cannot be
/// opened by another running as the same user that simply calls DPAPI.
const ENTROPY: &[u8] = b"Trenova Capture spool v1";

#[derive(Debug, Default, Clone, Copy)]
pub struct Dpapi;

fn blob(bytes: &[u8]) -> io::Result<CRYPT_INTEGER_BLOB> {
    Ok(CRYPT_INTEGER_BLOB {
        cbData: u32::try_from(bytes.len()).map_err(|_| io::Error::other("too large to protect"))?,
        pbData: bytes.as_ptr().cast_mut(),
    })
}

/// Copies DPAPI's output and frees it.
///
/// # Safety
///
/// `out` was filled by a successful DPAPI call.
unsafe fn take(out: CRYPT_INTEGER_BLOB) -> Vec<u8> {
    // SAFETY: DPAPI allocated `cbData` bytes at `pbData` with LocalAlloc.
    unsafe {
        let bytes = std::slice::from_raw_parts(out.pbData, out.cbData as usize).to_vec();
        LocalFree(Some(HLOCAL(out.pbData.cast())));
        bytes
    }
}

impl Protector for Dpapi {
    fn protect(&self, plain: &[u8]) -> io::Result<Vec<u8>> {
        let input = blob(plain)?;
        let entropy = blob(ENTROPY)?;
        let mut out = CRYPT_INTEGER_BLOB::default();
        // SAFETY: the blobs point at live slices for the call; no prompt is
        // ever shown.
        unsafe {
            CryptProtectData(
                &raw const input,
                PCWSTR::null(),
                Some(&raw const entropy),
                None,
                None,
                CRYPTPROTECT_UI_FORBIDDEN,
                &raw mut out,
            )
        }
        .map_err(io::Error::other)?;
        // SAFETY: filled by the call above.
        Ok(unsafe { take(out) })
    }

    fn unprotect(&self, sealed: &[u8]) -> io::Result<Vec<u8>> {
        let input = blob(sealed)?;
        let entropy = blob(ENTROPY)?;
        let mut out = CRYPT_INTEGER_BLOB::default();
        // SAFETY: as above.
        unsafe {
            CryptUnprotectData(
                &raw const input,
                None,
                Some(&raw const entropy),
                None,
                None,
                CRYPTPROTECT_UI_FORBIDDEN,
                &raw mut out,
            )
        }
        .map_err(io::Error::other)?;
        // SAFETY: filled by the call above.
        Ok(unsafe { take(out) })
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn a_page_round_trips_and_is_not_stored_plain() {
        let sealed = Dpapi.protect(b"%PDF-1.7 page").expect("protect");
        assert!(!sealed.windows(4).any(|w| w == b"%PDF"));
        assert_eq!(
            Dpapi.unprotect(&sealed).expect("unprotect"),
            b"%PDF-1.7 page"
        );
    }
}
