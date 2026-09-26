//! Trenova Capture: the tray agent.
//!
//! One runs per Windows session, started at sign-in. The window is the tray
//! icon and its menu; everything else happens in the agent
//! (`trenova_capture::agent`) on a background runtime.

#![cfg_attr(windows, windows_subsystem = "windows")]

#[cfg(windows)]
mod app;
#[cfg(windows)]
mod tray;

use std::process::ExitCode;

fn main() -> ExitCode {
    #[cfg(windows)]
    {
        app::run()
    }
    #[cfg(not(windows))]
    {
        use std::io::Write;
        let _ = writeln!(std::io::stderr(), "Trenova Capture runs on Windows only.");
        ExitCode::from(2)
    }
}
