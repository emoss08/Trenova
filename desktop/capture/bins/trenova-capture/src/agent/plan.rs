//! Turning a request, a profile and a scanner into a scan.

use capture_protocol::api::{
    BatchSource, CaptureProfile, CaptureRequest, Id, OpenBatchInput, PixelType, ProfileStatus,
    Settings, SourceInfo, SourceProtocol,
};
use capture_protocol::helper::{ScanJob, SourceRef};

/// The profile a scan uses with none chosen and none defined, the same
/// values the server fills in (`capture.CaptureProfile.ApplyDefaults` and the
/// column defaults): 300 DPI black and white, duplex from the feeder, blank
/// pages dropped, no separators beyond the patch and cover sheets that
/// always divide a stack.
pub fn default_profile() -> CaptureProfile {
    CaptureProfile {
        id: Id::from(""),
        name: "Paperwork".into(),
        status: ProfileStatus::Active,
        is_default: true,
        dpi: 300,
        pixel_type: PixelType::BlackWhite,
        duplex: true,
        use_feeder: true,
        discard_blank_pages: true,
        jpeg_quality: 80,
        show_driver_ui: false,
        separator_strategies: Vec::new(),
        fixed_page_count: 0,
    }
}

/// The profile for a scan: the one the request carries, else the one asked
/// for by ID, else the organization's default, else the built-in one.
pub fn choose_profile(
    request: Option<&CaptureProfile>,
    wanted: Option<&Id>,
    profiles: &[CaptureProfile],
) -> CaptureProfile {
    if let Some(profile) = request {
        return profile.clone();
    }
    let active = || {
        profiles
            .iter()
            .filter(|p| p.status != ProfileStatus::Inactive)
    };
    wanted
        .and_then(|id| active().find(|p| &p.id == id))
        .or_else(|| active().find(|p| p.is_default))
        .cloned()
        .unwrap_or_else(default_profile)
}

/// The scanner a request names. An empty name means the default scanner
/// (the one the computer marks default, else the first). When a scanner is
/// reachable over both TWAIN and WIA, TWAIN is used: it is the one that
/// reads patch codes and shows the driver's own dialog.
pub fn choose_source<'a>(name: &str, sources: &'a [SourceInfo]) -> Option<&'a SourceInfo> {
    let name = name.trim();
    if name.is_empty() {
        return sources
            .iter()
            .find(|s| s.is_default)
            .or_else(|| sources.iter().find(|s| s.protocol == SourceProtocol::Twain))
            .or_else(|| sources.first());
    }
    let named = || sources.iter().filter(move |s| s.name == name);
    named()
        .find(|s| s.protocol == SourceProtocol::Twain)
        .or_else(|| named().next())
}

/// The resolution nearest the profile's that the source offers, or the
/// profile's own when the source has not said what it offers.
fn resolution(profile: &CaptureProfile, source: &SourceInfo) -> u32 {
    source
        .resolutions
        .iter()
        .copied()
        .min_by_key(|r| r.abs_diff(profile.dpi))
        .unwrap_or(profile.dpi)
}

/// The pixel type to ask for: the profile's, or the nearest the source has.
fn pixel_type(profile: &CaptureProfile, source: &SourceInfo) -> PixelType {
    if source.pixel_types.is_empty() || source.pixel_types.contains(&profile.pixel_type) {
        return profile.pixel_type;
    }
    let order = match profile.pixel_type {
        PixelType::Color => [
            PixelType::Color,
            PixelType::Grayscale,
            PixelType::BlackWhite,
        ],
        PixelType::Grayscale => [
            PixelType::Grayscale,
            PixelType::Color,
            PixelType::BlackWhite,
        ],
        PixelType::BlackWhite | PixelType::Unknown => [
            PixelType::BlackWhite,
            PixelType::Grayscale,
            PixelType::Color,
        ],
    };
    order
        .into_iter()
        .find(|p| source.pixel_types.contains(p))
        .unwrap_or(profile.pixel_type)
}

/// Whether the source's capabilities are known yet. A TWAIN source is only
/// described once it has been opened for a scan.
fn described(source: &SourceInfo) -> bool {
    !source.resolutions.is_empty() || !source.pixel_types.is_empty()
}

/// The helper's job for a scan with this profile on this source.
pub fn job(source: &SourceInfo, profile: &CaptureProfile) -> ScanJob {
    let known = described(source);
    ScanJob {
        source: SourceRef {
            name: source.name.clone(),
            protocol: source.protocol,
        },
        dpi: resolution(profile, source),
        pixel_type: pixel_type(profile, source),
        duplex: profile.duplex && (!known || source.duplex),
        use_feeder: profile.use_feeder && (!known || source.feeder),
        discard_blank_pages: profile.discard_blank_pages,
        jpeg_quality: profile.jpeg_quality.clamp(30, 95),
        show_driver_ui: profile.show_driver_ui,
        // A patch sheet divides a stack whatever the profile says, so the
        // scanner reads them whenever it can.
        detect_patch_codes: source.protocol == SourceProtocol::Twain
            && (!known || source.patch_codes),
        detect_barcodes: false,
    }
}

/// What the tray and the server call a scan from this source.
pub fn label(source: &SourceInfo) -> String {
    source.name.clone()
}

/// The batch a scan opens on the server.
pub fn batch_input(
    key: &str,
    source: &SourceInfo,
    profile: &CaptureProfile,
    request: Option<&CaptureRequest>,
) -> OpenBatchInput {
    OpenBatchInput {
        client_key: key.to_owned(),
        source: BatchSource::Scan,
        request_id: request.map(|r| r.id.clone()),
        profile_id: (!profile.id.as_str().is_empty()).then(|| profile.id.clone()),
        source_name: source.name.clone(),
        job_name: String::new(),
        settings: Settings::default(),
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    fn source(name: &str, protocol: SourceProtocol) -> SourceInfo {
        SourceInfo {
            name: name.into(),
            protocol,
            bitness: 64,
            is_default: false,
            duplex: false,
            feeder: false,
            patch_codes: false,
            barcodes: false,
            blank_discard: false,
            resolutions: Vec::new(),
            pixel_types: Vec::new(),
        }
    }

    fn profile(id: &str, default: bool, status: ProfileStatus) -> CaptureProfile {
        CaptureProfile {
            id: Id::from(id),
            status,
            is_default: default,
            ..default_profile()
        }
    }

    #[test]
    fn the_request_profile_wins_then_the_named_one_then_the_default() {
        let profiles = vec![
            profile("cprf_a", false, ProfileStatus::Active),
            profile("cprf_b", true, ProfileStatus::Active),
            profile("cprf_off", false, ProfileStatus::Inactive),
        ];
        let carried = profile("cprf_req", false, ProfileStatus::Active);
        assert_eq!(
            choose_profile(Some(&carried), None, &profiles).id,
            Id::from("cprf_req")
        );
        assert_eq!(
            choose_profile(None, Some(&Id::from("cprf_a")), &profiles).id,
            Id::from("cprf_a")
        );
        assert_eq!(
            choose_profile(None, Some(&Id::from("cprf_off")), &profiles).id,
            Id::from("cprf_b")
        );
        assert_eq!(choose_profile(None, None, &[]).dpi, 300);
    }

    #[test]
    fn an_unnamed_scan_uses_the_default_scanner_and_a_named_one_prefers_twain() {
        let mut default = source("DS-530", SourceProtocol::Wia);
        default.is_default = true;
        let sources = vec![
            source("fi-8170", SourceProtocol::Wia),
            source("fi-8170", SourceProtocol::Twain),
            default,
        ];
        assert_eq!(
            choose_source("", &sources).map(|s| s.name.as_str()),
            Some("DS-530")
        );
        assert_eq!(
            choose_source("fi-8170", &sources).map(|s| s.protocol),
            Some(SourceProtocol::Twain)
        );
        assert_eq!(choose_source("DR-G2110", &sources), None);
        assert_eq!(choose_source("", &[]), None);
    }

    #[test]
    fn a_described_source_bends_the_profile_to_what_it_can_do() {
        let mut flatbed = source("Canon LiDE", SourceProtocol::Wia);
        flatbed.resolutions = vec![150, 600];
        flatbed.pixel_types = vec![PixelType::Grayscale, PixelType::Color];
        let job = job(&flatbed, &default_profile());
        assert_eq!(job.dpi, 150);
        assert_eq!(job.pixel_type, PixelType::Grayscale);
        assert!(!job.duplex && !job.use_feeder);
        assert!(!job.detect_patch_codes, "patch codes are a TWAIN feature");
    }

    #[test]
    fn an_undescribed_twain_source_is_asked_for_the_profile_as_it_is() {
        let job = job(
            &source("fi-8170", SourceProtocol::Twain),
            &default_profile(),
        );
        assert_eq!(job.dpi, 300);
        assert!(job.duplex && job.use_feeder && job.detect_patch_codes);
    }

    #[test]
    fn the_batch_names_its_request_and_only_a_real_profile() {
        let scanner = source("fi-8170", SourceProtocol::Twain);
        let input = batch_input("cap-1", &scanner, &default_profile(), None);
        assert_eq!(input.profile_id, None);
        assert_eq!(input.source, BatchSource::Scan);
        let named = profile("cprf_a", false, ProfileStatus::Active);
        assert_eq!(
            batch_input("cap-1", &scanner, &named, None).profile_id,
            Some(Id::from("cprf_a"))
        );
    }
}
