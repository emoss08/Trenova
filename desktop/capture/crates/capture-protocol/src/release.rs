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
//!
//! A manifest also carries the time it expires, signed with the rest. The
//! release build signs it for [`DEFAULT_VALIDITY_DAYS`] and a scheduled job
//! signs the current one again every day, so a manifest served long after
//! that (a mirror holding back a newer release) is refused, and the computer
//! keeps the release it has and says why.

use std::cmp::Ordering;
use std::fmt;
use std::str::FromStr;
use std::time::{SystemTime, UNIX_EPOCH};

use base64::Engine;
use base64::engine::general_purpose::STANDARD;
use ed25519_dalek::{Signature, Signer, SigningKey, VerifyingKey};
use serde::{Deserialize, Serialize};

/// The product every manifest names.
pub const PRODUCT: &str = "trenova-capture";
/// Where every server serves the manifest, below any path it is hosted at.
pub const MANIFEST_ROUTE: &str = "/api/v1/capture/releases/latest/";
/// The file name a manifest is published under.
pub const MANIFEST_FILE: &str = "trenova-capture-manifest.json";
/// The largest installer a manifest may describe.
pub const MAX_INSTALLER_BYTES: u64 = 512 << 20;
/// The largest signed manifest anyone should read.
pub const MAX_SIGNED_BYTES: usize = 64 << 10;
const MAX_NOTES_CHARS: usize = 4000;
/// How long a manifest is signed for unless the signer says otherwise. The
/// scheduled re-sign runs daily, so many missed runs pass before a manifest
/// expires.
pub const DEFAULT_VALIDITY_DAYS: i64 = 14;
/// The longest a manifest may be signed for.
pub const MAX_VALIDITY_DAYS: i64 = 31;
const SECONDS_PER_DAY: i64 = 24 * 60 * 60;

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
    #[error("the manifest carries no expiry")]
    NoExpiry,
    #[error("the manifest expired {days_ago} day(s) ago, at Unix time {expired_at}")]
    Expired { expired_at: i64, days_ago: i64 },
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
    /// Unix seconds after which the manifest is refused. Set by every
    /// signature, the release's own and each scheduled re-sign; a manifest
    /// without one is read only to be signed again.
    #[serde(default, skip_serializing_if = "Option::is_none")]
    pub expires_at: Option<i64>,
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
        if self
            .expires_at
            .is_some_and(|expires_at| expires_at <= self.published_at)
        {
            return Err(ReleaseError::Invalid(
                "the manifest expires before it was published",
            ));
        }
        Ok(version)
    }

    /// Whether the manifest may still be acted on at `now` (Unix seconds):
    /// it carries an expiry and that expiry is later than `now`.
    pub fn check_current(&self, now: i64) -> Result<(), ReleaseError> {
        match self.expires_at {
            None => Err(ReleaseError::NoExpiry),
            Some(expired_at) if now >= expired_at => Err(ReleaseError::Expired {
                expired_at,
                days_ago: now.saturating_sub(expired_at) / SECONDS_PER_DAY,
            }),
            Some(_) => Ok(()),
        }
    }
}

/// The time now in Unix seconds, as a manifest's expiry is compared with. A
/// clock set before 1970 reads as the end of time, so it expires everything
/// rather than nothing.
pub fn unix_now() -> i64 {
    SystemTime::now()
        .duration_since(UNIX_EPOCH)
        .map_or(i64::MAX, |elapsed| {
            i64::try_from(elapsed.as_secs()).unwrap_or(i64::MAX)
        })
}

/// When a manifest signed at `now` for `days` expires. `days` is between 1
/// and [`MAX_VALIDITY_DAYS`].
pub fn expiry(now: i64, days: i64) -> Result<i64, ReleaseError> {
    if !(1..=MAX_VALIDITY_DAYS).contains(&days) {
        return Err(ReleaseError::Invalid(
            "a manifest is signed for between 1 and 31 days",
        ));
    }
    now.checked_add(days * SECONDS_PER_DAY)
        .ok_or(ReleaseError::Invalid("the expiry is out of range"))
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

/// Signs a release, after checking it is one a reader would accept. It must
/// carry an expiry.
pub fn sign(release: &Release, key: &SigningKey) -> Result<SignedRelease, ReleaseError> {
    release.validate()?;
    if release.expires_at.is_none() {
        return Err(ReleaseError::NoExpiry);
    }
    let bytes = serde_json::to_vec(release).map_err(|e| ReleaseError::Format(e.to_string()))?;
    let signature = key.sign(&bytes);
    Ok(SignedRelease {
        manifest: STANDARD.encode(&bytes),
        signature: STANDARD.encode(signature.to_bytes()),
    })
}

/// Signs the release `signed` describes again, to expire at `expires_at`:
/// the scheduled re-sign. Only a manifest this same key signed is renewed,
/// whether it has expired or, signed before manifests carried one, has no
/// expiry at all. Nothing else in the release changes.
pub fn renew(
    signed: &SignedRelease,
    key: &SigningKey,
    expires_at: i64,
) -> Result<SignedRelease, ReleaseError> {
    let mut release = verify(signed, &key.verifying_key())?;
    release.expires_at = Some(expires_at);
    sign(&release, key)
}

/// Checks the signature, then the release it covers. Whether it has expired
/// is [`Release::check_current`]'s question, asked by whoever acts on it.
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

    const PUBLISHED: i64 = 1_780_000_000;
    const EXPIRES: i64 = PUBLISHED + 14 * SECONDS_PER_DAY;

    fn release(version: &str) -> Release {
        Release {
            product: PRODUCT.into(),
            version: version.into(),
            published_at: PUBLISHED,
            expires_at: Some(EXPIRES),
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
    fn the_manifest_route_is_a_capture_api_route() {
        assert_eq!(
            MANIFEST_ROUTE,
            format!("{}releases/latest/", crate::api::API_PREFIX)
        );
    }

    /// A manifest as releases were signed before they carried an expiry.
    fn signed_without_expiry(release: &Release, key: &SigningKey) -> SignedRelease {
        let bytes = serde_json::to_vec(release).expect("json");
        SignedRelease {
            manifest: STANDARD.encode(&bytes),
            signature: STANDARD.encode(key.sign(&bytes).to_bytes()),
        }
    }

    #[test]
    fn a_manifest_is_current_only_before_it_expires() {
        let current = release("1.4.2");
        assert_eq!(current.check_current(PUBLISHED), Ok(()));
        assert_eq!(current.check_current(EXPIRES - 1), Ok(()));
        assert_eq!(
            current.check_current(EXPIRES),
            Err(ReleaseError::Expired {
                expired_at: EXPIRES,
                days_ago: 0
            })
        );
        assert_eq!(
            current.check_current(EXPIRES + 3 * SECONDS_PER_DAY + 5),
            Err(ReleaseError::Expired {
                expired_at: EXPIRES,
                days_ago: 3
            })
        );

        let unbounded = Release {
            expires_at: None,
            ..release("1.4.2")
        };
        assert_eq!(
            unbounded.check_current(PUBLISHED),
            Err(ReleaseError::NoExpiry)
        );
    }

    #[test]
    fn the_expiry_is_signed_with_the_rest_and_required_to_sign() {
        let (private, public) = keys();
        let signed = sign(&release("1.4.2"), &private).expect("signs");
        let bytes = STANDARD.decode(&signed.manifest).expect("base64");
        let json: serde_json::Value = serde_json::from_slice(&bytes).expect("json");
        assert_eq!(json["expiresAt"], EXPIRES);
        assert_eq!(
            verify(&signed, &public).expect("verifies").expires_at,
            Some(EXPIRES)
        );

        let unbounded = Release {
            expires_at: None,
            ..release("1.4.2")
        };
        assert_eq!(sign(&unbounded, &private), Err(ReleaseError::NoExpiry));

        let backwards = Release {
            expires_at: Some(PUBLISHED),
            ..release("1.4.2")
        };
        assert!(matches!(
            sign(&backwards, &private),
            Err(ReleaseError::Invalid(_))
        ));
    }

    #[test]
    fn a_manifest_signed_before_expiries_reads_back_and_is_renewed_with_one() {
        let (private, public) = keys();
        let old = Release {
            expires_at: None,
            ..release("1.4.2")
        };
        let published = signed_without_expiry(&old, &private);
        let bytes = STANDARD.decode(&published.manifest).expect("base64");
        assert!(
            !String::from_utf8(bytes)
                .expect("utf-8")
                .contains("expiresAt"),
            "an older manifest has no expiry field"
        );

        let read = verify(&published, &public).expect("still verifies");
        assert_eq!(read.check_current(PUBLISHED), Err(ReleaseError::NoExpiry));

        let renewed = renew(&published, &private, EXPIRES).expect("renews");
        let renewed = verify(&renewed, &public).expect("verifies");
        assert_eq!(renewed.check_current(EXPIRES - 1), Ok(()));
        assert_eq!(
            renewed,
            Release {
                expires_at: Some(EXPIRES),
                ..old
            }
        );
    }

    #[test]
    fn an_expired_manifest_is_renewed_and_only_by_the_key_that_signed_it() {
        let (private, public) = keys();
        let signed = sign(&release("1.4.2"), &private).expect("signs");
        let later = EXPIRES + 30 * SECONDS_PER_DAY;

        let renewed = renew(&signed, &private, later).expect("renews");
        let renewed = verify(&renewed, &public).expect("verifies");
        assert_eq!(renewed.check_current(EXPIRES + SECONDS_PER_DAY), Ok(()));
        assert_eq!(renewed.published_at, PUBLISHED);
        assert_eq!(renewed.version, "1.4.2");

        let (stranger, _) = encode_key_pair([9u8; 32]);
        let stranger = signing_key(&stranger).expect("key");
        assert_eq!(
            renew(&signed, &stranger, later),
            Err(ReleaseError::Signature)
        );
    }

    #[test]
    fn a_manifest_is_signed_for_between_one_and_thirty_one_days() {
        assert_eq!(
            expiry(PUBLISHED, DEFAULT_VALIDITY_DAYS),
            Ok(PUBLISHED + 14 * SECONDS_PER_DAY)
        );
        assert_eq!(
            expiry(PUBLISHED, MAX_VALIDITY_DAYS),
            Ok(PUBLISHED + 31 * SECONDS_PER_DAY)
        );
        for days in [0, -1, MAX_VALIDITY_DAYS + 1] {
            assert!(
                matches!(expiry(PUBLISHED, days), Err(ReleaseError::Invalid(_))),
                "{days}"
            );
        }
        assert!(unix_now() > PUBLISHED);
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
