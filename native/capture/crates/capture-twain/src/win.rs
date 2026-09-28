//! The Windows side of TWAIN: the DSM library, the window sources are owned
//! by, and the message loop they report through.

use core::cell::{Cell, RefCell};
use core::ffi::c_void;
use std::collections::VecDeque;
use std::sync::atomic::{AtomicUsize, Ordering};
use std::time::{Duration, Instant};

use windows::Win32::Foundation::{
    FreeLibrary, GlobalFree, HGLOBAL, HMODULE, HWND, LPARAM, LRESULT, WPARAM,
};
use windows::Win32::System::LibraryLoader::{
    GetModuleHandleW, GetProcAddress, LOAD_LIBRARY_SEARCH_DLL_LOAD_DIR,
    LOAD_LIBRARY_SEARCH_SYSTEM32, LoadLibraryExW,
};
use windows::Win32::System::Memory::{GHND, GlobalAlloc, GlobalLock, GlobalUnlock};
use windows::Win32::System::SystemInformation::{GetSystemDirectoryW, GetWindowsDirectoryW};
use windows::Win32::UI::WindowsAndMessaging::{
    CreateWindowExW, DefWindowProcW, DestroyWindow, DispatchMessageW, MSG,
    MsgWaitForMultipleObjects, PM_REMOVE, PeekMessageW, PostMessageW, QS_ALLINPUT, RegisterClassW,
    TranslateMessage, WINDOW_EX_STYLE, WM_APP, WM_QUIT, WNDCLASSW, WS_POPUP,
};
use windows::core::{PCSTR, PCWSTR, w};

use crate::consts::{MSG_NULL, TWRC_SUCCESS};
use crate::dsm::Dsm;
use crate::pump::{EventPump, Processed, PumpEvent};
use crate::sys::{
    DSM_MEMALLOCATE, DSM_MEMFREE, DSM_MEMLOCK, DSM_MEMUNLOCK, TW_ENTRYPOINT, TW_HANDLE,
    TW_IDENTITY, TW_MEMREF, TW_UINT16, TW_UINT32, pTW_IDENTITY,
};
use crate::{TwainError, TwainResult};

type EntryFn = unsafe extern "system" fn(
    pTW_IDENTITY,
    pTW_IDENTITY,
    TW_UINT32,
    TW_UINT16,
    TW_UINT16,
    TW_MEMREF,
) -> TW_UINT16;

#[derive(Clone, Copy)]
struct MemoryFns {
    alloc: DSM_MEMALLOCATE,
    free: DSM_MEMFREE,
    lock: DSM_MEMLOCK,
    unlock: DSM_MEMUNLOCK,
}

/// `twaindsm.dll`, loaded by full path from the system directory, so no
/// directory on the search path can stand in for it.
pub struct LibraryDsm {
    module: HMODULE,
    entry: EntryFn,
    memory: Cell<Option<MemoryFns>>,
}

impl core::fmt::Debug for LibraryDsm {
    fn fmt(&self, f: &mut core::fmt::Formatter<'_>) -> core::fmt::Result {
        f.debug_struct("LibraryDsm").finish_non_exhaustive()
    }
}

fn directory(get: impl Fn(&mut [u16]) -> u32) -> Option<String> {
    let mut buffer = vec![0u16; 520];
    let len = get(&mut buffer) as usize;
    (len > 0 && len < buffer.len()).then(|| String::from_utf16_lossy(&buffer[..len]))
}

impl LibraryDsm {
    /// Loads the DSM this process's bitness uses. A 32-bit process on 64-bit
    /// Windows is redirected to `SysWOW64` by the system, which is where the
    /// 32-bit DSM lives. A 32-bit process also falls back to the TWAIN 1.x
    /// manager, `twain_32.dll`, which older drivers registered with.
    pub fn load() -> TwainResult<Self> {
        // SAFETY: the buffer is valid for its length.
        let system = directory(|buf| unsafe { GetSystemDirectoryW(Some(buf)) });
        let mut candidates = Vec::new();
        if let Some(system) = &system {
            candidates.push(format!("{system}\\TWAINDSM.dll"));
        }
        if cfg!(target_pointer_width = "32") {
            // SAFETY: the buffer is valid for its length.
            if let Some(windows) = directory(|buf| unsafe { GetWindowsDirectoryW(Some(buf)) }) {
                candidates.push(format!("{windows}\\twain_32.dll"));
            }
        }

        for path in &candidates {
            let wide: Vec<u16> = path.encode_utf16().chain(Some(0)).collect();
            // SAFETY: a NUL-terminated path; the flags confine the DLL's own
            // dependencies to its directory and System32.
            let Ok(module) = (unsafe {
                LoadLibraryExW(
                    PCWSTR(wide.as_ptr()),
                    None,
                    LOAD_LIBRARY_SEARCH_DLL_LOAD_DIR | LOAD_LIBRARY_SEARCH_SYSTEM32,
                )
            }) else {
                continue;
            };
            // SAFETY: the module is loaded; the name is NUL-terminated.
            let Some(proc_) =
                (unsafe { GetProcAddress(module, PCSTR(c"DSM_Entry".as_ptr().cast())) })
            else {
                // SAFETY: loaded just above and not otherwise used.
                unsafe {
                    let _ = FreeLibrary(module);
                }
                continue;
            };
            // SAFETY: `DSM_Entry` has exactly this signature in every DSM, and
            // `extern "system"` is its calling convention on both targets.
            let entry = unsafe {
                core::mem::transmute::<unsafe extern "system" fn() -> isize, EntryFn>(proc_)
            };
            return Ok(Self {
                module,
                entry,
                memory: Cell::new(None),
            });
        }
        Err(TwainError::Load(format!(
            "none of {} could be loaded",
            candidates.join(", ")
        )))
    }
}

impl Drop for LibraryDsm {
    fn drop(&mut self) {
        // SAFETY: loaded by `load` and freed once, after the manager (which
        // borrows this) has closed.
        unsafe {
            let _ = FreeLibrary(self.module);
        }
    }
}

impl Dsm for LibraryDsm {
    unsafe fn entry(
        &self,
        origin: *mut TW_IDENTITY,
        dest: *mut TW_IDENTITY,
        dg: u32,
        dat: u16,
        msg: u16,
        data: *mut c_void,
    ) -> u16 {
        // SAFETY: forwarded as the caller vouched for them.
        unsafe { (self.entry)(origin, dest, dg, dat, msg, data) }
    }

    fn use_entry_points(&self, entry_points: &TW_ENTRYPOINT) {
        let fns = MemoryFns {
            alloc: entry_points.DSM_MemAllocate,
            free: entry_points.DSM_MemFree,
            lock: entry_points.DSM_MemLock,
            unlock: entry_points.DSM_MemUnlock,
        };
        if fns.alloc.is_some() && fns.free.is_some() && fns.lock.is_some() && fns.unlock.is_some() {
            self.memory.set(Some(fns));
        }
    }

    fn alloc(&self, size: u32) -> TW_HANDLE {
        if let Some(MemoryFns {
            alloc: Some(alloc), ..
        }) = self.memory.get()
        {
            // SAFETY: the DSM's own allocator.
            return unsafe { alloc(size) };
        }
        // SAFETY: a movable, zeroed global block, as TWAIN 1.x expects.
        unsafe { GlobalAlloc(GHND, size as usize) }.map_or(core::ptr::null_mut(), |h| h.0)
    }

    unsafe fn free(&self, handle: TW_HANDLE) {
        if let Some(MemoryFns {
            free: Some(free), ..
        }) = self.memory.get()
        {
            // SAFETY: per the caller.
            return unsafe { free(handle) };
        }
        // SAFETY: per the caller, a global block.
        unsafe {
            let _ = GlobalFree(Some(HGLOBAL(handle)));
        }
    }

    unsafe fn lock(&self, handle: TW_HANDLE) -> *mut c_void {
        if let Some(MemoryFns {
            lock: Some(lock), ..
        }) = self.memory.get()
        {
            // SAFETY: per the caller.
            return unsafe { lock(handle) };
        }
        // SAFETY: per the caller, a global block.
        unsafe { GlobalLock(HGLOBAL(handle)) }
    }

    unsafe fn unlock(&self, handle: TW_HANDLE) {
        if let Some(MemoryFns {
            unlock: Some(unlock),
            ..
        }) = self.memory.get()
        {
            // SAFETY: per the caller.
            return unsafe { unlock(handle) };
        }
        // SAFETY: per the caller, a locked global block.
        unsafe {
            let _ = GlobalUnlock(HGLOBAL(handle));
        }
    }
}

const WM_TWAIN_CALLBACK: u32 = WM_APP + 1;
const WM_CAPTURE_CANCEL: u32 = WM_APP + 2;
const CLASS_NAME: PCWSTR = w!("TrenovaCaptureTwain");

/// The window a TWAIN 2 callback posts to. Posting to a window rather than
/// the thread means a driver's own modal dialog loop, which dispatches but
/// never peeks for thread messages, still delivers it.
static CALLBACK_WINDOW: AtomicUsize = AtomicUsize::new(0);

thread_local! {
    static EVENTS: RefCell<VecDeque<PumpEvent>> = const { RefCell::new(VecDeque::new()) };
}

extern "system" fn window_proc(
    hwnd: HWND,
    message: u32,
    wparam: WPARAM,
    lparam: LPARAM,
) -> LRESULT {
    match message {
        WM_TWAIN_CALLBACK => {
            let msg = u16::try_from(wparam.0).unwrap_or(MSG_NULL);
            EVENTS.with(|events| events.borrow_mut().push_back(PumpEvent::Twain(msg)));
            LRESULT(0)
        }
        WM_CAPTURE_CANCEL => {
            EVENTS.with(|events| events.borrow_mut().push_back(PumpEvent::Cancel));
            LRESULT(0)
        }
        // SAFETY: the default handling for everything else.
        _ => unsafe { DefWindowProcW(hwnd, message, wparam, lparam) },
    }
}

/// The TWAIN 2 callback. A source may call it from any thread, so it only
/// posts, and the pump picks the message up on the scan thread.
unsafe extern "system" fn twain_callback(
    _origin: pTW_IDENTITY,
    _dest: pTW_IDENTITY,
    _dg: TW_UINT32,
    _dat: TW_UINT16,
    msg: TW_UINT16,
    _data: TW_MEMREF,
) -> TW_UINT16 {
    let window = CALLBACK_WINDOW.load(Ordering::Acquire);
    if window != 0 {
        // SAFETY: posting to a window this process created; a failure only
        // means the window is gone and the scan is ending anyway.
        unsafe {
            let _ = PostMessageW(
                Some(HWND(core::ptr::with_exposed_provenance_mut(window))),
                WM_TWAIN_CALLBACK,
                WPARAM(usize::from(msg)),
                LPARAM(0),
            );
        }
    }
    TWRC_SUCCESS
}

/// Asks a running pump to stop, from any thread.
#[derive(Clone, Copy, Debug)]
pub struct Canceller {
    window: usize,
}

impl Canceller {
    pub fn cancel(self) {
        // SAFETY: posting to the pump's window; harmless if it is gone.
        unsafe {
            let _ = PostMessageW(
                Some(HWND(core::ptr::with_exposed_provenance_mut(self.window))),
                WM_CAPTURE_CANCEL,
                WPARAM(0),
                LPARAM(0),
            );
        }
    }
}

/// A hidden window and the message loop on the thread that created it.
#[derive(Debug)]
pub struct WindowPump {
    hwnd: HWND,
}

impl WindowPump {
    /// Creates the window. It must be created, used and dropped on the one
    /// thread that runs the scan.
    pub fn new() -> windows::core::Result<Self> {
        // SAFETY: registering a class with a static name and procedure;
        // registering it twice fails harmlessly.
        let instance = unsafe { GetModuleHandleW(None)? };
        let class = WNDCLASSW {
            lpfnWndProc: Some(window_proc),
            hInstance: instance.into(),
            lpszClassName: CLASS_NAME,
            ..WNDCLASSW::default()
        };
        // SAFETY: as above.
        unsafe { RegisterClassW(&raw const class) };
        // SAFETY: a hidden, zero-sized popup of that class.
        let hwnd = unsafe {
            CreateWindowExW(
                WINDOW_EX_STYLE(0),
                CLASS_NAME,
                w!("Trenova Capture"),
                WS_POPUP,
                0,
                0,
                0,
                0,
                None,
                None,
                Some(instance.into()),
                None,
            )?
        };
        CALLBACK_WINDOW.store(hwnd.0.expose_provenance(), Ordering::Release);
        Ok(Self { hwnd })
    }

    /// The window as the DSM takes it.
    pub fn parent(&self) -> *mut c_void {
        self.hwnd.0
    }

    pub fn canceller(&self) -> Canceller {
        Canceller {
            window: self.hwnd.0.expose_provenance(),
        }
    }
}

impl Drop for WindowPump {
    fn drop(&mut self) {
        CALLBACK_WINDOW.store(0, Ordering::Release);
        // SAFETY: created by `new` on this thread.
        unsafe {
            let _ = DestroyWindow(self.hwnd);
        }
    }
}

impl EventPump for WindowPump {
    fn callback(&self) -> Option<*mut c_void> {
        Some(twain_callback as *mut c_void)
    }

    fn wait(
        &mut self,
        timeout: Option<Duration>,
        process: &mut dyn FnMut(*mut c_void) -> Processed,
    ) -> PumpEvent {
        let deadline = timeout.map(|t| Instant::now() + t);
        loop {
            if let Some(event) = EVENTS.with(|events| events.borrow_mut().pop_front()) {
                return event;
            }

            let mut msg = MSG::default();
            // SAFETY: a message buffer on the stack, for this thread's queue.
            while unsafe { PeekMessageW(&raw mut msg, None, 0, 0, PM_REMOVE) }.as_bool() {
                if msg.message == WM_QUIT {
                    return PumpEvent::Cancel;
                }
                match process((&raw mut msg).cast()) {
                    Processed::DsEvent(twain) if twain != MSG_NULL => {
                        return PumpEvent::Twain(twain);
                    }
                    Processed::DsEvent(_) => {}
                    Processed::NotDsEvent => {
                        // SAFETY: the message just peeked.
                        unsafe {
                            let _ = TranslateMessage(&raw const msg);
                            DispatchMessageW(&raw const msg);
                        }
                    }
                }
                if let Some(event) = EVENTS.with(|events| events.borrow_mut().pop_front()) {
                    return event;
                }
            }

            let wait_ms = match deadline {
                Some(deadline) => {
                    let left = deadline.saturating_duration_since(Instant::now());
                    if left.is_zero() {
                        return PumpEvent::TimedOut;
                    }
                    u32::try_from(left.as_millis()).unwrap_or(u32::MAX)
                }
                None => u32::MAX,
            };
            // SAFETY: waits on this thread's queue only.
            unsafe { MsgWaitForMultipleObjects(None, false, wait_ms, QS_ALLINPUT) };
        }
    }
}
