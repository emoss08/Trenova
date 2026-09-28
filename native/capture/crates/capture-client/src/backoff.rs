//! Exponential backoff with full jitter.

use std::time::Duration;

#[derive(Clone, Debug)]
pub struct Backoff {
    base: Duration,
    cap: Duration,
    attempt: u32,
}

impl Backoff {
    pub fn new(base: Duration, cap: Duration) -> Self {
        Self {
            base,
            cap,
            attempt: 0,
        }
    }

    /// The ceiling of the next wait, before jitter: `base · 2ⁿ`, at most
    /// `cap`.
    pub fn ceiling(&self) -> Duration {
        let factor = 1u32.checked_shl(self.attempt.min(20)).unwrap_or(u32::MAX);
        self.base.saturating_mul(factor).min(self.cap)
    }

    /// The next wait: uniform in `[ceiling/2, ceiling]`, so a fleet of
    /// devices knocked offline together does not return together, and none
    /// retries sooner than half the ceiling.
    pub fn next_delay(&mut self) -> Duration {
        let ceiling = self.ceiling();
        self.attempt = self.attempt.saturating_add(1);
        let ceiling_ms = u64::try_from(ceiling.as_millis()).unwrap_or(u64::MAX);
        let half = ceiling_ms / 2;
        Duration::from_millis(half + fastrand::u64(0..=ceiling_ms - half))
    }

    /// Starts over after a success.
    pub fn reset(&mut self) {
        self.attempt = 0;
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn delays_double_up_to_the_cap_and_stay_within_their_band() {
        let mut backoff = Backoff::new(Duration::from_secs(1), Duration::from_secs(30));
        for ceiling in [1, 2, 4, 8, 16, 30, 30] {
            let ceiling = Duration::from_secs(ceiling);
            assert_eq!(backoff.ceiling(), ceiling);
            let delay = backoff.next_delay();
            assert!(
                delay >= ceiling / 2 && delay <= ceiling,
                "{delay:?} outside {ceiling:?}"
            );
        }
        backoff.reset();
        assert_eq!(backoff.ceiling(), Duration::from_secs(1));
    }

    #[test]
    fn many_attempts_do_not_overflow() {
        let mut backoff = Backoff::new(Duration::from_secs(1), Duration::from_secs(300));
        for _ in 0..100 {
            backoff.next_delay();
        }
        assert_eq!(backoff.ceiling(), Duration::from_secs(300));
    }
}
