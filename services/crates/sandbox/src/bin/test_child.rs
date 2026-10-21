//! A child for the sandbox's own tests: each job's body names a behaviour,
//! from a well-behaved echo to the ways a hostile parser fails.

use std::io::Read;
use std::process::ExitCode;

use trenova_sandbox::Child;

fn main() -> ExitCode {
    let Ok((mut child, policy, _)) = Child::start(1 << 20) else {
        return ExitCode::from(2);
    };
    if child.harden_and_ready(&policy).is_err() {
        return ExitCode::from(3);
    }
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
