//! How the print service hands a printed job to the person who printed it.
//!
//! The service owns `%ProgramData%\Trenova\Capture\spool`, and inside it one
//! directory per Windows user, named by SID and ACL'd to that user and the
//! service. For each job it writes the PDF, then a small JSON description;
//! each is written under a `.part` name and renamed, and the description is
//! renamed last, so a description on disk means its PDF is complete. The
//! user's agent reads the directory, takes each job into its own encrypted
//! spool and deletes both files. A job printed while the agent is not running
//! waits here until it is.

use std::fs;
use std::io::{self, Write};
use std::path::{Path, PathBuf};
use std::sync::atomic::{AtomicU32, Ordering};
use std::time::{Duration, SystemTime, UNIX_EPOCH};

use serde::{Deserialize, Serialize};

use crate::api::MAX_PRINT_JOB_BYTES;
use crate::manifest::page_checksum;

/// The description format this build writes and reads.
pub const VERSION: u32 = 1;
const PART: &str = "part";
const REJECTED: &str = "rejected";
const DESCRIPTION: &str = "json";
const DOCUMENT: &str = "pdf";
/// The most a description may be; anything larger was not written by the
/// service.
const MAX_DESCRIPTION_BYTES: u64 = 16 << 10;
const MAX_ID_LEN: usize = 64;
const MAX_NAME_CHARS: usize = 255;

#[derive(Debug, thiserror::Error)]
pub enum HandoffError {
    #[error(transparent)]
    Io(#[from] io::Error),
    #[error("the job description {0} is unreadable")]
    Description(String),
    #[error("{0} is not a print job's name")]
    InvalidId(String),
    #[error("the printed document for {0} does not match its description")]
    Mismatch(String),
}

/// A printed job waiting for its owner's agent.
#[derive(Clone, Debug, PartialEq, Eq, Serialize, Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct PrintedJob {
    pub version: u32,
    /// The file stem the job is stored under; also the agent's key for it.
    pub id: String,
    /// The document name the application printed under.
    pub name: String,
    /// Pages, when the service counted them (it does for raster jobs, which
    /// it converts; a PDF is split by the server).
    #[serde(default)]
    pub pages: Option<u32>,
    pub bytes: u64,
    pub sha256: String,
    /// Unix seconds.
    pub received_at: i64,
}

/// A job id is the file stem, so only what is safe in a file name is taken.
fn valid_id(id: &str) -> bool {
    !id.is_empty()
        && id.len() <= MAX_ID_LEN
        && id
            .bytes()
            .all(|b| b.is_ascii_lowercase() || b.is_ascii_digit() || b == b'-')
}

fn unix_seconds(time: SystemTime) -> i64 {
    time.duration_since(UNIX_EPOCH)
        .map_or(0, |d| i64::try_from(d.as_secs()).unwrap_or(i64::MAX))
}

/// A name for the next job: time first, so a listing sorts oldest first.
fn next_id() -> String {
    static COUNTER: AtomicU32 = AtomicU32::new(0);
    let millis = SystemTime::now()
        .duration_since(UNIX_EPOCH)
        .map_or(0, |d| d.as_millis());
    let count = COUNTER.fetch_add(1, Ordering::Relaxed);
    format!("{millis:013}-{:x}-{count:x}", std::process::id())
}

fn write_then_rename(dir: &Path, stem: &str, extension: &str, bytes: &[u8]) -> io::Result<()> {
    let target = dir.join(format!("{stem}.{extension}"));
    let part = dir.join(format!("{stem}.{extension}.{PART}"));
    let mut file = fs::File::create(&part)?;
    file.write_all(bytes)?;
    file.sync_all()?;
    drop(file);
    fs::rename(&part, &target)
}

/// One user's inbox.
#[derive(Clone, Debug)]
pub struct Inbox {
    dir: PathBuf,
}

impl Inbox {
    /// The inbox in `dir`, which the caller has created with the right ACL.
    pub fn new(dir: impl Into<PathBuf>) -> Self {
        Self { dir: dir.into() }
    }

    pub fn dir(&self) -> &Path {
        &self.dir
    }

    fn path(&self, id: &str, extension: &str) -> PathBuf {
        self.dir.join(format!("{id}.{extension}"))
    }

    /// Writes a printed document and its description. The service's half.
    pub fn deliver(
        &self,
        name: &str,
        pages: Option<u32>,
        pdf: &[u8],
    ) -> Result<PrintedJob, HandoffError> {
        let job = PrintedJob {
            version: VERSION,
            id: next_id(),
            name: name.chars().take(MAX_NAME_CHARS).collect(),
            pages,
            bytes: pdf.len() as u64,
            sha256: page_checksum(pdf),
            received_at: unix_seconds(SystemTime::now()),
        };
        let description = serde_json::to_vec(&job).map_err(io::Error::other)?;
        write_then_rename(&self.dir, &job.id, DOCUMENT, pdf)?;
        if let Err(err) = write_then_rename(&self.dir, &job.id, DESCRIPTION, &description) {
            let _ = fs::remove_file(self.path(&job.id, DOCUMENT));
            return Err(err.into());
        }
        Ok(job)
    }

    /// The jobs waiting, oldest first. A description that cannot be read is
    /// reported, not skipped silently, so the agent can set it aside.
    pub fn waiting(&self) -> Result<Vec<Result<PrintedJob, HandoffError>>, HandoffError> {
        let entries = match fs::read_dir(&self.dir) {
            Ok(entries) => entries,
            Err(err) if err.kind() == io::ErrorKind::NotFound => return Ok(Vec::new()),
            Err(err) => return Err(err.into()),
        };
        let mut stems = Vec::new();
        for entry in entries {
            let path = entry?.path();
            if path.extension().and_then(|e| e.to_str()) != Some(DESCRIPTION) {
                continue;
            }
            if let Some(stem) = path.file_stem().and_then(|s| s.to_str())
                && valid_id(stem)
            {
                stems.push(stem.to_owned());
            }
        }
        stems.sort_unstable();
        Ok(stems.into_iter().map(|id| self.describe(&id)).collect())
    }

    fn describe(&self, id: &str) -> Result<PrintedJob, HandoffError> {
        let path = self.path(id, DESCRIPTION);
        let unreadable = || HandoffError::Description(id.to_owned());
        if fs::metadata(&path)?.len() > MAX_DESCRIPTION_BYTES {
            return Err(unreadable());
        }
        let job: PrintedJob =
            serde_json::from_slice(&fs::read(&path)?).map_err(|_| unreadable())?;
        if job.version != VERSION || job.id != id {
            return Err(unreadable());
        }
        Ok(job)
    }

    /// The printed document, checked against its description.
    pub fn read(&self, job: &PrintedJob) -> Result<Vec<u8>, HandoffError> {
        let mismatch = || HandoffError::Mismatch(job.id.clone());
        if !valid_id(&job.id) || job.bytes > MAX_PRINT_JOB_BYTES as u64 {
            return Err(mismatch());
        }
        let path = self.path(&job.id, DOCUMENT);
        if fs::metadata(&path)?.len() != job.bytes {
            return Err(mismatch());
        }
        let pdf = fs::read(&path)?;
        if page_checksum(&pdf) != job.sha256 {
            return Err(mismatch());
        }
        Ok(pdf)
    }

    /// Deletes a job once the agent has it: the description first, so a
    /// crash in between leaves a document without one, which
    /// [`Inbox::sweep`] removes, never a description without its document.
    pub fn remove(&self, id: &str) -> Result<(), HandoffError> {
        if !valid_id(id) {
            return Err(HandoffError::InvalidId(id.to_owned()));
        }
        for extension in [DESCRIPTION, DOCUMENT] {
            match fs::remove_file(self.path(id, extension)) {
                Err(err) if err.kind() != io::ErrorKind::NotFound => return Err(err.into()),
                _ => {}
            }
        }
        Ok(())
    }

    /// Renames a job that cannot be read to `<name>.rejected`, so it stops
    /// being offered but is not deleted before someone has looked at it.
    pub fn set_aside(&self, id: &str) -> Result<(), HandoffError> {
        if !valid_id(id) {
            return Err(HandoffError::InvalidId(id.to_owned()));
        }
        for extension in [DESCRIPTION, DOCUMENT] {
            let from = self.path(id, extension);
            let to = self.dir.join(format!("{id}.{extension}.{REJECTED}"));
            match fs::rename(&from, &to) {
                Err(err) if err.kind() != io::ErrorKind::NotFound => return Err(err.into()),
                _ => {}
            }
        }
        Ok(())
    }

    /// Removes what an interrupted write or removal left behind: `.part`
    /// files, and documents with no description, once they are older than
    /// `age`. Returns how many files it removed.
    pub fn sweep(&self, age: Duration) -> Result<usize, HandoffError> {
        let entries = match fs::read_dir(&self.dir) {
            Ok(entries) => entries,
            Err(err) if err.kind() == io::ErrorKind::NotFound => return Ok(0),
            Err(err) => return Err(err.into()),
        };
        let now = SystemTime::now();
        let mut removed = 0;
        for entry in entries {
            let entry = entry?;
            let path = entry.path();
            let extension = path.extension().and_then(|e| e.to_str());
            let orphan = match extension {
                Some(PART) => true,
                Some(DOCUMENT) => !path.with_extension(DESCRIPTION).exists(),
                _ => false,
            };
            let old = entry
                .metadata()?
                .modified()
                .ok()
                .and_then(|modified| now.duration_since(modified).ok())
                .is_some_and(|elapsed| elapsed >= age);
            if orphan && old {
                match fs::remove_file(&path) {
                    Ok(()) => removed += 1,
                    Err(err) if err.kind() == io::ErrorKind::NotFound => {}
                    Err(err) => return Err(err.into()),
                }
            }
        }
        Ok(removed)
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    fn inbox() -> (tempfile::TempDir, Inbox) {
        let dir = tempfile::tempdir().expect("temp dir");
        let inbox = Inbox::new(dir.path());
        (dir, inbox)
    }

    #[test]
    fn a_delivered_job_is_listed_read_and_removed() {
        let (_dir, inbox) = inbox();
        let first = inbox
            .deliver("Rate confirmation", Some(2), b"%PDF-1.7 one")
            .expect("delivers");
        let second = inbox
            .deliver("BOL", None, b"%PDF-1.7 two")
            .expect("delivers");
        let waiting: Vec<PrintedJob> = inbox
            .waiting()
            .expect("lists")
            .into_iter()
            .map(|job| job.expect("readable"))
            .collect();
        assert_eq!(waiting, [first.clone(), second]);
        assert_eq!(first.pages, Some(2));
        assert_eq!(inbox.read(&first).expect("reads"), b"%PDF-1.7 one");

        inbox.remove(&first.id).expect("removes");
        inbox.remove(&first.id).expect("removing twice is harmless");
        assert_eq!(inbox.waiting().expect("lists").len(), 1);
    }

    #[test]
    fn a_document_that_changed_after_delivery_is_refused() {
        let (dir, inbox) = inbox();
        let job = inbox
            .deliver("Invoice", None, b"%PDF-1.7 a")
            .expect("delivers");
        fs::write(dir.path().join(format!("{}.pdf", job.id)), b"%PDF-1.7 b").expect("writes");
        assert!(matches!(inbox.read(&job), Err(HandoffError::Mismatch(_))));
        let forged = PrintedJob {
            id: "../escape".into(),
            ..job
        };
        assert!(matches!(
            inbox.read(&forged),
            Err(HandoffError::Mismatch(_))
        ));
        assert!(inbox.remove("../escape").is_err());
        assert!(inbox.set_aside("../escape").is_err());

        inbox.set_aside(&job.id).expect("sets aside");
        assert!(inbox.waiting().expect("lists").is_empty());
        assert!(dir.path().join(format!("{}.pdf.rejected", job.id)).exists());
        assert_eq!(
            inbox.sweep(Duration::ZERO).expect("sweeps"),
            0,
            "kept for a person"
        );
    }

    #[test]
    fn unreadable_descriptions_are_reported_and_strays_are_ignored() {
        let (dir, inbox) = inbox();
        fs::write(dir.path().join("0000000000001-1-0.json"), b"{not json").expect("writes");
        fs::write(dir.path().join("Not An Id.json"), b"{}").expect("writes");
        fs::write(dir.path().join("notes.txt"), b"hello").expect("writes");
        let waiting = inbox.waiting().expect("lists");
        assert_eq!(waiting.len(), 1);
        assert!(matches!(waiting[0], Err(HandoffError::Description(_))));
    }

    #[test]
    fn the_sweep_removes_only_old_leftovers() {
        let (dir, inbox) = inbox();
        let job = inbox.deliver("Kept", None, b"%PDF-1.7").expect("delivers");
        fs::write(dir.path().join("0000000000002-1-0.pdf.part"), b"half").expect("writes");
        fs::write(dir.path().join("0000000000003-1-0.pdf"), b"orphan").expect("writes");
        assert_eq!(inbox.sweep(Duration::from_secs(3600)).expect("sweeps"), 0);
        assert_eq!(inbox.sweep(Duration::ZERO).expect("sweeps"), 2);
        assert_eq!(inbox.read(&job).expect("kept"), b"%PDF-1.7");
        assert!(
            Inbox::new(dir.path().join("missing"))
                .waiting()
                .expect("empty")
                .is_empty()
        );
    }
}
