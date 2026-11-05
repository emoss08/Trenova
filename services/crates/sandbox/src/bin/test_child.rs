//! A child for the sandbox's own tests: each job's body names a behaviour,
//! from a well-behaved echo to the ways a hostile parser fails.

use std::io::Read;
use std::process::ExitCode;

use trenova_sandbox::Child;

/// What the sandbox lets a child do, asked without doing harm.
#[cfg(target_os = "linux")]
mod probes {
    use std::ptr;

    const PAGE: usize = 4096;

    /// Whether `tgkill` reaches `pid`. Signal 0 delivers nothing.
    pub fn may_signal(pid: u32) -> bool {
        let Ok(pid) = libc::pid_t::try_from(pid) else {
            return false;
        };
        // SAFETY: signal 0 only checks that the signal could be sent.
        unsafe { libc::syscall(libc::SYS_tgkill, pid, pid, 0) == 0 }
    }

    fn anonymous(protection: libc::c_int) -> Option<*mut libc::c_void> {
        // SAFETY: a new private mapping, at an address the kernel chooses.
        let mapped = unsafe {
            libc::mmap(
                ptr::null_mut(),
                PAGE,
                protection,
                libc::MAP_PRIVATE | libc::MAP_ANONYMOUS,
                -1,
                0,
            )
        };
        (mapped != libc::MAP_FAILED).then_some(mapped)
    }

    fn release(mapped: *mut libc::c_void) {
        // SAFETY: a mapping of `PAGE` bytes from `anonymous`, used by nothing.
        unsafe { libc::munmap(mapped, PAGE) };
    }

    pub fn may_map_executable() -> bool {
        anonymous(libc::PROT_READ | libc::PROT_EXEC)
            .map(release)
            .is_some()
    }

    pub fn may_protect_executable() -> bool {
        let Some(mapped) = anonymous(libc::PROT_READ | libc::PROT_WRITE) else {
            return false;
        };
        // SAFETY: the page mapped above, which nothing else refers to.
        let protected = unsafe { libc::mprotect(mapped, PAGE, libc::PROT_READ | libc::PROT_EXEC) };
        release(mapped);
        protected == 0
    }

    /// Whether any descriptor is a memory file, the only kind with seals.
    pub fn holds_a_sealed_file() -> bool {
        // SAFETY: `F_GET_SEALS` reads a descriptor's seals and takes no
        // argument; a number that is no descriptor just fails.
        (0..64).any(|fd| unsafe { libc::fcntl(fd, libc::F_GET_SEALS) } >= 0)
    }

    pub fn may_read_thread_name() -> bool {
        let mut name = [0u8; 16];
        // SAFETY: `PR_GET_NAME` writes at most 16 bytes, the buffer's size.
        unsafe { libc::prctl(libc::PR_GET_NAME, name.as_mut_ptr()) == 0 }
    }

    pub fn may_read_dumpable() -> bool {
        // SAFETY: `PR_GET_DUMPABLE` takes nothing and changes nothing.
        unsafe { libc::prctl(libc::PR_GET_DUMPABLE) >= 0 }
    }
}

fn main() -> ExitCode {
    let Ok((mut child, policy, _)) = Child::start(1 << 20) else {
        return ExitCode::from(2);
    };
    if child.harden_and_ready(&policy).is_err() {
        return ExitCode::from(3);
    }
    #[cfg(target_os = "linux")]
    let answer = |child: &mut Child, allowed: bool| {
        if allowed {
            child.done(b"allowed")
        } else {
            child.failed(b"refused")
        }
    };
    while let Ok(Some(job)) = child.next_job() {
        let body = String::from_utf8_lossy(&job.body).into_owned();
        let input = job
            .input
            .as_ref()
            .map(|input| input.as_bytes().to_vec())
            .unwrap_or_default();
        let sent = match body.split_once(':').unwrap_or((body.as_str(), "")) {
            ("echo", _) => child.output(&input).and_then(|()| child.done(b"echoed")),
            ("outputs", count) => {
                let count: usize = count.parse().unwrap_or(0);
                let mut result = Ok(());
                for i in 0..count {
                    result = child.output(&i.to_le_bytes());
                    if result.is_err() {
                        break;
                    }
                }
                result.and_then(|()| child.done(b"streamed"))
            }
            ("pid", _) => child.done(std::process::id().to_string().as_bytes()),
            ("fail", _) => child.failed(b"typed failure"),
            ("abort", _) => std::process::abort(),
            ("segv", _) => {
                // SAFETY: deliberately not safe: this job tests that a child
                // faulting on a wild write is contained by the pool.
                unsafe { std::ptr::null_mut::<u8>().write_volatile(1) };
                Ok(())
            }
            ("hang", _) => loop {
                std::hint::spin_loop();
            },
            ("alloc", _) => {
                let mut hoard: Vec<Vec<u8>> = Vec::new();
                loop {
                    let mut block = vec![0u8; 64 << 20];
                    block[0] = 1;
                    hoard.push(block);
                }
            }
            ("socket", _) => {
                let _ = std::net::TcpStream::connect("127.0.0.1:9");
                child.done(b"connected")
            }
            #[cfg(target_os = "linux")]
            ("signal", target) => {
                let pid = match target {
                    "self" => std::process::id(),
                    other => other.parse().unwrap_or(0),
                };
                answer(&mut child, probes::may_signal(pid))
            }
            #[cfg(target_os = "linux")]
            ("map-executable", _) => answer(&mut child, probes::may_map_executable()),
            #[cfg(target_os = "linux")]
            ("protect-executable", _) => answer(&mut child, probes::may_protect_executable()),
            #[cfg(target_os = "linux")]
            ("input-descriptor", _) => answer(&mut child, probes::holds_a_sealed_file()),
            #[cfg(target_os = "linux")]
            ("prctl-name", _) => answer(&mut child, probes::may_read_thread_name()),
            #[cfg(target_os = "linux")]
            ("prctl-dumpable", _) => answer(&mut child, probes::may_read_dumpable()),
            ("read", path) => {
                let mut text = String::new();
                match std::fs::File::open(path).and_then(|mut f| f.read_to_string(&mut text)) {
                    Ok(_) => child.done(b"read"),
                    Err(_) => child.failed(b"denied"),
                }
            }
            _ => child.failed(b"unknown behaviour"),
        };
        if sent.is_err() {
            return ExitCode::from(4);
        }
    }
    ExitCode::SUCCESS
}
