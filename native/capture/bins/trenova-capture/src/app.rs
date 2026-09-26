//! Starting the agent on Windows: logging, one instance per session, the
//! tray on this thread and the agent on a runtime beside it.

use std::path::{Path, PathBuf};
use std::process::ExitCode;
use std::sync::Arc;
use std::time::Duration;

use capture_client::{AgentInfo, SecretStore, Server};
use capture_platform::settings::{self, ServerSource};
use capture_platform::{
    CredentialManager, Dpapi, SingleInstance, accounts, logging, machine, paths, shell,
};
use tokio::sync::mpsc;
use tokio_util::sync::CancellationToken;
use trenova_capture::agent::{self, Environment, Machine, ServerSetting};
use trenova_capture::scanners::HelperHost;
use trenova_capture::state::{Command, Shared, Ui};
use windows::Win32::UI::HiDpi::{
    DPI_AWARENESS_CONTEXT_PER_MONITOR_AWARE_V2, SetProcessDpiAwarenessContext,
};

use crate::tray::Tray;

/// How long to wait before trying again after capture is paused.
const RECHECK_AFTER: Duration = Duration::from_secs(600);

/// What was asked for on the command line.
#[derive(Debug, Default)]
struct Arguments {
    /// `--server <address>`: remember this server for the Windows user.
    server: Option<String>,
    /// `--sign-in`: start pairing at once, as the installer's "sign in now"
    /// does.
    sign_in: bool,
}

fn arguments() -> Arguments {
    let mut parsed = Arguments::default();
    let mut args = std::env::args().skip(1);
    while let Some(arg) = args.next() {
        match arg.as_str() {
            "--server" => parsed.server = args.next(),
            "--sign-in" => parsed.sign_in = true,
            other => tracing::warn!(argument = other, "ignoring an unknown argument"),
        }
    }
    parsed
}

/// The server address from the registry.
struct RegistrySetting;

impl ServerSetting for RegistrySetting {
    fn load(&self) -> Option<String> {
        settings::server_url().map(|(url, _)| url)
    }

    fn save(&self, url: &str) -> std::io::Result<()> {
        settings::set_server_url(url)
    }

    fn locked(&self) -> bool {
        matches!(settings::server_url(), Some((_, ServerSource::Policy)))
    }
}

/// `%ProgramData%\Trenova\Capture\spool\<this user's SID>`, where the print
/// service leaves what this person prints.
fn print_inbox() -> Option<PathBuf> {
    let root = paths::print_spool_dir()
        .inspect_err(|err| tracing::warn!(error = %err, "no print spool directory"))
        .ok()?;
    let sid = accounts::current_user_sid()
        .inspect_err(|err| tracing::warn!(error = %err, "could not read this user's SID"))
        .ok()?;
    Some(root.join(sid))
}

fn environment(data_dir: &Path) -> Environment {
    let helpers = std::env::current_exe()
        .ok()
        .and_then(|exe| exe.parent().map(Path::to_path_buf))
        .unwrap_or_else(|| PathBuf::from("."));
    Environment {
        spool_dir: data_dir.join("spool"),
        print_inbox: print_inbox(),
        agent: AgentInfo {
            version: env!("CARGO_PKG_VERSION").to_owned(),
            os_version: machine::os_version(),
        },
        machine: Machine {
            name: machine::machine_name(),
            windows_user: machine::windows_user(),
        },
        scanners: Arc::new(HelperHost::new(helpers)),
        secrets: Arc::new(|server: &Server| {
            Arc::new(CredentialManager::for_host(server.host())) as Arc<dyn SecretStore>
        }),
        protector: Arc::new(Dpapi),
        server_setting: Arc::new(RegistrySetting),
        browser: Arc::new(|url: &str| {
            if let Err(err) = shell::open_url(url) {
                tracing::warn!(error = %err, "could not open the browser");
            }
        }),
        recheck_after: RECHECK_AFTER,
    }
}

pub fn run() -> ExitCode {
    // SAFETY: set once, before any window exists.
    unsafe {
        let _ = SetProcessDpiAwarenessContext(DPI_AWARENESS_CONTEXT_PER_MONITOR_AWARE_V2);
    }
    let Ok(data_dir) = paths::data_dir() else {
        return ExitCode::FAILURE;
    };
    let _log = logging::init(&data_dir.join("logs"), "agent");
    tracing::info!(
        version = env!("CARGO_PKG_VERSION"),
        "Trenova Capture starting"
    );

    let arguments = arguments();
    if let Some(server) = &arguments.server
        && let Err(err) = settings::set_server_url(server)
    {
        tracing::error!(error = %err, "could not save the server address");
    }

    let _instance = match SingleInstance::acquire() {
        Ok(Some(instance)) => instance,
        Ok(None) => {
            tracing::info!("Trenova Capture is already running in this session");
            return ExitCode::SUCCESS;
        }
        Err(err) => {
            tracing::error!(error = %err, "could not check for another instance");
            return ExitCode::FAILURE;
        }
    };

    let (commands, received) = mpsc::unbounded_channel();
    let (tray, ui) = match Tray::create(commands.clone()) {
        Ok(created) => created,
        Err(err) => {
            tracing::error!(error = %err, "could not create the tray icon");
            return ExitCode::FAILURE;
        }
    };
    let shared = Shared::new(Arc::clone(&ui) as Arc<dyn Ui>);
    tray.attach(Arc::clone(&shared));

    let cancel = CancellationToken::new();
    let env = environment(&data_dir);
    let agent_cancel = cancel.clone();
    let agent_ui = Arc::clone(&ui);
    let runtime = std::thread::Builder::new()
        .name("trenova-capture-agent".into())
        .spawn(move || {
            match tokio::runtime::Builder::new_multi_thread()
                .worker_threads(2)
                .enable_all()
                .build()
            {
                Ok(runtime) => {
                    if let Err(err) =
                        runtime.block_on(agent::run(env, shared, received, agent_cancel))
                    {
                        tracing::error!(error = %err, "the agent stopped");
                    }
                    runtime.shutdown_timeout(Duration::from_secs(5));
                }
                Err(err) => tracing::error!(error = %err, "could not start the agent"),
            }
            agent_ui.quit();
        });
    let runtime = match runtime {
        Ok(handle) => handle,
        Err(err) => {
            tracing::error!(error = %err, "could not start the agent thread");
            return ExitCode::FAILURE;
        }
    };

    if arguments.sign_in {
        let _ = commands.send(Command::SignIn);
    }
    tray.run();

    cancel.cancel();
    let _ = commands.send(Command::Quit);
    if runtime.join().is_err() {
        tracing::error!("the agent thread panicked");
    }
    tracing::info!("Trenova Capture stopped");
    ExitCode::SUCCESS
}
