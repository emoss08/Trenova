//! What the agent knows, as the tray shows it, and what the tray asks of it.
//!
//! The agent owns the state and changes it; the tray only reads a copy when
//! it redraws. Every change calls [`Ui::refresh`], and anything a person
//! should hear about right away goes through [`Ui::notify`].

use std::collections::VecDeque;
use std::path::PathBuf;
use std::sync::{Arc, Mutex, PoisonError};
use std::time::SystemTime;

use capture_protocol::api::{CaptureProfile, Id, SourceInfo, SourceProtocol};
use capture_protocol::helper::ScanCondition;

/// How many sent batches the tray lists.
pub const RECENT: usize = 5;

/// Where the agent stands with the server.
#[derive(Clone, Debug, Default, PartialEq, Eq)]
pub enum Connection {
    /// No server address yet.
    #[default]
    NeedsServer,
    SignedOut,
    /// Waiting for the person to approve this computer.
    Pairing {
        code: String,
        url: String,
    },
    Connecting,
    Online,
    /// The connection dropped; it is being retried.
    Offline {
        reason: String,
    },
    /// Nothing can be sent until something changes: an update, capture
    /// turned back on, a permission restored.
    Blocked {
        reason: String,
    },
}

/// A batch the server has in full.
#[derive(Clone, Debug, PartialEq, Eq)]
pub struct RecentBatch {
    pub label: String,
    pub pages: u32,
    pub link: String,
    pub at: SystemTime,
    pub requested: bool,
}

/// A scan the scanner stopped partway, waiting for the person to continue
/// or finish it.
#[derive(Clone, Debug, PartialEq, Eq)]
pub struct PausedBatch {
    pub key: String,
    pub label: String,
    pub pages: u32,
    pub condition: ScanCondition,
}

#[derive(Clone, Debug, Default)]
pub struct Snapshot {
    pub server: Option<String>,
    pub connection: Connection,
    pub person: Option<String>,
    pub organization: Option<String>,
    pub web_base: Option<String>,
    pub sources: Vec<SourceInfo>,
    pub profiles: Vec<CaptureProfile>,
    /// What is being scanned right now.
    pub scanning: Option<String>,
    pub paused: Vec<PausedBatch>,
    pub pages_waiting: u32,
    pub failed: u32,
    pub failed_dir: Option<PathBuf>,
    pub recent: VecDeque<RecentBatch>,
    /// The newest version the organization requires, when this one is older.
    pub update_required: Option<String>,
}

impl Snapshot {
    pub fn signed_in(&self) -> bool {
        matches!(
            self.connection,
            Connection::Connecting
                | Connection::Online
                | Connection::Offline { .. }
                | Connection::Blocked { .. }
        )
    }

    /// The intake queue, or one batch in it.
    pub fn intake_link(&self, batch: Option<&Id>) -> Option<String> {
        let base = self.web_base.as_deref()?;
        Some(match batch {
            Some(id) => format!("{base}/intake?batch={}", encode(id.as_str())),
            None => format!("{base}/intake"),
        })
    }
}

fn encode(value: &str) -> String {
    value
        .bytes()
        .map(|b| {
            if b.is_ascii_alphanumeric() || b == b'_' || b == b'-' {
                (b as char).to_string()
            } else {
                format!("%{b:02X}")
            }
        })
        .collect()
}

/// How loudly to tell the person.
#[derive(Clone, Copy, Debug, PartialEq, Eq)]
pub enum Severity {
    Info,
    Warning,
    Error,
}

/// Something to tell the person, as a notification.
#[derive(Clone, Debug, PartialEq, Eq)]
pub struct Notice {
    pub title: String,
    pub body: String,
    pub severity: Severity,
    /// Opened when the notification is clicked.
    pub link: Option<String>,
}

/// What the tray asks the agent to do.
#[derive(Clone, Debug, PartialEq, Eq)]
pub enum Command {
    SetServer(String),
    SignIn,
    CancelSignIn,
    SignOut,
    /// "Scan to intake": a scan nobody asked for from the web app.
    Scan {
        source: String,
        protocol: SourceProtocol,
        profile: Option<Id>,
    },
    /// Carry on scanning into a batch the scanner stopped.
    Continue(String),
    /// Send a stopped batch as it is.
    Finish(String),
    RefreshScanners,
    Quit,
}

/// The tray, as the agent sees it.
pub trait Ui: Send + Sync {
    /// The snapshot changed; redraw.
    fn refresh(&self);
    /// Tell the person something now.
    fn notify(&self, notice: Notice);
}

/// The state and the tray it is shown in.
pub struct Shared {
    snapshot: Mutex<Snapshot>,
    ui: Arc<dyn Ui>,
}

impl std::fmt::Debug for Shared {
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        f.debug_struct("Shared").finish_non_exhaustive()
    }
}

impl Shared {
    pub fn new(ui: Arc<dyn Ui>) -> Arc<Self> {
        Arc::new(Self {
            snapshot: Mutex::new(Snapshot::default()),
            ui,
        })
    }

    pub fn snapshot(&self) -> Snapshot {
        self.snapshot
            .lock()
            .unwrap_or_else(PoisonError::into_inner)
            .clone()
    }

    /// Changes the state and redraws the tray.
    pub fn update(&self, change: impl FnOnce(&mut Snapshot)) {
        {
            let mut snapshot = self.snapshot.lock().unwrap_or_else(PoisonError::into_inner);
            change(&mut snapshot);
        }
        self.ui.refresh();
    }

    pub fn notify(&self, notice: Notice) {
        self.ui.notify(notice);
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn intake_links_escape_the_batch_id() {
        let snapshot = Snapshot {
            web_base: Some("https://app.acme.com".into()),
            ..Snapshot::default()
        };
        assert_eq!(
            snapshot
                .intake_link(Some(&Id::from("cbat_01J&x=1")))
                .as_deref(),
            Some("https://app.acme.com/intake?batch=cbat_01J%26x%3D1")
        );
        assert_eq!(
            snapshot.intake_link(None).as_deref(),
            Some("https://app.acme.com/intake")
        );
        assert_eq!(Snapshot::default().intake_link(None), None);
    }
}
