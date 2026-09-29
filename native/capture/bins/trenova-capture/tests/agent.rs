//! The agent end to end: a mock server asks for a scan, a scripted scanner
//! produces pages, and the pages arrive and are sealed; a job the print
//! service left in the inbox is sent whole.

use std::collections::VecDeque;
use std::path::{Path, PathBuf};
use std::sync::atomic::{AtomicBool, Ordering};
use std::sync::{Arc, Mutex, PoisonError};
use std::time::{Duration, SystemTime, UNIX_EPOCH};

use capture_client::spool::{PagePictures, PictureSize};
use capture_client::{AgentInfo, Credential, MemoryStore, Protector, SecretStore, Server};
use capture_protocol::api::{Id, PixelType, Settings, SourceInfo, SourceProtocol, TokenPair};
use capture_protocol::handoff::Inbox;
use capture_protocol::helper::{PageMeta, ScanCondition, ScanJob};
use capture_protocol::page_checksum;
use capture_protocol::release::{
    Installer, PRODUCT, Release, encode_key_pair, public_key, sign, signing_key,
};
use serde_json::json;
use tokio::sync::mpsc;
use tokio_util::sync::CancellationToken;
use trenova_capture::agent::{
    self, Environment, Machine, PrinterCheck, ServerSetting, UpdateStarter,
};
use trenova_capture::scanners::{ScanOutcome, ScanRun, ScanUpdate, ScannerHost, SourcesFuture};
use trenova_capture::state::{
    Attention, Command, Connection, Notice, Pictures, PicturesRequest, PrinterAttempt, Shared,
    Snapshot, Ui, UpdateStatus,
};
use wiremock::matchers::{body_partial_json, method, path};
use wiremock::{Mock, MockServer, Request, Respond, ResponseTemplate};

#[derive(Default)]
struct TestUi {
    notices: Mutex<Vec<Notice>>,
    attention: Mutex<Vec<Attention>>,
    pictures: Mutex<Vec<Pictures>>,
}

impl Ui for TestUi {
    fn refresh(&self) {}
    fn notify(&self, notice: Notice) {
        self.notices
            .lock()
            .unwrap_or_else(PoisonError::into_inner)
            .push(notice);
    }
    fn attention(&self, attention: Attention) {
        self.attention
            .lock()
            .unwrap_or_else(PoisonError::into_inner)
            .push(attention);
    }
    fn pictures(&self, pictures: Pictures) {
        self.pictures
            .lock()
            .unwrap_or_else(PoisonError::into_inner)
            .push(pictures);
    }
}

impl TestUi {
    fn attention(&self) -> Vec<Attention> {
        self.attention
            .lock()
            .unwrap_or_else(PoisonError::into_inner)
            .clone()
    }

    fn notices(&self) -> Vec<Notice> {
        self.notices
            .lock()
            .unwrap_or_else(PoisonError::into_inner)
            .clone()
    }

    fn titles(&self) -> Vec<String> {
        self.notices
            .lock()
            .unwrap_or_else(PoisonError::into_inner)
            .iter()
            .map(|n| n.title.clone())
            .collect()
    }
}

struct Plain;

impl Protector for Plain {
    fn protect(&self, plain: &[u8]) -> std::io::Result<Vec<u8>> {
        Ok(plain.to_vec())
    }
    fn unprotect(&self, sealed: &[u8]) -> std::io::Result<Vec<u8>> {
        Ok(sealed.to_vec())
    }
}

/// Records what the agent asked the updater to install from.
/// A printer that is missing until a test adds it.
#[derive(Default)]
struct FakePrinter {
    missing: std::sync::atomic::AtomicBool,
}

impl PrinterCheck for FakePrinter {
    fn missing(&self) -> bool {
        self.missing.load(std::sync::atomic::Ordering::SeqCst)
    }
}

#[derive(Default)]
struct FakeUpdater {
    started: Mutex<Vec<String>>,
}

impl UpdateStarter for FakeUpdater {
    fn start(&self, manifest_url: &str) -> std::io::Result<()> {
        self.started
            .lock()
            .unwrap_or_else(PoisonError::into_inner)
            .push(manifest_url.to_owned());
        Ok(())
    }
}

const RELEASE_SEED: [u8; 32] = [5u8; 32];

fn release_key() -> capture_update::ed25519_dalek::VerifyingKey {
    let (_, public) = encode_key_pair(RELEASE_SEED);
    public_key(&public).expect("key")
}

/// Publishes a signed release on the mock server.
async fn publish(server: &MockServer, version: &str) {
    let (private, _) = encode_key_pair(RELEASE_SEED);
    let release = Release {
        product: PRODUCT.into(),
        version: version.into(),
        published_at: 1_780_000_000,
        minimum_windows_build: 19045,
        installer: Installer {
            file_name: format!("TrenovaCapture-{version}-x64.msi"),
            url: format!("https://releases.example.test/TrenovaCapture-{version}-x64.msi"),
            sha256: "ab".repeat(32),
            size: 1000,
        },
        notes: String::new(),
    };
    let signed = sign(&release, &signing_key(&private).expect("key")).expect("signs");
    Mock::given(method("GET"))
        .and(path("/api/v1/capture/releases/latest/"))
        .respond_with(ResponseTemplate::new(200).set_body_json(signed))
        .mount(server)
        .await;
}

struct FixedServer(String);

impl ServerSetting for FixedServer {
    fn load(&self) -> Option<String> {
        Some(self.0.clone())
    }
    fn save(&self, _: &str) -> std::io::Result<()> {
        Ok(())
    }
    fn locked(&self) -> bool {
        false
    }
}

/// One scripted scan: pages, then how it ends.
struct Script {
    pages: u32,
    end: ScanOutcome,
}

struct FakeScanner {
    scripts: Mutex<VecDeque<Script>>,
    jobs: Mutex<Vec<ScanJob>>,
    /// Keep scanning after the scripted pages until asked to stop, then
    /// finish with what was scanned, as a helper does.
    hold: AtomicBool,
}

fn fake_source() -> SourceInfo {
    SourceInfo {
        name: "Fake Scanner".into(),
        protocol: SourceProtocol::Twain,
        bitness: 64,
        is_default: true,
        duplex: false,
        feeder: false,
        patch_codes: false,
        barcodes: false,
        blank_discard: false,
        resolutions: Vec::new(),
        pixel_types: Vec::new(),
    }
}

impl ScannerHost for FakeScanner {
    fn enumerate(&self) -> SourcesFuture<'_> {
        Box::pin(async { vec![fake_source()] })
    }

    fn scan(&self, _source: &SourceInfo, job: ScanJob) -> ScanRun {
        self.jobs
            .lock()
            .unwrap_or_else(PoisonError::into_inner)
            .push(job);
        let script = self
            .scripts
            .lock()
            .unwrap_or_else(PoisonError::into_inner)
            .pop_front()
            .expect("a scripted scan");
        let (tx, updates) = mpsc::channel(8);
        let cancel = CancellationToken::new();
        let hold = self.hold.load(Ordering::SeqCst).then(|| cancel.clone());
        tokio::spawn(async move {
            let _ = tx
                .send(ScanUpdate::Started(Settings {
                    dpi: 300,
                    ..Settings::default()
                }))
                .await;
            for index in 1..=script.pages {
                let meta = PageMeta {
                    index,
                    width_px: 2550,
                    height_px: 3300,
                    dpi: 300,
                    pixel_type: PixelType::BlackWhite,
                    patch_code: None,
                    barcodes: Vec::new(),
                    preview: false,
                };
                let pdf = format!("%PDF-1.7 page {index} of {}", fastrand_like(index)).into_bytes();
                let pictures = Some(PagePictures {
                    thumb: format!("thumb {index}").into_bytes(),
                    view: format!("view {index}").into_bytes(),
                });
                let _ = tx
                    .send(ScanUpdate::Page {
                        meta,
                        pdf,
                        pictures,
                    })
                    .await;
            }
            if let Some(stop) = hold {
                stop.cancelled().await;
                let _ = tx.send(ScanUpdate::End(ScanOutcome::Finished)).await;
                return;
            }
            let _ = tx.send(ScanUpdate::End(script.end)).await;
        });
        ScanRun { updates, cancel }
    }
}

/// Distinct bytes per page and per scan, without a random source.
fn fastrand_like(index: u32) -> u128 {
    SystemTime::now()
        .duration_since(UNIX_EPOCH)
        .expect("clock")
        .as_nanos()
        + u128::from(index)
}

fn credential(server: &str) -> Credential {
    Credential {
        server: Server::parse(server).expect("server").as_str().to_owned(),
        web_base: "https://app.acme.test".into(),
        tokens: TokenPair {
            token_type: "Bearer".into(),
            access_token: "tcd_at_1".into(),
            access_token_expires_at: i64::try_from(
                SystemTime::now()
                    .duration_since(UNIX_EPOCH)
                    .expect("clock")
                    .as_secs(),
            )
            .expect("secs")
                + 900,
            refresh_token: "tcd_rt_1".into(),
            device_id: Id::from("cdev_1"),
            device_name: "DISPATCH-07".into(),
            user_id: Id::from("usr_1"),
            organization_id: Id::from("org_1"),
            business_unit_id: Id::from("bu_1"),
        },
    }
}

struct StorePage;

impl Respond for StorePage {
    fn respond(&self, request: &Request) -> ResponseTemplate {
        let sequence: u32 = request
            .url
            .path()
            .trim_end_matches('/')
            .rsplit('/')
            .next()
            .and_then(|s| s.parse().ok())
            .unwrap_or(0);
        ResponseTemplate::new(200).set_body_json(json!({
            "id": format!("cpg_{sequence}"), "batchId": "cbat_1", "sequence": sequence,
            "checksumSha256": page_checksum(&request.body), "byteSize": request.body.len()
        }))
    }
}

fn request_json(source: &str) -> serde_json::Value {
    json!({
        "id": "creq_1", "mode": "Scan", "status": "Pending", "targetType": "shipment",
        "targetId": "shp_1", "sourceName": source, "expiresAt": 1, "profile": null
    })
}

/// A server with a signed-in device and the requests it hands out once.
async fn server(requests: serde_json::Value) -> MockServer {
    let server = MockServer::start().await;
    Mock::given(method("GET"))
        .and(path("/api/v1/capture/device/"))
        .respond_with(ResponseTemplate::new(200).set_body_json(json!({
            "device": {"id": "cdev_1", "sources": null},
            "person": {"id": "usr_1", "name": "Jordan Doe"},
            "organization": {"id": "org_1", "name": "Acme Freight"}
        })))
        .mount(&server)
        .await;
    Mock::given(method("GET"))
        .and(path("/api/v1/capture/device/profiles/"))
        .respond_with(ResponseTemplate::new(200).set_body_json(json!([])))
        .mount(&server)
        .await;
    Mock::given(method("PUT"))
        .and(path("/api/v1/capture/device/sources/"))
        .respond_with(ResponseTemplate::new(200).set_body_json(json!({"id": "cdev_1"})))
        .mount(&server)
        .await;
    Mock::given(method("GET"))
        .and(path("/api/v1/capture/device/stream/"))
        .respond_with(
            ResponseTemplate::new(200)
                .set_body_raw("event: ready\ndata: {}\n\n", "text/event-stream")
                .set_delay(Duration::from_millis(50)),
        )
        .mount(&server)
        .await;
    Mock::given(method("GET"))
        .and(path("/api/v1/capture/device/requests/"))
        .respond_with(ResponseTemplate::new(200).set_body_json(requests))
        .up_to_n_times(1)
        .mount(&server)
        .await;
    Mock::given(method("GET"))
        .and(path("/api/v1/capture/device/requests/"))
        .respond_with(ResponseTemplate::new(200).set_body_json(json!([])))
        .mount(&server)
        .await;
    Mock::given(method("POST"))
        .and(path("/api/v1/capture/device/requests/creq_1/status/"))
        .respond_with(ResponseTemplate::new(200).set_body_json(request_json("")))
        .mount(&server)
        .await;
    Mock::given(method("POST"))
        .and(path("/api/v1/capture/device/batches/"))
        .respond_with(ResponseTemplate::new(200).set_body_json(json!({
            "id": "cbat_1", "status": "Receiving", "source": "Scan", "requestId": "creq_1"
        })))
        .mount(&server)
        .await;
    Mock::given(method("PUT"))
        .respond_with(StorePage)
        .mount(&server)
        .await;
    Mock::given(method("POST"))
        .and(path("/api/v1/capture/device/batches/cbat_1/seal/"))
        .respond_with(ResponseTemplate::new(200).set_body_json(json!({
            "id": "cbat_1", "status": "Sealed", "source": "Scan", "requestId": "creq_1"
        })))
        .mount(&server)
        .await;
    server
}

struct Running {
    ui: Arc<TestUi>,
    shared: Arc<Shared>,
    commands: mpsc::UnboundedSender<Command>,
    task: tokio::task::JoinHandle<()>,
    scanner: Arc<FakeScanner>,
    inbox: Inbox,
    updater: Arc<FakeUpdater>,
    printer: Arc<FakePrinter>,
    revealed: Arc<Mutex<Vec<PathBuf>>>,
    dir: tempfile::TempDir,
}

fn start(server: &MockServer, scripts: Vec<Script>) -> Running {
    start_with_printer(server, scripts, false)
}

fn start_with_printer(server: &MockServer, scripts: Vec<Script>, printer_missing: bool) -> Running {
    start_with(server, scripts, printer_missing, false)
}

/// Starts an agent whose scanner keeps scanning until it is stopped.
fn start_holding(server: &MockServer, scripts: Vec<Script>) -> Running {
    start_with(server, scripts, false, true)
}

fn start_with(
    server: &MockServer,
    scripts: Vec<Script>,
    printer_missing: bool,
    hold: bool,
) -> Running {
    let dir = tempfile::tempdir().expect("dir");
    let store: Arc<dyn SecretStore> = Arc::new(MemoryStore::with(credential(&server.uri())));
    let scanner = Arc::new(FakeScanner {
        scripts: Mutex::new(scripts.into()),
        jobs: Mutex::new(Vec::new()),
        hold: AtomicBool::new(hold),
    });
    let revealed = Arc::new(Mutex::new(Vec::new()));
    let reveal_into = Arc::clone(&revealed);
    let updater = Arc::new(FakeUpdater::default());
    let printer = Arc::new(FakePrinter::default());
    printer
        .missing
        .store(printer_missing, std::sync::atomic::Ordering::SeqCst);
    let inbox_dir = dir.path().join("inbox");
    std::fs::create_dir_all(&inbox_dir).expect("inbox");
    let env = Environment {
        spool_dir: dir.path().join("spool"),
        print_inbox: Some(inbox_dir.clone()),
        agent: AgentInfo {
            version: "1.0.0".into(),
            os_version: "Windows 10.0.22631".into(),
        },
        machine: Machine {
            name: "DISPATCH-07".into(),
            windows_user: "jdoe".into(),
        },
        scanners: Arc::clone(&scanner) as Arc<dyn ScannerHost>,
        secrets: Arc::new(move |_| Arc::clone(&store)),
        protector: Arc::new(Plain),
        server_setting: Arc::new(FixedServer(server.uri())),
        browser: Arc::new(|_| {}),
        reveal: Arc::new(move |path: &Path| {
            reveal_into
                .lock()
                .unwrap_or_else(PoisonError::into_inner)
                .push(path.to_path_buf());
        }),
        recheck_after: Duration::from_secs(600),
        updater: Arc::clone(&updater) as Arc<dyn UpdateStarter>,
        release_key: Some(release_key()),
        windows_build: 22631,
        machine_auto_update: true,
        printer: Arc::clone(&printer) as Arc<dyn PrinterCheck>,
        offline_notice_after: Duration::from_millis(200),
    };
    let ui = Arc::new(TestUi::default());
    let shared = Shared::new(Arc::clone(&ui) as Arc<dyn Ui>);
    let (commands, rx) = mpsc::unbounded_channel();
    let task_shared = Arc::clone(&shared);
    let task = tokio::spawn(async move {
        agent::run(env, task_shared, rx, CancellationToken::new())
            .await
            .expect("agent");
    });
    Running {
        ui,
        shared,
        commands,
        task,
        scanner,
        inbox: Inbox::new(inbox_dir),
        updater,
        printer,
        revealed,
        dir,
    }
}

async fn until(running: &Running, what: &str, done: impl Fn(&Snapshot, &[String]) -> bool) {
    let deadline = tokio::time::Instant::now() + Duration::from_secs(20);
    loop {
        if done(&running.shared.snapshot(), &running.ui.titles()) {
            return;
        }
        assert!(
            tokio::time::Instant::now() < deadline,
            "timed out waiting for {what}: {:?}",
            running.ui.titles()
        );
        tokio::time::sleep(Duration::from_millis(50)).await;
    }
}

async fn stop(running: Running) {
    running.commands.send(Command::Quit).expect("quit");
    running.task.await.expect("agent task");
}

#[tokio::test(flavor = "multi_thread", worker_threads = 2)]
async fn a_scan_asked_for_in_the_web_app_is_scanned_uploaded_and_sealed() {
    let server = server(json!([request_json("Fake Scanner")])).await;
    let running = start(
        &server,
        vec![Script {
            pages: 2,
            end: ScanOutcome::Finished,
        }],
    );

    until(&running, "the batch to be sent", |s, titles| {
        titles.iter().any(|t| t == "2 pages sent to Trenova") && s.pages_waiting == 0
    })
    .await;
    let snapshot = running.shared.snapshot();
    assert_eq!(snapshot.connection, Connection::Online);
    assert_eq!(snapshot.person.as_deref(), Some("Jordan Doe"));
    assert_eq!(
        snapshot.recent[0].link,
        "https://app.acme.test/intake?batch=cbat_1"
    );
    assert!(snapshot.recent[0].requested);

    let job = running
        .scanner
        .jobs
        .lock()
        .unwrap_or_else(PoisonError::into_inner)[0]
        .clone();
    assert_eq!(job.dpi, 300);
    assert!(job.detect_patch_codes);

    let requests = server.received_requests().await.expect("requests");
    assert!(
        requests
            .iter()
            .any(|r| r.url.path().ends_with("/requests/creq_1/status/")
                && serde_json::from_slice::<serde_json::Value>(&r.body).expect("json")["status"]
                    == "Delivered")
    );
    let opened = requests
        .iter()
        .find(|r| r.method.as_str() == "POST" && r.url.path() == "/api/v1/capture/device/batches/")
        .expect("opened");
    let opened: serde_json::Value = serde_json::from_slice(&opened.body).expect("json");
    assert_eq!(opened["requestId"], "creq_1");
    assert_eq!(opened["sourceName"], "Fake Scanner");
    assert_eq!(opened["settings"]["dpi"], 300);
    let sealed = requests
        .iter()
        .find(|r| r.url.path().ends_with("/seal/"))
        .expect("sealed");
    assert_eq!(
        serde_json::from_slice::<serde_json::Value>(&sealed.body).expect("json")["pageCount"],
        2
    );
    stop(running).await;
}

#[tokio::test(flavor = "multi_thread", worker_threads = 2)]
async fn a_jammed_scan_is_continued_into_the_same_batch() {
    let server = server(json!([request_json("")])).await;
    let running = start(
        &server,
        vec![
            Script {
                pages: 1,
                end: ScanOutcome::Stopped {
                    condition: ScanCondition::PaperJam,
                    message: "The scanner reported a paper jam.".into(),
                },
            },
            Script {
                pages: 2,
                end: ScanOutcome::Finished,
            },
        ],
    );

    until(&running, "the scan to pause", |s, _| !s.paused.is_empty()).await;
    let paused = running.shared.snapshot().paused[0].clone();
    assert_eq!(paused.pages, 1);
    assert_eq!(paused.condition, ScanCondition::PaperJam);
    assert!(
        server
            .received_requests()
            .await
            .expect("requests")
            .iter()
            .all(|r| !r.url.path().ends_with("/seal/")),
        "a paused batch is not sealed"
    );

    running
        .commands
        .send(Command::Continue(paused.key))
        .expect("continue");
    until(&running, "the batch to be sent", |s, titles| {
        s.paused.is_empty() && titles.iter().any(|t| t == "3 pages sent to Trenova")
    })
    .await;
    let requests = server.received_requests().await.expect("requests");
    let opens = requests
        .iter()
        .filter(|r| {
            r.method.as_str() == "POST" && r.url.path() == "/api/v1/capture/device/batches/"
        })
        .count();
    assert_eq!(opens, 1, "continuing adds to the batch already opened");
    stop(running).await;
}

#[tokio::test(flavor = "multi_thread", worker_threads = 2)]
async fn a_request_for_a_scanner_this_computer_lacks_is_reported_failed() {
    let server = server(json!([request_json("Canon DR-G2110")])).await;
    Mock::given(method("POST"))
        .and(path("/api/v1/capture/device/requests/creq_1/status/"))
        .and(body_partial_json(
            json!({"status": "Failed", "failureCode": "SOURCE_UNAVAILABLE"}),
        ))
        .respond_with(ResponseTemplate::new(200).set_body_json(request_json("")))
        .with_priority(1)
        .expect(1)
        .mount(&server)
        .await;
    let running = start(&server, Vec::new());
    until(&running, "the failure notice", |_, titles| {
        titles.iter().any(|t| t == "A scan could not start")
    })
    .await;
    assert!(
        running
            .scanner
            .jobs
            .lock()
            .unwrap_or_else(PoisonError::into_inner)
            .is_empty()
    );
    let deadline = tokio::time::Instant::now() + Duration::from_secs(10);
    while !server
        .received_requests()
        .await
        .expect("requests")
        .iter()
        .any(|r| r.url.path().ends_with("/requests/creq_1/status/"))
    {
        assert!(
            tokio::time::Instant::now() < deadline,
            "the failure was never reported"
        );
        tokio::time::sleep(Duration::from_millis(50)).await;
    }
    stop(running).await;
    server.verify().await;
}

#[tokio::test(flavor = "multi_thread", worker_threads = 2)]
async fn a_print_left_in_the_inbox_is_sent_whole_to_where_it_was_armed() {
    let server = server(json!([])).await;
    Mock::given(method("POST"))
        .and(path("/api/v1/capture/device/batches/"))
        .and(body_partial_json(
            json!({"source": "Print", "jobName": "Rate confirmation"}),
        ))
        .respond_with(ResponseTemplate::new(200).set_body_json(json!({
            "id": "cbat_p", "status": "Receiving", "source": "Print", "requestId": "creq_armed"
        })))
        .with_priority(1)
        .mount(&server)
        .await;
    Mock::given(method("PUT"))
        .and(path("/api/v1/capture/device/batches/cbat_p/print-job/"))
        .respond_with(ResponseTemplate::new(200).set_body_json(json!({
            "id": "cbat_p", "status": "Sealed", "source": "Print", "requestId": "creq_armed",
            "expectedPageCount": 3, "receivedPageCount": 3
        })))
        .with_priority(1)
        .expect(1)
        .mount(&server)
        .await;
    let running = start(&server, Vec::new());
    running
        .inbox
        .deliver("Rate confirmation", Some(3), b"%PDF-1.7 printed", &[])
        .expect("the service delivers");

    until(&running, "the print to be sent", |s, titles| {
        titles.iter().any(|t| t == "3 pages sent to Trenova") && s.pages_waiting == 0
    })
    .await;
    let snapshot = running.shared.snapshot();
    assert_eq!(snapshot.recent[0].label, "Rate confirmation");
    assert!(
        snapshot.recent[0].requested,
        "the armed destination took it"
    );
    assert!(running.inbox.waiting().expect("lists").is_empty());

    let requests = server.received_requests().await.expect("requests");
    let sent = requests
        .iter()
        .find(|r| r.url.path().ends_with("/print-job/"))
        .expect("sent");
    assert_eq!(sent.body, b"%PDF-1.7 printed");
    assert!(requests.iter().all(|r| !r.url.path().ends_with("/seal/")));
    stop(running).await;
}

#[tokio::test(flavor = "multi_thread", worker_threads = 2)]
async fn a_newer_release_is_installed_through_the_updater_service() {
    let server = server(json!([])).await;
    publish(&server, "9.9.9").await;
    let running = start(&server, Vec::new());

    until(&running, "the updater to be started", |s, titles| {
        titles
            .iter()
            .any(|t| t == "Updating Trenova Capture to 9.9.9")
            && s.update
                .as_ref()
                .is_some_and(|u| u.status == UpdateStatus::Installing)
    })
    .await;
    assert_eq!(
        *running
            .updater
            .started
            .lock()
            .unwrap_or_else(PoisonError::into_inner),
        [format!("{}/api/v1/capture/releases/latest/", server.uri())],
        "the updater is pointed at the server's manifest, once"
    );
    stop(running).await;
}

#[tokio::test(flavor = "multi_thread", worker_threads = 2)]
async fn where_the_organization_installs_updates_itself_the_release_is_only_announced() {
    let server = server(json!([])).await;
    publish(&server, "9.9.9").await;
    Mock::given(method("GET"))
        .and(path("/api/v1/capture/device/"))
        .respond_with(
            ResponseTemplate::new(200)
                .set_body_json(json!({
                    "device": {"id": "cdev_1", "sources": null},
                    "person": {"id": "usr_1", "name": "Jordan Doe"},
                    "organization": {"id": "org_1", "name": "Acme Freight"},
                    "updates": {"minimumVersion": "", "allowAutoUpdate": false}
                }))
                .set_delay(Duration::from_millis(500)),
        )
        .with_priority(1)
        .mount(&server)
        .await;
    let running = start(&server, Vec::new());

    until(&running, "the release to be announced", |s, titles| {
        titles
            .iter()
            .any(|t| t == "Trenova Capture 9.9.9 is available")
            && s.update
                .as_ref()
                .is_some_and(|u| u.status == UpdateStatus::AskAdministrator)
    })
    .await;
    assert!(
        running
            .updater
            .started
            .lock()
            .unwrap_or_else(PoisonError::into_inner)
            .is_empty()
    );
    let menu = running.shared.snapshot().menu();
    assert!(
        format!("{menu:?}").contains("ask your administrator"),
        "{menu:?}"
    );
    stop(running).await;
}

#[tokio::test(flavor = "multi_thread", worker_threads = 2)]
async fn a_missing_printer_is_offered_and_its_outcome_is_told() {
    let server = server(json!([])).await;
    let running = start_with_printer(&server, Vec::new(), true);

    until(&running, "the printer to be offered", |s, _| {
        s.printer_missing && format!("{:?}", s.menu()).contains("Add the Trenova printer")
    })
    .await;

    running
        .commands
        .send(Command::PrinterSetUp(PrinterAttempt::Declined))
        .expect("send");
    running
        .commands
        .send(Command::PrinterSetUp(PrinterAttempt::Failed))
        .expect("send");
    until(&running, "the failure to be told", |s, titles| {
        s.printer_missing && titles == ["The Trenova printer could not be added"]
    })
    .await;

    running
        .printer
        .missing
        .store(false, std::sync::atomic::Ordering::SeqCst);
    running
        .commands
        .send(Command::PrinterSetUp(PrinterAttempt::Added))
        .expect("send");
    until(&running, "the printer to be ready", |s, titles| {
        !s.printer_missing
            && titles.last().map(String::as_str) == Some("The Trenova printer is ready")
            && !format!("{:?}", s.menu()).contains("Add the Trenova printer")
    })
    .await;
    stop(running).await;
}

#[tokio::test(flavor = "multi_thread", worker_threads = 2)]
async fn a_lost_connection_is_told_once_and_its_return_is_routine_news() {
    let server = server(json!([])).await;
    Mock::given(method("GET"))
        .and(path("/api/v1/capture/device/stream/"))
        .respond_with(ResponseTemplate::new(503).insert_header("Retry-After", "1"))
        .with_priority(1)
        .up_to_n_times(2)
        .expect(2)
        .mount(&server)
        .await;
    let running = start(&server, Vec::new());

    until(&running, "the connection to come back", |s, titles| {
        s.connection == Connection::Online
            && titles.contains(&"Connected to Trenova again".to_owned())
    })
    .await;
    let notices = running.ui.notices();
    let offline: Vec<_> = notices
        .iter()
        .filter(|n| n.title == "Trenova Capture is offline")
        .collect();
    assert_eq!(offline.len(), 1, "told once per outage: {notices:?}");
    assert!(!offline[0].routine, "a lost connection is always told");
    let back = notices
        .iter()
        .find(|n| n.title == "Connected to Trenova again")
        .expect("back");
    assert!(back.routine);
    assert!(
        running.shared.snapshot().printing,
        "the print inbox means printing is offered"
    );
    stop(running).await;
}

#[tokio::test(flavor = "multi_thread", worker_threads = 2)]
async fn a_scan_held_for_review_is_turned_trimmed_and_sent_only_when_released() {
    let server = server(json!([])).await;
    let running = start(
        &server,
        vec![Script {
            pages: 3,
            end: ScanOutcome::Finished,
        }],
    );
    let key = held_scan(&running, 3).await;
    assert!(
        running
            .ui
            .titles()
            .iter()
            .any(|t| t == "Look over the scan before it is sent")
    );
    assert!(running.ui.attention().contains(&Attention::Review));
    let opened = |requests: &[Request]| {
        requests
            .iter()
            .filter(|r| {
                r.method.as_str() == "POST" && r.url.path() == "/api/v1/capture/device/batches/"
            })
            .count()
    };
    assert_eq!(
        opened(&server.received_requests().await.unwrap_or_default()),
        0,
        "nothing held leaves the computer"
    );

    running
        .commands
        .send(Command::RotatePage {
            key: key.clone(),
            page: 2,
            degrees: 90,
        })
        .expect("turn");
    running
        .commands
        .send(Command::DeletePage {
            key: key.clone(),
            page: 1,
        })
        .expect("delete");
    until(&running, "the first page to be gone", |s, _| {
        s.waiting
            .iter()
            .any(|b| b.key == key && b.pictures.len() == 2 && b.pictures[0].rotation == 90)
    })
    .await;

    running
        .commands
        .send(Command::Pictures(PicturesRequest {
            key: key.clone(),
            size: PictureSize::Thumb,
            pages: vec![1, 2, 3],
        }))
        .expect("pictures");
    let pictures = next_pictures(&running).await;
    assert_eq!(
        pictures.pictures,
        [(1, b"thumb 2".to_vec()), (2, b"thumb 3".to_vec())],
        "the pages moved up with their pictures; a page that is not there is left out"
    );

    running
        .commands
        .send(Command::SendHeld(key.clone()))
        .expect("send");
    until(&running, "the held scan to be sent", |_, titles| {
        titles.iter().any(|t| t == "2 pages sent to Trenova")
    })
    .await;
    let requests = server.received_requests().await.unwrap_or_default();
    let turned = requests
        .iter()
        .find(|r| r.method.as_str() == "PUT" && r.url.path().ends_with("/pages/1/"))
        .expect("page 1 sent");
    assert_eq!(
        turned
            .headers
            .get("x-capture-rotation")
            .and_then(|v| v.to_str().ok()),
        Some("90"),
        "the turn travels with the page"
    );
    stop(running).await;
}

#[tokio::test(flavor = "multi_thread", worker_threads = 2)]
async fn a_held_scan_can_be_discarded() {
    let server = server(json!([])).await;
    let running = start(
        &server,
        vec![Script {
            pages: 1,
            end: ScanOutcome::Finished,
        }],
    );
    let key = held_scan(&running, 1).await;
    running
        .commands
        .send(Command::DiscardHeld(key))
        .expect("discard");
    until(&running, "nothing to be waiting", |s, _| {
        s.waiting.is_empty()
    })
    .await;
    stop(running).await;
}

/// Turns review on, scans, and waits for the scan to be held with the
/// pictures of its `pages`; returns its key.
async fn held_scan(running: &Running, pages: usize) -> String {
    ready(running).await;
    running.shared.update(|s| s.review_before_sending = true);
    running.commands.send(scan_to_intake()).expect("scan");
    until(running, "the scan to be held", |s, _| {
        s.scan.is_none()
            && s.waiting
                .iter()
                .any(|b| b.held && b.complete && b.pictures.len() == pages)
    })
    .await;
    running.shared.snapshot().waiting[0].key.clone()
}

async fn next_pictures(running: &Running) -> Pictures {
    let deadline = tokio::time::Instant::now() + Duration::from_secs(10);
    loop {
        if let Some(pictures) = running
            .ui
            .pictures
            .lock()
            .unwrap_or_else(PoisonError::into_inner)
            .pop()
        {
            return pictures;
        }
        assert!(tokio::time::Instant::now() < deadline, "no pictures");
        tokio::time::sleep(Duration::from_millis(50)).await;
    }
}

fn scan_to_intake() -> Command {
    Command::Scan {
        source: "Fake Scanner".into(),
        protocol: SourceProtocol::Twain,
        profile: None,
    }
}

/// Waits until the agent is online with its scanner listed.
async fn ready(running: &Running) {
    until(running, "the agent to be ready", |s, _| {
        s.connection == Connection::Online && !s.sources.is_empty()
    })
    .await;
}

/// A server that refuses the first batch opened on it, as it would one it
/// cannot take, and accepts the rest.
async fn refusing_server() -> MockServer {
    let server = server(json!([])).await;
    Mock::given(method("POST"))
        .and(path("/api/v1/capture/device/batches/"))
        .respond_with(ResponseTemplate::new(422).set_body_json(json!({
            "type": "https://api.trenova.test/problems/validation",
            "title": "Invalid", "status": 422,
            "detail": "The scanner name is not valid"
        })))
        .with_priority(1)
        .up_to_n_times(1)
        .mount(&server)
        .await;
    server
}

#[tokio::test(flavor = "multi_thread", worker_threads = 2)]
async fn a_scan_the_person_stops_sends_what_was_scanned() {
    let server = server(json!([request_json("Fake Scanner")])).await;
    let running = start_holding(
        &server,
        vec![Script {
            pages: 2,
            end: ScanOutcome::Finished,
        }],
    );

    until(&running, "two pages scanned", |s, _| {
        s.scan.as_ref().is_some_and(|scan| scan.pages == 2)
    })
    .await;
    let scan = running.shared.snapshot().scan.expect("scanning");
    assert!(scan.requested);
    assert!(!scan.stopping);
    assert!(running.ui.attention().contains(&Attention::ScanStarted));

    running.commands.send(Command::StopScan).expect("stop");
    until(&running, "the scan to be sent", |s, titles| {
        s.scan.is_none() && titles.iter().any(|t| t == "2 pages sent to Trenova")
    })
    .await;
    assert!(running.ui.attention().contains(&Attention::ScanEnded));
    stop(running).await;
}

#[tokio::test(flavor = "multi_thread", worker_threads = 2)]
async fn a_scan_stopped_before_any_page_tells_the_web_app_it_was_cancelled() {
    let server = server(json!([request_json("Fake Scanner")])).await;
    let running = start_holding(
        &server,
        vec![Script {
            pages: 0,
            end: ScanOutcome::Finished,
        }],
    );

    until(&running, "the scan to start", |s, _| s.scan.is_some()).await;
    running.commands.send(Command::StopScan).expect("stop");
    until(&running, "the scan to end", |s, _| s.scan.is_none()).await;

    let deadline = tokio::time::Instant::now() + Duration::from_secs(10);
    loop {
        let requests = server.received_requests().await.expect("requests");
        let reported = requests.iter().any(|r| {
            r.url.path().ends_with("/requests/creq_1/status/")
                && serde_json::from_slice::<serde_json::Value>(&r.body).expect("json")
                    ["failureCode"]
                    == "CANCELED_BY_USER"
        });
        if reported {
            break;
        }
        assert!(
            tokio::time::Instant::now() < deadline,
            "the request was not reported cancelled"
        );
        tokio::time::sleep(Duration::from_millis(50)).await;
    }
    assert!(
        !running
            .ui
            .titles()
            .iter()
            .any(|t| t == "Nothing was scanned"),
        "stopping on purpose is not an empty feeder"
    );
    stop(running).await;
}

#[tokio::test(flavor = "multi_thread", worker_threads = 2)]
async fn a_refused_scan_is_listed_and_sent_again_to_intake_when_retried() {
    let server = refusing_server().await;
    let running = start(
        &server,
        vec![Script {
            pages: 2,
            end: ScanOutcome::Finished,
        }],
    );
    ready(&running).await;
    running.commands.send(scan_to_intake()).expect("scan");

    until(&running, "the batch to be refused", |s, _| {
        s.refused.len() == 1
    })
    .await;
    let snapshot = running.shared.snapshot();
    let refused = snapshot.refused[0].clone();
    assert_eq!(refused.label, "Fake Scanner");
    assert_eq!(refused.pages, 2);
    assert!(refused.reason.contains("The scanner name is not valid"));
    assert!(refused.readable);
    assert!(snapshot.waiting.is_empty());
    assert!(running.ui.attention().contains(&Attention::Refused));
    assert_eq!(
        snapshot.messages[0].notice.title, "A scan could not be sent",
        "what the person was told is kept for the window"
    );

    running
        .commands
        .send(Command::Retry(refused.key.clone()))
        .expect("retry");
    until(&running, "the retried batch to be sent", |s, titles| {
        s.refused.is_empty() && titles.iter().any(|t| t == "2 pages sent to Trenova")
    })
    .await;

    let requests = server.received_requests().await.expect("requests");
    let opened: Vec<serde_json::Value> = requests
        .iter()
        .filter(|r| {
            r.method.as_str() == "POST" && r.url.path() == "/api/v1/capture/device/batches/"
        })
        .map(|r| serde_json::from_slice(&r.body).expect("json"))
        .collect();
    assert_eq!(opened.len(), 2);
    assert_ne!(
        opened[0]["clientKey"], opened[1]["clientKey"],
        "sent again under a new key"
    );
    assert!(
        opened[1]
            .get("requestId")
            .is_none_or(serde_json::Value::is_null)
    );
    stop(running).await;
}

#[tokio::test(flavor = "multi_thread", worker_threads = 2)]
async fn a_refused_scan_can_be_saved_as_pdfs_and_then_discarded() {
    let server = refusing_server().await;
    let running = start(
        &server,
        vec![Script {
            pages: 2,
            end: ScanOutcome::Finished,
        }],
    );
    ready(&running).await;
    running.commands.send(scan_to_intake()).expect("scan");
    until(&running, "the batch to be refused", |s, _| {
        s.refused.len() == 1
    })
    .await;
    let key = running.shared.snapshot().refused[0].key.clone();

    let downloads = running.dir.path().join("Downloads");
    for _ in 0..2 {
        running
            .commands
            .send(Command::Save {
                key: key.clone(),
                into: downloads.clone(),
            })
            .expect("save");
    }
    until(&running, "both copies to be saved", |_, titles| {
        titles.iter().filter(|t| *t == "2 files saved").count() == 2
    })
    .await;
    let revealed = running
        .revealed
        .lock()
        .unwrap_or_else(PoisonError::into_inner)
        .clone();
    assert_eq!(
        revealed,
        [
            downloads.join("Fake Scanner scan not sent"),
            downloads.join("Fake Scanner scan not sent (2)"),
        ],
        "a second copy never overwrites the first"
    );
    let first = std::fs::read(revealed[0].join("page-0001.pdf")).expect("page");
    assert!(first.starts_with(b"%PDF-1.7 page 1"));
    assert_eq!(
        running.shared.snapshot().refused.len(),
        1,
        "saving keeps it"
    );

    running
        .commands
        .send(Command::Discard(key.clone()))
        .expect("discard");
    until(&running, "the batch to be discarded", |s, _| {
        s.refused.is_empty()
    })
    .await;
    assert!(
        !running
            .dir
            .path()
            .join("spool")
            .join("failed")
            .join(&key)
            .exists()
    );
    stop(running).await;
}
