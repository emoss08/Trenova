//! What the agent knows, as the tray and the window show it, and what they
//! ask of it.
//!
//! The agent owns the state and changes it; the tray and the window only
//! read a copy when they redraw. Every change calls [`Ui::refresh`], anything
//! a person should hear about right away goes through [`Ui::notify`] (and is
//! kept, so a notification missed is not lost), and the moments the window
//! should come forward for go through [`Ui::attention`].

use std::collections::VecDeque;
use std::path::PathBuf;
use std::sync::{Arc, Mutex, PoisonError};
use std::time::SystemTime;

use capture_client::spool::{PictureRef, PictureSize, RefusedBatch};
use capture_protocol::api::{BatchSource, CaptureProfile, Id, SourceInfo, SourceProtocol};
use capture_protocol::helper::ScanCondition;

/// How many sent batches the tray lists.
pub const RECENT: usize = 5;
/// How many past notices the window keeps.
pub const MESSAGES: usize = 20;

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

/// A newer release, and what is being done about it.
#[derive(Clone, Debug, PartialEq, Eq)]
pub struct UpdateState {
    pub version: String,
    /// Where the installer is, for a person who installs it themselves.
    pub download_url: String,
    pub status: UpdateStatus,
}

#[derive(Clone, Copy, Debug, PartialEq, Eq)]
pub enum UpdateStatus {
    /// It can be installed from the menu, and will be after the next check
    /// finds nothing scanning.
    Available,
    /// The organization or this computer's policy leaves installing to IT.
    AskAdministrator,
    /// The updater has been asked to install it.
    Installing,
    /// It needs a newer Windows than this one.
    WindowsTooOld,
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

/// The scan running now.
#[derive(Clone, Debug, PartialEq, Eq)]
pub struct ActiveScan {
    /// The spooled batch it scans into.
    pub key: String,
    /// The scanner's name.
    pub label: String,
    pub pages: u32,
    /// Asked for from the web app, to file onto a record.
    pub requested: bool,
    /// The person asked it to stop; it ends at the next page.
    pub stopping: bool,
}

/// A batch on this computer the server does not have in full yet.
#[derive(Clone, Debug, PartialEq, Eq)]
pub struct WaitingBatch {
    pub key: String,
    pub label: String,
    pub source: BatchSource,
    /// Pages not yet sent.
    pub pages: u32,
    /// When it was spooled, in Unix milliseconds.
    pub created_at: i64,
    /// Scanning has ended, so it can be sent in full.
    pub complete: bool,
    /// Held for the person to look over before it is sent.
    pub held: bool,
    /// For a record the person chose in Trenova, rather than intake.
    pub requested: bool,
    /// A printed job, sent whole.
    pub printed: bool,
    /// None of its pages has reached the server, so they can still be
    /// turned or taken out while it is held.
    pub editable: bool,
    /// Its pages that have pictures.
    pub pictures: Vec<PictureRef>,
}

/// Something the person was told, kept for the window.
#[derive(Clone, Debug, PartialEq, Eq)]
pub struct Message {
    pub notice: Notice,
    pub at: SystemTime,
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
    pub scan: Option<ActiveScan>,
    pub paused: Vec<PausedBatch>,
    pub pages_waiting: u32,
    /// Batches waiting to be sent, oldest first.
    pub waiting: Vec<WaitingBatch>,
    /// Batches the server refused, kept for the person to decide on, most
    /// recent first.
    pub refused: Vec<RefusedBatch>,
    pub recent: VecDeque<RecentBatch>,
    /// What the person was told, newest first.
    pub messages: VecDeque<Message>,
    /// The server address is set by policy and cannot be changed here.
    pub server_locked: bool,
    /// The newest version the organization requires, when this one is older.
    pub update_required: Option<String>,
    /// A newer release, once one is known.
    pub update: Option<UpdateState>,
    /// The print service is installed but its printer is not.
    pub printer_missing: bool,
    /// The print service is installed, so printing into Trenova is offered.
    pub printing: bool,
    /// The person chose not to be shown routine notifications.
    pub routine_muted: bool,
    /// Scans and prints wait for the person to look them over before they
    /// are sent.
    pub review_before_sending: bool,
    /// An administrator set review before sending, so the person cannot.
    pub review_locked: bool,
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
    /// News that nothing needs doing about (something sent, the connection
    /// back), which a person can choose not to be shown.
    pub routine: bool,
}

/// A moment the window comes forward for, if it is not already showing.
#[derive(Clone, Copy, Debug, PartialEq, Eq)]
pub enum Attention {
    /// No server address yet: nothing works until one is entered.
    SetUp,
    /// This computer is waiting to be approved; the code is shown.
    SignIn,
    ScanStarted,
    /// The scanner stopped partway and needs the person.
    ScanPaused,
    ScanEnded,
    /// The server refused a batch.
    Refused,
    /// A scan or a print is held for the person to look over.
    Review,
}

/// How an attempt to add the Trenova printer ended.
#[derive(Clone, Copy, Debug, PartialEq, Eq)]
pub enum PrinterAttempt {
    Added,
    /// The person said no at the Windows prompt.
    Declined,
    Failed,
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
    /// Stop the scan running now at the next page; what was scanned is sent.
    StopScan,
    /// Send a refused batch again, to intake.
    Retry(String),
    /// Delete a refused batch and its pages.
    Discard(String),
    /// Write a refused batch's pages as PDFs into a new folder under
    /// `into`, and show it.
    Save {
        key: String,
        into: PathBuf,
    },
    RefreshScanners,
    /// How adding the Trenova printer from the menu went.
    PrinterSetUp(PrinterAttempt),
    /// Install the release the menu offers.
    Update,
    /// Turn a page of a held batch by `degrees` clockwise.
    RotatePage {
        key: String,
        page: u32,
        degrees: i32,
    },
    /// Take a page out of a held batch.
    DeletePage {
        key: String,
        page: u32,
    },
    /// Send a held batch as it now is.
    SendHeld(String),
    /// Delete a held batch and its pages.
    DiscardHeld(String),
    /// Scan more pages onto the end of a held batch.
    ScanMore {
        key: String,
        source: String,
        protocol: SourceProtocol,
        profile: Option<Id>,
    },
    /// Read pages' pictures for the window.
    Pictures(PicturesRequest),
    Quit,
}

/// Pages' pictures the window asked for.
#[derive(Clone, Debug, PartialEq, Eq)]
pub struct PicturesRequest {
    pub key: String,
    pub size: PictureSize,
    pub pages: Vec<u32>,
}

/// Pictures read for the window, JPEG, by page; a page whose picture could
/// not be read is left out.
#[derive(Clone, Debug, PartialEq, Eq)]
pub struct Pictures {
    pub key: String,
    pub size: PictureSize,
    pub pictures: Vec<(u32, Vec<u8>)>,
}

/// The tray and the window, as the agent sees them.
pub trait Ui: Send + Sync {
    /// The snapshot changed; redraw.
    fn refresh(&self);
    /// Tell the person something now.
    fn notify(&self, notice: Notice);
    /// Something happened the window should come forward for.
    fn attention(&self, attention: Attention) {
        let _ = attention;
    }
    /// Pictures the window asked for are ready.
    fn pictures(&self, pictures: Pictures) {
        let _ = pictures;
    }
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

    /// Tells the person now, and keeps it for the window.
    pub fn notify(&self, notice: Notice) {
        let message = Message {
            notice: notice.clone(),
            at: SystemTime::now(),
        };
        self.update(|s| {
            s.messages.push_front(message);
            s.messages.truncate(MESSAGES);
        });
        self.ui.notify(notice);
    }

    pub fn attention(&self, attention: Attention) {
        self.ui.attention(attention);
    }

    pub fn pictures(&self, pictures: Pictures) {
        self.ui.pictures(pictures);
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
