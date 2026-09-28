//! Asking for the server address: a small window with a text box, run modal
//! on the tray's thread.

use std::cell::RefCell;

use windows::Win32::Foundation::{HINSTANCE, HWND, LPARAM, LRESULT, WPARAM};
use windows::Win32::Graphics::Gdi::{
    COLOR_BTNFACE, CreateFontIndirectW, DeleteObject, HBRUSH, HFONT, HGDIOBJ,
};
use windows::Win32::System::LibraryLoader::GetModuleHandleW;
use windows::Win32::UI::Controls::EM_SETSEL;
use windows::Win32::UI::HiDpi::GetDpiForSystem;
use windows::Win32::UI::Input::KeyboardAndMouse::{EnableWindow, SetFocus};
use windows::Win32::UI::WindowsAndMessaging::{
    BS_DEFPUSHBUTTON, BS_PUSHBUTTON, CreateWindowExW, DefWindowProcW, DestroyWindow,
    DispatchMessageW, ES_AUTOHSCROLL, GetMessageW, GetSystemMetrics, GetWindowTextLengthW,
    GetWindowTextW, HMENU, IDCANCEL, IDOK, IsDialogMessageW, MSG, NONCLIENTMETRICSW,
    RegisterClassW, SM_CXSCREEN, SM_CYSCREEN, SPI_GETNONCLIENTMETRICS,
    SYSTEM_PARAMETERS_INFO_UPDATE_FLAGS, SendMessageW, SetForegroundWindow, SystemParametersInfoW,
    TranslateMessage, WINDOW_EX_STYLE, WINDOW_STYLE, WM_CLOSE, WM_COMMAND, WM_DESTROY, WM_SETFONT,
    WNDCLASSW, WS_BORDER, WS_CAPTION, WS_CHILD, WS_EX_DLGMODALFRAME, WS_EX_TOPMOST, WS_POPUP,
    WS_SYSMENU, WS_TABSTOP, WS_VISIBLE,
};
use windows_core::{PCWSTR, w};

const CLASS_NAME: PCWSTR = w!("TrenovaCaptureServerPrompt");
const EDIT_ID: i32 = 100;

struct Prompt {
    edit: HWND,
    answer: Option<String>,
    done: bool,
}

thread_local! {
    static PROMPT: RefCell<Option<Prompt>> = const { RefCell::new(None) };
}

fn read_text(edit: HWND) -> String {
    // SAFETY: the edit control belongs to this thread's prompt.
    let len = unsafe { GetWindowTextLengthW(edit) };
    let mut buffer = vec![0u16; usize::try_from(len).unwrap_or(0) + 1];
    // SAFETY: the buffer holds the text and its NUL.
    let copied = unsafe { GetWindowTextW(edit, &mut buffer) };
    String::from_utf16_lossy(&buffer[..usize::try_from(copied).unwrap_or(0)])
}

extern "system" fn prompt_proc(
    hwnd: HWND,
    message: u32,
    wparam: WPARAM,
    lparam: LPARAM,
) -> LRESULT {
    match message {
        WM_COMMAND => {
            let id = i32::try_from(wparam.0 & 0xFFFF).unwrap_or(0);
            if id == IDOK.0 {
                PROMPT.with(|p| {
                    if let Some(prompt) = p.borrow_mut().as_mut() {
                        prompt.answer = Some(read_text(prompt.edit).trim().to_owned());
                    }
                });
            }
            if id == IDOK.0 || id == IDCANCEL.0 {
                // SAFETY: the prompt's own window.
                unsafe {
                    let _ = DestroyWindow(hwnd);
                }
            }
            LRESULT(0)
        }
        WM_CLOSE => {
            // SAFETY: as above.
            unsafe {
                let _ = DestroyWindow(hwnd);
            }
            LRESULT(0)
        }
        WM_DESTROY => {
            PROMPT.with(|p| {
                if let Some(prompt) = p.borrow_mut().as_mut() {
                    prompt.done = true;
                }
            });
            LRESULT(0)
        }
        // SAFETY: default handling.
        _ => unsafe { DefWindowProcW(hwnd, message, wparam, lparam) },
    }
}

/// The system's message font, so the prompt reads like the rest of Windows.
fn message_font() -> Option<HFONT> {
    let mut metrics = NONCLIENTMETRICSW {
        cbSize: u32::try_from(std::mem::size_of::<NONCLIENTMETRICSW>()).unwrap_or(0),
        ..NONCLIENTMETRICSW::default()
    };
    // SAFETY: a correctly sized structure for the system to fill.
    unsafe {
        SystemParametersInfoW(
            SPI_GETNONCLIENTMETRICS,
            metrics.cbSize,
            Some((&raw mut metrics).cast()),
            SYSTEM_PARAMETERS_INFO_UPDATE_FLAGS(0),
        )
        .ok()?;
        let font = CreateFontIndirectW(&raw const metrics.lfMessageFont);
        (!font.is_invalid()).then_some(font)
    }
}

struct Layout {
    scale: i32,
}

impl Layout {
    fn px(&self, value: i32) -> i32 {
        value * self.scale / 96
    }
}

/// The prompt's window, centred on the screen.
fn create_window(instance: HINSTANCE, layout: &Layout) -> Option<HWND> {
    let class = WNDCLASSW {
        lpfnWndProc: Some(prompt_proc),
        hInstance: instance,
        lpszClassName: CLASS_NAME,
        hbrBackground: HBRUSH(std::ptr::with_exposed_provenance_mut(
            usize::try_from(COLOR_BTNFACE.0 + 1).unwrap_or(16),
        )),
        ..WNDCLASSW::default()
    };
    // SAFETY: registering this application's class; a repeat fails harmlessly.
    unsafe { RegisterClassW(&raw const class) };
    // SAFETY: reads the screen size.
    let screen = unsafe { (GetSystemMetrics(SM_CXSCREEN), GetSystemMetrics(SM_CYSCREEN)) };
    let (width, height) = (layout.px(460), layout.px(170));
    // SAFETY: a top-level window the caller destroys.
    unsafe {
        CreateWindowExW(
            WS_EX_DLGMODALFRAME | WS_EX_TOPMOST,
            CLASS_NAME,
            w!("Trenova server address"),
            WS_POPUP | WS_CAPTION | WS_SYSMENU | WS_VISIBLE,
            (screen.0 - width) / 2,
            (screen.1 - height) / 2,
            width,
            height,
            None,
            None,
            Some(instance),
            None,
        )
    }
    .ok()
}

/// One child control, positioned in 96-DPI units.
struct Control {
    class: PCWSTR,
    text: PCWSTR,
    style: WINDOW_STYLE,
    frame: (i32, i32, i32, i32),
    id: i32,
}

fn create_control(
    window: HWND,
    instance: HINSTANCE,
    layout: &Layout,
    control: &Control,
) -> Option<HWND> {
    let (x, y, width, height) = control.frame;
    let id = usize::try_from(control.id).unwrap_or(0);
    // SAFETY: a child of the prompt window, destroyed with it.
    unsafe {
        CreateWindowExW(
            WINDOW_EX_STYLE(0),
            control.class,
            control.text,
            WS_CHILD | WS_VISIBLE | control.style,
            layout.px(x),
            layout.px(y),
            layout.px(width),
            layout.px(height),
            Some(window),
            Some(HMENU(std::ptr::with_exposed_provenance_mut(id))),
            Some(instance),
            None,
        )
    }
    .ok()
}

/// The label, text box and buttons; the text box is second.
fn create_controls(
    window: HWND,
    instance: HINSTANCE,
    layout: &Layout,
    current: &[u16],
) -> Option<Vec<HWND>> {
    let edit_style =
        WS_BORDER | WS_TABSTOP | WINDOW_STYLE(u32::try_from(ES_AUTOHSCROLL).unwrap_or(0));
    let button = |style: i32| WS_TABSTOP | WINDOW_STYLE(u32::try_from(style).unwrap_or(0));
    [
        Control {
            class: w!("STATIC"),
            text: w!("The address of your Trenova server, for example tms.yourcompany.com"),
            style: WINDOW_STYLE(0),
            frame: (16, 16, 420, 20),
            id: 0,
        },
        Control {
            class: w!("EDIT"),
            text: PCWSTR(current.as_ptr()),
            style: edit_style,
            frame: (16, 42, 420, 24),
            id: EDIT_ID,
        },
        Control {
            class: w!("BUTTON"),
            text: w!("Save"),
            style: button(BS_DEFPUSHBUTTON),
            frame: (256, 84, 86, 28),
            id: IDOK.0,
        },
        Control {
            class: w!("BUTTON"),
            text: w!("Cancel"),
            style: button(BS_PUSHBUTTON),
            frame: (350, 84, 86, 28),
            id: IDCANCEL.0,
        },
    ]
    .iter()
    .map(|control| create_control(window, instance, layout, control))
    .collect()
}

/// Runs the prompt until it closes, with the tray disabled meanwhile.
fn run_modal(owner: HWND, window: HWND, edit: HWND) {
    // SAFETY: focus to the text box with its text selected; the tray is
    // re-enabled below.
    unsafe {
        let _ = EnableWindow(owner, false);
        let _ = SetForegroundWindow(window);
        let _ = SetFocus(Some(edit));
        SendMessageW(edit, EM_SETSEL, Some(WPARAM(0)), Some(LPARAM(-1)));
    }
    let mut message = MSG::default();
    while !PROMPT.with(|p| p.borrow().as_ref().is_none_or(|prompt| prompt.done)) {
        // SAFETY: this thread's message loop, with the prompt's keyboard
        // handling (Tab, Enter, Escape).
        unsafe {
            if !GetMessageW(&raw mut message, None, 0, 0).as_bool() {
                break;
            }
            if !IsDialogMessageW(window, &raw const message).as_bool() {
                let _ = TranslateMessage(&raw const message);
                DispatchMessageW(&raw const message);
            }
        }
    }
    // SAFETY: as above.
    unsafe {
        let _ = EnableWindow(owner, true);
    }
}

/// Asks for the server address, returning what was entered, or `None` if
/// the person cancelled.
pub fn server_address(owner: HWND, current: &str) -> Option<String> {
    // SAFETY: this module's own instance handle.
    let instance: HINSTANCE = unsafe { GetModuleHandleW(None) }.ok()?.into();
    // SAFETY: reads the system DPI.
    let dpi = unsafe { GetDpiForSystem() };
    let layout = Layout {
        scale: i32::try_from(dpi).unwrap_or(96),
    };
    let window = create_window(instance, &layout)?;
    let current: Vec<u16> = current.encode_utf16().chain(Some(0)).collect();
    let Some(controls) = create_controls(window, instance, &layout, &current) else {
        // SAFETY: the prompt window created above.
        unsafe {
            let _ = DestroyWindow(window);
        }
        return None;
    };
    let edit = controls[1];

    let font = message_font();
    if let Some(font) = font {
        for child in &controls {
            // SAFETY: setting a font on this prompt's own controls.
            unsafe {
                SendMessageW(
                    *child,
                    WM_SETFONT,
                    Some(WPARAM(font.0 as usize)),
                    Some(LPARAM(1)),
                );
            }
        }
    }
    PROMPT.with(|p| {
        *p.borrow_mut() = Some(Prompt {
            edit,
            answer: None,
            done: false,
        });
    });
    run_modal(owner, window, edit);
    if let Some(font) = font {
        // SAFETY: the font created for the prompt, no longer used.
        unsafe {
            let _ = DeleteObject(HGDIOBJ(font.0));
        }
    }
    PROMPT
        .with(|p| p.borrow_mut().take())
        .and_then(|prompt| prompt.answer)
        .filter(|answer| !answer.is_empty())
}
