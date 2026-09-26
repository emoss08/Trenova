//! Installing and removing the service and the printer. The installer runs
//! these elevated; each can be run again safely.

use std::ffi::OsString;
use std::io;
use std::path::PathBuf;
use std::process::Command;
use std::time::{Duration, Instant};

use capture_platform::paths;
use trenova_capture_svc::{
    PRINTER_NAME, SERVICE_DESCRIPTION, SERVICE_DISPLAY_NAME, SERVICE_NAME, printer_url,
};
use windows::Win32::System::SystemInformation::GetSystemDirectoryW;
use windows_service::service::{
    ServiceAccess, ServiceAction, ServiceActionType, ServiceErrorControl, ServiceFailureActions,
    ServiceFailureResetPeriod, ServiceInfo, ServiceSidType, ServiceStartType, ServiceState,
    ServiceType,
};
use windows_service::service_manager::{ServiceManager, ServiceManagerAccess};

use super::acl;

const LOCAL_SERVICE: &str = "NT AUTHORITY\\LocalService";
const STOP_TIMEOUT: Duration = Duration::from_secs(30);
/// `ERROR_SERVICE_DOES_NOT_EXIST`.
const NO_SUCH_SERVICE: i32 = 1060;

fn service_error(err: windows_service::Error) -> io::Error {
    match err {
        windows_service::Error::Winapi(os) => os,
        other => io::Error::other(other),
    }
}

fn manager(access: ServiceManagerAccess) -> io::Result<ServiceManager> {
    ServiceManager::local_computer(None::<&str>, access).map_err(service_error)
}

fn service_info() -> io::Result<ServiceInfo> {
    Ok(ServiceInfo {
        name: SERVICE_NAME.into(),
        display_name: SERVICE_DISPLAY_NAME.into(),
        service_type: ServiceType::OWN_PROCESS,
        start_type: ServiceStartType::AutoStart,
        error_control: ServiceErrorControl::Normal,
        executable_path: std::env::current_exe()?,
        launch_arguments: vec![OsString::from("service")],
        dependencies: Vec::new(),
        account_name: Some(LOCAL_SERVICE.into()),
        account_password: None,
    })
}

/// Creates or updates the service, gives it its own SID, restarts it on
/// failure, creates its directories, and starts it.
pub fn install_service() -> io::Result<()> {
    let manager = manager(ServiceManagerAccess::CONNECT | ServiceManagerAccess::CREATE_SERVICE)?;
    let info = service_info()?;
    let access = ServiceAccess::QUERY_STATUS
        | ServiceAccess::START
        | ServiceAccess::CHANGE_CONFIG
        | ServiceAccess::QUERY_CONFIG;
    let service = match manager.open_service(SERVICE_NAME, access) {
        Ok(service) => {
            service.change_config(&info).map_err(service_error)?;
            service
        }
        Err(_) => manager
            .create_service(&info, access)
            .map_err(service_error)?,
    };
    service
        .set_description(SERVICE_DESCRIPTION)
        .map_err(service_error)?;
    service
        .set_config_service_sid_info(ServiceSidType::Unrestricted)
        .map_err(service_error)?;
    let restart = |seconds| ServiceAction {
        action_type: ServiceActionType::Restart,
        delay: Duration::from_secs(seconds),
    };
    service
        .update_failure_actions(ServiceFailureActions {
            reset_period: ServiceFailureResetPeriod::After(Duration::from_secs(86_400)),
            reboot_msg: None,
            command: None,
            actions: Some(vec![restart(5), restart(30), restart(120)]),
        })
        .map_err(service_error)?;
    service
        .set_failure_actions_on_non_crash_failures(true)
        .map_err(service_error)?;

    create_directories()?;

    if service.query_status().map_err(service_error)?.current_state == ServiceState::Stopped {
        service.start::<OsString>(&[]).map_err(service_error)?;
    }
    Ok(())
}

/// `%ProgramData%\Trenova\Capture` and, ACL'd to the service, its `logs` and
/// `spool` directories. The MSI runs this after it has created the service,
/// since Windows Installer cannot set these ACLs itself.
pub fn create_directories() -> io::Result<()> {
    let service_sid = acl::service_sid()?;
    let shared = paths::shared_dir()?;
    std::fs::create_dir_all(&shared)?;
    acl::create_secured(
        &shared.join("logs"),
        &trenova_capture_svc::security::logs_sddl(&service_sid),
    )?;
    acl::create_secured(
        &paths::print_spool_dir()?,
        &trenova_capture_svc::security::spool_sddl(&service_sid),
    )
}

/// Stops and deletes the service. What people printed and have not yet
/// uploaded stays in the spool.
pub fn uninstall_service() -> io::Result<()> {
    let manager = manager(ServiceManagerAccess::CONNECT)?;
    let service = match manager.open_service(
        SERVICE_NAME,
        ServiceAccess::QUERY_STATUS | ServiceAccess::STOP | ServiceAccess::DELETE,
    ) {
        Ok(service) => service,
        Err(windows_service::Error::Winapi(os)) if os.raw_os_error() == Some(NO_SUCH_SERVICE) => {
            return Ok(());
        }
        Err(err) => return Err(service_error(err)),
    };
    if service.query_status().map_err(service_error)?.current_state != ServiceState::Stopped {
        service.stop().map_err(service_error)?;
        let deadline = Instant::now() + STOP_TIMEOUT;
        while service.query_status().map_err(service_error)?.current_state != ServiceState::Stopped
        {
            if Instant::now() >= deadline {
                return Err(io::Error::new(
                    io::ErrorKind::TimedOut,
                    "the service did not stop",
                ));
            }
            std::thread::sleep(Duration::from_millis(250));
        }
    }
    service.delete().map_err(service_error)
}

/// Windows `PowerShell` from the system directory, never from `PATH`.
fn powershell() -> io::Result<PathBuf> {
    let mut buffer = [0u16; 260];
    // SAFETY: the buffer's length is passed with it.
    let len = unsafe { GetSystemDirectoryW(Some(&mut buffer)) } as usize;
    if len == 0 || len >= buffer.len() {
        return Err(io::Error::last_os_error());
    }
    Ok(PathBuf::from(String::from_utf16_lossy(&buffer[..len]))
        .join("WindowsPowerShell")
        .join("v1.0")
        .join("powershell.exe"))
}

fn run_powershell(script: &str) -> io::Result<()> {
    let output = Command::new(powershell()?)
        .args([
            "-NoProfile",
            "-NonInteractive",
            "-ExecutionPolicy",
            "Bypass",
            "-Command",
            script,
        ])
        .output()?;
    if output.status.success() {
        return Ok(());
    }
    let message = String::from_utf8_lossy(&output.stderr);
    let kind = if message.contains("Access is denied") || message.contains("0x80070005") {
        io::ErrorKind::PermissionDenied
    } else {
        io::ErrorKind::Other
    };
    Err(io::Error::new(kind, message.trim().to_owned()))
}

/// Adds the Trenova printer on the IPP Class Driver, pointed at the
/// service. An existing Trenova printer is replaced, so a changed port takes
/// effect.
pub fn install_printer(port: u16) -> io::Result<()> {
    run_powershell(&format!(
        "$ErrorActionPreference = 'Stop'; \
         if (Get-Printer -Name '{PRINTER_NAME}' -ErrorAction SilentlyContinue) {{ Remove-Printer -Name '{PRINTER_NAME}' }}; \
         Add-Printer -Name '{PRINTER_NAME}' -IppURL '{}'",
        printer_url(port)
    ))
}

pub fn uninstall_printer() -> io::Result<()> {
    run_powershell(&format!(
        "$ErrorActionPreference = 'Stop'; \
         if (Get-Printer -Name '{PRINTER_NAME}' -ErrorAction SilentlyContinue) {{ Remove-Printer -Name '{PRINTER_NAME}' }}"
    ))
}
