//! The agent end to end: a mock server asks for a scan, a scripted scanner
//! produces pages, and the pages arrive and are sealed; a job the print
//! service left in the inbox is sent whole.

use std::collections::VecDeque;
use std::sync::{Arc, Mutex, PoisonError};
use std::time::{Duration, SystemTime, UNIX_EPOCH};

use capture_client::{AgentInfo, Credential, MemoryStore, Protector, SecretStore, Server};
use capture_protocol::api::{Id, PixelType, Settings, SourceInfo, SourceProtocol, TokenPair};
use capture_protocol::handoff::Inbox;
use capture_protocol::helper::{PageMeta, ScanCondition, ScanJob};
use capture_protocol::page_checksum;
use serde_json::json;
use tokio::sync::mpsc;
use tokio_util::sync::CancellationToken;
use trenova_capture::agent::{self, Environment, Machine, ServerSetting};
use trenova_capture::scanners::{ScanOutcome, ScanRun, ScanUpdate, ScannerHost, SourcesFuture};
use trenova_capture::state::{Command, Connection, Notice, Shared, Snapshot, Ui};
use wiremock::matchers::{body_partial_json, method, path};
use wiremock::{Mock, MockServer, Request, Respond, ResponseTemplate};

#[derive(Default)]
struct TestUi {
    notices: Mutex<Vec<Notice>>,
}

impl Ui for TestUi {
    fn refresh(&self) {}
    fn notify(&self, notice: Notice) {
        self.notices
            .lock()
            .unwrap_or_else(PoisonError::into_inner)
            .push(notice);
    }
}

impl TestUi {
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
                };
                let pdf = format!("%PDF-1.7 page {index} of {}", fastrand_like(index)).into_bytes();
                let _ = tx.send(ScanUpdate::Page { meta, pdf }).await;
            }
            let _ = tx.send(ScanUpdate::End(script.end)).await;
        });
        ScanRun {
            updates,
            cancel: CancellationToken::new(),
        }
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
    _dir: tempfile::TempDir,
}

fn start(server: &MockServer, scripts: Vec<Script>) -> Running {
    let dir = tempfile::tempdir().expect("dir");
    let store: Arc<dyn SecretStore> = Arc::new(MemoryStore::with(credential(&server.uri())));
    let scanner = Arc::new(FakeScanner {
        scripts: Mutex::new(scripts.into()),
        jobs: Mutex::new(Vec::new()),
    });
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
        recheck_after: Duration::from_secs(600),
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
        _dir: dir,
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
        .deliver("Rate confirmation", Some(3), b"%PDF-1.7 printed")
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
