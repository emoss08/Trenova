//! Writes the window's page and views of it in several states, for looking
//! at in a browser: `TRENOVA_CAPTURE_PREVIEW=<dir> cargo test --test preview
//! -- --ignored`.

use std::time::{Duration, SystemTime};

use capture_client::spool::RefusedBatch;
use capture_protocol::api::{BatchSource, Id, PixelType, SourceInfo, SourceProtocol};
use capture_protocol::helper::ScanCondition;
use trenova_capture::agent::plan::default_profile;
use trenova_capture::state::{
    ActiveScan, Connection, Message, Notice, PausedBatch, RecentBatch, Severity, Snapshot,
    WaitingBatch,
};

fn source(name: &str, protocol: SourceProtocol, is_default: bool) -> SourceInfo {
    SourceInfo {
        name: name.into(),
        protocol,
        bitness: 64,
        is_default,
        duplex: true,
        feeder: true,
        patch_codes: true,
        barcodes: false,
        blank_discard: false,
        resolutions: Vec::new(),
        pixel_types: Vec::new(),
    }
}

fn millis_ago(minutes: u64) -> i64 {
    let at = SystemTime::now() - Duration::from_secs(minutes * 60);
    i64::try_from(
        at.duration_since(SystemTime::UNIX_EPOCH)
            .expect("clock")
            .as_millis(),
    )
    .expect("millis")
}

fn online() -> Snapshot {
    let mut colour = default_profile();
    colour.id = Id::from("cprf_colour");
    colour.name = "Colour documents".into();
    colour.is_default = false;
    colour.pixel_type = PixelType::Color;
    let mut default = default_profile();
    default.id = Id::from("cprf_default");
    Snapshot {
        server: Some("https://tms.acme-freight.com".into()),
        connection: Connection::Online,
        person: Some("Jordan Doe".into()),
        organization: Some("Acme Freight".into()),
        web_base: Some("https://tms.acme-freight.com".into()),
        sources: vec![
            source("fi-8170", SourceProtocol::Twain, true),
            source("HP LaserJet MFP M428", SourceProtocol::Wia, false),
        ],
        profiles: vec![colour, default],
        ..Snapshot::default()
    }
}

fn busy() -> Snapshot {
    let mut snapshot = online();
    snapshot.scan = Some(ActiveScan {
        key: "cap-now".into(),
        label: "fi-8170".into(),
        pages: 14,
        requested: true,
        stopping: false,
    });
    snapshot.paused = vec![PausedBatch {
        key: "cap-paused".into(),
        label: "fi-8170".into(),
        pages: 23,
        condition: ScanCondition::PaperJam,
    }];
    snapshot.refused = vec![
        RefusedBatch {
            key: "cap-refused".into(),
            label: "fi-8170".into(),
            source: BatchSource::Scan,
            pages: 6,
            reason: "The batch was ended before its pages arrived.".into(),
            created_at: millis_ago(30),
            refused_at: millis_ago(12),
            readable: true,
        },
        RefusedBatch {
            key: "cap-broken".into(),
            label: String::new(),
            source: BatchSource::Print,
            pages: 0,
            reason: "the manifest is unreadable: EOF while parsing".into(),
            created_at: millis_ago(4000),
            refused_at: millis_ago(4000),
            readable: false,
        },
    ];
    snapshot.waiting = vec![WaitingBatch {
        key: "cap-waiting".into(),
        label: "Invoice 88412.pdf".into(),
        source: BatchSource::Print,
        pages: 3,
        created_at: millis_ago(2),
        complete: true,
    }];
    snapshot.pages_waiting = 3;
    snapshot.recent.push_front(RecentBatch {
        label: "fi-8170".into(),
        pages: 2,
        link: "https://tms.acme-freight.com/intake?batch=cbat_1".into(),
        at: SystemTime::now() - Duration::from_secs(300),
        requested: false,
    });
    snapshot.recent.push_front(RecentBatch {
        label: "fi-8170".into(),
        pages: 11,
        link: "https://tms.acme-freight.com/intake?batch=cbat_2".into(),
        at: SystemTime::now() - Duration::from_secs(60),
        requested: true,
    });
    snapshot.messages.push_front(Message {
        notice: Notice {
            title: "A scan could not be sent".into(),
            body: "fi-8170: The batch was ended before its pages arrived. Its pages are kept on this computer; open Trenova Capture to send them again, save them, or discard them.".into(),
            severity: Severity::Error,
            link: None,
        },
        at: SystemTime::now() - Duration::from_secs(720),
    });
    snapshot.printer_missing = true;
    snapshot
}

#[test]
#[ignore = "writes preview files for looking at the page"]
fn write_preview() {
    let Ok(dir) = std::env::var("TRENOVA_CAPTURE_PREVIEW") else {
        return;
    };
    let dir = std::path::PathBuf::from(dir);
    std::fs::create_dir_all(&dir).expect("dir");
    std::fs::write(dir.join("page.html"), trenova_capture::page::html()).expect("page");
    let pairing = Snapshot {
        server: Some("https://tms.acme-freight.com".into()),
        connection: Connection::Pairing {
            code: "BCDF-GHJK".into(),
            url: "https://tms.acme-freight.com/capture/pair?code=BCDF-GHJK".into(),
        },
        ..Snapshot::default()
    };
    let mut offline = online();
    offline.connection = Connection::Offline {
        reason: "dns error".into(),
    };
    offline.waiting = busy().waiting;
    offline.pages_waiting = 3;
    let states = [
        ("first-run", Snapshot::default()),
        ("pairing", pairing),
        ("ready", online()),
        ("busy", busy()),
        ("offline", offline),
    ];
    for (name, snapshot) in states {
        let json = serde_json::to_string_pretty(&snapshot.view("1.0.0")).expect("json");
        std::fs::write(dir.join(format!("{name}.json")), json).expect("view");
    }
}
