//! The seccomp filters a child installs.
//!
//! Three filters are stacked; the kernel runs every one and applies the most
//! severe answer:
//!
//! 1. A kill list: system calls a document parser never has a reason to
//!    make (opening a socket, running a program, tracing or reaching into
//!    another process, namespaces, mounts, BPF, keys, `io_uring`). Any of
//!    them kills the process at once, so an exploit's first step is a crash
//!    the supervisor counts, not a quiet failure.
//! 2. An allow list of what the child does need: memory, reading its
//!    mapped input and the files landlock leaves it, the channel to the
//!    supervisor (`sendmsg`/`recvmsg` on the socket it already holds; it
//!    cannot make another), threads, time, signals for its own abort,
//!    reading (never raising) its resource limits. Anything else fails with
//!    `EPERM`.
//! 3. `clone3` answers `ENOSYS`. Its flags live in memory a filter cannot
//!    read, so it cannot be checked; glibc then falls back to `clone`, whose
//!    flags filter 2 checks: only threads, never new processes.

use std::collections::BTreeMap;

use seccompiler::{
    BpfProgram, SeccompAction, SeccompCmpArgLen, SeccompCmpOp, SeccompCondition, SeccompFilter,
    SeccompRule, TargetArch,
};

/// Calls that kill the process.
const KILL: &[i64] = &[
    libc::SYS_socket,
    libc::SYS_socketpair,
    libc::SYS_connect,
    libc::SYS_bind,
    libc::SYS_listen,
    libc::SYS_accept,
    libc::SYS_accept4,
    libc::SYS_execve,
    libc::SYS_execveat,
    libc::SYS_ptrace,
    libc::SYS_process_vm_readv,
    libc::SYS_process_vm_writev,
    libc::SYS_pidfd_getfd,
    libc::SYS_mount,
    libc::SYS_umount2,
    libc::SYS_pivot_root,
    libc::SYS_chroot,
    libc::SYS_unshare,
    libc::SYS_setns,
    libc::SYS_bpf,
    libc::SYS_perf_event_open,
    libc::SYS_keyctl,
    libc::SYS_add_key,
    libc::SYS_request_key,
    libc::SYS_userfaultfd,
    libc::SYS_io_uring_setup,
    libc::SYS_io_uring_enter,
    libc::SYS_io_uring_register,
    libc::SYS_kexec_load,
    libc::SYS_init_module,
    libc::SYS_finit_module,
    libc::SYS_delete_module,
    libc::SYS_setuid,
    libc::SYS_setgid,
    libc::SYS_setreuid,
    libc::SYS_setregid,
    libc::SYS_setresuid,
    libc::SYS_setresgid,
];

/// Calls allowed without conditions.
const ALLOW: &[i64] = &[
    libc::SYS_read,
    libc::SYS_write,
    libc::SYS_readv,
    libc::SYS_writev,
    libc::SYS_pread64,
    libc::SYS_lseek,
    libc::SYS_close,
    libc::SYS_fstat,
    libc::SYS_newfstatat,
    libc::SYS_statx,
    libc::SYS_openat,
    libc::SYS_getdents64,
    libc::SYS_readlinkat,
    libc::SYS_faccessat,
    libc::SYS_faccessat2,
    libc::SYS_getcwd,
    libc::SYS_fcntl,
    libc::SYS_dup,
    libc::SYS_dup3,
    libc::SYS_mmap,
    libc::SYS_munmap,
    libc::SYS_mremap,
    libc::SYS_mprotect,
    libc::SYS_madvise,
    libc::SYS_brk,
    libc::SYS_futex,
    libc::SYS_set_robust_list,
    libc::SYS_get_robust_list,
    libc::SYS_rseq,
    libc::SYS_set_tid_address,
    libc::SYS_clone3,
    libc::SYS_sched_yield,
    libc::SYS_sched_getaffinity,
    libc::SYS_rt_sigaction,
    libc::SYS_rt_sigprocmask,
    libc::SYS_rt_sigreturn,
    libc::SYS_sigaltstack,
    libc::SYS_getpid,
    libc::SYS_gettid,
    libc::SYS_getuid,
    libc::SYS_geteuid,
    libc::SYS_getgid,
    libc::SYS_getegid,
    libc::SYS_tgkill,
    libc::SYS_clock_gettime,
    libc::SYS_clock_getres,
    libc::SYS_clock_nanosleep,
    libc::SYS_nanosleep,
    libc::SYS_gettimeofday,
    libc::SYS_getrandom,
    libc::SYS_sendmsg,
    libc::SYS_sendto,
    libc::SYS_recvmsg,
    libc::SYS_recvfrom,
    libc::SYS_shutdown,
    libc::SYS_ppoll,
    libc::SYS_pselect6,
    libc::SYS_uname,
    libc::SYS_sysinfo,
    libc::SYS_getrusage,
    #[cfg(target_arch = "x86_64")]
    libc::SYS_getrlimit,
    libc::SYS_prctl,
    libc::SYS_membarrier,
    libc::SYS_exit,
    libc::SYS_exit_group,
    #[cfg(target_arch = "x86_64")]
    libc::SYS_arch_prctl,
    #[cfg(target_arch = "x86_64")]
    libc::SYS_poll,
    #[cfg(target_arch = "x86_64")]
    libc::SYS_open,
    #[cfg(target_arch = "x86_64")]
    libc::SYS_stat,
    #[cfg(target_arch = "x86_64")]
    libc::SYS_lstat,
    #[cfg(target_arch = "x86_64")]
    libc::SYS_access,
    #[cfg(target_arch = "x86_64")]
    libc::SYS_readlink,
    #[cfg(target_arch = "x86_64")]
    libc::SYS_getdents,
    #[cfg(target_arch = "x86_64")]
    libc::SYS_time,
];

#[derive(Debug, thiserror::Error)]
pub enum SeccompError {
    #[error("this architecture has no seccomp filter")]
    Arch,
    #[error("build the filter: {0}")]
    Build(String),
    #[error("install the filter: {0}")]
    Install(String),
}

fn arch() -> Result<TargetArch, SeccompError> {
    TargetArch::try_from(std::env::consts::ARCH).map_err(|_| SeccompError::Arch)
}

/// The three filters, compiled, in the order they are installed.
pub fn programs() -> Result<[BpfProgram; 3], SeccompError> {
    let arch = arch()?;
    let build = |rules: BTreeMap<i64, Vec<SeccompRule>>, mismatch, matched| {
        SeccompFilter::new(rules, mismatch, matched, arch)
            .and_then(BpfProgram::try_from)
            .map_err(|err| SeccompError::Build(err.to_string()))
    };

    let kill = build(
        KILL.iter().map(|&nr| (nr, Vec::new())).collect(),
        SeccompAction::Allow,
        SeccompAction::KillProcess,
    )?;

    let thread_clone = SeccompRule::new(vec![
        SeccompCondition::new(
            0,
            SeccompCmpArgLen::Qword,
            SeccompCmpOp::MaskedEq(clone_flag(libc::CLONE_THREAD)),
            clone_flag(libc::CLONE_THREAD),
        )
        .map_err(|err| SeccompError::Build(err.to_string()))?,
    ])
    .map_err(|err| SeccompError::Build(err.to_string()))?;
    let mut allowed: BTreeMap<i64, Vec<SeccompRule>> =
        ALLOW.iter().map(|&nr| (nr, Vec::new())).collect();
    allowed.insert(libc::SYS_clone, vec![thread_clone]);
    let read_limit_only = SeccompRule::new(vec![
        SeccompCondition::new(2, SeccompCmpArgLen::Qword, SeccompCmpOp::Eq, 0)
            .map_err(|err| SeccompError::Build(err.to_string()))?,
    ])
    .map_err(|err| SeccompError::Build(err.to_string()))?;
    allowed.insert(libc::SYS_prlimit64, vec![read_limit_only]);
    let allow = build(
        allowed,
        SeccompAction::Errno(u32::try_from(libc::EPERM).unwrap_or(1)),
        SeccompAction::Allow,
    )?;

    let no_clone3 = build(
        BTreeMap::from([(libc::SYS_clone3, Vec::new())]),
        SeccompAction::Allow,
        SeccompAction::Errno(u32::try_from(libc::ENOSYS).unwrap_or(38)),
    )?;

    Ok([no_clone3, kill, allow])
}

fn clone_flag(flag: libc::c_int) -> u64 {
    u64::try_from(flag).unwrap_or(0)
}

/// Installs the filters on every thread of the process. The allow list goes
/// in last: it does not allow `seccomp` itself, so nothing can be added
/// after it.
pub fn install() -> Result<(), SeccompError> {
    for program in programs()? {
        seccompiler::apply_filter_all_threads(&program)
            .map_err(|err| SeccompError::Install(err.to_string()))?;
    }
    Ok(())
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn the_filters_compile_for_this_architecture() {
        let programs = programs().expect("compiles");
        assert!(programs.iter().all(|p| !p.is_empty()));
    }

    #[test]
    fn nothing_is_both_allowed_and_killed() {
        for nr in KILL {
            assert!(!ALLOW.contains(nr), "syscall {nr} is on both lists");
        }
    }
}
