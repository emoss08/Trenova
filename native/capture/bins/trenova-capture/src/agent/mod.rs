//! The agent: one task that owns everything the tray shows.
//!
//! It keeps the session with the server (the device stream, the uploader),
//! turns requests into scans, runs one scan at a time through the helpers,
//! and spools every page before anything else happens to it. It also takes
//! printed jobs from this person's print inbox (`prints`) into the same spool.
//! Network calls
//! run as their own tasks and report back here, so a slow server never holds
//! up a command from the tray, and there is exactly one place state changes.

pub mod plan;
pub mod prints;

use std::collections::{HashMap, HashSet, VecDeque};
use std::io;
use std::path::PathBuf;
use std::sync::Arc;
use std::time::{Duration, SystemTime};

use capture_client::pairing::{PairingOutcome, pair};
use capture_client::stream::{self, DeviceEvent};
use capture_client::uploader::{UploadEvent, Uploader};
use capture_client::{
    AgentInfo, Api, ApiError, PageMarkers, Protector, SecretStore, Server, Spool, SpoolError,
};
use capture_protocol::api::{
    Architecture, BatchSource, CaptureProfile, CaptureRequest, DeviceIdentity, Id,
    RequestFailureCode, RequestMode, RequestStatus, RequestStatusReport, SourceInfo,
    SourceProtocol, StartPairingRequest,
};
use capture_protocol::handoff::Inbox;
use tokio::sync::{Notify, mpsc};
use tokio_util::sync::CancellationToken;

use self::prints::Imported;
use crate::scanners::{ScanOutcome, ScanRun, ScanUpdate, ScannerHost};
use crate::state::{
    Command, Connection, Notice, PausedBatch, RECENT, RecentBatch, Severity, Shared,
};

/// Where the server address is kept.
pub trait ServerSetting: Send + Sync {
    fn load(&self) -> Option<String>;
    fn save(&self, url: &str) -> io::Result<()>;
    /// Whether an administrator set it by policy, so a person cannot.
    fn locked(&self) -> bool;
}

pub type SecretsFor = dyn Fn(&Server) -> Arc<dyn SecretStore> + Send + Sync;
pub type Browser = dyn Fn(&str) + Send + Sync;

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
    /// How long to wait before trying again after being blocked.
    pub recheck_after: Duration,
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
    Recheck,
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
    };
    if let Some(dir) = agent.env.print_inbox.clone() {
        let (tx, rx) = mpsc::channel(16);
        tokio::spawn(prints::watch(
            Inbox::new(dir),
            Arc::clone(&agent.spool),
            tx,
            cancel.child_token(),
        ));
        agent.prints_rx = Some(rx);
    }
    let failed_dir = agent.spool.failed_dir();
    agent.shared.update(|s| s.failed_dir = Some(failed_dir));
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
                Some(imported) => agent.printed(imported).await,
                None => agent.prints_rx = None,
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
            return;
        };
        let server = match Server::parse(url) {
            Ok(server) => server,
            Err(err) => {
                self.shared
                    .update(|s| s.connection = Connection::NeedsServer);
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

    fn stop_session(&mut self) {
        if let Some(token) = self.session.take() {
            token.cancel();
        }
        self.stream_rx = None;
        self.upload_rx = None;
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
            Command::RefreshScanners => self.enumerate(),
            Command::Quit => {}
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
                browser(&grant.verification_uri_complete);
                shared.notify(Notice {
                    title: format!("Approve this computer with code {}", grant.user_code),
                    body: "Trenova opened in your browser. Check the code matches, then approve."
                        .into(),
                    severity: Severity::Info,
                    link: Some(grant.verification_uri_complete.clone()),
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

    fn scan_to_intake(&mut self, name: &str, protocol: SourceProtocol, profile: Option<&Id>) {
        if !self.signed_in() {
            self.notify(
                Severity::Warning,
                "Sign in to scan",
                "Choose Sign in in the Trenova Capture menu.",
                None,
            );
            return;
        }
        let Some(source) = self
            .sources
            .iter()
            .find(|s| s.name == name && s.protocol == protocol)
            .cloned()
        else {
            self.notify(Severity::Error, "That scanner is not connected", name, None);
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

    async fn internal(&mut self, message: Internal) {
        match message {
            Internal::Identity(Ok(identity)) => {
                self.shared.update(|s| {
                    s.person = Some(identity.person.name.clone());
                    s.organization = Some(identity.organization.name.clone());
                });
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
            Internal::Recheck => {
                if self.blocked && self.api.is_some() {
                    self.start_session().await;
                }
            }
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
            if let Err(err) = self.spool.create(input, label.clone()) {
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
        let scanning = format!("Scanning from {label}");
        self.shared.update(|s| s.scanning = Some(scanning));
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
            ScanUpdate::Page { meta, pdf } => self.page(meta, &pdf),
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

    fn page(&mut self, meta: capture_protocol::helper::PageMeta, pdf: &[u8]) {
        let Some(active) = &mut self.active else {
            return;
        };
        let markers = PageMarkers {
            dpi: meta.dpi,
            patch_code: meta.patch_code,
            barcodes: meta.barcodes,
        };
        match self.spool.append_page(&active.key, pdf, &markers) {
            Ok(_) => {
                active.pages += 1;
                let scanning = format!(
                    "Scanning from {}: {}",
                    active.label,
                    plural(active.pages, "page", "pages")
                );
                self.shared.update(|s| s.scanning = Some(scanning));
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
            Ok(_) => self.wake_uploader(),
            Err(err) => tracing::error!(key, error = %err, "could not finish a batch"),
        }
    }

    fn scan_ended(&mut self, outcome: ScanOutcome) {
        let Some(active) = self.active.take() else {
            return;
        };
        self.shared.update(|s| s.scanning = None);
        let request = active.order.request.as_ref().map(|r| r.id.clone());
        let fail = |agent: &Self, code: RequestFailureCode, message: &str| {
            if let Some(id) = &request {
                agent.report(id, RequestStatusReport::failed(code, message));
            }
        };

        match outcome {
            ScanOutcome::Finished if active.pages > 0 => self.complete(&active.key),
            ScanOutcome::Finished => {
                self.complete(&active.key);
                fail(
                    self,
                    RequestFailureCode::FeederEmpty,
                    "Nothing was scanned.",
                );
                if !active.run.cancel.is_cancelled() {
                    self.notify(
                        Severity::Warning,
                        "Nothing was scanned",
                        "Check the paper is loaded, then try again.",
                        None,
                    );
                }
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
                self.notify(
                    Severity::Warning,
                    format!("Scanning stopped after {}", plural(pages, "page", "pages")),
                    format!("{message} Fix it, then choose Continue scanning in the Trenova Capture menu, or Finish to send what was scanned."),
                    None,
                );
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
        self.start_next();
    }

    fn stream_event(&mut self, event: DeviceEvent) {
        match event {
            DeviceEvent::Connected => {
                self.blocked = false;
                self.shared.update(|s| s.connection = Connection::Online);
            }
            DeviceEvent::Disconnected { reason, .. } => {
                self.shared
                    .update(|s| s.connection = Connection::Offline { reason });
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
            UploadEvent::Progress(summary) => self.shared.update(|s| {
                s.pages_waiting = summary.pages_waiting;
                s.failed = summary.failed;
            }),
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
                let body = if requested {
                    "They are being filed where you asked. Anything that needs a look waits in Intake."
                } else {
                    "They are waiting in Intake."
                };
                self.notify(
                    Severity::Info,
                    format!("{} sent to Trenova", plural(pages, "page", "pages")),
                    body,
                    link,
                );
            }
            UploadEvent::Refused {
                source,
                label,
                reason,
            } => self.notify(
                Severity::Error,
                if source == BatchSource::Print {
                    "A print could not be sent"
                } else {
                    "A scan could not be sent"
                },
                format!("{label}: {reason} Its pages are kept in the failed uploads folder."),
                None,
            ),
            UploadEvent::Blocked(err) => self.api_error(&err),
            UploadEvent::Waiting { reason, retry_in } => {
                tracing::info!(reason, retry_in = ?retry_in, "uploads are waiting");
            }
        }
    }

    /// A printed job reached the spool, or could not be read.
    async fn printed(&mut self, imported: Imported) {
        match imported {
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
                let spool = Arc::clone(&self.spool);
                if let Ok(Ok(summary)) = tokio::task::spawn_blocking(move || spool.summary()).await
                {
                    self.shared.update(|s| {
                        s.pages_waiting = summary.pages_waiting;
                        s.failed = summary.failed;
                    });
                }
            }
            Imported::Rejected { name, reason } => self.notify(
                Severity::Error,
                "A print could not be read",
                format!("{name}: {reason}. Print it again; the unreadable copy was kept on this computer."),
                None,
            ),
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
            ApiError::Outdated { minimum_version } => {
                let minimum = minimum_version.clone();
                self.shared.update(|s| {
                    s.update_required = Some(minimum.clone());
                    s.connection = Connection::Blocked {
                        reason: format!("Trenova Capture {minimum} or later is required."),
                    };
                });
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
