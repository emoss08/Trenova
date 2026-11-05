//! The service's SID and its per-user inboxes, created with the ACLs in
//! `trenova_capture_svc::security`.

use std::io;

use capture_platform::accounts::account_sid;
pub use capture_platform::acl::{apply_dacl, create_secured, refuse_reparse};
use capture_protocol::handoff::Inbox;
use trenova_capture_svc::SERVICE_NAME;
use trenova_capture_svc::attribution::{Owner, valid_sid};
use trenova_capture_svc::handler::Inboxes;
use trenova_capture_svc::security::inbox_sddl;

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
        let sddl = inbox_sddl(&self.service_sid, &owner.sid);
        create_secured(&dir, &sddl)?;
        refuse_reparse(&dir)?;
        apply_dacl(&dir, &sddl)?;
        Ok(Inbox::new(dir))
    }
}
