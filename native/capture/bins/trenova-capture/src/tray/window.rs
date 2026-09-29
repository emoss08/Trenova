//! The Trenova Capture window: a Win32 frame holding the local page in
//! `WebView2`.
//!
//! It lives on the tray's thread. It is created when it is first shown and
//! destroyed when closed, so a computer that never opens it never runs a
//! browser engine for it. What the page asks for is queued and handled from
//! a posted message, never inside a `WebView2` callback, because building a
//! `WebView2` runs a nested message loop and the window's state must not be
//! borrowed across one.

use std::cell::{Cell, RefCell};
use std::collections::VecDeque;
use std::num::NonZeroIsize;
use std::path::PathBuf;
use std::rc::Rc;
use std::sync::Arc;

use capture_platform::{paths, shell};
use tokio::sync::mpsc::UnboundedSender;
use trenova_capture::page;
use trenova_capture::state::{Attention, Command, Notice, Severity, Shared};
use trenova_capture::view::{PageMessage, WindowAction, render_script};
use windows::Win32::Foundation::{HWND, LPARAM, LRESULT, RECT, WPARAM};
use windows::Win32::System::LibraryLoader::GetModuleHandleW;
use windows::Win32::UI::HiDpi::GetDpiForSystem;
use windows::Win32::UI::WindowsAndMessaging::{
    CreateWindowExW, DefWindowProcW, DestroyWindow, FLASHW_TIMERNOFG, FLASHW_TRAY, FLASHWINFO,
    FlashWindowEx, HICON, ICON_BIG, ICON_SMALL, IsIconic, IsWindowVisible, KillTimer, MINMAXINFO,
    PostMessageW, RegisterClassW, SPI_GETWORKAREA, SW_RESTORE, SW_SHOW, SW_SHOWNOACTIVATE,
    SWP_NOACTIVATE, SWP_NOZORDER, SYSTEM_PARAMETERS_INFO_UPDATE_FLAGS, SendMessageW,
    SetForegroundWindow, SetTimer, SetWindowPos, ShowWindow, SystemParametersInfoW, WA_INACTIVE,
    WINDOW_EX_STYLE, WM_ACTIVATE, WM_CLOSE, WM_DPICHANGED, WM_GETMINMAXINFO, WM_SETICON, WM_TIMER,
    WNDCLASSW, WS_OVERLAPPEDWINDOW,
};
use windows_core::{PCWSTR, w};
use wry::raw_window_handle::{
    HandleError, HasWindowHandle, RawWindowHandle, Win32WindowHandle, WindowHandle,
};
use wry::{NewWindowResponse, WebContext, WebView, WebViewBuilder};

const CLASS_NAME: PCWSTR = w!("TrenovaCaptureWindow");
/// The window's size, in 96-dpi pixels.
const WIDTH: i32 = 440;
const HEIGHT: i32 = 720;
const MIN_WIDTH: i32 = 380;
const MIN_HEIGHT: i32 = 460;
/// How far from the screen's edge, near the notification area.
const MARGIN: i32 = 16;
/// How long a window that opened for a scan stays after it, untouched.
const AUTO_HIDE_MS: u32 = 8000;
const AUTO_HIDE_TIMER: usize = 1;

/// What the window needs from the tray.
#[derive(Clone, Debug)]
pub struct WindowContext {
    pub shared: Arc<Shared>,
    pub commands: UnboundedSender<Command>,
    /// The tray's window, told to handle the page's messages.
    pub tray: HWND,
    /// Posted to the tray when a page message is queued.
    pub page_message: u32,
    pub icon_small: HICON,
    pub icon_big: HICON,
}

struct CaptureWindow {
    hwnd: HWND,
    /// Dropped before the context it was built with.
    webview: WebView,
    _context: WebContext,
    /// Opened by itself for a scan, and not yet used, so it may go again.
    auto_hide: bool,
}

thread_local! {
    static WINDOW: RefCell<Option<CaptureWindow>> = const { RefCell::new(None) };
    static INBOX: Rc<RefCell<VecDeque<PageMessage>>> = Rc::new(RefCell::new(VecDeque::new()));
    static REGISTERED: Cell<bool> = const { Cell::new(false) };
}

/// How to bring the window forward.
#[derive(Clone, Copy, Debug, PartialEq, Eq)]
pub enum Show {
    /// The person asked for it: bring it to the front.
    Activate,
    /// Something is happening: show it without taking the keyboard.
    Quietly,
    /// Something needs the person: show it without taking the keyboard,
    /// and flash its taskbar button.
    Alert,
}

/// A window handle for `WebView2` to build in.
struct Host(HWND);

impl HasWindowHandle for Host {
    fn window_handle(&self) -> Result<WindowHandle<'_>, HandleError> {
        let hwnd = NonZeroIsize::new(self.0.0 as isize).ok_or(HandleError::Unavailable)?;
        let handle = RawWindowHandle::Win32(Win32WindowHandle::new(hwnd));
        // SAFETY: the window outlives the builder that borrows this handle.
        Ok(unsafe { WindowHandle::borrow_raw(handle) })
    }
}

fn scale(value: i32, dpi: u32) -> i32 {
    value.saturating_mul(i32::try_from(dpi).unwrap_or(96)) / 96
}

/// Near the notification area: the bottom-right of the work area.
fn placement(dpi: u32) -> (i32, i32, i32, i32) {
    let mut area = RECT::default();
    // SAFETY: fills a RECT this function owns.
    let found = unsafe {
        SystemParametersInfoW(
            SPI_GETWORKAREA,
            0,
            Some((&raw mut area).cast()),
            SYSTEM_PARAMETERS_INFO_UPDATE_FLAGS(0),
        )
    }
    .is_ok();
    let margin = scale(MARGIN, dpi);
    let width = scale(WIDTH, dpi);
    if !found {
        return (margin, margin, width, scale(HEIGHT, dpi));
    }
    let height = scale(HEIGHT, dpi).min(area.bottom - area.top - 2 * margin);
    let x = (area.right - width - margin).max(area.left);
    let y = (area.bottom - height - margin).max(area.top);
    (x, y, width, height)
}

fn register_class() -> windows_core::Result<()> {
    if REGISTERED.with(Cell::get) {
        return Ok(());
    }
    // SAFETY: registering this application's window class once.
    let instance = unsafe { GetModuleHandleW(None)? };
    let class = WNDCLASSW {
        lpfnWndProc: Some(window_proc),
        hInstance: instance.into(),
        lpszClassName: CLASS_NAME,
        ..WNDCLASSW::default()
    };
    // SAFETY: as above.
    unsafe { RegisterClassW(&raw const class) };
    REGISTERED.with(|r| r.set(true));
    Ok(())
}

/// The folder `WebView2` keeps its profile in: this person's own, beside the
/// spool, never the install directory, which they cannot write.
fn webview_data_dir() -> Option<PathBuf> {
    paths::data_dir().ok().map(|dir| dir.join("WebView2"))
}

fn create(context: &WindowContext) -> Result<CaptureWindow, String> {
    register_class().map_err(|err| err.to_string())?;
    // SAFETY: reads the system DPI.
    let dpi = unsafe { GetDpiForSystem() };
    let (x, y, width, height) = placement(dpi);
    // SAFETY: as above.
    let instance = unsafe { GetModuleHandleW(None) }.map_err(|err| err.to_string())?;
    // SAFETY: a top-level window of the class registered above, shown
    // once the page is in it.
    let hwnd = unsafe {
        CreateWindowExW(
            WINDOW_EX_STYLE(0),
            CLASS_NAME,
            w!("Trenova Capture"),
            WS_OVERLAPPEDWINDOW,
            x,
            y,
            width,
            height,
            None,
            None,
            Some(instance.into()),
            None,
        )
    }
    .map_err(|err| err.to_string())?;
    // SAFETY: sets this window's icons; the tray owns and outlives them.
    unsafe {
        SendMessageW(
            hwnd,
            WM_SETICON,
            Some(WPARAM(ICON_SMALL as usize)),
            Some(LPARAM(context.icon_small.0 as isize)),
        );
        SendMessageW(
            hwnd,
            WM_SETICON,
            Some(WPARAM(ICON_BIG as usize)),
            Some(LPARAM(context.icon_big.0 as isize)),
        );
    }

    let mut web_context = WebContext::new(webview_data_dir());
    let inbox = INBOX.with(Rc::clone);
    let tray = context.tray.0.expose_provenance();
    let page_message = context.page_message;
    let first_load = Cell::new(true);
    let built = WebViewBuilder::new_with_web_context(&mut web_context)
        .with_html(page::html())
        .with_background_color((244, 245, 248, 255))
        .with_devtools(false)
        .with_navigation_handler(move |_| first_load.replace(false))
        .with_new_window_req_handler(|_, _| NewWindowResponse::Deny)
        .with_download_started_handler(|_, _| false)
        .with_drag_drop_handler(|_| true)
        .with_ipc_handler(move |request| {
            let Some(message) = PageMessage::parse(request.body()) else {
                tracing::warn!("the window's page sent a message that is not understood");
                return;
            };
            inbox.borrow_mut().push_back(message);
            // SAFETY: posting to the tray's own window, on its thread.
            unsafe {
                let _ = PostMessageW(
                    Some(HWND(std::ptr::with_exposed_provenance_mut(tray))),
                    page_message,
                    WPARAM(0),
                    LPARAM(0),
                );
            }
        })
        .build(&Host(hwnd));
    match built {
        Ok(webview) => Ok(CaptureWindow {
            hwnd,
            webview,
            _context: web_context,
            auto_hide: false,
        }),
        Err(err) => {
            // SAFETY: the window made above, never shown.
            unsafe {
                let _ = DestroyWindow(hwnd);
            }
            Err(err.to_string())
        }
    }
}

fn is_open() -> bool {
    WINDOW.with(|w| w.borrow().is_some())
}

fn hwnd() -> Option<HWND> {
    WINDOW.with(|w| w.borrow().as_ref().map(|window| window.hwnd))
}

/// Shows the window, creating it first if it is not open.
pub fn show(context: &WindowContext, how: Show) {
    if !is_open() {
        match create(context) {
            Ok(window) => WINDOW.with(|w| *w.borrow_mut() = Some(window)),
            Err(err) => {
                tracing::error!(error = %err, "the Trenova Capture window could not open");
                context.shared.notify(Notice {
                    title: "The Trenova Capture window could not open".into(),
                    body: format!(
                        "It needs the Microsoft Edge WebView2 Runtime, which Windows 10 and 11 include. {err}"
                    ),
                    severity: Severity::Error,
                    link: None,
                });
                return;
            }
        }
    }
    let Some(hwnd) = hwnd() else {
        return;
    };
    // SAFETY: this thread's own window.
    unsafe {
        let visible = IsWindowVisible(hwnd).as_bool();
        match how {
            Show::Activate => {
                WINDOW.with(|w| {
                    if let Some(window) = w.borrow_mut().as_mut() {
                        window.auto_hide = false;
                    }
                });
                let _ = KillTimer(Some(hwnd), AUTO_HIDE_TIMER);
                let _ = ShowWindow(
                    hwnd,
                    if IsIconic(hwnd).as_bool() {
                        SW_RESTORE
                    } else {
                        SW_SHOW
                    },
                );
                let _ = SetForegroundWindow(hwnd);
            }
            Show::Quietly | Show::Alert => {
                if !visible || IsIconic(hwnd).as_bool() {
                    let _ = ShowWindow(hwnd, SW_SHOWNOACTIVATE);
                }
                if how == Show::Quietly {
                    return refresh(&context.shared);
                }
                let flash = FLASHWINFO {
                    cbSize: u32::try_from(std::mem::size_of::<FLASHWINFO>()).unwrap_or(0),
                    hwnd,
                    dwFlags: FLASHW_TRAY | FLASHW_TIMERNOFG,
                    uCount: 3,
                    dwTimeout: 0,
                };
                let _ = FlashWindowEx(&raw const flash);
            }
        }
    }
    refresh(&context.shared);
}

/// Hands the page the current view, when the window is open.
pub fn refresh(shared: &Shared) {
    WINDOW.with(|w| {
        if let Some(window) = w.borrow().as_ref() {
            let script = render_script(&shared.snapshot().view(env!("CARGO_PKG_VERSION")));
            if let Err(err) = window.webview.evaluate_script(&script) {
                tracing::warn!(error = %err, "the window could not be redrawn");
            }
        }
    });
}

/// Closes the window, and the browser engine behind it.
pub fn close() {
    let window = WINDOW.with(|w| w.borrow_mut().take());
    if let Some(window) = window {
        let hwnd = window.hwnd;
        drop(window);
        // SAFETY: this thread's own window, now without its webview.
        unsafe {
            let _ = DestroyWindow(hwnd);
        }
    }
}

/// Brings the window forward for something that happened.
pub fn attention(context: &WindowContext, attention: Attention) {
    match attention {
        Attention::SignIn | Attention::SetUp => show(context, Show::Activate),
        Attention::ScanPaused | Attention::Refused => {
            WINDOW.with(|w| {
                if let Some(window) = w.borrow_mut().as_mut() {
                    window.auto_hide = false;
                }
            });
            if let Some(hwnd) = hwnd() {
                // SAFETY: this thread's own window.
                unsafe {
                    let _ = KillTimer(Some(hwnd), AUTO_HIDE_TIMER);
                }
            }
            show(context, Show::Alert);
        }
        Attention::ScanStarted => {
            if is_open() {
                if let Some(hwnd) = hwnd() {
                    // SAFETY: this thread's own window.
                    unsafe {
                        let _ = KillTimer(Some(hwnd), AUTO_HIDE_TIMER);
                    }
                }
                return;
            }
            show(context, Show::Quietly);
            WINDOW.with(|w| {
                if let Some(window) = w.borrow_mut().as_mut() {
                    window.auto_hide = true;
                }
            });
        }
        Attention::ScanEnded => {
            let hide = WINDOW.with(|w| {
                w.borrow()
                    .as_ref()
                    .filter(|window| window.auto_hide)
                    .map(|window| window.hwnd)
            });
            if let Some(hwnd) = hide {
                // SAFETY: a timer on this thread's own window.
                unsafe {
                    SetTimer(Some(hwnd), AUTO_HIDE_TIMER, AUTO_HIDE_MS, None);
                }
            }
        }
    }
}

/// Handles what the page asked for since the last time.
pub fn drain_inbox(context: &WindowContext) {
    loop {
        let Some(message) = INBOX.with(|inbox| inbox.borrow_mut().pop_front()) else {
            return;
        };
        let touched = !matches!(message, PageMessage::Ready {});
        if touched {
            WINDOW.with(|w| {
                if let Some(window) = w.borrow_mut().as_mut() {
                    window.auto_hide = false;
                }
            });
        }
        let snapshot = context.shared.snapshot();
        let Some(action) = snapshot.resolve(message) else {
            tracing::info!("the window asked for something it does not show");
            continue;
        };
        perform(context, action);
    }
}

fn send(context: &WindowContext, command: Command) {
    if context.commands.send(command).is_err() {
        tracing::error!("the agent is not running");
    }
}

fn perform(context: &WindowContext, action: WindowAction) {
    match action {
        WindowAction::Redraw => refresh(&context.shared),
        WindowAction::Close => close(),
        WindowAction::Command(command) => send(context, command),
        WindowAction::Open(url) => {
            if let Err(err) = shell::open_url(&url) {
                tracing::warn!(error = %err, "could not open the browser");
            }
        }
        WindowAction::Save(key) => match paths::downloads_dir() {
            Ok(into) => send(context, Command::Save { key, into }),
            Err(err) => context.shared.notify(Notice {
                title: "It could not be saved".into(),
                body: format!("Your Downloads folder could not be found: {err}"),
                severity: Severity::Error,
                link: None,
            }),
        },
        WindowAction::AddPrinter => super::add_printer_in_background(&context.commands),
        WindowAction::OpenLogs => match paths::data_dir() {
            Ok(dir) => {
                if let Err(err) = shell::open_folder(&dir.join("logs")) {
                    tracing::warn!(error = %err, "could not open the log folder");
                }
            }
            Err(err) => tracing::warn!(error = %err, "no log folder"),
        },
        WindowAction::Quit => send(context, Command::Quit),
    }
}

extern "system" fn window_proc(
    hwnd: HWND,
    message: u32,
    wparam: WPARAM,
    lparam: LPARAM,
) -> LRESULT {
    match message {
        WM_CLOSE => {
            close();
            LRESULT(0)
        }
        WM_ACTIVATE => {
            if u32::try_from(wparam.0 & 0xFFFF).unwrap_or(0) != WA_INACTIVE {
                WINDOW.with(|w| {
                    if let Ok(mut window) = w.try_borrow_mut()
                        && let Some(window) = window.as_mut()
                    {
                        window.auto_hide = false;
                    }
                });
                // SAFETY: this window's own timer.
                unsafe {
                    let _ = KillTimer(Some(hwnd), AUTO_HIDE_TIMER);
                }
            }
            // SAFETY: default handling, which gives the page the focus.
            unsafe { DefWindowProcW(hwnd, message, wparam, lparam) }
        }
        WM_TIMER if wparam.0 == AUTO_HIDE_TIMER => {
            // SAFETY: this window's own timer.
            unsafe {
                let _ = KillTimer(Some(hwnd), AUTO_HIDE_TIMER);
            }
            let still_auto = WINDOW.with(|w| {
                w.try_borrow()
                    .ok()
                    .and_then(|window| window.as_ref().map(|window| window.auto_hide))
                    .unwrap_or(false)
            });
            if still_auto {
                close();
            }
            LRESULT(0)
        }
        WM_GETMINMAXINFO => {
            // SAFETY: reads the system DPI.
            let dpi = unsafe { GetDpiForSystem() };
            // SAFETY: Windows passes a MINMAXINFO to fill in.
            if let Some(info) = unsafe { (lparam.0 as *mut MINMAXINFO).as_mut() } {
                info.ptMinTrackSize.x = scale(MIN_WIDTH, dpi);
                info.ptMinTrackSize.y = scale(MIN_HEIGHT, dpi);
            }
            LRESULT(0)
        }
        WM_DPICHANGED => {
            // SAFETY: Windows passes the rectangle it suggests for the new DPI.
            if let Some(rect) = unsafe { (lparam.0 as *const RECT).as_ref() } {
                // SAFETY: this thread's own window.
                unsafe {
                    let _ = SetWindowPos(
                        hwnd,
                        None,
                        rect.left,
                        rect.top,
                        rect.right - rect.left,
                        rect.bottom - rect.top,
                        SWP_NOZORDER | SWP_NOACTIVATE,
                    );
                }
            }
            LRESULT(0)
        }
        // SAFETY: default handling.
        _ => unsafe { DefWindowProcW(hwnd, message, wparam, lparam) },
    }
}
