//! Running under the Service Control Manager: started on demand with the
//! manifest address as the one start argument, and stopped when the update
//! has been tried.

use std::ffi::OsString;
use std::process::ExitCode;
use std::time::Duration;

use trenova_capture_update::{SERVICE_NAME, UpdaterError};
use windows_service::service::{
    ServiceControl, ServiceControlAccept, ServiceExitCode, ServiceState, ServiceStatus, ServiceType,
};
use windows_service::service_control_handler::{
    self, ServiceControlHandlerResult, ServiceStatusHandle,
};
use windows_service::{define_windows_service, service_dispatcher};

/// Reported when an update was asked for and did not happen. Nothing acts on
/// the code; the log says why.
const NOT_UPDATED: u32 = 1;

define_windows_service!(ffi_service_main, service_main);

pub fn dispatch() -> ExitCode {
    match service_dispatcher::start(SERVICE_NAME, ffi_service_main) {
        Ok(()) => ExitCode::SUCCESS,
        Err(err) => {
            tracing::error!(error = %err, "not started by the Service Control Manager");
            ExitCode::FAILURE
        }
    }
}

fn report(handle: ServiceStatusHandle, state: ServiceState, exit: ServiceExitCode) {
    let status = ServiceStatus {
        service_type: ServiceType::OWN_PROCESS,
        current_state: state,
        controls_accepted: ServiceControlAccept::empty(),
        exit_code: exit,
        checkpoint: 0,
        wait_hint: if state == ServiceState::Running {
            Duration::ZERO
        } else {
            Duration::from_secs(30)
        },
        process_id: None,
    };
    if let Err(err) = handle.set_service_status(status) {
        tracing::warn!(error = %err, ?state, "could not report the service status");
    }
}

/// The dispatcher macro hands the arguments over by value.
#[expect(clippy::needless_pass_by_value)]
fn service_main(arguments: Vec<OsString>) {
    serve(&arguments);
}

fn serve(arguments: &[OsString]) {
    let handle = match service_control_handler::register(SERVICE_NAME, |control| match control {
        ServiceControl::Interrogate => ServiceControlHandlerResult::NoError,
        _ => ServiceControlHandlerResult::NotImplemented,
    }) {
        Ok(handle) => handle,
        Err(err) => {
            tracing::error!(error = %err, "could not register with the Service Control Manager");
            return;
        }
    };
    report(handle, ServiceState::Running, ServiceExitCode::Win32(0));

    let outcome = match arguments.first().and_then(|a| a.to_str()) {
        Some(manifest) => super::update(manifest),
        None => Err(UpdaterError::Address),
    };
    let exit = match outcome {
        Ok(()) => ServiceExitCode::Win32(0),
        Err(UpdaterError::UpToDate { .. } | UpdaterError::NoRelease) => {
            tracing::info!("nothing to update");
            ServiceExitCode::Win32(0)
        }
        Err(err) => {
            tracing::error!(error = %err, "the update did not happen");
            ServiceExitCode::ServiceSpecific(NOT_UPDATED)
        }
    };
    report(handle, ServiceState::Stopped, exit);
}
