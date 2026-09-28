//! Whether the Trenova printer is set up on this computer.
//!
//! The installer adds the printer as the person installing, because Windows
//! refuses an IPP printer to the system account. A computer where that step
//! could not run (a deployment tool installing as the system account, or an
//! install without administrator rights) has the print service but no
//! printer; the tray finds that here and offers to add it.

use capture_protocol::handoff::{PRINT_SERVICE_NAME, PRINTER_NAME};

use crate::settings::machine_key_exists;

const SERVICES_KEY: &str = "SYSTEM\\CurrentControlSet\\Services";
const PRINTERS_KEY: &str = "SYSTEM\\CurrentControlSet\\Control\\Print\\Printers";

/// Whether the print service is installed on this computer.
pub fn print_service_installed() -> bool {
    machine_key_exists(&format!("{SERVICES_KEY}\\{PRINT_SERVICE_NAME}"))
}

/// Whether the Trenova printer exists.
pub fn printer_installed() -> bool {
    machine_key_exists(&format!("{PRINTERS_KEY}\\{PRINTER_NAME}"))
}

/// The print service is installed and its printer is not: the one case the
/// tray offers to fix. Without the service there is nothing to print to.
pub fn printer_missing() -> bool {
    print_service_installed() && !printer_installed()
}
