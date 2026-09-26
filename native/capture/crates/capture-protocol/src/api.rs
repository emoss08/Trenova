//! The companion's side of `/api/v1/capture/`.
//!
//! Every type here mirrors a Go type in `services/tms`, named in its doc
//! comment, and serializes to the same camelCase JSON. Types the server sends
//! tolerate what a newer server might add: unknown fields are ignored, an
//! unknown enum value reads as `Unknown`, and a Go nil slice (`null`) reads as
//! empty, so an older companion keeps working against a newer server.

use serde::{Deserialize, Deserializer, Serialize};

/// Where every capture route lives, below the server's origin.
pub const API_PREFIX: &str = "/api/v1/capture/";

/// The scanner's resolution for a page, in dots per inch.
pub const DPI_HEADER: &str = "X-Capture-Dpi";
/// A patch sheet the scanner recognised on a page.
pub const PATCH_CODE_HEADER: &str = "X-Capture-Patch-Code";
/// A barcode the scanner decoded on a page; sent once per barcode.
pub const BARCODE_HEADER: &str = "X-Capture-Barcode";

/// `capture.MaxPageBytes`: the largest single page the server takes.
pub const MAX_PAGE_BYTES: usize = 20 << 20;
/// `capture.MaxBatchPages`: the most pages one batch may hold.
pub const MAX_BATCH_PAGES: u32 = 1000;
/// `capturehandler.maxPrintJobBytes`: the largest print job the server takes.
pub const MAX_PRINT_JOB_BYTES: usize = 200 << 20;
/// `capture.maxClientKeyLength`.
pub const MAX_CLIENT_KEY_LENGTH: usize = 100;
/// `captureservice.validPatchCodes`: the patch types the server records.
/// Anything else a scanner reports is dropped before upload.
pub const PATCH_CODES: [&str; 6] = ["1", "2", "3", "4", "6", "T"];

/// Server-sent event names on `device/stream/`.
pub mod events {
    /// The stream is open; fetch open requests.
    pub const READY: &str = "ready";
    /// Proof of life, every heartbeat interval.
    pub const HEARTBEAT: &str = "heartbeat";
    /// The server lost the position; fetch open requests.
    pub const RESET: &str = "reset";
    /// The server is ending the stream; reconnect.
    pub const CLOSE: &str = "close";
    /// Something for this device to do; fetch open requests.
    pub const CAPTURE_REQUEST: &str = "capture.request";
    /// This device has been revoked; stop and sign out.
    pub const CAPTURE_REVOKED: &str = "capture.revoked";
}

/// A PULID as the server writes it (`cdev_01J…`). The companion never builds
/// one; it only carries them back.
#[derive(Clone, Debug, PartialEq, Eq, Hash, PartialOrd, Ord, Serialize, Deserialize)]
#[serde(transparent)]
pub struct Id(pub String);

impl Id {
    pub fn as_str(&self) -> &str {
        &self.0
    }
}

impl std::fmt::Display for Id {
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        f.write_str(&self.0)
    }
}

impl From<&str> for Id {
    fn from(value: &str) -> Self {
        Self(value.to_owned())
    }
}

/// Reads a Go nil slice or map (`null`) as the empty value.
fn null_as_default<'de, D, T>(deserializer: D) -> Result<T, D::Error>
where
    D: Deserializer<'de>,
    T: Default + Deserialize<'de>,
{
    Ok(Option::<T>::deserialize(deserializer)?.unwrap_or_default())
}

/// `capture.Architecture`. The agent is built for x64 only; the x86 scan
/// helper exists for 32-bit TWAIN drivers and is never paired itself.
#[derive(Clone, Copy, Debug, PartialEq, Eq, Serialize, Deserialize)]
pub enum Architecture {
    #[serde(rename = "x64")]
    X64,
    #[serde(other)]
    Unknown,
}

/// `capture.SourceProtocol`.
#[derive(Clone, Copy, Debug, PartialEq, Eq, Hash, Serialize, Deserialize)]
pub enum SourceProtocol {
    #[serde(rename = "TWAIN")]
    Twain,
    #[serde(rename = "WIA")]
    Wia,
    #[serde(other)]
    Unknown,
}

/// `capture.PixelType`.
#[derive(Clone, Copy, Debug, Default, PartialEq, Eq, Hash, Serialize, Deserialize)]
pub enum PixelType {
    #[default]
    BlackWhite,
    Grayscale,
    Color,
    #[serde(other)]
    Unknown,
}

/// `capture.SeparatorStrategy`.
#[derive(Clone, Copy, Debug, PartialEq, Eq, Serialize, Deserialize)]
pub enum SeparatorStrategy {
    PatchCode,
    CoverSheet,
    BlankPage,
    FixedPageCount,
    #[serde(other)]
    Unknown,
}

/// `capture.DeviceStatus`.
#[derive(Clone, Copy, Debug, Default, PartialEq, Eq, Serialize, Deserialize)]
pub enum DeviceStatus {
    #[default]
    Active,
    Revoked,
    #[serde(other)]
    Unknown,
}

/// `capture.ProfileStatus`.
#[derive(Clone, Copy, Debug, Default, PartialEq, Eq, Serialize, Deserialize)]
pub enum ProfileStatus {
    #[default]
    Active,
    Inactive,
    #[serde(other)]
    Unknown,
}

/// `capture.Source`: how a batch's pages came to be.
#[derive(Clone, Copy, Debug, PartialEq, Eq, Serialize, Deserialize)]
pub enum BatchSource {
    Scan,
    Print,
    #[serde(other)]
    Unknown,
}

/// `capture.RequestMode`.
#[derive(Clone, Copy, Debug, PartialEq, Eq, Serialize, Deserialize)]
pub enum RequestMode {
    Scan,
    Print,
    #[serde(other)]
    Unknown,
}

/// `capture.RequestStatus`.
#[derive(Clone, Copy, Debug, PartialEq, Eq, Serialize, Deserialize)]
pub enum RequestStatus {
    Pending,
    Delivered,
    InProgress,
    Completed,
    Canceled,
    Expired,
    Failed,
    #[serde(other)]
    Unknown,
}

/// `capture.RequestFailureCode`: why the device could not do what was asked.
#[derive(Clone, Copy, Debug, PartialEq, Eq, Serialize, Deserialize)]
#[serde(rename_all = "SCREAMING_SNAKE_CASE")]
pub enum RequestFailureCode {
    SourceUnavailable,
    SourceBusy,
    PaperJam,
    FeederEmpty,
    CanceledByUser,
    DriverError,
    UploadFailed,
    NotDelivered,
    Internal,
    #[serde(other)]
    Unknown,
}

/// `capture.BatchStatus`.
#[derive(Clone, Copy, Debug, PartialEq, Eq, Serialize, Deserialize)]
pub enum BatchStatus {
    Receiving,
    Sealed,
    Processing,
    Ready,
    PartiallyFiled,
    Filed,
    Discarded,
    Expired,
    Failed,
    #[serde(other)]
    Unknown,
}

/// `captureservice.StartPairingRequest`.
#[derive(Clone, Debug, PartialEq, Eq, Serialize)]
#[serde(rename_all = "camelCase")]
pub struct StartPairingRequest {
    pub machine_name: String,
    pub windows_user: String,
    pub agent_version: String,
    pub architecture: Architecture,
    pub os_version: String,
}

/// `captureservice.PairingGrant`, RFC 8628 §3.2.
#[derive(Clone, Debug, PartialEq, Eq, Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct PairingGrant {
    pub device_code: String,
    pub user_code: String,
    pub verification_uri: String,
    pub verification_uri_complete: String,
    /// Seconds until the grant lapses.
    pub expires_in: i64,
    /// Seconds to wait between polls.
    pub interval: i64,
}

/// `capturehandler.exchangePairingRequest`.
#[derive(Clone, Debug, PartialEq, Eq, Serialize)]
#[serde(rename_all = "camelCase")]
pub struct ExchangePairingRequest {
    pub device_code: String,
}

/// `captureservice.TokenPair`: a device's credential.
#[derive(Clone, PartialEq, Eq, Serialize, Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct TokenPair {
    #[serde(default)]
    pub token_type: String,
    pub access_token: String,
    /// Unix seconds.
    pub access_token_expires_at: i64,
    pub refresh_token: String,
    pub device_id: Id,
    #[serde(default)]
    pub device_name: String,
    pub user_id: Id,
    pub organization_id: Id,
    pub business_unit_id: Id,
}

impl std::fmt::Debug for TokenPair {
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        f.debug_struct("TokenPair")
            .field("access_token", &"<redacted>")
            .field("access_token_expires_at", &self.access_token_expires_at)
            .field("refresh_token", &"<redacted>")
            .field("device_id", &self.device_id)
            .field("user_id", &self.user_id)
            .finish_non_exhaustive()
    }
}

/// `captureservice.RefreshRequest`.
#[derive(Clone, PartialEq, Eq, Serialize)]
#[serde(rename_all = "camelCase")]
pub struct RefreshRequest {
    pub refresh_token: String,
    pub agent_version: String,
    pub os_version: String,
}

impl std::fmt::Debug for RefreshRequest {
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        f.debug_struct("RefreshRequest")
            .field("refresh_token", &"<redacted>")
            .field("agent_version", &self.agent_version)
            .finish_non_exhaustive()
    }
}

/// `captureservice.PairingErrorCode`: the RFC 8628 §3.5 answers the pairing
/// and refresh endpoints give as `{"error": "..."}` with status 400.
#[derive(Clone, Copy, Debug, PartialEq, Eq, Serialize, Deserialize)]
#[serde(rename_all = "snake_case")]
pub enum OAuthErrorCode {
    AuthorizationPending,
    SlowDown,
    AccessDenied,
    ExpiredToken,
    InvalidGrant,
    #[serde(other)]
    Unknown,
}

/// An RFC 8628 error body.
#[derive(Clone, Debug, PartialEq, Eq, Deserialize)]
pub struct OAuthError {
    pub error: OAuthErrorCode,
}

/// The `426 Upgrade Required` body: the organization requires a newer
/// companion than this one.
#[derive(Clone, Debug, PartialEq, Eq, Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct OutdatedAgent {
    #[serde(default)]
    pub message: String,
    pub minimum_version: String,
}

/// `helpers.ValidationError`.
#[derive(Clone, Debug, PartialEq, Eq, Deserialize)]
pub struct FieldError {
    #[serde(default)]
    pub field: String,
    #[serde(default)]
    pub message: String,
    #[serde(default)]
    pub code: String,
}

/// `helpers.ProblemDetail`, RFC 9457: every other error the API answers.
#[derive(Clone, Debug, Default, PartialEq, Eq, Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct ProblemDetail {
    #[serde(default, rename = "type")]
    pub kind: String,
    #[serde(default)]
    pub title: String,
    #[serde(default)]
    pub status: u16,
    #[serde(default)]
    pub detail: String,
    #[serde(default, deserialize_with = "null_as_default")]
    pub errors: Vec<FieldError>,
    #[serde(default)]
    pub trace_id: String,
    #[serde(default, deserialize_with = "null_as_default")]
    pub params: std::collections::HashMap<String, String>,
}

/// `captureservice.DisabledReasonParam` and `DisabledReason`.
pub const DISABLED_REASON_PARAM: &str = "reason";
pub const DISABLED_REASON: &str = "capture_disabled";

impl ProblemDetail {
    /// Whether the organization has turned capture off. Nothing can be sent
    /// until it is turned back on, and pages already captured are kept.
    pub fn is_capture_disabled(&self) -> bool {
        self.params
            .get(DISABLED_REASON_PARAM)
            .is_some_and(|reason| reason == DISABLED_REASON)
    }

    /// The most useful single line to show a person.
    pub fn summary(&self) -> &str {
        if let Some(first) = self.errors.first().filter(|e| !e.message.is_empty()) {
            return &first.message;
        }
        if !self.detail.is_empty() {
            return &self.detail;
        }
        &self.title
    }
}

/// `capture.SourceInfo`: one scanner the companion can reach.
#[derive(Clone, Debug, PartialEq, Eq, Serialize, Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct SourceInfo {
    pub name: String,
    pub protocol: SourceProtocol,
    /// 32 or 64: the TWAIN data source's own, which decides which scan helper
    /// runs it. WIA sources are always 64.
    pub bitness: u8,
    #[serde(default)]
    pub is_default: bool,
    #[serde(default)]
    pub duplex: bool,
    #[serde(default)]
    pub feeder: bool,
    #[serde(default)]
    pub patch_codes: bool,
    #[serde(default)]
    pub barcodes: bool,
    #[serde(default)]
    pub blank_discard: bool,
    #[serde(default, deserialize_with = "null_as_default")]
    pub resolutions: Vec<u32>,
    #[serde(default, deserialize_with = "null_as_default")]
    pub pixel_types: Vec<PixelType>,
}

/// `capturehandler.reportSourcesRequest`.
#[derive(Clone, Debug, PartialEq, Eq, Serialize)]
pub struct ReportSourcesRequest<'a> {
    pub sources: &'a [SourceInfo],
}

/// `capture.CaptureDevice`, the fields the companion reads.
#[derive(Clone, Debug, PartialEq, Eq, Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct CaptureDevice {
    pub id: Id,
    #[serde(default)]
    pub name: String,
    #[serde(default)]
    pub machine_name: String,
    #[serde(default)]
    pub windows_user: String,
    #[serde(default)]
    pub agent_version: String,
    #[serde(default)]
    pub status: DeviceStatus,
    #[serde(default, deserialize_with = "null_as_default")]
    pub sources: Vec<SourceInfo>,
    #[serde(default)]
    pub last_seen_at: Option<i64>,
}

/// `captureservice.DevicePerson`.
#[derive(Clone, Debug, PartialEq, Eq, Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct DevicePerson {
    pub id: Id,
    #[serde(default)]
    pub name: String,
    #[serde(default)]
    pub email_address: String,
}

/// `captureservice.DeviceOrganization`.
#[derive(Clone, Debug, PartialEq, Eq, Deserialize)]
pub struct DeviceOrganization {
    pub id: Id,
    #[serde(default)]
    pub name: String,
}

/// `captureservice.DeviceIdentity`: `GET device/`.
#[derive(Clone, Debug, PartialEq, Eq, Deserialize)]
pub struct DeviceIdentity {
    pub device: CaptureDevice,
    pub person: DevicePerson,
    pub organization: DeviceOrganization,
}

/// `capture.CaptureProfile`, the fields a scan needs.
#[derive(Clone, Debug, PartialEq, Eq, Serialize, Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct CaptureProfile {
    pub id: Id,
    pub name: String,
    #[serde(default)]
    pub status: ProfileStatus,
    #[serde(default)]
    pub is_default: bool,
    pub dpi: u32,
    pub pixel_type: PixelType,
    #[serde(default)]
    pub duplex: bool,
    #[serde(default)]
    pub use_feeder: bool,
    #[serde(default)]
    pub discard_blank_pages: bool,
    pub jpeg_quality: u8,
    #[serde(default)]
    pub show_driver_ui: bool,
    #[serde(default, deserialize_with = "null_as_default")]
    pub separator_strategies: Vec<SeparatorStrategy>,
    #[serde(default)]
    pub fixed_page_count: u32,
}

impl CaptureProfile {
    /// Whether the server would split this stack on patch sheets, which is
    /// when the scanner is asked to read them.
    pub fn wants_patch_codes(&self) -> bool {
        self.separator_strategies
            .contains(&SeparatorStrategy::PatchCode)
    }
}

/// `capture.CaptureRequest`: something a person asked this device to do.
#[derive(Clone, Debug, PartialEq, Eq, Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct CaptureRequest {
    pub id: Id,
    pub mode: RequestMode,
    pub status: RequestStatus,
    #[serde(default)]
    pub target_type: String,
    #[serde(default)]
    pub target_id: Option<Id>,
    #[serde(default)]
    pub document_type_id: Option<Id>,
    #[serde(default)]
    pub profile_id: Option<Id>,
    /// The scanner the person picked, by name; empty for the default.
    #[serde(default)]
    pub source_name: String,
    #[serde(default)]
    pub batch_id: Option<Id>,
    /// Unix seconds.
    #[serde(default)]
    pub expires_at: i64,
    /// The chosen profile, loaded with the request.
    #[serde(default)]
    pub profile: Option<CaptureProfile>,
}

/// `captureservice.RequestStatusReport`.
#[derive(Clone, Debug, PartialEq, Eq, Serialize)]
#[serde(rename_all = "camelCase")]
pub struct RequestStatusReport {
    pub status: RequestStatus,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub failure_code: Option<RequestFailureCode>,
    #[serde(skip_serializing_if = "String::is_empty")]
    pub failure_message: String,
}

impl RequestStatusReport {
    pub fn status(status: RequestStatus) -> Self {
        Self {
            status,
            failure_code: None,
            failure_message: String::new(),
        }
    }

    pub fn failed(code: RequestFailureCode, message: impl Into<String>) -> Self {
        Self {
            status: RequestStatus::Failed,
            failure_code: Some(code),
            failure_message: truncate(message.into(), 500),
        }
    }
}

/// `capture.Settings`: what the scan actually used, which is not always what
/// the profile asked for.
#[derive(Clone, Debug, Default, PartialEq, Eq, Serialize, Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct Settings {
    #[serde(default, skip_serializing_if = "Option::is_none")]
    pub protocol: Option<SourceProtocol>,
    #[serde(default, skip_serializing_if = "is_zero")]
    pub bitness: u8,
    #[serde(default, skip_serializing_if = "is_zero_u32")]
    pub dpi: u32,
    #[serde(default, skip_serializing_if = "Option::is_none")]
    pub pixel_type: Option<PixelType>,
    #[serde(default, skip_serializing_if = "is_false")]
    pub duplex: bool,
    #[serde(default, skip_serializing_if = "is_false")]
    pub feeder: bool,
    #[serde(default, skip_serializing_if = "is_false")]
    pub blank_discard: bool,
    #[serde(default, skip_serializing_if = "is_false")]
    pub show_driver_ui: bool,
    #[serde(default, skip_serializing_if = "String::is_empty")]
    pub driver_version: String,
    #[serde(default, skip_serializing_if = "String::is_empty")]
    pub application: String,
    /// Capabilities the source declined, by TWAIN name.
    #[serde(
        default,
        skip_serializing_if = "Vec::is_empty",
        deserialize_with = "null_as_default"
    )]
    pub refused: Vec<String>,
}

#[allow(clippy::trivially_copy_pass_by_ref)]
fn is_zero(value: &u8) -> bool {
    *value == 0
}

#[allow(clippy::trivially_copy_pass_by_ref)]
fn is_zero_u32(value: &u32) -> bool {
    *value == 0
}

#[allow(clippy::trivially_copy_pass_by_ref)]
fn is_false(value: &bool) -> bool {
    !*value
}

/// `captureservice.OpenBatchInput`.
#[derive(Clone, Debug, PartialEq, Eq, Serialize, Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct OpenBatchInput {
    /// The device's own name for the batch; opening twice with the same key
    /// returns the same batch.
    pub client_key: String,
    pub source: BatchSource,
    pub request_id: Option<Id>,
    pub profile_id: Option<Id>,
    #[serde(default)]
    pub source_name: String,
    #[serde(default)]
    pub job_name: String,
    #[serde(default)]
    pub settings: Settings,
}

/// `capture.CaptureBatch`, the fields the companion reads.
#[derive(Clone, Debug, PartialEq, Eq, Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct CaptureBatch {
    pub id: Id,
    pub status: BatchStatus,
    pub source: BatchSource,
    #[serde(default)]
    pub request_id: Option<Id>,
    #[serde(default)]
    pub target_type: String,
    #[serde(default)]
    pub target_id: Option<Id>,
    #[serde(default)]
    pub expected_page_count: u32,
    #[serde(default)]
    pub received_page_count: u32,
    #[serde(default)]
    pub failure_message: String,
}

/// `capture.CapturePage`, the fields the companion reads.
#[derive(Clone, Debug, PartialEq, Eq, Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct CapturePage {
    pub id: Id,
    pub batch_id: Id,
    pub sequence: u32,
    pub checksum_sha256: String,
    #[serde(default)]
    pub byte_size: u64,
}

/// `captureservice.SealBatchInput`.
#[derive(Clone, Debug, PartialEq, Eq, Serialize)]
#[serde(rename_all = "camelCase")]
pub struct SealBatchInput {
    pub page_count: u32,
    pub manifest_digest: String,
}

/// `captureservice.DeviceSignal`: the entity on a `capture.request` or
/// `capture.revoked` event.
#[derive(Clone, Debug, PartialEq, Eq, Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct DeviceSignal {
    pub device_id: Id,
    #[serde(default)]
    pub request_id: Option<Id>,
}

/// `services.RealtimeCloseEvent`.
#[derive(Clone, Debug, Default, PartialEq, Eq, Deserialize)]
pub struct CloseEvent {
    #[serde(default)]
    pub reason: String,
}

/// Cuts a message to the server's column width without splitting a character.
pub fn truncate(mut value: String, max_chars: usize) -> String {
    if let Some((index, _)) = value.char_indices().nth(max_chars) {
        value.truncate(index);
    }
    value
}

/// Keeps only the patch codes the server records.
pub fn known_patch_code(code: &str) -> Option<&'static str> {
    let code = code.trim();
    PATCH_CODES
        .iter()
        .copied()
        .find(|known| known.eq_ignore_ascii_case(code))
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn token_pair_reads_the_server_json_and_never_prints_secrets() {
        let raw = r#"{
            "tokenType": "Bearer",
            "accessToken": "tcd_at_abc",
            "accessTokenExpiresAt": 1790000000,
            "refreshToken": "tcd_rt_def",
            "deviceId": "cdev_01",
            "deviceName": "DISPATCH-07",
            "userId": "usr_01",
            "organizationId": "org_01",
            "businessUnitId": "bu_01"
        }"#;
        let pair: TokenPair = serde_json::from_str(raw).expect("token pair");
        assert_eq!(pair.access_token, "tcd_at_abc");
        assert_eq!(pair.device_id, Id::from("cdev_01"));
        let printed = format!("{pair:?}");
        assert!(!printed.contains("tcd_at_abc"));
        assert!(!printed.contains("tcd_rt_def"));
    }

    #[test]
    fn a_request_reads_go_nulls_and_an_embedded_profile() {
        let raw = r#"{
            "id": "creq_01", "businessUnitId": "bu_01", "organizationId": "org_01",
            "userId": "usr_01", "deviceId": "cdev_01", "mode": "Scan", "status": "Pending",
            "targetType": "shipment", "targetId": "shp_01", "documentTypeId": null,
            "profileId": "cprf_01", "sourceName": "", "batchId": null,
            "failureCode": "", "failureMessage": "", "expiresAt": 1790000120,
            "deliveredAt": null, "completedAt": null, "version": 0,
            "createdAt": 1790000000, "updatedAt": 1790000000,
            "profile": {
                "id": "cprf_01", "name": "Paperwork", "status": "Active", "isDefault": true,
                "dpi": 300, "pixelType": "BlackWhite", "duplex": true, "useFeeder": true,
                "discardBlankPages": true, "jpegQuality": 80, "showDriverUi": false,
                "separatorStrategies": ["PatchCode", "CoverSheet", "Handwriting"],
                "fixedPageCount": 0
            }
        }"#;
        let request: CaptureRequest = serde_json::from_str(raw).expect("request");
        assert_eq!(request.mode, RequestMode::Scan);
        assert_eq!(request.document_type_id, None);
        let profile = request.profile.expect("profile");
        assert!(profile.wants_patch_codes());
        assert_eq!(
            profile.separator_strategies,
            vec![
                SeparatorStrategy::PatchCode,
                SeparatorStrategy::CoverSheet,
                SeparatorStrategy::Unknown
            ]
        );
    }

    #[test]
    fn a_device_with_nil_sources_reads_as_none() {
        let raw = r#"{"device":{"id":"cdev_01","sources":null,"status":"Active","lastSeenAt":null},
            "person":{"id":"usr_01","name":"Jordan Doe","emailAddress":"j@x.test"},
            "organization":{"id":"org_01","name":"Acme Freight"}}"#;
        let identity: DeviceIdentity = serde_json::from_str(raw).expect("identity");
        assert!(identity.device.sources.is_empty());
        assert_eq!(identity.organization.name, "Acme Freight");
    }

    #[test]
    fn a_newer_server_status_does_not_break_an_older_companion() {
        let status: BatchStatus = serde_json::from_str(r#""Quarantined""#).expect("status");
        assert_eq!(status, BatchStatus::Unknown);
        let code: OAuthError = serde_json::from_str(r#"{"error":"slow_down"}"#).expect("error");
        assert_eq!(code.error, OAuthErrorCode::SlowDown);
    }

    #[test]
    fn settings_omit_what_was_not_negotiated_as_go_does() {
        let settings = Settings {
            protocol: Some(SourceProtocol::Twain),
            bitness: 32,
            dpi: 300,
            pixel_type: Some(PixelType::BlackWhite),
            duplex: true,
            refused: vec!["ICAP_AUTODISCARDBLANKPAGES".into()],
            ..Settings::default()
        };
        let json = serde_json::to_string(&settings).expect("json");
        assert_eq!(
            json,
            r#"{"protocol":"TWAIN","bitness":32,"dpi":300,"pixelType":"BlackWhite","duplex":true,"refused":["ICAP_AUTODISCARDBLANKPAGES"]}"#
        );
    }

    #[test]
    fn a_failure_report_carries_its_code_and_a_bounded_message() {
        let report = RequestStatusReport::failed(RequestFailureCode::PaperJam, "é".repeat(600));
        let json = serde_json::to_value(&report).expect("json");
        assert_eq!(json["status"], "Failed");
        assert_eq!(json["failureCode"], "PAPER_JAM");
        assert_eq!(
            json["failureMessage"].as_str().map(|m| m.chars().count()),
            Some(500)
        );
        let plain = serde_json::to_value(RequestStatusReport::status(RequestStatus::Delivered))
            .expect("json");
        assert_eq!(plain, serde_json::json!({"status": "Delivered"}));
    }

    #[test]
    fn a_problem_prefers_the_field_message() {
        let raw = r#"{"type":"validation","title":"Invalid","status":422,"detail":"One field",
            "errors":[{"field":"sequence","message":"Page sequence must be a number","code":"INVALID"}]}"#;
        let problem: ProblemDetail = serde_json::from_str(raw).expect("problem");
        assert_eq!(problem.summary(), "Page sequence must be a number");
        let bare: ProblemDetail =
            serde_json::from_str(r#"{"title":"Forbidden","status":403,"errors":null}"#)
                .expect("problem");
        assert_eq!(bare.summary(), "Forbidden");
        assert!(!bare.is_capture_disabled());
    }

    #[test]
    fn capture_turned_off_is_read_from_its_param_not_its_text() {
        let raw = r#"{"type":"business-rule-violation","title":"Business Rule Violation","status":422,
            "detail":"document capture is turned off for this organization",
            "params":{"reason":"capture_disabled"}}"#;
        let problem: ProblemDetail = serde_json::from_str(raw).expect("problem");
        assert!(problem.is_capture_disabled());
    }

    #[test]
    fn patch_codes_are_kept_only_when_the_server_records_them() {
        assert_eq!(known_patch_code(" t "), Some("T"));
        assert_eq!(known_patch_code("2"), Some("2"));
        assert_eq!(known_patch_code("5"), None);
    }
}
