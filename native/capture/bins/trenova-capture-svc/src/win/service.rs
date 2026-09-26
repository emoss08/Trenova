//! Running under the Service Control Manager.

use std::ffi::OsString;
use std::process::ExitCode;
use std::time::Duration;

use tokio_util::sync::CancellationToken;
use trenova_capture_svc::SERVICE_NAME;
use windows_service::service::{
    ServiceControl, ServiceControlAccept, ServiceExitCode, ServiceState, ServiceStatus, ServiceType,
};
use windows_service::service_control_handler::{
    self, ServiceControlHandlerResult, ServiceStatusHandle,
};
use windows_service::{define_windows_service, service_dispatcher};

/// The exit code the service reports when the printer could not start or
/// stopped on an error.
const FAILED: u32 = 1;

define_windows_service!(ffi_service_main, service_main);

/// Hands this thread to the Service Control Manager, which calls
/// `service_main` on another and returns when the service has stopped.
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
    let controls = if state == ServiceState::Running {
        ServiceControlAccept::STOP | ServiceControlAccept::SHUTDOWN
    } else {
        ServiceControlAccept::empty()
    };
    let pending = matches!(
        state,
        ServiceState::StartPending | ServiceState::StopPending
    );
    let status = ServiceStatus {
        service_type: ServiceType::OWN_PROCESS,
        current_state: state,
        controls_accepted: controls,
        exit_code: exit,
        checkpoint: 0,
        wait_hint: if pending {
            Duration::from_secs(40)
        } else {
            Duration::ZERO
        },
        process_id: None,
    };
    if let Err(err) = handle.set_service_status(status) {
        tracing::warn!(error = %err, ?state, "could not report the service status");
    }
}

fn service_main(_arguments: Vec<OsString>) {
    let stop = CancellationToken::new();
    let control_stop = stop.clone();
    let handle = match service_control_handler::register(SERVICE_NAME, move |control| match control
    {
        ServiceControl::Stop | ServiceControl::Shutdown => {
            control_stop.cancel();
            ServiceControlHandlerResult::NoError
        }
        ServiceControl::Interrogate => ServiceControlHandlerResult::NoError,
        _ => ServiceControlHandlerResult::NotImplemented,
    }) {
        Ok(handle) => handle,
        Err(err) => {
            tracing::error!(error = %err, "could not register with the Service Control Manager");
            return;
        }
    };
    report(
        handle,
        ServiceState::StartPending,
        ServiceExitCode::Win32(0),
    );
    tracing::info!(
        version = env!("CARGO_PKG_VERSION"),
        "the print service is starting"
    );

    let outcome = tokio::runtime::Builder::new_multi_thread()
        .worker_threads(2)
        .enable_all()
        .build()
        .map_err(std::io::Error::other)
        .and_then(|runtime| {
            let running = stop.clone();
            let result = runtime.block_on(async move {
                let (listener, printer) = super::start(false).await?;
                report(handle, ServiceState::Running, ServiceExitCode::Win32(0));
                super::serve(listener, printer, running).await
            });
            report(handle, ServiceState::StopPending, ServiceExitCode::Win32(0));
            runtime.shutdown_timeout(Duration::from_secs(10));
            result
        });

    let exit = match outcome {
        Ok(()) => {
            tracing::info!("the print service stopped");
            ServiceExitCode::Win32(0)
        }
        Err(err) => {
            tracing::error!(error = %err, "the print service failed");
            ServiceExitCode::ServiceSpecific(FAILED)
        }
    };
    report(handle, ServiceState::Stopped, exit);
}
