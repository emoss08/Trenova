//! Who printed a job.
//!
//! Any local process can connect to the loopback listener and claim any
//! `requesting-user-name`, so the claim is only a hint. What the service
//! trusts is the Windows print queue: while the IPP Class Driver sends a job
//! to us, that job sits on the Trenova queue under the account that printed
//! it. A document is attributed to a queued job with the same document name,
//! and each queued job can be claimed by one document only, so a process
//! that copies a name it sees on the queue takes that job's place rather
//! than adding a second document to someone's intake. The queue names the
//! user but not the domain; the signed-in sessions supply it, and Windows
//! turns the account into the SID that names the user's inbox.

use std::collections::VecDeque;
use std::io;
use std::sync::{Mutex, PoisonError};
use std::time::{Duration, Instant};

use capture_ipp::Refusal;

/// A job on the Trenova print queue.
#[derive(Clone, Debug, PartialEq, Eq)]
pub struct QueuedJob {
    /// The spooler's job id, unique while the job is queued.
    pub id: u32,
    pub document: String,
    /// The account that printed it, without its domain.
    pub user: String,
}

/// An interactive session's account.
#[derive(Clone, Debug, PartialEq, Eq)]
pub struct SignedIn {
    pub domain: String,
    pub user: String,
}

/// The person a job belongs to.
#[derive(Clone, Debug, PartialEq, Eq)]
pub struct Owner {
    /// `S-1-…`; checked by [`valid_sid`].
    pub sid: String,
    /// `DOMAIN\user`, for the log.
    pub account: String,
}

/// What attribution asks of Windows.
pub trait PrintSystem: Send + Sync {
    /// The jobs on the Trenova queue.
    fn queued_jobs(&self) -> io::Result<Vec<QueuedJob>>;
    /// The accounts signed in to this computer.
    fn signed_in(&self) -> io::Result<Vec<SignedIn>>;
    /// The SID of `DOMAIN\user`, or of `user` alone when the domain is
    /// empty.
    fn sid_of(&self, domain: &str, user: &str) -> io::Result<String>;
}

/// A SID in string form, which is also a directory name and part of an ACL:
/// `S-1-` then numbers separated by dashes, nothing else.
pub fn valid_sid(sid: &str) -> bool {
    let Some(rest) = sid.strip_prefix("S-1-") else {
        return false;
    };
    sid.len() <= 184
        && rest.split('-').all(|part| {
            !part.is_empty() && part.len() <= 10 && part.bytes().all(|b| b.is_ascii_digit())
        })
}

/// Names shorter than this must match exactly; a longer one may be cut
/// short by either side.
const PREFIX_MATCH_CHARS: usize = 32;
/// How long a claimed queue job stays claimed. The spooler reuses job ids
/// only after a restart, and a job is on the queue for seconds.
const CLAIM_TTL: Duration = Duration::from_secs(3600);
const MAX_CLAIMS: usize = 512;

fn same_document(queued: &str, printed: &str) -> bool {
    let (queued, printed) = (queued.trim(), printed.trim());
    if queued == printed {
        return true;
    }
    let (short, long) = if queued.len() < printed.len() {
        (queued, printed)
    } else {
        (printed, queued)
    };
    short.chars().count() >= PREFIX_MATCH_CHARS && long.starts_with(short)
}

/// How long to look for the job on the queue, and how often.
#[derive(Clone, Copy, Debug)]
pub struct Patience {
    pub wait: Duration,
    pub poll: Duration,
}

impl Default for Patience {
    fn default() -> Self {
        Self {
            wait: Duration::from_secs(3),
            poll: Duration::from_millis(250),
        }
    }
}

#[derive(Debug)]
pub struct Attributor<S> {
    system: S,
    patience: Patience,
    claimed: Mutex<VecDeque<(u32, Instant)>>,
}

fn refuse(message: &str) -> Refusal {
    Refusal::NotAttributed(message.to_owned())
}

impl<S: PrintSystem> Attributor<S> {
    pub fn new(system: S, patience: Patience) -> Self {
        Self {
            system,
            patience,
            claimed: Mutex::new(VecDeque::new()),
        }
    }

    /// The owner of the document called `name`, claiming its queue job.
    pub fn attribute(&self, name: &str, claimed_user: Option<&str>) -> Result<Owner, Refusal> {
        let deadline = Instant::now() + self.patience.wait;
        let job = loop {
            if let Some(job) = self.claim(name)? {
                break job;
            }
            if Instant::now() >= deadline {
                return Err(refuse(
                    "No job on the Trenova print queue matches this document.",
                ));
            }
            std::thread::sleep(self.patience.poll);
        };

        if let Some(claimed) = claimed_user
            && !claimed.eq_ignore_ascii_case(&job.user)
            && !claimed
                .rsplit('\\')
                .next()
                .is_some_and(|user| user.eq_ignore_ascii_case(&job.user))
        {
            return Err(refuse(
                "The document claims a different user than the print queue.",
            ));
        }
        self.owner(&job.user)
    }

    /// Finds and claims the queue job for `name`: the oldest unclaimed job
    /// with that document name, as long as every such job is one user's.
    /// Failing a name match, the only unclaimed job on the queue.
    fn claim(&self, name: &str) -> Result<Option<QueuedJob>, Refusal> {
        let queued = self.system.queued_jobs().map_err(|err| {
            tracing::warn!(error = %err, "could not read the Trenova print queue");
            Refusal::Unavailable("The print queue could not be read.".into())
        })?;
        let mut claims = self.claimed.lock().unwrap_or_else(PoisonError::into_inner);
        let now = Instant::now();
        claims.retain(|(_, at)| now.duration_since(*at) < CLAIM_TTL);
        let unclaimed: Vec<&QueuedJob> = queued
            .iter()
            .filter(|job| !claims.iter().any(|(id, _)| *id == job.id))
            .collect();

        let mut named: Vec<&QueuedJob> = unclaimed
            .iter()
            .copied()
            .filter(|job| same_document(&job.document, name))
            .collect();
        if named.is_empty() && unclaimed.len() == 1 {
            named = unclaimed;
        }
        let Some(first) = named.iter().min_by_key(|job| job.id) else {
            return Ok(None);
        };
        if named
            .iter()
            .any(|job| !job.user.eq_ignore_ascii_case(&first.user))
        {
            return Err(refuse(
                "Several people are printing a document with this name; print it again.",
            ));
        }
        let job = (*first).clone();
        claims.push_back((job.id, now));
        while claims.len() > MAX_CLAIMS {
            claims.pop_front();
        }
        Ok(Some(job))
    }

    /// The account for a queue's user name: the signed-in session with that
    /// name, or the name alone when nobody by it is signed in.
    fn owner(&self, user: &str) -> Result<Owner, Refusal> {
        let sessions = self.system.signed_in().unwrap_or_else(|err| {
            tracing::warn!(error = %err, "could not list signed-in sessions");
            Vec::new()
        });
        let mut domains: Vec<&str> = sessions
            .iter()
            .filter(|s| s.user.eq_ignore_ascii_case(user))
            .map(|s| s.domain.as_str())
            .collect();
        domains.sort_unstable_by_key(|d| d.to_ascii_lowercase());
        domains.dedup_by(|a, b| a.eq_ignore_ascii_case(b));
        let domain = match domains.as_slice() {
            [] => "",
            [domain] => domain,
            _ => {
                return Err(refuse(
                    "More than one signed-in account has this user name.",
                ));
            }
        };
        let sid = self.system.sid_of(domain, user).map_err(|err| {
            tracing::warn!(error = %err, user, domain, "could not look up the account");
            refuse("The account that printed could not be found.")
        })?;
        if !valid_sid(&sid) {
            return Err(Refusal::Internal(
                "Windows returned an unexpected SID.".into(),
            ));
        }
        let account = if domain.is_empty() {
            user.to_owned()
        } else {
            format!("{domain}\\{user}")
        };
        Ok(Owner { sid, account })
    }
}

#[cfg(test)]
mod tests {
    use std::sync::Arc;

    use super::*;

    #[derive(Default)]
    struct Fake {
        jobs: Mutex<Vec<QueuedJob>>,
        sessions: Vec<SignedIn>,
    }

    impl PrintSystem for Fake {
        fn queued_jobs(&self) -> io::Result<Vec<QueuedJob>> {
            Ok(self.jobs.lock().expect("lock").clone())
        }

        fn signed_in(&self) -> io::Result<Vec<SignedIn>> {
            Ok(self.sessions.clone())
        }

        fn sid_of(&self, domain: &str, user: &str) -> io::Result<String> {
            match (domain, user) {
                ("CONTOSO", "jdoe") => Ok("S-1-5-21-1-2-3-1001".into()),
                ("", "jdoe") => Ok("S-1-5-21-9-9-9-1001".into()),
                ("", "asmith" | "ASMITH") => Ok("S-1-5-21-1-2-3-1002".into()),
                _ => Err(io::Error::from(io::ErrorKind::NotFound)),
            }
        }
    }

    fn job(id: u32, document: &str, user: &str) -> QueuedJob {
        QueuedJob {
            id,
            document: document.into(),
            user: user.into(),
        }
    }

    fn quick() -> Patience {
        Patience {
            wait: Duration::from_millis(30),
            poll: Duration::from_millis(5),
        }
    }

    fn attributor_over(jobs: Vec<QueuedJob>, sessions: Vec<SignedIn>) -> Attributor<Fake> {
        Attributor::new(
            Fake {
                jobs: Mutex::new(jobs),
                sessions,
            },
            quick(),
        )
    }

    fn contoso_jdoe() -> SignedIn {
        SignedIn {
            domain: "CONTOSO".into(),
            user: "jdoe".into(),
        }
    }

    #[test]
    fn a_job_is_attributed_to_the_signed_in_account_that_queued_it() {
        let attributor = attributor_over(
            vec![
                job(7, "Rate confirmation", "jdoe"),
                job(8, "Invoice", "asmith"),
            ],
            vec![contoso_jdoe()],
        );
        let owner = attributor
            .attribute("Rate confirmation", Some("jdoe"))
            .expect("attributed");
        assert_eq!(owner.sid, "S-1-5-21-1-2-3-1001");
        assert_eq!(owner.account, "CONTOSO\\jdoe");

        let other = attributor.attribute("Invoice", None).expect("attributed");
        assert_eq!(other.account, "asmith", "nobody signed in: the name alone");
    }

    #[test]
    fn each_queued_job_is_claimed_once() {
        let attributor = attributor_over(
            vec![job(7, "Rate confirmation", "jdoe")],
            vec![contoso_jdoe()],
        );
        attributor
            .attribute("Rate confirmation", None)
            .expect("first");
        assert!(matches!(
            attributor.attribute("Rate confirmation", None),
            Err(Refusal::NotAttributed(_))
        ));
    }

    #[test]
    fn a_claim_that_contradicts_the_queue_or_an_ambiguous_name_is_refused() {
        let attributor = attributor_over(vec![job(7, "BOL", "jdoe")], vec![contoso_jdoe()]);
        assert!(matches!(
            attributor.attribute("BOL", Some("asmith")),
            Err(Refusal::NotAttributed(_))
        ));

        let shared = attributor_over(
            vec![job(1, "Report", "jdoe"), job(2, "Report", "asmith")],
            vec![],
        );
        assert!(matches!(
            shared.attribute("Report", None),
            Err(Refusal::NotAttributed(_))
        ));

        let twice = attributor_over(
            vec![job(3, "Report", "jdoe")],
            vec![
                contoso_jdoe(),
                SignedIn {
                    domain: "FABRIKAM".into(),
                    user: "JDOE".into(),
                },
            ],
        );
        assert!(matches!(
            twice.attribute("Report", None),
            Err(Refusal::NotAttributed(_))
        ));
    }

    #[test]
    fn long_names_match_when_cut_short_and_a_lone_job_matches_any_name() {
        let full_name = "Freight bill 20240611 - Contoso Logistics - Chicago to Denver.pdf";
        let attributor = attributor_over(vec![job(4, full_name, "jdoe")], vec![contoso_jdoe()]);
        attributor
            .attribute(&full_name[..40], Some("CONTOSO\\jdoe"))
            .expect("prefix");

        let lone = attributor_over(vec![job(5, "Microsoft Word - Document1", "jdoe")], vec![]);
        assert_eq!(
            lone.attribute("Document1", None).expect("lone").account,
            "jdoe"
        );

        let two = attributor_over(vec![job(5, "A", "jdoe"), job(6, "B", "jdoe")], vec![]);
        assert!(matches!(
            two.attribute("C", None),
            Err(Refusal::NotAttributed(_))
        ));
    }

    #[test]
    fn a_job_that_reaches_the_queue_late_is_waited_for() {
        let attributor = Arc::new(Attributor::new(
            Fake::default(),
            Patience {
                wait: Duration::from_secs(2),
                poll: Duration::from_millis(5),
            },
        ));
        let later = Arc::clone(&attributor);
        let handle = std::thread::spawn(move || later.attribute("Late", None));
        std::thread::sleep(Duration::from_millis(40));
        attributor
            .system
            .jobs
            .lock()
            .expect("lock")
            .push(job(9, "Late", "asmith"));
        assert_eq!(
            handle.join().expect("joins").expect("found").account,
            "asmith"
        );
    }

    #[test]
    fn only_well_formed_sids_are_accepted() {
        assert!(valid_sid("S-1-5-21-3623811015-3361044348-30300820-1013"));
        assert!(valid_sid("S-1-5-18"));
        for bad in [
            "",
            "S-1-",
            "S-1-5-",
            "S-1-5-x",
            "S-1-5-21;(A;;FA;;;WD)",
            "..\\S-1-5",
            "S-1-5--1",
        ] {
            assert!(!valid_sid(bad), "{bad}");
        }
    }
}
