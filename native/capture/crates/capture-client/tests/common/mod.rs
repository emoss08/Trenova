//! A device signed in against a mock server.

#![allow(dead_code)]

use std::sync::Arc;
use std::time::{SystemTime, UNIX_EPOCH};

use capture_client::{AgentInfo, Api, Credential, MemoryStore, SecretStore, Server};
use capture_protocol::api::{Id, TokenPair};
use serde_json::json;
use wiremock::MockServer;

pub fn now() -> i64 {
    i64::try_from(
        SystemTime::now()
            .duration_since(UNIX_EPOCH)
            .expect("clock")
            .as_secs(),
    )
    .expect("secs")
}

pub fn tokens(access: &str, refresh: &str, expires_in: i64) -> TokenPair {
    TokenPair {
        token_type: "Bearer".into(),
        access_token: access.into(),
        access_token_expires_at: now() + expires_in,
        refresh_token: refresh.into(),
        device_id: Id::from("cdev_1"),
        device_name: "DISPATCH-07".into(),
        user_id: Id::from("usr_1"),
        organization_id: Id::from("org_1"),
        business_unit_id: Id::from("bu_1"),
    }
}

pub fn token_json(access: &str, refresh: &str) -> serde_json::Value {
    json!({
        "tokenType": "Bearer",
        "accessToken": access,
        "accessTokenExpiresAt": now() + 900,
        "refreshToken": refresh,
        "deviceId": "cdev_1",
        "deviceName": "DISPATCH-07",
        "userId": "usr_1",
        "organizationId": "org_1",
        "businessUnitId": "bu_1"
    })
}

pub fn agent() -> AgentInfo {
    AgentInfo {
        version: "1.0.0".into(),
        os_version: "Windows 10.0.22631".into(),
    }
}

pub struct Device {
    pub api: Arc<Api>,
    pub store: Arc<MemoryStore>,
}

/// A device whose access token has `expires_in` seconds left.
pub fn device(server: &MockServer, expires_in: i64) -> Device {
    let base = Server::parse(&server.uri()).expect("server");
    let store = Arc::new(MemoryStore::with(Credential {
        server: base.as_str().to_owned(),
        web_base: "https://app.trenova.test".into(),
        tokens: tokens("tcd_at_old", "tcd_rt_old", expires_in),
    }));
    let api = Api::new(base, agent(), Arc::clone(&store) as Arc<dyn SecretStore>).expect("api");
    Device {
        api: Arc::new(api),
        store,
    }
}

pub fn problem(status: u16, kind: &str, detail: &str) -> serde_json::Value {
    json!({"type": kind, "title": kind, "status": status, "detail": detail})
}
