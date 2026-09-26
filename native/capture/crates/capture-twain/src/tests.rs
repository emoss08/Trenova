//! The session against the scripted source in [`crate::fake`].

use std::collections::HashMap;
use std::sync::atomic::{AtomicBool, Ordering};

use capture_imaging::PixelFormat;
use capture_protocol::api::{PixelType, RequestFailureCode};
use capture_protocol::helper::ScanCondition;

use crate::TwainError;
use crate::capability::to_fix32;
#[allow(clippy::wildcard_imports)]
use crate::consts::*;
use crate::fake::{FakeCap, FakeDsm, FakePage, FakePump, FakeSource};
use crate::session::{AppIdentity, Manager, ScanEnd, ScanSettings, ScannedPage};

fn fix(dpi: f64) -> u32 {
    let fix = to_fix32(dpi);
    u32::from(u16::from_ne_bytes(fix.Whole.to_ne_bytes())) | (u32::from(fix.Frac) << 16)
}

fn app() -> AppIdentity {
    AppIdentity {
        version: "1.0.0".into(),
        major: 1,
        minor: 0,
    }
}

fn caps() -> HashMap<u16, FakeCap> {
    HashMap::from([
        (ICAP_XFERMECH, FakeCap::new(TWTY_UINT16, 0)),
        (ICAP_UNITS, FakeCap::new(TWTY_UINT16, 0)),
        (
            ICAP_PIXELTYPE,
            FakeCap::new(TWTY_UINT16, 0).allowing(&[0, 1, 2]),
        ),
        (ICAP_BITDEPTH, FakeCap::new(TWTY_UINT16, 1)),
        (ICAP_COMPRESSION, FakeCap::new(TWTY_UINT16, 0)),
        (
            ICAP_XRESOLUTION,
            FakeCap::new(TWTY_FIX32, fix(200.0)).allowing(&[fix(200.0), fix(300.0), fix(600.0)]),
        ),
        (
            ICAP_YRESOLUTION,
            FakeCap::new(TWTY_FIX32, fix(200.0)).allowing(&[fix(200.0), fix(300.0), fix(600.0)]),
        ),
        (ICAP_PIXELFLAVOR, FakeCap::new(TWTY_UINT16, 0)),
        (CAP_FEEDERENABLED, FakeCap::new(TWTY_BOOL, 0)),
        (CAP_FEEDERLOADED, FakeCap::new(TWTY_BOOL, 1).read_only()),
        (CAP_AUTOFEED, FakeCap::new(TWTY_BOOL, 0)),
        (CAP_DUPLEX, FakeCap::new(TWTY_UINT16, 1).read_only()),
        (CAP_DUPLEXENABLED, FakeCap::new(TWTY_BOOL, 0)),
        (CAP_XFERCOUNT, FakeCap::new(TWTY_INT16, 1)),
        (CAP_UICONTROLLABLE, FakeCap::new(TWTY_BOOL, 1).read_only()),
        (CAP_INDICATORS, FakeCap::new(TWTY_BOOL, 1)),
        (
            ICAP_AUTODISCARDBLANKPAGES,
            FakeCap::new(TWTY_INT32, 0xFFFF_FFFE),
        ),
        (ICAP_EXTIMAGEINFO, FakeCap::new(TWTY_BOOL, 0)),
        (ICAP_BARCODEDETECTIONENABLED, FakeCap::new(TWTY_BOOL, 0)),
    ])
}

/// A bilevel page whose rows are distinguishable: row `y` is `y` repeated.
fn bw_page(width: u32, height: usize, patch: Option<u32>, barcodes: &[&str]) -> FakePage {
    let row_bytes = width.div_ceil(8) as usize;
    FakePage {
        width,
        pixel_type: TWPT_BW,
        bits_per_pixel: 1,
        rows: (0..height)
            .map(|y| vec![u8::try_from(y % 251).expect("byte"); row_bytes])
            .collect(),
        patch,
        barcodes: barcodes.iter().map(ToString::to_string).collect(),
    }
}

fn fi8170(pages: Vec<FakePage>) -> FakeSource {
    FakeSource {
        name: "fi-8170".into(),
        caps: caps(),
        pages,
        strip_rows: 7,
        row_padding: 3,
        ..FakeSource::default()
    }
}

fn settings() -> ScanSettings {
    ScanSettings {
        dpi: 300,
        pixel_type: PixelType::BlackWhite,
        duplex: true,
        use_feeder: true,
        discard_blank_pages: true,
        show_ui: false,
        detect_patch_codes: true,
        detect_barcodes: true,
    }
}

fn scan(
    dsm: &FakeDsm,
    cancel_after: Option<u32>,
) -> (Result<ScanEnd, TwainError>, Vec<ScannedPage>) {
    let mut manager = Manager::open(dsm, &app(), core::ptr::null_mut()).expect("dsm");
    let mut source = manager.open_source("fi-8170").expect("source");
    let want = settings();
    source.negotiate(&want, 64).expect("negotiate");
    let cancel = AtomicBool::new(false);
    let mut pages = Vec::new();
    let result = source.scan(&want, &mut FakePump::default(), &cancel, &mut |page| {
        pages.push(page);
        if cancel_after.is_some_and(|n| u32::try_from(pages.len()).is_ok_and(|len| len >= n)) {
            cancel.store(true, Ordering::Release);
        }
        Ok::<(), String>(())
    });
    (result, pages)
}

impl crate::Dsm for &FakeDsm {
    unsafe fn entry(
        &self,
        origin: *mut crate::sys::TW_IDENTITY,
        dest: *mut crate::sys::TW_IDENTITY,
        dg: u32,
        dat: u16,
        msg: u16,
        data: *mut core::ffi::c_void,
    ) -> u16 {
        // SAFETY: forwarded unchanged.
        unsafe { (**self).entry(origin, dest, dg, dat, msg, data) }
    }
    fn use_entry_points(&self, entry_points: &crate::sys::TW_ENTRYPOINT) {
        (**self).use_entry_points(entry_points);
    }
    fn alloc(&self, size: u32) -> crate::sys::TW_HANDLE {
        (**self).alloc(size)
    }
    unsafe fn free(&self, handle: crate::sys::TW_HANDLE) {
        // SAFETY: forwarded unchanged.
        unsafe { (**self).free(handle) }
    }
    unsafe fn lock(&self, handle: crate::sys::TW_HANDLE) -> *mut core::ffi::c_void {
        // SAFETY: forwarded unchanged.
        unsafe { (**self).lock(handle) }
    }
    unsafe fn unlock(&self, handle: crate::sys::TW_HANDLE) {
        // SAFETY: forwarded unchanged.
        unsafe { (**self).unlock(handle) }
    }
}

/// The triplets that close everything, in the order they must come.
fn assert_torn_down(dsm: &FakeDsm) {
    let calls = dsm.calls();
    let position = |call| calls.iter().rposition(|c| *c == call);
    let disable = position((DAT_USERINTERFACE, MSG_DISABLEDS));
    let close_source = position((DAT_IDENTITY, MSG_CLOSEDS)).expect("source closed");
    let close_dsm = position((DAT_PARENT, MSG_CLOSEDSM)).expect("dsm closed");
    if let Some(disable) = disable {
        assert!(disable < close_source);
    }
    assert!(close_source < close_dsm);
    assert_eq!(
        dsm.live_allocations(),
        0,
        "every container and handle was freed"
    );
}

#[test]
fn sources_are_listed_with_the_default_marked() {
    let mut second = fi8170(vec![]);
    second.name = "WIA-DR-C225".into();
    let mut dsm = FakeDsm::new(vec![fi8170(vec![]), second]);
    dsm.default = 1;
    let manager = Manager::open(&dsm, &app(), core::ptr::null_mut()).expect("dsm");
    let sources = manager.sources().expect("sources");
    assert_eq!(sources.len(), 2);
    assert_eq!(sources[0].name, "fi-8170");
    assert!(!sources[0].is_default);
    assert!(sources[1].is_default);
    assert!(sources[0].twain2);
    assert_eq!(sources[0].version, "9.1.0");
}

#[test]
fn no_sources_installed_is_an_empty_list_not_an_error() {
    let dsm = FakeDsm::new(vec![]);
    let manager = Manager::open(&dsm, &app(), core::ptr::null_mut()).expect("dsm");
    assert!(manager.sources().expect("sources").is_empty());
}

#[test]
fn a_source_that_is_not_installed_reads_as_unavailable() {
    let dsm = FakeDsm::new(vec![fi8170(vec![])]);
    let mut manager = Manager::open(&dsm, &app(), core::ptr::null_mut()).expect("dsm");
    let err = manager.open_source("Canon DR-G2110").expect_err("missing");
    assert_eq!(err.failure_code(), RequestFailureCode::SourceUnavailable);
}

#[test]
fn describing_a_source_reports_what_it_can_do() {
    let dsm = FakeDsm::new(vec![fi8170(vec![])]);
    let mut manager = Manager::open(&dsm, &app(), core::ptr::null_mut()).expect("dsm");
    let source = manager.open_source("fi-8170").expect("source");
    let info = source.describe(true, 32);
    assert_eq!(info.bitness, 32);
    assert!(info.is_default && info.duplex && info.feeder && info.barcodes && info.blank_discard);
    assert!(
        !info.patch_codes,
        "patch detection is not among its capabilities"
    );
    assert_eq!(info.resolutions, vec![200, 300, 600]);
    assert_eq!(
        info.pixel_types,
        vec![
            PixelType::BlackWhite,
            PixelType::Grayscale,
            PixelType::Color
        ]
    );
    drop(source);
    drop(manager);
    assert_torn_down(&dsm);
}

#[test]
fn negotiation_applies_the_profile_and_records_what_the_source_refused() {
    let dsm = FakeDsm::new(vec![fi8170(vec![])]);
    let mut manager = Manager::open(&dsm, &app(), core::ptr::null_mut()).expect("dsm");
    let mut source = manager.open_source("fi-8170").expect("source");
    let mut want = settings();
    want.dpi = 400;
    let negotiated = source.negotiate(&want, 64).expect("negotiate");

    assert_eq!(
        negotiated.settings.dpi, 300,
        "the source took the nearest it supports"
    );
    assert_eq!(
        negotiated.settings.refused,
        vec!["ICAP_PATCHCODEDETECTIONENABLED"]
    );
    assert!(negotiated.settings.duplex && negotiated.settings.feeder);
    assert!(negotiated.source_discards_blanks);
    assert_eq!(negotiated.settings.driver_version, "9.1.0");
    assert_eq!(dsm.cap(0, ICAP_XFERMECH), Some(u32::from(TWSX_MEMORY)));
    assert_eq!(
        dsm.cap(0, CAP_XFERCOUNT),
        Some(0xFFFF),
        "-1: as many pages as the feeder holds"
    );
    assert_eq!(
        dsm.cap(0, ICAP_AUTODISCARDBLANKPAGES),
        Some(0xFFFF_FFFF),
        "TWBP_AUTO"
    );
    drop(source);
    drop(manager);
    assert_torn_down(&dsm);
}

#[test]
fn a_source_without_memory_transfer_cannot_be_used() {
    let mut source = fi8170(vec![]);
    source
        .caps
        .insert(ICAP_XFERMECH, FakeCap::new(TWTY_UINT16, 0).read_only());
    let dsm = FakeDsm::new(vec![source]);
    let mut manager = Manager::open(&dsm, &app(), core::ptr::null_mut()).expect("dsm");
    let mut source = manager.open_source("fi-8170").expect("source");
    let err = source.negotiate(&settings(), 64).expect_err("refused");
    assert_eq!(err.failure_code(), RequestFailureCode::DriverError);
    drop(source);
    drop(manager);
    assert_torn_down(&dsm);
}

#[test]
fn pages_are_assembled_from_padded_strips_of_unknown_length_with_their_markers() {
    let first = bw_page(61, 40, None, &["PRO-1042", "BOL 77"]);
    let second = bw_page(61, 23, Some(TWPCH_PATCHT), &[]);
    let mut source = fi8170(vec![first.clone(), second.clone()]);
    source.unknown_length = true;
    let dsm = FakeDsm::new(vec![source]);

    let (result, pages) = scan(&dsm, None);
    assert_eq!(result.expect("scan"), ScanEnd::Finished { pages: 2 });
    assert_eq!(pages.len(), 2);

    for (page, expected) in pages.iter().zip([&first, &second]) {
        assert_eq!(page.raster.width, 61);
        assert_eq!(page.raster.height as usize, expected.rows.len());
        assert_eq!(
            page.raster.format,
            PixelFormat::Bilevel {
                zero_is_black: true
            }
        );
        assert_eq!(
            page.raster.data,
            expected.rows.concat(),
            "padding was dropped"
        );
        assert_eq!(page.resolution.x, 300);
    }
    assert_eq!(pages[0].barcodes, vec!["PRO-1042", "BOL 77"]);
    assert_eq!(pages[0].patch_code, None);
    assert_eq!(pages[1].patch_code, Some("T"));
    assert!(pages[1].barcodes.is_empty());
    assert_torn_down(&dsm);
}

#[test]
fn a_jam_keeps_the_pages_scanned_before_it() {
    let pages = (0..4).map(|_| bw_page(16, 20, None, &[])).collect();
    let mut source = fi8170(pages);
    source.fail_page = Some((2, TWCC_PAPERJAM));
    let dsm = FakeDsm::new(vec![source]);

    let (result, pages) = scan(&dsm, None);
    assert_eq!(
        result.expect("scan"),
        ScanEnd::Stopped {
            condition: ScanCondition::PaperJam,
            pages: 2
        }
    );
    assert_eq!(pages.len(), 2);
    assert!(dsm.calls().contains(&(DAT_PENDINGXFERS, MSG_RESET)));
    assert_torn_down(&dsm);
}

#[test]
fn a_cancel_between_pages_stops_and_drops_what_is_pending() {
    let pages = (0..4).map(|_| bw_page(16, 10, None, &[])).collect();
    let dsm = FakeDsm::new(vec![fi8170(pages)]);

    let (result, pages) = scan(&dsm, Some(1));
    assert_eq!(result.expect("scan"), ScanEnd::Canceled { pages: 1 });
    assert_eq!(pages.len(), 1);
    assert!(dsm.calls().contains(&(DAT_PENDINGXFERS, MSG_RESET)));
    assert_torn_down(&dsm);
}

#[test]
fn an_empty_feeder_is_reported_before_the_source_is_enabled() {
    let mut source = fi8170(vec![bw_page(16, 10, None, &[])]);
    source
        .caps
        .insert(CAP_FEEDERLOADED, FakeCap::new(TWTY_BOOL, 0).read_only());
    let dsm = FakeDsm::new(vec![source]);

    let (result, pages) = scan(&dsm, None);
    assert_eq!(
        result.expect("scan"),
        ScanEnd::Stopped {
            condition: ScanCondition::FeederEmpty,
            pages: 0
        }
    );
    assert!(pages.is_empty());
    assert!(!dsm.calls().contains(&(DAT_USERINTERFACE, MSG_ENABLEDS)));
    assert_torn_down(&dsm);
}

#[test]
fn a_page_that_cannot_be_handed_on_ends_the_scan_cleanly() {
    let pages = (0..2).map(|_| bw_page(16, 10, None, &[])).collect();
    let dsm = FakeDsm::new(vec![fi8170(pages)]);
    let mut manager = Manager::open(&dsm, &app(), core::ptr::null_mut()).expect("dsm");
    let mut source = manager.open_source("fi-8170").expect("source");
    let want = settings();
    let err = source
        .scan(
            &want,
            &mut FakePump::default(),
            &AtomicBool::new(false),
            &mut |_| Err::<(), _>("the pipe closed"),
        )
        .expect_err("delivery failed");
    assert!(matches!(err, TwainError::Delivery(ref m) if m == "the pipe closed"));
    assert_eq!(err.failure_code(), RequestFailureCode::Internal);
    drop(source);
    drop(manager);
    assert_torn_down(&dsm);
}

#[test]
fn a_vanilla_grayscale_page_is_turned_the_right_way_round() {
    let page = FakePage {
        width: 4,
        pixel_type: TWPT_GRAY,
        bits_per_pixel: 8,
        rows: vec![vec![0, 64, 128, 255], vec![255, 255, 0, 0]],
        patch: None,
        barcodes: vec![],
    };
    let mut source = fi8170(vec![page]);
    source.caps.insert(
        ICAP_PIXELFLAVOR,
        FakeCap::new(TWTY_UINT16, u32::from(TWPF_VANILLA)).read_only(),
    );
    let dsm = FakeDsm::new(vec![source]);

    let (result, pages) = scan(&dsm, None);
    assert_eq!(result.expect("scan"), ScanEnd::Finished { pages: 1 });
    assert_eq!(pages[0].raster.format, PixelFormat::Gray8);
    assert_eq!(pages[0].raster.data, vec![255, 191, 127, 0, 0, 0, 255, 255]);
    assert_torn_down(&dsm);
}

#[test]
fn the_pump_cancel_before_the_source_is_ready_ends_with_nothing() {
    let dsm = FakeDsm::new(vec![fi8170(vec![bw_page(16, 10, None, &[])])]);
    let mut manager = Manager::open(&dsm, &app(), core::ptr::null_mut()).expect("dsm");
    let mut source = manager.open_source("fi-8170").expect("source");
    let result = source.scan(
        &settings(),
        &mut FakePump { cancel: true },
        &AtomicBool::new(false),
        &mut |_| Ok::<(), String>(()),
    );
    assert_eq!(result.expect("scan"), ScanEnd::Canceled { pages: 0 });
    drop(source);
    drop(manager);
    assert_torn_down(&dsm);
}
