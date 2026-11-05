//! The updater's flow with a scripted network and installer: what it checks,
//! in what order, and what it refuses.

use std::future::pending;
use std::path::{Path, PathBuf};
use std::sync::Mutex;

use capture_protocol::release::{
    Installer as ManifestInstaller, PRODUCT, Release, SignedRelease, encode_key_pair, public_key,
    sign, signing_key,
};
use capture_update::UpdateError;
use capture_update::ed25519_dalek::{SigningKey, VerifyingKey};
use sha2::{Digest, Sha256};
use tokio::sync::watch;
use trenova_capture_update::{
    Installed, Installer, Source, UpdaterError, manifest_url, run as run_until,
};
use url::Url;

const MSI: &[u8] = b"a genuine installer";

fn keys(seed: u8) -> (SigningKey, VerifyingKey) {
    let (private, public) = encode_key_pair([seed; 32]);
    (
        signing_key(&private).expect("private"),
        public_key(&public).expect("public"),
    )
}

fn release(version: &str, minimum_windows_build: u32) -> Release {
    Release {
        product: PRODUCT.into(),
        version: version.into(),
        published_at: 1_780_000_000,
        minimum_windows_build,
        installer: ManifestInstaller {
            file_name: format!("TrenovaCapture-{version}-x64.msi"),
            url: format!("https://releases.example.test/TrenovaCapture-{version}-x64.msi"),
            sha256: hex::encode(Sha256::digest(MSI)),
            size: MSI.len() as u64,
        },
        notes: String::new(),
    }
}

/// A network that serves one manifest and one installer.
struct Scripted {
    manifest: Option<SignedRelease>,
    bytes: Vec<u8>,
}

impl Source for Scripted {
    async fn fetch_manifest(&self, _url: &Url) -> Result<Option<SignedRelease>, UpdateError> {
        Ok(self.manifest.clone())
    }

    async fn download(&self, release: &Release, dir: &Path) -> Result<PathBuf, UpdateError> {
        if hex::encode(Sha256::digest(&self.bytes)) != release.installer.sha256 {
            return Err(UpdateError::Checksum);
        }
        let path = dir.join(&release.installer.file_name);
        std::fs::write(&path, &self.bytes)?;
        Ok(path)
    }
}

#[derive(Default)]
struct Fake {
    installed: String,
    build: u32,
    policy_off: bool,
    dir: Option<tempfile::TempDir>,
    untrusted: bool,
    calls: Mutex<Vec<String>>,
}

impl Fake {
    fn record(&self, what: &str) {
        self.calls.lock().expect("lock").push(what.to_owned());
    }
}

impl Installer for Fake {
    fn installed_version(&self) -> String {
        self.installed.clone()
    }

    fn windows_build(&self) -> u32 {
        self.build
    }

    fn policy_allows(&self) -> bool {
        !self.policy_off
    }

    fn download_dir(&self) -> Result<PathBuf, UpdaterError> {
        Ok(self.dir.as_ref().expect("a dir").path().to_path_buf())
    }

    fn verify_installer(&self, path: &Path) -> Result<(), UpdaterError> {
        self.record("verify");
        assert_eq!(std::fs::read(path).expect("reads"), MSI);
        if self.untrusted {
            return Err(UpdaterError::Untrusted("signed by somebody else".into()));
        }
        Ok(())
    }

    fn install(&self, path: &Path) -> Result<(), UpdaterError> {
        self.record("install");
        assert!(path.exists(), "the installer is still there while it runs");
        Ok(())
    }

    fn relaunch_agents(&self) -> Result<(), UpdaterError> {
        self.record("relaunch");
        Ok(())
    }
}

fn fake(installed: &str) -> Fake {
    Fake {
        installed: installed.into(),
        build: 22631,
        dir: Some(tempfile::tempdir().expect("dir")),
        ..Fake::default()
    }
}

fn url() -> Url {
    manifest_url("https://app.acme.test/api/v1/capture/releases/latest/").expect("url")
}

/// One update nobody stops.
async fn run(
    manifest: &Url,
    key: Option<&VerifyingKey>,
    source: &impl Source,
    installer: &dyn Installer,
) -> Result<Installed, UpdaterError> {
    let (_never, stop) = watch::channel(false);
    run_until(manifest, key, source, installer, &stop, || {}).await
}

/// A manifest that never arrives.
struct NeverAnswers;

impl Source for NeverAnswers {
    async fn fetch_manifest(&self, _url: &Url) -> Result<Option<SignedRelease>, UpdateError> {
        pending().await
    }

    async fn download(&self, _release: &Release, _dir: &Path) -> Result<PathBuf, UpdateError> {
        pending().await
    }
}

/// The service is told to stop partway through the download, which then
/// hangs, or finishes when `finishes` is set.
struct StoppedDuringDownload {
    manifest: SignedRelease,
    stop: watch::Sender<bool>,
    finishes: bool,
}

impl Source for StoppedDuringDownload {
    async fn fetch_manifest(&self, _url: &Url) -> Result<Option<SignedRelease>, UpdateError> {
        Ok(Some(self.manifest.clone()))
    }

    async fn download(&self, release: &Release, dir: &Path) -> Result<PathBuf, UpdateError> {
        self.stop.send_replace(true);
        if !self.finishes {
            return pending().await;
        }
        let path = dir.join(&release.installer.file_name);
        std::fs::write(&path, MSI)?;
        Ok(path)
    }
}

#[tokio::test]
async fn a_newer_signed_release_is_verified_installed_and_the_agent_restarted() {
    let (private, public) = keys(5);
    let source = Scripted {
        manifest: Some(sign(&release("1.5.0", 19045), &private).expect("signs")),
        bytes: MSI.to_vec(),
    };
    let installer = fake("1.4.2");

    let outcome = run(&url(), Some(&public), &source, &installer)
        .await
        .expect("installs");
    assert_eq!(outcome.from, "1.4.2");
    assert_eq!(outcome.to, "1.5.0");
    assert_eq!(
        *installer.calls.lock().expect("lock"),
        ["verify", "install", "relaunch"]
    );
    assert!(
        std::fs::read_dir(installer.dir.as_ref().expect("dir").path())
            .expect("lists")
            .next()
            .is_none(),
        "the installer is removed afterwards"
    );
}

#[tokio::test]
async fn the_service_stops_answering_to_stop_just_before_it_installs() {
    let (private, public) = keys(5);
    let source = Scripted {
        manifest: Some(sign(&release("1.5.0", 19045), &private).expect("signs")),
        bytes: MSI.to_vec(),
    };
    let installer = fake("1.4.2");
    let (_never, stop) = watch::channel(false);

    run_until(&url(), Some(&public), &source, &installer, &stop, || {
        installer.record("committing");
    })
    .await
    .expect("installs");

    assert_eq!(
        *installer.calls.lock().expect("lock"),
        ["verify", "committing", "install", "relaunch"]
    );
}

#[tokio::test]
async fn a_stop_ends_a_manifest_fetch_that_never_answers() {
    let (_, public) = keys(5);
    let installer = fake("1.4.2");
    let (asked, stop) = watch::channel(false);
    let unblock = async {
        tokio::task::yield_now().await;
        asked.send_replace(true);
    };

    let (result, ()) = tokio::join!(
        run_until(
            &url(),
            Some(&public),
            &NeverAnswers,
            &installer,
            &stop,
            || {
                installer.record("committing");
            }
        ),
        unblock,
    );

    assert!(matches!(result, Err(UpdaterError::Stopped)), "{result:?}");
    assert!(installer.calls.lock().expect("lock").is_empty());
}

#[tokio::test]
async fn a_stop_ends_a_download_that_hangs() {
    let (private, public) = keys(5);
    let (asked, stop) = watch::channel(false);
    let source = StoppedDuringDownload {
        manifest: sign(&release("1.5.0", 19045), &private).expect("signs"),
        stop: asked,
        finishes: false,
    };
    let installer = fake("1.4.2");

    let result = run_until(&url(), Some(&public), &source, &installer, &stop, || {
        installer.record("committing");
    })
    .await;

    assert!(matches!(result, Err(UpdaterError::Stopped)), "{result:?}");
    assert!(installer.calls.lock().expect("lock").is_empty());
}

#[tokio::test]
async fn a_stop_that_arrives_before_the_install_installs_nothing() {
    let (private, public) = keys(5);
    let (asked, stop) = watch::channel(false);
    let source = StoppedDuringDownload {
        manifest: sign(&release("1.5.0", 19045), &private).expect("signs"),
        stop: asked,
        finishes: true,
    };
    let installer = fake("1.4.2");

    let result = run_until(&url(), Some(&public), &source, &installer, &stop, || {
        installer.record("committing");
    })
    .await;

    assert!(matches!(result, Err(UpdaterError::Stopped)), "{result:?}");
    assert_eq!(*installer.calls.lock().expect("lock"), ["verify"]);
    assert!(
        std::fs::read_dir(installer.dir.as_ref().expect("dir").path())
            .expect("lists")
            .next()
            .is_none(),
        "the download is removed"
    );
}

#[tokio::test]
async fn nothing_is_installed_without_a_key_a_release_or_a_reason() {
    let (private, public) = keys(5);
    let current = Scripted {
        manifest: Some(sign(&release("1.5.0", 19045), &private).expect("signs")),
        bytes: MSI.to_vec(),
    };

    let no_key = run(&url(), None, &current, &fake("1.4.2")).await;
    assert!(matches!(no_key, Err(UpdaterError::NoKey)));

    let none = Scripted {
        manifest: None,
        bytes: Vec::new(),
    };
    assert!(matches!(
        run(&url(), Some(&public), &none, &fake("1.4.2")).await,
        Err(UpdaterError::NoRelease)
    ));

    let same = run(&url(), Some(&public), &current, &fake("1.5.0")).await;
    assert!(
        matches!(same, Err(UpdaterError::UpToDate { .. })),
        "{same:?}"
    );

    let newer_installed = run(&url(), Some(&public), &current, &fake("1.6.0")).await;
    assert!(matches!(
        newer_installed,
        Err(UpdaterError::UpToDate { .. })
    ));

    let off = Fake {
        policy_off: true,
        ..fake("1.4.2")
    };
    assert!(matches!(
        run(&url(), Some(&public), &current, &off).await,
        Err(UpdaterError::PolicyOff)
    ));
    assert!(off.calls.lock().expect("lock").is_empty());

    let old_windows = Scripted {
        manifest: Some(sign(&release("1.5.0", 26100), &private).expect("signs")),
        bytes: MSI.to_vec(),
    };
    let too_old = run(&url(), Some(&public), &old_windows, &fake("1.4.2")).await;
    assert!(matches!(
        too_old,
        Err(UpdaterError::WindowsTooOld { needs: 26100, .. })
    ));
}

#[tokio::test]
async fn a_forged_manifest_a_wrong_download_or_a_foreign_signer_stops_before_installing() {
    let (private, public) = keys(5);
    let (other_key, _) = keys(9);

    let forged = Scripted {
        manifest: Some(sign(&release("1.5.0", 19045), &other_key).expect("signs")),
        bytes: MSI.to_vec(),
    };
    let installer = fake("1.4.2");
    let result = run(&url(), Some(&public), &forged, &installer).await;
    assert!(
        matches!(result, Err(UpdaterError::Update(UpdateError::Release(_)))),
        "{result:?}"
    );
    assert!(installer.calls.lock().expect("lock").is_empty());

    let tampered = Scripted {
        manifest: Some(sign(&release("1.5.0", 19045), &private).expect("signs")),
        bytes: b"something else".to_vec(),
    };
    let installer = fake("1.4.2");
    let result = run(&url(), Some(&public), &tampered, &installer).await;
    assert!(
        matches!(result, Err(UpdaterError::Update(UpdateError::Checksum))),
        "{result:?}"
    );
    assert!(installer.calls.lock().expect("lock").is_empty());

    let genuine = Scripted {
        manifest: Some(sign(&release("1.5.0", 19045), &private).expect("signs")),
        bytes: MSI.to_vec(),
    };
    let foreign = Fake {
        untrusted: true,
        ..fake("1.4.2")
    };
    let result = run(&url(), Some(&public), &genuine, &foreign).await;
    assert!(
        matches!(result, Err(UpdaterError::Untrusted(_))),
        "{result:?}"
    );
    assert_eq!(*foreign.calls.lock().expect("lock"), ["verify"]);
    assert!(
        std::fs::read_dir(foreign.dir.as_ref().expect("dir").path())
            .expect("lists")
            .next()
            .is_none(),
        "a refused installer is removed"
    );
}
