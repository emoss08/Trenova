//! The agent: one task that owns everything the tray shows.
//!
//! It keeps the session with the server (the device stream, the uploader),
//! turns requests into scans, runs one scan at a time through the helpers,
//! and spools every page before anything else happens to it. It also takes
//! printed jobs from this person's print inbox (`prints`) into the same spool,
//! and keeps itself current (`updates`).
//! Network calls
//! run as their own tasks and report back here, so a slow server never holds
//! up a command from the tray, and there is exactly one place state changes.

pub mod plan;
pub mod prints;
pub mod updates;

use std::collections::{HashMap, HashSet, VecDeque};
use std::fmt::Write as _;
use std::io;
use std::path::{Path, PathBuf};
use std::sync::Arc;
use std::sync::atomic::{AtomicU64, Ordering};
use std::time::{Duration, SystemTime};

use capture_client::pairing::{PairingOutcome, pair};
use capture_client::spool::{Export, PagePictures, RefusedBatch};
use capture_client::stream::{self, DeviceEvent};
use capture_client::uploader::{UploadEvent, Uploader};
use capture_client::{
    AgentInfo, Api, ApiError, PageMarkers, Protector, SecretStore, Server, Spool, SpoolError,
    SpooledBatch,
};
use capture_protocol::api::{
    Architecture, BatchSource, CaptureProfile, CaptureRequest, DeviceIdentity, DeviceUpdatePolicy,
    Id, RequestFailureCode, RequestMode, RequestStatus, RequestStatusReport, SourceInfo,
    SourceProtocol, StartPairingRequest,
};
use capture_protocol::handoff::Inbox;
use capture_protocol::release::Release;
use capture_update::ed25519_dalek::VerifyingKey;
use capture_update::{Decision, decide};
use tokio::sync::{Notify, mpsc};
use tokio_util::sync::CancellationToken;

use self::prints::Imported;
pub use self::updates::UpdateStarter;
use crate::scanners::{ScanOutcome, ScanRun, ScanUpdate, ScannerHost};
use crate::state::{
    ActiveScan, Attention, Command, Connection, Notice, PausedBatch, Pictures, PicturesRequest,
    PrinterAttempt, RECENT, RecentBatch, Severity, Shared, UpdateState, UpdateStatus, WaitingBatch,
};

/// The most pictures one ask from the window reads.
const MAX_PICTURES_PER_ASK: usize = 24;

/// Where the server address is kept.
pub trait ServerSetting: Send + Sync {
    fn load(&self) -> Option<String>;
    fn save(&self, url: &str) -> io::Result<()>;
    /// Whether an administrator set it by policy, so a person cannot.
    fn locked(&self) -> bool;
}

pub type SecretsFor = dyn Fn(&Server) -> Arc<dyn SecretStore> + Send + Sync;
pub type Browser = dyn Fn(&str) + Send + Sync;
/// Shows a folder on this computer, as File Explorer does.
pub type Reveal = dyn Fn(&Path) + Send + Sync;

/// This computer, as pairing describes it.
#[derive(Clone, Debug)]
pub struct Machine {
    pub name: String,
    pub windows_user: String,
}

/// Everything the agent needs from the platform.
pub struct Environment {
    pub spool_dir: PathBuf,
    /// This Windows user's print inbox, when the print service is installed.
    pub print_inbox: Option<PathBuf>,
    pub agent: AgentInfo,
    pub machine: Machine,
    pub scanners: Arc<dyn ScannerHost>,
    pub secrets: Arc<SecretsFor>,
    pub protector: Arc<dyn Protector>,
    pub server_setting: Arc<dyn ServerSetting>,
    pub browser: Arc<Browser>,
    pub reveal: Arc<Reveal>,
    /// How long to wait before trying again after being blocked.
    pub recheck_after: Duration,
    /// Starts the updater service.
    pub updater: Arc<dyn UpdateStarter>,
    /// The key releases are signed with; none means this build never updates.
    pub release_key: Option<VerifyingKey>,
    /// This Windows, as a release's minimum names it.
    pub windows_build: u32,
    /// Whether this computer's own policy lets Trenova Capture update itself.
    pub machine_auto_update: bool,
    /// Whether the print service is installed without its printer.
    pub printer: Arc<dyn PrinterCheck>,
    /// How long the connection stays lost before the person is told.
    pub offline_notice_after: Duration,
}

/// Whether the Trenova printer needs adding on this computer.
pub trait PrinterCheck: Send + Sync {
    /// The print service is installed and its printer is not.
    fn missing(&self) -> bool;
}

impl std::fmt::Debug for Environment {
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        f.debug_struct("Environment")
            .field("spool_dir", &self.spool_dir)
            .field("print_inbox", &self.print_inbox)
            .field("agent", &self.agent)
            .finish_non_exhaustive()
    }
}

/// What spawned work reports back.
enum Internal {
    Identity(Result<Box<DeviceIdentity>, ApiError>),
    Profiles(Result<Vec<CaptureProfile>, ApiError>),
    Sources(Vec<SourceInfo>),
    Requests(Result<Vec<CaptureRequest>, ApiError>),
    Paired(Result<PairingOutcome, ApiError>),
    Reported(Result<(), ApiError>),
    Release(Result<Option<Release>, ApiError>),
    Recheck,
    /// The connection lost in this outage has not come back yet.
    StillOffline(u64),
}

/// A scan waiting its turn, or paused to be continued.
#[derive(Clone, Debug)]
struct Order {
    request: Option<CaptureRequest>,
    source: SourceInfo,
    profile: CaptureProfile,
    /// The spooled batch to add to, when continuing one.
    resume: Option<String>,
    /// Pages already in that batch.
    pages: u32,
}

struct Active {
    key: String,
    label: String,
    pages: u32,
    run: ScanRun,
    order: Order,
}

struct Agent {
    env: Environment,
    shared: Arc<Shared>,
    spool: Arc<Spool>,
    api: Option<Arc<Api>>,
    session: Option<CancellationToken>,
    stream_rx: Option<mpsc::Receiver<DeviceEvent>>,
    upload_rx: Option<mpsc::Receiver<UploadEvent>>,
    prints_rx: Option<mpsc::Receiver<Imported>>,
    releases_rx: Option<mpsc::Receiver<Result<Option<Release>, ApiError>>>,
    /// The organization's say over updates, from the device identity.
    update_policy: DeviceUpdatePolicy,
    /// Whether the server has said what the organization allows. Until it
    /// has, a release is held rather than acted on: the check for a release
    /// and the identity are fetched at the same time, and installing on a
    /// default the organization may refuse is the wrong guess.
    policy_known: bool,
    /// A release that arrived before the policy did.
    held_release: Option<Release>,
    /// The release the updater was asked to install, so it is asked once.
    update_started: Option<String>,
    wake: Option<Arc<Notify>>,
    pairing: Option<CancellationToken>,
    internal_tx: mpsc::Sender<Internal>,
    internal_rx: mpsc::Receiver<Internal>,
    queue: VecDeque<Order>,
    active: Option<Active>,
    paused: HashMap<String, Order>,
    handled: HashSet<Id>,
    /// Requests waiting for the scanner list before they can be planned.
    deferred: Vec<CaptureRequest>,
    sources_known: bool,
    sources: Vec<SourceInfo>,
    /// What opened sources said they can do, which enumeration cannot learn.
    learned: HashMap<(String, SourceProtocol), SourceInfo>,
    profiles: Vec<CaptureProfile>,
    fetching: bool,
    fetch_again: bool,
    blocked: bool,
    /// Counts connection losses, so a late check for one that ended is
    /// ignored.
    outage: u64,
    /// The person was told about the loss that is going on now.
    outage_announced: bool,
    /// Counts spool reads, so a slow one never overwrites a newer one.
    spool_reads: Arc<AtomicU64>,
}

async fn next<T>(rx: Option<&mut mpsc::Receiver<T>>) -> Option<T> {
    match rx {
        Some(rx) => rx.recv().await,
        None => std::future::pending().await,
    }
}

/// Runs the agent until the tray quits or `cancel` fires.
pub async fn run(
    env: Environment,
    shared: Arc<Shared>,
    mut commands: mpsc::UnboundedReceiver<Command>,
    cancel: CancellationToken,
) -> Result<(), SpoolError> {
    let spool = Arc::new(Spool::open(&env.spool_dir, Arc::clone(&env.protector))?);
    finish_interrupted(&spool);
    let (internal_tx, internal_rx) = mpsc::channel(64);
    let mut agent = Agent {
        env,
        shared,
        spool,
        api: None,
        session: None,
        stream_rx: None,
        upload_rx: None,
        prints_rx: None,
        releases_rx: None,
        update_policy: DeviceUpdatePolicy::default(),
        policy_known: false,
        held_release: None,
        update_started: None,
        wake: None,
        pairing: None,
        internal_tx,
        internal_rx,
        queue: VecDeque::new(),
        active: None,
        paused: HashMap::new(),
        handled: HashSet::new(),
        deferred: Vec::new(),
        sources_known: false,
        sources: Vec::new(),
        learned: HashMap::new(),
        profiles: Vec::new(),
        fetching: false,
        fetch_again: false,
        blocked: false,
        outage: 0,
        outage_announced: false,
        spool_reads: Arc::new(AtomicU64::new(0)),
    };
    if let Some(dir) = agent.env.print_inbox.clone() {
        let (tx, rx) = mpsc::channel(16);
        let shared = Arc::clone(&agent.shared);
        tokio::spawn(prints::watch(
            Inbox::new(dir),
            Arc::clone(&agent.spool),
            tx,
            Arc::new(move || shared.snapshot().review_before_sending),
            cancel.child_token(),
        ));
        agent.prints_rx = Some(rx);
    }
    let locked = agent.env.server_setting.locked();
    let printing = agent.env.print_inbox.is_some();
    agent.shared.update(|s| {
        s.server_locked = locked;
        s.printing = printing;
    });
    agent.refresh_spool();
    agent.check_printer();
    let configured = agent.env.server_setting.load();
    agent.configure(configured.as_deref()).await;

    loop {
        tokio::select! {
            () = cancel.cancelled() => break,
            command = commands.recv() => match command {
                None | Some(Command::Quit) => break,
                Some(command) => agent.command(command).await,
            },
            Some(message) = agent.internal_rx.recv() => agent.internal(message).await,
            event = next(agent.stream_rx.as_mut()) => match event {
                Some(event) => agent.stream_event(event),
                None => agent.stream_rx = None,
            },
            event = next(agent.upload_rx.as_mut()) => match event {
                Some(event) => agent.upload_event(event),
                None => agent.upload_rx = None,
            },
            imported = next(agent.prints_rx.as_mut()) => match imported {
                Some(imported) => agent.printed(imported),
                None => agent.prints_rx = None,
            },
            release = next(agent.releases_rx.as_mut()) => match release {
                Some(release) => agent.release(release),
                None => agent.releases_rx = None,
            },
            update = next(agent.active.as_mut().map(|a| &mut a.run.updates)) => match update {
                Some(update) => agent.scan_update(update),
                None => agent.scan_ended(ScanOutcome::Failed {
                    code: RequestFailureCode::Internal,
                    message: "The scan stopped without saying why.".into(),
                }),
            },
        }
    }

    agent.shutdown();
    Ok(())
}

/// A scan the agent was taking when it last stopped is sent as it is: its
/// pages are good, and nobody is left to continue it.
fn finish_interrupted(spool: &Spool) {
    match spool.pending() {
        Ok(batches) => {
            for batch in batches.iter().filter(|b| !b.complete) {
                if let Err(err) = spool.complete(batch.key()) {
                    tracing::error!(key = batch.key(), error = %err, "could not finish an interrupted batch");
                }
            }
        }
        Err(err) => tracing::error!(error = %err, "could not read the spool"),
    }
}

/// A folder name for a saved batch: its label with what Windows forbids in
/// a file name taken out.
fn saved_folder_name(batch: &RefusedBatch) -> String {
    let cleaned: String = batch
        .label
        .chars()
        .map(|c| {
            if c.is_control() || matches!(c, '<' | '>' | ':' | '"' | '/' | '\\' | '|' | '?' | '*') {
                ' '
            } else {
                c
            }
        })
        .take(60)
        .collect();
    let cleaned = cleaned.split_whitespace().collect::<Vec<_>>().join(" ");
    let cleaned = cleaned.trim_end_matches(['.', ' ']);
    let what = match batch.source {
        BatchSource::Print => "print",
        _ => "scan",
    };
    if cleaned.is_empty() {
        format!("Trenova Capture {what} not sent")
    } else {
        format!("{cleaned} {what} not sent")
    }
}

/// `parent\name`, or `parent\name (2)` and so on when it is taken.
fn unused_folder(parent: &Path, name: &str) -> Option<PathBuf> {
    let first = parent.join(name);
    if !first.exists() {
        return Some(first);
    }
    (2..1000)
        .map(|n| parent.join(format!("{name} ({n})")))
        .find(|path| !path.exists())
}

fn plural(count: u32, one: &str, many: &str) -> String {
    if count == 1 {
        format!("{count} {one}")
    } else {
        format!("{count} {many}")
    }
}

impl Agent {
    fn notify(
        &self,
        severity: Severity,
        title: impl Into<String>,
        body: impl Into<String>,
        link: Option<String>,
    ) {
        self.shared.notify(Notice {
            title: title.into(),
            body: body.into(),
            severity,
            link,
            routine: false,
        });
    }

    /// Tells the person something that needs nothing from them, unless they
    /// chose not to hear about such things.
    fn notify_routine(
        &self,
        title: impl Into<String>,
        body: impl Into<String>,
        link: Option<String>,
    ) {
        self.shared.notify(Notice {
            title: title.into(),
            body: body.into(),
            severity: Severity::Info,
            link,
            routine: true,
        });
    }

    fn signed_in(&self) -> bool {
        self.session.is_some()
    }

    /// Points the agent at a server, signing in if a credential is saved.
    async fn configure(&mut self, url: Option<&str>) {
        self.stop_session();
        self.api = None;
        let Some(url) = url else {
            self.shared.update(|s| {
                s.server = None;
                s.connection = Connection::NeedsServer;
            });
            self.shared.attention(Attention::SetUp);
            return;
        };
        let server = match Server::parse(url) {
            Ok(server) => server,
            Err(err) => {
                self.shared
                    .update(|s| s.connection = Connection::NeedsServer);
                self.shared.attention(Attention::SetUp);
                self.notify(
                    Severity::Error,
                    "The server address is not valid",
                    err.to_string(),
                    None,
                );
                return;
            }
        };
        let store = (self.env.secrets)(&server);
        let address = server.as_str().to_owned();
        let api = match Api::new(server, self.env.agent.clone(), store) {
            Ok(api) => Arc::new(api),
            Err(err) => {
                self.shared
                    .update(|s| s.connection = Connection::NeedsServer);
                self.notify(
                    Severity::Error,
                    "Trenova Capture could not start",
                    err.to_string(),
                    None,
                );
                return;
            }
        };
        let signed_in = api.credential().await.is_some();
        self.api = Some(api);
        self.shared.update(|s| s.server = Some(address));
        if signed_in {
            self.start_session().await;
        } else {
            self.shared.update(|s| s.connection = Connection::SignedOut);
        }
    }

    async fn start_session(&mut self) {
        self.stop_session();
        let Some(api) = self.api.clone() else {
            return;
        };
        let web_base = api.credential().await.map(|c| c.web_base);
        self.handled.clear();
        self.shared.update(|s| {
            s.connection = Connection::Connecting;
            s.web_base = web_base;
            s.update_required = None;
        });

        let token = CancellationToken::new();
        let (stream_tx, stream_rx) = mpsc::channel(32);
        let (upload_tx, upload_rx) = mpsc::channel(64);
        let uploader = Uploader::new(Arc::clone(&api), Arc::clone(&self.spool), upload_tx);
        self.wake = Some(uploader.waker());
        tokio::spawn(uploader.run(token.child_token()));
        tokio::spawn(stream::run(
            Arc::clone(&api),
            stream_tx,
            token.child_token(),
        ));
        self.stream_rx = Some(stream_rx);
        self.upload_rx = Some(upload_rx);
        let (release_tx, release_rx) = mpsc::channel(4);
        tokio::spawn(updates::watch(
            Arc::clone(&api),
            self.env.release_key,
            release_tx,
            token.child_token(),
        ));
        self.releases_rx = Some(release_rx);
        self.session = Some(token);

        let tx = self.internal_tx.clone();
        let hello = Arc::clone(&api);
        tokio::spawn(async move {
            let _ = tx
                .send(Internal::Identity(hello.identity().await.map(Box::new)))
                .await;
            let _ = tx.send(Internal::Profiles(hello.profiles().await)).await;
        });
        self.enumerate();
    }

    /// Tells the person the connection has been lost for a while, once per
    /// outage.
    fn still_offline(&mut self, outage: u64) {
        let snapshot = self.shared.snapshot();
        if outage != self.outage
            || self.outage_announced
            || !matches!(snapshot.connection, Connection::Offline { .. })
        {
            return;
        }
        self.outage_announced = true;
        let body = if snapshot.pages_waiting > 0 {
            format!(
                "It keeps trying to reconnect. The {} not yet sent are kept on this computer and sent when it does.",
                plural(snapshot.pages_waiting, "page", "pages")
            )
        } else {
            "It keeps trying to reconnect. Anything scanned or printed meanwhile is kept on this computer and sent when it does.".to_owned()
        };
        self.notify(Severity::Warning, "Trenova Capture is offline", body, None);
    }

    fn stop_session(&mut self) {
        self.outage += 1;
        self.outage_announced = false;
        if let Some(token) = self.session.take() {
            token.cancel();
        }
        self.stream_rx = None;
        self.upload_rx = None;
        self.releases_rx = None;
        self.policy_known = false;
        self.held_release = None;
        self.wake = None;
        self.fetching = false;
        self.fetch_again = false;
    }

    fn shutdown(&mut self) {
        if let Some(active) = &self.active {
            active.run.cancel.cancel();
        }
        if let Some(pairing) = self.pairing.take() {
            pairing.cancel();
        }
        self.stop_session();
    }

    fn wake_uploader(&self) {
        if let Some(wake) = &self.wake {
            wake.notify_one();
        }
    }

    fn enumerate(&self) {
        let scanners = Arc::clone(&self.env.scanners);
        let tx = self.internal_tx.clone();
        tokio::spawn(async move {
            let sources = scanners.enumerate().await;
            let _ = tx.send(Internal::Sources(sources)).await;
        });
    }

    fn fetch_requests(&mut self) {
        let Some(api) = self.api.clone().filter(|_| self.signed_in()) else {
            return;
        };
        if self.fetching {
            self.fetch_again = true;
            return;
        }
        self.fetching = true;
        let tx = self.internal_tx.clone();
        tokio::spawn(async move {
            let _ = tx.send(Internal::Requests(api.open_requests().await)).await;
        });
    }

    fn report(&self, request: &Id, report: RequestStatusReport) {
        let Some(api) = self.api.clone() else {
            return;
        };
        let (tx, request) = (self.internal_tx.clone(), request.clone());
        tokio::spawn(async move {
            let result = api.report_request(&request, &report).await.map(|_| ());
            let _ = tx.send(Internal::Reported(result)).await;
        });
    }

    fn report_sources(&self) {
        let Some(api) = self.api.clone().filter(|_| self.signed_in()) else {
            return;
        };
        let (tx, sources) = (self.internal_tx.clone(), self.sources.clone());
        tokio::spawn(async move {
            let result = api.report_sources(&sources).await.map(|_| ());
            let _ = tx.send(Internal::Reported(result)).await;
        });
    }

    async fn command(&mut self, command: Command) {
        match command {
            Command::SetServer(url) => self.set_server(&url).await,
            Command::SignIn => self.sign_in(),
            Command::CancelSignIn => {
                if let Some(pairing) = self.pairing.take() {
                    pairing.cancel();
                }
            }
            Command::SignOut => self.sign_out(),
            Command::Scan {
                source,
                protocol,
                profile,
            } => self.scan_to_intake(&source, protocol, profile.as_ref()),
            Command::Continue(key) => {
                if let Some(mut order) = self.paused.remove(&key) {
                    order.resume = Some(key.clone());
                    self.queue.push_front(order);
                    self.shared.update(|s| s.paused.retain(|p| p.key != key));
                    self.start_next();
                }
            }
            Command::Finish(key) => {
                if self.paused.remove(&key).is_some() {
                    self.complete(&key);
                    self.shared.update(|s| s.paused.retain(|p| p.key != key));
                }
            }
            Command::RefreshScanners => {
                self.check_printer();
                self.enumerate();
            }
            Command::PrinterSetUp(attempt) => self.printer_set_up(attempt),
            Command::Update => self.install_update(true),
            Command::StopScan => self.stop_scan(),
            Command::Retry(key) => self.retry(key).await,
            Command::Discard(key) => self.discard(key).await,
            Command::Save { key, into } => self.save(key, into).await,
            Command::RotatePage { key, page, degrees } => {
                if let Err(err) = self
                    .change_spool(move |spool| spool.rotate(&key, page, degrees))
                    .await
                {
                    self.notify(
                        Severity::Error,
                        "The page could not be turned",
                        err.to_string(),
                        None,
                    );
                }
            }
            Command::DeletePage { key, page } => self.delete_page(key, page).await,
            Command::SendHeld(key) => {
                match self.change_spool(move |spool| spool.release(&key)).await {
                    Ok(()) => self.wake_uploader(),
                    Err(err) => {
                        self.notify(
                            Severity::Error,
                            "It could not be sent",
                            err.to_string(),
                            None,
                        );
                    }
                }
            }
            Command::DiscardHeld(key) => {
                if let Err(err) = self
                    .change_spool(move |spool| spool.discard_held(&key))
                    .await
                {
                    self.notify(
                        Severity::Error,
                        "It could not be discarded",
                        err.to_string(),
                        None,
                    );
                }
            }
            Command::ScanMore {
                key,
                source,
                protocol,
                profile,
            } => {
                self.scan_more(key, &source, protocol, profile.as_ref())
                    .await;
            }
            Command::Pictures(request) => self.read_pictures(request),
            Command::Quit => {}
        }
    }

    fn stop_scan(&mut self) {
        let Some(active) = &self.active else {
            return;
        };
        active.run.cancel.cancel();
        self.shared.update(|s| {
            if let Some(scan) = &mut s.scan {
                scan.stopping = true;
            }
        });
    }

    /// Reads what is waiting and what was refused, off this task, and shows
    /// it. A read that finishes after a newer one has started is dropped, so
    /// what is shown never goes back in time.
    fn refresh_spool(&self) {
        let spool = Arc::clone(&self.spool);
        let shared = Arc::clone(&self.shared);
        let reads = Arc::clone(&self.spool_reads);
        let this = reads.fetch_add(1, Ordering::SeqCst) + 1;
        tokio::spawn(async move {
            let read = tokio::task::spawn_blocking(move || {
                Ok::<_, SpoolError>((spool.pending()?, spool.refused()?))
            })
            .await;
            let (pending, refused) = match read {
                Ok(Ok(read)) => read,
                Ok(Err(err)) => {
                    tracing::warn!(error = %err, "could not read the spool");
                    return;
                }
                Err(err) => {
                    tracing::warn!(error = %err, "reading the spool stopped");
                    return;
                }
            };
            if reads.load(Ordering::SeqCst) != this {
                return;
            }
            let pages_waiting = pending.iter().map(SpooledBatch::pages_waiting).sum();
            let waiting = pending
                .iter()
                .map(|batch| WaitingBatch {
                    key: batch.key().to_owned(),
                    label: batch.label.clone(),
                    source: batch.input.source,
                    pages: batch.pages_waiting(),
                    created_at: batch.created_at,
                    complete: batch.complete,
                    held: batch.held,
                    requested: batch.input.request_id.is_some(),
                    printed: batch.document.is_some(),
                    editable: batch.document.is_none() && batch.pages.iter().all(|p| !p.uploaded),
                    pictures: batch.picture_refs(),
                })
                .collect();
            shared.update(|s| {
                s.pages_waiting = pages_waiting;
                s.waiting = waiting;
                s.refused = refused;
            });
        });
    }

    /// Runs a spool change off this task, then shows the spool as it now is.
    async fn change_spool<T: Send + 'static>(
        &self,
        change: impl FnOnce(&Spool) -> Result<T, SpoolError> + Send + 'static,
    ) -> Result<T, SpoolError> {
        let spool = Arc::clone(&self.spool);
        let result = tokio::task::spawn_blocking(move || change(&spool))
            .await
            .unwrap_or_else(|err| Err(SpoolError::Io(io::Error::other(err))));
        self.refresh_spool();
        result
    }

    async fn retry(&mut self, key: String) {
        match self.change_spool(move |spool| spool.retry(&key)).await {
            Ok(_) => self.wake_uploader(),
            Err(err) => self.notify(
                Severity::Error,
                "It could not be sent again",
                err.to_string(),
                None,
            ),
        }
    }

    async fn discard(&mut self, key: String) {
        if let Err(err) = self.change_spool(move |spool| spool.discard(&key)).await {
            self.notify(
                Severity::Error,
                "It could not be discarded",
                err.to_string(),
                None,
            );
        }
    }

    async fn save(&mut self, key: String, into: PathBuf) {
        let saved = self
            .change_spool(move |spool| {
                let batch = spool
                    .refused()?
                    .into_iter()
                    .find(|b| b.key == key)
                    .ok_or_else(|| SpoolError::Missing(key.clone()))?;
                std::fs::create_dir_all(&into)?;
                let folder = unused_folder(&into, &saved_folder_name(&batch)).ok_or_else(|| {
                    SpoolError::Io(io::Error::other("no free folder name to save it under"))
                })?;
                let export = spool.export(&key, &folder)?;
                Ok::<(PathBuf, Export), SpoolError>((folder, export))
            })
            .await;
        match saved {
            Ok((folder, export)) => {
                let written = u32::try_from(export.written.len()).unwrap_or(u32::MAX);
                let mut body = format!("Saved to {}.", folder.display());
                if !export.unreadable.is_empty() {
                    let missing = u32::try_from(export.unreadable.len()).unwrap_or(u32::MAX);
                    let _ = write!(
                        body,
                        " {} could not be read and {} left out.",
                        plural(missing, "page", "pages"),
                        if missing == 1 { "was" } else { "were" }
                    );
                }
                (self.env.reveal)(&folder);
                self.notify(
                    if export.unreadable.is_empty() {
                        Severity::Info
                    } else {
                        Severity::Warning
                    },
                    if written == 0 {
                        "Nothing could be saved".to_owned()
                    } else {
                        format!("{} saved", plural(written, "file", "files"))
                    },
                    body,
                    None,
                );
            }
            Err(err) => self.notify(
                Severity::Error,
                "It could not be saved",
                err.to_string(),
                None,
            ),
        }
    }

    async fn set_server(&mut self, url: &str) {
        if self.env.server_setting.locked() {
            self.notify(
                Severity::Warning,
                "The server address is set by your organization",
                "Ask your administrator to change it.",
                None,
            );
            return;
        }
        let server = match Server::parse(url) {
            Ok(server) => server,
            Err(err) => {
                self.notify(
                    Severity::Error,
                    "The server address is not valid",
                    err.to_string(),
                    None,
                );
                return;
            }
        };
        let current = self.api.as_ref().map(|a| a.server().as_str().to_owned());
        if current.as_deref() == Some(server.as_str()) {
            return;
        }
        let waiting = self.spool.summary().map(|s| s.pages_waiting).unwrap_or(0);
        if waiting > 0 && current.is_some() {
            self.notify(
                Severity::Warning,
                "Pages are still waiting to upload",
                format!(
                    "Wait until the {} waiting have reached Trenova before changing the server address.",
                    plural(waiting, "page", "pages")
                ),
                None,
            );
            return;
        }
        if let Err(err) = self.env.server_setting.save(server.as_str()) {
            self.notify(
                Severity::Error,
                "The server address could not be saved",
                err.to_string(),
                None,
            );
            return;
        }
        self.configure(Some(server.as_str())).await;
    }

    fn sign_in(&mut self) {
        let Some(api) = self.api.clone() else {
            self.notify(
                Severity::Warning,
                "Set the server address first",
                "Choose Set server address in the Trenova Capture menu.",
                None,
            );
            return;
        };
        if self.pairing.is_some() || self.signed_in() {
            return;
        }
        let machine = StartPairingRequest {
            machine_name: self.env.machine.name.clone(),
            windows_user: self.env.machine.windows_user.clone(),
            agent_version: self.env.agent.version.clone(),
            architecture: Architecture::X64,
            os_version: self.env.agent.os_version.clone(),
        };
        let token = CancellationToken::new();
        self.pairing = Some(token.clone());
        let (shared, browser, tx) = (
            Arc::clone(&self.shared),
            Arc::clone(&self.env.browser),
            self.internal_tx.clone(),
        );
        tokio::spawn(async move {
            let show = |grant: &capture_protocol::api::PairingGrant| {
                shared.update(|s| {
                    s.connection = Connection::Pairing {
                        code: grant.user_code.clone(),
                        url: grant.verification_uri_complete.clone(),
                    };
                });
                shared.attention(Attention::SignIn);
                browser(&grant.verification_uri_complete);
                shared.notify(Notice {
                    title: format!("Approve this computer with code {}", grant.user_code),
                    body: "Trenova opened in your browser. Check the code matches, then approve."
                        .into(),
                    severity: Severity::Info,
                    link: Some(grant.verification_uri_complete.clone()),
                    routine: false,
                });
            };
            let result = pair(&api, &machine, show, &token).await;
            let _ = tx.send(Internal::Paired(result)).await;
        });
    }

    fn sign_out(&mut self) {
        self.stop_session();
        if let Some(pairing) = self.pairing.take() {
            pairing.cancel();
        }
        if let Some(api) = self.api.clone() {
            tokio::spawn(async move {
                if let Err(err) = api.revoke_self().await {
                    tracing::warn!(error = %err, "the server was not told about the sign-out");
                }
            });
        }
        self.shared.update(|s| {
            s.connection = Connection::SignedOut;
            s.person = None;
            s.organization = None;
            s.profiles.clear();
        });
    }

    /// The scanner a person chose, when this computer is signed in and the
    /// scanner is here; says why not otherwise.
    fn chosen_source(&self, name: &str, protocol: SourceProtocol) -> Option<SourceInfo> {
        if !self.signed_in() {
            self.notify(
                Severity::Warning,
                "Sign in to scan",
                "Choose Sign in in the Trenova Capture menu.",
                None,
            );
            return None;
        }
        let source = self
            .sources
            .iter()
            .find(|s| s.name == name && s.protocol == protocol)
            .cloned();
        if source.is_none() {
            self.notify(Severity::Error, "That scanner is not connected", name, None);
        }
        source
    }

    fn scan_to_intake(&mut self, name: &str, protocol: SourceProtocol, profile: Option<&Id>) {
        let Some(source) = self.chosen_source(name, protocol) else {
            return;
        };
        let profile = plan::choose_profile(None, profile, &self.profiles);
        self.queue.push_back(Order {
            request: None,
            source,
            profile,
            resume: None,
            pages: 0,
        });
        self.start_next();
    }

    /// Scans more pages onto the end of a batch held for review.
    async fn scan_more(
        &mut self,
        key: String,
        name: &str,
        protocol: SourceProtocol,
        profile: Option<&Id>,
    ) {
        let Some(source) = self.chosen_source(name, protocol) else {
            return;
        };
        let profile = plan::choose_profile(None, profile, &self.profiles);
        let reopen = key.clone();
        let reopened = self
            .change_spool(move |spool| {
                spool.reopen(&reopen)?;
                Ok(spool.get(&reopen)?.pages.len())
            })
            .await;
        match reopened {
            Ok(pages) => {
                self.queue.push_back(Order {
                    request: None,
                    source,
                    profile,
                    resume: Some(key),
                    pages: u32::try_from(pages).unwrap_or(u32::MAX),
                });
                self.start_next();
            }
            Err(err) => self.notify(
                Severity::Error,
                "More pages could not be scanned",
                err.to_string(),
                None,
            ),
        }
    }

    /// Takes a page out of a held batch; taking out the last one discards
    /// the batch.
    async fn delete_page(&mut self, key: String, page: u32) {
        let result = self
            .change_spool(move |spool| {
                let left = spool.delete_page(&key, page)?;
                if left == 0 {
                    spool.discard_held(&key)?;
                }
                Ok(left)
            })
            .await;
        if let Err(err) = result {
            self.notify(
                Severity::Error,
                "The page could not be taken out",
                err.to_string(),
                None,
            );
        }
    }

    /// Reads pages' pictures for the window, off this task. A page whose
    /// picture cannot be read is left out, and the window shows it without.
    fn read_pictures(&self, request: PicturesRequest) {
        let spool = Arc::clone(&self.spool);
        let shared = Arc::clone(&self.shared);
        tokio::spawn(async move {
            let key = request.key.clone();
            let size = request.size;
            let read = tokio::task::spawn_blocking(move || {
                request
                    .pages
                    .iter()
                    .take(MAX_PICTURES_PER_ASK)
                    .filter_map(|&page| {
                        spool
                            .picture(&request.key, page, request.size)
                            .inspect_err(|err| {
                                tracing::debug!(page, error = %err, "a picture could not be read");
                            })
                            .ok()
                            .map(|picture| (page, picture))
                    })
                    .collect::<Vec<_>>()
            })
            .await;
            if let Ok(pictures) = read {
                shared.pictures(Pictures {
                    key,
                    size,
                    pictures,
                });
            }
        });
    }

    async fn internal(&mut self, message: Internal) {
        match message {
            Internal::Identity(Ok(identity)) => {
                self.update_policy = identity.updates.clone();
                self.policy_known = true;
                self.shared.update(|s| {
                    s.person = Some(identity.person.name.clone());
                    s.organization = Some(identity.organization.name.clone());
                });
                if let Some(release) = self.held_release.take() {
                    self.release(Ok(Some(release)));
                } else if self.shared.snapshot().update.is_some() {
                    self.install_update(false);
                }
            }
            Internal::Profiles(Ok(profiles)) => {
                self.profiles = profiles;
                let profiles = self.profiles.clone();
                self.shared.update(|s| s.profiles = profiles);
            }
            Internal::Identity(Err(err))
            | Internal::Profiles(Err(err))
            | Internal::Reported(Err(err)) => {
                self.api_error(&err);
            }
            Internal::Reported(Ok(())) => {}
            Internal::Sources(sources) => self.sources_listed(sources),
            Internal::Requests(result) => {
                self.fetching = false;
                match result {
                    Ok(requests) => self.requests(requests),
                    Err(err) => self.api_error(&err),
                }
                if std::mem::take(&mut self.fetch_again) {
                    self.fetch_requests();
                }
            }
            Internal::Paired(result) => self.paired(result).await,
            Internal::Release(release) => self.release(release),
            Internal::Recheck => {
                if self.blocked && self.api.is_some() {
                    self.start_session().await;
                }
            }
            Internal::StillOffline(outage) => self.still_offline(outage),
        }
    }

    async fn paired(&mut self, result: Result<PairingOutcome, ApiError>) {
        self.pairing = None;
        match result {
            Ok(PairingOutcome::Paired(_)) => {
                self.notify(
                    Severity::Info,
                    "This computer is signed in to Trenova",
                    "You can now scan into Trenova.",
                    None,
                );
                self.start_session().await;
            }
            Ok(outcome) => {
                self.shared.update(|s| s.connection = Connection::SignedOut);
                match outcome {
                    PairingOutcome::Denied => {
                        self.notify(
                            Severity::Warning,
                            "Sign-in was declined",
                            "This computer was not signed in.",
                            None,
                        );
                    }
                    PairingOutcome::Expired => self.notify(
                        Severity::Warning,
                        "The sign-in code expired",
                        "Choose Sign in to get a new one.",
                        None,
                    ),
                    PairingOutcome::Canceled | PairingOutcome::Paired(_) => {}
                }
            }
            Err(err) => {
                self.shared.update(|s| s.connection = Connection::SignedOut);
                self.notify(Severity::Error, "Sign-in failed", err.to_string(), None);
            }
        }
    }

    fn sources_listed(&mut self, listed: Vec<SourceInfo>) {
        self.sources = listed
            .into_iter()
            .map(
                |source| match self.learned.get(&(source.name.clone(), source.protocol)) {
                    Some(known) => SourceInfo {
                        is_default: source.is_default,
                        bitness: source.bitness,
                        ..known.clone()
                    },
                    None => source,
                },
            )
            .collect();
        self.sources_known = true;
        let sources = self.sources.clone();
        self.shared.update(|s| s.sources = sources);
        self.report_sources();
        let deferred = std::mem::take(&mut self.deferred);
        if !deferred.is_empty() {
            self.requests(deferred);
        }
    }

    /// Plans the scans the person asked for from the web app.
    fn requests(&mut self, requests: Vec<CaptureRequest>) {
        let open: HashSet<Id> = requests.iter().map(|r| r.id.clone()).collect();
        self.queue
            .retain(|order| order.request.as_ref().is_none_or(|r| open.contains(&r.id)));
        if let Some(active) = &self.active
            && active.pages == 0
            && active
                .order
                .request
                .as_ref()
                .is_some_and(|r| !open.contains(&r.id))
        {
            active.run.cancel.cancel();
        }

        for request in requests {
            if request.mode != RequestMode::Scan
                || !matches!(
                    request.status,
                    RequestStatus::Pending | RequestStatus::Delivered
                )
                || self.handled.contains(&request.id)
            {
                continue;
            }
            if !self.sources_known {
                self.deferred.push(request);
                continue;
            }
            self.handled.insert(request.id.clone());
            let Some(source) = plan::choose_source(&request.source_name, &self.sources).cloned()
            else {
                let message = if request.source_name.is_empty() {
                    "No scanner is connected to this computer.".to_owned()
                } else {
                    format!(
                        "No scanner named {} is connected to this computer.",
                        request.source_name
                    )
                };
                self.report(
                    &request.id,
                    RequestStatusReport::failed(RequestFailureCode::SourceUnavailable, &message),
                );
                self.notify(Severity::Error, "A scan could not start", message, None);
                continue;
            };
            if request.status == RequestStatus::Pending {
                self.report(
                    &request.id,
                    RequestStatusReport::status(RequestStatus::Delivered),
                );
            }
            let profile = plan::choose_profile(
                request.profile.as_ref(),
                request.profile_id.as_ref(),
                &self.profiles,
            );
            self.queue.push_back(Order {
                request: Some(request),
                source,
                profile,
                resume: None,
                pages: 0,
            });
        }
        self.start_next();
    }

    fn start_next(&mut self) {
        if self.active.is_some() || !self.signed_in() {
            return;
        }
        let Some(order) = self.queue.pop_front() else {
            return;
        };
        let label = plan::label(&order.source);
        let key = if let Some(key) = &order.resume {
            key.clone()
        } else {
            let key = Spool::new_key();
            let input =
                plan::batch_input(&key, &order.source, &order.profile, order.request.as_ref());
            let created = if self.shared.snapshot().review_before_sending {
                self.spool.create_held(input, label.clone())
            } else {
                self.spool.create(input, label.clone())
            };
            if let Err(err) = created {
                self.notify(
                    Severity::Error,
                    "A scan could not start",
                    err.to_string(),
                    None,
                );
                if let Some(request) = &order.request {
                    self.report(
                        &request.id,
                        RequestStatusReport::failed(RequestFailureCode::Internal, err.to_string()),
                    );
                }
                self.start_next();
                return;
            }
            key
        };
        let job = plan::job(&order.source, &order.profile);
        let run = self.env.scanners.scan(&order.source, job);
        let scan = ActiveScan {
            key: key.clone(),
            label: label.clone(),
            pages: order.pages,
            requested: order.request.is_some(),
            stopping: false,
        };
        self.shared.update(|s| s.scan = Some(scan));
        self.shared.attention(Attention::ScanStarted);
        self.active = Some(Active {
            key,
            label,
            pages: order.pages,
            run,
            order,
        });
    }

    fn scan_update(&mut self, update: ScanUpdate) {
        match update {
            ScanUpdate::Described(source) => self.described(source),
            ScanUpdate::Started(settings) => {
                if let Some(active) = &self.active
                    && let Err(err) = self.spool.set_settings(&active.key, settings)
                {
                    tracing::warn!(error = %err, "could not record the scan settings");
                }
            }
            ScanUpdate::Page {
                meta,
                pdf,
                pictures,
            } => self.page(meta, &pdf, pictures.as_ref()),
            ScanUpdate::End(outcome) => self.scan_ended(outcome),
        }
    }

    fn described(&mut self, source: SourceInfo) {
        self.learned
            .insert((source.name.clone(), source.protocol), source.clone());
        if let Some(known) = self
            .sources
            .iter_mut()
            .find(|s| s.name == source.name && s.protocol == source.protocol)
        {
            *known = SourceInfo {
                is_default: known.is_default,
                bitness: known.bitness,
                ..source
            };
        }
        let sources = self.sources.clone();
        self.shared.update(|s| s.sources = sources);
        self.report_sources();
    }

    fn page(
        &mut self,
        meta: capture_protocol::helper::PageMeta,
        pdf: &[u8],
        pictures: Option<&PagePictures>,
    ) {
        let Some(active) = &mut self.active else {
            return;
        };
        let markers = PageMarkers {
            dpi: meta.dpi,
            patch_code: meta.patch_code,
            barcodes: meta.barcodes,
            rotation: 0,
        };
        match self.spool.append_page(&active.key, pdf, &markers) {
            Ok(sequence) => {
                if let Some(pictures) = pictures
                    && let Err(err) = self.spool.store_pictures(&active.key, sequence, pictures)
                {
                    tracing::warn!(sequence, error = %err, "could not keep a page's pictures");
                }
                active.pages += 1;
                let pages = active.pages;
                self.shared.update(|s| {
                    if let Some(scan) = &mut s.scan {
                        scan.pages = pages;
                    }
                });
                self.wake_uploader();
            }
            Err(err) => {
                active.run.cancel.cancel();
                let body = match err {
                    SpoolError::Full => {
                        "A scan holds at most 1,000 pages. Scan the rest as a new batch.".to_owned()
                    }
                    other => format!("A page could not be saved on this computer: {other}"),
                };
                self.notify(Severity::Error, "Scanning stopped", body, None);
            }
        }
    }

    fn complete(&self, key: &str) {
        match self.spool.complete(key) {
            Ok(true) => match self.spool.get(key) {
                Ok(batch) if batch.held => {
                    self.refresh_spool();
                    self.review_ready(
                        "Look over the scan before it is sent",
                        format!(
                            "{} from {} wait for you in Trenova Capture.",
                            plural(
                                u32::try_from(batch.pages.len()).unwrap_or(u32::MAX),
                                "page",
                                "pages"
                            ),
                            batch.label
                        ),
                    );
                }
                _ => self.wake_uploader(),
            },
            Ok(false) => self.refresh_spool(),
            Err(err) => tracing::error!(key, error = %err, "could not finish a batch"),
        }
    }

    /// Something is held for the person to look over.
    fn review_ready(&self, title: &str, body: String) {
        self.shared.attention(Attention::Review);
        self.notify(Severity::Info, title, body, None);
    }

    fn scan_ended(&mut self, outcome: ScanOutcome) {
        let Some(active) = self.active.take() else {
            return;
        };
        self.shared.update(|s| s.scan = None);
        let stopped_by_person = active.run.cancel.is_cancelled();
        let request = active.order.request.as_ref().map(|r| r.id.clone());
        let fail = |agent: &Self, code: RequestFailureCode, message: &str| {
            if let Some(id) = &request {
                agent.report(id, RequestStatusReport::failed(code, message));
            }
        };

        match outcome {
            ScanOutcome::Finished if active.pages > 0 => self.complete(&active.key),
            ScanOutcome::Finished if stopped_by_person => {
                self.complete(&active.key);
                fail(
                    self,
                    RequestFailureCode::CanceledByUser,
                    "The scan was stopped before any page was scanned.",
                );
            }
            ScanOutcome::Finished => {
                self.complete(&active.key);
                fail(
                    self,
                    RequestFailureCode::FeederEmpty,
                    "Nothing was scanned.",
                );
                self.notify(
                    Severity::Warning,
                    "Nothing was scanned",
                    "Check the paper is loaded, then try again.",
                    None,
                );
            }
            ScanOutcome::Stopped { condition, message } if active.pages > 0 => {
                let pages = active.pages;
                let mut order = active.order.clone();
                order.pages = pages;
                self.paused.insert(active.key.clone(), order);
                let paused = PausedBatch {
                    key: active.key.clone(),
                    label: active.label.clone(),
                    pages,
                    condition,
                };
                self.shared.update(|s| s.paused.push(paused));
                self.shared.attention(Attention::ScanPaused);
                self.notify(
                    Severity::Warning,
                    format!("Scanning stopped after {}", plural(pages, "page", "pages")),
                    format!("{message} Fix it, then choose Continue scanning in Trenova Capture, or Finish to send what was scanned."),
                    None,
                );
                self.start_next();
                return;
            }
            ScanOutcome::Stopped { condition, message } => {
                self.complete(&active.key);
                fail(self, condition.failure_code(), &message);
                self.notify(Severity::Warning, "Nothing was scanned", message, None);
            }
            ScanOutcome::Failed { code, message } => {
                self.complete(&active.key);
                if active.pages > 0 {
                    self.notify(
                        Severity::Error,
                        "Scanning stopped",
                        format!(
                            "{message} The {} scanned first are being sent.",
                            plural(active.pages, "page", "pages")
                        ),
                        None,
                    );
                } else {
                    fail(self, code, &message);
                    self.notify(Severity::Error, "The scan failed", message, None);
                }
            }
        }
        self.shared.attention(Attention::ScanEnded);
        self.start_next();
    }

    fn stream_event(&mut self, event: DeviceEvent) {
        match event {
            DeviceEvent::Connected => {
                self.blocked = false;
                self.outage += 1;
                self.shared.update(|s| s.connection = Connection::Online);
                if std::mem::take(&mut self.outage_announced) {
                    let waiting = self.shared.snapshot().pages_waiting;
                    let body = if waiting > 0 {
                        format!(
                            "The {} kept on this computer are being sent.",
                            plural(waiting, "page", "pages")
                        )
                    } else {
                        "Scans and prints reach Trenova again.".to_owned()
                    };
                    self.notify_routine("Connected to Trenova again", body, None);
                }
            }
            DeviceEvent::Disconnected { reason, .. } => {
                let already = matches!(
                    self.shared.snapshot().connection,
                    Connection::Offline { .. }
                );
                self.shared
                    .update(|s| s.connection = Connection::Offline { reason });
                if !already {
                    self.outage += 1;
                    let (tx, wait, outage) = (
                        self.internal_tx.clone(),
                        self.env.offline_notice_after,
                        self.outage,
                    );
                    tokio::spawn(async move {
                        tokio::time::sleep(wait).await;
                        let _ = tx.send(Internal::StillOffline(outage)).await;
                    });
                }
            }
            DeviceEvent::FetchRequests => self.fetch_requests(),
            DeviceEvent::Revoked => {
                self.stop_session();
                self.shared.update(|s| {
                    s.connection = Connection::SignedOut;
                    s.person = None;
                    s.organization = None;
                });
                self.notify(
                    Severity::Warning,
                    "This computer was removed from Trenova",
                    "Sign in again to keep scanning. Pages not yet sent are kept.",
                    None,
                );
            }
            DeviceEvent::Stopped(err) => self.api_error(&err),
        }
    }

    fn upload_event(&mut self, event: UploadEvent) {
        match event {
            UploadEvent::Progress(_) => self.refresh_spool(),
            UploadEvent::Sent {
                batch_id,
                pages,
                label,
                requested,
                ..
            } => {
                let link = self.shared.snapshot().intake_link(Some(&batch_id));
                let recent = RecentBatch {
                    label: label.clone(),
                    pages,
                    link: link.clone().unwrap_or_default(),
                    at: SystemTime::now(),
                    requested,
                };
                self.shared.update(|s| {
                    s.recent.push_front(recent);
                    s.recent.truncate(RECENT);
                });
                self.refresh_spool();
                let body = if requested {
                    "They are being filed where you asked. Anything that needs a look waits in Intake."
                } else {
                    "They are waiting in Intake."
                };
                self.notify_routine(
                    format!("{} sent to Trenova", plural(pages, "page", "pages")),
                    body,
                    link,
                );
            }
            UploadEvent::Refused {
                source,
                label,
                reason,
            } => {
                self.refresh_spool();
                self.shared.attention(Attention::Refused);
                self.notify(
                    Severity::Error,
                    if source == BatchSource::Print {
                        "A print could not be sent"
                    } else {
                        "A scan could not be sent"
                    },
                    format!(
                        "{label}: {reason} Its pages are kept on this computer; open Trenova Capture to send them again, save them, or discard them."
                    ),
                    None,
                );
            }
            UploadEvent::Blocked(err) => self.api_error(&err),
            UploadEvent::Waiting { reason, retry_in } => {
                tracing::info!(reason, retry_in = ?retry_in, "uploads are waiting");
            }
        }
    }

    /// A printed job reached the spool, or could not be read.
    fn printed(&mut self, imported: Imported) {
        match imported {
            Imported::Spooled {
                name, held: true, ..
            } => {
                tracing::info!(name, "took a print from the print inbox to be looked over");
                self.refresh_spool();
                self.review_ready(
                    "Look over the print before it is sent",
                    format!("{name} waits for you in Trenova Capture."),
                );
            }
            Imported::Spooled { name, .. } => {
                tracing::info!(name, "took a print from the print inbox");
                if self.signed_in() {
                    self.wake_uploader();
                } else {
                    self.notify(
                        Severity::Warning,
                        "Your print is waiting",
                        format!("Sign in to Trenova Capture to send {name} to Trenova."),
                        None,
                    );
                }
                self.refresh_spool();
            }
            Imported::Rejected { name, reason } => self.notify(
                Severity::Error,
                "A print could not be read",
                format!("{name}: {reason}. Print it again; the unreadable copy was kept on this computer."),
                None,
            ),
        }
    }

    /// Asks the server for the current release once, outside the regular
    /// schedule: after a 426, when the update is what unblocks everything.
    fn check_release(&self) {
        let Some(api) = self.api.clone() else {
            return;
        };
        let (tx, key) = (self.internal_tx.clone(), self.env.release_key);
        tokio::spawn(async move {
            let _ = tx
                .send(Internal::Release(updates::check(&api, key.as_ref()).await))
                .await;
        });
    }

    /// Whether the print service is here without its printer, for the menu.
    fn check_printer(&self) -> bool {
        let missing = self.env.printer.missing();
        self.shared.update(|s| s.printer_missing = missing);
        missing
    }

    /// The menu tried to add the printer; say how it went.
    fn printer_set_up(&mut self, attempt: PrinterAttempt) {
        let missing = self.check_printer();
        if !missing {
            self.notify(
                Severity::Info,
                "The Trenova printer is ready",
                "Print to Trenova from any program to send the pages to Intake.",
                None,
            );
            return;
        }
        if attempt == PrinterAttempt::Declined {
            return;
        }
        self.notify(
            Severity::Warning,
            "The Trenova printer could not be added",
            "Try again from the Trenova Capture menu, or ask your administrator to add it.",
            None,
        );
    }

    /// Whether the organization and this computer both let the companion
    /// install a release itself.
    fn may_self_update(&self) -> bool {
        self.update_policy.allow_auto_update && self.env.machine_auto_update
    }

    /// What the server publishes, compared with what this is.
    fn release(&mut self, release: Result<Option<Release>, ApiError>) {
        let release = match release {
            Ok(Some(release)) => release,
            Ok(None) => {
                self.shared.update(|s| s.update = None);
                return;
            }
            Err(err) => {
                tracing::warn!(error = %err, "could not check for a new release");
                return;
            }
        };
        if !self.policy_known {
            self.held_release = Some(release);
            return;
        }
        let status = match decide(&self.env.agent.version, &release, self.env.windows_build) {
            Decision::UpToDate => {
                self.shared.update(|s| s.update = None);
                return;
            }
            Decision::WindowsTooOld { .. } => UpdateStatus::WindowsTooOld,
            Decision::Install if self.may_self_update() => UpdateStatus::Available,
            Decision::Install => UpdateStatus::AskAdministrator,
        };
        let known = self.shared.snapshot().update;
        let version = release.version.clone();
        self.shared.update(|s| {
            s.update = Some(UpdateState {
                version: version.clone(),
                download_url: release.installer.url.clone(),
                status: match &s.update {
                    Some(current) if current.status == UpdateStatus::Installing => {
                        UpdateStatus::Installing
                    }
                    _ => status,
                },
            });
        });
        match status {
            UpdateStatus::Available => self.install_update(false),
            UpdateStatus::AskAdministrator
                if known.as_ref().is_none_or(|k| k.version != version) =>
            {
                self.notify(
                    Severity::Info,
                    format!("Trenova Capture {version} is available"),
                    "Your organization installs updates itself; ask your administrator.",
                    None,
                );
            }
            _ => {}
        }
    }

    /// Starts the updater for the release the snapshot names. A scan in
    /// progress is never interrupted; `asked` is a person choosing it from
    /// the menu, who is told why it waits.
    fn install_update(&mut self, asked: bool) {
        let Some(update) = self.shared.snapshot().update else {
            return;
        };
        if update.status != UpdateStatus::Available || !self.may_self_update() {
            return;
        }
        if self.active.is_some() {
            if asked {
                self.notify(
                    Severity::Info,
                    "The update will start after this scan",
                    "Trenova Capture does not update while a scanner is running.",
                    None,
                );
            }
            return;
        }
        if self.update_started.as_deref() == Some(update.version.as_str()) {
            return;
        }
        let Some(api) = &self.api else {
            return;
        };
        match self.env.updater.start(&updates::manifest_url(api)) {
            Ok(()) => {
                self.update_started = Some(update.version.clone());
                self.shared.update(|s| {
                    if let Some(u) = &mut s.update {
                        u.status = UpdateStatus::Installing;
                    }
                });
                self.notify(
                    Severity::Info,
                    format!("Updating Trenova Capture to {}", update.version),
                    "It restarts by itself when the update is done.",
                    None,
                );
            }
            Err(err) => {
                tracing::error!(error = %err, "could not start the updater");
                self.notify(
                    Severity::Warning,
                    "Trenova Capture could not update",
                    format!(
                        "{err}. Ask your administrator to install version {}.",
                        update.version
                    ),
                    Some(update.download_url.clone()),
                );
            }
        }
    }

    /// Acts on an error from anywhere: most are logged, but one that blocks
    /// everything ends the session and says why.
    fn api_error(&mut self, err: &ApiError) {
        if !err.blocks_everything() {
            tracing::warn!(error = %err, "a call to Trenova failed");
            return;
        }
        let already = self.blocked;
        self.stop_session();
        match err {
            ApiError::SignedOut | ApiError::NotSignedIn => {
                self.shared.update(|s| {
                    s.connection = Connection::SignedOut;
                    s.person = None;
                    s.organization = None;
                });
                self.notify(
                    Severity::Warning,
                    "Signed out of Trenova",
                    "Sign in again from the Trenova Capture menu. Pages not yet sent are kept.",
                    None,
                );
                return;
            }
            ApiError::Outdated {
                minimum_version,
                auto_update,
            } => {
                let minimum = minimum_version.clone();
                self.update_policy.minimum_version.clone_from(&minimum);
                self.update_policy.allow_auto_update = *auto_update;
                self.policy_known = true;
                self.shared.update(|s| {
                    s.update_required = Some(minimum.clone());
                    s.connection = Connection::Blocked {
                        reason: format!("Trenova Capture {minimum} or later is required."),
                    };
                });
                self.check_release();
            }
            other => {
                let reason = other.to_string();
                self.shared
                    .update(|s| s.connection = Connection::Blocked { reason });
            }
        }
        self.blocked = true;
        if !already {
            self.notify(
                Severity::Warning,
                "Scanning into Trenova is paused",
                format!("{err} Pages already scanned are kept and sent once this is resolved."),
                None,
            );
        }
        let (tx, wait) = (self.internal_tx.clone(), self.env.recheck_after);
        tokio::spawn(async move {
            tokio::time::sleep(wait).await;
            let _ = tx.send(Internal::Recheck).await;
        });
    }
}
