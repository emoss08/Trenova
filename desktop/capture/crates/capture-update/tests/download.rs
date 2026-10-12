//! The manifest and the installer, fetched from a mock release host.

use capture_protocol::release::{
    Installer, PRODUCT, Release, encode_key_pair, public_key, sign, signing_key,
};
use capture_update::{UpdateError, download, fetch_release};
use sha2::{Digest, Sha256};
use wiremock::matchers::{method, path};
use wiremock::{Mock, MockServer, ResponseTemplate};

const MSI: &[u8] = b"MSI bytes, pretend";

fn release(server: &MockServer, msi: &[u8]) -> Release {
    Release {
        product: PRODUCT.into(),
        version: "1.5.0".into(),
        published_at: 1_780_000_000,
        minimum_windows_build: 19045,
        installer: Installer {
            file_name: "TrenovaCapture-1.5.0-x64.msi".into(),
            // wiremock serves plain HTTP; the HTTPS rule is the manifest's,
            // checked when it is signed, so the URL is rewritten after.
            url: format!("https://example.test{}", "/TrenovaCapture-1.5.0-x64.msi"),
            sha256: hex::encode(Sha256::digest(msi)),
            size: msi.len() as u64,
        },
        notes: String::new(),
    }
    .with_download_host(server)
}

trait WithHost {
    fn with_download_host(self, server: &MockServer) -> Self;
}

impl WithHost for Release {
    fn with_download_host(mut self, server: &MockServer) -> Self {
        self.installer.url = format!("{}/TrenovaCapture-1.5.0-x64.msi", server.uri());
        self
    }
}

fn keys() -> (ed25519_dalek::SigningKey, ed25519_dalek::VerifyingKey) {
    let (private, public) = encode_key_pair([5u8; 32]);
    (
        signing_key(&private).expect("private"),
        public_key(&public).expect("public"),
    )
}

/// Signs with the HTTPS URL the validator wants, then points the release at
/// the mock host: the signature check is what is under test, not TLS.
async fn publish(server: &MockServer, msi: &[u8], key: &ed25519_dalek::SigningKey) {
    let mut signed_release = release(server, msi);
    signed_release.installer.url = "https://example.test/TrenovaCapture-1.5.0-x64.msi".into();
    let signed = sign(&signed_release, key).expect("signs");
    Mock::given(method("GET"))
        .and(path("/releases/latest/"))
        .respond_with(ResponseTemplate::new(200).set_body_json(signed))
        .mount(server)
        .await;
}

#[tokio::test]
async fn a_signed_manifest_is_read_and_its_installer_is_checked_as_it_downloads() {
    let server = MockServer::start().await;
    let (private, public) = keys();
    publish(&server, MSI, &private).await;
    Mock::given(method("GET"))
        .and(path("/TrenovaCapture-1.5.0-x64.msi"))
        .respond_with(ResponseTemplate::new(200).set_body_bytes(MSI))
        .mount(&server)
        .await;

    let client = reqwest::Client::new();
    let fetched = fetch_release(
        &client,
        &format!("{}/releases/latest/", server.uri()),
        &public,
    )
    .await
    .expect("fetches")
    .expect("published");
    assert_eq!(fetched.version, "1.5.0");

    let dir = tempfile::tempdir().expect("dir");
    let path = download(&client, &release(&server, MSI), dir.path())
        .await
        .expect("downloads");
    assert_eq!(path, dir.path().join("TrenovaCapture-1.5.0-x64.msi"));
    assert_eq!(std::fs::read(&path).expect("reads"), MSI);
    assert!(
        !dir.path()
            .join("TrenovaCapture-1.5.0-x64.msi.part")
            .exists()
    );
}

#[tokio::test]
async fn a_download_that_differs_from_the_manifest_leaves_nothing_behind() {
    let server = MockServer::start().await;
    Mock::given(method("GET"))
        .and(path("/TrenovaCapture-1.5.0-x64.msi"))
        .respond_with(ResponseTemplate::new(200).set_body_bytes(b"MSI bytes, tampered"))
        .mount(&server)
        .await;
    let dir = tempfile::tempdir().expect("dir");
    let client = reqwest::Client::new();

    let wrong_size = download(&client, &release(&server, MSI), dir.path()).await;
    assert!(
        matches!(wrong_size, Err(UpdateError::Size { .. })),
        "{wrong_size:?}"
    );

    let mut same_size = release(&server, b"MSI bytes, tampered");
    same_size.installer.sha256 = "00".repeat(32);
    let wrong_hash = download(&client, &same_size, dir.path()).await;
    assert!(
        matches!(wrong_hash, Err(UpdateError::Checksum)),
        "{wrong_hash:?}"
    );

    assert!(
        std::fs::read_dir(dir.path())
            .expect("lists")
            .next()
            .is_none(),
        "no part or unchecked file remains"
    );
}

#[tokio::test]
async fn an_unpublished_forged_or_oversized_manifest_is_not_a_release() {
    let server = MockServer::start().await;
    let (_, public) = keys();
    let client = reqwest::Client::new();
    let url = format!("{}/releases/latest/", server.uri());

    let none = fetch_release(&client, &url, &public)
        .await
        .expect("fetches");
    assert!(none.is_none());

    let (other_key, _) = encode_key_pair([9u8; 32]);
    publish(&server, MSI, &signing_key(&other_key).expect("key")).await;
    let forged = fetch_release(&client, &url, &public).await;
    assert!(matches!(forged, Err(UpdateError::Release(_))), "{forged:?}");

    server.reset().await;
    Mock::given(method("GET"))
        .and(path("/releases/latest/"))
        .respond_with(ResponseTemplate::new(200).set_body_bytes(vec![b'x'; 70_000]))
        .mount(&server)
        .await;
    let huge = fetch_release(&client, &url, &public).await;
    assert!(matches!(huge, Err(UpdateError::Manifest(_))), "{huge:?}");
}
