//! The Trenova Capture updater.
//!
//! `trenova-capture-update service` is how the Service Control Manager
//! starts it, with the manifest address as the start argument; the installer
//! runs `install` and `uninstall` elevated; `relaunch` starts the agent again
//! for everyone signed in, which the installer asks for at the end of an
//! upgrade; `run <url>` updates from a console, for development.

#[cfg(windows)]
mod win;

use std::process::ExitCode;

fn main() -> ExitCode {
    #[cfg(windows)]
    {
        win::main()
    }
    #[cfg(not(windows))]
    {
        use std::io::Write;
        let _ = writeln!(
            std::io::stderr(),
            "The Trenova Capture updater runs on Windows only."
        );
        ExitCode::from(2)
    }
}
