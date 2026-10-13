//! Finding a Trenova server from the address a person gave.
//!
//! People know Trenova by the address they open in a browser. On a
//! self-hosted install the API answers under that same address; on Trenova
//! Cloud, and on any deployment that serves the web app and the API from
//! different origins, it does not. So before pairing, the agent reads the
//! file the web app publishes at `/.well-known/trenova-capture.json`, which
//! names where its API is, and then asks that API to describe itself at
//! `capture/discovery/`. A server from before discovery existed still
//! answers at `capture/releases/latest/`, and is taken as a Trenova that
//! says nothing about itself.

use std::time::Duration;

use reqwest::StatusCode;
use reqwest::header::CONTENT_TYPE;
use serde::Deserialize;

use crate::api::{AgentInfo, http_client};
use crate::server::{Server, ServerUrlError};

/// Where the web app says where its API is.
pub const WELL_KNOWN_PATH: &str = ".well-known/trenova-capture.json";
/// What a Trenova names itself as, in the well-known file and at discovery.
pub const PRODUCT: &str = "trenova";
/// Discovery is one small answer; anything slower is a server that is down.
const TIMEOUT: Duration = Duration::from_secs(15);
/// The most of any answer read while discovering.
const MAX_BODY: usize = 64 << 10;

/// What a server said about itself.
#[derive(Clone, Debug, PartialEq, Eq)]
pub struct Discovery {
    /// The connection, with the API where discovery found it.
    pub server: Server,
    /// What the server calls itself: Trenova Cloud, or Trenova.
    pub name: String,
    /// The edition answering; empty from a server older than discovery.
    pub edition: String,
    /// Many organizations share this server, as on Trenova Cloud.
    pub shared_tenancy: bool,
    /// The server's version; empty from a server older than discovery.
    pub version: String,
    /// The web app, when the server names it.
    pub web_base: Option<String>,
    /// Computers can be paired here.
    pub pairing: bool,
    /// The server answered before discovery existed, so it is a Trenova
    /// whose edition and web app are not known until pairing.
    pub legacy: bool,
}

#[derive(Debug, thiserror::Error)]
pub enum DiscoveryError {
    #[error("{0}")]
    Address(#[from] ServerUrlError),
    #[error("nothing answered at {address}. Check the address and the connection, then try again.")]
    Unreachable { address: String, reason: String },
    #[error("{address} does not look like a Trenova server. Check the address.")]
    NotTrenova { address: String },
    #[error("the Trenova at {address} names an API address Trenova Capture cannot use: {reason}")]
    BadApiAddress {
        address: String,
        reason: ServerUrlError,
    },
    #[error("{address} answered with an error ({status}). Try again in a moment.")]
    Refused { address: String, status: u16 },
    #[error("{address} gave an answer Trenova Capture could not read")]
    Unreadable { address: String },
}

impl From<reqwest::Error> for DiscoveryError {
    fn from(err: reqwest::Error) -> Self {
        let address = err
            .url()
            .map(|url| url.as_str().trim_end_matches('/').to_owned())
            .unwrap_or_default();
        Self::Unreachable {
            address,
            reason: err.without_url().to_string(),
        }
    }
}

/// The well-known file as the web app writes it.
#[derive(Deserialize)]
#[serde(rename_all = "camelCase")]
struct WellKnown {
    #[serde(default)]
    product: String,
    #[serde(default)]
    api_base_url: String,
}

/// `capturehandler.Discovery`.
#[derive(Deserialize)]
#[serde(rename_all = "camelCase")]
struct DiscoveryResponse {
    #[serde(default)]
    product: String,
    #[serde(default)]
    edition: String,
    #[serde(default)]
    shared_tenancy: bool,
    #[serde(default)]
    name: String,
    #[serde(default)]
    version: String,
    #[serde(default)]
    web_base_url: String,
    #[serde(default)]
    pairing: bool,
}

/// Finds the Trenova at `entered`: where its API is, and what it says about
/// itself.
pub async fn discover(agent: &AgentInfo, entered: Server) -> Result<Discovery, DiscoveryError> {
    let http = http_client(agent, entered.api_is_https(), Some(TIMEOUT))?;
    let server = match well_known(&http, &entered).await? {
        Some(api_base) => entered.clone().with_api_base(&api_base).map_err(|reason| {
            DiscoveryError::BadApiAddress {
                address: entered.as_str().to_owned(),
                reason,
            }
        })?,
        None => entered,
    };
    describe(&http, server).await
}

async fn body(response: reqwest::Response) -> Result<(StatusCode, bool, Vec<u8>), DiscoveryError> {
    let status = response.status();
    let json = response
        .headers()
        .get(CONTENT_TYPE)
        .and_then(|value| value.to_str().ok())
        .is_some_and(|value| value.to_ascii_lowercase().contains("json"));
    let bytes = response.bytes().await?;
    let mut bytes = bytes.to_vec();
    bytes.truncate(MAX_BODY);
    Ok((status, json, bytes))
}

/// The API address the web app at `entered` publishes, if it is a Trenova
/// web app and says one. Anything else there (nothing, another site's
/// page, the app's own HTML shell for an unknown path) means the API is
/// taken to be at the entered address.
async fn well_known(
    http: &reqwest::Client,
    entered: &Server,
) -> Result<Option<String>, DiscoveryError> {
    let response = http
        .get(entered.entered_url(WELL_KNOWN_PATH))
        .send()
        .await?;
    let (status, _, bytes) = body(response).await?;
    if status != StatusCode::OK {
        return Ok(None);
    }
    let Ok(known) = serde_json::from_slice::<WellKnown>(&bytes) else {
        return Ok(None);
    };
    if known.product != PRODUCT || known.api_base_url.trim().is_empty() {
        return Ok(None);
    }
    Ok(Some(known.api_base_url))
}

async fn describe(http: &reqwest::Client, server: Server) -> Result<Discovery, DiscoveryError> {
    let address = server.api_base().to_owned();
    let response = http.get(server.api("discovery/")).send().await?;
    let (status, json, bytes) = body(response).await?;
    match status {
        StatusCode::OK => {
            let Ok(described) = serde_json::from_slice::<DiscoveryResponse>(&bytes) else {
                return Err(if json {
                    DiscoveryError::Unreadable { address }
                } else {
                    DiscoveryError::NotTrenova { address }
                });
            };
            if described.product != PRODUCT {
                return Err(DiscoveryError::NotTrenova { address });
            }
            Ok(Discovery {
                server,
                name: if described.name.trim().is_empty() {
                    "Trenova".to_owned()
                } else {
                    described.name.trim().to_owned()
                },
                edition: described.edition,
                shared_tenancy: described.shared_tenancy,
                version: described.version,
                web_base: Some(
                    described
                        .web_base_url
                        .trim()
                        .trim_end_matches('/')
                        .to_owned(),
                )
                .filter(|base| !base.is_empty()),
                pairing: described.pairing,
                legacy: false,
            })
        }
        StatusCode::NOT_FOUND => legacy(http, server).await,
        other if other.is_server_error() || other == StatusCode::TOO_MANY_REQUESTS => {
            Err(DiscoveryError::Refused {
                address,
                status: other.as_u16(),
            })
        }
        _ => Err(DiscoveryError::NotTrenova { address }),
    }
}

/// A server with no discovery route is a Trenova if it serves the release
/// route every Trenova has had: a manifest, or a problem saying there is
/// none. A plain 404 is some other web server.
async fn legacy(http: &reqwest::Client, server: Server) -> Result<Discovery, DiscoveryError> {
    let address = server.api_base().to_owned();
    let response = http.get(server.api("releases/latest/")).send().await?;
    let (status, json, _) = body(response).await?;
    let trenova = matches!(status, StatusCode::OK | StatusCode::NOT_FOUND) && json;
    if !trenova {
        return Err(DiscoveryError::NotTrenova { address });
    }
    Ok(Discovery {
        server,
        name: "Trenova".to_owned(),
        edition: String::new(),
        shared_tenancy: false,
        version: String::new(),
        web_base: None,
        pairing: true,
        legacy: true,
    })
}
