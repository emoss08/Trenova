//! The WebAssembly boundary: three exports and no imports.
//!
//! A host writes a UTF-8 JSON request into memory it got from [`rules_alloc`], calls
//! [`rules_call`] with its address and length, and gets back one `u64`: the response's address in
//! the high 32 bits and its length in the low 32. It reads the response, then hands both buffers
//! back with [`rules_free`]. The Go host (`services/tms/pkg/rules`) and the browser loader
//! (`client/packages/shared/src/rules`) do exactly this and nothing else, so one artifact serves
//! both.
//!
//! The host frees the request buffer as well as the response; the module never frees a buffer
//! behind the host's back. The exports exist only on `wasm32`, where a pointer fits the `u32`
//! halves of the packed result; native tests call [`ops::dispatch`] directly.

use crate::ops;

/// Allocates `len` bytes inside the module for the host to write a request into.
///
/// Returns a null pointer when `len` is zero.
#[unsafe(no_mangle)]
pub extern "C" fn rules_alloc(len: u32) -> *mut u8 {
    if len == 0 {
        return std::ptr::null_mut();
    }
    let mut buffer = vec![0u8; len as usize].into_boxed_slice();
    let pointer = buffer.as_mut_ptr();
    std::mem::forget(buffer);
    pointer
}

/// Returns a buffer from [`rules_alloc`] or a response from [`rules_call`] to the module.
///
/// # Safety
///
/// `pointer` and `len` must describe exactly one buffer this module handed out and the host has
/// not freed yet. A null pointer or zero length is ignored.
#[unsafe(no_mangle)]
pub unsafe extern "C" fn rules_free(pointer: *mut u8, len: u32) {
    if pointer.is_null() || len == 0 {
        return;
    }
    // SAFETY: the caller promises `pointer` and `len` came from one live boxed slice this module
    // leaked in `rules_alloc` or `rules_call`, so rebuilding and dropping it frees it once.
    unsafe {
        drop(Box::from_raw(std::ptr::slice_from_raw_parts_mut(
            pointer,
            len as usize,
        )));
    }
}

/// Runs one request and returns the response's address and length packed into a `u64`.
///
/// # Safety
///
/// `pointer` and `len` must describe initialised memory the host owns for the duration of the
/// call (normally a buffer from [`rules_alloc`]).
#[unsafe(no_mangle)]
pub unsafe extern "C" fn rules_call(pointer: *const u8, len: u32) -> u64 {
    let request: &[u8] = if pointer.is_null() || len == 0 {
        &[]
    } else {
        // SAFETY: the caller promises `pointer..pointer + len` is initialised and stays valid
        // while this call runs.
        unsafe { std::slice::from_raw_parts(pointer, len as usize) }
    };
    let response = ops::dispatch(request).into_boxed_slice();
    let response_len = u32::try_from(response.len()).unwrap_or(u32::MAX);
    let response_pointer = Box::into_raw(response).cast::<u8>();
    let address = u32::try_from(response_pointer as usize).unwrap_or(0);
    (u64::from(address) << 32) | u64::from(response_len)
}
