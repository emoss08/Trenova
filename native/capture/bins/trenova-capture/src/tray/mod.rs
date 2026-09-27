//! The notification-area icon, its menu and its notifications.
//!
//! Everything shown is decided by `trenova_capture::menu` from a snapshot of
//! the agent's state; this module only draws it and hands choices back. The
//! agent's thread asks for a redraw or a notification by posting a message
//! to the tray's window, so all drawing happens on this thread.

mod prompt;

use std::cell::RefCell;
use std::collections::VecDeque;
use std::sync::{Arc, Mutex, PoisonError};

use capture_platform::shell;
use tokio::sync::mpsc::UnboundedSender;
use trenova_capture::icon::{LOGO, best_for};
use trenova_capture::menu::{MenuAction, MenuEntry};
use trenova_capture::state::{Command, Notice, Severity, Shared, Ui};
use windows::Win32::Foundation::{HWND, LPARAM, LRESULT, POINT, WPARAM};
use windows::Win32::System::LibraryLoader::GetModuleHandleW;
use windows::Win32::UI::Shell::{
    NIF_ICON, NIF_INFO, NIF_MESSAGE, NIF_SHOWTIP, NIF_TIP, NIIF_ERROR, NIIF_INFO,
    NIIF_RESPECT_QUIET_TIME, NIIF_WARNING, NIM_ADD, NIM_DELETE, NIM_MODIFY, NIM_SETVERSION,
    NIN_BALLOONUSERCLICK, NIN_SELECT, NINF_KEY, NOTIFYICON_VERSION_4, NOTIFYICONDATAW,
    Shell_NotifyIconW,
};
use windows::Win32::UI::WindowsAndMessaging::{
    AppendMenuW, CreateIconFromResourceEx, CreatePopupMenu, CreateWindowExW, DefWindowProcW,
    DestroyIcon, DestroyMenu, DestroyWindow, DispatchMessageW, GetCursorPos, GetMessageW,
    GetSystemMetrics, HICON, HMENU, IDI_APPLICATION, LR_DEFAULTCOLOR, LoadIconW, MF_GRAYED,
    MF_POPUP, MF_SEPARATOR, MF_STRING, MSG, PostMessageW, PostQuitMessage, RegisterClassW,
    RegisterWindowMessageW, SM_CXSMICON, SetForegroundWindow, TPM_BOTTOMALIGN, TPM_NONOTIFY,
    TPM_RETURNCMD, TPM_RIGHTBUTTON, TrackPopupMenuEx, TranslateMessage, WINDOW_EX_STYLE, WM_APP,
    WM_CLOSE, WM_CONTEXTMENU, WM_DESTROY, WM_NULL, WNDCLASSW, WS_OVERLAPPED,
};
use windows_core::{PCWSTR, w};

/// `NIN_KEYSELECT` from `shellapi.h`, which the bindings leave out: the icon
/// chosen from the keyboard.
const NIN_KEYSELECT: u32 = NIN_SELECT | NINF_KEY;
const WM_TRAY: u32 = WM_APP + 1;
const WM_REFRESH: u32 = WM_APP + 2;
const WM_NOTICE: u32 = WM_APP + 3;
const ICON_ID: u32 = 1;
const CLASS_NAME: PCWSTR = w!("TrenovaCaptureTray");

/// The tray, as the agent's thread reaches it.
#[derive(Debug)]
pub struct TrayUi {
    window: usize,
    notices: Arc<Mutex<VecDeque<Notice>>>,
}

impl TrayUi {
    fn post(&self, message: u32) {
        // SAFETY: posting to the tray's own window; harmless once it is gone.
        unsafe {
            let _ = PostMessageW(
                Some(HWND(std::ptr::with_exposed_provenance_mut(self.window))),
                message,
                WPARAM(0),
                LPARAM(0),
            );
        }
    }

    /// Closes the tray, which ends the application.
    pub fn quit(&self) {
        self.post(WM_CLOSE);
    }
}

impl Ui for TrayUi {
    fn refresh(&self) {
        self.post(WM_REFRESH);
    }

    fn notify(&self, notice: Notice) {
        self.notices
            .lock()
            .unwrap_or_else(PoisonError::into_inner)
            .push_back(notice);
        self.post(WM_NOTICE);
    }
}

struct TrayState {
    hwnd: HWND,
    shared: Option<Arc<Shared>>,
    commands: UnboundedSender<Command>,
    notices: Arc<Mutex<VecDeque<Notice>>>,
    icon: HICON,
    /// Opened when the last notification is clicked.
    notice_link: Option<String>,
    taskbar_created: u32,
}

thread_local! {
    static STATE: RefCell<Option<TrayState>> = const { RefCell::new(None) };
}

/// The tray's window, on the thread that runs its message loop.
#[derive(Debug)]
pub struct Tray {
    hwnd: HWND,
}

/// A fixed UTF-16 field holding `text`, truncated to leave its NUL. Fields
/// are passed by value because the 32-bit `NOTIFYICONDATAW` is packed and its
/// fields cannot be borrowed in place.
fn filled<const N: usize>(mut field: [u16; N], text: &str) -> [u16; N] {
    let mut units: Vec<u16> = text.encode_utf16().take(N - 1).collect();
    units.push(0);
    field[..units.len()].copy_from_slice(&units);
    field[units.len()..].fill(0);
    field
}

fn icon_data(hwnd: HWND) -> NOTIFYICONDATAW {
    NOTIFYICONDATAW {
        cbSize: u32::try_from(std::mem::size_of::<NOTIFYICONDATAW>()).unwrap_or(0),
        hWnd: hwnd,
        uID: ICON_ID,
        ..NOTIFYICONDATAW::default()
    }
}

/// The Trenova mark at the small-icon size for this display.
fn load_icon() -> HICON {
    // SAFETY: reads a system metric.
    let size = unsafe { GetSystemMetrics(SM_CXSMICON) };
    let size_u32 = u32::try_from(size).unwrap_or(16);
    if let Some(image) = best_for(LOGO, size_u32)
        // SAFETY: the bytes are one image from the icon file, PNG or DIB,
        // which is what this call takes.
        && let Ok(icon) = unsafe {
            CreateIconFromResourceEx(image.bytes, true, 0x0003_0000, size, size, LR_DEFAULTCOLOR)
        }
    {
        return icon;
    }
    // SAFETY: a stock icon.
    unsafe { LoadIconW(None, IDI_APPLICATION) }.unwrap_or_default()
}

fn add_icon(state: &TrayState) {
    let mut data = icon_data(state.hwnd);
    data.uFlags = NIF_ICON | NIF_MESSAGE | NIF_TIP | NIF_SHOWTIP;
    data.uCallbackMessage = WM_TRAY;
    data.hIcon = state.icon;
    data.szTip = filled(data.szTip, "Trenova Capture");
    // SAFETY: a filled structure for this window's icon.
    unsafe {
        if !Shell_NotifyIconW(NIM_ADD, &raw const data).as_bool() {
            tracing::warn!("the tray icon could not be added");
        }
        data.Anonymous.uVersion = NOTIFYICON_VERSION_4;
        let _ = Shell_NotifyIconW(NIM_SETVERSION, &raw const data);
    }
    refresh_tip(state);
}

fn refresh_tip(state: &TrayState) {
    let Some(shared) = &state.shared else {
        return;
    };
    let mut data = icon_data(state.hwnd);
    data.uFlags = NIF_TIP | NIF_SHOWTIP;
    data.szTip = filled(data.szTip, &shared.snapshot().tooltip());
    // SAFETY: a filled structure for this window's icon.
    unsafe {
        let _ = Shell_NotifyIconW(NIM_MODIFY, &raw const data);
    }
}

/// Shows the newest notification. Windows shows one at a time, and the
/// menu keeps the state that older ones reported.
fn show_notice(state: &mut TrayState) {
    let newest = {
        let mut queue = state.notices.lock().unwrap_or_else(PoisonError::into_inner);
        let newest = queue.pop_back();
        queue.clear();
        newest
    };
    let Some(notice) = newest else {
        return;
    };
    let mut data = icon_data(state.hwnd);
    data.uFlags = NIF_INFO;
    data.szInfoTitle = filled(data.szInfoTitle, &notice.title);
    data.szInfo = filled(data.szInfo, &notice.body);
    data.dwInfoFlags = NIIF_RESPECT_QUIET_TIME
        | match notice.severity {
            Severity::Info => NIIF_INFO,
            Severity::Warning => NIIF_WARNING,
            Severity::Error => NIIF_ERROR,
        };
    state.notice_link = notice.link;
    // SAFETY: a filled structure for this window's icon.
    unsafe {
        let _ = Shell_NotifyIconW(NIM_MODIFY, &raw const data);
    }
}

/// Menu text, with `&` doubled so Windows does not take it as a mnemonic.
fn menu_text(label: &str) -> Vec<u16> {
    label
        .replace('&', "&&")
        .encode_utf16()
        .chain(Some(0))
        .collect()
}

/// Builds a popup menu, collecting each item's action in id order.
fn build_menu(entries: &[MenuEntry], actions: &mut Vec<MenuAction>) -> windows_core::Result<HMENU> {
    // SAFETY: a new, empty popup menu; destroyed by the caller.
    let menu = unsafe { CreatePopupMenu()? };
    for entry in entries {
        match entry {
            MenuEntry::Separator => {
                // SAFETY: appending to a menu this function owns.
                unsafe { AppendMenuW(menu, MF_SEPARATOR, 0, PCWSTR::null())? };
            }
            MenuEntry::Item { label, action } => {
                let text = menu_text(label);
                let (flags, id) = match action {
                    Some(action) => {
                        actions.push(action.clone());
                        (MF_STRING, actions.len())
                    }
                    None => (MF_STRING | MF_GRAYED, 0),
                };
                // SAFETY: as above; the text outlives the call.
                unsafe { AppendMenuW(menu, flags, id, PCWSTR(text.as_ptr()))? };
            }
            MenuEntry::Submenu { label, entries } => {
                let submenu = build_menu(entries, actions)?;
                let text = menu_text(label);
                // SAFETY: the submenu becomes part of `menu` and is destroyed
                // with it.
                unsafe { AppendMenuW(menu, MF_POPUP, submenu.0 as usize, PCWSTR(text.as_ptr()))? };
            }
        }
    }
    Ok(menu)
}

/// Shows the menu at the pointer and returns the chosen action.
fn choose(hwnd: HWND, entries: &[MenuEntry]) -> Option<MenuAction> {
    let mut actions = Vec::new();
    let menu = match build_menu(entries, &mut actions) {
        Ok(menu) => menu,
        Err(err) => {
            tracing::error!(error = %err, "the menu could not be built");
            return None;
        }
    };
    let mut point = POINT::default();
    // SAFETY: the menu and window are this thread's; the foreground and
    // WM_NULL dance is what makes a tray menu close when clicked away from.
    let chosen = unsafe {
        let _ = GetCursorPos(&raw mut point);
        let _ = SetForegroundWindow(hwnd);
        let chosen = TrackPopupMenuEx(
            menu,
            (TPM_RETURNCMD | TPM_NONOTIFY | TPM_RIGHTBUTTON | TPM_BOTTOMALIGN).0,
            point.x,
            point.y,
            hwnd,
            None,
        );
        let _ = PostMessageW(Some(hwnd), WM_NULL, WPARAM(0), LPARAM(0));
        let _ = DestroyMenu(menu);
        chosen
    };
    usize::try_from(chosen.0)
        .ok()
        .filter(|&id| id > 0)
        .and_then(|id| actions.get(id - 1).cloned())
}

fn perform(
    hwnd: HWND,
    action: MenuAction,
    commands: &UnboundedSender<Command>,
    shared: Option<&Arc<Shared>>,
) {
    let send = |command: Command| {
        if commands.send(command).is_err() {
            tracing::error!("the agent is not running");
        }
    };
    match action {
        MenuAction::Command(command) => send(command),
        MenuAction::Open(url) => {
            if let Err(err) = shell::open_url(&url) {
                tracing::warn!(error = %err, "could not open the browser");
            }
        }
        MenuAction::OpenFolder(path) => {
            if let Err(err) = shell::open_folder(&path) {
                tracing::warn!(error = %err, "could not open the folder");
            }
        }
        MenuAction::SetServer => {
            let current = shared.and_then(|s| s.snapshot().server).unwrap_or_default();
            if let Some(url) = prompt::server_address(hwnd, &current) {
                send(Command::SetServer(url));
            }
        }
        MenuAction::Quit => send(Command::Quit),
    }
}

extern "system" fn window_proc(
    hwnd: HWND,
    message: u32,
    wparam: WPARAM,
    lparam: LPARAM,
) -> LRESULT {
    let taskbar_created =
        STATE.with(|s| s.borrow().as_ref().map_or(0, |state| state.taskbar_created));
    match message {
        WM_TRAY => {
            let event = u32::try_from(lparam.0 & 0xFFFF).unwrap_or(0);
            match event {
                WM_CONTEXTMENU | NIN_SELECT | NIN_KEYSELECT => {
                    let context = STATE.with(|s| {
                        s.borrow()
                            .as_ref()
                            .map(|state| (state.shared.clone(), state.commands.clone()))
                    });
                    if let Some((shared, commands)) = context {
                        let entries = shared
                            .as_ref()
                            .map(|s| s.snapshot().menu())
                            .unwrap_or_default();
                        if let Some(action) = choose(hwnd, &entries) {
                            perform(hwnd, action, &commands, shared.as_ref());
                        }
                    }
                }
                NIN_BALLOONUSERCLICK => {
                    let link = STATE.with(|s| {
                        s.borrow_mut()
                            .as_mut()
                            .and_then(|state| state.notice_link.take())
                    });
                    if let Some(link) = link
                        && let Err(err) = shell::open_url(&link)
                    {
                        tracing::warn!(error = %err, "could not open the browser");
                    }
                }
                _ => {}
            }
            LRESULT(0)
        }
        WM_REFRESH => {
            STATE.with(|s| {
                if let Some(state) = s.borrow().as_ref() {
                    refresh_tip(state);
                }
            });
            LRESULT(0)
        }
        WM_NOTICE => {
            STATE.with(|s| {
                if let Some(state) = s.borrow_mut().as_mut() {
                    show_notice(state);
                }
            });
            LRESULT(0)
        }
        WM_CLOSE => {
            // SAFETY: this thread's own window.
            unsafe {
                let _ = DestroyWindow(hwnd);
            }
            LRESULT(0)
        }
        WM_DESTROY => {
            STATE.with(|s| {
                if let Some(state) = s.borrow().as_ref() {
                    let data = icon_data(state.hwnd);
                    // SAFETY: removes this window's icon and frees the one
                    // loaded for it.
                    unsafe {
                        let _ = Shell_NotifyIconW(NIM_DELETE, &raw const data);
                        let _ = DestroyIcon(state.icon);
                    }
                }
            });
            // SAFETY: ends this thread's message loop.
            unsafe { PostQuitMessage(0) };
            LRESULT(0)
        }
        _ if taskbar_created != 0 && message == taskbar_created => {
            STATE.with(|s| {
                if let Some(state) = s.borrow().as_ref() {
                    add_icon(state);
                }
            });
            LRESULT(0)
        }
        // SAFETY: default handling.
        _ => unsafe { DefWindowProcW(hwnd, message, wparam, lparam) },
    }
}

impl Tray {
    /// Creates the icon's window and puts the icon in the notification area.
    pub fn create(commands: UnboundedSender<Command>) -> windows_core::Result<(Self, Arc<TrayUi>)> {
        // SAFETY: registering this application's window class.
        let instance = unsafe { GetModuleHandleW(None)? };
        let class = WNDCLASSW {
            lpfnWndProc: Some(window_proc),
            hInstance: instance.into(),
            lpszClassName: CLASS_NAME,
            ..WNDCLASSW::default()
        };
        // SAFETY: as above.
        unsafe { RegisterClassW(&raw const class) };
        // SAFETY: a hidden top-level window: the tray needs one that receives
        // the TaskbarCreated broadcast, which message-only windows do not.
        let hwnd = unsafe {
            CreateWindowExW(
                WINDOW_EX_STYLE(0),
                CLASS_NAME,
                w!("Trenova Capture"),
                WS_OVERLAPPED,
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
        // SAFETY: registers (or looks up) a system-wide message name.
        let taskbar_created = unsafe { RegisterWindowMessageW(w!("TaskbarCreated")) };
        let notices = Arc::new(Mutex::new(VecDeque::new()));
        let state = TrayState {
            hwnd,
            shared: None,
            commands,
            notices: Arc::clone(&notices),
            icon: load_icon(),
            notice_link: None,
            taskbar_created,
        };
        add_icon(&state);
        STATE.with(|s| *s.borrow_mut() = Some(state));
        let ui = Arc::new(TrayUi {
            window: hwnd.0.expose_provenance(),
            notices,
        });
        Ok((Self { hwnd }, ui))
    }

    /// Gives the tray the state it draws.
    pub fn attach(&self, shared: Arc<Shared>) {
        STATE.with(|s| {
            if let Some(state) = s
                .borrow_mut()
                .as_mut()
                .filter(|state| state.hwnd == self.hwnd)
            {
                state.shared = Some(shared);
                refresh_tip(state);
            }
        });
    }

    /// Runs the message loop until the tray closes.
    pub fn run(&self) {
        let mut message = MSG::default();
        // SAFETY: this thread's message loop.
        while unsafe { GetMessageW(&raw mut message, None, 0, 0) }.as_bool() {
            // SAFETY: the message just received.
            unsafe {
                let _ = TranslateMessage(&raw const message);
                DispatchMessageW(&raw const message);
            }
        }
        tracing::debug!(hwnd = ?self.hwnd, "the tray closed");
    }
}
