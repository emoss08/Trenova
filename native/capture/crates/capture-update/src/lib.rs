//! Getting from "a release is published" to "its installer is on disk and
//! is what the manifest said", for the agent and the updater alike.
//!
//! The agent uses [`decide`] to know whether to offer or start an update; the
//! updater, which runs with the rights to install, does not take the agent's
//! word for any of it and repeats every check: it reads the manifest itself,
//! verifies the signature against the key built into it, decides again with
//! its own version, and [`download`]s the installer, refusing anything whose
//! size or SHA-256 differs from the manifest. Whether the file is a
//! Trenova-signed installer is Windows' question, asked afterwards.

use std::path::{Path, PathBuf};

use capture_protocol::release::{
    MAX_SIGNED_BYTES, Release, ReleaseError, SignedRelease, Version, is_newer, verify,
};
use ed25519_dalek::VerifyingKey;
use futures_util::StreamExt;
use reqwest::{Client, StatusCode};
use sha2::{Digest, Sha256};
use tokio::io::AsyncWriteExt;

pub use ed25519_dalek;

#[derive(Debug, thiserror::Error)]
pub enum UpdateError {
    #[error("the release manifest could not be fetched: {0}")]
    Fetch(#[source] reqwest::Error),
    #[error("the release manifest could not be fetched: status {0}")]
    FetchStatus(u16),
    #[error("the release manifest is not what a manifest looks like: {0}")]
    Manifest(String),
    #[error(transparent)]
    Release(#[from] ReleaseError),
    #[error("the installer could not be downloaded: {0}")]
    Download(#[source] reqwest::Error),
    #[error("the installer could not be downloaded: status {0}")]
    DownloadStatus(u16),
    #[error("the download is {actual} bytes but the manifest says {expected}")]
    Size { expected: u64, actual: u64 },
    #[error("the download's checksum does not match the manifest")]
    Checksum,
    #[error("the installer could not be saved: {0}")]
    Io(#[from] std::io::Error),
}

/// What a release means for this installation.
#[derive(Clone, Debug, PartialEq, Eq)]
pub enum Decision {
    /// Newer than what is installed, and this Windows can run it.
    Install,
    /// The same, or older, than what is installed.
    UpToDate,
    /// Newer, but needs a newer Windows than this one.
    WindowsTooOld { needs: u32 },
}

/// Whether `release` should be installed over `installed` on Windows build
/// `windows_build`. A version that cannot be read is never installed over.
pub fn decide(installed: &str, release: &Release, windows_build: u32) -> Decision {
    if !is_newer(&release.version, installed) {
        return Decision::UpToDate;
    }
    if windows_build < release.minimum_windows_build {
        return Decision::WindowsTooOld {
            needs: release.minimum_windows_build,
        };
    }
    Decision::Install
}

/// Reads the signed manifest at `url`, unverified. `None` when none is
/// published (404).
pub async fn fetch_signed(
    client: &Client,
    url: &str,
) -> Result<Option<SignedRelease>, UpdateError> {
    let response = client.get(url).send().await.map_err(UpdateError::Fetch)?;
    if response.status() == StatusCode::NOT_FOUND {
        return Ok(None);
    }
    if !response.status().is_success() {
        return Err(UpdateError::FetchStatus(response.status().as_u16()));
    }
    let body = bounded_body(response, MAX_SIGNED_BYTES)
        .await
        .map_err(UpdateError::Fetch)?
        .ok_or_else(|| UpdateError::Manifest("it is larger than any manifest".into()))?;
    serde_json::from_slice(&body)
        .map(Some)
        .map_err(|e| UpdateError::Manifest(e.to_string()))
}

/// Reads and verifies the manifest at `url`. `None` when none is published.
/// Nothing about the URL is trusted: only a manifest the key signed comes
/// back, and only after [`Release::validate`] passed.
pub async fn fetch_release(
    client: &Client,
    url: &str,
    key: &VerifyingKey,
) -> Result<Option<Release>, UpdateError> {
    match fetch_signed(client, url).await? {
        Some(signed) => Ok(Some(verify(&signed, key)?)),
        None => Ok(None),
    }
}

/// Reads a body of at most `limit` bytes; `None` when it is larger.
async fn bounded_body(
    response: reqwest::Response,
    limit: usize,
) -> reqwest::Result<Option<Vec<u8>>> {
    let mut body = Vec::new();
    let mut stream = response.bytes_stream();
    while let Some(chunk) = stream.next().await {
        let chunk = chunk?;
        if body.len() + chunk.len() > limit {
            return Ok(None);
        }
        body.extend_from_slice(&chunk);
    }
    Ok(Some(body))
}

/// Downloads the release's installer into `dir`, checking its size and
/// SHA-256 against the manifest as it arrives. Returns the file's path. The
/// file is written under a `.part` name and renamed only once it checks out,
/// so a file with the installer's name is always a checked one.
pub async fn download(
    client: &Client,
    release: &Release,
    dir: &Path,
) -> Result<PathBuf, UpdateError> {
    let installer = &release.installer;
    let target = dir.join(&installer.file_name);
    let part = dir.join(format!("{}.part", installer.file_name));
    let expected = installer.size;

    let response = client
        .get(&installer.url)
        .send()
        .await
        .map_err(UpdateError::Download)?;
    if !response.status().is_success() {
        return Err(UpdateError::DownloadStatus(response.status().as_u16()));
    }
    if let Some(declared) = response.content_length()
        && declared != expected
    {
        return Err(UpdateError::Size {
            expected,
            actual: declared,
        });
    }

    let mut file = tokio::fs::File::create(&part).await?;
    let mut hasher = Sha256::new();
    let mut received = 0u64;
    let mut stream = response.bytes_stream();
    let outcome: Result<(), UpdateError> = async {
        while let Some(chunk) = stream.next().await {
            let chunk = chunk.map_err(UpdateError::Download)?;
            received += chunk.len() as u64;
            if received > expected {
                return Err(UpdateError::Size {
                    expected,
                    actual: received,
                });
            }
            hasher.update(&chunk);
            file.write_all(&chunk).await?;
        }
        file.sync_all().await?;
        if received != expected {
            return Err(UpdateError::Size {
                expected,
                actual: received,
            });
        }
        if hex::encode(hasher.finalize()) != installer.sha256 {
            return Err(UpdateError::Checksum);
        }
        Ok(())
    }
    .await;
    drop(file);
    if let Err(err) = outcome {
        let _ = tokio::fs::remove_file(&part).await;
        return Err(err);
    }
    tokio::fs::rename(&part, &target).await?;
    Ok(target)
}

/// The version this build is, as [`decide`] compares it.
pub fn parse_version(value: &str) -> Option<Version> {
    value.parse().ok()
}

#[cfg(test)]
mod tests {
    use capture_protocol::release::{Installer, PRODUCT};

    use super::*;

    fn release(version: &str, minimum_windows_build: u32) -> Release {
        Release {
            product: PRODUCT.into(),
            version: version.into(),
            published_at: 0,
            minimum_windows_build,
            installer: Installer {
                file_name: "TrenovaCapture.msi".into(),
                url: "https://example.test/TrenovaCapture.msi".into(),
                sha256: "00".repeat(32),
                size: 1,
            },
            notes: String::new(),
        }
    }

    #[test]
    fn a_release_is_installed_only_when_newer_and_runnable() {
        assert_eq!(
            decide("1.4.2", &release("1.5.0", 19045), 22631),
            Decision::Install
        );
        assert_eq!(
            decide("1.5.0", &release("1.5.0", 19045), 22631),
            Decision::UpToDate
        );
        assert_eq!(
            decide("1.6.0", &release("1.5.0", 19045), 22631),
            Decision::UpToDate
        );
        assert_eq!(
            decide("1.4.2", &release("1.5.0", 22631), 19045),
            Decision::WindowsTooOld { needs: 22631 }
        );
        assert_eq!(
            decide("garbage", &release("1.5.0", 19045), 22631),
            Decision::UpToDate
        );
    }
}
