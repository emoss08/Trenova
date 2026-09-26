//! What happens to a printed document: whose it is, what it becomes, where
//! it goes.

use std::io;
use std::path::PathBuf;

use capture_imaging::{ConvertLimits, PwgError, pwg_to_pdf};
use capture_ipp::{DocumentFormat, IncomingJob, JobHandler, Refusal};
use capture_protocol::api::{MAX_BATCH_PAGES, MAX_PRINT_JOB_BYTES};
use capture_protocol::handoff::Inbox;

use crate::attribution::{Attributor, Owner, PrintSystem, valid_sid};

/// The JPEG quality a printed colour or gray page is stored at. Printed
/// pages are mostly text on white, which JPEG at this quality keeps sharp.
pub const PRINT_JPEG_QUALITY: u8 = 85;

/// Each person's inbox.
pub trait Inboxes: Send + Sync {
    /// The inbox for `owner`, created with its ACL if it is not there yet.
    fn inbox_for(&self, owner: &Owner) -> io::Result<Inbox>;
}

/// Inboxes as plain directories under a root, for tests and for the
/// development console run. The Windows service uses ACL'd directories.
#[derive(Clone, Debug)]
pub struct PlainInboxes {
    pub root: PathBuf,
}

impl Inboxes for PlainInboxes {
    fn inbox_for(&self, owner: &Owner) -> io::Result<Inbox> {
        if !valid_sid(&owner.sid) {
            return Err(io::Error::from(io::ErrorKind::InvalidInput));
        }
        let dir = self.root.join(&owner.sid);
        std::fs::create_dir_all(&dir)?;
        Ok(Inbox::new(dir))
    }
}

#[derive(Debug)]
pub struct PrintHandler<S, I> {
    attributor: Attributor<S>,
    inboxes: I,
}

impl<S: PrintSystem, I: Inboxes> PrintHandler<S, I> {
    pub fn new(attributor: Attributor<S>, inboxes: I) -> Self {
        Self {
            attributor,
            inboxes,
        }
    }
}

/// A printed document as the PDF the server takes, and its page count when
/// the conversion counted it.
fn to_pdf(job: &IncomingJob) -> Result<(Vec<u8>, Option<u32>), Refusal> {
    match job.format {
        DocumentFormat::Pdf => {
            if !job.data.starts_with(b"%PDF-") {
                return Err(Refusal::BadDocument("The document is not a PDF.".into()));
            }
            if job.data.len() > MAX_PRINT_JOB_BYTES {
                return Err(Refusal::TooLarge);
            }
            Ok((job.data.clone(), None))
        }
        DocumentFormat::PwgRaster => {
            let limits = ConvertLimits {
                max_pages: MAX_BATCH_PAGES,
                max_bytes: MAX_PRINT_JOB_BYTES,
                jpeg_quality: PRINT_JPEG_QUALITY,
            };
            match pwg_to_pdf(&job.data, limits) {
                Ok(document) => Ok((document.pdf, Some(document.pages))),
                Err(PwgError::TooManyPages { .. } | PwgError::TooLarge { .. }) => {
                    Err(Refusal::TooLarge)
                }
                Err(err) => Err(Refusal::BadDocument(err.to_string())),
            }
        }
    }
}

impl<S: PrintSystem, I: Inboxes> JobHandler for PrintHandler<S, I> {
    fn accept(&self, job: IncomingJob) -> Result<(), Refusal> {
        let owner = self
            .attributor
            .attribute(&job.name, job.requesting_user.as_deref())
            .inspect_err(|refusal| {
                tracing::warn!(
                    job = job.job_id,
                    ?refusal,
                    "refused a print job it could not attribute"
                );
            })?;
        let (pdf, pages) = to_pdf(&job).inspect_err(|refusal| {
            tracing::warn!(job = job.job_id, sid = %owner.sid, ?refusal, "refused a printed document");
        })?;
        let inbox = self.inboxes.inbox_for(&owner).map_err(|err| {
            tracing::error!(sid = %owner.sid, error = %err, "could not open the inbox");
            Refusal::Internal("The print could not be stored.".into())
        })?;
        let delivered = inbox.deliver(&job.name, pages, &pdf).map_err(|err| {
            tracing::error!(sid = %owner.sid, error = %err, "could not store a print");
            Refusal::Internal("The print could not be stored.".into())
        })?;
        tracing::info!(
            job = job.job_id,
            id = %delivered.id,
            account = %owner.account,
            sid = %owner.sid,
            bytes = delivered.bytes,
            pages = ?delivered.pages,
            "handed a print to its owner"
        );
        Ok(())
    }
}
