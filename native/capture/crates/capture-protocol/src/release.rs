//! The release manifest: which Trenova Capture is current and where its
//! installer is.
//!
//! A release is described once, by the build that made it, and signed with
//! an ed25519 key only that build holds. The signature covers the manifest's
//! exact bytes, carried base64-encoded beside it, so nothing has to agree on
//! how to re-serialize JSON before checking it. The agent and the updater pin
//! the public key at build time, and the server checks it too before serving
//! a manifest, so a manifest is trusted for its signature and never for where
//! it came from.

use std::cmp::Ordering;
use std::fmt;
use std::str::FromStr;

use base64::Engine;
use base64::engine::general_purpose::STANDARD;
use ed25519_dalek::{Signature, Signer, SigningKey, VerifyingKey};
use serde::{Deserialize, Serialize};

/// The product every manifest names.
pub const PRODUCT: &str = "trenova-capture";
/// The file name a manifest is published under.
pub const MANIFEST_FILE: &str = "trenova-capture-manifest.json";
/// The largest installer a manifest may describe.
pub const MAX_INSTALLER_BYTES: u64 = 512 << 20;
/// The largest signed manifest anyone should read.
pub const MAX_SIGNED_BYTES: usize = 64 << 10;
const MAX_NOTES_CHARS: usize = 4000;

#[derive(Clone, Debug, thiserror::Error, PartialEq, Eq)]
pub enum ReleaseError {
    #[error("the manifest is not base64")]
    Encoding,
    #[error("the manifest's signature does not verify")]
    Signature,
    #[error("the key is not a base64 ed25519 key")]
    Key,
    #[error("the manifest is unreadable: {0}")]
    Format(String),
    #[error("the manifest is invalid: {0}")]
    Invalid(&'static str),
}

/// `MAJOR.MINOR.PATCH`, as the server's `versionutils` reads it.
#[derive(Clone, Copy, Debug, PartialEq, Eq, Hash, PartialOrd, Ord)]
pub struct Version {
    pub major: u32,
    pub minor: u32,
    pub patch: u32,
}

impl FromStr for Version {
    type Err = ReleaseError;

    fn from_str(value: &str) -> Result<Self, Self::Err> {
        let invalid = ReleaseError::Invalid("a version is MAJOR.MINOR.PATCH");
        let value = value.trim();
        let value = value.strip_prefix('v').unwrap_or(value);
        let mut parts = value.split('.');
        let mut next = || -> Result<u32, ReleaseError> {
            let part = parts.next().ok_or(invalid.clone())?;
            if part.is_empty()
                || part.len() > 9
                || (part.len() > 1 && part.starts_with('0'))
                || !part.bytes().all(|b: u8| b.is_ascii_digit())
            {
                return Err(invalid.clone());
            }
            part.parse().map_err(|_| invalid.clone())
        };
        let version = Self {
            major: next()?,
            minor: next()?,
            patch: next()?,
        };
        if parts.next().is_some() {
            return Err(invalid);
        }
        Ok(version)
    }
}

impl fmt::Display for Version {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        write!(f, "{}.{}.{}", self.major, self.minor, self.patch)
    }
}

/// Whether `candidate` is newer than `current`. An unreadable version is
/// never newer, so a bad value cannot start an install.
pub fn is_newer(candidate: &str, current: &str) -> bool {
    match (candidate.parse::<Version>(), current.parse::<Version>()) {
        (Ok(candidate), Ok(current)) => candidate.cmp(&current) == Ordering::Greater,
        _ => false,
    }
}

/// The installer a release ships.
#[derive(Clone, Debug, PartialEq, Eq, Serialize, Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct Installer {
    pub file_name: String,
    /// Where it is downloaded from; always HTTPS.
    pub url: String,
    /// Lowercase hex SHA-256 of the file.
    pub sha256: String,
    pub size: u64,
}

/// One release.
#[derive(Clone, Debug, PartialEq, Eq, Serialize, Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct Release {
    pub product: String,
    pub version: String,
    /// Unix seconds.
    pub published_at: i64,
    /// The oldest Windows build it runs on (19045 is Windows 10 22H2).
    pub minimum_windows_build: u32,
    pub installer: Installer,
    #[serde(default)]
    pub notes: String,
}

impl Release {
    /// Checks everything a reader acts on, so a signed but malformed manifest
    /// still cannot point an install anywhere unexpected.
    pub fn validate(&self) -> Result<Version, ReleaseError> {
        if self.product != PRODUCT {
            return Err(ReleaseError::Invalid("the manifest is for another product"));
        }
        if self.version.starts_with('v') {
            return Err(ReleaseError::Invalid("a manifest version has no prefix"));
        }
        let version = self.version.parse::<Version>()?;
        let installer = &self.installer;
        if !installer.url.starts_with("https://") || installer.url.len() > 2048 {
            return Err(ReleaseError::Invalid("the installer must come over HTTPS"));
        }
        if installer.sha256.len() != 64
            || !installer
                .sha256
                .bytes()
                .all(|b| b.is_ascii_digit() || (b'a'..=b'f').contains(&b))
        {
            return Err(ReleaseError::Invalid(
                "the installer checksum is not SHA-256",
            ));
        }
        if installer.size == 0 || installer.size > MAX_INSTALLER_BYTES {
            return Err(ReleaseError::Invalid("the installer size is out of range"));
        }
        let name = &installer.file_name;
        if name.is_empty()
            || name.len() > 128
            || !name.to_ascii_lowercase().ends_with(".msi")
            || !name
                .bytes()
                .all(|b| b.is_ascii_alphanumeric() || matches!(b, b'.' | b'-' | b'_'))
        {
            return Err(ReleaseError::Invalid(
                "the installer file name is not a plain .msi name",
            ));
        }
        if self.notes.chars().count() > MAX_NOTES_CHARS {
            return Err(ReleaseError::Invalid("the release notes are too long"));
        }
        Ok(version)
    }
}

/// A manifest as published: its exact bytes and their signature.
#[derive(Clone, Debug, PartialEq, Eq, Serialize, Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct SignedRelease {
    /// The manifest's JSON, base64.
    pub manifest: String,
    /// The ed25519 signature of those bytes, base64.
    pub signature: String,
}

/// The key releases are signed with, fixed at build time from
/// `TRENOVA_CAPTURE_PUBLIC_KEY`. A build made without one, as a development
/// build is, trusts no release and never updates itself.
pub fn pinned_public_key() -> Option<VerifyingKey> {
    option_env!("TRENOVA_CAPTURE_PUBLIC_KEY").and_then(|key| public_key(key).ok())
}

/// Reads a base64 ed25519 public key.
pub fn public_key(value: &str) -> Result<VerifyingKey, ReleaseError> {
    let bytes: [u8; 32] = STANDARD
        .decode(value.trim())
        .map_err(|_| ReleaseError::Key)?
        .try_into()
        .map_err(|_| ReleaseError::Key)?;
    VerifyingKey::from_bytes(&bytes).map_err(|_| ReleaseError::Key)
}

/// Reads a base64 ed25519 private key (its 32-byte seed).
pub fn signing_key(value: &str) -> Result<SigningKey, ReleaseError> {
    let bytes: [u8; 32] = STANDARD
        .decode(value.trim())
        .map_err(|_| ReleaseError::Key)?
        .try_into()
        .map_err(|_| ReleaseError::Key)?;
    Ok(SigningKey::from_bytes(&bytes))
}

/// A key pair from 32 random bytes: the private seed and the public key,
/// both base64.
pub fn encode_key_pair(seed: [u8; 32]) -> (String, String) {
    let key = SigningKey::from_bytes(&seed);
    (
        STANDARD.encode(key.to_bytes()),
        STANDARD.encode(key.verifying_key().to_bytes()),
    )
}

/// Signs a release, after checking it is one a reader would accept.
pub fn sign(release: &Release, key: &SigningKey) -> Result<SignedRelease, ReleaseError> {
    release.validate()?;
    let bytes = serde_json::to_vec(release).map_err(|e| ReleaseError::Format(e.to_string()))?;
    let signature = key.sign(&bytes);
    Ok(SignedRelease {
        manifest: STANDARD.encode(&bytes),
        signature: STANDARD.encode(signature.to_bytes()),
    })
}

/// Checks the signature, then the release it covers.
pub fn verify(signed: &SignedRelease, key: &VerifyingKey) -> Result<Release, ReleaseError> {
    let bytes = STANDARD
        .decode(&signed.manifest)
        .map_err(|_| ReleaseError::Encoding)?;
    let signature: [u8; 64] = STANDARD
        .decode(&signed.signature)
        .map_err(|_| ReleaseError::Encoding)?
        .try_into()
        .map_err(|_| ReleaseError::Signature)?;
    key.verify_strict(&bytes, &Signature::from_bytes(&signature))
        .map_err(|_| ReleaseError::Signature)?;
    let release: Release =
        serde_json::from_slice(&bytes).map_err(|e| ReleaseError::Format(e.to_string()))?;
    release.validate()?;
    Ok(release)
}

#[cfg(test)]
mod tests {
    use super::*;

    fn release(version: &str) -> Release {
        Release {
            product: PRODUCT.into(),
            version: version.into(),
            published_at: 1_780_000_000,
            minimum_windows_build: 19045,
            installer: Installer {
                file_name: format!("TrenovaCapture-{version}-x64.msi"),
                url: format!(
                    "https://github.com/emoss08/trenova/releases/download/capture-v{version}/TrenovaCapture-{version}-x64.msi"
                ),
                sha256: "ab".repeat(32),
                size: 12_345_678,
            },
            notes: "Printing into Trenova.".into(),
        }
    }

    fn keys() -> (SigningKey, VerifyingKey) {
        let (private, public) = encode_key_pair([7u8; 32]);
        (
            signing_key(&private).expect("private"),
            public_key(&public).expect("public"),
        )
    }

    #[test]
    fn a_signed_release_verifies_and_reads_back() {
        let (private, public) = keys();
        let signed = sign(&release("1.4.2"), &private).expect("signs");
        assert_eq!(
            verify(&signed, &public).expect("verifies"),
            release("1.4.2")
        );
    }

    #[test]
    fn any_change_to_the_manifest_or_a_different_key_fails() {
        let (private, public) = keys();
        let signed = sign(&release("1.4.2"), &private).expect("signs");

        let mut bytes = STANDARD.decode(&signed.manifest).expect("base64");
        let at = bytes.iter().position(|&b| b == b'4').expect("a digit");
        bytes[at] = b'9';
        let tampered = SignedRelease {
            manifest: STANDARD.encode(&bytes),
            ..signed.clone()
        };
        assert_eq!(verify(&tampered, &public), Err(ReleaseError::Signature));

        let (_, other) = encode_key_pair([9u8; 32]);
        assert_eq!(
            verify(&signed, &public_key(&other).expect("key")),
            Err(ReleaseError::Signature)
        );
        let garbled = SignedRelease {
            signature: "not base64!".into(),
            ..signed
        };
        assert_eq!(verify(&garbled, &public), Err(ReleaseError::Encoding));
    }

    #[test]
    fn a_manifest_that_points_somewhere_unexpected_is_refused_even_signed() {
        let (private, _) = keys();
        let mut plain_http = release("1.0.0");
        plain_http.installer.url = "http://example.test/a.msi".into();
        let mut path = release("1.0.0");
        path.installer.file_name = "..\\evil.msi".into();
        let mut exe = release("1.0.0");
        exe.installer.file_name = "setup.exe".into();
        let mut other = release("1.0.0");
        other.product = "something-else".into();
        let mut huge = release("1.0.0");
        huge.installer.size = MAX_INSTALLER_BYTES + 1;
        let mut checksum = release("1.0.0");
        checksum.installer.sha256 = "AB".repeat(32);
        for bad in [
            plain_http,
            path,
            exe,
            other,
            huge,
            checksum,
            release("1.0"),
            release("v1.0.0"),
        ] {
            assert!(
                matches!(sign(&bad, &private), Err(ReleaseError::Invalid(_))),
                "{bad:?}"
            );
        }
    }

    #[test]
    fn versions_compare_numerically_and_nonsense_is_never_newer() {
        assert!(is_newer("1.10.0", "1.9.9"));
        assert!(is_newer("v2.0.0", "1.99.99"));
        assert!(!is_newer("1.4.2", "1.4.2"));
        assert!(!is_newer("1.4.1", "1.4.2"));
        for nonsense in [
            "",
            "1.4",
            "1.4.2.1",
            "1.x.0",
            "-1.0.0",
            "1.0.0-beta",
            "9999999999.0.0",
        ] {
            assert!(!is_newer(nonsense, "0.0.1"), "{nonsense}");
        }
        assert_eq!(
            "1.4.2".parse::<Version>().expect("parses").to_string(),
            "1.4.2"
        );
    }

    #[test]
    fn keys_must_be_well_formed() {
        assert_eq!(public_key("short").map(|_| ()), Err(ReleaseError::Key));
        assert_eq!(
            signing_key(&STANDARD.encode([1u8; 31])).map(|_| ()),
            Err(ReleaseError::Key)
        );
    }
}
