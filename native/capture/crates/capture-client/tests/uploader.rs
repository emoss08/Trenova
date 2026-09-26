//! Spooled batches reaching a mock server.

mod common;

use std::io;
use std::sync::Arc;
use std::time::Duration;

use capture_client::spool::Spool;
use capture_client::uploader::{UploadEvent, Uploader};
use capture_client::{PageMarkers, Protector};
use capture_protocol::api::{BatchSource, Id, OpenBatchInput, Settings};
use capture_protocol::manifest::manifest_digest;
use capture_protocol::page_checksum;
use common::{device, problem};
use serde_json::json;
use tokio::sync::mpsc;
use tokio_util::sync::CancellationToken;
use wiremock::matchers::{body_bytes, body_partial_json, header, method, path};
use wiremock::{Mock, MockServer, Request, Respond, ResponseTemplate};

#[derive(Debug)]
struct Xor;

impl Protector for Xor {
    fn protect(&self, plain: &[u8]) -> io::Result<Vec<u8>> {
        Ok(plain.iter().map(|b| b ^ 0x5A).collect())
    }
    fn unprotect(&self, sealed: &[u8]) -> io::Result<Vec<u8>> {
        self.protect(sealed)
    }
}

fn input(key: &str, request: Option<&str>) -> OpenBatchInput {
    OpenBatchInput {
        client_key: key.to_owned(),
        source: BatchSource::Scan,
        request_id: request.map(Id::from),
        profile_id: None,
        source_name: "fi-8170".into(),
        job_name: String::new(),
        settings: Settings::default(),
    }
}

const PAGE_ONE: &[u8] = b"%PDF-1.7 page one";
const PAGE_TWO: &[u8] = b"%PDF-1.7 page two";

fn spooled(dir: &std::path::Path, request: Option<&str>) -> (Arc<Spool>, String) {
    let spool = Arc::new(Spool::open(dir, Arc::new(Xor)).expect("spool"));
    let key = Spool::new_key();
    spool
        .create(input(&key, request), "fi-8170")
        .expect("create");
    spool
        .append_page(
            &key,
            PAGE_ONE,
            &PageMarkers {
                dpi: 300,
                patch_code: Some("T".into()),
                barcodes: vec!["PRO 1042".into(), "bad\u{7}code".into()],
            },
        )
        .expect("page");
    spool
        .append_page(&key, PAGE_TWO, &PageMarkers::default())
        .expect("page");
    spool.complete(&key).expect("complete");
    (spool, key)
}

/// Answers a page upload as the server does: the stored page, with the
/// checksum of what it received.
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
            "id": format!("cpg_{sequence}"),
            "batchId": "cbat_1",
            "sequence": sequence,
            "checksumSha256": page_checksum(&request.body),
            "byteSize": request.body.len()
        }))
    }
}

fn batch_json(request: Option<&str>) -> serde_json::Value {
    json!({"id": "cbat_1", "status": "Receiving", "source": "Scan", "requestId": request})
}

async fn run_until_settled(
    uploader: Uploader,
    events: &mut mpsc::Receiver<UploadEvent>,
) -> Vec<UploadEvent> {
    let cancel = CancellationToken::new();
    let task = tokio::spawn(uploader.run(cancel.clone()));
    let mut seen = Vec::new();
    while let Ok(Some(event)) = tokio::time::timeout(Duration::from_secs(10), events.recv()).await {
        let done = matches!(event, UploadEvent::Progress(summary) if summary.batches == 0);
        seen.push(event);
        if done {
            break;
        }
    }
    cancel.cancel();
    task.await.expect("uploader");
    seen
}

#[tokio::test]
async fn a_batch_is_opened_uploaded_page_by_page_and_sealed_with_its_digest() {
    let server = MockServer::start().await;
    let dir = tempfile::tempdir().expect("dir");
    let (spool, key) = spooled(dir.path(), Some("creq_1"));

    Mock::given(method("POST"))
        .and(path("/api/v1/capture/device/batches/"))
        .and(body_partial_json(
            json!({"clientKey": key, "requestId": "creq_1", "source": "Scan"}),
        ))
        .respond_with(ResponseTemplate::new(200).set_body_json(batch_json(Some("creq_1"))))
        .expect(1)
        .mount(&server)
        .await;
    Mock::given(method("PUT"))
        .and(path("/api/v1/capture/device/batches/cbat_1/pages/1/"))
        .and(header("content-type", "application/pdf"))
        .and(header("x-capture-dpi", "300"))
        .and(header("x-capture-patch-code", "T"))
        .and(header("x-capture-barcode", "PRO 1042"))
        .and(body_bytes(PAGE_ONE))
        .respond_with(StorePage)
        .expect(1)
        .mount(&server)
        .await;
    Mock::given(method("PUT"))
        .and(path("/api/v1/capture/device/batches/cbat_1/pages/2/"))
        .and(body_bytes(PAGE_TWO))
        .respond_with(StorePage)
        .expect(1)
        .mount(&server)
        .await;
    let digest = manifest_digest([
        page_checksum(PAGE_ONE).as_str(),
        page_checksum(PAGE_TWO).as_str(),
    ]);
    Mock::given(method("POST"))
        .and(path("/api/v1/capture/device/batches/cbat_1/seal/"))
        .and(body_partial_json(
            json!({"pageCount": 2, "manifestDigest": digest}),
        ))
        .respond_with(ResponseTemplate::new(200).set_body_json(batch_json(Some("creq_1"))))
        .expect(1)
        .mount(&server)
        .await;

    let device = device(&server, 900);
    let (tx, mut rx) = mpsc::channel(32);
    let events =
        run_until_settled(Uploader::new(device.api, Arc::clone(&spool), tx), &mut rx).await;

    assert!(events.iter().any(|e| matches!(e,
        UploadEvent::Sent { batch_id, pages: 2, requested: true, .. } if batch_id.as_str() == "cbat_1")));
    assert!(
        spool.pending().expect("pending").is_empty(),
        "a sealed batch leaves the spool"
    );

    let requests = server.received_requests().await.expect("requests");
    let first_page = requests
        .iter()
        .find(|r| r.url.path().ends_with("/pages/1/"))
        .expect("page one");
    assert_eq!(
        first_page
            .headers
            .get_all("x-capture-barcode")
            .iter()
            .count(),
        1,
        "a barcode that cannot travel as a header is dropped"
    );
}

#[tokio::test]
async fn a_request_that_closed_while_offline_sends_the_pages_to_intake() {
    let server = MockServer::start().await;
    let dir = tempfile::tempdir().expect("dir");
    let (spool, key) = spooled(dir.path(), Some("creq_gone"));

    Mock::given(method("POST"))
        .and(path("/api/v1/capture/device/batches/"))
        .and(body_partial_json(json!({"requestId": "creq_gone"})))
        .respond_with(ResponseTemplate::new(409).set_body_json(problem(
            409,
            "conflict",
            "That capture request is no longer open",
        )))
        .expect(1)
        .mount(&server)
        .await;
    Mock::given(method("POST"))
        .and(path("/api/v1/capture/device/batches/"))
        .and(body_partial_json(
            json!({"clientKey": key, "requestId": null}),
        ))
        .respond_with(ResponseTemplate::new(200).set_body_json(batch_json(None)))
        .expect(1)
        .mount(&server)
        .await;
    Mock::given(method("PUT"))
        .respond_with(StorePage)
        .mount(&server)
        .await;
    Mock::given(method("POST"))
        .and(path("/api/v1/capture/device/batches/cbat_1/seal/"))
        .respond_with(ResponseTemplate::new(200).set_body_json(batch_json(None)))
        .mount(&server)
        .await;

    let device = device(&server, 900);
    let (tx, mut rx) = mpsc::channel(32);
    let events = run_until_settled(Uploader::new(device.api, spool, tx), &mut rx).await;
    assert!(events.iter().any(|e| matches!(
        e,
        UploadEvent::Sent {
            requested: false,
            ..
        }
    )));
}

#[tokio::test]
async fn an_interrupted_upload_resumes_with_the_pages_the_server_lacks() {
    let server = MockServer::start().await;
    let dir = tempfile::tempdir().expect("dir");
    let (spool, key) = spooled(dir.path(), None);
    spool.set_batch_id(&key, Id::from("cbat_1")).expect("id");
    spool.mark_uploaded(&key, 1).expect("uploaded");

    Mock::given(method("POST"))
        .and(path("/api/v1/capture/device/batches/"))
        .respond_with(ResponseTemplate::new(500))
        .expect(0)
        .mount(&server)
        .await;
    Mock::given(method("PUT"))
        .and(path("/api/v1/capture/device/batches/cbat_1/pages/2/"))
        .respond_with(StorePage)
        .expect(1)
        .mount(&server)
        .await;
    Mock::given(method("PUT"))
        .and(path("/api/v1/capture/device/batches/cbat_1/pages/1/"))
        .respond_with(StorePage)
        .expect(0)
        .mount(&server)
        .await;
    Mock::given(method("POST"))
        .and(path("/api/v1/capture/device/batches/cbat_1/seal/"))
        .respond_with(ResponseTemplate::new(200).set_body_json(batch_json(None)))
        .expect(1)
        .mount(&server)
        .await;

    let device = device(&server, 900);
    let (tx, mut rx) = mpsc::channel(32);
    run_until_settled(Uploader::new(device.api, spool, tx), &mut rx).await;
}

#[tokio::test]
async fn a_server_error_is_retried_and_a_refused_page_is_set_aside() {
    let server = MockServer::start().await;
    let dir = tempfile::tempdir().expect("dir");
    let (spool, _key) = spooled(dir.path(), None);

    Mock::given(method("POST"))
        .and(path("/api/v1/capture/device/batches/"))
        .respond_with(ResponseTemplate::new(503))
        .up_to_n_times(1)
        .expect(1)
        .mount(&server)
        .await;
    Mock::given(method("POST"))
        .and(path("/api/v1/capture/device/batches/"))
        .respond_with(ResponseTemplate::new(200).set_body_json(batch_json(None)))
        .mount(&server)
        .await;
    Mock::given(method("PUT"))
        .respond_with(ResponseTemplate::new(422).set_body_json(problem(
            422,
            "validation",
            "The upload is not a PDF",
        )))
        .mount(&server)
        .await;

    let device = device(&server, 900);
    let (tx, mut rx) = mpsc::channel(32);
    let events =
        run_until_settled(Uploader::new(device.api, Arc::clone(&spool), tx), &mut rx).await;

    assert!(
        events
            .iter()
            .any(|e| matches!(e, UploadEvent::Waiting { .. }))
    );
    assert!(events.iter().any(|e| matches!(e,
        UploadEvent::Refused { reason, .. } if reason == "The upload is not a PDF")));
    assert_eq!(
        spool.summary().expect("summary").failed,
        1,
        "the pages are kept aside, not deleted"
    );
}

#[tokio::test]
async fn capture_turned_off_stops_uploads_and_keeps_every_page() {
    let server = MockServer::start().await;
    let dir = tempfile::tempdir().expect("dir");
    let (spool, _key) = spooled(dir.path(), None);
    let mut disabled = problem(
        422,
        "business-rule-violation",
        "document capture is turned off for this organization",
    );
    disabled["params"] = json!({"reason": "capture_disabled"});
    Mock::given(method("POST"))
        .and(path("/api/v1/capture/device/batches/"))
        .respond_with(ResponseTemplate::new(422).set_body_json(disabled))
        .mount(&server)
        .await;

    let device = device(&server, 900);
    let (tx, mut rx) = mpsc::channel(32);
    let cancel = CancellationToken::new();
    let task = tokio::spawn(Uploader::new(device.api, Arc::clone(&spool), tx).run(cancel.clone()));
    let blocked = tokio::time::timeout(Duration::from_secs(10), async {
        loop {
            match rx.recv().await {
                Some(UploadEvent::Blocked(err)) => return err,
                Some(_) => {}
                None => panic!("uploader stopped"),
            }
        }
    })
    .await
    .expect("blocked");
    cancel.cancel();
    task.await.expect("uploader");

    assert!(matches!(blocked, capture_client::ApiError::CaptureDisabled));
    let summary = spool.summary().expect("summary");
    assert_eq!(
        (summary.batches, summary.pages_waiting, summary.failed),
        (1, 2, 0)
    );
}
