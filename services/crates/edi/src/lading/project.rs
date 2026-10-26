//! What a project is on the wire: its files, its samples, and what Trenova tells it about the
//! organisation.

use std::collections::BTreeMap;

use serde::Deserialize;

/// One source file.
#[derive(Debug, Clone, Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct SourceFile {
    /// Its path in the project (`maps/tender-in.lading`).
    pub path: String,
    /// Its text.
    pub source: String,
}

/// One sample: a partner's message, or a document.
#[derive(Debug, Clone, Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct Sample {
    /// Its path (`samples/two-stops.x12`).
    pub path: String,
    /// Its text.
    pub content: String,
}

/// A project.
#[derive(Debug, Clone, Default, Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct Project {
    /// The source files.
    #[serde(default)]
    pub files: Vec<SourceFile>,
    /// The samples.
    #[serde(default)]
    pub samples: Vec<Sample>,
}

impl Project {
    /// A sample by path.
    #[must_use]
    pub fn sample(&self, path: &str) -> Option<&Sample> {
        self.samples.iter().find(|sample| sample.path == path)
    }
}

/// The organisation, as `org` reads it.
#[derive(Debug, Clone, Default, Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct Org {
    /// Its name.
    #[serde(default)]
    pub name: String,
    /// Its SCAC.
    #[serde(default)]
    pub scac: String,
    /// Its home time zone.
    #[serde(default)]
    pub zone: String,
}

/// One of the organisation's custom fields.
#[derive(Debug, Clone, Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct CustomField {
    /// Its key, the name maps use (`poNumber`).
    pub name: String,
    /// `Text`, `Int`, `Decimal`, `Bool`, `Date` or `DateTime`.
    #[serde(rename = "type")]
    pub ty: String,
    /// What people call it.
    #[serde(default)]
    pub label: String,
}

/// What Trenova tells a check or a run about the organisation.
#[derive(Debug, Clone, Default, Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct Context {
    /// The organisation.
    #[serde(default)]
    pub org: Org,
    /// Its custom fields, by the record they are on (`shipment`, `invoice`).
    #[serde(default)]
    pub custom_fields: BTreeMap<String, Vec<CustomField>>,
}
