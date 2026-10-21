//! Arbitrary bytes on the channel: the reader returns frames within the
//! limit or an error, and never panics or allocates past the limit.

#![no_main]

use std::io::Write;
use std::os::unix::net::UnixStream;

use libfuzzer_sys::fuzz_target;
use trenova_sandbox::frame::Channel;

const LIMIT: u32 = 1 << 16;

fuzz_target!(|data: &[u8]| {
    let Ok((mut tx, rx)) = UnixStream::pair() else {
        return;
    };
    if data.len() > 1 << 16 || tx.write_all(data).is_err() {
        return;
    }
    drop(tx);
    let mut channel = Channel::new(rx, LIMIT);
    while let Ok(frame) = channel.recv() {
        assert!(frame.body.len() <= LIMIT as usize);
        assert!(frame.fds.is_empty());
    }
});
