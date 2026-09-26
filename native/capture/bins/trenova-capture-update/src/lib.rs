//! The Trenova Capture updater.
//!
//! Installing an MSI needs rights the tray agent does not have, so the agent
//! asks this service to do it: it is started on demand with one argument, the
//! address of a release manifest. The updater trusts nothing about that
//! request. It reads the manifest itself over HTTPS, verifies the signature
//! against the key built into it, decides against its own version, downloads
//! the installer and checks its size and SHA-256, and then, on Windows, has
//! the system verify the installer's Authenticode signature and checks that
//! the signer is the same publisher as the running updater. Only then does it
//! run the installer. An unprivileged process can therefore make this
//! computer install a genuine, newer Trenova Capture release, and nothing
//! else.
//!
//! The flow is here so it runs, and is tested, anywhere; what needs Windows
//! ([`Installer`]) is in the binary.

use std::path::{Path, PathBuf};
use std::time::Duration;

use capture_protocol::release::{Release, SignedRelease, verify};
use capture_update::ed25519_dalek::VerifyingKey;
use capture_update::{Decision, UpdateError, decide, download, fetch_signed};
use url::Url;

/// The Windows service's name.
pub const SERVICE_NAME: &str = "TrenovaCaptureUpdater";
pub const SERVICE_DISPLAY_NAME: &str = "Trenova Capture updater";
pub const SERVICE_DESCRIPTION: &str = "Installs new releases of Trenova Capture after verifying them. Started by Trenova Capture when an update is due.";
/// How long a fetch or download may take.
pub const NETWORK_TIMEOUT: Duration = Duration::from_secs(600);
const MAX_URL_LENGTH: usize = 2048;

#[derive(Debug, thiserror::Error)]
pub enum UpdaterError {
    #[error("the manifest address is not an https URL")]
    Address,
    #[error("this build has no release key and cannot verify an update")]
    NoKey,
    #[error("no release is published")]
    NoRelease,
    #[error("the installed {installed} is already current ({available} is published)")]
    UpToDate {
        installed: String,
        available: String,
    },
    #[error("release {version} needs Windows build {needs}, and this is {have}")]
    WindowsTooOld {
        version: String,
        needs: u32,
        have: u32,
    },
    #[error("updates are turned off on this computer")]
    PolicyOff,
    #[error(transparent)]
    Update(#[from] UpdateError),
    #[error("the installer is not a Trenova release: {0}")]
    Untrusted(String),
    #[error("the installer failed: {0}")]
    Install(String),
    #[error(transparent)]
    Io(#[from] std::io::Error),
}

/// Where the manifest and the installer come from. [`Http`] is the real one;
/// a test supplies bytes.
pub trait Source: Send + Sync {
    fn fetch_manifest(
        &self,
        url: &Url,
    ) -> impl Future<Output = Result<Option<SignedRelease>, UpdateError>> + Send;
    /// Downloads the release's installer into `dir`, checked against the
    /// manifest, and returns its path.
    fn download(
        &self,
        release: &Release,
        dir: &Path,
    ) -> impl Future<Output = Result<PathBuf, UpdateError>> + Send;
}

/// The network, over HTTPS only.
#[derive(Clone, Debug)]
pub struct Http {
    client: reqwest::Client,
}

impl Http {
    pub fn new() -> Result<Self, UpdateError> {
        let client = reqwest::Client::builder()
            .timeout(NETWORK_TIMEOUT)
            .https_only(true)
            .build()
            .map_err(UpdateError::Fetch)?;
        Ok(Self { client })
    }
}

impl Source for Http {
    async fn fetch_manifest(&self, url: &Url) -> Result<Option<SignedRelease>, UpdateError> {
        fetch_signed(&self.client, url.as_str()).await
    }

    async fn download(&self, release: &Release, dir: &Path) -> Result<PathBuf, UpdateError> {
        download(&self.client, release, dir).await
    }
}

/// What the updater needs from where it runs.
pub trait Installer: Send + Sync {
    /// The version installed now.
    fn installed_version(&self) -> String;
    /// The Windows build, as `RtlGetVersion` reports it.
    fn windows_build(&self) -> u32;
    /// Whether this computer's policy allows updates at all. The
    /// organization's say is the agent's to enforce; this is IT's, and the
    /// updater enforces it so an agent cannot talk it round.
    fn policy_allows(&self) -> bool;
    /// Where installers are downloaded to: a directory only the updater can
    /// write.
    fn download_dir(&self) -> Result<PathBuf, UpdaterError>;
    /// Checks the installer's own signature and publisher.
    fn verify_installer(&self, path: &Path) -> Result<(), UpdaterError>;
    /// Runs the installer and waits for it.
    fn install(&self, path: &Path) -> Result<(), UpdaterError>;
    /// Starts the agent again for the people signed in, once it is updated.
    fn relaunch_agents(&self) -> Result<(), UpdaterError>;
}

/// Only an HTTPS address is followed. `http://localhost` is not accepted
/// either: the updater runs as the system, and a development server is a
/// per-user setting.
pub fn manifest_url(value: &str) -> Result<Url, UpdaterError> {
    if value.len() > MAX_URL_LENGTH {
        return Err(UpdaterError::Address);
    }
    let url = Url::parse(value).map_err(|_| UpdaterError::Address)?;
    if url.scheme() != "https" || url.host_str().is_none() {
        return Err(UpdaterError::Address);
    }
    Ok(url)
}

/// What the run found and did.
#[derive(Clone, Debug, PartialEq, Eq)]
pub struct Installed {
    pub from: String,
    pub to: String,
}

/// Runs one update: everything in the module doc, in that order. `manifest`
/// has passed [`manifest_url`].
pub async fn run(
    manifest: &Url,
    key: Option<&VerifyingKey>,
    source: &impl Source,
    installer: &dyn Installer,
) -> Result<Installed, UpdaterError> {
    let key = key.ok_or(UpdaterError::NoKey)?;
    if !installer.policy_allows() {
        return Err(UpdaterError::PolicyOff);
    }

    let signed = source
        .fetch_manifest(manifest)
        .await?
        .ok_or(UpdaterError::NoRelease)?;
    let release = verify(&signed, key).map_err(UpdateError::from)?;
    let current = installer.installed_version();
    match decide(&current, &release, installer.windows_build()) {
        Decision::Install => {}
        Decision::UpToDate => {
            return Err(UpdaterError::UpToDate {
                installed: current,
                available: release.version,
            });
        }
        Decision::WindowsTooOld { needs } => {
            return Err(UpdaterError::WindowsTooOld {
                version: release.version,
                needs,
                have: installer.windows_build(),
            });
        }
    }
    tracing::info!(from = %current, to = %release.version, "installing a new release");

    let dir = installer.download_dir()?;
    let path = source.download(&release, &dir).await?;
    let outcome = async {
        installer.verify_installer(&path)?;
        installer.install(&path)
    }
    .await;
    let _ = tokio::fs::remove_file(&path).await;
    outcome?;

    if let Err(err) = installer.relaunch_agents() {
        tracing::warn!(error = %err, "installed, but could not start Trenova Capture again");
    }
    Ok(Installed {
        from: current,
        to: release.version,
    })
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn only_https_addresses_are_followed() {
        assert!(manifest_url("https://app.acme.test/api/v1/capture/releases/latest/").is_ok());
        for bad in [
            "http://app.acme.test/api/v1/capture/releases/latest/",
            "http://localhost:8080/x",
            "file:///C:/manifest.json",
            "https://",
            "not a url",
        ] {
            assert!(
                matches!(manifest_url(bad), Err(UpdaterError::Address)),
                "{bad}"
            );
        }
    }
}
