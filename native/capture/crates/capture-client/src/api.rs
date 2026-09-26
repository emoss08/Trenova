//! The capture API, as the device calls it.
//!
//! The access token lives fifteen minutes and is refreshed a minute before it
//! runs out, or at once when a call is refused with 401. The refresh token is
//! single-use, so refreshes are serialised: whoever finds the token stale
//! refreshes it while holding the credential, and everyone waiting behind
//! them gets the new one instead of spending the old refresh token again,
//! which the server would treat as theft and revoke the device for.

use std::sync::Arc;
use std::time::{Duration, SystemTime, UNIX_EPOCH};

use bytes::Bytes;
use capture_protocol::api::{
    BARCODE_HEADER, CaptureBatch, CaptureDevice, CapturePage, CaptureProfile, CaptureRequest,
    DPI_HEADER, DeviceIdentity, ExchangePairingRequest, Id, OAuthError, OAuthErrorCode,
    OpenBatchInput, OutdatedAgent, PATCH_CODE_HEADER, PairingGrant, ProblemDetail, RefreshRequest,
    ReportSourcesRequest, RequestStatusReport, SealBatchInput, SourceInfo, StartPairingRequest,
    TokenPair, known_patch_code,
};
use reqwest::header::{ACCEPT, CONTENT_TYPE, HeaderMap, HeaderValue, RETRY_AFTER};
use reqwest::{RequestBuilder, Response, StatusCode};
use serde::de::DeserializeOwned;
use tokio::sync::{Mutex, MutexGuard};

use crate::credentials::{Credential, SecretStore};
use crate::error::ApiError;
use crate::server::Server;

/// Refresh this long before the access token runs out.
const REFRESH_MARGIN_SECS: i64 = 60;
/// Every call but the stream and uploads gives up after this.
const CALL_TIMEOUT: Duration = Duration::from_secs(30);
/// A page is at most 20 MB; on a slow uplink that is minutes.
const UPLOAD_TIMEOUT: Duration = Duration::from_secs(600);
const CONNECT_TIMEOUT: Duration = Duration::from_secs(15);
/// The longest barcode passed on in a header; longer ones are dropped.
const MAX_BARCODE_HEADER: usize = 256;

/// Who is calling, sent with pairing and every refresh.
#[derive(Clone, Debug, PartialEq, Eq)]
pub struct AgentInfo {
    pub version: String,
    pub os_version: String,
}

/// How an approval poll went.
#[derive(Clone, Debug, PartialEq, Eq)]
pub enum PairingPoll {
    Pending,
    SlowDown,
    Denied,
    Expired,
    Approved(TokenPair),
}

/// Scanner markers sent with a page.
#[derive(Clone, Debug, Default, PartialEq, Eq)]
pub struct PageMarkers {
    pub dpi: u32,
    pub patch_code: Option<String>,
    pub barcodes: Vec<String>,
}

pub struct Api {
    http: reqwest::Client,
    stream_http: reqwest::Client,
    server: Server,
    agent: AgentInfo,
    store: Arc<dyn SecretStore>,
    credential: Mutex<Option<Credential>>,
}

impl std::fmt::Debug for Api {
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        f.debug_struct("Api")
            .field("server", &self.server)
            .field("agent", &self.agent)
            .finish_non_exhaustive()
    }
}

fn now_unix() -> i64 {
    SystemTime::now()
        .duration_since(UNIX_EPOCH)
        .map_or(0, |d| i64::try_from(d.as_secs()).unwrap_or(i64::MAX))
}

fn user_agent(agent: &AgentInfo) -> String {
    format!("TrenovaCapture/{} ({})", agent.version, agent.os_version)
}

impl Api {
    /// A client for `server`, signed in if `store` holds a credential for it.
    pub fn new(
        server: Server,
        agent: AgentInfo,
        store: Arc<dyn SecretStore>,
    ) -> Result<Self, ApiError> {
        let build = |timeout: Option<Duration>| {
            let mut builder = reqwest::Client::builder()
                .user_agent(user_agent(&agent))
                .connect_timeout(CONNECT_TIMEOUT)
                .https_only(server.as_str().starts_with("https://"))
                .redirect(reqwest::redirect::Policy::none());
            if let Some(timeout) = timeout {
                builder = builder.timeout(timeout);
            }
            builder.build()
        };
        let http = build(Some(UPLOAD_TIMEOUT))?;
        let stream_http = build(None)?;

        let credential = store
            .load()
            .map_err(|e| ApiError::Network(format!("could not read the saved credential: {e}")))?
            .filter(|c| c.server == server.as_str());

        Ok(Self {
            http,
            stream_http,
            server,
            agent,
            store,
            credential: Mutex::new(credential),
        })
    }

    pub fn server(&self) -> &Server {
        &self.server
    }

    /// The credential in use, if signed in.
    pub async fn credential(&self) -> Option<Credential> {
        self.credential.lock().await.clone()
    }

    /// Takes a fresh credential, from pairing.
    pub async fn sign_in(&self, credential: Credential) -> Result<(), ApiError> {
        let mut guard = self.credential.lock().await;
        self.store
            .save(&credential)
            .map_err(|e| ApiError::Network(format!("could not save the credential: {e}")))?;
        *guard = Some(credential);
        Ok(())
    }

    /// Forgets the credential here. The server side is revoked by the
    /// person, from the web app.
    pub async fn sign_out(&self) {
        let mut guard = self.credential.lock().await;
        forget(&self.store, &mut guard);
    }

    /// Starts pairing: the grant whose code the person approves.
    pub async fn start_pairing(
        &self,
        request: &StartPairingRequest,
    ) -> Result<PairingGrant, ApiError> {
        let response = self
            .http
            .post(self.server.api("pair/"))
            .timeout(CALL_TIMEOUT)
            .json(request)
            .send()
            .await?;
        decode(response).await
    }

    /// Asks whether the person has approved yet.
    pub async fn poll_pairing(&self, device_code: &str) -> Result<PairingPoll, ApiError> {
        let response = self
            .http
            .post(self.server.api("pair/token/"))
            .timeout(CALL_TIMEOUT)
            .json(&ExchangePairingRequest {
                device_code: device_code.to_owned(),
            })
            .send()
            .await?;
        if response.status() == StatusCode::BAD_REQUEST {
            let body = response.bytes().await?;
            let error: OAuthError =
                serde_json::from_slice(&body).map_err(|e| ApiError::Decode(e.to_string()))?;
            return Ok(match error.error {
                OAuthErrorCode::AuthorizationPending => PairingPoll::Pending,
                OAuthErrorCode::SlowDown => PairingPoll::SlowDown,
                OAuthErrorCode::AccessDenied => PairingPoll::Denied,
                OAuthErrorCode::ExpiredToken
                | OAuthErrorCode::InvalidGrant
                | OAuthErrorCode::Unknown => PairingPoll::Expired,
            });
        }
        decode(response).await.map(PairingPoll::Approved)
    }

    /// A current access token, refreshed first if it is about to run out.
    async fn token(&self) -> Result<String, ApiError> {
        let mut guard = self.credential.lock().await;
        let credential = guard.as_ref().ok_or(ApiError::NotSignedIn)?;
        if credential.tokens.access_token_expires_at - now_unix() > REFRESH_MARGIN_SECS {
            return Ok(credential.tokens.access_token.clone());
        }
        self.refresh(&mut guard).await
    }

    /// A token to retry with after `rejected` was refused. If someone else
    /// already refreshed, theirs is used.
    async fn token_after_rejection(&self, rejected: &str) -> Result<String, ApiError> {
        let mut guard = self.credential.lock().await;
        let credential = guard.as_ref().ok_or(ApiError::NotSignedIn)?;
        if credential.tokens.access_token != rejected {
            return Ok(credential.tokens.access_token.clone());
        }
        self.refresh(&mut guard).await
    }

    /// Rotates the credential. Holding the guard is what makes it single
    /// flight.
    async fn refresh(
        &self,
        guard: &mut MutexGuard<'_, Option<Credential>>,
    ) -> Result<String, ApiError> {
        let Some(credential) = guard.as_ref() else {
            return Err(ApiError::NotSignedIn);
        };
        let response = self
            .http
            .post(self.server.api("token/refresh/"))
            .timeout(CALL_TIMEOUT)
            .json(&RefreshRequest {
                refresh_token: credential.tokens.refresh_token.clone(),
                agent_version: self.agent.version.clone(),
                os_version: self.agent.os_version.clone(),
            })
            .send()
            .await?;

        if response.status() == StatusCode::BAD_REQUEST {
            let body = response.bytes().await?;
            if let Ok(error) = serde_json::from_slice::<OAuthError>(&body)
                && matches!(
                    error.error,
                    OAuthErrorCode::InvalidGrant
                        | OAuthErrorCode::AccessDenied
                        | OAuthErrorCode::ExpiredToken
                )
            {
                forget(&self.store, guard);
                return Err(ApiError::SignedOut);
            }
            return Err(error_for(StatusCode::BAD_REQUEST, &body, None));
        }
        if response.status() == StatusCode::UNAUTHORIZED {
            forget(&self.store, guard);
            return Err(ApiError::SignedOut);
        }

        let tokens: TokenPair = decode(response).await?;
        let renewed = Credential {
            server: credential.server.clone(),
            web_base: credential.web_base.clone(),
            tokens,
        };
        if let Err(err) = self.store.save(&renewed) {
            tracing::error!(error = %err, "could not save the renewed credential");
        }
        let access = renewed.tokens.access_token.clone();
        **guard = Some(renewed);
        Ok(access)
    }

    /// Sends an authenticated request, refreshing and retrying once on 401.
    async fn send(&self, build: impl Fn(&str) -> RequestBuilder) -> Result<Response, ApiError> {
        let token = self.token().await?;
        let response = build(&token).send().await?;
        if response.status() != StatusCode::UNAUTHORIZED {
            return Ok(response);
        }
        let token = self.token_after_rejection(&token).await?;
        let response = build(&token).send().await?;
        if response.status() == StatusCode::UNAUTHORIZED {
            self.sign_out().await;
            return Err(ApiError::SignedOut);
        }
        Ok(response)
    }

    async fn call<T: DeserializeOwned>(
        &self,
        build: impl Fn(&str) -> RequestBuilder,
    ) -> Result<T, ApiError> {
        decode(self.send(build).await?).await
    }

    fn get(&self, path: &str, token: &str) -> RequestBuilder {
        self.http
            .get(self.server.api(path))
            .timeout(CALL_TIMEOUT)
            .bearer_auth(token)
    }

    /// The device and who it acts for.
    pub async fn identity(&self) -> Result<DeviceIdentity, ApiError> {
        self.call(|token| self.get("device/", token)).await
    }

    /// Reports the scanners this computer can reach.
    pub async fn report_sources(&self, sources: &[SourceInfo]) -> Result<CaptureDevice, ApiError> {
        self.call(|token| {
            self.http
                .put(self.server.api("device/sources/"))
                .timeout(CALL_TIMEOUT)
                .bearer_auth(token)
                .json(&ReportSourcesRequest { sources })
        })
        .await
    }

    /// The profiles the person may scan with.
    pub async fn profiles(&self) -> Result<Vec<CaptureProfile>, ApiError> {
        self.call(|token| self.get("device/profiles/", token)).await
    }

    /// What the person has asked this computer to do, oldest first.
    pub async fn open_requests(&self) -> Result<Vec<CaptureRequest>, ApiError> {
        self.call(|token| self.get("device/requests/", token)).await
    }

    pub async fn report_request(
        &self,
        id: &Id,
        report: &RequestStatusReport,
    ) -> Result<CaptureRequest, ApiError> {
        let path = format!("device/requests/{}/status/", segment(id));
        self.call(|token| {
            self.http
                .post(self.server.api(&path))
                .timeout(CALL_TIMEOUT)
                .bearer_auth(token)
                .json(report)
        })
        .await
    }

    /// Opens a batch, or returns the one already opened with this key.
    pub async fn open_batch(&self, input: &OpenBatchInput) -> Result<CaptureBatch, ApiError> {
        self.call(|token| {
            self.http
                .post(self.server.api("device/batches/"))
                .timeout(CALL_TIMEOUT)
                .bearer_auth(token)
                .json(input)
        })
        .await
    }

    /// Uploads one page. Sending the same page again returns what is stored.
    pub async fn put_page(
        &self,
        batch_id: &Id,
        sequence: u32,
        pdf: Bytes,
        markers: &PageMarkers,
    ) -> Result<CapturePage, ApiError> {
        let path = format!("device/batches/{}/pages/{sequence}/", segment(batch_id));
        let headers = marker_headers(markers);
        self.call(|token| {
            self.http
                .put(self.server.api(&path))
                .bearer_auth(token)
                .headers(headers.clone())
                .header(CONTENT_TYPE, "application/pdf")
                .body(pdf.clone())
        })
        .await
    }

    /// Uploads a whole print job, which the server splits and seals.
    pub async fn put_print_job(&self, batch_id: &Id, pdf: Bytes) -> Result<CaptureBatch, ApiError> {
        let path = format!("device/batches/{}/print-job/", segment(batch_id));
        self.call(|token| {
            self.http
                .put(self.server.api(&path))
                .bearer_auth(token)
                .header(CONTENT_TYPE, "application/pdf")
                .body(pdf.clone())
        })
        .await
    }

    pub async fn seal_batch(
        &self,
        batch_id: &Id,
        seal: &SealBatchInput,
    ) -> Result<CaptureBatch, ApiError> {
        let path = format!("device/batches/{}/seal/", segment(batch_id));
        self.call(|token| {
            self.http
                .post(self.server.api(&path))
                .timeout(CALL_TIMEOUT)
                .bearer_auth(token)
                .json(seal)
        })
        .await
    }

    /// Opens the device stream. The response body is the event stream.
    pub async fn open_stream(&self, last_event_id: Option<&str>) -> Result<Response, ApiError> {
        let response = self
            .send(|token| {
                let mut request = self
                    .stream_http
                    .get(self.server.api("device/stream/"))
                    .bearer_auth(token)
                    .header(ACCEPT, "text/event-stream");
                if let Some(id) = last_event_id {
                    request = request.header("Last-Event-ID", id);
                }
                request
            })
            .await?;
        if response.status().is_success() {
            return Ok(response);
        }
        Err(error_from(response).await)
    }
}

fn forget(store: &Arc<dyn SecretStore>, guard: &mut MutexGuard<'_, Option<Credential>>) {
    if let Err(err) = store.clear() {
        tracing::error!(error = %err, "could not remove the saved credential");
    }
    **guard = None;
}

/// A path segment from a server ID, escaped so nothing in it can change the
/// route.
fn segment(id: &Id) -> String {
    url::form_urlencoded::byte_serialize(id.as_str().as_bytes()).collect()
}

/// The marker headers for a page, dropping any value that cannot travel as a
/// header or that the server would not record.
fn marker_headers(markers: &PageMarkers) -> HeaderMap {
    let mut headers = HeaderMap::new();
    if markers.dpi > 0 {
        headers.insert(DPI_HEADER, HeaderValue::from(markers.dpi));
    }
    if let Some(code) = markers.patch_code.as_deref().and_then(known_patch_code) {
        headers.insert(PATCH_CODE_HEADER, HeaderValue::from_static(code));
    }
    for barcode in &markers.barcodes {
        let barcode = barcode.trim();
        if barcode.is_empty()
            || barcode.len() > MAX_BARCODE_HEADER
            || !barcode.bytes().all(|b| (0x20..0x7F).contains(&b))
        {
            continue;
        }
        if let Ok(value) = HeaderValue::from_str(barcode) {
            headers.append(BARCODE_HEADER, value);
        }
    }
    headers
}

async fn decode<T: DeserializeOwned>(response: Response) -> Result<T, ApiError> {
    if response.status().is_success() {
        let body = response.bytes().await?;
        return serde_json::from_slice(&body).map_err(|e| ApiError::Decode(e.to_string()));
    }
    Err(error_from(response).await)
}

async fn error_from(response: Response) -> ApiError {
    let status = response.status();
    let retry_after = response
        .headers()
        .get(RETRY_AFTER)
        .and_then(|v| v.to_str().ok())
        .and_then(|v| v.trim().parse::<u64>().ok())
        .map(Duration::from_secs);
    let body = response.bytes().await.unwrap_or_default();
    error_for(status, &body, retry_after)
}

fn error_for(status: StatusCode, body: &[u8], retry_after: Option<Duration>) -> ApiError {
    let problem = || serde_json::from_slice::<ProblemDetail>(body).unwrap_or_default();
    match status.as_u16() {
        401 => ApiError::SignedOut,
        403 => ApiError::Denied(Box::new(problem())),
        404 => ApiError::NotFound(Box::new(problem())),
        409 => ApiError::Conflict(Box::new(problem())),
        413 => ApiError::TooLarge,
        400 | 422 => {
            let problem = problem();
            if problem.is_capture_disabled() {
                ApiError::CaptureDisabled
            } else {
                ApiError::Invalid(Box::new(problem))
            }
        }
        426 => ApiError::Outdated {
            minimum_version: serde_json::from_slice::<OutdatedAgent>(body)
                .map(|o| o.minimum_version)
                .unwrap_or_default(),
        },
        429 => ApiError::RateLimited { retry_after },
        500..=599 => ApiError::Server {
            status: status.as_u16(),
        },
        other => ApiError::Unexpected { status: other },
    }
}
