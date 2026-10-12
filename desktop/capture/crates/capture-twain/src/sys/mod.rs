//! Raw TWAIN 2.5 declarations, generated from the TWAIN Working Group's
//! `twain.h` by `tools/generate-bindings.sh`.
//!
//! `twain.h` packs every structure to two bytes on Windows, and its integer
//! types are Windows' own: `TW_UINT32` is an `unsigned long`, which is 32
//! bits there. [`ctypes`] pins the C types to those widths, so the same
//! declarations hold on the Linux machine the unit tests run on, and the
//! generated layout assertions check both files at compile time for the
//! target in use.

#![allow(
    non_camel_case_types,
    non_snake_case,
    non_upper_case_globals,
    dead_code,
    clippy::all,
    clippy::pedantic,
    missing_debug_implementations
)]

/// C types as Windows sizes them.
pub mod ctypes {
    pub type c_char = i8;
    pub type c_schar = i8;
    pub type c_uchar = u8;
    pub type c_short = i16;
    pub type c_ushort = u16;
    pub type c_int = i32;
    pub type c_uint = u32;
    pub type c_long = i32;
    pub type c_ulong = u32;
    pub type c_longlong = i64;
    pub type c_ulonglong = u64;
    pub type c_void = core::ffi::c_void;
}

#[cfg(target_pointer_width = "64")]
mod generated {
    include!("x86_64.rs");
}

#[cfg(target_pointer_width = "32")]
mod generated {
    include!("x86.rs");
}

pub use generated::*;
