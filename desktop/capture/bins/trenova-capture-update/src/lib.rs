//! The Trenova Capture updater.
//!
//! Installing an MSI needs rights the tray agent does not have, so the agent
//! asks this service to do it: it is started on demand with one argument, the
//! address of a release manifest. The updater trusts nothing about that
//! request, and anyone signed in can make it. So the address must be a
//! server's manifest route on a host off this machine, and a redirect may not
//! leave that origin: the system's network position answers for one fixed
//! route and nothing else. It reads the manifest itself, verifies the signature
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

use capture_protocol::release::{MANIFEST_ROUTE, Release, SignedRelease, verify};
use capture_update::ed25519_dalek::VerifyingKey;
use capture_update::{Decision, UpdateError, decide, download, fetch_signed};
use tokio::sync::watch;
use url::{Host, Url};

/// The Windows service's name.
pub const SERVICE_NAME: &str = "TrenovaCaptureUpdater";
pub const SERVICE_DISPLAY_NAME: &str = "Trenova Capture updater";
pub const SERVICE_DESCRIPTION: &str = "Installs new releases of Trenova Capture after verifying them. Started by Trenova Capture when an update is due.";
/// How long a download may take.
pub const NETWORK_TIMEOUT: Duration = Duration::from_mins(10);
/// How long reading a manifest may take. It is at most 64 KiB.
const MANIFEST_TIMEOUT: Duration = Duration::from_mins(1);
const MAX_URL_LENGTH: usize = 2048;
const MAX_MANIFEST_REDIRECTS: usize = 5;

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
    #[error("the update was stopped before it installed anything")]
    Stopped,
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

/// The network, over HTTPS only. The manifest's address comes from whoever
/// started the service, so its client stays on that origin; the installer's
/// is signed, and a release host may send it on to a CDN.
#[derive(Clone, Debug)]
pub struct Http {
    manifests: reqwest::Client,
    installers: reqwest::Client,
}

/// Whether a redirect stays where the manifest's address pointed.
fn same_origin(first: &Url, next: &Url) -> bool {
    first.origin() == next.origin()
}

impl Http {
    pub fn new() -> Result<Self, UpdateError> {
        let same_origin_only = reqwest::redirect::Policy::custom(|attempt| {
            let allowed = attempt.previous().len() < MAX_MANIFEST_REDIRECTS
                && attempt
                    .previous()
                    .first()
                    .is_some_and(|first| same_origin(first, attempt.url()));
            if allowed {
                attempt.follow()
            } else {
                attempt.stop()
            }
        });
        let manifests = reqwest::Client::builder()
            .timeout(MANIFEST_TIMEOUT)
            .https_only(true)
            .redirect(same_origin_only)
            .build()
            .map_err(UpdateError::Fetch)?;
        let installers = reqwest::Client::builder()
            .timeout(NETWORK_TIMEOUT)
            .https_only(true)
            .build()
            .map_err(UpdateError::Fetch)?;
        Ok(Self {
            manifests,
            installers,
        })
    }
}

impl Source for Http {
    async fn fetch_manifest(&self, url: &Url) -> Result<Option<SignedRelease>, UpdateError> {
        fetch_signed(&self.manifests, url.as_str()).await
    }

    async fn download(&self, release: &Release, dir: &Path) -> Result<PathBuf, UpdateError> {
        download(&self.installers, release, dir).await
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

/// Whether a host is this machine or its link, which no server is reached at
/// from the system account. Other private addresses stay allowed: a server
/// hosted inside a company's network is an ordinary one.
fn this_machine(host: &Host<&str>) -> bool {
    match host {
        Host::Domain(domain) => domain
            .trim_end_matches('.')
            .eq_ignore_ascii_case("localhost"),
        Host::Ipv4(ip) => ip.is_loopback() || ip.is_link_local() || ip.is_unspecified(),
        Host::Ipv6(ip) => ip.is_loopback() || ip.is_unicast_link_local() || ip.is_unspecified(),
    }
}

/// Only a server's manifest route over HTTPS is followed, with nothing added
/// to it. `http://localhost` is not accepted either: the updater runs as the
/// system, and a development server is a per-user setting.
pub fn manifest_url(value: &str) -> Result<Url, UpdaterError> {
    if value.len() > MAX_URL_LENGTH {
        return Err(UpdaterError::Address);
    }
    let url = Url::parse(value).map_err(|_| UpdaterError::Address)?;
    let plain = url.scheme() == "https"
        && url.username().is_empty()
        && url.password().is_none()
        && url.query().is_none()
        && url.fragment().is_none()
        && url.path().ends_with(MANIFEST_ROUTE);
    if !plain || url.host().is_none_or(|host| this_machine(&host)) {
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

/// Turns true when the service is told to stop. A sender that goes away is
/// not a request to stop.
pub type Stop = watch::Receiver<bool>;

async fn stopped(stop: &mut Stop) {
    if stop.wait_for(|stopped| *stopped).await.is_err() {
        std::future::pending::<()>().await;
    }
}

/// Runs `work`, giving it up as soon as the service is told to stop.
async fn until_stopped<T>(
    stop: &Stop,
    work: impl Future<Output = Result<T, UpdaterError>>,
) -> Result<T, UpdaterError> {
    let mut stop = stop.clone();
    tokio::select! {
        biased;
        () = stopped(&mut stop) => Err(UpdaterError::Stopped),
        outcome = work => outcome,
    }
}

/// Runs one update: everything in the module doc, in that order. `manifest`
/// has passed [`manifest_url`].
///
/// `stop` ends the run at any point up to the install. `committing` is called
/// just before it, the last moment that can happen: the installer replaces
/// and restarts this service, so a service that could still be stopped would
/// be stopped by its own upgrade.
pub async fn run(
    manifest: &Url,
    key: Option<&VerifyingKey>,
    source: &impl Source,
    installer: &dyn Installer,
    stop: &Stop,
    committing: impl FnOnce(),
) -> Result<Installed, UpdaterError> {
    let key = key.ok_or(UpdaterError::NoKey)?;
    if !installer.policy_allows() {
        return Err(UpdaterError::PolicyOff);
    }

    let fetched = async {
        source
            .fetch_manifest(manifest)
            .await
            .map_err(UpdaterError::from)
    };
    let signed = until_stopped(stop, fetched)
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
    let downloaded = async {
        source
            .download(&release, &dir)
            .await
            .map_err(UpdaterError::from)
    };
    let path = until_stopped(stop, downloaded).await?;
    let outcome = installer.verify_installer(&path).and_then(|()| {
        if *stop.borrow() {
            return Err(UpdaterError::Stopped);
        }
        committing();
        installer.install(&path)
    });
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
        for good in [
            "https://app.acme.test/api/v1/capture/releases/latest/",
            "https://app.acme.test:8443/api/v1/capture/releases/latest/",
            "https://acme.test/tms/api/v1/capture/releases/latest/",
            "https://10.20.30.40/api/v1/capture/releases/latest/",
        ] {
            assert!(manifest_url(good).is_ok(), "{good}");
        }
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

    #[test]
    fn only_the_manifest_route_is_followed() {
        for bad in [
            "https://app.acme.test/",
            "https://app.acme.test/api/v1/users/",
            "https://app.acme.test/api/v1/capture/releases/latest",
            "https://app.acme.test/api/v1/capture/releases/latest/x",
            "https://app.acme.test/api/v1/capture/releases/latest/?a=1",
            "https://app.acme.test/api/v1/capture/releases/latest/#a",
            "https://me:secret@app.acme.test/api/v1/capture/releases/latest/",
        ] {
            assert!(
                matches!(manifest_url(bad), Err(UpdaterError::Address)),
                "{bad}"
            );
        }
    }

    #[test]
    fn this_machine_and_its_link_are_not_servers() {
        for host in [
            "localhost",
            "LOCALHOST",
            "localhost.",
            "127.0.0.1",
            "127.1.2.3",
            "0.0.0.0",
            "169.254.169.254",
            "[::1]",
            "[fe80::1]",
        ] {
            let address = format!("https://{host}/api/v1/capture/releases/latest/");
            assert!(
                matches!(manifest_url(&address), Err(UpdaterError::Address)),
                "{address}"
            );
        }
    }

    #[test]
    fn a_redirect_may_not_leave_the_origin_it_started_on() {
        let first = Url::parse("https://app.acme.test/a").expect("url");
        let same = Url::parse("https://app.acme.test/b").expect("url");
        assert!(same_origin(&first, &same));
        for elsewhere in [
            "https://other.test/a",
            "https://app.acme.test:8443/a",
            "http://app.acme.test/a",
        ] {
            let next = Url::parse(elsewhere).expect("url");
            assert!(!same_origin(&first, &next), "{elsewhere}");
        }
    }
}
