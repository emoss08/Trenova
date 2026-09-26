//! Taking printed jobs from this person's print inbox.
//!
//! The print service leaves each job in an inbox only this Windows user and
//! the service can open. The agent looks every couple of seconds, moves each
//! job into its own encrypted spool as a finished Print batch, and only then
//! removes it from the inbox, so a crash in between sends the job once: the
//! batch key comes from the job's name, and spooling it again is a no-op.
//! A job keeps waiting in the inbox while the agent is not running, and is
//! spooled whether or not anyone is signed in; the uploader sends it once
//! someone is.

use std::sync::Arc;
use std::time::{Duration, Instant};

use capture_client::spool::{Spool, SpoolError};
use capture_protocol::api::{BatchSource, OpenBatchInput, Settings};
use capture_protocol::handoff::{HandoffError, Inbox, PrintedJob};
use tokio::sync::mpsc;
use tokio_util::sync::CancellationToken;

/// How often the inbox is looked at.
pub const POLL: Duration = Duration::from_secs(2);
/// How often leftovers of interrupted writes are cleared, and how old they
/// must be.
const SWEEP_EVERY: Duration = Duration::from_secs(3600);
const SWEEP_AFTER: Duration = Duration::from_secs(86_400);
/// The name the batch's source carries in Trenova.
pub const PRINTER_SOURCE_NAME: &str = "Trenova printer";

/// What happened to a job found in the inbox.
#[derive(Clone, Debug, PartialEq, Eq)]
pub enum Imported {
    /// It is in the spool, waiting to be sent.
    Spooled { name: String, pages: Option<u32> },
    /// It could not be read, and was set aside in the inbox.
    Rejected { name: String, reason: String },
}

/// The spool key for an inbox job: `prt-` and the job's own name, which is
/// unique and made of the same characters a key is.
fn key_for(job: &PrintedJob) -> String {
    format!("prt-{}", job.id)
}

fn batch_input(job: &PrintedJob) -> OpenBatchInput {
    OpenBatchInput {
        client_key: key_for(job),
        source: BatchSource::Print,
        request_id: None,
        profile_id: None,
        source_name: PRINTER_SOURCE_NAME.into(),
        job_name: job.name.clone(),
        settings: Settings::default(),
    }
}

/// The name a person sees for a job.
fn label(job: &PrintedJob) -> String {
    let name = job.name.trim();
    if name.is_empty() {
        "Printed document".into()
    } else {
        name.to_owned()
    }
}

fn set_aside(inbox: &Inbox, id: &str, name: String, reason: String) -> Imported {
    tracing::warn!(id, reason, "set aside a print that could not be read");
    if let Err(err) = inbox.set_aside(id) {
        tracing::error!(id, error = %err, "could not set a print aside");
    }
    Imported::Rejected { name, reason }
}

/// Moves one job into the spool. `None` when it should be tried again later.
fn import_one(inbox: &Inbox, spool: &Spool, job: &PrintedJob) -> Option<Imported> {
    let pdf = match inbox.read(job) {
        Ok(pdf) => pdf,
        Err(HandoffError::Io(err)) => {
            tracing::warn!(id = %job.id, error = %err, "could not read a print yet");
            return None;
        }
        Err(err) => return Some(set_aside(inbox, &job.id, label(job), err.to_string())),
    };
    match spool.create_print(batch_input(job), label(job), &pdf, job.pages) {
        Ok(_) => {}
        Err(SpoolError::Io(err)) => {
            tracing::warn!(id = %job.id, error = %err, "could not spool a print yet");
            return None;
        }
        Err(err) => return Some(set_aside(inbox, &job.id, label(job), err.to_string())),
    }
    if let Err(err) = inbox.remove(&job.id) {
        tracing::warn!(id = %job.id, error = %err, "spooled a print but could not remove it");
    }
    Some(Imported::Spooled {
        name: label(job),
        pages: job.pages,
    })
}

/// Moves every job waiting in the inbox into the spool, oldest first.
pub fn import(inbox: &Inbox, spool: &Spool) -> Vec<Imported> {
    let waiting = match inbox.waiting() {
        Ok(waiting) => waiting,
        Err(err) => {
            tracing::warn!(error = %err, "could not read the print inbox");
            return Vec::new();
        }
    };
    let mut imported = Vec::new();
    for job in waiting {
        let outcome = match job {
            Ok(job) => import_one(inbox, spool, &job),
            Err(HandoffError::Description(id)) => Some(set_aside(
                inbox,
                &id,
                "Printed document".into(),
                "its description is unreadable".into(),
            )),
            Err(err) => {
                tracing::warn!(error = %err, "could not read a print yet");
                None
            }
        };
        imported.extend(outcome);
    }
    imported
}

/// Watches the inbox until `cancel`, reporting each job as it is taken.
pub async fn watch(
    inbox: Inbox,
    spool: Arc<Spool>,
    report: mpsc::Sender<Imported>,
    cancel: CancellationToken,
) {
    let inbox = Arc::new(inbox);
    let mut swept = Instant::now()
        .checked_sub(SWEEP_EVERY)
        .unwrap_or_else(Instant::now);
    loop {
        let (from, into) = (Arc::clone(&inbox), Arc::clone(&spool));
        let sweep = swept.elapsed() >= SWEEP_EVERY;
        if sweep {
            swept = Instant::now();
        }
        let imported = tokio::task::spawn_blocking(move || {
            if sweep && let Err(err) = from.sweep(SWEEP_AFTER) {
                tracing::warn!(error = %err, "could not tidy the print inbox");
            }
            import(&from, &into)
        })
        .await
        .unwrap_or_default();
        for outcome in imported {
            if report.send(outcome).await.is_err() {
                return;
            }
        }
        tokio::select! {
            () = cancel.cancelled() => return,
            () = tokio::time::sleep(POLL) => {}
        }
    }
}

#[cfg(test)]
mod tests {
    use std::io;

    use capture_client::Protector;

    use super::*;

    #[derive(Debug)]
    struct Plain;

    impl Protector for Plain {
        fn protect(&self, plain: &[u8]) -> io::Result<Vec<u8>> {
            Ok(plain.to_vec())
        }
        fn unprotect(&self, sealed: &[u8]) -> io::Result<Vec<u8>> {
            Ok(sealed.to_vec())
        }
    }

    fn setup() -> (tempfile::TempDir, Inbox, Spool) {
        let dir = tempfile::tempdir().expect("dir");
        std::fs::create_dir_all(dir.path().join("inbox")).expect("inbox");
        let inbox = Inbox::new(dir.path().join("inbox"));
        let spool = Spool::open(dir.path().join("spool"), Arc::new(Plain)).expect("spool");
        (dir, inbox, spool)
    }

    #[test]
    fn printed_jobs_become_finished_print_batches_and_leave_the_inbox() {
        let (_dir, inbox, spool) = setup();
        let raster = inbox
            .deliver("Rate confirmation", Some(2), b"%PDF-1.7 raster")
            .expect("delivers");
        inbox
            .deliver("  ", None, b"%PDF-1.7 pdf")
            .expect("delivers");

        let imported = import(&inbox, &spool);
        assert_eq!(
            imported,
            [
                Imported::Spooled {
                    name: "Rate confirmation".into(),
                    pages: Some(2)
                },
                Imported::Spooled {
                    name: "Printed document".into(),
                    pages: None
                },
            ]
        );
        assert!(inbox.waiting().expect("lists").is_empty());

        let batches = spool.pending().expect("pending");
        assert_eq!(batches.len(), 2);
        let first = batches
            .iter()
            .find(|b| b.key() == format!("prt-{}", raster.id))
            .expect("keyed by the job");
        assert!(first.complete);
        assert_eq!(first.input.source, BatchSource::Print);
        assert_eq!(first.input.request_id, None);
        assert_eq!(first.input.job_name, "Rate confirmation");
        let document = first.document.as_ref().expect("a document");
        assert_eq!(
            spool.read_document(first.key(), document).expect("reads"),
            b"%PDF-1.7 raster"
        );
        assert_eq!(first.pages_waiting(), 2);
    }

    #[test]
    fn a_job_taken_twice_is_spooled_once() {
        let (dir, inbox, spool) = setup();
        let job = inbox
            .deliver("BOL", None, b"%PDF-1.7 bol")
            .expect("delivers");
        let pdf =
            std::fs::read(dir.path().join("inbox").join(format!("{}.pdf", job.id))).expect("pdf");
        let json =
            std::fs::read(dir.path().join("inbox").join(format!("{}.json", job.id))).expect("json");
        import(&inbox, &spool);
        std::fs::write(
            dir.path().join("inbox").join(format!("{}.pdf", job.id)),
            pdf,
        )
        .expect("restore");
        std::fs::write(
            dir.path().join("inbox").join(format!("{}.json", job.id)),
            json,
        )
        .expect("restore");
        import(&inbox, &spool);
        assert_eq!(spool.pending().expect("pending").len(), 1);
        assert!(inbox.waiting().expect("lists").is_empty());
    }

    #[test]
    fn an_unreadable_job_is_set_aside_not_lost_or_retried_forever() {
        let (dir, inbox, spool) = setup();
        let job = inbox
            .deliver("Invoice", None, b"%PDF-1.7 a")
            .expect("delivers");
        std::fs::write(
            dir.path().join("inbox").join(format!("{}.pdf", job.id)),
            b"%PDF-1.7 b",
        )
        .expect("tamper");
        std::fs::write(
            dir.path().join("inbox").join("0000000000009-1-0.json"),
            b"{",
        )
        .expect("junk");

        let imported = import(&inbox, &spool);
        assert_eq!(imported.len(), 2);
        assert!(
            imported
                .iter()
                .all(|i| matches!(i, Imported::Rejected { .. }))
        );
        assert!(spool.pending().expect("pending").is_empty());
        assert!(import(&inbox, &spool).is_empty(), "not offered again");
        assert!(
            dir.path()
                .join("inbox")
                .join(format!("{}.pdf.rejected", job.id))
                .exists()
        );
    }
}
