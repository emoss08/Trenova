//! `trenova-capture-release`: the release build's signing tool.
//!
//! ```text
//! trenova-capture-release keygen
//! trenova-capture-release sign --msi <file> --version <x.y.z> --url <https://…> [--notes <file>] [--min-windows-build <n>] [--valid-days <n>] [--out <file>]
//! trenova-capture-release resign --manifest <file> --public-key <base64> [--version <x.y.z>] [--valid-days <n>] [--out <file>]
//! trenova-capture-release verify --manifest <file> --public-key <base64>
//! ```
//!
//! A manifest expires `--valid-days` after it is signed (14 unless given, at
//! most 31). `resign` signs a published manifest again with a fresh expiry
//! and nothing else changed, which a scheduled job does every day; it renews
//! only a manifest the same key signed, expired or not, with an expiry or
//! from before manifests carried one. `verify` also refuses an expired one.
//!
//! `sign` and `resign` read the private key from
//! `TRENOVA_CAPTURE_SIGNING_KEY`, never from the command line, where it would
//! land in process listings and shell history.

use std::collections::HashMap;
use std::io::{Read, Write};
use std::path::Path;
use std::process::ExitCode;
use std::time::{SystemTime, UNIX_EPOCH};

use capture_protocol::release::{
    self, DEFAULT_VALIDITY_DAYS, Installer, PRODUCT, Release, SignedRelease, encode_key_pair,
    expiry, public_key, signing_key, unix_now,
};
use sha2::{Digest, Sha256};

const KEY_VARIABLE: &str = "TRENOVA_CAPTURE_SIGNING_KEY";
/// Windows 10 22H2.
const DEFAULT_MIN_WINDOWS_BUILD: u32 = 19045;

type Failure = Box<dyn std::error::Error>;

fn options(args: &[String]) -> Result<HashMap<String, String>, Failure> {
    let mut parsed = HashMap::new();
    let mut iter = args.iter();
    while let Some(flag) = iter.next() {
        let name = flag
            .strip_prefix("--")
            .ok_or_else(|| format!("unexpected argument {flag}"))?;
        let value = iter.next().ok_or_else(|| format!("{flag} needs a value"))?;
        parsed.insert(name.to_owned(), value.clone());
    }
    Ok(parsed)
}

fn required<'a>(options: &'a HashMap<String, String>, name: &str) -> Result<&'a str, Failure> {
    options
        .get(name)
        .map(String::as_str)
        .ok_or_else(|| format!("--{name} is required").into())
}

/// How many days a manifest signed now is valid for.
fn valid_days(options: &HashMap<String, String>) -> Result<i64, Failure> {
    match options.get("valid-days") {
        Some(days) => Ok(days.parse()?),
        None => Ok(DEFAULT_VALIDITY_DAYS),
    }
}

/// Reads a signed manifest from disk, no larger than any reader would take.
fn read_signed(path: &str) -> Result<SignedRelease, Failure> {
    let bytes = std::fs::read(path)?;
    if bytes.len() > release::MAX_SIGNED_BYTES {
        return Err("the manifest is too large".into());
    }
    Ok(serde_json::from_slice(&bytes)?)
}

fn write_signed(
    options: &HashMap<String, String>,
    signed: &SignedRelease,
    out: &mut impl Write,
) -> Result<(), Failure> {
    let json = serde_json::to_string_pretty(signed)?;
    match options.get("out") {
        Some(path) => std::fs::write(path, json + "\n")?,
        None => writeln!(out, "{json}")?,
    }
    Ok(())
}

/// SHA-256 and size of a file, read in blocks.
fn digest(path: &Path) -> Result<(String, u64), Failure> {
    let mut file = std::fs::File::open(path)?;
    let mut hasher = Sha256::new();
    let mut buffer = vec![0u8; 1 << 20];
    let mut size = 0u64;
    loop {
        let read = file.read(&mut buffer)?;
        if read == 0 {
            break;
        }
        hasher.update(&buffer[..read]);
        size += read as u64;
    }
    Ok((hex::encode(hasher.finalize()), size))
}

fn keygen(out: &mut impl Write) -> Result<(), Failure> {
    let mut seed = [0u8; 32];
    getrandom::fill(&mut seed).map_err(|err| format!("no randomness: {err}"))?;
    let (private, public) = encode_key_pair(seed);
    writeln!(out, "private (keep secret, {KEY_VARIABLE}): {private}")?;
    writeln!(out, "public (pin in builds and the server): {public}")?;
    Ok(())
}

fn sign(options: &HashMap<String, String>, key: &str, out: &mut impl Write) -> Result<(), Failure> {
    let msi = Path::new(required(options, "msi")?);
    let (sha256, size) = digest(msi)?;
    let file_name = msi
        .file_name()
        .and_then(|n| n.to_str())
        .ok_or("the installer path has no file name")?
        .to_owned();
    let notes = match options.get("notes") {
        Some(path) => std::fs::read_to_string(path)?.trim().to_owned(),
        None => String::new(),
    };
    let minimum_windows_build = match options.get("min-windows-build") {
        Some(build) => build.parse()?,
        None => DEFAULT_MIN_WINDOWS_BUILD,
    };
    let published_at = i64::try_from(SystemTime::now().duration_since(UNIX_EPOCH)?.as_secs())?;
    let release = Release {
        product: PRODUCT.into(),
        version: required(options, "version")?
            .trim_start_matches('v')
            .to_owned(),
        published_at,
        expires_at: Some(expiry(published_at, valid_days(options)?)?),
        minimum_windows_build,
        installer: Installer {
            file_name,
            url: required(options, "url")?.to_owned(),
            sha256,
            size,
        },
        notes,
    };
    let signed = release::sign(&release, &signing_key(key)?)?;
    write_signed(options, &signed, out)
}

fn resign(
    options: &HashMap<String, String>,
    key: &str,
    out: &mut impl Write,
) -> Result<(), Failure> {
    let signed = read_signed(required(options, "manifest")?)?;
    let pinned = public_key(required(options, "public-key")?)?;
    let key = signing_key(key)?;
    if key.verifying_key() != pinned {
        return Err(format!("{KEY_VARIABLE} is not the private half of --public-key").into());
    }
    let current = release::verify(&signed, &pinned)?;
    if let Some(expected) = options.get("version")
        && current.version != expected.trim_start_matches('v')
    {
        return Err(format!("the manifest is for {}, not {expected}", current.version).into());
    }
    let expires_at = expiry(unix_now(), valid_days(options)?)?;
    let renewed = release::renew(&signed, &key, expires_at)?;
    write_signed(options, &renewed, out)
}

fn verify(options: &HashMap<String, String>, out: &mut impl Write) -> Result<(), Failure> {
    let signed = read_signed(required(options, "manifest")?)?;
    let release = release::verify(&signed, &public_key(required(options, "public-key")?)?)?;
    release.check_current(unix_now())?;
    writeln!(
        out,
        "verified: {} {} ({} bytes, sha256 {}), expires at Unix time {}",
        release.product,
        release.version,
        release.installer.size,
        release.installer.sha256,
        release.expires_at.unwrap_or_default()
    )?;
    Ok(())
}

fn run(args: &[String], key: Option<String>, out: &mut impl Write) -> Result<(), Failure> {
    let (command, rest) = args
        .split_first()
        .ok_or("usage: keygen | sign | resign | verify")?;
    match command.as_str() {
        "keygen" => keygen(out),
        "sign" => {
            let key = key.ok_or_else(|| format!("{KEY_VARIABLE} is not set"))?;
            sign(&options(rest)?, &key, out)
        }
        "resign" => {
            let key = key.ok_or_else(|| format!("{KEY_VARIABLE} is not set"))?;
            resign(&options(rest)?, &key, out)
        }
        "verify" => verify(&options(rest)?, out),
        other => Err(format!("unknown command {other}").into()),
    }
}

fn main() -> ExitCode {
    let args: Vec<String> = std::env::args().skip(1).collect();
    let key = std::env::var(KEY_VARIABLE).ok();
    match run(&args, key, &mut std::io::stdout()) {
        Ok(()) => ExitCode::SUCCESS,
        Err(err) => {
            let _ = writeln!(std::io::stderr(), "trenova-capture-release: {err}");
            ExitCode::FAILURE
        }
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    fn text(bytes: Vec<u8>) -> String {
        String::from_utf8(bytes).expect("utf-8")
    }

    fn field<'a>(output: &'a str, label: &str) -> &'a str {
        output
            .lines()
            .find(|l| l.starts_with(label))
            .and_then(|l| l.rsplit(' ').next())
            .expect("a key line")
    }

    fn args(values: &[&str]) -> Vec<String> {
        values.iter().map(|v| (*v).to_owned()).collect()
    }

    #[test]
    fn a_manifest_signed_with_a_new_key_verifies_against_its_public_half() {
        let dir = tempfile::tempdir().expect("dir");
        let msi = dir.path().join("TrenovaCapture-1.2.0-x64.msi");
        std::fs::write(&msi, b"not really an msi").expect("msi");
        let manifest = dir.path().join("manifest.json");

        let mut keys = Vec::new();
        run(&args(&["keygen"]), None, &mut keys).expect("keygen");
        let keys = text(keys);
        let (private, public) = (field(&keys, "private"), field(&keys, "public"));

        run(
            &args(&[
                "sign",
                "--msi",
                msi.to_str().expect("path"),
                "--version",
                "v1.2.0",
                "--url",
                "https://example.test/TrenovaCapture-1.2.0-x64.msi",
                "--out",
                manifest.to_str().expect("path"),
            ]),
            Some(private.to_owned()),
            &mut Vec::new(),
        )
        .expect("signs");

        let mut verified = Vec::new();
        run(
            &args(&[
                "verify",
                "--manifest",
                manifest.to_str().expect("path"),
                "--public-key",
                public,
            ]),
            None,
            &mut verified,
        )
        .expect("verifies");
        let verified = text(verified);
        assert!(
            verified.contains("trenova-capture 1.2.0 (17 bytes"),
            "{verified}"
        );

        let mut other = Vec::new();
        run(&args(&["keygen"]), None, &mut other).expect("keygen");
        let stranger = text(other);
        assert!(
            run(
                &args(&[
                    "verify",
                    "--manifest",
                    manifest.to_str().expect("path"),
                    "--public-key",
                    field(&stranger, "public"),
                ]),
                None,
                &mut Vec::new(),
            )
            .is_err()
        );
    }

    /// The release a manifest on disk describes, checked against `public`.
    fn read_back(path: &Path, public: &str) -> Release {
        let signed = read_signed(path.to_str().expect("path")).expect("reads");
        release::verify(&signed, &public_key(public).expect("key")).expect("verifies")
    }

    fn sign_one(dir: &Path, private: &str, days: &str) -> std::path::PathBuf {
        let msi = dir.join("TrenovaCapture-1.2.0-x64.msi");
        std::fs::write(&msi, b"not really an msi").expect("msi");
        let manifest = dir.join(format!("manifest-{days}.json"));
        run(
            &args(&[
                "sign",
                "--msi",
                msi.to_str().expect("path"),
                "--version",
                "1.2.0",
                "--url",
                "https://example.test/TrenovaCapture-1.2.0-x64.msi",
                "--valid-days",
                days,
                "--out",
                manifest.to_str().expect("path"),
            ]),
            Some(private.to_owned()),
            &mut Vec::new(),
        )
        .expect("signs");
        manifest
    }

    #[test]
    fn a_manifest_expires_the_given_days_after_it_is_signed() {
        let dir = tempfile::tempdir().expect("dir");
        let (private, public) = encode_key_pair([3u8; 32]);

        let before = unix_now();
        let release = read_back(&sign_one(dir.path(), &private, "14"), &public);
        let expires_at = release.expires_at.expect("an expiry");
        assert!(expires_at >= before + 14 * 86_400 && expires_at <= unix_now() + 14 * 86_400);

        for days in ["0", "32", "soon"] {
            let msi = dir.path().join("TrenovaCapture-1.2.0-x64.msi");
            assert!(
                run(
                    &args(&[
                        "sign",
                        "--msi",
                        msi.to_str().expect("path"),
                        "--version",
                        "1.2.0",
                        "--url",
                        "https://example.test/TrenovaCapture-1.2.0-x64.msi",
                        "--valid-days",
                        days,
                    ]),
                    Some(private.clone()),
                    &mut Vec::new(),
                )
                .is_err(),
                "{days}"
            );
        }
    }

    #[test]
    fn resigning_renews_the_expiry_and_changes_nothing_else() {
        let dir = tempfile::tempdir().expect("dir");
        let (private, public) = encode_key_pair([3u8; 32]);
        let first = sign_one(dir.path(), &private, "1");
        let renewed = dir.path().join("renewed.json");

        run(
            &args(&[
                "resign",
                "--manifest",
                first.to_str().expect("path"),
                "--public-key",
                &public,
                "--version",
                "1.2.0",
                "--valid-days",
                "20",
                "--out",
                renewed.to_str().expect("path"),
            ]),
            Some(private.clone()),
            &mut Vec::new(),
        )
        .expect("resigns");

        let original = read_back(&first, &public);
        let again = read_back(&renewed, &public);
        assert!(
            again.expires_at.expect("an expiry")
                >= original.expires_at.expect("an expiry") + 19 * 86_400
        );
        assert_eq!(
            again,
            Release {
                expires_at: again.expires_at,
                ..original
            }
        );

        let mut verified = Vec::new();
        run(
            &args(&[
                "verify",
                "--manifest",
                renewed.to_str().expect("path"),
                "--public-key",
                &public,
            ]),
            None,
            &mut verified,
        )
        .expect("verifies");
        assert!(text(verified).contains("expires at Unix time"));
    }

    #[test]
    fn resigning_refuses_another_key_another_version_or_a_long_expiry() {
        let dir = tempfile::tempdir().expect("dir");
        let (private, public) = encode_key_pair([3u8; 32]);
        let (stranger, stranger_public) = encode_key_pair([4u8; 32]);
        let manifest = sign_one(dir.path(), &private, "14");
        let manifest = manifest.to_str().expect("path");
        let resign = |key: &str, public: &str, extra: &[&str]| {
            let mut all = vec!["resign", "--manifest", manifest, "--public-key", public];
            all.extend_from_slice(extra);
            run(&args(&all), Some(key.to_owned()), &mut Vec::new())
        };

        assert!(resign(&private, &public, &[]).is_ok());
        assert!(
            resign(&stranger, &public, &[]).is_err(),
            "the key is not the pinned key's pair"
        );
        assert!(
            resign(&stranger, &stranger_public, &[]).is_err(),
            "the manifest was signed by another key"
        );
        assert!(resign(&private, &public, &["--version", "1.3.0"]).is_err());
        assert!(resign(&private, &public, &["--valid-days", "32"]).is_err());
        assert!(
            run(
                &args(&["resign", "--manifest", manifest, "--public-key", &public]),
                None,
                &mut Vec::new()
            )
            .is_err(),
            "the key comes from the environment"
        );
    }

    #[test]
    fn signing_needs_the_key_from_the_environment_and_every_field() {
        assert!(run(&args(&["sign", "--msi", "x.msi"]), None, &mut Vec::new()).is_err());
        let (private, _) = encode_key_pair([3u8; 32]);
        assert!(
            run(
                &args(&["sign", "--version", "1.0.0"]),
                Some(private),
                &mut Vec::new()
            )
            .is_err()
        );
        assert!(run(&args(&["bogus"]), None, &mut Vec::new()).is_err());
    }
}
