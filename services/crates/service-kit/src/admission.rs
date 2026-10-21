//! Admission control: a process takes on only the work its budget covers and
//! tells everyone else when to come back.
//!
//! The budget is two quantities, cost units (what the service's own cost
//! model calls a unit of CPU work) and memory bytes, and every job declares
//! its estimate of both before it runs. Work comes in two priority classes.
//! Interactive work (a person waiting on a page) may use the whole budget;
//! batch work (a background job with its own durable retries) may use only
//! what is left after the interactive reserve, so a backlog of batch work
//! never makes a person wait.
//!
//! A job that does not fit may wait a short, bounded time for capacity, and
//! only a bounded number may wait at once. Beyond that it is refused with a
//! retry-after taken from how long recent jobs held their permits, so callers
//! back off by what the work actually costs rather than queueing without
//! bound. A single job larger than everything its class may use is admitted
//! alone, when nothing else of its class is running, rather than refused
//! forever.

use std::sync::atomic::{AtomicU64, Ordering};
use std::sync::{Arc, Mutex, PoisonError};
use std::time::{Duration, Instant};

use prometheus_client::encoding::{EncodeLabelSet, EncodeLabelValue};
use prometheus_client::metrics::counter::Counter;
use prometheus_client::metrics::family::Family;
use prometheus_client::metrics::gauge::Gauge;
use prometheus_client::registry::Registry;
use serde::Deserialize;
use tokio::sync::Notify;

/// Which class a job belongs to. Carried in request metadata.
#[derive(Clone, Copy, Debug, PartialEq, Eq, Hash, EncodeLabelValue)]
pub enum Priority {
    /// A person is waiting: a page view, a synchronous check.
    Interactive,
    /// A background job that retries durably on its own.
    Batch,
}

impl Priority {
    /// The metadata key the class travels under.
    pub const METADATA_KEY: &'static str = "x-trenova-priority";

    /// Reads the class from its metadata value. Anything but `interactive`
    /// is batch, so a caller cannot claim the reserve by mistake.
    pub fn from_metadata(value: Option<&str>) -> Self {
        match value {
            Some(v) if v.eq_ignore_ascii_case("interactive") => Self::Interactive,
            _ => Self::Batch,
        }
    }

    pub fn as_str(self) -> &'static str {
        match self {
            Self::Interactive => "interactive",
            Self::Batch => "batch",
        }
    }

    fn index(self) -> usize {
        match self {
            Self::Interactive => 0,
            Self::Batch => 1,
        }
    }
}

/// The budget, as configured.
#[derive(Clone, Debug, Deserialize, PartialEq)]
#[serde(deny_unknown_fields)]
pub struct AdmissionConfig {
    /// Cost units the process takes on at once.
    pub cost_units: u64,
    /// Memory the admitted jobs may hold at once.
    #[serde(deserialize_with = "crate::config::bytes::deserialize")]
    pub memory_bytes: u64,
    /// The share of both, from 0 to 0.9, that batch work may not use.
    #[serde(default = "default_reserve")]
    pub interactive_reserve: f64,
    /// How long a job may wait for capacity before it is refused.
    #[serde(
        default = "default_max_wait",
        deserialize_with = "crate::config::duration::deserialize"
    )]
    pub max_wait: Duration,
    /// How many jobs may wait at once; the rest are refused immediately.
    #[serde(default = "default_max_waiters")]
    pub max_waiters: usize,
}

fn default_reserve() -> f64 {
    0.25
}

fn default_max_wait() -> Duration {
    Duration::from_millis(250)
}

fn default_max_waiters() -> usize {
    64
}

impl AdmissionConfig {
    pub fn validate(&self, prefix: &str, v: &mut crate::Validation) {
        v.check(
            self.cost_units > 0,
            &format!("{prefix}.cost_units"),
            "must be positive",
        );
        v.check(
            self.memory_bytes > 0,
            &format!("{prefix}.memory_bytes"),
            "must be positive",
        );
        v.range(
            self.interactive_reserve,
            0.0..=0.9,
            &format!("{prefix}.interactive_reserve"),
        );
        v.check(
            self.max_wait <= Duration::from_secs(30),
            &format!("{prefix}.max_wait"),
            "must be at most 30s",
        );
    }
}

/// What a job is expected to need while it runs.
#[derive(Clone, Copy, Debug, Default, PartialEq, Eq)]
pub struct Cost {
    pub units: u64,
    pub memory_bytes: u64,
}

/// Why a job was not admitted.
#[derive(Clone, Copy, Debug, PartialEq, Eq, thiserror::Error)]
pub enum Rejection {
    #[error("the service is at capacity; retry after {retry_after:?}")]
    Exhausted { retry_after: Duration },
    #[error("the service is shutting down")]
    Draining,
}

#[derive(Clone, Debug, Hash, PartialEq, Eq, EncodeLabelSet)]
struct PriorityLabel {
    priority: Priority,
}

#[derive(Clone, Debug, Hash, PartialEq, Eq, EncodeLabelSet)]
struct OutcomeLabel {
    priority: Priority,
    outcome: Outcome,
}

#[derive(Clone, Copy, Debug, Hash, PartialEq, Eq, EncodeLabelValue)]
enum Outcome {
    Admitted,
    Waited,
    Rejected,
    Draining,
}

#[derive(Debug, Default)]
struct Metrics {
    decisions: Family<OutcomeLabel, Counter>,
    in_flight: Family<PriorityLabel, Gauge>,
    units_in_use: Gauge,
    memory_in_use: Gauge,
    waiting: Gauge,
}

#[derive(Debug, Default)]
struct State {
    units: u64,
    memory: u64,
    in_flight: [u64; 2],
    waiting: usize,
    draining: bool,
}

#[derive(Debug)]
struct Inner {
    config: AdmissionConfig,
    state: Mutex<State>,
    released: Notify,
    held_ewma_micros: AtomicU64,
    metrics: Metrics,
}

/// The process's admission controller. Cheap to clone.
#[derive(Clone, Debug)]
pub struct Admission {
    inner: Arc<Inner>,
}

/// Capacity held by an admitted job, returned when it is dropped.
#[derive(Debug)]
pub struct Permit {
    inner: Arc<Inner>,
    cost: Cost,
    priority: Priority,
    since: Instant,
}

impl Admission {
    /// A controller for `config`, registering its metrics under `admission_`.
    pub fn new(config: AdmissionConfig, registry: &mut Registry) -> Self {
        let metrics = Metrics::default();
        let sub = registry.sub_registry_with_prefix("admission");
        sub.register(
            "decisions",
            "Admission decisions by priority and outcome",
            metrics.decisions.clone(),
        );
        sub.register(
            "in_flight",
            "Admitted jobs running",
            metrics.in_flight.clone(),
        );
        sub.register(
            "cost_units_in_use",
            "Cost units held by admitted jobs",
            metrics.units_in_use.clone(),
        );
        sub.register(
            "memory_bytes_in_use",
            "Memory held by admitted jobs",
            metrics.memory_in_use.clone(),
        );
        sub.register(
            "waiting",
            "Jobs waiting for capacity",
            metrics.waiting.clone(),
        );
        sub.register(
            "cost_units_capacity",
            "Cost units the process takes on",
            gauge(config.cost_units),
        );
        Self {
            inner: Arc::new(Inner {
                config,
                state: Mutex::new(State::default()),
                released: Notify::new(),
                held_ewma_micros: AtomicU64::new(500_000),
                metrics,
            }),
        }
    }

    /// Admits a job, waiting up to the configured bound for capacity.
    pub async fn admit(&self, cost: Cost, priority: Priority) -> Result<Permit, Rejection> {
        let deadline = Instant::now() + self.inner.config.max_wait;
        let mut waited = false;
        loop {
            let released = self.inner.released.notified();
            tokio::pin!(released);
            released.as_mut().enable();

            match self.try_take(cost, priority, waited) {
                Ok(permit) => return Ok(permit),
                Err(Some(rejection)) => return Err(rejection),
                Err(None) => {}
            }

            let now = Instant::now();
            if now >= deadline {
                self.leave_waiting(waited);
                return Err(self.reject(priority));
            }
            if !waited {
                if !self.join_waiting() {
                    return Err(self.reject(priority));
                }
                waited = true;
            }
            if tokio::time::timeout(deadline - now, released)
                .await
                .is_err()
            {
                return match self.try_take(cost, priority, waited) {
                    Ok(permit) => Ok(permit),
                    Err(Some(rejection)) => Err(rejection),
                    Err(None) => {
                        self.leave_waiting(waited);
                        Err(self.reject(priority))
                    }
                };
            }
        }
    }

    /// Stops admitting; jobs already admitted run to the end.
    pub fn drain(&self) {
        self.lock().draining = true;
        self.inner.released.notify_waiters();
    }

    /// Whether interactive work would be refused right now: the signal the
    /// service's readiness follows.
    pub fn saturated(&self) -> bool {
        let state = self.lock();
        state.draining
            || state.units >= self.inner.config.cost_units
            || state.memory >= self.inner.config.memory_bytes
    }

    /// Jobs running now, by class.
    pub fn in_flight(&self, priority: Priority) -> u64 {
        self.lock().in_flight[priority.index()]
    }

    fn lock(&self) -> std::sync::MutexGuard<'_, State> {
        self.inner
            .state
            .lock()
            .unwrap_or_else(PoisonError::into_inner)
    }

    fn limits(&self, priority: Priority) -> (u64, u64) {
        let config = &self.inner.config;
        match priority {
            Priority::Interactive => (config.cost_units, config.memory_bytes),
            Priority::Batch => {
                let share = 1.0 - config.interactive_reserve.clamp(0.0, 0.9);
                (
                    scaled(config.cost_units, share),
                    scaled(config.memory_bytes, share),
                )
            }
        }
    }

    /// Takes capacity if it fits. `Err(None)` means "not now".
    fn try_take(
        &self,
        cost: Cost,
        priority: Priority,
        waited: bool,
    ) -> Result<Permit, Option<Rejection>> {
        let (unit_limit, memory_limit) = self.limits(priority);
        let mut state = self.lock();
        if state.draining {
            if waited {
                state.waiting -= 1;
                self.inner.metrics.waiting.set(gauge_value(state.waiting));
            }
            self.count(priority, Outcome::Draining);
            return Err(Some(Rejection::Draining));
        }

        let alone = state.in_flight[priority.index()] == 0;
        let units = cost.units.min(unit_limit);
        let memory = cost.memory_bytes.min(memory_limit);
        let fits = state.units + units <= unit_limit && state.memory + memory <= memory_limit;
        let oversized = cost.units > unit_limit || cost.memory_bytes > memory_limit;
        if !(fits || (oversized && alone && state.units + units <= self.inner.config.cost_units)) {
            return Err(None);
        }

        state.units += units;
        state.memory += memory;
        state.in_flight[priority.index()] += 1;
        if waited {
            state.waiting -= 1;
        }
        let metrics = &self.inner.metrics;
        metrics.units_in_use.set(gauge_value(state.units));
        metrics.memory_in_use.set(gauge_value(state.memory));
        metrics.waiting.set(gauge_value(state.waiting));
        metrics
            .in_flight
            .get_or_create(&PriorityLabel { priority })
            .set(gauge_value(state.in_flight[priority.index()]));
        drop(state);
        self.count(
            priority,
            if waited {
                Outcome::Waited
            } else {
                Outcome::Admitted
            },
        );

        Ok(Permit {
            inner: Arc::clone(&self.inner),
            cost: Cost {
                units,
                memory_bytes: memory,
            },
            priority,
            since: Instant::now(),
        })
    }

    fn join_waiting(&self) -> bool {
        let mut state = self.lock();
        if state.waiting >= self.inner.config.max_waiters {
            return false;
        }
        state.waiting += 1;
        self.inner.metrics.waiting.set(gauge_value(state.waiting));
        true
    }

    fn leave_waiting(&self, waited: bool) {
        if waited {
            let mut state = self.lock();
            state.waiting = state.waiting.saturating_sub(1);
            self.inner.metrics.waiting.set(gauge_value(state.waiting));
        }
    }

    fn reject(&self, priority: Priority) -> Rejection {
        self.count(priority, Outcome::Rejected);
        let micros = self.inner.held_ewma_micros.load(Ordering::Relaxed);
        let retry_after = Duration::from_micros(micros)
            .clamp(Duration::from_millis(100), Duration::from_secs(30));
        Rejection::Exhausted { retry_after }
    }

    fn count(&self, priority: Priority, outcome: Outcome) {
        self.inner
            .metrics
            .decisions
            .get_or_create(&OutcomeLabel { priority, outcome })
            .inc();
    }
}

impl Drop for Permit {
    fn drop(&mut self) {
        let held = u64::try_from(self.since.elapsed().as_micros()).unwrap_or(u64::MAX);
        let previous = self.inner.held_ewma_micros.load(Ordering::Relaxed);
        let next = previous - previous / 8 + held / 8;
        self.inner.held_ewma_micros.store(next, Ordering::Relaxed);

        {
            let mut state = self
                .inner
                .state
                .lock()
                .unwrap_or_else(PoisonError::into_inner);
            state.units = state.units.saturating_sub(self.cost.units);
            state.memory = state.memory.saturating_sub(self.cost.memory_bytes);
            let index = self.priority.index();
            state.in_flight[index] = state.in_flight[index].saturating_sub(1);
            let metrics = &self.inner.metrics;
            metrics.units_in_use.set(gauge_value(state.units));
            metrics.memory_in_use.set(gauge_value(state.memory));
            metrics
                .in_flight
                .get_or_create(&PriorityLabel {
                    priority: self.priority,
                })
                .set(gauge_value(state.in_flight[index]));
        }
        self.inner.released.notify_waiters();
    }
}

#[allow(
    clippy::cast_possible_truncation,
    clippy::cast_sign_loss,
    clippy::cast_precision_loss
)]
fn scaled(value: u64, share: f64) -> u64 {
    ((value as f64) * share).floor().max(1.0) as u64
}

fn gauge_value<T: TryInto<i64>>(value: T) -> i64 {
    value.try_into().unwrap_or(i64::MAX)
}

fn gauge(value: u64) -> Gauge {
    let g = Gauge::default();
    g.set(gauge_value(value));
    g
}

#[cfg(test)]
mod tests {
    use super::*;

    fn admission(units: u64, reserve: f64, max_wait: Duration) -> Admission {
        Admission::new(
            AdmissionConfig {
                cost_units: units,
                memory_bytes: 1 << 30,
                interactive_reserve: reserve,
                max_wait,
                max_waiters: 4,
            },
            &mut Registry::default(),
        )
    }

    const fn units(n: u64) -> Cost {
        Cost {
            units: n,
            memory_bytes: 1,
        }
    }

    #[tokio::test]
    async fn batch_work_cannot_use_the_interactive_reserve() {
        let admission = admission(4, 0.5, Duration::ZERO);
        let _a = admission
            .admit(units(1), Priority::Batch)
            .await
            .expect("first");
        let _b = admission
            .admit(units(1), Priority::Batch)
            .await
            .expect("second");
        assert!(matches!(
            admission.admit(units(1), Priority::Batch).await,
            Err(Rejection::Exhausted { .. })
        ));
        let _c = admission
            .admit(units(2), Priority::Interactive)
            .await
            .expect("interactive uses the reserve");
        assert!(admission.saturated());
    }

    #[tokio::test]
    async fn a_released_permit_lets_a_waiting_job_in() {
        let admission = admission(1, 0.0, Duration::from_secs(5));
        let first = admission
            .admit(units(1), Priority::Batch)
            .await
            .expect("first");
        let waiter = {
            let admission = admission.clone();
            tokio::spawn(async move { admission.admit(units(1), Priority::Batch).await })
        };
        tokio::time::sleep(Duration::from_millis(20)).await;
        drop(first);
        let second = waiter
            .await
            .expect("joins")
            .expect("admitted after the release");
        drop(second);
        assert_eq!(admission.in_flight(Priority::Batch), 0);
    }

    #[tokio::test]
    async fn an_oversized_job_runs_alone_rather_than_never() {
        let admission = admission(4, 0.0, Duration::ZERO);
        let big = admission
            .admit(units(100), Priority::Batch)
            .await
            .expect("admitted alone");
        assert!(admission.admit(units(1), Priority::Batch).await.is_err());
        drop(big);
        assert!(admission.admit(units(1), Priority::Batch).await.is_ok());
    }

    #[tokio::test]
    async fn a_refusal_carries_a_bounded_retry_after() {
        let admission = admission(1, 0.0, Duration::ZERO);
        let _held = admission
            .admit(units(1), Priority::Batch)
            .await
            .expect("held");
        let Err(Rejection::Exhausted { retry_after }) =
            admission.admit(units(1), Priority::Batch).await
        else {
            panic!("expected a refusal");
        };
        assert!(retry_after >= Duration::from_millis(100));
        assert!(retry_after <= Duration::from_secs(30));
    }

    #[tokio::test]
    async fn draining_refuses_new_work() {
        let admission = admission(4, 0.0, Duration::ZERO);
        admission.drain();
        assert_eq!(
            admission.admit(units(1), Priority::Interactive).await.err(),
            Some(Rejection::Draining)
        );
    }

    #[test]
    fn only_interactive_metadata_claims_the_reserve() {
        assert_eq!(
            Priority::from_metadata(Some("Interactive")),
            Priority::Interactive
        );
        assert_eq!(Priority::from_metadata(Some("urgent")), Priority::Batch);
        assert_eq!(Priority::from_metadata(None), Priority::Batch);
    }
}
