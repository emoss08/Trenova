//! What the Trenova Capture window shows, and what its page may ask for.
//!
//! The window is a local page in `WebView2`. Everything it shows is decided
//! here from a snapshot of the agent's state and handed over as JSON, so the
//! page only draws. What the page sends back is parsed strictly and checked
//! against the same snapshot: it names a batch by its key, a scanner by its
//! name, a link by one the snapshot already holds, and nothing it sends is
//! used as an address to open or a path to write. A message that does not
//! match is dropped.

use std::time::{SystemTime, UNIX_EPOCH};

use capture_protocol::api::{BatchSource, PixelType, ProfileStatus, SourceProtocol};
use capture_protocol::helper::ScanCondition;
use serde::{Deserialize, Serialize};

use crate::state::{Command, Connection, Severity, Snapshot, UpdateStatus};

/// The longest server address the page may send.
const MAX_ADDRESS: usize = 2048;

/// How a line is coloured, from the design system's tones.
#[derive(Clone, Copy, Debug, PartialEq, Eq, Serialize)]
#[serde(rename_all = "camelCase")]
pub enum Tone {
    Neutral,
    Info,
    Success,
    Warning,
    Danger,
}

#[derive(Clone, Debug, PartialEq, Eq, Serialize)]
#[serde(rename_all = "camelCase")]
pub struct Status {
    pub tone: Tone,
    pub title: String,
    pub detail: String,
}

#[derive(Clone, Debug, PartialEq, Eq, Serialize)]
#[serde(rename_all = "camelCase")]
pub struct Account {
    pub person: Option<String>,
    pub organization: Option<String>,
    pub server: Option<String>,
    pub server_locked: bool,
    pub signed_in: bool,
}

#[derive(Clone, Copy, Debug, PartialEq, Eq, Serialize)]
#[serde(rename_all = "camelCase")]
pub enum Stage {
    NeedsServer,
    SignedOut,
    Pairing,
    Connecting,
    Online,
    Offline,
    Blocked,
}

#[derive(Clone, Debug, PartialEq, Eq, Serialize)]
#[serde(rename_all = "camelCase")]
pub struct ScanView {
    pub label: String,
    pub pages: u32,
    pub requested: bool,
    pub stopping: bool,
}

#[derive(Clone, Debug, PartialEq, Eq, Serialize)]
#[serde(rename_all = "camelCase")]
pub struct PausedView {
    pub key: String,
    pub label: String,
    pub pages: u32,
    pub problem: String,
}

#[derive(Clone, Debug, PartialEq, Eq, Serialize)]
#[serde(rename_all = "camelCase")]
pub struct ScannerView {
    pub name: String,
    pub protocol: &'static str,
    pub title: String,
    pub is_default: bool,
}

#[derive(Clone, Debug, PartialEq, Eq, Serialize)]
#[serde(rename_all = "camelCase")]
pub struct ProfileView {
    pub id: String,
    pub name: String,
    pub is_default: bool,
    pub summary: String,
}

#[derive(Clone, Copy, Debug, PartialEq, Eq, Serialize)]
#[serde(rename_all = "camelCase")]
pub enum Kind {
    Scan,
    Print,
}

impl From<BatchSource> for Kind {
    fn from(source: BatchSource) -> Self {
        match source {
            BatchSource::Print => Self::Print,
            _ => Self::Scan,
        }
    }
}

#[derive(Clone, Debug, PartialEq, Eq, Serialize)]
#[serde(rename_all = "camelCase")]
pub struct RefusedView {
    pub key: String,
    pub label: String,
    pub kind: Kind,
    pub pages: u32,
    pub reason: String,
    pub refused_at: i64,
    pub readable: bool,
}

#[derive(Clone, Debug, PartialEq, Eq, Serialize)]
#[serde(rename_all = "camelCase")]
pub struct WaitingView {
    pub key: String,
    pub label: String,
    pub kind: Kind,
    pub pages: u32,
    pub created_at: i64,
    /// Why it has not reached Trenova yet.
    pub state: String,
}

#[derive(Clone, Debug, PartialEq, Eq, Serialize)]
#[serde(rename_all = "camelCase")]
pub struct RecentView {
    pub label: String,
    pub pages: u32,
    pub at: i64,
    pub requested: bool,
    pub link: Option<String>,
}

#[derive(Clone, Debug, PartialEq, Eq, Serialize)]
#[serde(rename_all = "camelCase")]
pub struct MessageView {
    pub title: String,
    pub body: String,
    pub tone: Tone,
    pub at: i64,
    pub link: Option<String>,
}

#[derive(Clone, Debug, PartialEq, Eq, Serialize)]
#[serde(rename_all = "camelCase")]
pub struct UpdateView {
    pub version: String,
    pub status: &'static str,
}

/// Everything the page draws.
#[derive(Clone, Debug, PartialEq, Eq, Serialize)]
#[serde(rename_all = "camelCase")]
pub struct View {
    pub version: String,
    pub stage: Stage,
    pub status: Status,
    pub account: Account,
    pub pairing_code: Option<String>,
    pub scan: Option<ScanView>,
    pub paused: Vec<PausedView>,
    /// A new scan to intake can be started.
    pub can_scan: bool,
    pub scanners: Vec<ScannerView>,
    pub profiles: Vec<ProfileView>,
    pub refused: Vec<RefusedView>,
    pub waiting: Vec<WaitingView>,
    pub recent: Vec<RecentView>,
    pub messages: Vec<MessageView>,
    pub printer_missing: bool,
    pub update: Option<UpdateView>,
    pub update_required: Option<String>,
    pub can_open_intake: bool,
}

fn millis(at: SystemTime) -> i64 {
    at.duration_since(UNIX_EPOCH)
        .map_or(0, |d| i64::try_from(d.as_millis()).unwrap_or(i64::MAX))
}

fn pages(count: u32) -> String {
    if count == 1 {
        "1 page".to_owned()
    } else {
        format!("{count} pages")
    }
}

fn protocol_name(protocol: SourceProtocol) -> &'static str {
    match protocol {
        SourceProtocol::Wia => "wia",
        _ => "twain",
    }
}

/// What a person does about a stopped scanner.
pub fn condition_problem(condition: ScanCondition) -> &'static str {
    match condition {
        ScanCondition::PaperJam => "The paper jammed. Clear it, then continue.",
        ScanCondition::DoubleFeed => {
            "More than one sheet went through at once. Put them back, then continue."
        }
        ScanCondition::CoverOpen => "The scanner's cover is open. Close it, then continue.",
        ScanCondition::FeederEmpty => "The feeder is empty. Load the rest, then continue.",
        ScanCondition::CanceledByOperator => "Scanning was cancelled at the scanner.",
    }
}

fn profile_summary(dpi: u32, pixel_type: PixelType, duplex: bool) -> String {
    let colour = match pixel_type {
        PixelType::BlackWhite => Some("Black and white"),
        PixelType::Grayscale => Some("Grayscale"),
        PixelType::Color => Some("Colour"),
        PixelType::Unknown => None,
    };
    let sides = if duplex { "both sides" } else { "one side" };
    match colour {
        Some(colour) => format!("{dpi} dpi, {colour}, {sides}"),
        None => format!("{dpi} dpi, {sides}"),
    }
}

impl Snapshot {
    fn stage(&self) -> Stage {
        match self.connection {
            Connection::NeedsServer => Stage::NeedsServer,
            Connection::SignedOut => Stage::SignedOut,
            Connection::Pairing { .. } => Stage::Pairing,
            Connection::Connecting => Stage::Connecting,
            Connection::Online => Stage::Online,
            Connection::Offline { .. } => Stage::Offline,
            Connection::Blocked { .. } => Stage::Blocked,
        }
    }

    fn status(&self) -> Status {
        let status = |tone, title: &str, detail: String| Status {
            tone,
            title: title.to_owned(),
            detail,
        };
        let who = match (&self.person, &self.organization) {
            (Some(person), Some(org)) => format!("Signed in as {person} at {org}."),
            (Some(person), None) => format!("Signed in as {person}."),
            _ => String::new(),
        };
        let kept = if self.pages_waiting > 0 {
            format!(
                "{} {} kept on this computer until then.",
                pages(self.pages_waiting),
                if self.pages_waiting == 1 { "is" } else { "are" }
            )
        } else {
            String::new()
        };
        match &self.connection {
            Connection::NeedsServer => status(
                Tone::Warning,
                "Connect to Trenova",
                "Enter the address your organization uses for Trenova to begin.".into(),
            ),
            Connection::SignedOut => status(
                Tone::Warning,
                "Not signed in",
                format!("Sign in to scan and print into Trenova. {kept}")
                    .trim_end()
                    .to_owned(),
            ),
            Connection::Pairing { .. } => status(
                Tone::Info,
                "Approve this computer",
                "Trenova opened in your browser. Check the code there matches this one, then approve it."
                    .into(),
            ),
            Connection::Connecting => status(Tone::Neutral, "Connecting to Trenova", who),
            Connection::Online => match &self.scan {
                Some(scan) if scan.stopping => status(
                    Tone::Info,
                    "Stopping the scan",
                    format!("The {} scanned so far will be sent.", pages(scan.pages)),
                ),
                Some(scan) => status(
                    Tone::Info,
                    "Scanning",
                    format!("{} from {}.", pages(scan.pages), scan.label),
                ),
                None if !self.refused.is_empty() => status(
                    Tone::Danger,
                    "Something was not sent",
                    "Trenova refused it. Send it again, save a copy, or discard it below.".into(),
                ),
                None if !self.paused.is_empty() => status(
                    Tone::Warning,
                    "A scan is waiting for you",
                    "The scanner stopped partway. Continue or finish it below.".into(),
                ),
                None => status(Tone::Success, "Ready", who),
            },
            Connection::Offline { .. } => status(
                Tone::Warning,
                "Offline, reconnecting",
                format!("Trenova Capture keeps trying. {kept}")
                    .trim_end()
                    .to_owned(),
            ),
            Connection::Blocked { reason } => status(
                Tone::Danger,
                "Sending is paused",
                format!("{reason} {kept}").trim_end().to_owned(),
            ),
        }
    }

    fn waiting_state(&self) -> &'static str {
        match self.connection {
            Connection::Online => "Sending",
            Connection::Connecting => "Waiting to connect",
            Connection::Offline { .. } => "Waiting for the connection",
            Connection::Blocked { .. } => "Paused",
            Connection::NeedsServer | Connection::SignedOut | Connection::Pairing { .. } => {
                "Waiting for you to sign in"
            }
        }
    }

    fn profile_views(&self) -> Vec<ProfileView> {
        let mut profiles: Vec<_> = self
            .profiles
            .iter()
            .filter(|p| p.status != ProfileStatus::Inactive)
            .map(|p| ProfileView {
                id: p.id.as_str().to_owned(),
                name: p.name.clone(),
                is_default: p.is_default,
                summary: profile_summary(p.dpi, p.pixel_type, p.duplex),
            })
            .collect();
        profiles.sort_by_key(|p| !p.is_default);
        profiles
    }

    fn scanner_views(&self) -> Vec<ScannerView> {
        self.sources
            .iter()
            .map(|source| ScannerView {
                name: source.name.clone(),
                protocol: protocol_name(source.protocol),
                title: match source.protocol {
                    SourceProtocol::Wia => format!("{} (WIA)", source.name),
                    _ => source.name.clone(),
                },
                is_default: source.is_default,
            })
            .collect()
    }

    fn refused_views(&self) -> Vec<RefusedView> {
        self.refused
            .iter()
            .map(|batch| RefusedView {
                key: batch.key.clone(),
                label: if batch.label.is_empty() {
                    "Unreadable batch".to_owned()
                } else {
                    batch.label.clone()
                },
                kind: batch.source.into(),
                pages: batch.pages,
                reason: if batch.readable {
                    batch.reason.clone()
                } else {
                    format!(
                        "Its record on this computer is damaged, so it can only be discarded. ({})",
                        batch.reason
                    )
                },
                refused_at: batch.refused_at,
                readable: batch.readable,
            })
            .collect()
    }

    /// Batches that have finished scanning and wait to be sent; the one
    /// scanning now is shown as the scan.
    fn waiting_views(&self) -> Vec<WaitingView> {
        let scanning_key = self.scan.as_ref().map(|scan| scan.key.as_str());
        let state = self.waiting_state();
        self.waiting
            .iter()
            .filter(|batch| {
                batch.complete && Some(batch.key.as_str()) != scanning_key && batch.pages > 0
            })
            .map(|batch| WaitingView {
                key: batch.key.clone(),
                label: batch.label.clone(),
                kind: batch.source.into(),
                pages: batch.pages,
                created_at: batch.created_at,
                state: state.to_owned(),
            })
            .collect()
    }

    fn recent_views(&self) -> Vec<RecentView> {
        self.recent
            .iter()
            .map(|batch| RecentView {
                label: batch.label.clone(),
                pages: batch.pages,
                at: millis(batch.at),
                requested: batch.requested,
                link: (!batch.link.is_empty()).then(|| batch.link.clone()),
            })
            .collect()
    }

    fn message_views(&self) -> Vec<MessageView> {
        self.messages
            .iter()
            .map(|message| MessageView {
                title: message.notice.title.clone(),
                body: message.notice.body.clone(),
                tone: match message.notice.severity {
                    Severity::Info => Tone::Info,
                    Severity::Warning => Tone::Warning,
                    Severity::Error => Tone::Danger,
                },
                at: millis(message.at),
                link: message.notice.link.clone(),
            })
            .collect()
    }

    /// What the window shows now.
    pub fn view(&self, version: &str) -> View {
        let online = self.signed_in() && !matches!(self.connection, Connection::Blocked { .. });
        View {
            version: version.to_owned(),
            stage: self.stage(),
            status: self.status(),
            account: Account {
                person: self.person.clone(),
                organization: self.organization.clone(),
                server: self.server.clone(),
                server_locked: self.server_locked,
                signed_in: self.signed_in(),
            },
            pairing_code: match &self.connection {
                Connection::Pairing { code, .. } => Some(code.clone()),
                _ => None,
            },
            scan: self.scan.as_ref().map(|scan| ScanView {
                label: scan.label.clone(),
                pages: scan.pages,
                requested: scan.requested,
                stopping: scan.stopping,
            }),
            paused: self
                .paused
                .iter()
                .map(|paused| PausedView {
                    key: paused.key.clone(),
                    label: paused.label.clone(),
                    pages: paused.pages,
                    problem: condition_problem(paused.condition).to_owned(),
                })
                .collect(),
            can_scan: online && self.scan.is_none() && !self.sources.is_empty(),
            scanners: self.scanner_views(),
            profiles: self.profile_views(),
            refused: self.refused_views(),
            waiting: self.waiting_views(),
            recent: self.recent_views(),
            messages: self.message_views(),
            printer_missing: self.printer_missing,
            update: self.update.as_ref().map(|update| UpdateView {
                version: update.version.clone(),
                status: match update.status {
                    UpdateStatus::Available => "available",
                    UpdateStatus::AskAdministrator => "askAdministrator",
                    UpdateStatus::Installing => "installing",
                    UpdateStatus::WindowsTooOld => "windowsTooOld",
                },
            }),
            update_required: self.update_required.clone(),
            can_open_intake: self.signed_in() && self.web_base.is_some(),
        }
    }

    /// Whether `link` is one this snapshot shows, and so one the page may
    /// ask to open.
    fn shows_link(&self, link: &str) -> bool {
        !link.is_empty()
            && (self.recent.iter().any(|batch| batch.link == link)
                || self
                    .messages
                    .iter()
                    .any(|message| message.notice.link.as_deref() == Some(link)))
    }

    /// What a message from the page asks for, checked against what the
    /// window shows. `None` when it names something that is not there.
    pub fn resolve(&self, message: PageMessage) -> Option<WindowAction> {
        let command = |command| Some(WindowAction::Command(command));
        let refused = |key: &str| self.refused.iter().any(|batch| batch.key == key);
        let readable = |key: &str| {
            self.refused
                .iter()
                .any(|batch| batch.key == key && batch.readable)
        };
        let paused = |key: &str| self.paused.iter().any(|batch| batch.key == key);
        match message {
            PageMessage::Ready {} => Some(WindowAction::Redraw),
            PageMessage::Close {} => Some(WindowAction::Close),
            PageMessage::SignIn {} => command(Command::SignIn),
            PageMessage::CancelSignIn {} => command(Command::CancelSignIn),
            PageMessage::SignOut {} => command(Command::SignOut),
            PageMessage::SetServer { address } => {
                let address = address.trim();
                (!self.server_locked && !address.is_empty() && address.len() <= MAX_ADDRESS)
                    .then(|| WindowAction::Command(Command::SetServer(address.to_owned())))
            }
            PageMessage::OpenApproval {} => match &self.connection {
                Connection::Pairing { url, .. } => Some(WindowAction::Open(url.clone())),
                _ => None,
            },
            PageMessage::OpenIntake {} => self
                .intake_link(None)
                .filter(|_| self.signed_in())
                .map(WindowAction::Open),
            PageMessage::OpenLink { link } => {
                self.shows_link(&link).then_some(WindowAction::Open(link))
            }
            PageMessage::Scan {
                scanner,
                protocol,
                profile,
            } => {
                let source = self.sources.iter().find(|source| {
                    source.name == scanner && protocol_name(source.protocol) == protocol
                })?;
                let profile = match profile {
                    Some(id) => Some(
                        self.profiles
                            .iter()
                            .find(|p| p.id.as_str() == id && p.status != ProfileStatus::Inactive)?
                            .id
                            .clone(),
                    ),
                    None => None,
                };
                command(Command::Scan {
                    source: source.name.clone(),
                    protocol: source.protocol,
                    profile,
                })
            }
            PageMessage::StopScan {} => self.scan.as_ref().and(command(Command::StopScan)),
            PageMessage::Continue { key } => {
                paused(&key).then_some(WindowAction::Command(Command::Continue(key)))
            }
            PageMessage::Finish { key } => {
                paused(&key).then_some(WindowAction::Command(Command::Finish(key)))
            }
            PageMessage::Retry { key } => {
                readable(&key).then_some(WindowAction::Command(Command::Retry(key)))
            }
            PageMessage::Save { key } => readable(&key).then_some(WindowAction::Save(key)),
            PageMessage::Discard { key } => {
                refused(&key).then_some(WindowAction::Command(Command::Discard(key)))
            }
            PageMessage::RefreshScanners {} => command(Command::RefreshScanners),
            PageMessage::AddPrinter {} => self.printer_missing.then_some(WindowAction::AddPrinter),
            PageMessage::Update {} => self
                .update
                .as_ref()
                .filter(|update| update.status == UpdateStatus::Available)
                .and(command(Command::Update)),
            PageMessage::OpenDownload {} => self
                .update
                .as_ref()
                .map(|update| WindowAction::Open(update.download_url.clone())),
            PageMessage::OpenLogs {} => Some(WindowAction::OpenLogs),
            PageMessage::Quit {} => Some(WindowAction::Quit),
        }
    }
}

/// What the page asks for. Anything else it sends is refused when parsed.
#[derive(Clone, Debug, PartialEq, Eq, Deserialize)]
#[serde(tag = "type", rename_all = "camelCase", deny_unknown_fields)]
pub enum PageMessage {
    /// The page has loaded and wants the view.
    Ready {},
    /// Hide the window.
    Close {},
    SignIn {},
    CancelSignIn {},
    SignOut {},
    #[serde(rename_all = "camelCase")]
    SetServer {
        address: String,
    },
    OpenApproval {},
    OpenIntake {},
    #[serde(rename_all = "camelCase")]
    OpenLink {
        link: String,
    },
    #[serde(rename_all = "camelCase")]
    Scan {
        scanner: String,
        protocol: String,
        profile: Option<String>,
    },
    StopScan {},
    Continue {
        key: String,
    },
    Finish {
        key: String,
    },
    Retry {
        key: String,
    },
    Save {
        key: String,
    },
    Discard {
        key: String,
    },
    RefreshScanners {},
    AddPrinter {},
    Update {},
    OpenDownload {},
    OpenLogs {},
    Quit {},
}

impl PageMessage {
    /// Parses what the page posted.
    pub fn parse(body: &str) -> Option<Self> {
        serde_json::from_str(body).ok()
    }
}

/// What the window does about a message from its page.
#[derive(Clone, Debug, PartialEq, Eq)]
pub enum WindowAction {
    /// Send the view again.
    Redraw,
    Close,
    Command(Command),
    /// Open a Trenova page in the browser.
    Open(String),
    /// Save a refused batch's pages into the person's Downloads folder.
    Save(String),
    AddPrinter,
    OpenLogs,
    Quit,
}

/// The script that hands a view to the page. JSON is a JavaScript
/// expression, so the view is passed as a literal, never as markup.
pub fn render_script(view: &View) -> String {
    let json = serde_json::to_string(view).unwrap_or_else(|_| "null".to_owned());
    format!("window.trenova && window.trenova.render({json});")
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::agent::plan::default_profile;
    use crate::state::{
        ActiveScan, Message, Notice, PausedBatch, RecentBatch, UpdateState, WaitingBatch,
    };
    use capture_client::spool::RefusedBatch;
    use capture_protocol::api::{Id, SourceInfo};

    fn source(name: &str, protocol: SourceProtocol) -> SourceInfo {
        SourceInfo {
            name: name.into(),
            protocol,
            bitness: 64,
            is_default: false,
            duplex: true,
            feeder: true,
            patch_codes: false,
            barcodes: false,
            blank_discard: false,
            resolutions: Vec::new(),
            pixel_types: Vec::new(),
        }
    }

    fn refused(key: &str, readable: bool) -> RefusedBatch {
        RefusedBatch {
            key: key.into(),
            label: if readable {
                "fi-8170".into()
            } else {
                String::new()
            },
            source: BatchSource::Scan,
            pages: 3,
            reason: "The scanner name is not valid".into(),
            created_at: 1,
            refused_at: 2,
            readable,
        }
    }

    fn waiting(key: &str, complete: bool) -> WaitingBatch {
        WaitingBatch {
            key: key.into(),
            label: "fi-8170".into(),
            source: BatchSource::Scan,
            pages: 4,
            created_at: 10,
            complete,
        }
    }

    fn online() -> Snapshot {
        let mut colour = default_profile();
        colour.id = Id::from("cprf_colour");
        colour.name = "Colour".into();
        colour.is_default = false;
        colour.pixel_type = PixelType::Color;
        colour.duplex = false;
        let mut inactive = default_profile();
        inactive.id = Id::from("cprf_old");
        inactive.status = ProfileStatus::Inactive;
        let mut default = default_profile();
        default.id = Id::from("cprf_default");
        Snapshot {
            server: Some("https://tms.acme.test".into()),
            connection: Connection::Online,
            person: Some("Jordan Doe".into()),
            organization: Some("Acme Freight".into()),
            web_base: Some("https://app.acme.test".into()),
            sources: vec![
                source("fi-8170", SourceProtocol::Twain),
                source("DS-530", SourceProtocol::Wia),
            ],
            profiles: vec![colour, inactive, default],
            ..Snapshot::default()
        }
    }

    #[test]
    fn a_ready_computer_says_who_is_signed_in_and_offers_its_scanners_and_active_profiles() {
        let view = online().view("1.2.0");
        assert_eq!(view.version, "1.2.0");
        assert_eq!(view.stage, Stage::Online);
        assert_eq!(view.status.tone, Tone::Success);
        assert_eq!(view.status.title, "Ready");
        assert_eq!(
            view.status.detail,
            "Signed in as Jordan Doe at Acme Freight."
        );
        assert!(view.can_scan);
        assert!(view.can_open_intake);
        assert_eq!(view.scanners[1].title, "DS-530 (WIA)");
        assert_eq!(view.scanners[1].protocol, "wia");
        assert_eq!(
            view.profiles
                .iter()
                .map(|p| p.id.as_str())
                .collect::<Vec<_>>(),
            ["cprf_default", "cprf_colour"],
            "the default first, and no inactive profile"
        );
        assert_eq!(view.profiles[1].summary, "300 dpi, Colour, one side");
    }

    #[test]
    fn a_scan_in_progress_is_its_own_card_and_not_a_waiting_batch() {
        let mut snapshot = online();
        snapshot.scan = Some(ActiveScan {
            key: "cap-now".into(),
            label: "fi-8170".into(),
            pages: 7,
            requested: true,
            stopping: false,
        });
        snapshot.waiting = vec![
            waiting("cap-now", true),
            waiting("cap-paused", false),
            waiting("cap-done", true),
        ];
        snapshot.pages_waiting = 12;

        let view = snapshot.view("1.0.0");

        assert_eq!(view.status.title, "Scanning");
        assert_eq!(view.status.detail, "7 pages from fi-8170.");
        assert!(!view.can_scan, "one scan at a time");
        assert_eq!(
            view.waiting
                .iter()
                .map(|w| w.key.as_str())
                .collect::<Vec<_>>(),
            ["cap-done"]
        );
        assert_eq!(view.waiting[0].state, "Sending");
    }

    #[test]
    fn a_refusal_leads_and_offline_or_signed_out_batches_say_what_they_wait_for() {
        let mut snapshot = online();
        snapshot.refused = vec![refused("cap-1", true)];
        let view = snapshot.view("1.0.0");
        assert_eq!(view.status.tone, Tone::Danger);
        assert_eq!(view.status.title, "Something was not sent");
        assert_eq!(view.refused[0].kind, Kind::Scan);

        let mut offline = online();
        offline.connection = Connection::Offline {
            reason: "dns".into(),
        };
        offline.waiting = vec![waiting("cap-2", true)];
        offline.pages_waiting = 1;
        let view = offline.view("1.0.0");
        assert_eq!(view.status.title, "Offline, reconnecting");
        assert_eq!(
            view.status.detail,
            "Trenova Capture keeps trying. 1 page is kept on this computer until then."
        );
        assert_eq!(view.waiting[0].state, "Waiting for the connection");

        let mut signed_out = offline.clone();
        signed_out.connection = Connection::SignedOut;
        assert_eq!(
            signed_out.view("1.0.0").waiting[0].state,
            "Waiting for you to sign in"
        );
        assert!(!signed_out.view("1.0.0").can_open_intake);
    }

    #[test]
    fn pairing_shows_the_code_and_a_stopped_scanner_says_what_to_do() {
        let mut snapshot = Snapshot {
            connection: Connection::Pairing {
                code: "BCDF-GHJK".into(),
                url: "https://app.acme.test/capture/pair?code=BCDF-GHJK".into(),
            },
            ..Snapshot::default()
        };
        let view = snapshot.view("1.0.0");
        assert_eq!(view.pairing_code.as_deref(), Some("BCDF-GHJK"));
        assert_eq!(view.status.title, "Approve this computer");

        snapshot.connection = Connection::Online;
        snapshot.paused = vec![PausedBatch {
            key: "cap-1".into(),
            label: "fi-8170".into(),
            pages: 12,
            condition: ScanCondition::DoubleFeed,
        }];
        let view = snapshot.view("1.0.0");
        assert_eq!(view.status.title, "A scan is waiting for you");
        assert_eq!(
            view.paused[0].problem,
            "More than one sheet went through at once. Put them back, then continue."
        );
    }

    #[test]
    fn messages_and_recent_batches_carry_their_times_and_links() {
        let mut snapshot = online();
        let at = UNIX_EPOCH + std::time::Duration::from_millis(1_780_000_000_123);
        snapshot.recent.push_front(RecentBatch {
            label: "fi-8170".into(),
            pages: 2,
            link: "https://app.acme.test/intake?batch=cbat_1".into(),
            at,
            requested: false,
        });
        snapshot.messages.push_front(Message {
            notice: Notice {
                title: "A scan could not be sent".into(),
                body: "fi-8170: invalid".into(),
                severity: Severity::Error,
                link: None,
            },
            at,
        });
        let view = snapshot.view("1.0.0");
        assert_eq!(view.recent[0].at, 1_780_000_000_123);
        assert_eq!(
            view.recent[0].link.as_deref(),
            Some("https://app.acme.test/intake?batch=cbat_1")
        );
        assert_eq!(view.messages[0].tone, Tone::Danger);
        assert_eq!(view.messages[0].link, None);
    }

    fn parse(json: &str) -> PageMessage {
        PageMessage::parse(json).unwrap_or_else(|| panic!("{json} should parse"))
    }

    #[test]
    fn page_messages_are_parsed_strictly() {
        assert_eq!(parse(r#"{"type":"ready"}"#), PageMessage::Ready {});
        assert_eq!(
            parse(r#"{"type":"retry","key":"cap-1"}"#),
            PageMessage::Retry {
                key: "cap-1".into()
            }
        );
        for bad in [
            r#"{"type":"retry"}"#,
            r#"{"type":"retry","key":"cap-1","path":"C:\\"}"#,
            r#"{"type":"open","url":"https://evil.test"}"#,
            r#"{"type":"ready","extra":1}"#,
            "not json",
            r#"["retry"]"#,
        ] {
            assert_eq!(PageMessage::parse(bad), None, "{bad} should be refused");
        }
    }

    #[test]
    fn a_page_can_only_act_on_what_the_window_shows() {
        let mut snapshot = online();
        snapshot.refused = vec![refused("cap-1", true), refused("cap-bad", false)];
        snapshot.paused = vec![PausedBatch {
            key: "cap-p".into(),
            label: "fi-8170".into(),
            pages: 1,
            condition: ScanCondition::PaperJam,
        }];
        snapshot.recent.push_front(RecentBatch {
            label: "fi-8170".into(),
            pages: 2,
            link: "https://app.acme.test/intake?batch=cbat_1".into(),
            at: UNIX_EPOCH,
            requested: false,
        });

        let resolve = |json: &str| snapshot.resolve(parse(json));
        assert_eq!(
            resolve(r#"{"type":"retry","key":"cap-1"}"#),
            Some(WindowAction::Command(Command::Retry("cap-1".into())))
        );
        assert_eq!(resolve(r#"{"type":"retry","key":"cap-nope"}"#), None);
        assert_eq!(
            resolve(r#"{"type":"retry","key":"cap-bad"}"#),
            None,
            "an unreadable batch cannot be sent again"
        );
        assert_eq!(resolve(r#"{"type":"save","key":"cap-bad"}"#), None);
        let unreadable = &snapshot.view("1.0.0").refused[1];
        assert_eq!(unreadable.label, "Unreadable batch");
        assert_eq!(
            unreadable.reason,
            "Its record on this computer is damaged, so it can only be discarded. (The scanner name is not valid)"
        );
        assert_eq!(
            resolve(r#"{"type":"discard","key":"cap-bad"}"#),
            Some(WindowAction::Command(Command::Discard("cap-bad".into()))),
            "but it can be discarded"
        );
        assert_eq!(
            resolve(r#"{"type":"save","key":"cap-1"}"#),
            Some(WindowAction::Save("cap-1".into()))
        );
        assert_eq!(resolve(r#"{"type":"continue","key":"cap-1"}"#), None);
        assert_eq!(
            resolve(r#"{"type":"finish","key":"cap-p"}"#),
            Some(WindowAction::Command(Command::Finish("cap-p".into())))
        );
        assert_eq!(
            resolve(r#"{"type":"openLink","link":"https://app.acme.test/intake?batch=cbat_1"}"#),
            Some(WindowAction::Open(
                "https://app.acme.test/intake?batch=cbat_1".into()
            ))
        );
        assert_eq!(
            resolve(r#"{"type":"openLink","link":"https://evil.test/"}"#),
            None,
            "only a link the window shows is opened"
        );
        assert_eq!(
            resolve(r#"{"type":"openIntake"}"#),
            Some(WindowAction::Open("https://app.acme.test/intake".into()))
        );
        assert_eq!(
            resolve(r#"{"type":"stopScan"}"#),
            None,
            "nothing is scanning"
        );
        assert_eq!(resolve(r#"{"type":"addPrinter"}"#), None);
        assert_eq!(resolve(r#"{"type":"openApproval"}"#), None);
    }

    #[test]
    fn a_scan_names_a_scanner_and_profile_the_computer_has() {
        let snapshot = online();
        assert_eq!(
            snapshot.resolve(parse(
                r#"{"type":"scan","scanner":"DS-530","protocol":"wia","profile":"cprf_colour"}"#
            )),
            Some(WindowAction::Command(Command::Scan {
                source: "DS-530".into(),
                protocol: SourceProtocol::Wia,
                profile: Some(Id::from("cprf_colour")),
            }))
        );
        assert_eq!(
            snapshot.resolve(parse(
                r#"{"type":"scan","scanner":"DS-530","protocol":"twain","profile":null}"#
            )),
            None,
            "the protocol has to match too"
        );
        assert_eq!(
            snapshot.resolve(parse(
                r#"{"type":"scan","scanner":"fi-8170","protocol":"twain","profile":"cprf_old"}"#
            )),
            None,
            "not an inactive profile"
        );
    }

    #[test]
    fn the_server_address_is_trimmed_and_cannot_be_changed_when_set_by_policy() {
        let mut snapshot = Snapshot::default();
        assert_eq!(
            snapshot.resolve(parse(
                r#"{"type":"setServer","address":"  https://tms.acme.test  "}"#
            )),
            Some(WindowAction::Command(Command::SetServer(
                "https://tms.acme.test".into()
            )))
        );
        assert_eq!(
            snapshot.resolve(parse(r#"{"type":"setServer","address":"   "}"#)),
            None
        );
        let long = format!(
            r#"{{"type":"setServer","address":"https://{}"}}"#,
            "a".repeat(3000)
        );
        assert_eq!(snapshot.resolve(parse(&long)), None);
        snapshot.server_locked = true;
        assert_eq!(
            snapshot.resolve(parse(r#"{"type":"setServer","address":"https://x.test"}"#)),
            None
        );
    }

    #[test]
    fn an_update_is_installed_only_when_it_is_offered() {
        let mut snapshot = online();
        snapshot.update = Some(UpdateState {
            version: "2.0.0".into(),
            download_url: "https://releases.acme.test/2.0.0.msi".into(),
            status: UpdateStatus::AskAdministrator,
        });
        assert_eq!(snapshot.resolve(PageMessage::Update {}), None);
        assert_eq!(
            snapshot.resolve(PageMessage::OpenDownload {}),
            Some(WindowAction::Open(
                "https://releases.acme.test/2.0.0.msi".into()
            ))
        );
        assert_eq!(
            snapshot.view("1.0.0").update.expect("update").status,
            "askAdministrator"
        );
    }

    #[test]
    fn the_view_reaches_the_page_as_a_literal_not_markup() {
        let mut snapshot = online();
        snapshot.person = Some("</script><img src=x onerror=alert(1)>".into());
        let script = render_script(&snapshot.view("1.0.0"));
        assert!(script.starts_with("window.trenova && window.trenova.render({"));
        assert!(script.ends_with("});"));
        let json = &script["window.trenova && window.trenova.render(".len()..script.len() - 2];
        let parsed: serde_json::Value = serde_json::from_str(json).expect("valid JSON");
        assert_eq!(
            parsed["account"]["person"],
            "</script><img src=x onerror=alert(1)>"
        );
    }
}
