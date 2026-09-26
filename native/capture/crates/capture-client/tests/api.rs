//! The API client's credential handling against a mock server.

mod common;

use capture_client::api::PairingPoll;
use capture_client::pairing::{PairingOutcome, pair};
use capture_client::{ApiError, SecretStore};
use capture_protocol::api::{Architecture, StartPairingRequest};
use common::{device, device_at, problem, token_json};
use serde_json::json;
use tokio_util::sync::CancellationToken;
use wiremock::matchers::{body_partial_json, header, method, path};
use wiremock::{Mock, MockServer, ResponseTemplate};

const IDENTITY: &str = r#"{"device":{"id":"cdev_1","sources":null},"person":{"id":"usr_1","name":"Jordan Doe"},"organization":{"id":"org_1","name":"Acme Freight"}}"#;

#[tokio::test]
async fn a_stale_token_is_refreshed_once_for_every_caller_waiting_on_it() {
    let server = MockServer::start().await;
    Mock::given(method("POST"))
        .and(path("/api/v1/capture/token/refresh/"))
        .and(body_partial_json(
            json!({"refreshToken": "tcd_rt_old", "agentVersion": "1.0.0"}),
        ))
        .respond_with(
            ResponseTemplate::new(200).set_body_json(token_json("tcd_at_new", "tcd_rt_new")),
        )
        .expect(1)
        .mount(&server)
        .await;
    Mock::given(method("GET"))
        .and(path("/api/v1/capture/device/"))
        .and(header("authorization", "Bearer tcd_at_new"))
        .respond_with(ResponseTemplate::new(200).set_body_raw(IDENTITY, "application/json"))
        .expect(3)
        .mount(&server)
        .await;

    let device = device(&server, 10);
    let (a, b, c) = tokio::join!(
        device.api.identity(),
        device.api.identity(),
        device.api.identity()
    );
    for identity in [a, b, c] {
        assert_eq!(
            identity.expect("identity").organization.name,
            "Acme Freight"
        );
    }
    let saved = device.store.load().expect("load").expect("credential");
    assert_eq!(
        saved.tokens.refresh_token, "tcd_rt_new",
        "the rotated credential is kept"
    );
}

#[tokio::test]
async fn a_refused_token_is_refreshed_and_the_call_retried_once() {
    let server = MockServer::start().await;
    Mock::given(method("GET"))
        .and(path("/api/v1/capture/device/"))
        .and(header("authorization", "Bearer tcd_at_old"))
        .respond_with(ResponseTemplate::new(401).set_body_json(problem(
            401,
            "authentication",
            "expired",
        )))
        .expect(1)
        .mount(&server)
        .await;
    Mock::given(method("POST"))
        .and(path("/api/v1/capture/token/refresh/"))
        .respond_with(
            ResponseTemplate::new(200).set_body_json(token_json("tcd_at_new", "tcd_rt_new")),
        )
        .expect(1)
        .mount(&server)
        .await;
    Mock::given(method("GET"))
        .and(path("/api/v1/capture/device/"))
        .and(header("authorization", "Bearer tcd_at_new"))
        .respond_with(ResponseTemplate::new(200).set_body_raw(IDENTITY, "application/json"))
        .expect(1)
        .mount(&server)
        .await;

    let device = device(&server, 900);
    assert_eq!(
        device.api.identity().await.expect("identity").person.name,
        "Jordan Doe"
    );
}

#[tokio::test]
async fn a_revoked_device_is_signed_out_and_forgets_its_credential() {
    let server = MockServer::start().await;
    Mock::given(method("POST"))
        .and(path("/api/v1/capture/token/refresh/"))
        .respond_with(ResponseTemplate::new(400).set_body_json(json!({"error": "invalid_grant"})))
        .expect(1)
        .mount(&server)
        .await;

    let device = device(&server, 0);
    assert!(matches!(
        device.api.identity().await,
        Err(ApiError::SignedOut)
    ));
    assert!(device.store.load().expect("load").is_none());
    assert!(device.api.credential().await.is_none());
    assert!(matches!(
        device.api.identity().await,
        Err(ApiError::NotSignedIn)
    ));
}

#[tokio::test]
async fn an_outdated_agent_and_capture_turned_off_are_told_apart_from_other_refusals() {
    let server = MockServer::start().await;
    Mock::given(method("POST"))
        .and(path("/api/v1/capture/token/refresh/"))
        .respond_with(ResponseTemplate::new(426).set_body_json(json!({
            "error": "agent_outdated", "message": "update", "minimumVersion": "1.4.0"
        })))
        .mount(&server)
        .await;
    let outdated = device(&server, 0);
    match outdated.api.identity().await {
        Err(ApiError::Outdated { minimum_version }) => assert_eq!(minimum_version, "1.4.0"),
        other => panic!("expected outdated, got {other:?}"),
    }
    assert!(
        outdated.store.load().expect("load").is_some(),
        "an outdated device keeps its credential"
    );

    let server = MockServer::start().await;
    let mut disabled = problem(
        422,
        "business-rule-violation",
        "document capture is turned off for this organization",
    );
    disabled["params"] = json!({"reason": "capture_disabled"});
    Mock::given(method("GET"))
        .and(path("/api/v1/capture/device/profiles/"))
        .respond_with(ResponseTemplate::new(422).set_body_json(disabled))
        .mount(&server)
        .await;
    Mock::given(method("GET"))
        .and(path("/api/v1/capture/device/requests/"))
        .respond_with(ResponseTemplate::new(422).set_body_json(problem(422, "validation", "bad")))
        .mount(&server)
        .await;
    let device = device(&server, 900);
    let err = device.api.profiles().await.expect_err("disabled");
    assert!(matches!(err, ApiError::CaptureDisabled));
    assert!(err.blocks_everything());
    let err = device.api.open_requests().await.expect_err("invalid");
    assert!(matches!(err, ApiError::Invalid(_)));
    assert!(!err.blocks_everything() && !err.is_retryable());
}

#[tokio::test]
async fn rate_limits_carry_their_retry_after() {
    let server = MockServer::start().await;
    Mock::given(method("GET"))
        .and(path("/api/v1/capture/device/requests/"))
        .respond_with(ResponseTemplate::new(429).insert_header("Retry-After", "7"))
        .mount(&server)
        .await;
    let device = device(&server, 900);
    let err = device.api.open_requests().await.expect_err("limited");
    assert!(err.is_retryable());
    assert_eq!(err.retry_after(), Some(std::time::Duration::from_secs(7)));
}

fn machine() -> StartPairingRequest {
    StartPairingRequest {
        machine_name: "DISPATCH-07".into(),
        windows_user: "jdoe".into(),
        agent_version: "1.0.0".into(),
        architecture: Architecture::X64,
        os_version: "Windows 10.0.22631".into(),
    }
}

async fn grant(server: &MockServer) {
    Mock::given(method("POST"))
        .and(path("/api/v1/capture/pair/"))
        .and(body_partial_json(
            json!({"machineName": "DISPATCH-07", "architecture": "x64"}),
        ))
        .respond_with(ResponseTemplate::new(200).set_body_json(json!({
            "deviceCode": "dc_secret",
            "userCode": "BCDF-GHJK",
            "verificationUri": "https://app.trenova.test/capture/pair",
            "verificationUriComplete": "https://app.trenova.test/capture/pair?code=BCDF-GHJK",
            "expiresIn": 600,
            "interval": 1
        })))
        .expect(1)
        .mount(server)
        .await;
}

#[tokio::test]
async fn pairing_waits_for_approval_then_signs_in() {
    let server = MockServer::start().await;
    grant(&server).await;
    Mock::given(method("POST"))
        .and(path("/api/v1/capture/pair/token/"))
        .and(body_partial_json(json!({"deviceCode": "dc_secret"})))
        .respond_with(
            ResponseTemplate::new(400).set_body_json(json!({"error": "authorization_pending"})),
        )
        .up_to_n_times(1)
        .expect(1)
        .mount(&server)
        .await;
    Mock::given(method("POST"))
        .and(path("/api/v1/capture/pair/token/"))
        .respond_with(ResponseTemplate::new(200).set_body_json(token_json("tcd_at_1", "tcd_rt_1")))
        .expect(1)
        .mount(&server)
        .await;

    let device = device(&server, 900);
    device.api.sign_out().await;
    let mut shown = None;
    let outcome = pair(
        &device.api,
        &machine(),
        |g| shown = Some(g.user_code.clone()),
        &CancellationToken::new(),
    )
    .await
    .expect("pairing");
    assert_eq!(shown.as_deref(), Some("BCDF-GHJK"));
    let PairingOutcome::Paired(credential) = outcome else {
        panic!("expected paired, got {outcome:?}");
    };
    assert_eq!(credential.web_base, "https://app.trenova.test");
    assert_eq!(device.store.load().expect("load"), Some(*credential));
}

#[tokio::test]
async fn a_denied_pairing_signs_nothing_in() {
    let server = MockServer::start().await;
    grant(&server).await;
    Mock::given(method("POST"))
        .and(path("/api/v1/capture/pair/token/"))
        .respond_with(ResponseTemplate::new(400).set_body_json(json!({"error": "access_denied"})))
        .mount(&server)
        .await;
    let device = device(&server, 900);
    device.api.sign_out().await;
    let outcome = pair(&device.api, &machine(), |_| {}, &CancellationToken::new())
        .await
        .expect("pairing");
    assert_eq!(outcome, PairingOutcome::Denied);
    assert!(device.store.load().expect("load").is_none());
}

#[tokio::test]
async fn poll_answers_map_to_rfc_8628() {
    let server = MockServer::start().await;
    Mock::given(method("POST"))
        .and(path("/api/v1/capture/pair/token/"))
        .respond_with(ResponseTemplate::new(400).set_body_json(json!({"error": "slow_down"})))
        .mount(&server)
        .await;
    let device = device(&server, 900);
    assert_eq!(
        device.api.poll_pairing("dc").await.expect("poll"),
        PairingPoll::SlowDown
    );
}

#[tokio::test]
async fn signing_out_revokes_the_device_and_forgets_it_even_offline() {
    let server = MockServer::start().await;
    Mock::given(method("DELETE"))
        .and(path("/api/v1/capture/device/"))
        .and(header("authorization", "Bearer tcd_at_old"))
        .respond_with(ResponseTemplate::new(204))
        .expect(1)
        .mount(&server)
        .await;
    let device = device(&server, 900);
    device.api.revoke_self().await.expect("signed out");
    assert!(device.store.load().expect("load").is_none());

    let unreachable = device_at("http://127.0.0.1:9");
    let err = unreachable.api.revoke_self().await.expect_err("offline");
    assert!(err.is_retryable());
    assert!(
        unreachable.store.load().expect("load").is_none(),
        "forgotten anyway"
    );
}
