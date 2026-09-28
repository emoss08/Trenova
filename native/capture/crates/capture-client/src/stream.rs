//! Keeping the device stream open.
//!
//! The stream only ever says "look at your requests" (or "you are revoked"),
//! and the agent fetches requests on every connect as well as on every
//! signal, so a signal lost to a reconnect loses nothing. The stream is also
//! how the server learns the device is online.

use std::sync::Arc;
use std::time::Duration;

use capture_protocol::api::{DeviceSignal, events};
use futures_util::StreamExt;
use tokio::sync::mpsc;
use tokio_util::sync::CancellationToken;

use crate::api::Api;
use crate::backoff::Backoff;
use crate::error::ApiError;
use crate::sse::SseParser;

/// Heartbeats come every 15 seconds; four missed means the connection is
/// dead even if the socket has not noticed.
const IDLE_TIMEOUT: Duration = Duration::from_secs(60);
/// A stream the server closed on purpose is reopened after about this long.
const ROTATE_DELAY: Duration = Duration::from_secs(1);

/// What the stream tells the agent.
#[derive(Debug)]
pub enum DeviceEvent {
    /// The stream is open.
    Connected,
    /// The stream dropped and will be retried.
    Disconnected { reason: String, retry_in: Duration },
    /// Fetch open requests now.
    FetchRequests,
    /// The device was revoked; the credential has been forgotten.
    Revoked,
    /// Nothing more can happen until something changes: signed out, too old,
    /// or capture turned off. The stream has stopped.
    Stopped(ApiError),
}

enum Ending {
    /// The server ended the stream; open another.
    Closed,
    /// The connection failed; back off.
    Failed(String),
    Revoked,
    Canceled,
}

/// Keeps the stream open until cancelled, revoked or blocked, reporting what
/// happens on `events`.
pub async fn run(api: Arc<Api>, events: mpsc::Sender<DeviceEvent>, cancel: CancellationToken) {
    let mut backoff = Backoff::new(Duration::from_secs(1), Duration::from_secs(60));
    let mut last_id: Option<String> = None;

    loop {
        let opened = tokio::select! {
            () = cancel.cancelled() => return,
            opened = api.open_stream(last_id.as_deref()) => opened,
        };
        let ending = match opened {
            Ok(response) => {
                backoff.reset();
                consume(response, &events, &mut last_id, &cancel).await
            }
            Err(err) if err.blocks_everything() => {
                let _ = events.send(DeviceEvent::Stopped(err)).await;
                return;
            }
            Err(err) => {
                let wait = err.retry_after().unwrap_or_else(|| backoff.next_delay());
                if !disconnected(&events, err.to_string(), wait, &cancel).await {
                    return;
                }
                continue;
            }
        };

        match ending {
            Ending::Canceled => return,
            Ending::Revoked => {
                api.sign_out().await;
                let _ = events.send(DeviceEvent::Revoked).await;
                return;
            }
            Ending::Closed => {
                let wait = ROTATE_DELAY + Duration::from_millis(fastrand::u64(0..1000));
                tokio::select! {
                    () = cancel.cancelled() => return,
                    () = tokio::time::sleep(wait) => {}
                }
            }
            Ending::Failed(reason) => {
                let wait = backoff.next_delay();
                if !disconnected(&events, reason, wait, &cancel).await {
                    return;
                }
            }
        }
    }
}

/// Reports a drop and waits, returning false if cancelled meanwhile.
async fn disconnected(
    events: &mpsc::Sender<DeviceEvent>,
    reason: String,
    wait: Duration,
    cancel: &CancellationToken,
) -> bool {
    let _ = events
        .send(DeviceEvent::Disconnected {
            reason,
            retry_in: wait,
        })
        .await;
    tokio::select! {
        () = cancel.cancelled() => false,
        () = tokio::time::sleep(wait) => true,
    }
}

async fn consume(
    response: reqwest::Response,
    events: &mpsc::Sender<DeviceEvent>,
    last_id: &mut Option<String>,
    cancel: &CancellationToken,
) -> Ending {
    let mut body = response.bytes_stream();
    let mut parser = SseParser::new();

    loop {
        let chunk = tokio::select! {
            () = cancel.cancelled() => return Ending::Canceled,
            chunk = tokio::time::timeout(IDLE_TIMEOUT, body.next()) => chunk,
        };
        let bytes = match chunk {
            Err(_) => return Ending::Failed("no heartbeat from the server".into()),
            Ok(None) => return Ending::Closed,
            Ok(Some(Err(err))) => return Ending::Failed(err.to_string()),
            Ok(Some(Ok(bytes))) => bytes,
        };
        let parsed = match parser.push(&bytes) {
            Ok(parsed) => parsed,
            Err(err) => return Ending::Failed(err.to_string()),
        };
        for event in parsed {
            if event.id.is_some() {
                last_id.clone_from(&event.id);
            }
            let signal = match event.event.as_str() {
                events::READY => {
                    let _ = events.send(DeviceEvent::Connected).await;
                    Some(DeviceEvent::FetchRequests)
                }
                events::RESET | events::CAPTURE_REQUEST => Some(DeviceEvent::FetchRequests),
                events::CAPTURE_REVOKED => {
                    if serde_json::from_str::<DeviceSignal>(&event.data).is_ok() {
                        return Ending::Revoked;
                    }
                    None
                }
                events::CLOSE => return Ending::Closed,
                _ => None,
            };
            if let Some(signal) = signal {
                let _ = events.send(signal).await;
            }
        }
    }
}
