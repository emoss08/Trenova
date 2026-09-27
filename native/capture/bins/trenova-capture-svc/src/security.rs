//! The ACLs on the service's directories, as SDDL.
//!
//! Every directory the service creates carries a protected DACL, so nothing
//! is inherited from `%ProgramData%`, where every user may create files. The
//! service is named by its own SID (`NT SERVICE\TrenovaCaptureSvc`), not by
//! `LocalService`, which other services share. Both SIDs are checked by
//! [`valid_sid`](crate::attribution::valid_sid) before they reach here, since
//! they are spliced into the SDDL.

/// Full control for SYSTEM, Administrators and the service.
fn owners(service_sid: &str) -> String {
    format!("(A;OICI;FA;;;SY)(A;OICI;FA;;;BA)(A;OICI;FA;;;{service_sid})")
}

/// The log directory: nobody else.
pub fn logs_sddl(service_sid: &str) -> String {
    format!("D:P{}", owners(service_sid))
}

/// The spool root: signed-in users may pass through to their own inbox
/// (traverse and synchronize) but not list who else has one.
pub fn spool_sddl(service_sid: &str) -> String {
    format!("D:P{}(A;;0x100020;;;AU)", owners(service_sid))
}

/// One person's inbox: they may read and delete what is in it (modify, not
/// full control, so they cannot change who else may).
pub fn inbox_sddl(service_sid: &str, user_sid: &str) -> String {
    format!("D:P{}(A;OICI;0x1301bf;;;{user_sid})", owners(service_sid))
}

#[cfg(test)]
mod tests {
    use super::*;

    const SERVICE: &str = "S-1-5-80-1-2-3-4-5";
    const USER: &str = "S-1-5-21-7-8-9-1001";

    #[test]
    fn every_directory_is_protected_and_only_an_inbox_admits_its_owner() {
        for sddl in [
            logs_sddl(SERVICE),
            spool_sddl(SERVICE),
            inbox_sddl(SERVICE, USER),
        ] {
            assert!(sddl.starts_with("D:P"), "protected: {sddl}");
            assert!(sddl.contains("(A;OICI;FA;;;S-1-5-80-1-2-3-4-5)"));
            assert!(!sddl.contains(";LS)"), "not every LocalService process");
            assert!(!sddl.contains(";WD)") && !sddl.contains(";BU)"));
        }
        assert!(!logs_sddl(SERVICE).contains(";AU)"));
        assert!(
            spool_sddl(SERVICE).ends_with("(A;;0x100020;;;AU)"),
            "traverse only, not inherited"
        );
        assert!(inbox_sddl(SERVICE, USER).ends_with("(A;OICI;0x1301bf;;;S-1-5-21-7-8-9-1001)"));
    }
}
