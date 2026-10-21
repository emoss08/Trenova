//! Stopping: SIGTERM or SIGINT starts a drain, in-flight work gets a grace
//! period to finish, and whatever is left after it is abandoned.

use std::future::Future;
use std::time::Duration;

use tokio::sync::watch;

/// A shutdown that can be triggered once and awaited from anywhere.
#[derive(Clone, Debug)]
pub struct Shutdown {
    tx: watch::Sender<bool>,
}

impl Default for Shutdown {
    fn default() -> Self {
        Self::new()
    }
}

impl Shutdown {
    pub fn new() -> Self {
        let (tx, _) = watch::channel(false);
        Self { tx }
    }

    /// Triggers the shutdown when the process receives SIGTERM or SIGINT.
    pub fn trigger_on_signals(&self) {
        let shutdown = self.clone();
        tokio::spawn(async move {
            wait_for_signal().await;
            tracing::info!("shutdown signal received; draining");
            shutdown.trigger();
        });
    }

    pub fn trigger(&self) {
        self.tx.send_replace(true);
    }

    pub fn is_triggered(&self) -> bool {
        *self.tx.borrow()
    }

    /// Resolves once the shutdown is triggered.
    pub fn triggered(&self) -> impl Future<Output = ()> + Send + 'static {
        let mut rx = self.tx.subscribe();
        async move {
            let _ = rx.wait_for(|stopped| *stopped).await;
        }
    }
}

/// Runs `work` until it finishes, or until `grace` has passed after the
/// shutdown was triggered. Returns whether it finished.
pub async fn drain<F: Future<Output = ()>>(shutdown: &Shutdown, grace: Duration, work: F) -> bool {
    let triggered = shutdown.triggered();
    tokio::pin!(work);
    tokio::select! {
        () = &mut work => return true,
        () = triggered => {}
    }
    if tokio::time::timeout(grace, &mut work).await.is_ok() {
        true
    } else {
        tracing::warn!(
            grace_seconds = grace.as_secs(),
            "work was still running when the grace period ended; abandoning it"
        );
        false
    }
}

#[cfg(unix)]
async fn wait_for_signal() {
    use tokio::signal::unix::{SignalKind, signal};

    let terminate = signal(SignalKind::terminate());
    let interrupt = signal(SignalKind::interrupt());
    match (terminate, interrupt) {
        (Ok(mut terminate), Ok(mut interrupt)) => {
            tokio::select! {
                _ = terminate.recv() => {}
                _ = interrupt.recv() => {}
            }
        }
        (Err(err), _) | (_, Err(err)) => {
            tracing::error!(error = %err, "cannot listen for shutdown signals; using ctrl-c only");
            let _ = tokio::signal::ctrl_c().await;
        }
    }
}

#[cfg(not(unix))]
async fn wait_for_signal() {
    let _ = tokio::signal::ctrl_c().await;
}

#[cfg(test)]
mod tests {
    use super::*;

    #[tokio::test]
    async fn work_that_finishes_inside_the_grace_period_is_waited_for() {
        let shutdown = Shutdown::new();
        shutdown.trigger();
        let finished = drain(&shutdown, Duration::from_secs(5), async {
            tokio::time::sleep(Duration::from_millis(10)).await;
        })
        .await;
        assert!(finished);
    }

    #[tokio::test(start_paused = true)]
    async fn work_still_running_after_the_grace_period_is_abandoned() {
        let shutdown = Shutdown::new();
        let trigger = shutdown.clone();
        tokio::spawn(async move {
            tokio::time::sleep(Duration::from_millis(5)).await;
            trigger.trigger();
        });
        let finished = drain(
            &shutdown,
            Duration::from_secs(1),
            std::future::pending::<()>(),
        )
        .await;
        assert!(!finished);
        assert!(shutdown.is_triggered());
    }
}
