//! Directories created with an access control list of their own.

use std::io;
use std::os::windows::fs::MetadataExt;
use std::path::Path;

use windows::Win32::Foundation::{
    CloseHandle, ERROR_ALREADY_EXISTS, HANDLE, HLOCAL, LUID, LocalFree,
};
use windows::Win32::Security::Authorization::{
    ConvertStringSecurityDescriptorToSecurityDescriptorW, GetNamedSecurityInfoW, SDDL_REVISION_1,
    SE_FILE_OBJECT, SetNamedSecurityInfoW,
};
use windows::Win32::Security::{
    ACL, AdjustTokenPrivileges, DACL_SECURITY_INFORMATION, GetSecurityDescriptorDacl,
    GetSecurityDescriptorOwner, IsWellKnownSid, LUID_AND_ATTRIBUTES, LookupPrivilegeValueW,
    OWNER_SECURITY_INFORMATION, PROTECTED_DACL_SECURITY_INFORMATION, PSECURITY_DESCRIPTOR, PSID,
    SE_PRIVILEGE_ENABLED, SE_RESTORE_NAME, SE_TAKE_OWNERSHIP_NAME, SECURITY_ATTRIBUTES,
    TOKEN_ADJUST_PRIVILEGES, TOKEN_PRIVILEGES, TOKEN_QUERY, WinBuiltinAdministratorsSid,
    WinLocalSystemSid,
};
use windows::Win32::Storage::FileSystem::{CreateDirectoryW, FILE_ATTRIBUTE_REPARSE_POINT};
use windows::Win32::System::Threading::{GetCurrentProcess, OpenProcessToken};
use windows_core::{BOOL, HSTRING, PCWSTR};

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

/// Enables a privilege the process holds but has not switched on.
fn enable_privilege(name: PCWSTR) -> io::Result<()> {
    let mut luid = LUID::default();
    // SAFETY: a static privilege name and an out LUID.
    unsafe { LookupPrivilegeValueW(None, name, &raw mut luid) }.map_err(io::Error::other)?;
    let mut token = HANDLE::default();
    // SAFETY: the pseudo handle of this process and an out token, closed below.
    unsafe {
        OpenProcessToken(
            GetCurrentProcess(),
            TOKEN_ADJUST_PRIVILEGES | TOKEN_QUERY,
            &raw mut token,
        )
    }
    .map_err(io::Error::other)?;
    let state = TOKEN_PRIVILEGES {
        PrivilegeCount: 1,
        Privileges: [LUID_AND_ATTRIBUTES {
            Luid: luid,
            Attributes: SE_PRIVILEGE_ENABLED,
        }],
    };
    // SAFETY: the token opened above and a state that outlives the call.
    let adjusted =
        unsafe { AdjustTokenPrivileges(token, false, Some(&raw const state), 0, None, None) };
    // SAFETY: the handle opened above, closed once.
    let _ = unsafe { CloseHandle(token) };
    adjusted.map_err(io::Error::other)
}

/// Makes the Administrators group the owner of `path`, whoever made it. An
/// owner can always rewrite a DACL, so a directory someone else created is not
/// secured until it stops being theirs.
fn take_ownership(path: &Path) -> io::Result<()> {
    enable_privilege(SE_TAKE_OWNERSHIP_NAME)?;
    enable_privilege(SE_RESTORE_NAME)?;
    let descriptor = Descriptor::from_sddl("O:BA")?;
    let mut owner = PSID::default();
    let mut defaulted = BOOL::default();
    // SAFETY: a descriptor that outlives the owner it points to.
    unsafe { GetSecurityDescriptorOwner(descriptor.as_ptr(), &raw mut owner, &raw mut defaulted) }
        .map_err(io::Error::other)?;
    // SAFETY: a NUL-terminated path and an owner inside the live descriptor.
    unsafe {
        SetNamedSecurityInfoW(
            &HSTRING::from(path.as_os_str()),
            SE_FILE_OBJECT,
            OWNER_SECURITY_INFORMATION,
            Some(owner),
            None,
            None,
            None,
        )
    }
    .ok()
    .map_err(io::Error::other)
}

/// Makes `path` a directory only this installation controls: it must not
/// redirect elsewhere, and it ends up owned by Administrators with the DACL
/// `sddl`, even if a standard user created it first.
pub fn secure_directory(path: &Path, sddl: &str) -> io::Result<()> {
    create_secured(path, sddl)?;
    refuse_reparse(path)?;
    take_ownership(path)?;
    apply_dacl(path, sddl)
}

/// Creates `path` for a process running as SYSTEM to trust. It and the
/// `ancestors` directories above it must be real directories that only
/// administrators own, because the owner of any one of them can swap what is
/// inside. Anything else is refused rather than used.
pub fn create_trusted(path: &Path, sddl: &str, ancestors: usize) -> io::Result<()> {
    create_secured(path, sddl)?;
    for directory in path.ancestors().take(ancestors + 1) {
        refuse_reparse(directory)?;
        if !owned_by_administrators(directory)? {
            return Err(io::Error::new(
                io::ErrorKind::PermissionDenied,
                format!(
                    "{} was not created by this computer's administrators",
                    directory.display()
                ),
            ));
        }
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
