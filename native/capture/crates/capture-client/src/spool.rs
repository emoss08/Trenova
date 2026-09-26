//! Pages on disk until the server has them.
//!
//! Every page is written here, encrypted, before any upload is tried, so an
//! unplugged cable, a sleeping laptop or a crash never loses a scan. Each
//! batch is a directory holding a manifest and one file per page:
//!
//! ```text
//! <root>/batches/<client key>/manifest.json
//! <root>/batches/<client key>/page-0001.bin
//! <root>/failed/<client key>/...           (the server refused it)
//! ```
//!
//! The manifest says which pages the server already has and what the batch
//! is called there, so an upload interrupted anywhere resumes exactly where
//! it stopped. It carries no page content and no credential. Every file is
//! written to a temporary name, flushed and renamed over the old one, so a
//! crash leaves either the old version or the new one, never half of one.

use std::fs;
use std::io::{self, Write};
use std::path::{Path, PathBuf};
use std::sync::{Arc, Mutex, PoisonError};
use std::time::{SystemTime, UNIX_EPOCH};

use capture_protocol::api::{Id, MAX_BATCH_PAGES, MAX_PAGE_BYTES, OpenBatchInput, Settings};
use capture_protocol::page_checksum;
use serde::{Deserialize, Serialize};

use crate::api::PageMarkers;

const MANIFEST: &str = "manifest.json";
const FAILURE: &str = "failure.txt";
const MANIFEST_VERSION: u32 = 1;

/// Encrypts page files at rest. On Windows this is DPAPI, scoped to the
/// signed-in user, so another account on the machine cannot read them.
pub trait Protector: Send + Sync {
    fn protect(&self, plain: &[u8]) -> io::Result<Vec<u8>>;
    fn unprotect(&self, sealed: &[u8]) -> io::Result<Vec<u8>>;
}

/// One page as the manifest records it.
#[derive(Clone, Debug, PartialEq, Eq, Serialize, Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct SpooledPage {
    pub sequence: u32,
    /// SHA-256 of the PDF, as the server will compute it.
    pub checksum: String,
    pub byte_size: u64,
    pub dpi: u32,
    #[serde(default)]
    pub patch_code: Option<String>,
    #[serde(default)]
    pub barcodes: Vec<String>,
    pub uploaded: bool,
}

impl SpooledPage {
    pub fn markers(&self) -> PageMarkers {
        PageMarkers {
            dpi: self.dpi,
            patch_code: self.patch_code.clone(),
            barcodes: self.barcodes.clone(),
        }
    }
}

/// A batch waiting to reach the server.
#[derive(Clone, Debug, PartialEq, Eq, Serialize, Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct SpooledBatch {
    pub version: u32,
    pub created_at: i64,
    /// How the tray names it: the scanner, or the printing application.
    pub label: String,
    pub input: OpenBatchInput,
    /// The server's ID, once opened there.
    #[serde(default)]
    pub batch_id: Option<Id>,
    pub pages: Vec<SpooledPage>,
    /// Acquisition has ended; the batch can be sealed once uploaded.
    pub complete: bool,
}

impl SpooledBatch {
    pub fn key(&self) -> &str {
        &self.input.client_key
    }

    pub fn pages_waiting(&self) -> u32 {
        u32::try_from(self.pages.iter().filter(|p| !p.uploaded).count()).unwrap_or(u32::MAX)
    }
}

/// What the tray shows about the spool.
#[derive(Clone, Copy, Debug, Default, PartialEq, Eq)]
pub struct SpoolSummary {
    pub batches: u32,
    pub pages_waiting: u32,
    pub failed: u32,
}

#[derive(Debug, thiserror::Error)]
pub enum SpoolError {
    #[error("the spool: {0}")]
    Io(#[from] io::Error),
    #[error("no spooled batch {0:?}")]
    Missing(String),
    #[error("a batch holds at most {MAX_BATCH_PAGES} pages")]
    Full,
    #[error("a page of {0} bytes is over the server's limit")]
    PageTooLarge(usize),
    #[error("the batch has already been finished")]
    Complete,
    #[error("page {sequence} on disk does not match what was scanned")]
    Corrupt { sequence: u32 },
    #[error("the manifest is unreadable: {0}")]
    Manifest(String),
}

pub struct Spool {
    root: PathBuf,
    protector: Arc<dyn Protector>,
    /// Serialises manifest read-modify-writes between the scan and upload
    /// threads.
    lock: Mutex<()>,
}

impl std::fmt::Debug for Spool {
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        f.debug_struct("Spool")
            .field("root", &self.root)
            .finish_non_exhaustive()
    }
}

fn now_unix_millis() -> i64 {
    SystemTime::now()
        .duration_since(UNIX_EPOCH)
        .map_or(0, |d| i64::try_from(d.as_millis()).unwrap_or(i64::MAX))
}

/// Only keys this module made are accepted as directory names.
fn valid_key(key: &str) -> bool {
    !key.is_empty()
        && key.len() <= capture_protocol::api::MAX_CLIENT_KEY_LENGTH
        && key
            .bytes()
            .all(|b| b.is_ascii_lowercase() || b.is_ascii_digit() || b == b'-')
}

/// Writes a file so a crash leaves the old contents or the new, never part.
fn write_atomic(path: &Path, bytes: &[u8]) -> io::Result<()> {
    let temp = path.with_extension("tmp");
    {
        let mut file = fs::File::create(&temp)?;
        file.write_all(bytes)?;
        file.sync_all()?;
    }
    fs::rename(&temp, path)
}

impl Spool {
    /// Opens (creating if needed) the spool under `root`.
    pub fn open(
        root: impl Into<PathBuf>,
        protector: Arc<dyn Protector>,
    ) -> Result<Self, SpoolError> {
        let root = root.into();
        fs::create_dir_all(root.join("batches"))?;
        fs::create_dir_all(root.join("failed"))?;
        Ok(Self {
            root,
            protector,
            lock: Mutex::new(()),
        })
    }

    /// A new, unique name for a batch, which is also the server's
    /// idempotency key for opening it.
    pub fn new_key() -> String {
        format!("cap-{}-{:016x}", now_unix_millis(), fastrand::u64(..))
    }

    fn dir(&self, key: &str) -> Result<PathBuf, SpoolError> {
        if !valid_key(key) {
            return Err(SpoolError::Missing(key.to_owned()));
        }
        Ok(self.root.join("batches").join(key))
    }

    fn guard(&self) -> std::sync::MutexGuard<'_, ()> {
        self.lock.lock().unwrap_or_else(PoisonError::into_inner)
    }

    fn read_manifest(&self, key: &str) -> Result<SpooledBatch, SpoolError> {
        let path = self.dir(key)?.join(MANIFEST);
        let bytes = fs::read(&path).map_err(|err| match err.kind() {
            io::ErrorKind::NotFound => SpoolError::Missing(key.to_owned()),
            _ => SpoolError::Io(err),
        })?;
        serde_json::from_slice(&bytes).map_err(|e| SpoolError::Manifest(e.to_string()))
    }

    fn write_manifest(&self, batch: &SpooledBatch) -> Result<(), SpoolError> {
        let bytes =
            serde_json::to_vec_pretty(batch).map_err(|e| SpoolError::Manifest(e.to_string()))?;
        write_atomic(&self.dir(batch.key())?.join(MANIFEST), &bytes)?;
        Ok(())
    }

    fn update<R>(
        &self,
        key: &str,
        change: impl FnOnce(&mut SpooledBatch) -> Result<R, SpoolError>,
    ) -> Result<R, SpoolError> {
        let _guard = self.guard();
        let mut batch = self.read_manifest(key)?;
        let result = change(&mut batch)?;
        self.write_manifest(&batch)?;
        Ok(result)
    }

    fn page_path(dir: &Path, sequence: u32) -> PathBuf {
        dir.join(format!("page-{sequence:04}.bin"))
    }

    /// Starts a batch. Its key is `input.client_key`, from [`Spool::new_key`].
    pub fn create(
        &self,
        input: OpenBatchInput,
        label: impl Into<String>,
    ) -> Result<SpooledBatch, SpoolError> {
        let _guard = self.guard();
        let dir = self.dir(&input.client_key)?;
        fs::create_dir_all(&dir)?;
        let batch = SpooledBatch {
            version: MANIFEST_VERSION,
            created_at: now_unix_millis(),
            label: label.into(),
            input,
            batch_id: None,
            pages: Vec::new(),
            complete: false,
        };
        self.write_manifest(&batch)?;
        Ok(batch)
    }

    /// Records the settings the scan actually used, once the source says.
    pub fn set_settings(&self, key: &str, settings: Settings) -> Result<(), SpoolError> {
        self.update(key, |batch| {
            batch.input.settings = settings;
            Ok(())
        })
    }

    /// Adds the next page, encrypted, returning its sequence (from 1).
    pub fn append_page(
        &self,
        key: &str,
        pdf: &[u8],
        markers: &PageMarkers,
    ) -> Result<u32, SpoolError> {
        if pdf.len() > MAX_PAGE_BYTES {
            return Err(SpoolError::PageTooLarge(pdf.len()));
        }
        let sealed = self.protector.protect(pdf)?;
        let checksum = page_checksum(pdf);
        self.update(key, |batch| {
            if batch.complete {
                return Err(SpoolError::Complete);
            }
            let sequence = u32::try_from(batch.pages.len())
                .unwrap_or(u32::MAX)
                .saturating_add(1);
            if sequence > MAX_BATCH_PAGES {
                return Err(SpoolError::Full);
            }
            write_atomic(&Self::page_path(&self.dir(key)?, sequence), &sealed)?;
            batch.pages.push(SpooledPage {
                sequence,
                checksum,
                byte_size: u64::try_from(pdf.len()).unwrap_or(u64::MAX),
                dpi: markers.dpi,
                patch_code: markers.patch_code.clone(),
                barcodes: markers.barcodes.clone(),
                uploaded: false,
            });
            Ok(sequence)
        })
    }

    /// Ends acquisition. A batch with no pages has nothing to send and is
    /// removed; returns whether it had any.
    pub fn complete(&self, key: &str) -> Result<bool, SpoolError> {
        let has_pages = self.update(key, |batch| {
            batch.complete = true;
            Ok(!batch.pages.is_empty())
        })?;
        if !has_pages {
            self.remove(key)?;
        }
        Ok(has_pages)
    }

    /// Every batch waiting, oldest first.
    pub fn pending(&self) -> Result<Vec<SpooledBatch>, SpoolError> {
        let _guard = self.guard();
        let mut batches = Vec::new();
        for entry in fs::read_dir(self.root.join("batches"))? {
            let entry = entry?;
            let Some(key) = entry.file_name().to_str().map(str::to_owned) else {
                continue;
            };
            if !valid_key(&key) || !entry.file_type()?.is_dir() {
                continue;
            }
            match self.read_manifest(&key) {
                Ok(batch) => batches.push(batch),
                Err(SpoolError::Missing(_)) => {}
                Err(err) => tracing::error!(key, error = %err, "a spooled batch could not be read"),
            }
        }
        batches.sort_by_key(|b| b.created_at);
        Ok(batches)
    }

    pub fn get(&self, key: &str) -> Result<SpooledBatch, SpoolError> {
        let _guard = self.guard();
        self.read_manifest(key)
    }

    /// A page's PDF, decrypted and checked against what was scanned.
    pub fn read_page(&self, key: &str, page: &SpooledPage) -> Result<Vec<u8>, SpoolError> {
        let sealed = fs::read(Self::page_path(&self.dir(key)?, page.sequence))?;
        let pdf = self.protector.unprotect(&sealed)?;
        if page_checksum(&pdf) != page.checksum {
            return Err(SpoolError::Corrupt {
                sequence: page.sequence,
            });
        }
        Ok(pdf)
    }

    pub fn set_batch_id(&self, key: &str, id: Id) -> Result<(), SpoolError> {
        self.update(key, |batch| {
            batch.batch_id = Some(id);
            Ok(())
        })
    }

    /// Drops the request a batch was for, when that request is no longer
    /// open, so its pages go to intake instead.
    pub fn detach_request(&self, key: &str) -> Result<(), SpoolError> {
        self.update(key, |batch| {
            batch.input.request_id = None;
            Ok(())
        })
    }

    pub fn mark_uploaded(&self, key: &str, sequence: u32) -> Result<(), SpoolError> {
        self.update(key, |batch| {
            if let Some(page) = batch.pages.iter_mut().find(|p| p.sequence == sequence) {
                page.uploaded = true;
            }
            Ok(())
        })
    }

    /// Deletes a batch the server has in full.
    pub fn remove(&self, key: &str) -> Result<(), SpoolError> {
        let dir = self.dir(key)?;
        match fs::remove_dir_all(&dir) {
            Err(err) if err.kind() != io::ErrorKind::NotFound => Err(err.into()),
            _ => Ok(()),
        }
    }

    /// Sets aside a batch the server refused, with the reason, so nothing is
    /// deleted that a person has not seen.
    pub fn fail(&self, key: &str, reason: &str) -> Result<(), SpoolError> {
        let _guard = self.guard();
        let from = self.dir(key)?;
        let to = self.root.join("failed").join(key);
        fs::rename(&from, &to)?;
        write_atomic(&to.join(FAILURE), reason.as_bytes())?;
        Ok(())
    }

    /// Where refused batches are kept.
    pub fn failed_dir(&self) -> PathBuf {
        self.root.join("failed")
    }

    pub fn summary(&self) -> Result<SpoolSummary, SpoolError> {
        let pending = self.pending()?;
        let failed = fs::read_dir(self.failed_dir())?
            .filter_map(Result::ok)
            .filter(|e| e.file_type().is_ok_and(|t| t.is_dir()))
            .count();
        Ok(SpoolSummary {
            batches: u32::try_from(pending.len()).unwrap_or(u32::MAX),
            pages_waiting: pending.iter().map(SpooledBatch::pages_waiting).sum(),
            failed: u32::try_from(failed).unwrap_or(u32::MAX),
        })
    }
}

#[cfg(test)]
pub(crate) mod tests {
    use super::*;
    use capture_protocol::api::BatchSource;

    /// Reverses the bytes, so a test can tell sealed from plain without any
    /// real key.
    #[derive(Debug)]
    pub(crate) struct Reverse;

    impl Protector for Reverse {
        fn protect(&self, plain: &[u8]) -> io::Result<Vec<u8>> {
            Ok(plain.iter().rev().copied().collect())
        }

        fn unprotect(&self, sealed: &[u8]) -> io::Result<Vec<u8>> {
            Ok(sealed.iter().rev().copied().collect())
        }
    }

    pub(crate) fn input(key: &str) -> OpenBatchInput {
        OpenBatchInput {
            client_key: key.to_owned(),
            source: BatchSource::Scan,
            request_id: Some(Id::from("creq_1")),
            profile_id: None,
            source_name: "fi-8170".into(),
            job_name: String::new(),
            settings: Settings::default(),
        }
    }

    fn markers() -> PageMarkers {
        PageMarkers {
            dpi: 300,
            patch_code: Some("T".into()),
            barcodes: vec!["PRO 1042".into()],
        }
    }

    #[test]
    fn pages_survive_a_restart_encrypted_and_in_order() {
        let dir = tempfile::tempdir().expect("dir");
        let key = Spool::new_key();
        {
            let spool = Spool::open(dir.path(), Arc::new(Reverse)).expect("spool");
            spool.create(input(&key), "fi-8170").expect("create");
            assert_eq!(
                spool
                    .append_page(&key, b"%PDF one", &markers())
                    .expect("page"),
                1
            );
            assert_eq!(
                spool
                    .append_page(&key, b"%PDF two", &PageMarkers::default())
                    .expect("page"),
                2
            );
        }

        let spool = Spool::open(dir.path(), Arc::new(Reverse)).expect("reopen");
        let batches = spool.pending().expect("pending");
        assert_eq!(batches.len(), 1);
        let batch = &batches[0];
        assert_eq!(batch.pages.len(), 2);
        assert_eq!(batch.pages[0].patch_code.as_deref(), Some("T"));
        assert_eq!(batch.pages[0].checksum, page_checksum(b"%PDF one"));
        assert!(!batch.complete);

        let on_disk =
            fs::read(dir.path().join("batches").join(&key).join("page-0001.bin")).expect("file");
        assert_ne!(on_disk, b"%PDF one", "stored sealed");
        assert_eq!(
            spool.read_page(&key, &batch.pages[1]).expect("read"),
            b"%PDF two"
        );
        assert_eq!(spool.summary().expect("summary").pages_waiting, 2);
    }

    #[test]
    fn a_page_changed_on_disk_is_refused() {
        let dir = tempfile::tempdir().expect("dir");
        let spool = Spool::open(dir.path(), Arc::new(Reverse)).expect("spool");
        let key = Spool::new_key();
        spool.create(input(&key), "fi-8170").expect("create");
        spool
            .append_page(&key, b"%PDF one", &markers())
            .expect("page");
        fs::write(
            dir.path().join("batches").join(&key).join("page-0001.bin"),
            b"tampered",
        )
        .expect("write");
        let page = spool.get(&key).expect("batch").pages[0].clone();
        assert!(matches!(
            spool.read_page(&key, &page),
            Err(SpoolError::Corrupt { sequence: 1 })
        ));
    }

    #[test]
    fn a_finished_batch_takes_no_more_pages_and_an_empty_one_is_dropped() {
        let dir = tempfile::tempdir().expect("dir");
        let spool = Spool::open(dir.path(), Arc::new(Reverse)).expect("spool");
        let key = Spool::new_key();
        spool.create(input(&key), "fi-8170").expect("create");
        spool.append_page(&key, b"%PDF", &markers()).expect("page");
        assert!(spool.complete(&key).expect("complete"));
        assert!(matches!(
            spool.append_page(&key, b"%PDF", &markers()),
            Err(SpoolError::Complete)
        ));

        let empty = Spool::new_key();
        spool.create(input(&empty), "fi-8170").expect("create");
        assert!(!spool.complete(&empty).expect("complete"));
        assert_eq!(spool.pending().expect("pending").len(), 1);
    }

    #[test]
    fn progress_is_recorded_and_a_refused_batch_is_set_aside_with_its_reason() {
        let dir = tempfile::tempdir().expect("dir");
        let spool = Spool::open(dir.path(), Arc::new(Reverse)).expect("spool");
        let key = Spool::new_key();
        spool.create(input(&key), "fi-8170").expect("create");
        spool.append_page(&key, b"%PDF", &markers()).expect("page");
        spool.set_batch_id(&key, Id::from("cbat_9")).expect("id");
        spool.detach_request(&key).expect("detach");
        spool.mark_uploaded(&key, 1).expect("uploaded");
        let batch = spool.get(&key).expect("batch");
        assert_eq!(batch.batch_id, Some(Id::from("cbat_9")));
        assert_eq!(batch.input.request_id, None);
        assert_eq!(batch.pages_waiting(), 0);

        spool.fail(&key, "not a PDF").expect("fail");
        assert!(spool.pending().expect("pending").is_empty());
        let reason = fs::read_to_string(dir.path().join("failed").join(&key).join("failure.txt"))
            .expect("reason");
        assert_eq!(reason, "not a PDF");
        assert_eq!(spool.summary().expect("summary").failed, 1);
    }

    #[test]
    fn keys_cannot_escape_the_spool() {
        let dir = tempfile::tempdir().expect("dir");
        let spool = Spool::open(dir.path(), Arc::new(Reverse)).expect("spool");
        assert!(matches!(
            spool.create(input("../evil"), "x"),
            Err(SpoolError::Missing(_))
        ));
        assert!(matches!(
            spool.get("C:\\Windows"),
            Err(SpoolError::Missing(_))
        ));
        assert!(valid_key(&Spool::new_key()));
    }
}
