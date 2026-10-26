//! The chassis every Trenova WebAssembly module runs on.
//!
//! A module (`trenova-rules`, `trenova-edi`) is pure computation behind three exports. The host
//! writes a JSON request `{"op": "<area>.<name>", "input": {...}}` into memory it got from
//! `<prefix>_alloc`, calls `<prefix>_call`, reads the response at the address and length packed
//! into the `u64` it returns, and hands both buffers back with `<prefix>_free`. The Go host
//! (`services/tms/pkg/wasmmodule`) and the browser loaders do exactly this for every module, so
//! the protocol is written once here:
//!
//! - [`message`]: user-facing text as English ICU templates and their arguments;
//! - [`ops`]: the request envelope, the response envelope and request errors, and dispatch over
//!   a module's operation table;
//! - [`export_abi!`]: the three exports, named for the module.

pub mod message;
pub mod ops;

/// Defines a module's three exports, `<prefix>_alloc`, `<prefix>_call` and `<prefix>_free`,
/// around its dispatch function (`fn(&[u8]) -> Vec<u8>`).
///
/// The exports exist only on `wasm32`, where a pointer fits the `u32` halves of the packed
/// result; native tests call the dispatch function directly. The host frees the request buffer
/// as well as the response; the module never frees a buffer behind the host's back.
#[macro_export]
macro_rules! export_abi {
    ($alloc:ident, $call:ident, $free:ident, $dispatch:path) => {
        /// Allocates `len` bytes inside the module for the host to write a request into.
        ///
        /// Returns a null pointer when `len` is zero.
        #[unsafe(no_mangle)]
        pub extern "C" fn $alloc(len: u32) -> *mut u8 {
            if len == 0 {
                return ::std::ptr::null_mut();
            }
            let mut buffer = vec![0u8; len as usize].into_boxed_slice();
            let pointer = buffer.as_mut_ptr();
            ::std::mem::forget(buffer);
            pointer
        }

        /// Returns a buffer from the alloc export or a response from the call export.
        ///
        /// # Safety
        ///
        /// `pointer` and `len` must describe exactly one buffer this module handed out and the
        /// host has not freed yet. A null pointer or zero length is ignored.
        #[unsafe(no_mangle)]
        pub unsafe extern "C" fn $free(pointer: *mut u8, len: u32) {
            if pointer.is_null() || len == 0 {
                return;
            }
            // SAFETY: the caller promises `pointer` and `len` came from one live boxed slice
            // this module leaked in the alloc or call export, so rebuilding and dropping it
            // frees it once.
            unsafe {
                drop(Box::from_raw(::std::ptr::slice_from_raw_parts_mut(
                    pointer,
                    len as usize,
                )));
            }
        }

        /// Runs one request and returns the response's address and length packed into a `u64`.
        ///
        /// # Safety
        ///
        /// `pointer` and `len` must describe initialised memory the host owns for the duration
        /// of the call (normally a buffer from the alloc export).
        #[unsafe(no_mangle)]
        pub unsafe extern "C" fn $call(pointer: *const u8, len: u32) -> u64 {
            let request: &[u8] = if pointer.is_null() || len == 0 {
                &[]
            } else {
                // SAFETY: the caller promises `pointer..pointer + len` is initialised and
                // stays valid while this call runs.
                unsafe { ::std::slice::from_raw_parts(pointer, len as usize) }
            };
            let response = $dispatch(request).into_boxed_slice();
            let response_len = u32::try_from(response.len()).unwrap_or(u32::MAX);
            let response_pointer = Box::into_raw(response).cast::<u8>();
            let address = u32::try_from(response_pointer as usize).unwrap_or(0);
            (u64::from(address) << 32) | u64::from(response_len)
        }
    };
}
