//! Pairing this computer with a person: RFC 8628, from the device's side.

use std::time::Duration;

use capture_protocol::api::{PairingGrant, StartPairingRequest};
use tokio::time::Instant;
use tokio_util::sync::CancellationToken;

use crate::api::{Api, PairingPoll};
use crate::credentials::Credential;
use crate::error::ApiError;
use crate::server::web_base_from_verification;

/// RFC 8628 §3.5: a `slow_down` answer adds five seconds to the interval.
const SLOW_DOWN_STEP: Duration = Duration::from_secs(5);
/// Never poll faster than this, whatever the grant says.
const MIN_INTERVAL: Duration = Duration::from_secs(1);

/// How pairing ended.
#[derive(Clone, Debug, PartialEq, Eq)]
pub enum PairingOutcome {
    /// Approved; the credential is saved and in use.
    Paired(Box<Credential>),
    /// The person said no.
    Denied,
    /// Nobody approved it in time.
    Expired,
    /// Stopped from this side.
    Canceled,
}

/// Runs a pairing: asks for a code, hands the grant to `show` (to display the
/// code and open the approval page), then polls until the person decides.
pub async fn pair(
    api: &Api,
    machine: &StartPairingRequest,
    show: impl FnOnce(&PairingGrant),
    cancel: &CancellationToken,
) -> Result<PairingOutcome, ApiError> {
    let grant = api.start_pairing(machine).await?;
    let web_base = web_base_from_verification(&grant.verification_uri).ok_or_else(|| {
        ApiError::Decode(format!(
            "an approval address of {:?}",
            grant.verification_uri
        ))
    })?;
    show(&grant);

    let lifetime = Duration::from_secs(u64::try_from(grant.expires_in).unwrap_or(0));
    let deadline = Instant::now() + lifetime;
    let mut interval =
        Duration::from_secs(u64::try_from(grant.interval).unwrap_or(5)).max(MIN_INTERVAL);

    loop {
        tokio::select! {
            () = cancel.cancelled() => return Ok(PairingOutcome::Canceled),
            () = tokio::time::sleep(interval) => {}
        }
        if Instant::now() >= deadline {
            return Ok(PairingOutcome::Expired);
        }
        match api.poll_pairing(&grant.device_code).await {
            Ok(PairingPoll::Pending) => {}
            Ok(PairingPoll::SlowDown) => interval += SLOW_DOWN_STEP,
            Ok(PairingPoll::Denied) => return Ok(PairingOutcome::Denied),
            Ok(PairingPoll::Expired) => return Ok(PairingOutcome::Expired),
            Ok(PairingPoll::Approved(tokens)) => {
                let credential = Credential {
                    server: api.server().as_str().to_owned(),
                    web_base,
                    tokens,
                };
                api.sign_in(credential.clone()).await?;
                return Ok(PairingOutcome::Paired(Box::new(credential)));
            }
            Err(err) if err.is_retryable() => {
                tracing::warn!(error = %err, "pairing poll failed; trying again");
                if let Some(wait) = err.retry_after() {
                    interval = interval.max(wait);
                }
            }
            Err(err) => return Err(err),
        }
    }
}
