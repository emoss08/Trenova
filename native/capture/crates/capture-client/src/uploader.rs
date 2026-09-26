//! Moving spooled batches to the server.
//!
//! Each batch is opened (idempotent on its client key), its pages sent in
//! order (idempotent on their sequence), and, once acquisition has ended and
//! every page is there, sealed with the manifest digest. A printed job is
//! sent whole instead, and the server splits and seals it; sending it again
//! returns the batch as it is. Every step is
//! recorded in the spool as it succeeds, so a batch interrupted anywhere
//! resumes where it stopped, and a retry never duplicates anything.
//!
//! What happens on failure depends on who can fix it:
//!
//! - the network or the server (5xx, 429): wait with backoff and try again;
//! - the person or the organization (signed out, outdated, capture off, a
//!   permission removed): stop everything and wait to be woken;
//! - nobody (the server refuses the batch itself): set it aside, keep its
//!   pages on disk, and say so.

use std::sync::Arc;
use std::time::Duration;

use bytes::Bytes;
use capture_protocol::api::{BatchSource, BatchStatus, CaptureBatch, Id, SealBatchInput};
use capture_protocol::manifest::manifest_digest;
use tokio::sync::{Notify, mpsc};
use tokio_util::sync::CancellationToken;

use crate::api::Api;
use crate::backoff::Backoff;
use crate::error::ApiError;
use crate::spool::{Spool, SpoolError, SpoolSummary, SpooledBatch, SpooledDocument};

/// What the uploader tells the tray.
#[derive(Debug)]
pub enum UploadEvent {
    /// How much is still waiting.
    Progress(SpoolSummary),
    /// A batch reached the server in full and was sealed.
    Sent {
        batch_id: Id,
        source: BatchSource,
        pages: u32,
        label: String,
        /// Whether it was for a record the person chose, rather than intake.
        requested: bool,
    },
    /// The server refused a batch; its pages are kept aside.
    Refused {
        source: BatchSource,
        label: String,
        reason: String,
    },
    /// Uploads are stopped until something changes.
    Blocked(ApiError),
    /// A failure worth retrying; the next attempt is in `retry_in`.
    Waiting { reason: String, retry_in: Duration },
}

enum Step {
    Retry(String, Option<Duration>),
    Blocked(ApiError),
    Refused(String),
}

impl From<ApiError> for Step {
    fn from(err: ApiError) -> Self {
        if err.is_retryable() {
            let wait = err.retry_after();
            return Self::Retry(err.to_string(), wait);
        }
        if err.blocks_everything() {
            return Self::Blocked(err);
        }
        Self::Refused(err.to_string())
    }
}

impl From<SpoolError> for Step {
    fn from(err: SpoolError) -> Self {
        match err {
            SpoolError::Io(_) => Self::Retry(err.to_string(), None),
            SpoolError::Missing(_)
            | SpoolError::Full
            | SpoolError::PageTooLarge(_)
            | SpoolError::DocumentTooLarge(_)
            | SpoolError::Complete
            | SpoolError::Corrupt { .. }
            | SpoolError::CorruptDocument
            | SpoolError::Manifest(_) => Self::Refused(err.to_string()),
        }
    }
}

pub struct Uploader {
    api: Arc<Api>,
    spool: Arc<Spool>,
    wake: Arc<Notify>,
    events: mpsc::Sender<UploadEvent>,
}

impl std::fmt::Debug for Uploader {
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        f.debug_struct("Uploader").finish_non_exhaustive()
    }
}

/// Runs a spool operation off the async threads; the spool is plain file I/O.
async fn blocking<T, F>(spool: &Arc<Spool>, work: F) -> Result<T, SpoolError>
where
    T: Send + 'static,
    F: FnOnce(&Spool) -> Result<T, SpoolError> + Send + 'static,
{
    let spool = Arc::clone(spool);
    tokio::task::spawn_blocking(move || work(&spool))
        .await
        .map_err(|e| SpoolError::Io(std::io::Error::other(e.to_string())))?
}

impl Uploader {
    pub fn new(api: Arc<Api>, spool: Arc<Spool>, events: mpsc::Sender<UploadEvent>) -> Self {
        Self {
            api,
            spool,
            wake: Arc::new(Notify::new()),
            events,
        }
    }

    /// Wakes the uploader: a page was spooled, the person signed in, or the
    /// network came back.
    pub fn waker(&self) -> Arc<Notify> {
        Arc::clone(&self.wake)
    }

    pub async fn run(self, cancel: CancellationToken) {
        let mut backoff = Backoff::new(Duration::from_secs(2), Duration::from_secs(300));
        loop {
            let pause = self.pass().await;
            if let Ok(summary) = blocking(&self.spool, Spool::summary).await {
                let _ = self.events.send(UploadEvent::Progress(summary)).await;
            }

            let wait = match pause {
                None => {
                    backoff.reset();
                    None
                }
                Some(Step::Blocked(err)) => {
                    let _ = self.events.send(UploadEvent::Blocked(err)).await;
                    None
                }
                Some(Step::Retry(reason, after)) => {
                    let retry_in = after.unwrap_or_else(|| backoff.next_delay());
                    let _ = self
                        .events
                        .send(UploadEvent::Waiting { reason, retry_in })
                        .await;
                    Some(retry_in)
                }
                Some(Step::Refused(_)) => None,
            };

            tokio::select! {
                () = cancel.cancelled() => return,
                () = self.wake.notified() => {}
                () = async {
                    match wait {
                        Some(wait) => tokio::time::sleep(wait).await,
                        None => std::future::pending().await,
                    }
                } => {}
            }
        }
    }

    /// One pass over the spool. Returns why it stopped early, if it did.
    async fn pass(&self) -> Option<Step> {
        let batches = match blocking(&self.spool, Spool::pending).await {
            Ok(batches) => batches,
            Err(err) => return Some(Step::Retry(err.to_string(), None)),
        };
        for batch in batches {
            match self.send(&batch).await {
                Ok(()) => {}
                Err(Step::Refused(reason)) => {
                    let key = batch.key().to_owned();
                    let why = reason.clone();
                    if let Err(err) = blocking(&self.spool, move |s| s.fail(&key, &why)).await {
                        tracing::error!(error = %err, "could not set a refused batch aside");
                        return Some(Step::Retry(err.to_string(), None));
                    }
                    let _ = self
                        .events
                        .send(UploadEvent::Refused {
                            source: batch.input.source,
                            label: batch.label.clone(),
                            reason,
                        })
                        .await;
                }
                Err(stop) => return Some(stop),
            }
        }
        None
    }

    /// Takes one batch as far as it can go.
    async fn send(&self, batch: &SpooledBatch) -> Result<(), Step> {
        if let Some(document) = &batch.document {
            return self.send_document(batch, document).await;
        }
        let key = batch.key().to_owned();
        let (batch_id, requested) = match &batch.batch_id {
            Some(id) => (id.clone(), batch.input.request_id.is_some()),
            None => self.open(batch).await?,
        };

        for page in batch.pages.iter().filter(|p| !p.uploaded) {
            let (read_key, read_page) = (key.clone(), page.clone());
            let pdf = blocking(&self.spool, move |s| s.read_page(&read_key, &read_page)).await?;
            let stored = self
                .api
                .put_page(&batch_id, page.sequence, Bytes::from(pdf), &page.markers())
                .await?;
            if stored.checksum_sha256 != page.checksum {
                return Err(Step::Refused(format!(
                    "the server holds a different page {}",
                    page.sequence
                )));
            }
            let (mark_key, sequence) = (key.clone(), page.sequence);
            blocking(&self.spool, move |s| s.mark_uploaded(&mark_key, sequence)).await?;
        }

        if !batch.complete {
            return Ok(());
        }
        let digest = manifest_digest(batch.pages.iter().map(|p| p.checksum.as_str()));
        let seal = SealBatchInput {
            page_count: u32::try_from(batch.pages.len()).unwrap_or(u32::MAX),
            manifest_digest: digest,
        };
        self.api.seal_batch(&batch_id, &seal).await?;
        let remove_key = key.clone();
        blocking(&self.spool, move |s| s.remove(&remove_key)).await?;
        let _ = self
            .events
            .send(UploadEvent::Sent {
                batch_id,
                source: batch.input.source,
                pages: seal.page_count,
                label: batch.label.clone(),
                requested,
            })
            .await;
        Ok(())
    }

    /// Sends a printed job whole. Where it went is the server's to say: a
    /// print with no request named goes to the destination the person armed,
    /// if there is one.
    async fn send_document(
        &self,
        batch: &SpooledBatch,
        document: &SpooledDocument,
    ) -> Result<(), Step> {
        let key = batch.key().to_owned();
        let batch_id = match &batch.batch_id {
            Some(id) => id.clone(),
            None => self.open(batch).await?.0,
        };
        let (read_key, read_document) = (key.clone(), document.clone());
        let pdf = blocking(&self.spool, move |s| {
            s.read_document(&read_key, &read_document)
        })
        .await?;
        let stored = self.api.put_print_job(&batch_id, Bytes::from(pdf)).await?;
        refused_print(&stored)?;
        blocking(&self.spool, move |s| s.remove(&key)).await?;
        let _ = self
            .events
            .send(UploadEvent::Sent {
                batch_id,
                source: batch.input.source,
                pages: stored.received_page_count,
                label: batch.label.clone(),
                requested: stored.request_id.is_some(),
            })
            .await;
        Ok(())
    }

    /// Opens the batch on the server. If the request it was for is no longer
    /// open (the person cancelled it, or it expired while the computer was
    /// offline), the pages go to intake rather than nowhere. Returns the
    /// server's ID and whether the batch is still for its request.
    async fn open(&self, batch: &SpooledBatch) -> Result<(Id, bool), Step> {
        let key = batch.key().to_owned();
        let mut requested = batch.input.request_id.is_some();
        let opened = match self.api.open_batch(&batch.input).await {
            Ok(opened) => opened,
            Err(ApiError::Conflict(_) | ApiError::NotFound(_) | ApiError::Invalid(_))
                if batch.input.request_id.is_some() =>
            {
                let detach_key = key.clone();
                blocking(&self.spool, move |s| s.detach_request(&detach_key)).await?;
                let mut input = batch.input.clone();
                input.request_id = None;
                requested = false;
                self.api.open_batch(&input).await?
            }
            Err(err) => return Err(err.into()),
        };
        let id = opened.id.clone();
        blocking(&self.spool, move |s| s.set_batch_id(&key, opened.id)).await?;
        Ok((id, requested))
    }
}

/// A print batch the server ended without taking the document.
fn refused_print(stored: &CaptureBatch) -> Result<(), Step> {
    match stored.status {
        BatchStatus::Discarded | BatchStatus::Expired | BatchStatus::Failed => {
            let reason = if stored.failure_message.is_empty() {
                format!("the server ended the batch ({:?})", stored.status)
            } else {
                stored.failure_message.clone()
            };
            Err(Step::Refused(reason))
        }
        _ => Ok(()),
    }
}
