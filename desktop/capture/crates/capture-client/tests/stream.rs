//! The device stream against a mock server.

mod common;

use std::time::Duration;

use capture_client::SecretStore;
use capture_client::stream::{DeviceEvent, run};
use common::device;
use tokio::sync::mpsc;
use tokio_util::sync::CancellationToken;
use wiremock::matchers::{header, method, path};
use wiremock::{Mock, MockServer, ResponseTemplate};

#[tokio::test]
async fn signals_become_fetches_and_a_revocation_signs_the_device_out() {
    let server = MockServer::start().await;
    let body = concat!(
        ": hello\n\n",
        "id: 1-0\nevent: ready\ndata: {}\n\n",
        "event: heartbeat\ndata: {}\n\n",
        "id: 2-0\nevent: capture.request\ndata: {\"deviceId\":\"cdev_1\",\"requestId\":\"creq_1\"}\n\n",
        "event: capture.revoked\ndata: {\"deviceId\":\"cdev_1\"}\n\n",
    );
    Mock::given(method("GET"))
        .and(path("/api/v1/capture/device/stream/"))
        .and(header("accept", "text/event-stream"))
        .and(header("authorization", "Bearer tcd_at_old"))
        .respond_with(ResponseTemplate::new(200).set_body_raw(body, "text/event-stream"))
        .expect(1)
        .mount(&server)
        .await;

    let device = device(&server, 900);
    let (tx, mut rx) = mpsc::channel(16);
    let task = tokio::spawn(run(device.api.clone(), tx, CancellationToken::new()));
    let mut events = Vec::new();
    while let Ok(Some(event)) = tokio::time::timeout(Duration::from_secs(10), rx.recv()).await {
        events.push(event);
    }
    task.await.expect("stream");

    assert!(
        matches!(
            events.as_slice(),
            [
                DeviceEvent::Connected,
                DeviceEvent::FetchRequests,
                DeviceEvent::FetchRequests,
                DeviceEvent::Revoked
            ]
        ),
        "{events:?}"
    );
    assert!(device.store.load().expect("load").is_none());
}

#[tokio::test]
async fn a_closed_stream_is_reopened_with_the_last_event_id() {
    let server = MockServer::start().await;
    Mock::given(method("GET"))
        .and(path("/api/v1/capture/device/stream/"))
        .and(header("last-event-id", "5-0"))
        .respond_with(ResponseTemplate::new(200).set_body_raw(
            "event: capture.revoked\ndata: {\"deviceId\":\"cdev_1\"}\n\n",
            "text/event-stream",
        ))
        .expect(1)
        .mount(&server)
        .await;
    Mock::given(method("GET"))
        .and(path("/api/v1/capture/device/stream/"))
        .respond_with(ResponseTemplate::new(200).set_body_raw(
            "id: 5-0\nevent: ready\ndata: {}\n\nevent: close\ndata: {\"reason\":\"rotate\"}\n\n",
            "text/event-stream",
        ))
        .up_to_n_times(1)
        .expect(1)
        .mount(&server)
        .await;

    let device = device(&server, 900);
    let (tx, mut rx) = mpsc::channel(16);
    let task = tokio::spawn(run(device.api.clone(), tx, CancellationToken::new()));
    let mut revoked = false;
    while let Ok(Some(event)) = tokio::time::timeout(Duration::from_secs(10), rx.recv()).await {
        revoked |= matches!(event, DeviceEvent::Revoked);
    }
    task.await.expect("stream");
    assert!(revoked);
}

#[tokio::test]
async fn an_outdated_agent_stops_the_stream_rather_than_hammering_the_server() {
    let server = MockServer::start().await;
    Mock::given(method("GET"))
        .and(path("/api/v1/capture/device/stream/"))
        .respond_with(ResponseTemplate::new(426).set_body_json(serde_json::json!({
            "error": "agent_outdated", "minimumVersion": "2.0.0"
        })))
        .expect(1)
        .mount(&server)
        .await;
    let device = device(&server, 900);
    let (tx, mut rx) = mpsc::channel(16);
    run(device.api.clone(), tx, CancellationToken::new()).await;
    assert!(matches!(
        rx.recv().await,
        Some(DeviceEvent::Stopped(
            capture_client::ApiError::Outdated { .. }
        ))
    ));
}
