//! Installing the updater service. It runs as the system, since it installs
//! software, and starts on demand: the tray agent, running as whoever is
//! signed in, is allowed to start it and nothing else, so the service's own
//! checks are all that stand between a request and an install.

use std::ffi::OsString;
use std::io;
use std::time::{Duration, Instant};

use capture_platform::acl::Descriptor;
use trenova_capture_update::{
    SERVICE_DESCRIPTION, SERVICE_DISPLAY_NAME, SERVICE_NAME, UpdaterError,
};
use windows::Win32::Security::DACL_SECURITY_INFORMATION;
use windows::Win32::System::Services::{SC_HANDLE, SetServiceObjectSecurity};
use windows_service::service::{
    ServiceAccess, ServiceErrorControl, ServiceInfo, ServiceStartType, ServiceState, ServiceType,
};
use windows_service::service_manager::{ServiceManager, ServiceManagerAccess};

/// The default DACL of a service, plus start and query for Authenticated
/// Users (`RP` is `SERVICE_START`).
const SERVICE_SDDL: &str = "D:(A;;CCLCSWRPWPDTLOCRRC;;;SY)(A;;CCDCLCSWRPWPDTLOCRSDRCWDWO;;;BA)\
    (A;;CCLCSWLOCRRC;;;IU)(A;;CCLCSWLOCRRC;;;SU)(A;;CCLCSWRPLOCRRC;;;AU)";
const STOP_TIMEOUT: Duration = Duration::from_secs(60);
/// `ERROR_SERVICE_DOES_NOT_EXIST`.
const NO_SUCH_SERVICE: i32 = 1060;

fn service_error(err: windows_service::Error) -> UpdaterError {
    match err {
        windows_service::Error::Winapi(os) => UpdaterError::Io(os),
        other => UpdaterError::Io(io::Error::other(other)),
    }
}

fn manager(access: ServiceManagerAccess) -> Result<ServiceManager, UpdaterError> {
    ServiceManager::local_computer(None::<&str>, access).map_err(service_error)
}

fn service_info() -> Result<ServiceInfo, UpdaterError> {
    Ok(ServiceInfo {
        name: SERVICE_NAME.into(),
        display_name: SERVICE_DISPLAY_NAME.into(),
        service_type: ServiceType::OWN_PROCESS,
        start_type: ServiceStartType::OnDemand,
        error_control: ServiceErrorControl::Normal,
        executable_path: std::env::current_exe()?,
        launch_arguments: vec![OsString::from("service")],
        dependencies: Vec::new(),
        account_name: None,
        account_password: None,
    })
}

/// Creates or updates the service and lets signed-in users start it.
pub fn install_service() -> Result<(), UpdaterError> {
    let manager = manager(ServiceManagerAccess::CONNECT | ServiceManagerAccess::CREATE_SERVICE)?;
    let info = service_info()?;
    let access = ServiceAccess::QUERY_STATUS
        | ServiceAccess::CHANGE_CONFIG
        | ServiceAccess::QUERY_CONFIG
        | ServiceAccess::WRITE_DAC;
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
    allow_users_to_start(&service)
}

/// Lets signed-in users start the service the MSI created. Windows Installer
/// cannot set a service's DACL itself, so the MSI runs this afterwards.
pub fn configure_service() -> Result<(), UpdaterError> {
    let manager = manager(ServiceManagerAccess::CONNECT)?;
    let service = manager
        .open_service(SERVICE_NAME, ServiceAccess::WRITE_DAC)
        .map_err(service_error)?;
    allow_users_to_start(&service)
}

fn allow_users_to_start(service: &windows_service::service::Service) -> Result<(), UpdaterError> {
    let descriptor = Descriptor::from_sddl(SERVICE_SDDL)?;
    // SAFETY: an open service handle with WRITE_DAC and a descriptor that
    // lives until after the call.
    unsafe {
        SetServiceObjectSecurity(
            SC_HANDLE(service.raw_handle().cast()),
            DACL_SECURITY_INFORMATION,
            descriptor.as_ptr(),
        )
    }
    .map_err(|err| UpdaterError::Io(io::Error::other(err)))
}

/// Stops and deletes the service.
pub fn uninstall_service() -> Result<(), UpdaterError> {
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
        let deadline = Instant::now() + STOP_TIMEOUT;
        while service.query_status().map_err(service_error)?.current_state != ServiceState::Stopped
        {
            if Instant::now() >= deadline {
                return Err(UpdaterError::Io(io::Error::new(
                    io::ErrorKind::TimedOut,
                    "the updater is still running",
                )));
            }
            std::thread::sleep(Duration::from_millis(500));
        }
    }
    service.delete().map_err(service_error)
}
