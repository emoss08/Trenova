//! The print service end to end: an IPP request over HTTP on loopback, to
//! the PDF in the inbox of the person who printed it.

use std::io;
use std::sync::{Arc, Mutex};
use std::time::Duration;

use capture_imaging::pwg::encode::{Kind, Page, stream};
use capture_ipp::codec::{decode, encode};
use capture_ipp::printer::{op, status};
use capture_ipp::value::{Group, GroupTag, Value};
use capture_ipp::{Printer, PrinterConfig};
use capture_protocol::handoff::Inbox;
use tokio_util::sync::CancellationToken;
use trenova_capture_svc::attribution::{Attributor, Patience, PrintSystem, QueuedJob, SignedIn};
use trenova_capture_svc::handler::{PlainInboxes, PrintHandler};
use trenova_capture_svc::listener::{Limits, bind, serve};

const SID: &str = "S-1-5-21-1-2-3-1001";

struct Queue(Mutex<Vec<QueuedJob>>);

impl PrintSystem for Queue {
    fn queued_jobs(&self) -> io::Result<Vec<QueuedJob>> {
        Ok(self.0.lock().expect("lock").clone())
    }

    fn signed_in(&self) -> io::Result<Vec<SignedIn>> {
        Ok(vec![SignedIn {
            domain: "CONTOSO".into(),
            user: "jdoe".into(),
        }])
    }

    fn sid_of(&self, _domain: &str, user: &str) -> io::Result<String> {
        if user == "jdoe" {
            Ok(SID.into())
        } else {
            Err(io::Error::from(io::ErrorKind::NotFound))
        }
    }
}

struct Running {
    url: String,
    root: tempfile::TempDir,
    stop: CancellationToken,
    served: tokio::task::JoinHandle<io::Result<()>>,
}

async fn start(max_request_bytes: usize) -> Running {
    let root = tempfile::tempdir().expect("temp dir");
    let queue = Queue(Mutex::new(vec![QueuedJob {
        id: 12,
        document: "Bill of lading".into(),
        user: "jdoe".into(),
    }]));
    let attributor = Attributor::new(
        queue,
        Patience {
            wait: Duration::from_millis(50),
            poll: Duration::from_millis(10),
        },
    );
    let handler = PrintHandler::new(
        attributor,
        PlainInboxes {
            root: root.path().to_path_buf(),
        },
    );
    let listener = bind(0).await.expect("binds");
    let port = listener.local_addr().expect("address").port();
    let printer = Arc::new(Printer::new(
        PrinterConfig {
            max_document_bytes: max_request_bytes,
            ..trenova_capture_svc::printer_config(port, "test machine", max_request_bytes)
        },
        handler,
    ));
    let stop = CancellationToken::new();
    let served = tokio::spawn(serve(
        listener,
        printer,
        Limits {
            max_request_bytes,
            concurrent_jobs: 1,
        },
        stop.clone(),
    ));
    Running {
        url: format!("http://127.0.0.1:{port}/ipp/print"),
        root,
        stop,
        served,
    }
}

fn print_job(name: &str, user: &str, document: &[u8]) -> Vec<u8> {
    let mut operation = Group::new(GroupTag::Operation);
    operation.push("attributes-charset", Value::Charset("utf-8".into()));
    operation.push(
        "attributes-natural-language",
        Value::NaturalLanguage("en".into()),
    );
    operation.push(
        "printer-uri",
        Value::Uri("ipp://127.0.0.1/ipp/print".into()),
    );
    operation.push("requesting-user-name", Value::Name(user.into()));
    operation.push("job-name", Value::Name(name.into()));
    operation.push(
        "document-format",
        Value::MimeMediaType("image/pwg-raster".into()),
    );
    let mut body = encode((2, 0), op::PRINT_JOB, 1, &[operation]);
    body.extend_from_slice(document);
    body
}

fn one_page() -> Vec<u8> {
    let (width, height) = (170u32, 220u32);
    let row = width.div_ceil(8) as usize;
    let mut ink = vec![0u8; row * height as usize];
    for y in 0..height as usize {
        ink[y * row] = 0x80;
    }
    stream(&[Page {
        kind: Kind::Black1,
        width,
        height,
        dpi: 100,
        pixels: &ink,
    }])
}

fn post(url: &str, body: Vec<u8>) -> reqwest::RequestBuilder {
    reqwest::Client::new()
        .post(url)
        .header("content-type", "application/ipp")
        .body(body)
}

#[tokio::test]
async fn a_printed_raster_job_lands_in_its_owners_inbox_as_a_pdf() {
    let running = start(8 << 20).await;
    let response = post(
        &running.url,
        print_job("Bill of lading", "jdoe", &one_page()),
    )
    .send()
    .await
    .expect("sends");
    assert_eq!(response.status(), 200);
    assert_eq!(response.headers()["content-type"], "application/ipp",);
    let answer = decode(&response.bytes().await.expect("body")).expect("IPP");
    assert_eq!(answer.code, status::SUCCESSFUL_OK);

    let inbox = Inbox::new(running.root.path().join(SID));
    let waiting = inbox.waiting().expect("lists");
    assert_eq!(waiting.len(), 1);
    let job = waiting[0].as_ref().expect("readable");
    assert_eq!(job.name, "Bill of lading");
    assert_eq!(job.pages, Some(1));
    let pdf = inbox.read(job).expect("reads");
    assert!(pdf.starts_with(b"%PDF-"));
    assert!(String::from_utf8_lossy(&pdf).contains("/CCITTFaxDecode"));

    let again = post(
        &running.url,
        print_job("Bill of lading", "jdoe", &one_page()),
    )
    .send()
    .await
    .expect("sends");
    let refused = decode(&again.bytes().await.expect("body")).expect("IPP");
    assert_eq!(
        refused.code,
        status::CLIENT_ERROR_NOT_AUTHORIZED,
        "the queued job was already claimed"
    );
    assert_eq!(inbox.waiting().expect("lists").len(), 1);

    running.stop.cancel();
    running.served.await.expect("joins").expect("served");
}

#[tokio::test]
async fn requests_a_browser_or_a_stranger_could_send_are_refused_before_the_body() {
    let running = start(4096).await;
    let body = || print_job("Bill of lading", "jdoe", b"RaS2");
    let client = reqwest::Client::new();

    let from_page = post(&running.url, body())
        .header("origin", "https://evil.example")
        .send()
        .await
        .expect("sends");
    assert_eq!(from_page.status(), 403);

    let rebound = post(&running.url, body())
        .header("host", "evil.example:8631")
        .send()
        .await
        .expect("sends");
    assert_eq!(rebound.status(), 403);

    let form = client
        .post(&running.url)
        .header("content-type", "text/plain")
        .body(body())
        .send()
        .await
        .expect("sends");
    assert_eq!(form.status(), 415);

    let get = client.get(&running.url).send().await.expect("sends");
    assert_eq!(get.status(), 405);
    assert_eq!(get.headers()["allow"], "POST");

    let elsewhere = post(&running.url.replace("/ipp/print", "/admin"), body())
        .send()
        .await
        .expect("sends");
    assert_eq!(elsewhere.status(), 404);

    let large = post(&running.url, vec![0u8; 8192])
        .send()
        .await
        .expect("sends");
    assert_eq!(large.status(), 413);

    assert!(
        std::fs::read_dir(running.root.path())
            .expect("lists")
            .next()
            .is_none(),
        "nothing reached an inbox"
    );
    running.stop.cancel();
    running.served.await.expect("joins").expect("served");
}
