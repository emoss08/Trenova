//! The test scanner: scanners that exist only in this helper, for trying
//! every part of scanning on a computer without one.
//!
//! Each is listed as a TWAIN source with a feeder, duplex, patch codes,
//! barcodes and blank-page removal, and scans sample freight paperwork
//! through the same path a real scanner's pages take: encoded, pictured,
//! sent to the agent, spooled, looked over and uploaded. What each feeds is
//! fixed, so what should happen is known:
//!
//! - **Trenova Test Scanner**: a three-sheet bill of lading whose first
//!   sheet has its terms on the back, a Patch T sheet, and a two-sheet
//!   delivery receipt; the other backs are blank.
//! - **(paper jam)**: five sheets that jam after the third; continuing feeds
//!   the last two.
//! - **(double feed)**: four sheets that double-feed after the second;
//!   continuing feeds the rest.
//! - **(empty feeder)**: nothing to scan.
//!
//! Where a stopped test scanner left off is kept in a small file, so
//! "Continue scanning" picks up the next sheet as it would on a real one.

use std::path::{Path, PathBuf};
use std::sync::atomic::{AtomicBool, Ordering};
use std::time::Duration;

use capture_imaging::Resolution;
use capture_imaging::sample::{SampleKind, SamplePage, draw};
use capture_imaging::scan::{ScanEnd, ScanSettings, ScannedPage};
use capture_protocol::api::{PixelType, SourceInfo, SourceProtocol};
use capture_protocol::helper::ScanCondition;

/// Every test scanner's name starts with this.
pub const NAME: &str = "Trenova Test Scanner";
/// How long a sheet takes to feed, so pages arrive one by one as they do.
#[cfg(windows)]
const SHEET_TIME: Duration = Duration::from_millis(350);

struct Sheet {
    front: SamplePage,
    back: SamplePage,
}

struct Scenario {
    name: &'static str,
    sheets: Vec<Sheet>,
    /// The scanner stops after this many sheets, the first time through.
    stop: Option<(usize, ScanCondition)>,
}

fn page(kind: SampleKind, pro: &str, page: u32, of: u32) -> SamplePage {
    SamplePage {
        kind,
        pro: pro.to_owned(),
        page,
        of,
    }
}

fn document(kind: &SampleKind, pro: &str, sheets: u32, terms_on_first: bool) -> Vec<Sheet> {
    (1..=sheets)
        .map(|n| Sheet {
            front: page(kind.clone(), pro, n, sheets),
            back: if terms_on_first && n == 1 {
                page(SampleKind::Terms, pro, n, sheets)
            } else {
                page(SampleKind::Blank, pro, n, sheets)
            },
        })
        .collect()
}

fn scenarios() -> Vec<Scenario> {
    let mut mixed = document(&SampleKind::BillOfLading, "1042", 3, true);
    mixed.push(Sheet {
        front: page(SampleKind::PatchT, "1042", 1, 1),
        back: page(SampleKind::Blank, "1042", 1, 1),
    });
    mixed.extend(document(&SampleKind::DeliveryReceipt, "1043", 2, false));
    vec![
        Scenario {
            name: NAME,
            sheets: mixed,
            stop: None,
        },
        Scenario {
            name: "Trenova Test Scanner (paper jam)",
            sheets: document(&SampleKind::BillOfLading, "2001", 5, false),
            stop: Some((3, ScanCondition::PaperJam)),
        },
        Scenario {
            name: "Trenova Test Scanner (double feed)",
            sheets: document(&SampleKind::DeliveryReceipt, "3001", 4, false),
            stop: Some((2, ScanCondition::DoubleFeed)),
        },
        Scenario {
            name: "Trenova Test Scanner (empty feeder)",
            sheets: Vec::new(),
            stop: None,
        },
    ]
}

/// Whether `name` is one of the test scanners.
pub fn is_test_source(name: &str) -> bool {
    scenarios().iter().any(|s| s.name == name)
}

fn describe(name: &str, is_default: bool) -> SourceInfo {
    SourceInfo {
        name: name.to_owned(),
        protocol: SourceProtocol::Twain,
        bitness: 64,
        is_default,
        duplex: true,
        feeder: true,
        patch_codes: true,
        barcodes: true,
        blank_discard: true,
        resolutions: vec![100, 150, 200, 300],
        pixel_types: vec![
            PixelType::BlackWhite,
            PixelType::Grayscale,
            PixelType::Color,
        ],
    }
}

/// The test scanners as a source list shows them.
pub fn sources() -> Vec<SourceInfo> {
    scenarios()
        .iter()
        .map(|scenario| describe(scenario.name, false))
        .collect()
}

/// What a test scanner says it can do once opened.
#[cfg(windows)]
pub fn described(name: &str) -> SourceInfo {
    describe(name, false)
}

/// The settings a test scanner scans with: what was asked, within what it
/// can do.
#[cfg(windows)]
pub fn settings(want: &ScanSettings) -> capture_protocol::api::Settings {
    capture_protocol::api::Settings {
        protocol: Some(SourceProtocol::Twain),
        bitness: 64,
        dpi: want.dpi.clamp(100, 300),
        pixel_type: Some(want.pixel_type),
        duplex: want.duplex,
        feeder: want.use_feeder,
        blank_discard: want.discard_blank_pages,
        show_driver_ui: false,
        driver_version: env!("CARGO_PKG_VERSION").to_owned(),
        application: NAME.to_owned(),
        refused: Vec::new(),
    }
}

/// Where a stopped test scanner left off.
struct Position {
    path: PathBuf,
}

impl Position {
    fn for_scanner(dir: &Path, name: &str) -> Self {
        let file: String = name
            .chars()
            .map(|c| {
                if c.is_ascii_alphanumeric() {
                    c.to_ascii_lowercase()
                } else {
                    '-'
                }
            })
            .collect();
        Self {
            path: dir.join(format!("{file}.position")),
        }
    }

    fn read(&self) -> usize {
        std::fs::read_to_string(&self.path)
            .ok()
            .and_then(|text| text.trim().parse().ok())
            .unwrap_or(0)
    }

    fn write(&self, sheet: usize) {
        if let Some(dir) = self.path.parent()
            && let Err(err) = std::fs::create_dir_all(dir)
                .and_then(|()| std::fs::write(&self.path, sheet.to_string()))
        {
            tracing::warn!(error = %err, "could not keep the test scanner's place");
        }
    }

    fn clear(&self) {
        let _ = std::fs::remove_file(&self.path);
    }
}

/// Feeds a test scanner's sheets, handing each page on as a scanner would:
/// both sides when duplex, blank sides dropped when asked, the patch sheet
/// and barcodes reported when asked for. `pause` is how long a sheet takes.
pub fn scan(
    name: &str,
    want: &ScanSettings,
    state_dir: &Path,
    pause: Duration,
    cancel: &AtomicBool,
    deliver: &mut dyn FnMut(ScannedPage) -> Result<(), String>,
) -> Result<ScanEnd, String> {
    let scenario = scenarios()
        .into_iter()
        .find(|s| s.name == name)
        .ok_or_else(|| format!("{name} is not a test scanner"))?;
    let position = Position::for_scanner(state_dir, name);
    let start = position.read().min(scenario.sheets.len());
    if scenario.sheets.len() <= start {
        position.clear();
        return Ok(ScanEnd::Stopped {
            condition: ScanCondition::FeederEmpty,
            pages: 0,
        });
    }
    let dpi = want.dpi.clamp(100, 300);
    let mut pages = 0u32;
    for (at, sheet) in scenario.sheets.iter().enumerate().skip(start) {
        if let Some((after, condition)) = scenario.stop
            && at == after
            && start < after
        {
            position.write(at);
            return Ok(ScanEnd::Stopped { condition, pages });
        }
        if cancel.load(Ordering::Acquire) {
            position.clear();
            return Ok(ScanEnd::Canceled { pages });
        }
        std::thread::sleep(pause);
        let sides = if want.duplex {
            vec![&sheet.front, &sheet.back]
        } else {
            vec![&sheet.front]
        };
        for side in sides {
            if side.is_blank() && want.discard_blank_pages {
                continue;
            }
            let is_patch = side.kind == SampleKind::PatchT;
            deliver(ScannedPage {
                raster: draw(side, dpi, want.pixel_type),
                resolution: Resolution::square(dpi),
                patch_code: (is_patch && want.detect_patch_codes).then_some("T"),
                barcodes: if want.detect_barcodes {
                    side.barcode().into_iter().collect()
                } else {
                    Vec::new()
                },
            })?;
            pages += 1;
        }
        if !want.use_feeder {
            break;
        }
    }
    position.clear();
    Ok(ScanEnd::Finished { pages })
}

/// How long a sheet takes, outside tests.
#[cfg(windows)]
pub fn sheet_time() -> Duration {
    SHEET_TIME
}

#[cfg(test)]
mod tests {
    use super::*;

    fn want(duplex: bool, discard: bool) -> ScanSettings {
        ScanSettings {
            dpi: 100,
            pixel_type: PixelType::BlackWhite,
            duplex,
            use_feeder: true,
            discard_blank_pages: discard,
            show_ui: false,
            detect_patch_codes: true,
            detect_barcodes: true,
        }
    }

    fn run(name: &str, want: &ScanSettings, dir: &Path) -> (ScanEnd, Vec<ScannedPage>) {
        let mut pages = Vec::new();
        let end = scan(
            name,
            want,
            dir,
            Duration::ZERO,
            &AtomicBool::new(false),
            &mut |page| {
                pages.push(page);
                Ok(())
            },
        )
        .expect("scans");
        (end, pages)
    }

    #[test]
    fn the_mixed_feeder_reports_its_patch_sheet_and_barcodes() {
        let dir = tempfile::tempdir().expect("dir");
        let (end, pages) = run(NAME, &want(false, false), dir.path());
        assert_eq!(end, ScanEnd::Finished { pages: 6 });
        let patches: Vec<_> = pages.iter().map(|p| p.patch_code).collect();
        assert_eq!(patches, [None, None, None, Some("T"), None, None]);
        assert_eq!(pages[0].barcodes, ["PRO 1042"]);
        assert_eq!(pages[4].barcodes, ["PRO 1043"]);
        assert!(pages[1].barcodes.is_empty());
    }

    #[test]
    fn duplex_scans_the_backs_and_blank_removal_drops_the_blank_ones() {
        let dir = tempfile::tempdir().expect("dir");
        let (end, _) = run(NAME, &want(true, false), dir.path());
        assert_eq!(end, ScanEnd::Finished { pages: 12 });
        let (end, _) = run(NAME, &want(true, true), dir.path());
        assert_eq!(
            end,
            ScanEnd::Finished { pages: 7 },
            "six fronts and the terms"
        );
    }

    #[test]
    fn a_jam_stops_partway_and_continuing_feeds_the_rest() {
        let dir = tempfile::tempdir().expect("dir");
        let name = "Trenova Test Scanner (paper jam)";
        let (end, pages) = run(name, &want(false, false), dir.path());
        assert_eq!(
            end,
            ScanEnd::Stopped {
                condition: ScanCondition::PaperJam,
                pages: 3
            }
        );
        assert_eq!(pages.len(), 3);
        let (end, _) = run(name, &want(false, false), dir.path());
        assert_eq!(end, ScanEnd::Finished { pages: 2 });
        let (end, _) = run(name, &want(false, false), dir.path());
        assert!(
            matches!(
                end,
                ScanEnd::Stopped {
                    condition: ScanCondition::PaperJam,
                    pages: 3
                }
            ),
            "a finished feeder is loaded again"
        );
    }

    #[test]
    fn the_empty_feeder_scans_nothing_and_a_flatbed_scans_one_side() {
        let dir = tempfile::tempdir().expect("dir");
        let (end, pages) = run(
            "Trenova Test Scanner (empty feeder)",
            &want(false, false),
            dir.path(),
        );
        assert_eq!(
            end,
            ScanEnd::Stopped {
                condition: ScanCondition::FeederEmpty,
                pages: 0
            }
        );
        assert!(pages.is_empty());
        let flatbed = ScanSettings {
            use_feeder: false,
            ..want(true, false)
        };
        let (end, _) = run(NAME, &flatbed, dir.path());
        assert_eq!(end, ScanEnd::Finished { pages: 2 });
    }

    #[test]
    fn cancelling_stops_before_the_next_sheet() {
        let dir = tempfile::tempdir().expect("dir");
        let end = scan(
            NAME,
            &want(false, false),
            dir.path(),
            Duration::ZERO,
            &AtomicBool::new(true),
            &mut |_| Ok(()),
        )
        .expect("scans");
        assert_eq!(end, ScanEnd::Canceled { pages: 0 });
    }

    #[test]
    fn only_the_test_scanners_are_test_scanners() {
        assert!(is_test_source(NAME));
        assert!(is_test_source("Trenova Test Scanner (paper jam)"));
        assert!(!is_test_source("fi-8170"));
        assert_eq!(sources().len(), 4);
        assert!(
            sources()
                .iter()
                .all(|s| s.protocol == SourceProtocol::Twain)
        );
    }
}
