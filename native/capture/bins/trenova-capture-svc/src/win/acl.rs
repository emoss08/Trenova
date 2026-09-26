//! Creating the service's directories with their ACLs
//! (`trenova_capture_svc::security`).

use std::io;
use std::path::Path;

use capture_platform::accounts::account_sid;
use capture_protocol::handoff::Inbox;
use trenova_capture_svc::SERVICE_NAME;
use trenova_capture_svc::attribution::{Owner, valid_sid};
use trenova_capture_svc::handler::Inboxes;
use trenova_capture_svc::security::inbox_sddl;
use windows::Win32::Foundation::{ERROR_ALREADY_EXISTS, HLOCAL, LocalFree};
use windows::Win32::Security::Authorization::{
    ConvertStringSecurityDescriptorToSecurityDescriptorW, SDDL_REVISION_1,
};
use windows::Win32::Security::{PSECURITY_DESCRIPTOR, SECURITY_ATTRIBUTES};
use windows::Win32::Storage::FileSystem::CreateDirectoryW;
use windows_core::HSTRING;

/// The service's own SID.
pub fn service_sid() -> io::Result<String> {
    let sid = account_sid(&format!("NT SERVICE\\{SERVICE_NAME}"))?;
    if valid_sid(&sid) {
        Ok(sid)
    } else {
        Err(io::Error::other(
            "Windows returned an unexpected service SID",
        ))
    }
}

/// Frees a security descriptor Windows allocated.
struct Descriptor(PSECURITY_DESCRIPTOR);

impl Drop for Descriptor {
    fn drop(&mut self) {
        // SAFETY: allocated with LocalAlloc by the SDDL conversion; freed once.
        unsafe {
            LocalFree(Some(HLOCAL(self.0.0)));
        }
    }
}

/// Creates `path` with the DACL `sddl`. A directory that is already there
/// was created the same way, since only this service and administrators can
/// create anything beside it, and is left as it is.
pub fn create_secured(path: &Path, sddl: &str) -> io::Result<()> {
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
    let descriptor = Descriptor(descriptor);
    let attributes = SECURITY_ATTRIBUTES {
        nLength: u32::try_from(std::mem::size_of::<SECURITY_ATTRIBUTES>()).unwrap_or(0),
        lpSecurityDescriptor: descriptor.0.0,
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

/// Inboxes under the ACL'd spool root, each ACL'd to its owner.
#[derive(Clone, Debug)]
pub struct SecuredInboxes {
    pub root: std::path::PathBuf,
    pub service_sid: String,
}

impl Inboxes for SecuredInboxes {
    fn inbox_for(&self, owner: &Owner) -> io::Result<Inbox> {
        if !valid_sid(&owner.sid) {
            return Err(io::Error::from(io::ErrorKind::InvalidInput));
        }
        let dir = self.root.join(&owner.sid);
        create_secured(&dir, &inbox_sddl(&self.service_sid, &owner.sid))?;
        Ok(Inbox::new(dir))
    }
}
