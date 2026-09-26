//! The Data Source Manager, as the session sees it.
//!
//! On Windows this is `twaindsm.dll` ([`crate::win::LibraryDsm`]). Everything
//! above it speaks only this trait, which is what lets the session, the
//! capability negotiation and the memory transfer run in unit tests against a
//! scripted source.

use core::ffi::c_void;
use core::ptr::NonNull;

use crate::sys::{TW_ENTRYPOINT, TW_HANDLE, TW_IDENTITY};

/// A TWAIN Data Source Manager.
pub trait Dsm {
    /// `DSM_Entry`.
    ///
    /// # Safety
    ///
    /// The pointers must be valid for the operation the triplet names, for
    /// the whole call, exactly as the TWAIN specification requires of an
    /// application calling `DSM_Entry`.
    unsafe fn entry(
        &self,
        origin: *mut TW_IDENTITY,
        dest: *mut TW_IDENTITY,
        dg: u32,
        dat: u16,
        msg: u16,
        data: *mut c_void,
    ) -> u16;

    /// Takes the memory functions a TWAIN 2 DSM hands out after it opens.
    /// Until then (and for a DSM that never does) the platform's own
    /// allocator is used, as TWAIN 1.x requires.
    fn use_entry_points(&self, entry_points: &TW_ENTRYPOINT);

    /// Allocates `size` bytes the way the DSM expects capability containers
    /// to be allocated. Null when out of memory.
    fn alloc(&self, size: u32) -> TW_HANDLE;

    /// # Safety
    ///
    /// `handle` came from [`Dsm::alloc`] or from a data source, and is not
    /// used again.
    unsafe fn free(&self, handle: TW_HANDLE);

    /// # Safety
    ///
    /// `handle` came from [`Dsm::alloc`] or from a data source and is live.
    unsafe fn lock(&self, handle: TW_HANDLE) -> *mut c_void;

    /// # Safety
    ///
    /// `handle` is locked.
    unsafe fn unlock(&self, handle: TW_HANDLE);
}

/// Memory handed between the application and a source through the DSM, freed
/// when dropped.
pub(crate) struct DsmMemory<'d, D: Dsm + ?Sized> {
    dsm: &'d D,
    handle: NonNull<c_void>,
}

impl<'d, D: Dsm + ?Sized> DsmMemory<'d, D> {
    /// Allocates zeroed memory.
    pub(crate) fn alloc(dsm: &'d D, size: usize) -> Option<Self> {
        let size32 = u32::try_from(size).ok()?;
        let handle = NonNull::new(dsm.alloc(size32))?;
        let memory = Self { dsm, handle };
        memory.with(|ptr| {
            // SAFETY: the allocation is at least `size` bytes and locked for
            // the duration of `with`.
            unsafe { core::ptr::write_bytes(ptr.cast::<u8>(), 0, size) };
        })?;
        Some(memory)
    }

    /// Takes ownership of a handle a source allocated. Null is `None`.
    ///
    /// # Safety
    ///
    /// `handle` was allocated by the source through the DSM's allocator and
    /// the caller is now responsible for freeing it.
    pub(crate) unsafe fn adopt(dsm: &'d D, handle: TW_HANDLE) -> Option<Self> {
        NonNull::new(handle).map(|handle| Self { dsm, handle })
    }

    pub(crate) fn handle(&self) -> TW_HANDLE {
        self.handle.as_ptr()
    }

    /// Runs `f` with the memory locked. `None` if it could not be locked.
    pub(crate) fn with<R>(&self, f: impl FnOnce(*mut c_void) -> R) -> Option<R> {
        // SAFETY: the handle is live for as long as `self` is.
        let ptr = unsafe { self.dsm.lock(self.handle.as_ptr()) };
        if ptr.is_null() {
            return None;
        }
        let result = f(ptr);
        // SAFETY: locked just above.
        unsafe { self.dsm.unlock(self.handle.as_ptr()) };
        Some(result)
    }
}

impl<D: Dsm + ?Sized> Drop for DsmMemory<'_, D> {
    fn drop(&mut self) {
        // SAFETY: the handle is owned by this value and freed exactly once.
        unsafe { self.dsm.free(self.handle.as_ptr()) };
    }
}

impl<D: Dsm + ?Sized> core::fmt::Debug for DsmMemory<'_, D> {
    fn fmt(&self, f: &mut core::fmt::Formatter<'_>) -> core::fmt::Result {
        f.debug_struct("DsmMemory")
            .field("handle", &self.handle)
            .finish()
    }
}
