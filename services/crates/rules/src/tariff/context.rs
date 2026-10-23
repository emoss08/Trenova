//! What every `tariff.*` operation that reads a file is given besides the file: the
//! organisation's libraries it may import, its rate tables, its accessorial charge codes and its
//! settings. `tariff.check`, `tariff.evaluate`, `tariff.test` and the editor operations all take
//! this shape under `"context"`, so the hosts build it once.

use super::schema::{ChargeCode, Settings, TableShape};
use serde::Deserialize;

/// One organisation library a file may import with `use org.<name>`.
#[derive(Debug, Clone, Deserialize)]
#[serde(rename_all = "camelCase", deny_unknown_fields)]
pub struct Library {
    /// The module path: `org.accessorials`.
    pub path: String,
    /// The pinned version's ID, echoed back in the imports `tariff.check` reports.
    #[serde(default)]
    pub version: Option<String>,
    /// The library's source.
    pub source: String,
}

/// The context of a file.
#[derive(Debug, Clone, Default, Deserialize)]
#[serde(rename_all = "camelCase", deny_unknown_fields)]
pub struct Context {
    /// The organisation libraries the file may import, by path.
    #[serde(default)]
    pub libraries: Vec<Library>,
    /// The organisation's active rate tables. The checker reads their shapes; the evaluator
    /// also reads their cells.
    #[serde(default)]
    pub tables: Vec<TableShape>,
    /// The organisation's accessorial charge codes. `None` means the host has not loaded them,
    /// so codes are not checked (the editor's first paint); an empty list means there are none.
    #[serde(default)]
    pub codes: Option<Vec<ChargeCode>>,
    /// How the organisation stores what the schema exposes.
    #[serde(default)]
    pub settings: Settings,
}
