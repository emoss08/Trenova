//! The framed protocol on the socket between the supervisor and a child.
//!
//! A frame is an eight-byte header followed by its body:
//!
//! ```text
//! magic  u16 LE  0x5354 ("TS")
//! kind   u8      what the frame is (see [`Kind`])
//! flags  u8      bit 0: file descriptors ride on the header
//! length u32 LE  body bytes, at most the configured frame limit
//! ```
//!
//! File descriptors (the sealed memfd a job's input is in) are sent as
//! `SCM_RIGHTS` ancillary data on the `sendmsg` that carries the header, so
//! the receiver always finds them on the read that returns the header's
//! first byte. Bodies are opaque here; the service built on the sandbox
//! encodes its own messages in them.
//!
//! The reader trusts nothing: a bad magic, an unknown kind, a length over
//! the limit, descriptors on a frame that may not carry them, or a stream
//! that ends inside a frame is a protocol error, and the supervisor kills a
//! child that commits one.

use std::io::{self, IoSlice, IoSliceMut, Read, Write};
use std::mem::MaybeUninit;
use std::os::fd::{AsFd, BorrowedFd, OwnedFd};
use std::os::unix::net::UnixStream;

use rustix::net::{
    RecvAncillaryBuffer, RecvAncillaryMessage, RecvFlags, SendAncillaryBuffer,
    SendAncillaryMessage, SendFlags, recvmsg, sendmsg,
};

const MAGIC: u16 = 0x5354;
const HEADER: usize = 8;
const FLAG_FDS: u8 = 1;
/// The most descriptors one frame may carry.
pub const MAX_FDS: usize = 4;

/// What a frame is.
#[derive(Clone, Copy, Debug, PartialEq, Eq)]
#[repr(u8)]
pub enum Kind {
    /// Supervisor → child, first frame: the hardening policy and the
    /// service's own start-up parameters.
    Init = 1,
    /// Child → supervisor, once initialised and hardened: what protections
    /// are in force.
    Ready = 2,
    /// Supervisor → child: a job, with its input as a descriptor.
    Job = 3,
    /// Child → supervisor: one piece of a job's output.
    Output = 4,
    /// Child → supervisor: the job finished; its summary.
    Done = 5,
    /// Child → supervisor: the job failed; the service's typed error.
    Failed = 6,
    /// Supervisor → child: exit now.
    Shutdown = 7,
}

impl Kind {
    fn from_byte(byte: u8) -> Option<Self> {
        Some(match byte {
            1 => Self::Init,
            2 => Self::Ready,
            3 => Self::Job,
            4 => Self::Output,
            5 => Self::Done,
            6 => Self::Failed,
            7 => Self::Shutdown,
            _ => return None,
        })
    }

    fn may_carry_fds(self) -> bool {
        matches!(self, Self::Job)
    }
}

/// A received frame.
#[derive(Debug)]
pub struct Frame {
    pub kind: Kind,
    pub body: Vec<u8>,
    pub fds: Vec<OwnedFd>,
}

#[derive(Debug, thiserror::Error)]
pub enum FrameError {
    #[error("the peer closed the channel")]
    Closed,
    #[error("i/o on the channel failed: {0}")]
    Io(#[from] io::Error),
    #[error("protocol violation: {0}")]
    Protocol(&'static str),
    #[error("a frame of {length} bytes is over the {limit}-byte limit")]
    TooLarge { length: u64, limit: u32 },
}

/// One end of the channel, blocking.
#[derive(Debug)]
pub struct Channel {
    stream: UnixStream,
    max_body: u32,
}

impl Channel {
    pub fn new(stream: UnixStream, max_body: u32) -> Self {
        Self { stream, max_body }
    }

    pub fn stream(&self) -> &UnixStream {
        &self.stream
    }

    /// Sends a frame, with `fds` as ancillary data on its header.
    pub fn send(
        &mut self,
        kind: Kind,
        body: &[u8],
        fds: &[BorrowedFd<'_>],
    ) -> Result<(), FrameError> {
        let length = u32::try_from(body.len()).map_err(|_| FrameError::TooLarge {
            length: body.len() as u64,
            limit: self.max_body,
        })?;
        if length > self.max_body {
            return Err(FrameError::TooLarge {
                length: u64::from(length),
                limit: self.max_body,
            });
        }
        if fds.len() > MAX_FDS || (!fds.is_empty() && !kind.may_carry_fds()) {
            return Err(FrameError::Protocol(
                "descriptors on a frame that cannot carry them",
            ));
        }

        let mut header = [0u8; HEADER];
        header[0..2].copy_from_slice(&MAGIC.to_le_bytes());
        header[2] = kind as u8;
        header[3] = if fds.is_empty() { 0 } else { FLAG_FDS };
        header[4..8].copy_from_slice(&length.to_le_bytes());

        if fds.is_empty() {
            self.stream.write_all(&header)?;
        } else {
            let mut space = [MaybeUninit::<u8>::uninit(); rustix::cmsg_space!(ScmRights(MAX_FDS))];
            let mut control = SendAncillaryBuffer::new(&mut space);
            if !control.push(SendAncillaryMessage::ScmRights(fds)) {
                return Err(FrameError::Protocol(
                    "descriptors do not fit the control buffer",
                ));
            }
            let mut sent = 0;
            while sent < HEADER {
                let n = if sent == 0 {
                    sendmsg(
                        self.stream.as_fd(),
                        &[IoSlice::new(&header)],
                        &mut control,
                        SendFlags::NOSIGNAL,
                    )
                } else {
                    sendmsg(
                        self.stream.as_fd(),
                        &[IoSlice::new(&header[sent..])],
                        &mut SendAncillaryBuffer::default(),
                        SendFlags::NOSIGNAL,
                    )
                }
                .map_err(io::Error::from)?;
                if n == 0 {
                    return Err(FrameError::Closed);
                }
                sent += n;
            }
        }
        self.stream.write_all(body)?;
        Ok(())
    }

    /// Receives the next frame.
    pub fn recv(&mut self) -> Result<Frame, FrameError> {
        let mut header = [0u8; HEADER];
        let mut space = [MaybeUninit::<u8>::uninit(); rustix::cmsg_space!(ScmRights(MAX_FDS))];
        let mut control = RecvAncillaryBuffer::new(&mut space);
        let mut fds = Vec::new();

        let mut filled = 0;
        while filled < HEADER {
            let received = recvmsg(
                self.stream.as_fd(),
                &mut [IoSliceMut::new(&mut header[filled..])],
                &mut control,
                RecvFlags::CMSG_CLOEXEC,
            )
            .map_err(io::Error::from)?;
            if received.flags.contains(rustix::net::ReturnFlags::CTRUNC) {
                return Err(FrameError::Protocol(
                    "more descriptors than a frame may carry",
                ));
            }
            for message in control.drain() {
                if let RecvAncillaryMessage::ScmRights(rights) = message {
                    fds.extend(rights);
                }
            }
            if received.bytes == 0 {
                return Err(if filled == 0 {
                    FrameError::Closed
                } else {
                    FrameError::Protocol("the channel closed inside a frame header")
                });
            }
            filled += received.bytes;
        }

        if u16::from_le_bytes([header[0], header[1]]) != MAGIC {
            return Err(FrameError::Protocol("bad frame magic"));
        }
        let kind = Kind::from_byte(header[2]).ok_or(FrameError::Protocol("unknown frame kind"))?;
        let flagged = header[3] & FLAG_FDS != 0;
        if header[3] & !FLAG_FDS != 0 {
            return Err(FrameError::Protocol("unknown frame flags"));
        }
        if flagged == fds.is_empty()
            || (!fds.is_empty() && !kind.may_carry_fds())
            || fds.len() > MAX_FDS
        {
            return Err(FrameError::Protocol("descriptors do not match the frame"));
        }
        let length = u32::from_le_bytes([header[4], header[5], header[6], header[7]]);
        if length > self.max_body {
            return Err(FrameError::TooLarge {
                length: u64::from(length),
                limit: self.max_body,
            });
        }

        let mut body = vec![0u8; length as usize];
        self.stream.read_exact(&mut body).map_err(|err| {
            if err.kind() == io::ErrorKind::UnexpectedEof {
                FrameError::Protocol("the channel closed inside a frame body")
            } else {
                FrameError::Io(err)
            }
        })?;

        Ok(Frame { kind, body, fds })
    }
}

/// Parses a frame header from untrusted bytes, for the fuzz target: returns
/// the kind and length a reader would accept, or why it would not.
pub fn parse_header(bytes: &[u8; HEADER], max_body: u32) -> Result<(Kind, u32, bool), FrameError> {
    if u16::from_le_bytes([bytes[0], bytes[1]]) != MAGIC {
        return Err(FrameError::Protocol("bad frame magic"));
    }
    let kind = Kind::from_byte(bytes[2]).ok_or(FrameError::Protocol("unknown frame kind"))?;
    if bytes[3] & !FLAG_FDS != 0 {
        return Err(FrameError::Protocol("unknown frame flags"));
    }
    let length = u32::from_le_bytes([bytes[4], bytes[5], bytes[6], bytes[7]]);
    if length > max_body {
        return Err(FrameError::TooLarge {
            length: u64::from(length),
            limit: max_body,
        });
    }
    Ok((kind, length, bytes[3] & FLAG_FDS != 0))
}

#[cfg(test)]
mod tests {
    use std::io::Seek;

    use proptest::prelude::*;

    use super::*;

    fn pair(limit: u32) -> (Channel, Channel) {
        let (a, b) = UnixStream::pair().expect("socketpair");
        (Channel::new(a, limit), Channel::new(b, limit))
    }

    #[test]
    fn a_frame_and_its_descriptor_arrive_together() {
        let (mut tx, mut rx) = pair(1 << 20);
        let mut file = tempfile_in_memory();
        file.write_all(b"input").expect("write");
        file.rewind().expect("rewind");
        tx.send(Kind::Job, b"job", &[file.as_fd()]).expect("send");
        let frame = rx.recv().expect("recv");
        assert_eq!(frame.kind, Kind::Job);
        assert_eq!(frame.body, b"job");
        assert_eq!(frame.fds.len(), 1);
        let mut received = std::fs::File::from(frame.fds.into_iter().next().expect("fd"));
        let mut text = String::new();
        received.read_to_string(&mut text).expect("read");
        assert_eq!(text, "input");
    }

    #[test]
    fn a_frame_over_the_limit_is_refused_on_both_ends() {
        let (mut tx, mut rx) = pair(8);
        assert!(matches!(
            tx.send(Kind::Output, &[0; 9], &[]),
            Err(FrameError::TooLarge { .. })
        ));
        let mut header = [0u8; HEADER];
        header[0..2].copy_from_slice(&MAGIC.to_le_bytes());
        header[2] = Kind::Output as u8;
        header[4..8].copy_from_slice(&100u32.to_le_bytes());
        tx.stream.write_all(&header).expect("raw write");
        assert!(matches!(rx.recv(), Err(FrameError::TooLarge { .. })));
    }

    #[test]
    fn descriptors_on_a_frame_that_cannot_carry_them_are_refused() {
        let (mut tx, _rx) = pair(64);
        let file = tempfile_in_memory();
        assert!(matches!(
            tx.send(Kind::Output, b"", &[file.as_fd()]),
            Err(FrameError::Protocol(_))
        ));
    }

    #[test]
    fn a_stream_that_ends_inside_a_frame_is_a_protocol_error() {
        let (mut tx, mut rx) = pair(64);
        let mut header = [0u8; HEADER];
        header[0..2].copy_from_slice(&MAGIC.to_le_bytes());
        header[2] = Kind::Done as u8;
        header[4..8].copy_from_slice(&10u32.to_le_bytes());
        tx.stream.write_all(&header).expect("raw write");
        tx.stream.write_all(b"short").expect("raw write");
        drop(tx);
        assert!(matches!(rx.recv(), Err(FrameError::Protocol(_))));
    }

    fn tempfile_in_memory() -> std::fs::File {
        let fd =
            rustix::fs::memfd_create("frame-test", rustix::fs::MemfdFlags::CLOEXEC).expect("memfd");
        std::fs::File::from(fd)
    }

    proptest! {
        #[test]
        fn any_header_either_parses_within_the_limit_or_is_refused(bytes in any::<[u8; 8]>(), limit in 0u32..1 << 20) {
            if let Ok((_, length, _)) = parse_header(&bytes, limit) {
                prop_assert!(length <= limit);
            }
        }

        #[test]
        fn bodies_round_trip(body in proptest::collection::vec(any::<u8>(), 0..4096)) {
            let (mut tx, mut rx) = pair(1 << 20);
            tx.send(Kind::Output, &body, &[]).expect("send");
            let frame = rx.recv().expect("recv");
            prop_assert_eq!(frame.body, body);
        }
    }
}
