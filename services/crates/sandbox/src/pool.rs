//! The supervisor's side: a fixed number of children, each running one job at
//! a time.
//!
//! A job takes a slot, runs on that slot's child, and gives the slot back.
//! The child is replaced, and the job fails with a typed error, whenever the
//! child does not finish cleanly: it crashed or was killed by a signal
//! (a parser fault, the address-space limit, a seccomp violation), passed
//! its deadline, broke the protocol, or its caller went away. A healthy
//! child is retired after a number of jobs, when its resident memory passes
//! a high-water mark, or as it nears its CPU limit, so leaks and
//! fragmentation in the native libraries never accumulate.
//!
//! The output of a job flows through a bounded channel, so a child that
//! produces faster than the caller consumes simply waits: neither side holds
//! more than a few frames.
//!
//! Children are signalled through a pidfd, so a kill can never reach an
//! unrelated process that reused a reaped child's pid.

use std::ffi::OsString;
use std::io::{BufRead, BufReader};
use std::os::fd::{AsFd, OwnedFd};
use std::os::unix::net::UnixStream;
use std::os::unix::process::{CommandExt, ExitStatusExt};
use std::path::PathBuf;
use std::process::{Command, Stdio};
use std::sync::atomic::{AtomicBool, AtomicU8, AtomicU64, Ordering};
use std::sync::{Arc, Mutex, PoisonError};
use std::time::Duration;

use prometheus_client::encoding::{EncodeLabelSet, EncodeLabelValue};
use prometheus_client::metrics::counter::Counter;
use prometheus_client::metrics::family::Family;
use prometheus_client::metrics::gauge::Gauge;
use prometheus_client::registry::Registry;
use rustix::process::{Pid, PidfdFlags, Signal, pidfd_open, pidfd_send_signal};
use tokio::sync::{Semaphore, mpsc, oneshot};
use tokio::time::Instant;

use crate::frame::{Channel, FrameError, Kind};
use crate::harden::{HardeningPolicy, HardeningReport};
use crate::input::SealedInput;
use crate::wire::encode_init;

/// The limits set on every child before it runs any code of its own.
#[derive(Clone, Copy, Debug, PartialEq, Eq)]
pub struct Limits {
    /// `RLIMIT_AS`: the most address space the child may map.
    pub address_space_bytes: u64,
    /// `RLIMIT_CPU`: CPU seconds over the child's whole life.
    pub cpu_seconds: u64,
    /// `RLIMIT_FSIZE`: the largest file it may write.
    pub file_size_bytes: u64,
    /// `RLIMIT_NOFILE`: descriptors it may hold.
    pub open_files: u64,
}

/// How the pool runs.
#[derive(Clone, Debug)]
pub struct PoolConfig {
    pub size: usize,
    pub max_jobs_per_child: u32,
    pub rss_high_water_bytes: u64,
    pub limits: Limits,
    pub max_frame_bytes: u32,
    /// How long a new child has to initialise, harden and report.
    pub start_timeout: Duration,
    pub policy: HardeningPolicy,
    /// Refuse a child whose protections fall short of the policy. Turning
    /// this off is for development machines without landlock or seccomp,
    /// and is logged at error level on every spawn.
    pub require_hardening: bool,
}

/// What to run as a child.
#[derive(Clone, Debug)]
pub struct ChildSpec {
    pub program: PathBuf,
    pub args: Vec<OsString>,
    /// The service's start-up parameters, passed in the `Init` frame.
    pub init: Vec<u8>,
}

/// A job for a child.
#[derive(Debug)]
pub struct Job {
    /// The service's request, opaque to the pool.
    pub body: Vec<u8>,
    pub input: Option<SealedInput>,
    pub deadline: Instant,
}

/// How a job ended when the child finished it.
#[derive(Debug, PartialEq, Eq)]
pub enum Outcome {
    Done(Vec<u8>),
    /// The child ran the job and reports the service's typed error.
    Failed(Vec<u8>),
}

/// How a job ended when the child did not finish it. The child is replaced
/// in every case.
#[derive(Debug, Clone, PartialEq, Eq, thiserror::Error)]
pub enum JobError {
    #[error("the job passed its deadline and its process was killed")]
    DeadlineExceeded,
    #[error("the job's process ended abnormally ({0})")]
    Crashed(String),
    #[error("the job's process broke the protocol: {0}")]
    Protocol(String),
    #[error("the caller went away and the job was stopped")]
    Cancelled,
    #[error("the pool is closed")]
    Closed,
    #[error("no process could be started for the job: {0}")]
    Spawn(String),
}

#[derive(Debug, thiserror::Error)]
pub enum SandboxError {
    #[error("start a child: {0}")]
    Spawn(String),
    #[error("a child is not hardened as required: {0}")]
    Unhardened(String),
}

#[derive(Clone, Copy, Debug, Hash, PartialEq, Eq, EncodeLabelValue)]
enum KillReason {
    Deadline,
    Cancelled,
    Protocol,
    Crash,
}

#[derive(Clone, Copy, Debug, Hash, PartialEq, Eq, EncodeLabelValue)]
enum RetireReason {
    Jobs,
    Rss,
    Cpu,
}

#[derive(Clone, Debug, Hash, PartialEq, Eq, EncodeLabelSet)]
struct KillLabel {
    reason: KillReason,
}

#[derive(Clone, Debug, Hash, PartialEq, Eq, EncodeLabelSet)]
struct RetireLabel {
    reason: RetireReason,
}

#[derive(Debug, Default)]
struct Metrics {
    spawned: Counter,
    spawn_failures: Counter,
    killed: Family<KillLabel, Counter>,
    retired: Family<RetireLabel, Counter>,
    live: Gauge,
    size: Gauge,
    busy: Gauge,
    child_rss_bytes: Gauge,
}

struct Worker {
    process: std::process::Child,
    pidfd: Arc<OwnedFd>,
    channel: Channel,
    jobs: u32,
}

impl std::fmt::Debug for Worker {
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        f.debug_struct("Worker")
            .field("pid", &self.process.id())
            .field("jobs", &self.jobs)
            .finish_non_exhaustive()
    }
}

#[derive(Debug)]
struct Inner {
    config: PoolConfig,
    spec: ChildSpec,
    slots: Arc<Semaphore>,
    idle: Mutex<Vec<Worker>>,
    live: AtomicU64,
    closed: AtomicBool,
    metrics: Metrics,
}

/// The pool. Cheap to clone.
#[derive(Clone, Debug)]
pub struct Pool {
    inner: Arc<Inner>,
}

const RUNNING: u8 = 0;
const FINISHED: u8 = 1;
const KILLED_DEADLINE: u8 = 2;
const KILLED_CANCELLED: u8 = 3;

/// Shared between a job's caller, its deadline and the thread talking to its
/// child: who stopped the job, if anyone did.
#[derive(Debug)]
struct Control {
    pidfd: Arc<OwnedFd>,
    state: AtomicU8,
}

impl Control {
    fn kill(&self, reason: u8) {
        if self
            .state
            .compare_exchange(RUNNING, reason, Ordering::AcqRel, Ordering::Acquire)
            .is_ok()
        {
            let _ = pidfd_send_signal(self.pidfd.as_fd(), Signal::KILL);
        }
    }

    fn finish(&self) {
        let _ = self
            .state
            .compare_exchange(RUNNING, FINISHED, Ordering::AcqRel, Ordering::Acquire);
    }

    fn reason(&self) -> u8 {
        self.state.load(Ordering::Acquire)
    }
}

/// Stops the job if the caller's future is dropped before it ends. Once the
/// job has ended the kill finds the state settled, or a reaped process the
/// pidfd still names, and does nothing.
struct CancelOnDrop(Arc<Control>);

impl Drop for CancelOnDrop {
    fn drop(&mut self) {
        self.0.kill(KILLED_CANCELLED);
    }
}

impl Pool {
    /// Starts `config.size` children and waits until each has reported in.
    /// Fails, starting nothing, if any child cannot start or, when hardening
    /// is required, reports less than the policy.
    pub async fn start(
        config: PoolConfig,
        spec: ChildSpec,
        registry: &mut Registry,
    ) -> Result<Self, SandboxError> {
        let metrics = Metrics::default();
        let sub = registry.sub_registry_with_prefix("sandbox");
        sub.register(
            "spawned",
            "Child processes started",
            metrics.spawned.clone(),
        );
        sub.register(
            "spawn_failures",
            "Child processes that failed to start",
            metrics.spawn_failures.clone(),
        );
        sub.register(
            "killed",
            "Child processes killed, by reason",
            metrics.killed.clone(),
        );
        sub.register(
            "retired",
            "Healthy child processes replaced, by reason",
            metrics.retired.clone(),
        );
        sub.register("live", "Child processes running", metrics.live.clone());
        sub.register(
            "size",
            "Child processes the pool is configured to keep",
            metrics.size.clone(),
        );
        sub.register(
            "busy",
            "Child processes running a job",
            metrics.busy.clone(),
        );
        sub.register(
            "child_rss_bytes",
            "Resident memory of the last child to finish a job",
            metrics.child_rss_bytes.clone(),
        );

        let size = config.size.max(1);
        metrics.size.set(i64::try_from(size).unwrap_or(i64::MAX));
        let inner = Arc::new(Inner {
            config,
            spec,
            slots: Arc::new(Semaphore::new(size)),
            idle: Mutex::new(Vec::with_capacity(size)),
            live: AtomicU64::new(0),
            closed: AtomicBool::new(false),
            metrics,
        });

        let mut started = Vec::with_capacity(size);
        for _ in 0..size {
            let inner = Arc::clone(&inner);
            started.push(tokio::task::spawn_blocking(move || spawn_worker(&inner)));
        }
        let mut failure = None;
        for handle in started {
            match handle.await {
                Ok(Ok(worker)) => lock(&inner.idle).push(worker),
                Ok(Err(err)) => failure = failure.or(Some(err)),
                Err(err) => failure = failure.or(Some(SandboxError::Spawn(err.to_string()))),
            }
        }
        let pool = Self { inner };
        if let Some(err) = failure {
            pool.close().await;
            return Err(err);
        }
        Ok(pool)
    }

    /// Runs a job on the next free child. Output frames are sent to
    /// `outputs` as they arrive; the channel's bound is the backpressure.
    /// Dropping the returned future stops the job.
    pub async fn run(&self, job: Job, outputs: mpsc::Sender<Vec<u8>>) -> Result<Outcome, JobError> {
        if self.inner.closed.load(Ordering::Acquire) {
            return Err(JobError::Closed);
        }
        let permit = Arc::clone(&self.inner.slots)
            .acquire_owned()
            .await
            .map_err(|_| JobError::Closed)?;
        if Instant::now() >= job.deadline {
            return Err(JobError::DeadlineExceeded);
        }

        let worker = lock(&self.inner.idle).pop();
        let inner = Arc::clone(&self.inner);
        let worker = if let Some(worker) = worker {
            worker
        } else {
            let spawner = Arc::clone(&inner);
            tokio::task::spawn_blocking(move || spawn_worker(&spawner))
                .await
                .map_err(|err| JobError::Spawn(err.to_string()))?
                .map_err(|err| JobError::Spawn(err.to_string()))?
        };

        let control = Arc::new(Control {
            pidfd: Arc::clone(&worker.pidfd),
            state: AtomicU8::new(RUNNING),
        });
        let guard = CancelOnDrop(Arc::clone(&control));
        let (done_tx, done_rx) = oneshot::channel();
        let deadline = job.deadline;

        tokio::spawn(async move {
            inner.metrics.busy.inc();
            let thread_control = Arc::clone(&control);
            let mut execution = tokio::task::spawn_blocking(move || {
                execute(worker, job, &outputs, &thread_control)
            });
            let joined = tokio::select! {
                joined = &mut execution => joined,
                () = tokio::time::sleep_until(deadline) => {
                    control.kill(KILLED_DEADLINE);
                    execution.await
                }
            };
            inner.metrics.busy.dec();

            let (worker, result) = match joined {
                Ok(pair) => pair,
                Err(err) => (None, Err(JobError::Crashed(err.to_string()))),
            };
            if let Err(err) = &result {
                inner
                    .metrics
                    .killed
                    .get_or_create(&KillLabel {
                        reason: kill_reason(err),
                    })
                    .inc();
            }
            let recycle = Arc::clone(&inner);
            let _ = tokio::task::spawn_blocking(move || settle(&recycle, worker)).await;
            drop(permit);
            let _ = done_tx.send(result);
        });

        let result = done_rx.await.unwrap_or(Err(JobError::Closed));
        drop(guard);
        result
    }

    /// Whether every slot has a live child.
    pub fn healthy(&self) -> bool {
        !self.inner.closed.load(Ordering::Acquire)
            && self.inner.live.load(Ordering::Acquire) >= self.inner.config.size as u64
    }

    /// Slots free for a job right now.
    pub fn available(&self) -> usize {
        self.inner.slots.available_permits()
    }

    pub fn size(&self) -> usize {
        self.inner.config.size
    }

    /// Stops taking jobs, waits for running ones, and stops every child.
    pub async fn close(&self) {
        self.inner.closed.store(true, Ordering::Release);
        let size = u32::try_from(self.inner.config.size).unwrap_or(u32::MAX);
        let all = Arc::clone(&self.inner.slots).acquire_many_owned(size).await;
        let workers: Vec<Worker> = lock(&self.inner.idle).drain(..).collect();
        let inner = Arc::clone(&self.inner);
        let _ = tokio::task::spawn_blocking(move || {
            for worker in workers {
                retire(&inner, worker);
            }
        })
        .await;
        drop(all);
        self.inner.slots.close();
    }
}

fn kill_reason(err: &JobError) -> KillReason {
    match err {
        JobError::DeadlineExceeded => KillReason::Deadline,
        JobError::Cancelled => KillReason::Cancelled,
        JobError::Protocol(_) => KillReason::Protocol,
        JobError::Crashed(_) | JobError::Closed | JobError::Spawn(_) => KillReason::Crash,
    }
}

fn lock<T>(mutex: &Mutex<T>) -> std::sync::MutexGuard<'_, T> {
    mutex.lock().unwrap_or_else(PoisonError::into_inner)
}

/// Talks to the child for one job. Returns the worker when it is still fit
/// to use.
fn execute(
    mut worker: Worker,
    job: Job,
    outputs: &mpsc::Sender<Vec<u8>>,
    control: &Control,
) -> (Option<Worker>, Result<Outcome, JobError>) {
    let fds: Vec<_> = job.input.iter().map(SealedInput::as_fd).collect();
    if let Err(err) = worker.channel.send(Kind::Job, &job.body, &fds) {
        return (None, Err(abandon(&mut worker, control, &err)));
    }
    drop(fds);
    drop(job.input);
    worker.jobs += 1;

    loop {
        match worker.channel.recv() {
            Ok(frame) => match frame.kind {
                Kind::Output => {
                    if outputs.blocking_send(frame.body).is_err() {
                        control.kill(KILLED_CANCELLED);
                    }
                }
                Kind::Done | Kind::Failed => {
                    control.finish();
                    if control.reason() != FINISHED {
                        let reason = control.reason();
                        reap(&mut worker);
                        return (None, Err(stopped(reason)));
                    }
                    let outcome = if frame.kind == Kind::Done {
                        Outcome::Done(frame.body)
                    } else {
                        Outcome::Failed(frame.body)
                    };
                    return (Some(worker), Ok(outcome));
                }
                _ => {
                    let err = FrameError::Protocol("unexpected frame from a child");
                    return (None, Err(abandon(&mut worker, control, &err)));
                }
            },
            Err(err) => return (None, Err(abandon(&mut worker, control, &err))),
        }
    }
}

fn stopped(reason: u8) -> JobError {
    match reason {
        KILLED_DEADLINE => JobError::DeadlineExceeded,
        _ => JobError::Cancelled,
    }
}

/// The channel failed: decide why, make sure the child is gone, reap it.
fn abandon(worker: &mut Worker, control: &Control, err: &FrameError) -> JobError {
    let reason = control.reason();
    let _ = pidfd_send_signal(worker.pidfd.as_fd(), Signal::KILL);
    let status = reap(worker);
    match reason {
        KILLED_DEADLINE | KILLED_CANCELLED => stopped(reason),
        _ => match err {
            FrameError::Protocol(_) | FrameError::TooLarge { .. } => {
                JobError::Protocol(err.to_string())
            }
            _ => JobError::Crashed(status),
        },
    }
}

fn reap(worker: &mut Worker) -> String {
    let _ = pidfd_send_signal(worker.pidfd.as_fd(), Signal::KILL);
    match worker.process.wait() {
        Ok(status) => match (status.signal(), status.code()) {
            (Some(signal), _) => format!("signal {signal}"),
            (None, Some(code)) => format!("exit code {code}"),
            (None, None) => "unknown status".into(),
        },
        Err(err) => format!("wait failed: {err}"),
    }
}

/// After a job: keep the worker, retire it, or replace a lost one.
fn settle(inner: &Inner, worker: Option<Worker>) {
    let Some(worker) = worker else {
        inner.live.fetch_sub(1, Ordering::AcqRel);
        inner.metrics.live.dec();
        replace(inner);
        return;
    };
    if let Some(reason) = retirement(inner, &worker) {
        inner
            .metrics
            .retired
            .get_or_create(&RetireLabel { reason })
            .inc();
        retire(inner, worker);
        replace(inner);
    } else {
        lock(&inner.idle).push(worker);
    }
}

fn replace(inner: &Inner) {
    if inner.closed.load(Ordering::Acquire) {
        return;
    }
    match spawn_worker(inner) {
        Ok(worker) => lock(&inner.idle).push(worker),
        Err(err) => {
            tracing::error!(error = %err, "could not replace a child; the next job will try again");
        }
    }
}

fn retirement(inner: &Inner, worker: &Worker) -> Option<RetireReason> {
    if worker.jobs >= inner.config.max_jobs_per_child {
        return Some(RetireReason::Jobs);
    }
    let pid = worker.process.id();
    if let Some(rss) = resident_bytes(pid) {
        inner
            .metrics
            .child_rss_bytes
            .set(i64::try_from(rss).unwrap_or(i64::MAX));
        if rss >= inner.config.rss_high_water_bytes {
            return Some(RetireReason::Rss);
        }
    }
    if let Some(cpu) = cpu_seconds(pid)
        && cpu * 10 >= inner.config.limits.cpu_seconds * 8
    {
        return Some(RetireReason::Cpu);
    }
    None
}

/// Asks a healthy child to exit, killing it if it does not within a second.
fn retire(inner: &Inner, mut worker: Worker) {
    let _ = worker.channel.send(Kind::Shutdown, &[], &[]);
    let deadline = std::time::Instant::now() + Duration::from_secs(1);
    loop {
        match worker.process.try_wait() {
            Ok(Some(_)) => break,
            Ok(None) if std::time::Instant::now() < deadline => {
                std::thread::sleep(Duration::from_millis(10));
            }
            _ => {
                reap(&mut worker);
                break;
            }
        }
    }
    inner.live.fetch_sub(1, Ordering::AcqRel);
    inner.metrics.live.dec();
}

fn resident_bytes(pid: u32) -> Option<u64> {
    let statm = std::fs::read_to_string(format!("/proc/{pid}/statm")).ok()?;
    let pages: u64 = statm.split_whitespace().nth(1)?.parse().ok()?;
    Some(pages * rustix::param::page_size() as u64)
}

fn cpu_seconds(pid: u32) -> Option<u64> {
    let stat = std::fs::read_to_string(format!("/proc/{pid}/stat")).ok()?;
    let after_name = stat.rsplit_once(')')?.1;
    let fields: Vec<&str> = after_name.split_whitespace().collect();
    let utime: u64 = fields.get(11)?.parse().ok()?;
    let stime: u64 = fields.get(12)?.parse().ok()?;
    let ticks = rustix::param::clock_ticks_per_second();
    Some((utime + stime) / ticks.max(1))
}

fn spawn_worker(inner: &Inner) -> Result<Worker, SandboxError> {
    let result = start_child(inner);
    match &result {
        Ok(_) => {
            inner.metrics.spawned.inc();
            inner.live.fetch_add(1, Ordering::AcqRel);
            inner.metrics.live.inc();
        }
        Err(_) => {
            inner.metrics.spawn_failures.inc();
        }
    }
    result
}

fn start_child(inner: &Inner) -> Result<Worker, SandboxError> {
    let spawn_err =
        |what: &str, err: &dyn std::fmt::Display| SandboxError::Spawn(format!("{what}: {err}"));
    let (ours, theirs) = UnixStream::pair().map_err(|err| spawn_err("socketpair", &err))?;
    let limits = inner.config.limits;

    let mut command = Command::new(&inner.spec.program);
    command
        .args(&inner.spec.args)
        .env_clear()
        .current_dir("/")
        .stdin(Stdio::from(OwnedFd::from(theirs)))
        .stdout(Stdio::null())
        .stderr(Stdio::piped());
    // SAFETY: the closure runs in the forked child before exec and calls only
    // `setrlimit` and `prctl`, which are async-signal-safe system calls; it
    // allocates nothing and touches no lock.
    unsafe {
        command.pre_exec(move || {
            set_limit(libc::RLIMIT_AS, limits.address_space_bytes)?;
            set_limit(libc::RLIMIT_CPU, limits.cpu_seconds)?;
            set_limit(libc::RLIMIT_FSIZE, limits.file_size_bytes)?;
            set_limit(libc::RLIMIT_NOFILE, limits.open_files)?;
            set_limit(libc::RLIMIT_CORE, 0)?;
            if libc::prctl(libc::PR_SET_PDEATHSIG, libc::SIGKILL, 0, 0, 0) != 0 {
                return Err(std::io::Error::last_os_error());
            }
            Ok(())
        });
    }

    let spawned = command.spawn();
    drop(command);
    let mut process = spawned.map_err(|err| spawn_err("spawn", &err))?;
    let pid = Pid::from_raw(i32::try_from(process.id()).unwrap_or(0))
        .ok_or_else(|| SandboxError::Spawn("no pid".into()))?;
    let pidfd = match pidfd_open(pid, PidfdFlags::empty()) {
        Ok(fd) => Arc::new(fd),
        Err(err) => {
            let _ = process.kill();
            let _ = process.wait();
            return Err(spawn_err("pidfd_open", &err));
        }
    };
    if let Some(stderr) = process.stderr.take() {
        let pid = process.id();
        std::thread::Builder::new()
            .name(format!("sandbox-log-{pid}"))
            .spawn(move || forward_log(pid, stderr))
            .map_err(|err| spawn_err("log thread", &err))?;
    }

    let mut worker = Worker {
        process,
        pidfd,
        channel: Channel::new(ours, inner.config.max_frame_bytes),
        jobs: 0,
    };
    match handshake(inner, &mut worker) {
        Ok(()) => Ok(worker),
        Err(err) => {
            reap(&mut worker);
            Err(err)
        }
    }
}

fn handshake(inner: &Inner, worker: &mut Worker) -> Result<(), SandboxError> {
    let init = encode_init(&inner.config.policy, &inner.spec.init);
    worker
        .channel
        .send(Kind::Init, &init, &[])
        .map_err(|err| SandboxError::Spawn(format!("send init: {err}")))?;
    worker
        .channel
        .stream()
        .set_read_timeout(Some(inner.config.start_timeout))
        .map_err(|err| SandboxError::Spawn(err.to_string()))?;
    let frame = worker
        .channel
        .recv()
        .map_err(|err| SandboxError::Spawn(format!("wait for ready: {err}")))?;
    worker
        .channel
        .stream()
        .set_read_timeout(None)
        .map_err(|err| SandboxError::Spawn(err.to_string()))?;
    if frame.kind != Kind::Ready {
        return Err(SandboxError::Spawn(
            "the child's first frame was not Ready".into(),
        ));
    }
    let report: HardeningReport = serde_json::from_slice(&frame.body)
        .map_err(|err| SandboxError::Spawn(format!("ready report: {err}")))?;

    if report.satisfies(&inner.config.policy) {
        tracing::debug!(pid = worker.process.id(), ?report.landlock, "child ready and hardened");
        return Ok(());
    }
    if inner.config.require_hardening {
        return Err(SandboxError::Unhardened(format!(
            "no_new_privs={} landlock={:?} seccomp={} address_space_limit={} problems={:?}",
            report.no_new_privs,
            report.landlock,
            report.seccomp,
            report.address_space_limit,
            report.problems
        )));
    }
    tracing::error!(
        pid = worker.process.id(),
        no_new_privs = report.no_new_privs,
        landlock = ?report.landlock,
        seccomp = report.seccomp,
        problems = ?report.problems,
        "SANDBOX PROTECTIONS ARE NOT IN FORCE: a child runs untrusted documents without them because require_hardening is off"
    );
    Ok(())
}

#[cfg(target_env = "gnu")]
type RlimitResource = libc::__rlimit_resource_t;
#[cfg(not(target_env = "gnu"))]
type RlimitResource = libc::c_int;

fn set_limit(resource: RlimitResource, value: u64) -> std::io::Result<()> {
    let limit = libc::rlimit {
        rlim_cur: value,
        rlim_max: value,
    };
    // SAFETY: `limit` is a valid, initialised rlimit for the duration of the call.
    if unsafe { libc::setrlimit(resource, &raw const limit) } != 0 {
        return Err(std::io::Error::last_os_error());
    }
    Ok(())
}

fn forward_log(pid: u32, stderr: std::process::ChildStderr) {
    const MAX_LINE: usize = 512;
    let reader = BufReader::new(stderr);
    for line in reader.split(b'\n') {
        let Ok(line) = line else { break };
        let text = String::from_utf8_lossy(&line[..line.len().min(MAX_LINE)]);
        if text.trim().is_empty() {
            continue;
        }
        tracing::info!(target: "trenova_sandbox::child", pid, "{text}");
    }
}
