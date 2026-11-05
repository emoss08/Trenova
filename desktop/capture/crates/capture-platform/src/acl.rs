//! Directories created with an access control list of their own.

use std::io;
use std::os::windows::fs::MetadataExt;
use std::path::Path;

use windows::Win32::Foundation::{ERROR_ALREADY_EXISTS, HLOCAL, LocalFree};
use windows::Win32::Security::Authorization::{
    ConvertStringSecurityDescriptorToSecurityDescriptorW, GetNamedSecurityInfoW, SDDL_REVISION_1,
    SE_FILE_OBJECT, SetNamedSecurityInfoW,
};
use windows::Win32::Security::{
    ACL, DACL_SECURITY_INFORMATION, GetSecurityDescriptorDacl, IsWellKnownSid,
    OWNER_SECURITY_INFORMATION, PROTECTED_DACL_SECURITY_INFORMATION, PSECURITY_DESCRIPTOR, PSID,
    SECURITY_ATTRIBUTES, WinBuiltinAdministratorsSid, WinLocalSystemSid,
};
use windows::Win32::Storage::FileSystem::{CreateDirectoryW, FILE_ATTRIBUTE_REPARSE_POINT};
use windows_core::{BOOL, HSTRING};

/// A protected DACL for SYSTEM and Administrators only.
pub const SYSTEM_ONLY_SDDL: &str = "D:P(A;OICI;FA;;;SY)(A;OICI;FA;;;BA)";

/// The shared directory: signed-in users may read it, and only administrators
/// may create anything in it, so nobody can make a directory there first.
pub const SHARED_DIR_SDDL: &str = "D:P(A;OICI;FA;;;SY)(A;OICI;FA;;;BA)(A;OICI;0x1200a9;;;AU)";

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

/// Refuses a path that is a junction, mount point or symbolic link, so a write
/// meant for it cannot land somewhere else.
pub fn refuse_reparse(path: &Path) -> io::Result<()> {
    let attributes = std::fs::symlink_metadata(path)?.file_attributes();
    if attributes & FILE_ATTRIBUTE_REPARSE_POINT.0 != 0 {
        return Err(io::Error::new(
            io::ErrorKind::PermissionDenied,
            format!("{} redirects elsewhere", path.display()),
        ));
    }
    Ok(())
}

/// Replaces the DACL on `path` with the protected one in `sddl`.
pub fn apply_dacl(path: &Path, sddl: &str) -> io::Result<()> {
    let descriptor = Descriptor::from_sddl(sddl)?;
    let mut present = BOOL::default();
    let mut defaulted = BOOL::default();
    let mut dacl: *mut ACL = std::ptr::null_mut();
    // SAFETY: a descriptor that outlives the out pointers, which it owns.
    unsafe {
        GetSecurityDescriptorDacl(
            descriptor.as_ptr(),
            &raw mut present,
            &raw mut dacl,
            &raw mut defaulted,
        )
    }
    .map_err(io::Error::other)?;
    // SAFETY: a NUL-terminated path and a DACL inside the live descriptor.
    unsafe {
        SetNamedSecurityInfoW(
            &HSTRING::from(path.as_os_str()),
            SE_FILE_OBJECT,
            DACL_SECURITY_INFORMATION | PROTECTED_DACL_SECURITY_INFORMATION,
            None,
            None,
            Some(dacl.cast_const()),
            None,
        )
    }
    .ok()
    .map_err(io::Error::other)
}

/// Whether SYSTEM or the Administrators group owns `path`.
fn owned_by_administrators(path: &Path) -> io::Result<bool> {
    let mut owner = PSID::default();
    let mut descriptor = PSECURITY_DESCRIPTOR::default();
    // SAFETY: a NUL-terminated path; the owner points into the descriptor,
    // which Windows allocates and `Descriptor` frees.
    unsafe {
        GetNamedSecurityInfoW(
            &HSTRING::from(path.as_os_str()),
            SE_FILE_OBJECT,
            OWNER_SECURITY_INFORMATION,
            Some(&raw mut owner),
            None,
            None,
            None,
            &raw mut descriptor,
        )
    }
    .ok()
    .map_err(io::Error::other)?;
    let descriptor = Descriptor(descriptor);
    // SAFETY: the owner stays valid while `descriptor` is alive.
    let trusted = unsafe {
        IsWellKnownSid(owner, WinLocalSystemSid).as_bool()
            || IsWellKnownSid(owner, WinBuiltinAdministratorsSid).as_bool()
    };
    drop(descriptor);
    Ok(trusted)
}

/// Creates `path` for a process running as SYSTEM to trust: one a standard user
/// made first, or one that redirects elsewhere, is refused rather than used.
pub fn create_trusted(path: &Path, sddl: &str) -> io::Result<()> {
    create_secured(path, sddl)?;
    refuse_reparse(path)?;
    if !owned_by_administrators(path)? {
        return Err(io::Error::new(
            io::ErrorKind::PermissionDenied,
            format!("{} was not created by this computer's administrators", path.display()),
        ));
    }
    Ok(())
}

/// Creates `path` with the DACL `sddl`. A directory that is already there is
/// left as it is: its creator chose its ACL, and only the accounts an ACL
/// here names can create anything beside it. A caller that cannot trust that
/// uses [`create_trusted`].
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
