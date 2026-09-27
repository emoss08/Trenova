//! Credential Manager, for the device credential.
//!
//! The credential is a generic credential named for the server, persisted to
//! this machine for this Windows user. Windows encrypts it with DPAPI; it
//! never touches a file of ours.

use std::io;

use capture_client::{Credential, SecretStore};
use windows::Win32::Foundation::ERROR_NOT_FOUND;
use windows::Win32::Security::Credentials::{
    CRED_FLAGS, CRED_PERSIST_LOCAL_MACHINE, CRED_TYPE_GENERIC, CREDENTIALW, CredDeleteW, CredFree,
    CredReadW, CredWriteW,
};
use windows_core::{PCWSTR, PWSTR};
use zeroize::Zeroize;

use crate::wide::wide;

/// Windows' limit on a generic credential's blob.
const MAX_BLOB: usize = 5 * 512;

#[derive(Debug, Clone)]
pub struct CredentialManager {
    target: Vec<u16>,
}

impl CredentialManager {
    /// The store for the credential paired with `host`.
    pub fn for_host(host: &str) -> Self {
        Self {
            target: wide(&format!("Trenova Capture/{host}")),
        }
    }
}

fn not_found(err: &windows_core::Error) -> bool {
    err.code() == ERROR_NOT_FOUND.to_hresult()
}

impl SecretStore for CredentialManager {
    fn load(&self) -> io::Result<Option<Credential>> {
        let mut found: *mut CREDENTIALW = std::ptr::null_mut();
        // SAFETY: a NUL-terminated name and an out pointer Windows fills.
        if let Err(err) = unsafe {
            CredReadW(
                PCWSTR(self.target.as_ptr()),
                CRED_TYPE_GENERIC,
                None,
                &raw mut found,
            )
        } {
            return if not_found(&err) {
                Ok(None)
            } else {
                Err(io::Error::other(err))
            };
        }
        // SAFETY: filled by the successful read; freed below exactly once.
        let mut blob = unsafe {
            let credential = &*found;
            let blob = std::slice::from_raw_parts(
                credential.CredentialBlob,
                credential.CredentialBlobSize as usize,
            )
            .to_vec();
            CredFree(found.cast());
            blob
        };
        let parsed = serde_json::from_slice::<Credential>(&blob);
        blob.zeroize();
        match parsed {
            Ok(credential) => Ok(Some(credential)),
            Err(err) => {
                tracing::error!(error = %err, "the saved credential is unreadable and is being removed");
                self.clear()?;
                Ok(None)
            }
        }
    }

    fn save(&self, credential: &Credential) -> io::Result<()> {
        let mut blob = serde_json::to_vec(credential).map_err(io::Error::other)?;
        if blob.len() > MAX_BLOB {
            blob.zeroize();
            return Err(io::Error::other(
                "the credential is larger than Windows can store",
            ));
        }
        let mut user = wide(&credential.tokens.device_id.to_string());
        let entry = CREDENTIALW {
            Flags: CRED_FLAGS(0),
            Type: CRED_TYPE_GENERIC,
            TargetName: PWSTR(self.target.as_ptr().cast_mut()),
            CredentialBlobSize: u32::try_from(blob.len()).unwrap_or(0),
            CredentialBlob: blob.as_mut_ptr(),
            Persist: CRED_PERSIST_LOCAL_MACHINE,
            UserName: PWSTR(user.as_mut_ptr()),
            ..CREDENTIALW::default()
        };
        // SAFETY: every pointer in `entry` is live for the call.
        let written = unsafe { CredWriteW(&raw const entry, 0) };
        blob.zeroize();
        written.map_err(io::Error::other)
    }

    fn clear(&self) -> io::Result<()> {
        // SAFETY: a NUL-terminated name.
        match unsafe { CredDeleteW(PCWSTR(self.target.as_ptr()), CRED_TYPE_GENERIC, None) } {
            Err(err) if !not_found(&err) => Err(io::Error::other(err)),
            _ => Ok(()),
        }
    }
}
