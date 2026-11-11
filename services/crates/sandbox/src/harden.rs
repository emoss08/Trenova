//! What a child does to itself once it has loaded its libraries and before it
//! reads any untrusted input.
//!
//! In order:
//!
//! 1. `PR_SET_NO_NEW_PRIVS`, so nothing it runs can gain privileges, which
//!    is also what lets an unprivileged process install a seccomp filter.
//! 2. Landlock: the filesystem shrinks to the read-only paths the policy
//!    names (shared libraries, OCR models, fonts) and the read-write paths
//!    it names, if any. Everything else is gone, including `/proc` of other
//!    processes and every writable directory.
//! 3. seccomp-bpf: an allow-list of the system calls parsing, rendering and
//!    talking to the supervisor need. No call that opens a network socket,
//!    starts a program, traces a process or changes a namespace is on it,
//!    and those are matched first and kill the process outright; anything
//!    else unlisted fails with `EPERM`.
//!
//! The resource limits (address space, CPU, file size, open files, no core
//! dumps) are set by the supervisor between fork and exec, so they hold even
//! while the child loads its libraries. The child reports what is in force
//! and the supervisor refuses a child whose report falls short of the
//! policy, unless it is configured to run unprotected.

use std::path::PathBuf;

use serde::{Deserialize, Serialize};

/// What a child must do to itself. Sent in the `Init` frame.
#[derive(Clone, Debug, Default, Serialize, Deserialize, PartialEq, Eq)]
pub struct HardeningPolicy {
    /// Paths the child may read (and map, and execute from) after hardening.
    pub read_only: Vec<PathBuf>,
    /// Paths the child may also write.
    pub read_write: Vec<PathBuf>,
    pub landlock: bool,
    pub seccomp: bool,
}

/// How completely landlock is in force.
#[derive(Clone, Copy, Debug, Serialize, Deserialize, PartialEq, Eq)]
#[serde(rename_all = "snake_case")]
pub enum LandlockStatus {
    /// Every restriction the policy depends on is enforced.
    Full,
    /// The kernel supports landlock but lacks an access right the policy
    /// depends on.
    Partial,
    /// The kernel has no landlock, or it is turned off.
    None,
    /// The policy did not ask for it.
    Off,
}

/// What a child reports is in force. Sent in the `Ready` frame.
#[derive(Clone, Debug, Serialize, Deserialize, PartialEq, Eq)]
pub struct HardeningReport {
    pub no_new_privs: bool,
    pub landlock: LandlockStatus,
    pub seccomp: bool,
    /// The address-space limit in force, in bytes; 0 if none.
    pub address_space_limit: u64,
    /// Why anything asked for is not in force.
    pub problems: Vec<String>,
}

impl HardeningReport {
    /// Whether everything `policy` asked for is in force.
    pub fn satisfies(&self, policy: &HardeningPolicy) -> bool {
        self.no_new_privs
            && (!policy.landlock || self.landlock == LandlockStatus::Full)
            && (!policy.seccomp || self.seccomp)
            && self.address_space_limit > 0
    }
}

/// Applies `policy` to the calling process (all of its threads for
/// seccomp; landlock and no-new-privs are per thread and inherited, so a
/// child calls this from its only thread).
#[cfg(target_os = "linux")]
pub fn harden(policy: &HardeningPolicy) -> HardeningReport {
    let mut problems = Vec::new();

    let no_new_privs = match rustix::thread::set_no_new_privs(true) {
        Ok(()) => true,
        Err(err) => {
            problems.push(format!("no_new_privs: {err}"));
            false
        }
    };

    let landlock = if policy.landlock {
        match landlock::apply(policy) {
            Ok(status) => status,
            Err(err) => {
                problems.push(format!("landlock: {err}"));
                LandlockStatus::None
            }
        }
    } else {
        LandlockStatus::Off
    };

    let seccomp = if policy.seccomp && no_new_privs {
        match crate::seccomp::install() {
            Ok(()) => true,
            Err(err) => {
                problems.push(format!("seccomp: {err}"));
                false
            }
        }
    } else {
        if policy.seccomp {
            problems.push("seccomp: needs no_new_privs".into());
        }
        false
    };

    let address_space_limit = rustix::process::getrlimit(rustix::process::Resource::As)
        .current
        .unwrap_or(0);

    HardeningReport {
        no_new_privs,
        landlock,
        seccomp,
        address_space_limit,
        problems,
    }
}

#[cfg(not(target_os = "linux"))]
pub fn harden(_policy: &HardeningPolicy) -> HardeningReport {
    HardeningReport {
        no_new_privs: false,
        landlock: LandlockStatus::None,
        seccomp: false,
        address_space_limit: 0,
        problems: vec!["the sandbox runs only on Linux".into()],
    }
}

#[cfg(target_os = "linux")]
mod landlock {
    use landlock::{
        ABI, Access, AccessFs, CompatLevel, Compatible, PathBeneath, PathFd, Ruleset, RulesetAttr,
        RulesetCreatedAttr, RulesetStatus, Scope,
    };

    use super::{HardeningPolicy, LandlockStatus};

    /// The newest ABI this code knows the access rights of. Every right the
    /// running kernel has is asked for; whether the ones it lacks matter is
    /// decided by [`required_abi`].
    const ABI_TARGET: ABI = ABI::V5;

    /// The oldest ABI that enforces the policy in full beside seccomp: the
    /// third (kernel 6.2), the first that can refuse truncation. The fifth
    /// (kernel 6.10) adds only `ioctl` on device files, and the seccomp allow
    /// list has no `ioctl` on it, so a child under both loses nothing on an
    /// older kernel such as Ubuntu 24.04's 6.8.
    const ABI_BESIDE_SECCOMP: ABI = ABI::V3;

    /// The oldest ABI on which `policy` is fully in force.
    pub(super) fn required_abi(policy: &HardeningPolicy) -> ABI {
        if policy.seccomp {
            ABI_BESIDE_SECCOMP
        } else {
            ABI_TARGET
        }
    }

    /// Whether the running kernel enforces every filesystem right of `abi`.
    fn kernel_enforces(abi: ABI) -> bool {
        Ruleset::default()
            .set_compatibility(CompatLevel::HardRequirement)
            .handle_access(AccessFs::from_all(abi))
            .is_ok()
    }

    pub fn apply(policy: &HardeningPolicy) -> Result<LandlockStatus, String> {
        let all = AccessFs::from_all(ABI_TARGET);
        let read = AccessFs::from_read(ABI_TARGET);
        let mut ruleset = Ruleset::default()
            .handle_access(all)
            .map_err(|err| err.to_string())?
            .create()
            .map_err(|err| err.to_string())?;
        for path in &policy.read_only {
            let Ok(fd) = PathFd::new(path) else {
                continue;
            };
            ruleset = ruleset
                .add_rule(PathBeneath::new(fd, read))
                .map_err(|err| err.to_string())?;
        }
        for path in &policy.read_write {
            let fd = PathFd::new(path).map_err(|err| format!("{}: {err}", path.display()))?;
            ruleset = ruleset
                .add_rule(PathBeneath::new(fd, all))
                .map_err(|err| err.to_string())?;
        }
        let status = ruleset.restrict_self().map_err(|err| err.to_string())?;
        scope_signals()?;
        Ok(match status.ruleset {
            RulesetStatus::FullyEnforced => LandlockStatus::Full,
            RulesetStatus::PartiallyEnforced if kernel_enforces(required_abi(policy)) => {
                LandlockStatus::Full
            }
            RulesetStatus::PartiallyEnforced => LandlockStatus::Partial,
            RulesetStatus::NotEnforced => LandlockStatus::None,
        })
    }

    /// Keeps signals from reaching any process outside this child, including
    /// the ones `fcntl` can arrange. It is a ruleset of its own because not
    /// every kernel the pool runs on has it: seccomp already holds `tgkill`
    /// to the child itself, so a kernel without this is no reason to report
    /// the filesystem rules as partial and refuse the child.
    fn scope_signals() -> Result<(), String> {
        Ruleset::default()
            .scope(Scope::Signal)
            .map_err(|err| err.to_string())?
            .create()
            .map_err(|err| err.to_string())?
            .restrict_self()
            .map(|_| ())
            .map_err(|err| err.to_string())
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[cfg(target_os = "linux")]
    #[test]
    fn seccomp_lowers_the_landlock_abi_the_policy_needs() {
        use ::landlock::ABI;

        let both = HardeningPolicy {
            landlock: true,
            seccomp: true,
            ..HardeningPolicy::default()
        };
        assert_eq!(landlock::required_abi(&both), ABI::V3);
        let alone = HardeningPolicy {
            seccomp: false,
            ..both
        };
        assert_eq!(landlock::required_abi(&alone), ABI::V5);
    }

    #[test]
    fn a_report_short_of_the_policy_does_not_satisfy_it() {
        let policy = HardeningPolicy {
            landlock: true,
            seccomp: true,
            ..HardeningPolicy::default()
        };
        let mut report = HardeningReport {
            no_new_privs: true,
            landlock: LandlockStatus::Full,
            seccomp: true,
            address_space_limit: 1 << 30,
            problems: Vec::new(),
        };
        assert!(report.satisfies(&policy));
        report.landlock = LandlockStatus::Partial;
        assert!(!report.satisfies(&policy));
        report.landlock = LandlockStatus::Full;
        report.address_space_limit = 0;
        assert!(!report.satisfies(&policy));
    }
}
