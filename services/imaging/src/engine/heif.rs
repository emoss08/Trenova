//! HEIC and HEIF through libheif, loaded at child start.
//!
//! libheif is LGPL-3.0, so it is linked dynamically: the worker opens the
//! shared library the image installs (and its HEVC decoder plugin), and a
//! deployment can replace it. Only the few functions decoding the primary
//! image needs are bound. libheif applies the image's rotation and mirror
//! transforms while decoding, so what comes out is upright.

use std::ffi::{CStr, c_char, c_int, c_void};

use libloading::Library;

use super::error::{EngineError, EngineResult};
use super::raster::Rgba;

const COLORSPACE_RGB: c_int = 1;
const CHROMA_INTERLEAVED_RGBA: c_int = 11;
const CHANNEL_INTERLEAVED: c_int = 10;

#[repr(C)]
#[derive(Clone, Copy)]
struct HeifError {
    code: c_int,
    subcode: c_int,
    message: *const c_char,
}

type ContextAlloc = unsafe extern "C" fn() -> *mut c_void;
type ContextFree = unsafe extern "C" fn(*mut c_void);
type ReadFromMemory =
    unsafe extern "C" fn(*mut c_void, *const c_void, usize, *const c_void) -> HeifError;
type PrimaryHandle = unsafe extern "C" fn(*mut c_void, *mut *mut c_void) -> HeifError;
type HandleDimension = unsafe extern "C" fn(*const c_void) -> c_int;
type HandleRelease = unsafe extern "C" fn(*const c_void);
type DecodeImage =
    unsafe extern "C" fn(*const c_void, *mut *mut c_void, c_int, c_int, *const c_void) -> HeifError;
type ImageDimension = unsafe extern "C" fn(*const c_void) -> c_int;
type PlaneReadonly = unsafe extern "C" fn(*const c_void, c_int, *mut c_int) -> *const u8;
type ImageRelease = unsafe extern "C" fn(*const c_void);
type Init = unsafe extern "C" fn(*mut c_void) -> HeifError;
type Version = unsafe extern "C" fn() -> *const c_char;

/// The bound library.
pub struct Heif {
    _library: Library,
    context_alloc: ContextAlloc,
    context_free: ContextFree,
    read_from_memory: ReadFromMemory,
    primary_handle: PrimaryHandle,
    handle_width: HandleDimension,
    handle_height: HandleDimension,
    handle_release: HandleRelease,
    decode_image: DecodeImage,
    plane_readonly: PlaneReadonly,
    image_width: ImageDimension,
    image_height: ImageDimension,
    image_release: ImageRelease,
    version: String,
}

impl std::fmt::Debug for Heif {
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        f.debug_struct("Heif")
            .field("version", &self.version)
            .finish_non_exhaustive()
    }
}

macro_rules! symbol {
    ($library:expr, $name:literal) => {
        // SAFETY: the symbol is declared with the signature libheif's public
        // header gives it; it is copied out of the library, which `Heif` keeps
        // loaded for as long as the pointer is used.
        *unsafe { $library.get(concat!($name, "\0").as_bytes()) }
            .map_err(|err| EngineError::internal(format!("libheif lacks {}: {err}", $name)))?
    };
}

impl Heif {
    /// Loads libheif from `path` and initialises it (which loads its decoder
    /// plugins, so this must run before the child hardens).
    pub fn load(path: &str) -> EngineResult<Self> {
        // SAFETY: loading a shared library runs its initialisers; libheif's
        // are its own, and this runs before any untrusted input is read.
        let library = unsafe { Library::new(path) }
            .map_err(|err| EngineError::internal(format!("load libheif from {path}: {err}")))?;
        let init: Init = symbol!(library, "heif_init");
        let version: Version = symbol!(library, "heif_get_version");
        // SAFETY: `heif_init(NULL)` initialises the library with default
        // parameters; a null argument is documented as allowed.
        let initialised = unsafe { init(std::ptr::null_mut()) };
        if initialised.code != 0 {
            return Err(EngineError::internal(format!(
                "libheif did not initialise: {}",
                message(initialised)
            )));
        }
        // SAFETY: `heif_get_version` returns a static NUL-terminated string.
        let version = unsafe { CStr::from_ptr(version()) }
            .to_string_lossy()
            .into_owned();
        Ok(Self {
            context_alloc: symbol!(library, "heif_context_alloc"),
            context_free: symbol!(library, "heif_context_free"),
            read_from_memory: symbol!(library, "heif_context_read_from_memory_without_copy"),
            primary_handle: symbol!(library, "heif_context_get_primary_image_handle"),
            handle_width: symbol!(library, "heif_image_handle_get_width"),
            handle_height: symbol!(library, "heif_image_handle_get_height"),
            handle_release: symbol!(library, "heif_image_handle_release"),
            decode_image: symbol!(library, "heif_decode_image"),
            plane_readonly: symbol!(library, "heif_image_get_plane_readonly"),
            image_width: symbol!(library, "heif_image_get_primary_width"),
            image_height: symbol!(library, "heif_image_get_primary_height"),
            image_release: symbol!(library, "heif_image_release"),
            version,
            _library: library,
        })
    }

    pub fn version(&self) -> &str {
        &self.version
    }

    /// Decodes the primary image, refusing one larger than `max_pixels`.
    /// Returns the upright image and its stored size.
    pub fn decode(&self, bytes: &[u8], max_pixels: u64) -> EngineResult<(Rgba, u32, u32)> {
        let context = Context::new(self)?;
        // SAFETY: `context.ptr` is a live context; `bytes` outlives it (the
        // context is dropped at the end of this function) and libheif only
        // reads it.
        let read = unsafe {
            (self.read_from_memory)(
                context.ptr,
                bytes.as_ptr().cast(),
                bytes.len(),
                std::ptr::null(),
            )
        };
        if read.code != 0 {
            return Err(EngineError::invalid(format!(
                "the HEIF image could not be read: {}",
                message(read)
            )));
        }
        let mut handle = std::ptr::null_mut();
        // SAFETY: writes a handle owned by the caller into `handle`.
        let got = unsafe { (self.primary_handle)(context.ptr, &raw mut handle) };
        if got.code != 0 || handle.is_null() {
            return Err(EngineError::invalid(format!(
                "the HEIF image has no primary image: {}",
                message(got)
            )));
        }
        let handle = Handle {
            heif: self,
            ptr: handle,
        };
        // SAFETY: `handle.ptr` is a live image handle.
        let (width, height) = unsafe {
            (
                (self.handle_width)(handle.ptr),
                (self.handle_height)(handle.ptr),
            )
        };
        let width = u32::try_from(width)
            .map_err(|_| EngineError::invalid("the HEIF image has no width"))?;
        let height = u32::try_from(height)
            .map_err(|_| EngineError::invalid("the HEIF image has no height"))?;
        if width == 0 || height == 0 {
            return Err(EngineError::invalid("the HEIF image has no pixels"));
        }
        if max_pixels > 0 && u64::from(width) * u64::from(height) > max_pixels {
            return Err(EngineError::pixels(format!(
                "a {width} × {height} image is over the {max_pixels}-pixel budget"
            )));
        }

        let mut image = std::ptr::null_mut();
        // SAFETY: decodes into a new image owned by the caller.
        let decoded = unsafe {
            (self.decode_image)(
                handle.ptr,
                &raw mut image,
                COLORSPACE_RGB,
                CHROMA_INTERLEAVED_RGBA,
                std::ptr::null(),
            )
        };
        if decoded.code != 0 || image.is_null() {
            return Err(EngineError::invalid(format!(
                "the HEIF image could not be decoded: {}",
                message(decoded)
            )));
        }
        let image = Image {
            heif: self,
            ptr: image,
        };
        // The transforms can turn the image, so its decoded size is read from
        // the image rather than assumed to be the handle's.
        // SAFETY: `image.ptr` is a live decoded image.
        let (out_w, out_h) = unsafe {
            (
                (self.image_width)(image.ptr),
                (self.image_height)(image.ptr),
            )
        };
        let out_w = u32::try_from(out_w)
            .map_err(|_| EngineError::invalid("the HEIF image decoded to no width"))?;
        let out_h = u32::try_from(out_h)
            .map_err(|_| EngineError::invalid("the HEIF image decoded to no height"))?;
        let mut stride: c_int = 0;
        // SAFETY: the interleaved plane of a live RGBA image; the pointer stays
        // valid until the image is released, after the copy below.
        let plane =
            unsafe { (self.plane_readonly)(image.ptr, CHANNEL_INTERLEAVED, &raw mut stride) };
        let stride = usize::try_from(stride).unwrap_or(0);
        let row = out_w as usize * 4;
        if plane.is_null() || stride < row {
            return Err(EngineError::invalid("the HEIF image decoded to no pixels"));
        }
        let mut data = Vec::with_capacity(row * out_h as usize);
        for y in 0..out_h as usize {
            // SAFETY: row `y` of the plane starts `y * stride` bytes in and holds
            // at least `row` bytes, as libheif reported through `stride`.
            let line = unsafe { std::slice::from_raw_parts(plane.add(y * stride), row) };
            data.extend_from_slice(line);
        }
        let mut rgba = Rgba::new(out_w, out_h, data)?;
        rgba.flatten_on_white();
        Ok((rgba, width, height))
    }
}

struct Context<'a> {
    heif: &'a Heif,
    ptr: *mut c_void,
}

impl<'a> Context<'a> {
    fn new(heif: &'a Heif) -> EngineResult<Self> {
        // SAFETY: allocates a fresh context, freed in `drop`.
        let ptr = unsafe { (heif.context_alloc)() };
        if ptr.is_null() {
            return Err(EngineError::internal(
                "libheif could not allocate a context",
            ));
        }
        Ok(Self { heif, ptr })
    }
}

impl Drop for Context<'_> {
    fn drop(&mut self) {
        // SAFETY: frees the context allocated in `new`, once.
        unsafe { (self.heif.context_free)(self.ptr) };
    }
}

struct Handle<'a> {
    heif: &'a Heif,
    ptr: *mut c_void,
}

impl Drop for Handle<'_> {
    fn drop(&mut self) {
        // SAFETY: releases the handle obtained from the context, once.
        unsafe { (self.heif.handle_release)(self.ptr) };
    }
}

struct Image<'a> {
    heif: &'a Heif,
    ptr: *mut c_void,
}

impl Drop for Image<'_> {
    fn drop(&mut self) {
        // SAFETY: releases the decoded image, once.
        unsafe { (self.heif.image_release)(self.ptr) };
    }
}

fn message(error: HeifError) -> String {
    if error.message.is_null() {
        return format!("error {}.{}", error.code, error.subcode);
    }
    // SAFETY: libheif's error messages are static NUL-terminated strings.
    let text = unsafe { CStr::from_ptr(error.message) }.to_string_lossy();
    text.chars().take(200).collect()
}
