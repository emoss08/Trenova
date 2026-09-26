//! The device credential and where it is kept.

use capture_protocol::api::TokenPair;
use serde::{Deserialize, Serialize};

/// Everything the agent needs to act as its paired device.
#[derive(Clone, Debug, PartialEq, Eq, Serialize, Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct Credential {
    /// The server it was paired with, as [`crate::Server::as_str`] writes it.
    pub server: String,
    /// The web app, for links a person opens.
    pub web_base: String,
    pub tokens: TokenPair,
}

/// Where a credential lives between runs. On Windows this is Credential
/// Manager, protected by DPAPI to the signed-in Windows user.
pub trait SecretStore: Send + Sync {
    fn load(&self) -> std::io::Result<Option<Credential>>;
    fn save(&self, credential: &Credential) -> std::io::Result<()>;
    fn clear(&self) -> std::io::Result<()>;
}

/// A store held in memory only, for a run that must not persist anything and
/// for tests.
#[derive(Debug, Default)]
pub struct MemoryStore {
    credential: std::sync::Mutex<Option<Credential>>,
}

impl MemoryStore {
    pub fn with(credential: Credential) -> Self {
        Self {
            credential: std::sync::Mutex::new(Some(credential)),
        }
    }
}

impl SecretStore for MemoryStore {
    fn load(&self) -> std::io::Result<Option<Credential>> {
        Ok(self
            .credential
            .lock()
            .unwrap_or_else(std::sync::PoisonError::into_inner)
            .clone())
    }

    fn save(&self, credential: &Credential) -> std::io::Result<()> {
        *self
            .credential
            .lock()
            .unwrap_or_else(std::sync::PoisonError::into_inner) = Some(credential.clone());
        Ok(())
    }

    fn clear(&self) -> std::io::Result<()> {
        *self
            .credential
            .lock()
            .unwrap_or_else(std::sync::PoisonError::into_inner) = None;
        Ok(())
    }
}
