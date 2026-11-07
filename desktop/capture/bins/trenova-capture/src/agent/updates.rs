//! Keeping the companion current.
//!
//! The agent asks the server for the current release when a session starts
//! and every few hours after, verifies the manifest against the key built in
//! and that it has not expired, and compares it with its own version. A newer release is installed by the
//! updater service when both the organization (a server setting) and this
//! computer (a registry policy) allow it, and offered as a note to pass on to
//! IT when they do not. A scan in progress is never interrupted for an update;
//! the next check finds the release again.

use std::io;
use std::sync::Arc;
use std::time::Duration;

use capture_client::{Api, ApiError};
use capture_protocol::release::{Release, unix_now, verify};
use capture_update::ed25519_dalek::VerifyingKey;
use tokio::sync::mpsc;
use tokio_util::sync::CancellationToken;

/// How often a signed-in agent looks for a release.
pub const CHECK_EVERY: Duration = Duration::from_hours(6);

/// Starts the updater service, which does the installing.
pub trait UpdateStarter: Send + Sync {
    /// Asks for an update from the manifest at `manifest_url`. The updater
    /// verifies everything itself; this only wakes it.
    fn start(&self, manifest_url: &str) -> io::Result<()>;
}

/// Where the updater reads the manifest: the server's public mirror of it.
pub fn manifest_url(api: &Api) -> String {
    api.server().api("releases/latest/").to_string()
}

/// The current release the server publishes, verified. A manifest that does
/// not verify, or has expired, is logged and treated as no release: a server
/// that serves a bad or stale manifest gets no say over what is installed,
/// and the installed release carries on as it is.
pub async fn check(api: &Api, key: Option<&VerifyingKey>) -> Result<Option<Release>, ApiError> {
    let Some(key) = key else {
        return Ok(None);
    };
    let Some(signed) = api.latest_release().await? else {
        return Ok(None);
    };
    let release = match verify(&signed, key) {
        Ok(release) => release,
        Err(err) => {
            tracing::warn!(error = %err, "the server's release manifest does not verify");
            return Ok(None);
        }
    };
    if let Err(err) = release.check_current(unix_now()) {
        tracing::warn!(
            error = %err,
            version = %release.version,
            "the server's release manifest is out of date; updates are paused until a current one is served"
        );
        return Ok(None);
    }
    Ok(Some(release))
}

/// Checks now and then every [`CHECK_EVERY`] until `cancel`.
pub async fn watch(
    api: Arc<Api>,
    key: Option<VerifyingKey>,
    report: mpsc::Sender<Result<Option<Release>, ApiError>>,
    cancel: CancellationToken,
) {
    loop {
        if report.send(check(&api, key.as_ref()).await).await.is_err() {
            return;
        }
        tokio::select! {
            () = cancel.cancelled() => return,
            () = tokio::time::sleep(CHECK_EVERY) => {}
        }
    }
}
