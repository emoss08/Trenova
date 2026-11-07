//! A job's input, handed to a child as a sealed memfd.
//!
//! The supervisor writes the input into an anonymous memory file as it
//! arrives, so it is never one buffer on the supervisor's heap, then seals
//! it against writing, growing and shrinking, and passes the descriptor. The
//! child checks the seals and maps the file read-only: the parser reads the
//! bytes where they are, and nothing either side does can change them under
//! the other. Nothing touches a filesystem, so a read-only root and a
//! landlock policy with no writable paths cost nothing.
//!
//! The file holds bytes the document's author chose, so a child keeps no
//! descriptor for it once it is mapped: seccomp lets a file be mapped
//! executable, for the dynamic loader, and a child with no descriptor has
//! nothing to map. `MFD_NOEXEC_SEAL` is a separate guarantee, that the file
//! can never be made an executable program. It does not refuse an executable
//! mapping, so the descriptor has to go whatever the kernel supports.

use std::fs::File;
use std::io::{self, Write};
use std::os::fd::{AsFd, BorrowedFd, OwnedFd};
use std::ptr::NonNull;

use rustix::fs::{MemfdFlags, SealFlags, fcntl_add_seals, fcntl_get_seals, memfd_create};
use rustix::io::Errno;
use rustix::mm::{MapFlags, ProtFlags, mmap, munmap};

const REQUIRED_SEALS: SealFlags = SealFlags::SEAL
    .union(SealFlags::SHRINK)
    .union(SealFlags::GROW)
    .union(SealFlags::WRITE);

/// Creates the file through `create`, sealed against ever being executable.
/// A kernel before 6.3 answers `EINVAL` to that flag and gets the same file
/// without it.
fn sealed_against_execution(
    create: impl Fn(MemfdFlags) -> rustix::io::Result<OwnedFd>,
) -> io::Result<OwnedFd> {
    let flags = MemfdFlags::CLOEXEC | MemfdFlags::ALLOW_SEALING;
    match create(flags | MemfdFlags::NOEXEC_SEAL) {
        Err(Errno::INVAL) => Ok(create(flags)?),
        created => Ok(created?),
    }
}

/// An input being written. Every write is checked against the limit, so a
/// caller that streams more than it declared is stopped at the limit.
#[derive(Debug)]
pub struct InputWriter {
    file: File,
    written: u64,
    limit: u64,
}

impl InputWriter {
    pub fn new(name: &str, limit: u64) -> io::Result<Self> {
        let fd = sealed_against_execution(|flags| memfd_create(name, flags))?;
        Ok(Self {
            file: File::from(fd),
            written: 0,
            limit,
        })
    }

    /// Appends a chunk. Fails, writing nothing, once the limit would pass.
    pub fn write_chunk(&mut self, chunk: &[u8]) -> io::Result<()> {
        let next = self.written + chunk.len() as u64;
        if next > self.limit {
            return Err(io::Error::new(
                io::ErrorKind::FileTooLarge,
                format!("the input passes its {}-byte limit", self.limit),
            ));
        }
        self.file.write_all(chunk)?;
        self.written = next;
        Ok(())
    }

    pub fn len(&self) -> u64 {
        self.written
    }

    pub fn is_empty(&self) -> bool {
        self.written == 0
    }

    /// Seals the file so neither side can change it.
    pub fn seal(self) -> io::Result<SealedInput> {
        fcntl_add_seals(&self.file, REQUIRED_SEALS)?;
        Ok(SealedInput {
            fd: OwnedFd::from(self.file),
            len: self.written,
        })
    }
}

/// A sealed input, ready to pass to a child.
#[derive(Debug)]
pub struct SealedInput {
    fd: OwnedFd,
    len: u64,
}

impl SealedInput {
    pub fn len(&self) -> u64 {
        self.len
    }

    pub fn is_empty(&self) -> bool {
        self.len == 0
    }

    pub fn as_fd(&self) -> BorrowedFd<'_> {
        self.fd.as_fd()
    }

    /// A second descriptor for the same sealed file, so one input can go to
    /// several jobs.
    pub fn try_clone(&self) -> io::Result<Self> {
        Ok(Self {
            fd: self.fd.try_clone()?,
            len: self.len,
        })
    }
}

/// An input as a child sees it: mapped read-only.
#[derive(Debug)]
pub struct MappedInput {
    ptr: Option<NonNull<u8>>,
    len: usize,
}

// SAFETY: the mapping is read-only and backed by a sealed file whose contents
// and size cannot change, so sharing the bytes between threads is sound.
unsafe impl Send for MappedInput {}
// SAFETY: as above; no interior mutability is reachable through `&MappedInput`.
unsafe impl Sync for MappedInput {}

impl MappedInput {
    /// Maps a descriptor received from the supervisor, refusing one that is
    /// not sealed against writing, growing and shrinking. The descriptor is
    /// closed on return; the mapping does not need it.
    pub fn map(fd: OwnedFd) -> io::Result<Self> {
        let seals = fcntl_get_seals(&fd)?;
        if !seals.contains(REQUIRED_SEALS) {
            return Err(io::Error::new(
                io::ErrorKind::InvalidInput,
                "the input is not sealed",
            ));
        }
        let size = rustix::fs::fstat(&fd)?.st_size;
        let len = usize::try_from(size)
            .map_err(|_| io::Error::new(io::ErrorKind::InvalidInput, "the input has no size"))?;
        if len == 0 {
            return Ok(Self { ptr: None, len: 0 });
        }
        // SAFETY: a fresh read-only private mapping of `len` bytes of a sealed
        // file at offset 0; the kernel picks the address, and the file cannot
        // shrink under the mapping, so every byte of it stays readable until
        // `munmap` in `drop`.
        let ptr = unsafe {
            mmap(
                std::ptr::null_mut(),
                len,
                ProtFlags::READ,
                MapFlags::PRIVATE,
                &fd,
                0,
            )
        }?;
        drop(fd);
        Ok(Self {
            ptr: NonNull::new(ptr.cast::<u8>()),
            len,
        })
    }

    pub fn as_bytes(&self) -> &[u8] {
        match self.ptr {
            // SAFETY: `ptr` is the start of a live read-only mapping of `len`
            // bytes (see `map`), unmapped only when `self` is dropped.
            Some(ptr) => unsafe { std::slice::from_raw_parts(ptr.as_ptr(), self.len) },
            None => &[],
        }
    }
}

impl Drop for MappedInput {
    fn drop(&mut self) {
        if let Some(ptr) = self.ptr.take() {
            // SAFETY: `ptr` and `len` are exactly what `mmap` returned, and no
            // slice from `as_bytes` outlives `self`.
            let _ = unsafe { munmap(ptr.as_ptr().cast(), self.len) };
        }
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn written_input_is_mapped_back_exactly() {
        let mut writer = InputWriter::new("test", 1 << 20).expect("memfd");
        writer.write_chunk(b"hello, ").expect("write");
        writer.write_chunk(b"page").expect("write");
        let sealed = writer.seal().expect("seal");
        assert_eq!(sealed.len(), 11);
        let mapped = MappedInput::map(sealed.fd).expect("map");
        assert_eq!(mapped.as_bytes(), b"hello, page");
    }

    #[test]
    fn an_input_is_never_executable_where_the_kernel_can_promise_it() {
        let sealed = InputWriter::new("test", 64)
            .expect("memfd")
            .seal()
            .expect("seal");
        let seals = fcntl_get_seals(&sealed.fd).expect("seals");
        assert!(seals.contains(REQUIRED_SEALS));
        if seals.contains(SealFlags::EXEC) {
            let status = rustix::fs::fstat(&sealed.fd).expect("stat");
            assert_eq!(status.st_mode & 0o111, 0);
        }
    }

    #[test]
    fn a_kernel_without_the_execution_seal_still_gets_an_input() {
        let plain = MemfdFlags::CLOEXEC | MemfdFlags::ALLOW_SEALING;
        let asked = std::cell::RefCell::new(Vec::new());
        let created = sealed_against_execution(|flags| {
            asked.borrow_mut().push(flags);
            if flags.contains(MemfdFlags::NOEXEC_SEAL) {
                Err(Errno::INVAL)
            } else {
                memfd_create("test", flags)
            }
        });
        assert!(created.is_ok());
        assert_eq!(*asked.borrow(), [plain | MemfdFlags::NOEXEC_SEAL, plain]);
    }

    #[test]
    fn any_other_failure_to_create_an_input_is_reported() {
        let attempts = std::cell::Cell::new(0);
        let created = sealed_against_execution(|_| {
            attempts.set(attempts.get() + 1);
            Err(Errno::NOMEM)
        });
        assert!(created.is_err());
        assert_eq!(attempts.get(), 1);
    }

    #[test]
    fn writing_past_the_limit_is_stopped() {
        let mut writer = InputWriter::new("test", 4).expect("memfd");
        writer.write_chunk(b"abc").expect("fits");
        assert!(writer.write_chunk(b"de").is_err());
        assert_eq!(writer.len(), 3);
    }

    #[test]
    fn an_unsealed_descriptor_is_refused() {
        let fd = memfd_create("unsealed", MemfdFlags::CLOEXEC | MemfdFlags::ALLOW_SEALING)
            .expect("memfd");
        assert!(MappedInput::map(fd).is_err());
    }

    #[test]
    fn a_sealed_input_cannot_be_written_again() {
        let mut writer = InputWriter::new("test", 64).expect("memfd");
        writer.write_chunk(b"abc").expect("write");
        let sealed = writer.seal().expect("seal");
        let mut file = File::from(sealed.fd.try_clone().expect("dup"));
        assert!(file.write_all(b"x").is_err());
    }

    #[test]
    fn an_empty_input_maps_to_no_bytes() {
        let sealed = InputWriter::new("empty", 64)
            .expect("memfd")
            .seal()
            .expect("seal");
        let mapped = MappedInput::map(sealed.fd).expect("map");
        assert!(mapped.as_bytes().is_empty());
    }
}
