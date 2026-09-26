//! The Trenova Capture print service.
//!
//! Windows prints to the "Trenova" printer through its inbox IPP Class
//! Driver, which sends each job over IPP to this service on the loopback
//! interface. The service works out who printed it from the Windows print
//! queue ([`attribution`]), turns raster into PDF, and leaves the document in
//! that person's inbox ([`handler`], `capture_protocol::handoff`), where their
//! agent picks it up and uploads it. [`listener`] is the HTTP side, and
//! [`security`] the ACLs on the service's directories.
//!
//! Everything here runs anywhere and is tested on any host; the service
//! host, the spooler and session queries, the inbox ACLs and installation
//! are Windows code in the binary.

pub mod attribution;
pub mod handler;
pub mod listener;
pub mod security;

use capture_ipp::PrinterConfig;
use sha2::{Digest, Sha256};

/// The Windows service's name.
pub const SERVICE_NAME: &str = "TrenovaCaptureSvc";
pub const SERVICE_DISPLAY_NAME: &str = "Trenova Capture print service";
pub const SERVICE_DESCRIPTION: &str =
    "Receives what is printed to the Trenova printer and hands it to Trenova Capture for upload.";
/// The printer people choose in the print dialog.
pub const PRINTER_NAME: &str = "Trenova";
/// The loopback port when neither policy nor the installer chose one.
pub const DEFAULT_PORT: u16 = 8631;

/// The printer's URN, the same for the life of the Windows installation so
/// Windows keeps treating it as the same printer.
pub fn printer_uuid(machine_guid: &str) -> String {
    let digest = Sha256::digest(format!("trenova-capture-printer:{machine_guid}").as_bytes());
    let mut bytes = [0u8; 16];
    bytes.copy_from_slice(&digest[..16]);
    bytes[6] = (bytes[6] & 0x0F) | 0x50;
    bytes[8] = (bytes[8] & 0x3F) | 0x80;
    let hex = hex::encode(bytes);
    format!(
        "urn:uuid:{}-{}-{}-{}-{}",
        &hex[0..8],
        &hex[8..12],
        &hex[12..16],
        &hex[16..20],
        &hex[20..32]
    )
}

/// How the printer describes itself on `port`.
pub fn printer_config(port: u16, machine_guid: &str, max_document_bytes: usize) -> PrinterConfig {
    PrinterConfig {
        uri: format!("ipp://127.0.0.1:{port}{}", listener::IPP_PATH),
        uuid: printer_uuid(machine_guid),
        name: PRINTER_NAME.into(),
        info: "Sends what you print to Trenova".into(),
        location: "This computer".into(),
        make_and_model: "Trenova Capture".into(),
        max_document_bytes,
    }
}

/// The printer's IPP URL, as `Add-Printer -IppURL` takes it.
pub fn printer_url(port: u16) -> String {
    format!("http://127.0.0.1:{port}{}", listener::IPP_PATH)
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn the_printer_uuid_is_stable_and_well_formed() {
        let uuid = printer_uuid("4c4c4544-0033-3510-8057-b4c04f4e3732");
        assert_eq!(uuid, printer_uuid("4c4c4544-0033-3510-8057-b4c04f4e3732"));
        assert_ne!(uuid, printer_uuid("another machine"));
        let body = uuid.strip_prefix("urn:uuid:").expect("urn");
        let parts: Vec<&str> = body.split('-').collect();
        assert_eq!(
            parts.iter().map(|p| p.len()).collect::<Vec<_>>(),
            [8, 4, 4, 4, 12]
        );
        assert!(parts[2].starts_with('5'), "version 5 style");
        assert_eq!(printer_url(8631), "http://127.0.0.1:8631/ipp/print");
    }
}
