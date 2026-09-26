//! The service on Windows: its command line, the printer it serves, and the
//! Service Control Manager.

mod acl;
mod install;
mod service;
mod spooler;

use std::io::Write;
use std::path::PathBuf;
use std::process::ExitCode;
use std::sync::Arc;

use capture_ipp::Printer;
use capture_platform::{accounts, logging, machine, paths, settings};
use tokio::net::TcpListener;
use tokio_util::sync::CancellationToken;
use trenova_capture_svc::attribution::{Attributor, Patience};
use trenova_capture_svc::handler::PrintHandler;
use trenova_capture_svc::listener::{self, Limits};
use trenova_capture_svc::{DEFAULT_PORT, printer_config};

use self::acl::SecuredInboxes;
use self::spooler::WindowsPrintSystem;

type Handler = PrintHandler<WindowsPrintSystem, SecuredInboxes>;

fn say(message: &str) {
    let _ = writeln!(std::io::stderr(), "{message}");
}

const USAGE: &str = "usage: trenova-capture-svc <service | run | configure | install | uninstall | install-printer | uninstall-printer>";

/// The loopback port: policy's, the installer's, or the default.
pub(crate) fn port() -> u16 {
    settings::print_port().unwrap_or(DEFAULT_PORT)
}

/// Where the service writes its logs.
fn logs_dir() -> std::io::Result<PathBuf> {
    Ok(paths::shared_dir()?.join("logs"))
}

/// The printer, with everything it needs from Windows. `console` is a
/// development run, which may not be the service and so falls back to its
/// own account for the inbox ACLs.
fn printer(console: bool) -> std::io::Result<Arc<Printer<Handler>>> {
    let service_sid = match acl::service_sid() {
        Ok(sid) => sid,
        Err(err) if console => {
            tracing::warn!(error = %err, "the service is not installed; granting inboxes to this account");
            accounts::current_user_sid()?
        }
        Err(err) => return Err(err),
    };
    let inboxes = SecuredInboxes {
        root: paths::print_spool_dir()?,
        service_sid,
    };
    let handler = PrintHandler::new(
        Attributor::new(WindowsPrintSystem, Patience::default()),
        inboxes,
    );
    let machine = settings::machine_guid().unwrap_or_else(machine::machine_name);
    Ok(Arc::new(Printer::new(
        printer_config(port(), &machine, Limits::default().max_request_bytes),
        handler,
    )))
}

/// Binds the printer's port and builds the printer, so a failure is known
/// before the service reports that it is running.
pub(crate) async fn start(console: bool) -> std::io::Result<(TcpListener, Arc<Printer<Handler>>)> {
    let printer = printer(console)?;
    let listener = listener::bind(port()).await?;
    Ok((listener, printer))
}

/// Serves until `stop`.
pub(crate) async fn serve(
    listener: TcpListener,
    printer: Arc<Printer<Handler>>,
    stop: CancellationToken,
) -> std::io::Result<()> {
    listener::serve(listener, printer, Limits::default(), stop).await
}

fn run_console() -> ExitCode {
    let stop = CancellationToken::new();
    let runtime = match tokio::runtime::Builder::new_multi_thread()
        .enable_all()
        .build()
    {
        Ok(runtime) => runtime,
        Err(err) => {
            say(&format!("could not start: {err}"));
            return ExitCode::FAILURE;
        }
    };
    say(&format!(
        "Serving the Trenova printer at {}; press Ctrl+C to stop.",
        trenova_capture_svc::printer_url(port())
    ));
    let ctrl_c = stop.clone();
    runtime.spawn(async move {
        let _ = tokio::signal::ctrl_c().await;
        ctrl_c.cancel();
    });
    let served = runtime.block_on(async move {
        let (listener, printer) = start(true).await?;
        serve(listener, printer, stop).await
    });
    match served {
        Ok(()) => ExitCode::SUCCESS,
        Err(err) => {
            say(&format!("stopped: {err}"));
            tracing::error!(error = %err, "the printer stopped");
            ExitCode::FAILURE
        }
    }
}

/// Runs an install step and reports it.
fn step(what: &str, result: std::io::Result<()>) -> ExitCode {
    match result {
        Ok(()) => {
            say(&format!("{what}: done."));
            ExitCode::SUCCESS
        }
        Err(err) => {
            say(&format!("{what}: {err}"));
            if err.kind() == std::io::ErrorKind::PermissionDenied {
                say("Run this from an elevated prompt.");
            }
            ExitCode::FAILURE
        }
    }
}

pub fn main() -> ExitCode {
    let command = std::env::args().nth(1).unwrap_or_default();
    let _log = matches!(command.as_str(), "service" | "run")
        .then(|| {
            logs_dir()
                .ok()
                .and_then(|dir| logging::init(&dir, "service"))
        })
        .flatten();
    match command.as_str() {
        "service" => service::dispatch(),
        "run" => run_console(),
        "configure" => step(
            "Creating the service's directories",
            install::create_directories(),
        ),
        "install" => step("Installing the print service", install::install_service()),
        "uninstall" => step("Removing the print service", install::uninstall_service()),
        "install-printer" => step(
            "Adding the Trenova printer",
            install::install_printer(port()),
        ),
        "uninstall-printer" => step("Removing the Trenova printer", install::uninstall_printer()),
        _ => {
            say(USAGE);
            ExitCode::from(2)
        }
    }
}
