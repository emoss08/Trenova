//! Directories created with an access control list of their own.

use std::io;
use std::path::Path;

use windows::Win32::Foundation::{ERROR_ALREADY_EXISTS, HLOCAL, LocalFree};
use windows::Win32::Security::Authorization::{
    ConvertStringSecurityDescriptorToSecurityDescriptorW, SDDL_REVISION_1,
};
use windows::Win32::Security::{PSECURITY_DESCRIPTOR, SECURITY_ATTRIBUTES};
use windows::Win32::Storage::FileSystem::CreateDirectoryW;
use windows_core::HSTRING;

/// A protected DACL for SYSTEM and Administrators only.
pub const SYSTEM_ONLY_SDDL: &str = "D:P(A;OICI;FA;;;SY)(A;OICI;FA;;;BA)";

/// Frees a security descriptor Windows allocated.
pub struct Descriptor(PSECURITY_DESCRIPTOR);

impl std::fmt::Debug for Descriptor {
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        f.write_str("Descriptor")
    }
}

impl Descriptor {
    /// A security descriptor from its SDDL.
    pub fn from_sddl(sddl: &str) -> io::Result<Self> {
        let mut descriptor = PSECURITY_DESCRIPTOR::default();
        // SAFETY: a NUL-terminated SDDL string and an out descriptor.
        unsafe {
            ConvertStringSecurityDescriptorToSecurityDescriptorW(
                &HSTRING::from(sddl),
                SDDL_REVISION_1,
                &raw mut descriptor,
                None,
            )
        }
        .map_err(io::Error::other)?;
        Ok(Self(descriptor))
    }

    pub fn as_ptr(&self) -> PSECURITY_DESCRIPTOR {
        self.0
    }
}

impl Drop for Descriptor {
    fn drop(&mut self) {
        // SAFETY: allocated with LocalAlloc by the SDDL conversion; freed once.
        unsafe {
            LocalFree(Some(HLOCAL(self.0.0)));
        }
    }
}

/// Creates `path` with the DACL `sddl`. A directory that is already there is
/// left as it is: its creator chose its ACL, and only the accounts an ACL
/// here names can create anything beside it.
pub fn create_secured(path: &Path, sddl: &str) -> io::Result<()> {
    let descriptor = Descriptor::from_sddl(sddl)?;
    let attributes = SECURITY_ATTRIBUTES {
        nLength: u32::try_from(std::mem::size_of::<SECURITY_ATTRIBUTES>()).unwrap_or(0),
        lpSecurityDescriptor: descriptor.as_ptr().0,
        bInheritHandle: false.into(),
    };
    // SAFETY: a NUL-terminated path and attributes whose descriptor lives
    // until after the call.
    match unsafe {
        CreateDirectoryW(
            &HSTRING::from(path.as_os_str()),
            Some(&raw const attributes),
        )
    } {
        Ok(()) => Ok(()),
        Err(err) if err.code() == ERROR_ALREADY_EXISTS.to_hresult() => Ok(()),
        Err(err) => Err(io::Error::other(err)),
    }
}
