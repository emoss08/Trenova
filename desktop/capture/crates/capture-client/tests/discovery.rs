//! Finding a server from the address a person typed, against mock servers.

mod common;

use capture_client::{DiscoveryError, Server, ServerUrlError, discover};
use serde_json::json;
use wiremock::matchers::{method, path};
use wiremock::{Mock, MockServer, ResponseTemplate};

fn described(name: &str, shared: bool, web: &str) -> serde_json::Value {
    json!({
        "product": "trenova", "edition": if shared { "cloud" } else { "community" },
        "sharedTenancy": shared, "name": name, "version": "0.10.5",
        "webBaseUrl": web, "pairing": !web.is_empty()
    })
}

#[tokio::test]
async fn a_self_hosted_server_answers_at_the_address_entered() {
    let server = MockServer::start().await;
    Mock::given(method("GET"))
        .and(path("/.well-known/trenova-capture.json"))
        .respond_with(ResponseTemplate::new(404))
        .mount(&server)
        .await;
    Mock::given(method("GET"))
        .and(path("/api/v1/capture/discovery/"))
        .respond_with(ResponseTemplate::new(200).set_body_json(described(
            "Trenova",
            false,
            &server.uri(),
        )))
        .mount(&server)
        .await;

    let found = discover(
        &common::agent(),
        Server::parse(&server.uri()).expect("server"),
    )
    .await
    .expect("discovered");
    assert_eq!(found.name, "Trenova");
    assert_eq!(found.edition, "community");
    assert!(!found.shared_tenancy);
    assert!(!found.legacy);
    assert!(found.pairing);
    assert_eq!(found.web_base.as_deref(), Some(server.uri().as_str()));
    assert!(!found.server.api_elsewhere());
    assert_eq!(found.version, "0.10.5");
}

#[tokio::test]
async fn the_web_app_points_at_an_api_on_another_origin() {
    let web = MockServer::start().await;
    let api = MockServer::start().await;
    Mock::given(method("GET"))
        .and(path("/.well-known/trenova-capture.json"))
        .respond_with(ResponseTemplate::new(200).set_body_json(json!({
            "product": "trenova", "apiBaseUrl": api.uri()
        })))
        .mount(&web)
        .await;
    Mock::given(method("GET"))
        .and(path("/api/v1/capture/discovery/"))
        .respond_with(ResponseTemplate::new(200).set_body_json(described(
            "Trenova Cloud",
            true,
            &web.uri(),
        )))
        .mount(&api)
        .await;

    let found = discover(&common::agent(), Server::parse(&web.uri()).expect("server"))
        .await
        .expect("discovered");
    assert_eq!(found.name, "Trenova Cloud");
    assert!(found.shared_tenancy);
    assert!(found.server.api_elsewhere());
    assert_eq!(found.server.as_str(), web.uri());
    assert_eq!(found.server.api_base(), api.uri());
    assert_eq!(
        found.server.api("pair/").as_str(),
        format!("{}/api/v1/capture/pair/", api.uri())
    );
    assert_eq!(found.web_base.as_deref(), Some(web.uri().as_str()));
}

#[tokio::test]
async fn a_relative_api_address_resolves_against_the_entered_one() {
    let server = MockServer::start().await;
    Mock::given(method("GET"))
        .and(path("/tms/.well-known/trenova-capture.json"))
        .respond_with(ResponseTemplate::new(200).set_body_json(json!({
            "product": "trenova", "apiBaseUrl": "/"
        })))
        .mount(&server)
        .await;
    Mock::given(method("GET"))
        .and(path("/api/v1/capture/discovery/"))
        .respond_with(ResponseTemplate::new(200).set_body_json(described("Trenova", false, "")))
        .mount(&server)
        .await;

    let entered = format!("{}/tms", server.uri());
    let found = discover(&common::agent(), Server::parse(&entered).expect("server"))
        .await
        .expect("discovered");
    assert_eq!(found.server.as_str(), entered);
    assert_eq!(found.server.api_base(), server.uri());
    assert!(!found.pairing, "no web address, nothing to approve at");
    assert_eq!(found.web_base, None);
}

#[tokio::test]
async fn the_apps_own_html_shell_at_the_well_known_path_is_ignored() {
    let server = MockServer::start().await;
    Mock::given(method("GET"))
        .and(path("/.well-known/trenova-capture.json"))
        .respond_with(
            ResponseTemplate::new(200).set_body_raw("<!doctype html><html></html>", "text/html"),
        )
        .mount(&server)
        .await;
    Mock::given(method("GET"))
        .and(path("/api/v1/capture/discovery/"))
        .respond_with(ResponseTemplate::new(200).set_body_json(described("Trenova", false, "")))
        .mount(&server)
        .await;

    let found = discover(
        &common::agent(),
        Server::parse(&server.uri()).expect("server"),
    )
    .await
    .expect("discovered");
    assert!(!found.server.api_elsewhere());
}

#[tokio::test]
async fn an_api_address_the_agent_would_refuse_is_refused_here_too() {
    let web = MockServer::start().await;
    Mock::given(method("GET"))
        .and(path("/.well-known/trenova-capture.json"))
        .respond_with(ResponseTemplate::new(200).set_body_json(json!({
            "product": "trenova", "apiBaseUrl": "http://api.acme.test"
        })))
        .mount(&web)
        .await;

    let err = discover(&common::agent(), Server::parse(&web.uri()).expect("server"))
        .await
        .expect_err("refused");
    assert!(
        matches!(
            err,
            DiscoveryError::BadApiAddress {
                reason: ServerUrlError::NotHttps,
                ..
            }
        ),
        "{err:?}"
    );
}

#[tokio::test]
async fn a_server_older_than_discovery_is_still_a_trenova() {
    let server = MockServer::start().await;
    Mock::given(method("GET"))
        .and(path("/api/v1/capture/releases/latest/"))
        .respond_with(ResponseTemplate::new(404).set_body_raw(
            r#"{"title":"No Trenova Capture release is published","status":404}"#,
            "application/problem+json",
        ))
        .mount(&server)
        .await;

    let found = discover(
        &common::agent(),
        Server::parse(&server.uri()).expect("server"),
    )
    .await
    .expect("discovered");
    assert!(found.legacy);
    assert_eq!(found.name, "Trenova");
    assert!(found.pairing);
    assert_eq!(found.web_base, None);
}

#[tokio::test]
async fn some_other_web_server_is_not_a_trenova() {
    let server = MockServer::start().await;
    Mock::given(method("GET"))
        .respond_with(ResponseTemplate::new(404).set_body_raw("404 page not found", "text/plain"))
        .mount(&server)
        .await;

    let err = discover(
        &common::agent(),
        Server::parse(&server.uri()).expect("server"),
    )
    .await
    .expect_err("not a trenova");
    assert!(matches!(err, DiscoveryError::NotTrenova { .. }), "{err:?}");
    assert!(
        err.to_string()
            .contains("does not look like a Trenova server")
    );

    let page = MockServer::start().await;
    Mock::given(method("GET"))
        .respond_with(ResponseTemplate::new(200).set_body_raw("<html>Welcome</html>", "text/html"))
        .mount(&page)
        .await;
    let err = discover(
        &common::agent(),
        Server::parse(&page.uri()).expect("server"),
    )
    .await
    .expect_err("not a trenova");
    assert!(matches!(err, DiscoveryError::NotTrenova { .. }), "{err:?}");
}

#[tokio::test]
async fn nothing_answering_says_so() {
    let err = discover(
        &common::agent(),
        Server::parse("http://127.0.0.1:9").expect("server"),
    )
    .await
    .expect_err("unreachable");
    assert!(matches!(err, DiscoveryError::Unreachable { .. }), "{err:?}");
    assert!(err.to_string().contains("nothing answered at"));

    let busy = MockServer::start().await;
    Mock::given(method("GET"))
        .respond_with(ResponseTemplate::new(503))
        .mount(&busy)
        .await;
    let err = discover(
        &common::agent(),
        Server::parse(&busy.uri()).expect("server"),
    )
    .await
    .expect_err("refused");
    assert!(
        matches!(err, DiscoveryError::Refused { status: 503, .. }),
        "{err:?}"
    );
}
