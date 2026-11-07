//! The updater on Windows: what installing needs from the system, and the
//! command line.

mod install;
mod service;
mod sessions;
mod trust;

use std::io::Write;
use std::path::{Path, PathBuf};
use std::process::{Command, ExitCode};
use std::time::Duration;

use capture_platform::acl::{SYSTEM_ONLY_SDDL, create_trusted};
use capture_platform::{logging, machine, paths, settings};
use capture_protocol::release::{pinned_public_key, unix_now};
use tokio::sync::watch;
use trenova_capture_update::signature::Revocation;
use trenova_capture_update::{Http, Installer, Stop, UpdaterError, manifest_url, run};
use windows::Win32::System::SystemInformation::GetSystemDirectoryW;

/// The tray agent, beside the updater.
const AGENT_EXE: &str = "trenova-capture.exe";
/// The processes an install must not find running.
const CLOSE_BEFORE_INSTALL: [&str; 3] = [
    "trenova-capture.exe",
    "trenova-capture-scan-x64.exe",
    "trenova-capture-scan-x86.exe",
];
/// Downloads older than this are leftovers of an interrupted update.
const STALE_DOWNLOAD: Duration = Duration::from_hours(24);
/// Windows Installer's "done, reboot to finish" codes.
const REBOOT_REQUIRED: [i32; 2] = [3010, 1641];

/// Writes a line the installer's log keeps (Windows Installer records what
/// its custom actions print).
pub(crate) fn say(message: &str) {
    let _ = writeln!(std::io::stderr(), "{message}");
}

const USAGE: &str = "usage: trenova-capture-update <service | run <manifest url> | verify-publisher <file> | relaunch | configure | install | uninstall>";

/// The system directory, for tools that must not come from `PATH`.
pub(crate) fn system_dir() -> std::io::Result<PathBuf> {
    let mut buffer = [0u16; 260];
    // SAFETY: the buffer's length is passed with it.
    let len = unsafe { GetSystemDirectoryW(Some(&mut buffer)) } as usize;
    if len == 0 || len >= buffer.len() {
        return Err(std::io::Error::last_os_error());
    }
    Ok(PathBuf::from(String::from_utf16_lossy(&buffer[..len])))
}

fn logs_dir() -> std::io::Result<PathBuf> {
    Ok(paths::shared_dir()?.join("logs"))
}

/// The installed product on this computer.
#[derive(Debug)]
pub(crate) struct WindowsInstaller {
    exe: PathBuf,
}

impl WindowsInstaller {
    pub(crate) fn new() -> std::io::Result<Self> {
        Ok(Self {
            exe: std::env::current_exe()?,
        })
    }

    /// Where the installer beside this one puts the agent.
    fn agent_exe(&self) -> PathBuf {
        self.exe.with_file_name(AGENT_EXE)
    }
}

impl Installer for WindowsInstaller {
    fn installed_version(&self) -> String {
        env!("CARGO_PKG_VERSION").to_owned()
    }

    fn windows_build(&self) -> u32 {
        machine::windows_build()
    }

    fn now(&self) -> i64 {
        unix_now()
    }

    fn policy_allows(&self) -> bool {
        settings::auto_update_allowed()
    }

    fn download_dir(&self) -> Result<PathBuf, UpdaterError> {
        let dir = paths::shared_dir()?.join("updates");
        create_trusted(&dir, SYSTEM_ONLY_SDDL, 2)?;
        sweep(&dir);
        Ok(dir)
    }

    fn verify_installer(&self, path: &Path) -> Result<(), UpdaterError> {
        if trust::verify_same_publisher(path, &self.exe)? == Revocation::Unreachable {
            tracing::warn!(
                "the revocation servers could not be reached; the installer's signature was accepted without them"
            );
        }
        Ok(())
    }

    fn install(&self, path: &Path) -> Result<(), UpdaterError> {
        sessions::close_processes(&CLOSE_BEFORE_INSTALL);
        let log = logs_dir()?.join("install.log");
        let status = Command::new(system_dir()?.join("msiexec.exe"))
            .arg("/i")
            .arg(path)
            .args(["/qn", "/norestart", "/l*v"])
            .arg(&log)
            .status()?;
        match status.code() {
            Some(0) => Ok(()),
            Some(code) if REBOOT_REQUIRED.contains(&code) => {
                tracing::warn!(code, "installed; Windows wants a restart to finish");
                Ok(())
            }
            Some(code) => Err(UpdaterError::Install(format!(
                "msiexec exited with {code}; see {}",
                log.display()
            ))),
            None => Err(UpdaterError::Install("msiexec was interrupted".into())),
        }
    }

    fn relaunch_agents(&self) -> Result<(), UpdaterError> {
        sessions::start_in_every_session(&self.agent_exe())
    }
}

/// Removes downloads an interrupted update left behind.
fn sweep(dir: &Path) {
    let Ok(entries) = std::fs::read_dir(dir) else {
        return;
    };
    for entry in entries.flatten() {
        let old = entry
            .metadata()
            .and_then(|m| m.modified())
            .ok()
            .and_then(|modified| modified.elapsed().ok())
            .is_some_and(|age| age >= STALE_DOWNLOAD);
        if old {
            let _ = std::fs::remove_file(entry.path());
        }
    }
}

/// One update, reported. `stop` and `committing` are [`run`]'s.
pub(crate) fn update(
    manifest: &str,
    stop: &Stop,
    committing: impl FnOnce(),
) -> Result<(), UpdaterError> {
    let url = manifest_url(manifest)?;
    let installer = WindowsInstaller::new()?;
    let runtime = tokio::runtime::Builder::new_current_thread()
        .enable_all()
        .build()?;
    let source = Http::new()?;
    let outcome = runtime.block_on(run(
        &url,
        pinned_public_key().as_ref(),
        &source,
        &installer,
        stop,
        committing,
    ))?;
    tracing::info!(from = %outcome.from, to = %outcome.to, "updated");
    Ok(())
}

/// Checks `file` as an update's installer would be checked, and says how far
/// revocation was checked.
fn verify_publisher(file: &Path) -> Result<(), UpdaterError> {
    let installer = WindowsInstaller::new()?;
    match trust::verify_same_publisher(file, &installer.exe)? {
        Revocation::Checked => say("No certificate in its chain is revoked."),
        Revocation::Unreachable => say(
            "The revocation servers could not be reached; its signature was accepted without them.",
        ),
    }
    Ok(())
}

fn step(what: &str, result: Result<(), UpdaterError>) -> ExitCode {
    match result {
        Ok(()) => {
            say(&format!("{what}: done."));
            ExitCode::SUCCESS
        }
        Err(err) => {
            say(&format!("{what}: {err}"));
            if matches!(&err, UpdaterError::Io(io) if io.kind() == std::io::ErrorKind::PermissionDenied)
            {
                say("Run this from an elevated prompt.");
            }
            ExitCode::FAILURE
        }
    }
}

pub fn main() -> ExitCode {
    let mut args = std::env::args().skip(1);
    let command = args.next().unwrap_or_default();
    let _log = matches!(command.as_str(), "service" | "run" | "relaunch")
        .then(|| {
            logs_dir()
                .ok()
                .and_then(|dir| logging::init(&dir, "updater"))
        })
        .flatten();
    match command.as_str() {
        "service" => service::dispatch(),
        "run" => {
            if let Some(manifest) = args.next() {
                let (_never, stop) = watch::channel(false);
                step("Updating Trenova Capture", update(&manifest, &stop, || {}))
            } else {
                say(USAGE);
                ExitCode::from(2)
            }
        }
        "verify-publisher" => {
            if let Some(file) = args.next() {
                step(
                    "Checking the installer's signature",
                    verify_publisher(Path::new(&file)),
                )
            } else {
                say(USAGE);
                ExitCode::from(2)
            }
        }
        "relaunch" => step(
            "Starting Trenova Capture",
            WindowsInstaller::new()
                .map_err(UpdaterError::from)
                .and_then(|installer| installer.relaunch_agents()),
        ),
        "configure" => step("Configuring the updater", install::configure_service()),
        "install" => step("Installing the updater", install::install_service()),
        "uninstall" => step("Removing the updater", install::uninstall_service()),
        _ => {
            say(USAGE);
            ExitCode::from(2)
        }
    }
}
