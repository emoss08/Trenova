//! Trenova Capture's print service.
//!
//! `trenova-capture-svc service` is how the Service Control Manager starts
//! it. The installer runs `install` and `install-printer` (and their
//! `uninstall` counterparts) elevated; `run` serves in a console, for
//! development.

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
            "The Trenova Capture print service runs on Windows only."
        );
        ExitCode::from(2)
    }
}
