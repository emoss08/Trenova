//! Pages on disk until the server has them.
//!
//! Every page is written here, encrypted, before any upload is tried, so an
//! unplugged cable, a sleeping laptop or a crash never loses a scan. Each
//! batch is a directory holding a manifest and one file per page:
//!
//! ```text
//! <root>/batches/<client key>/manifest.json
//! <root>/batches/<client key>/page-0001.bin
//! <root>/batches/<client key>/page-0001.thumb, page-0001.view  (its pictures)
//! <root>/batches/<client key>/document.bin  (a printed job, sent whole)
//! <root>/batches/<client key>/document-0001.thumb, ...  (its pages' pictures)
//! <root>/failed/<client key>/...           (the server refused it)
//! ```
//!
//! The manifest says which pages the server already has and what the batch
//! is called there, so an upload interrupted anywhere resumes exactly where
//! it stopped. It carries no page content and no credential. Every file is
//! written to a temporary name, flushed and renamed over the old one, so a
//! crash leaves either the old version or the new one, never half of one.
//!
//! What changes once per page (a page added, its pictures kept, the server
//! taking it) is appended to a journal beside the manifest instead of
//! rewriting it, so the thousandth page of a stack costs what the first did.
//! The manifest names the journal that continues it; writing a new manifest
//! starts a new journal, so a crash between the two leaves an old journal
//! that is never read again. A line cut short by a crash is the last one and
//! is dropped, and the change it carried is made again. Each batch is kept
//! in memory once read, so nothing is parsed twice.
//!
//! A batch held for review is not sent until the person releases it; until
//! then its pages can be turned or taken out. Pictures are encrypted like
//! the pages, and are only ever shown, never sent.

use std::collections::HashMap;
use std::fs;
use std::io::{self, Write};
use std::path::{Path, PathBuf};
use std::sync::{Arc, Mutex, PoisonError};
use std::time::{SystemTime, UNIX_EPOCH};

use capture_protocol::api::{
    BatchSource, Id, MAX_BATCH_PAGES, MAX_PAGE_BYTES, MAX_PRINT_JOB_BYTES, OpenBatchInput, Settings,
};
use capture_protocol::page_checksum;
use serde::{Deserialize, Serialize};

use crate::api::PageMarkers;

const MANIFEST: &str = "manifest.json";
const FAILURE: &str = "failure.txt";
const DOCUMENT: &str = "document.bin";
const THUMB: &str = "thumb";
const VIEW: &str = "view";
const MANIFEST_VERSION: u32 = 1;
/// Journal lines kept before the manifest is written afresh, so reading a
/// batch after a restart replays at most this many.
const JOURNAL_COMPACT_AFTER: usize = 512;

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
    /// Degrees clockwise the person turned it before sending: 0, 90, 180
    /// or 270. The page itself is sent as scanned, with this beside it.
    #[serde(default)]
    pub rotation: u16,
    /// Its pictures are on disk.
    #[serde(default)]
    pub pictures: bool,
}

impl SpooledPage {
    pub fn markers(&self) -> PageMarkers {
        PageMarkers {
            dpi: self.dpi,
            patch_code: self.patch_code.clone(),
            barcodes: self.barcodes.clone(),
            rotation: self.rotation,
        }
    }
}

/// A page that has pictures.
#[derive(Clone, Copy, Debug, PartialEq, Eq)]
pub struct PictureRef {
    pub page: u32,
    /// Degrees clockwise it was turned.
    pub rotation: u16,
}

/// A page's two pictures, as JPEG.
#[derive(Clone, Debug, PartialEq, Eq)]
pub struct PagePictures {
    pub thumb: Vec<u8>,
    pub view: Vec<u8>,
}

/// Which of a page's pictures.
#[derive(Clone, Copy, Debug, PartialEq, Eq)]
pub enum PictureSize {
    Thumb,
    View,
}

impl PictureSize {
    fn extension(self) -> &'static str {
        match self {
            Self::Thumb => THUMB,
            Self::View => VIEW,
        }
    }
}

/// Keeps a rotation to the four a page can have.
pub fn normalize_rotation(degrees: i32) -> u16 {
    let quarter = (degrees.rem_euclid(360) + 45) / 90 % 4;
    u16::try_from(quarter * 90).unwrap_or(0)
}

/// A printed job as the manifest records it. The server splits it into
/// pages, so it is sent whole.
#[derive(Clone, Debug, PartialEq, Eq, Serialize, Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct SpooledDocument {
    /// SHA-256 of the PDF.
    pub checksum: String,
    pub byte_size: u64,
    /// Pages, when the print service counted them.
    #[serde(default)]
    pub pages: Option<u32>,
    /// Pictures of its pages on disk, when the print service made them.
    #[serde(default)]
    pub pictures: u32,
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
    /// A printed job, which takes the place of pages.
    #[serde(default)]
    pub document: Option<SpooledDocument>,
    /// Acquisition has ended; the batch can be sealed once uploaded.
    pub complete: bool,
    /// Waiting for the person to look it over; nothing is sent until they
    /// release it.
    #[serde(default)]
    pub held: bool,
}

impl SpooledBatch {
    pub fn key(&self) -> &str {
        &self.input.client_key
    }

    /// The pages with pictures, and how each is turned: a scanned page by
    /// its sequence, a printed job's by its number.
    pub fn picture_refs(&self) -> Vec<PictureRef> {
        match &self.document {
            Some(document) => (1..=document.pictures)
                .map(|page| PictureRef { page, rotation: 0 })
                .collect(),
            None => self
                .pages
                .iter()
                .filter(|page| page.pictures)
                .map(|page| PictureRef {
                    page: page.sequence,
                    rotation: page.rotation,
                })
                .collect(),
        }
    }

    pub fn pages_waiting(&self) -> u32 {
        if let Some(document) = &self.document {
            return document.pages.unwrap_or(1);
        }
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

/// A batch the server refused, as it is kept aside for a person to decide.
#[derive(Clone, Debug, PartialEq, Eq)]
pub struct RefusedBatch {
    pub key: String,
    pub label: String,
    pub source: BatchSource,
    /// Pages held, or the printed job's pages when they were counted.
    pub pages: u32,
    /// What the server said, or why the batch could not be read.
    pub reason: String,
    /// When it was spooled, in Unix milliseconds.
    pub created_at: i64,
    /// When it was set aside, in Unix milliseconds.
    pub refused_at: i64,
    /// Whether its manifest could be read, and so whether it can be sent
    /// again or saved.
    pub readable: bool,
    /// Its pages that have pictures.
    pub pictures: Vec<PictureRef>,
}

/// What saving a refused batch wrote.
#[derive(Clone, Debug, Default, PartialEq, Eq)]
pub struct Export {
    pub written: Vec<PathBuf>,
    /// Pages whose file no longer matched what was scanned, left out.
    pub unreadable: Vec<u32>,
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
    #[error("a printed document of {0} bytes is over the server's limit")]
    DocumentTooLarge(usize),
    #[error("the batch has already been finished")]
    Complete,
    #[error("page {sequence} on disk does not match what was scanned")]
    Corrupt { sequence: u32 },
    #[error("the printed document on disk does not match what was printed")]
    CorruptDocument,
    #[error("the manifest is unreadable: {0}")]
    Manifest(String),
    #[error("{0} already exists")]
    Exists(PathBuf),
    #[error("only a batch held for review can be changed")]
    NotHeld,
    #[error("page {0} is not in the batch")]
    NoPage(u32),
    #[error("that picture is too large")]
    PictureTooLarge,
}

/// The largest picture kept. A 1100-pixel JPEG of a busy colour page is a
/// few hundred kilobytes.
pub const MAX_PICTURE_BYTES: usize = 2 << 20;

/// A batch kept in memory, as its manifest and journal leave it.
struct Cached {
    batch: SpooledBatch,
    journal: u64,
    entries: usize,
}

pub struct Spool {
    root: PathBuf,
    protector: Arc<dyn Protector>,
    /// Serialises manifest read-modify-writes between the scan and upload
    /// threads.
    lock: Mutex<()>,
    /// Batches waiting, as last read or written. Only this process writes
    /// the spool, and only while holding `lock`.
    cache: Mutex<HashMap<String, Cached>>,
}

impl std::fmt::Debug for Spool {
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        f.debug_struct("Spool")
            .field("root", &self.root)
            .finish_non_exhaustive()
    }
}

fn now_unix_millis() -> i64 {
    unix_millis(SystemTime::now())
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

/// Writes a file that must not exist yet.
fn write_new(path: &Path, bytes: &[u8]) -> io::Result<()> {
    let mut file = fs::OpenOptions::new()
        .write(true)
        .create_new(true)
        .open(path)?;
    file.write_all(bytes)?;
    file.sync_all()
}

/// The manifest as it is written: the batch, and which journal continues it.
#[derive(Deserialize)]
struct Snapshot {
    #[serde(flatten)]
    batch: SpooledBatch,
    #[serde(default)]
    journal: u64,
}

#[derive(Serialize)]
struct SnapshotRef<'a> {
    #[serde(flatten)]
    batch: &'a SpooledBatch,
    journal: u64,
}

/// One change to a batch, as a journal line.
#[derive(Debug, Serialize, Deserialize)]
#[serde(tag = "op", rename_all = "camelCase")]
enum Change {
    Page { page: SpooledPage },
    Pictures { sequence: u32 },
    Uploaded { sequence: u32 },
}

impl Change {
    /// Makes the change. Each is safe to make twice, as replaying a journal
    /// over a manifest written after it does.
    fn apply(self, batch: &mut SpooledBatch) {
        match self {
            Self::Page { page } => {
                let next = u32::try_from(batch.pages.len())
                    .unwrap_or(u32::MAX)
                    .saturating_add(1);
                if page.sequence == next {
                    batch.pages.push(page);
                }
            }
            Self::Pictures { sequence } => {
                if let Some(page) = batch.pages.iter_mut().find(|p| p.sequence == sequence) {
                    page.pictures = true;
                }
            }
            Self::Uploaded { sequence } => {
                if let Some(page) = batch.pages.iter_mut().find(|p| p.sequence == sequence) {
                    page.uploaded = true;
                }
            }
        }
    }
}

/// A batch as read from disk.
struct Loaded {
    batch: SpooledBatch,
    journal: u64,
    entries: usize,
    /// The journal ends in a line a crash cut short.
    torn: bool,
}

fn journal_path(dir: &Path, journal: u64) -> PathBuf {
    dir.join(format!("journal-{journal:08}.jsonl"))
}

fn load(dir: &Path, key: &str) -> Result<Loaded, SpoolError> {
    let bytes = fs::read(dir.join(MANIFEST)).map_err(|err| match err.kind() {
        io::ErrorKind::NotFound => SpoolError::Missing(key.to_owned()),
        _ => SpoolError::Io(err),
    })?;
    let Snapshot { mut batch, journal } =
        serde_json::from_slice(&bytes).map_err(|e| SpoolError::Manifest(e.to_string()))?;
    let lines = match fs::read(journal_path(dir, journal)) {
        Ok(lines) => lines,
        Err(err) if err.kind() == io::ErrorKind::NotFound => Vec::new(),
        Err(err) => return Err(err.into()),
    };
    let mut entries = 0;
    let mut rest = lines.as_slice();
    while let Some(end) = rest.iter().position(|&b| b == b'\n') {
        let change: Change = serde_json::from_slice(&rest[..end])
            .map_err(|e| SpoolError::Manifest(format!("journal line {}: {e}", entries + 1)))?;
        change.apply(&mut batch);
        entries += 1;
        rest = &rest[end + 1..];
    }
    Ok(Loaded {
        batch,
        journal,
        entries,
        torn: !rest.is_empty(),
    })
}

fn read_manifest_in(dir: &Path, key: &str) -> Result<SpooledBatch, SpoolError> {
    load(dir, key).map(|loaded| loaded.batch)
}

/// Writes a manifest continued by an empty journal numbered `journal`, and
/// removes the journal the previous manifest named.
fn write_snapshot(
    dir: &Path,
    batch: &SpooledBatch,
    previous: Option<u64>,
    journal: u64,
) -> Result<(), SpoolError> {
    let bytes = serde_json::to_vec_pretty(&SnapshotRef { batch, journal })
        .map_err(|e| SpoolError::Manifest(e.to_string()))?;
    write_atomic(&dir.join(MANIFEST), &bytes)?;
    if let Some(previous) = previous {
        remove_if_present(&journal_path(dir, previous))?;
    }
    Ok(())
}

fn append_line(path: &Path, change: &Change) -> Result<(), SpoolError> {
    let mut line = serde_json::to_vec(change).map_err(|e| SpoolError::Manifest(e.to_string()))?;
    line.push(b'\n');
    let mut file = fs::OpenOptions::new()
        .create(true)
        .append(true)
        .open(path)?;
    file.write_all(&line)?;
    file.sync_data()?;
    Ok(())
}

fn describe_refused(dir: &Path, key: String) -> RefusedBatch {
    let failure = dir.join(FAILURE);
    let refused_at = fs::metadata(&failure)
        .or_else(|_| fs::metadata(dir))
        .and_then(|meta| meta.modified())
        .map_or(0, unix_millis);
    let recorded = fs::read_to_string(&failure)
        .map(|reason| reason.trim().to_owned())
        .unwrap_or_default();
    match read_manifest_in(dir, &key) {
        Ok(batch) => RefusedBatch {
            label: batch.label.clone(),
            source: batch.input.source,
            pages: batch.document.as_ref().map_or_else(
                || u32::try_from(batch.pages.len()).unwrap_or(u32::MAX),
                |document| document.pages.unwrap_or(1),
            ),
            reason: recorded,
            created_at: batch.created_at,
            refused_at,
            readable: true,
            pictures: batch.picture_refs(),
            key,
        },
        Err(err) => RefusedBatch {
            label: String::new(),
            source: BatchSource::Scan,
            pages: 0,
            reason: if recorded.is_empty() {
                err.to_string()
            } else {
                recorded
            },
            created_at: refused_at,
            refused_at,
            readable: false,
            pictures: Vec::new(),
            key,
        },
    }
}

fn remove_if_present(path: &Path) -> io::Result<()> {
    match fs::remove_file(path) {
        Err(err) if err.kind() != io::ErrorKind::NotFound => Err(err),
        _ => Ok(()),
    }
}

/// A printed job to spool.
#[derive(Debug)]
pub struct NewPrint<'a> {
    pub input: OpenBatchInput,
    pub label: &'a str,
    pub pdf: &'a [u8],
    /// Pages, when the print service counted them.
    pub pages: Option<u32>,
    /// Pictures of its pages, in order, when the print service made them.
    pub pictures: &'a [PagePictures],
    pub held: bool,
}

fn unix_millis(at: SystemTime) -> i64 {
    at.duration_since(UNIX_EPOCH)
        .map_or(0, |d| i64::try_from(d.as_millis()).unwrap_or(i64::MAX))
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
            cache: Mutex::new(HashMap::new()),
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

    fn refused_dir(&self, key: &str) -> Result<PathBuf, SpoolError> {
        if !valid_key(key) {
            return Err(SpoolError::Missing(key.to_owned()));
        }
        Ok(self.failed_dir().join(key))
    }

    fn cache(&self) -> std::sync::MutexGuard<'_, HashMap<String, Cached>> {
        self.cache.lock().unwrap_or_else(PoisonError::into_inner)
    }

    /// Runs `read` over a waiting batch, reading it from disk the first time.
    /// The caller holds `lock`.
    fn with_batch<R>(
        &self,
        key: &str,
        read: impl FnOnce(&mut Cached) -> Result<R, SpoolError>,
    ) -> Result<R, SpoolError> {
        let dir = self.dir(key)?;
        let mut cache = self.cache();
        if !cache.contains_key(key) {
            let loaded = load(&dir, key)?;
            let mut cached = Cached {
                batch: loaded.batch,
                journal: loaded.journal,
                entries: loaded.entries,
            };
            if loaded.torn {
                let next = cached.journal.wrapping_add(1);
                write_snapshot(&dir, &cached.batch, Some(cached.journal), next)?;
                cached.journal = next;
                cached.entries = 0;
            }
            cache.insert(key.to_owned(), cached);
        }
        let cached = cache
            .get_mut(key)
            .ok_or_else(|| SpoolError::Missing(key.to_owned()))?;
        read(cached)
    }

    fn read_manifest(&self, key: &str) -> Result<SpooledBatch, SpoolError> {
        self.with_batch(key, |cached| Ok(cached.batch.clone()))
    }

    fn write_manifest(&self, batch: &SpooledBatch) -> Result<(), SpoolError> {
        let key = batch.key();
        let dir = self.dir(key)?;
        let mut cache = self.cache();
        let previous = cache.get(key).map(|cached| cached.journal);
        let journal = previous.map_or(0, |journal| journal.wrapping_add(1));
        cache.remove(key);
        write_snapshot(&dir, batch, previous, journal)?;
        cache.insert(
            key.to_owned(),
            Cached {
                batch: batch.clone(),
                journal,
                entries: 0,
            },
        );
        Ok(())
    }

    /// Records one change to a waiting batch by appending it to the journal.
    /// `decide` sees the batch as it stands and returns what to record, or
    /// nothing to leave it as it is.
    fn record<R>(
        &self,
        key: &str,
        decide: impl FnOnce(&SpooledBatch) -> Result<(R, Option<Change>), SpoolError>,
    ) -> Result<R, SpoolError> {
        let _guard = self.guard();
        let dir = self.dir(key)?;
        let mut unsure = false;
        let recorded = self.with_batch(key, |cached| {
            let (result, change) = decide(&cached.batch)?;
            let Some(change) = change else {
                return Ok(result);
            };
            if let Err(err) = append_line(&journal_path(&dir, cached.journal), &change) {
                unsure = true;
                return Err(err);
            }
            change.apply(&mut cached.batch);
            cached.entries += 1;
            if cached.entries >= JOURNAL_COMPACT_AFTER {
                let next = cached.journal.wrapping_add(1);
                write_snapshot(&dir, &cached.batch, Some(cached.journal), next)?;
                cached.journal = next;
                cached.entries = 0;
            }
            Ok(result)
        });
        if unsure {
            self.forget(key);
        }
        recorded
    }

    fn forget(&self, key: &str) {
        self.cache().remove(key);
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

    fn page_picture_path(dir: &Path, sequence: u32, size: PictureSize) -> PathBuf {
        dir.join(format!("page-{sequence:04}.{}", size.extension()))
    }

    fn document_picture_path(dir: &Path, page: u32, size: PictureSize) -> PathBuf {
        dir.join(format!("document-{page:04}.{}", size.extension()))
    }

    /// Starts a batch. Its key is `input.client_key`, from [`Spool::new_key`].
    pub fn create(
        &self,
        input: OpenBatchInput,
        label: impl Into<String>,
    ) -> Result<SpooledBatch, SpoolError> {
        self.create_batch(input, label.into(), false)
    }

    /// Starts a batch held for review: nothing of it is sent until
    /// [`Spool::release`].
    pub fn create_held(
        &self,
        input: OpenBatchInput,
        label: impl Into<String>,
    ) -> Result<SpooledBatch, SpoolError> {
        self.create_batch(input, label.into(), true)
    }

    fn create_batch(
        &self,
        input: OpenBatchInput,
        label: String,
        held: bool,
    ) -> Result<SpooledBatch, SpoolError> {
        let _guard = self.guard();
        let dir = self.dir(&input.client_key)?;
        fs::create_dir_all(&dir)?;
        let batch = SpooledBatch {
            version: MANIFEST_VERSION,
            created_at: now_unix_millis(),
            label,
            input,
            batch_id: None,
            pages: Vec::new(),
            document: None,
            complete: false,
            held,
        };
        self.write_manifest(&batch)?;
        Ok(batch)
    }

    /// Spools a printed job, complete, under `input.client_key`, with the
    /// pictures of its pages when there are any, held for review when
    /// `held`. Spooling the same key again returns the batch already there,
    /// so a job taken from the print inbox twice is sent once.
    pub fn create_print(&self, print: NewPrint<'_>) -> Result<SpooledBatch, SpoolError> {
        let NewPrint {
            input,
            label,
            pdf,
            pages,
            pictures,
            held,
        } = print;
        if pdf.len() > MAX_PRINT_JOB_BYTES {
            return Err(SpoolError::DocumentTooLarge(pdf.len()));
        }
        let sealed = self.protector.protect(pdf)?;
        let sealed_pictures = pictures
            .iter()
            .map(|picture| self.seal_pictures(picture))
            .collect::<Result<Vec<_>, _>>()?;
        let _guard = self.guard();
        match self.read_manifest(&input.client_key) {
            Ok(existing) => return Ok(existing),
            Err(SpoolError::Missing(_)) => {}
            Err(err) => return Err(err),
        }
        let dir = self.dir(&input.client_key)?;
        fs::create_dir_all(&dir)?;
        write_atomic(&dir.join(DOCUMENT), &sealed)?;
        for (index, (thumb, view)) in sealed_pictures.iter().enumerate() {
            let page = u32::try_from(index + 1).unwrap_or(u32::MAX);
            write_atomic(
                &Self::document_picture_path(&dir, page, PictureSize::Thumb),
                thumb,
            )?;
            write_atomic(
                &Self::document_picture_path(&dir, page, PictureSize::View),
                view,
            )?;
        }
        let batch = SpooledBatch {
            version: MANIFEST_VERSION,
            created_at: now_unix_millis(),
            label: label.to_owned(),
            input,
            batch_id: None,
            pages: Vec::new(),
            document: Some(SpooledDocument {
                checksum: page_checksum(pdf),
                byte_size: u64::try_from(pdf.len()).unwrap_or(u64::MAX),
                pages,
                pictures: u32::try_from(sealed_pictures.len()).unwrap_or(u32::MAX),
            }),
            complete: true,
            held,
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
        self.record(key, |batch| {
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
            let page = SpooledPage {
                sequence,
                checksum,
                byte_size: u64::try_from(pdf.len()).unwrap_or(u64::MAX),
                dpi: markers.dpi,
                patch_code: markers.patch_code.clone(),
                barcodes: markers.barcodes.clone(),
                uploaded: false,
                rotation: normalize_rotation(i32::from(markers.rotation)),
                pictures: false,
            };
            Ok((sequence, Some(Change::Page { page })))
        })
    }

    fn seal_pictures(&self, pictures: &PagePictures) -> Result<(Vec<u8>, Vec<u8>), SpoolError> {
        if pictures.thumb.len() > MAX_PICTURE_BYTES || pictures.view.len() > MAX_PICTURE_BYTES {
            return Err(SpoolError::PictureTooLarge);
        }
        Ok((
            self.protector.protect(&pictures.thumb)?,
            self.protector.protect(&pictures.view)?,
        ))
    }

    /// Keeps a spooled page's pictures beside it.
    pub fn store_pictures(
        &self,
        key: &str,
        sequence: u32,
        pictures: &PagePictures,
    ) -> Result<(), SpoolError> {
        let (thumb, view) = self.seal_pictures(pictures)?;
        self.record(key, |batch| {
            let dir = self.dir(key)?;
            if !batch.pages.iter().any(|p| p.sequence == sequence) {
                return Err(SpoolError::NoPage(sequence));
            }
            write_atomic(
                &Self::page_picture_path(&dir, sequence, PictureSize::Thumb),
                &thumb,
            )?;
            write_atomic(
                &Self::page_picture_path(&dir, sequence, PictureSize::View),
                &view,
            )?;
            Ok(((), Some(Change::Pictures { sequence })))
        })
    }

    /// One picture of a page, of a batch waiting or refused: a scanned page
    /// by its sequence, or a printed job's page by its number.
    pub fn picture(&self, key: &str, page: u32, size: PictureSize) -> Result<Vec<u8>, SpoolError> {
        let path = {
            let _guard = self.guard();
            match self.with_batch(key, |cached| {
                Self::picture_path(&self.dir(key)?, &cached.batch, page, size)
            }) {
                Err(SpoolError::Missing(_)) => {
                    let dir = self.refused_dir(key)?;
                    Self::picture_path(&dir, &read_manifest_in(&dir, key)?, page, size)
                }
                other => other,
            }?
        };
        let sealed = fs::read(path)?;
        let picture = self.protector.unprotect(&sealed)?;
        if picture.len() > MAX_PICTURE_BYTES {
            return Err(SpoolError::PictureTooLarge);
        }
        Ok(picture)
    }

    fn picture_path(
        dir: &Path,
        batch: &SpooledBatch,
        page: u32,
        size: PictureSize,
    ) -> Result<PathBuf, SpoolError> {
        match &batch.document {
            Some(document) if page >= 1 && page <= document.pictures => {
                Ok(Self::document_picture_path(dir, page, size))
            }
            None if batch.pages.iter().any(|p| p.sequence == page && p.pictures) => {
                Ok(Self::page_picture_path(dir, page, size))
            }
            Some(_) | None => Err(SpoolError::NoPage(page)),
        }
    }

    /// Holds a batch for review; nothing more of it is sent until released.
    pub fn hold(&self, key: &str) -> Result<(), SpoolError> {
        self.update(key, |batch| {
            batch.held = true;
            Ok(())
        })
    }

    /// Lets a held batch be sent.
    pub fn release(&self, key: &str) -> Result<(), SpoolError> {
        self.update(key, |batch| {
            batch.held = false;
            Ok(())
        })
    }

    /// Opens a held batch for more pages, as scanning more into it does.
    pub fn reopen(&self, key: &str) -> Result<(), SpoolError> {
        self.update(key, |batch| {
            if !batch.held || batch.document.is_some() {
                return Err(SpoolError::NotHeld);
            }
            batch.complete = false;
            Ok(())
        })
    }

    /// Turns a page of a held batch by `degrees` clockwise, returning where
    /// it now stands.
    pub fn rotate(&self, key: &str, sequence: u32, degrees: i32) -> Result<u16, SpoolError> {
        self.update(key, |batch| {
            if !batch.held {
                return Err(SpoolError::NotHeld);
            }
            let page = batch
                .pages
                .iter_mut()
                .find(|p| p.sequence == sequence && !p.uploaded)
                .ok_or(SpoolError::NoPage(sequence))?;
            page.rotation = normalize_rotation(i32::from(page.rotation) + degrees);
            Ok(page.rotation)
        })
    }

    /// Takes a page out of a held batch, moving the pages after it up so the
    /// sequence stays unbroken, and returns how many pages are left. The
    /// manifest is written before any file moves, and a crash part way
    /// leaves pages whose checksums no longer match, which reading reports
    /// rather than sends.
    pub fn delete_page(&self, key: &str, sequence: u32) -> Result<u32, SpoolError> {
        let _guard = self.guard();
        let dir = self.dir(key)?;
        let mut batch = self.read_manifest(key)?;
        if !batch.held || batch.document.is_some() {
            return Err(SpoolError::NotHeld);
        }
        if batch.pages.iter().any(|p| p.uploaded) {
            return Err(SpoolError::NotHeld);
        }
        let index = batch
            .pages
            .iter()
            .position(|p| p.sequence == sequence)
            .ok_or(SpoolError::NoPage(sequence))?;
        let removed = batch.pages.remove(index);
        let moved: Vec<(u32, u32)> = batch.pages[index..]
            .iter_mut()
            .map(|page| {
                let from = page.sequence;
                page.sequence -= 1;
                (from, page.sequence)
            })
            .collect();
        self.write_manifest(&batch)?;

        remove_if_present(&Self::page_path(&dir, removed.sequence))?;
        for size in [PictureSize::Thumb, PictureSize::View] {
            remove_if_present(&Self::page_picture_path(&dir, removed.sequence, size))?;
        }
        for (from, to) in moved {
            fs::rename(Self::page_path(&dir, from), Self::page_path(&dir, to))?;
            for size in [PictureSize::Thumb, PictureSize::View] {
                let old = Self::page_picture_path(&dir, from, size);
                if old.exists() {
                    fs::rename(old, Self::page_picture_path(&dir, to, size))?;
                }
            }
        }
        Ok(u32::try_from(batch.pages.len()).unwrap_or(u32::MAX))
    }

    /// Deletes a batch held for review, and its pages, for good.
    pub fn discard_held(&self, key: &str) -> Result<(), SpoolError> {
        {
            let _guard = self.guard();
            if !self.read_manifest(key)?.held {
                return Err(SpoolError::NotHeld);
            }
        }
        self.remove(key)
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

    /// Every batch waiting to be sent, oldest first, leaving out those held
    /// for review.
    pub fn sendable(&self) -> Result<Vec<SpooledBatch>, SpoolError> {
        let mut batches = self.pending()?;
        batches.retain(|batch| !batch.held);
        Ok(batches)
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
        self.read_page_in(&self.dir(key)?, page)
    }

    fn read_page_in(&self, dir: &Path, page: &SpooledPage) -> Result<Vec<u8>, SpoolError> {
        let sealed = fs::read(Self::page_path(dir, page.sequence))?;
        let pdf = self.protector.unprotect(&sealed)?;
        if page_checksum(&pdf) != page.checksum {
            return Err(SpoolError::Corrupt {
                sequence: page.sequence,
            });
        }
        Ok(pdf)
    }

    /// A printed job's PDF, decrypted and checked against what was printed.
    pub fn read_document(
        &self,
        key: &str,
        document: &SpooledDocument,
    ) -> Result<Vec<u8>, SpoolError> {
        self.read_document_in(&self.dir(key)?, document)
    }

    fn read_document_in(
        &self,
        dir: &Path,
        document: &SpooledDocument,
    ) -> Result<Vec<u8>, SpoolError> {
        let sealed = fs::read(dir.join(DOCUMENT))?;
        let pdf = self.protector.unprotect(&sealed)?;
        if page_checksum(&pdf) != document.checksum {
            return Err(SpoolError::CorruptDocument);
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
        self.record(key, |batch| {
            let waiting = batch
                .pages
                .iter()
                .any(|p| p.sequence == sequence && !p.uploaded);
            Ok(((), waiting.then_some(Change::Uploaded { sequence })))
        })
    }

    /// Deletes a batch the server has in full.
    pub fn remove(&self, key: &str) -> Result<(), SpoolError> {
        let dir = self.dir(key)?;
        self.forget(key);
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
        self.forget(key);
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

    /// Every batch the server refused, most recently refused first.
    pub fn refused(&self) -> Result<Vec<RefusedBatch>, SpoolError> {
        let _guard = self.guard();
        let mut refused = Vec::new();
        for entry in fs::read_dir(self.failed_dir())? {
            let entry = entry?;
            let Some(key) = entry.file_name().to_str().map(str::to_owned) else {
                continue;
            };
            if !valid_key(&key) || !entry.file_type()?.is_dir() {
                continue;
            }
            refused.push(describe_refused(&entry.path(), key));
        }
        refused.sort_by(|a, b| {
            b.refused_at
                .cmp(&a.refused_at)
                .then_with(|| a.key.cmp(&b.key))
        });
        Ok(refused)
    }

    /// Sends a refused batch again as a new batch, to intake: the server's
    /// batch for the old key is closed, and the request it was for may be
    /// too. Every page is sent afresh. Returns the new key.
    ///
    /// The manifest is rewritten in place before the directory moves, so a
    /// crash in between leaves a refused batch that can be retried again.
    pub fn retry(&self, key: &str) -> Result<String, SpoolError> {
        let _guard = self.guard();
        let from = self.refused_dir(key)?;
        let Loaded {
            mut batch, journal, ..
        } = load(&from, key)?;
        let new_key = Self::new_key();
        batch.input.client_key.clone_from(&new_key);
        batch.input.request_id = None;
        batch.batch_id = None;
        batch.complete = true;
        for page in &mut batch.pages {
            page.uploaded = false;
        }
        write_snapshot(&from, &batch, Some(journal), journal.wrapping_add(1))?;
        match fs::remove_file(from.join(FAILURE)) {
            Err(err) if err.kind() != io::ErrorKind::NotFound => return Err(err.into()),
            _ => {}
        }
        fs::rename(&from, self.dir(&new_key)?)?;
        Ok(new_key)
    }

    /// Deletes a refused batch and its pages for good.
    pub fn discard(&self, key: &str) -> Result<(), SpoolError> {
        let _guard = self.guard();
        let dir = self.refused_dir(key)?;
        match fs::remove_dir_all(&dir) {
            Err(err) if err.kind() == io::ErrorKind::NotFound => {
                Err(SpoolError::Missing(key.to_owned()))
            }
            Err(err) => Err(err.into()),
            Ok(()) => Ok(()),
        }
    }

    /// Writes a refused batch's pages, decrypted, into `into` as PDFs, so a
    /// person can send them another way. `into` must not exist yet; nothing
    /// already on disk is overwritten. A page that no longer matches what
    /// was scanned is left out and reported.
    pub fn export(&self, key: &str, into: &Path) -> Result<Export, SpoolError> {
        let _guard = self.guard();
        let dir = self.refused_dir(key)?;
        let batch = read_manifest_in(&dir, key)?;
        match fs::create_dir(into) {
            Err(err) if err.kind() == io::ErrorKind::AlreadyExists => {
                return Err(SpoolError::Exists(into.to_path_buf()));
            }
            other => other?,
        }
        let mut export = Export::default();
        if let Some(document) = &batch.document {
            match self.read_document_in(&dir, document) {
                Ok(pdf) => {
                    let path = into.join("document.pdf");
                    write_new(&path, &pdf)?;
                    export.written.push(path);
                }
                Err(SpoolError::CorruptDocument | SpoolError::Io(_)) => export.unreadable.push(1),
                Err(err) => return Err(err),
            }
            return Ok(export);
        }
        for page in &batch.pages {
            match self.read_page_in(&dir, page) {
                Ok(pdf) => {
                    let path = into.join(format!("page-{:04}.pdf", page.sequence));
                    write_new(&path, &pdf)?;
                    export.written.push(path);
                }
                Err(SpoolError::Corrupt { sequence }) => export.unreadable.push(sequence),
                Err(SpoolError::Io(err)) if err.kind() == io::ErrorKind::NotFound => {
                    export.unreadable.push(page.sequence);
                }
                Err(err) => return Err(err),
            }
        }
        Ok(export)
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
            rotation: 0,
        }
    }

    fn print<'a>(key: &str, pictures: &'a [PagePictures], held: bool) -> NewPrint<'a> {
        let mut input = input(key);
        input.source = BatchSource::Print;
        NewPrint {
            input,
            label: "Invoice.docx",
            pdf: b"%PDF printed",
            pages: Some(3),
            pictures,
            held,
        }
    }

    fn pictures(n: u8) -> PagePictures {
        PagePictures {
            thumb: vec![0xFF, 0xD8, n, 1],
            view: vec![0xFF, 0xD8, n, 2],
        }
    }

    #[test]
    fn a_held_batch_is_not_sendable_until_released() {
        let dir = tempfile::tempdir().expect("dir");
        let spool = Spool::open(dir.path(), Arc::new(Reverse)).expect("spool");
        let key = Spool::new_key();
        spool.create_held(input(&key), "fi-8170").expect("create");
        spool
            .append_page(&key, b"%PDF one", &markers())
            .expect("page");
        spool.complete(&key).expect("complete");
        assert!(spool.sendable().expect("sendable").is_empty());
        assert_eq!(spool.pending().expect("pending").len(), 1);

        spool.release(&key).expect("release");
        assert_eq!(spool.sendable().expect("sendable").len(), 1);
    }

    #[test]
    fn a_held_page_turns_and_is_sent_turned() {
        let dir = tempfile::tempdir().expect("dir");
        let spool = Spool::open(dir.path(), Arc::new(Reverse)).expect("spool");
        let key = Spool::new_key();
        spool.create(input(&key), "fi-8170").expect("create");
        spool
            .append_page(&key, b"%PDF one", &markers())
            .expect("page");
        assert!(matches!(
            spool.rotate(&key, 1, 90),
            Err(SpoolError::NotHeld)
        ));

        spool.hold(&key).expect("hold");
        assert_eq!(spool.rotate(&key, 1, 90).expect("turn"), 90);
        assert_eq!(spool.rotate(&key, 1, -180).expect("turn"), 270);
        assert_eq!(spool.rotate(&key, 1, 90).expect("turn"), 0);
        assert_eq!(spool.rotate(&key, 1, 180).expect("turn"), 180);
        let batch = spool.get(&key).expect("batch");
        assert_eq!(batch.pages[0].markers().rotation, 180);
        assert!(matches!(
            spool.rotate(&key, 2, 90),
            Err(SpoolError::NoPage(2))
        ));
    }

    #[test]
    fn rotations_are_kept_to_quarter_turns() {
        assert_eq!(normalize_rotation(0), 0);
        assert_eq!(normalize_rotation(-90), 270);
        assert_eq!(normalize_rotation(450), 90);
        assert_eq!(normalize_rotation(200), 180);
        assert_eq!(normalize_rotation(359), 0);
    }

    #[test]
    fn taking_a_page_out_keeps_the_rest_in_order_with_their_pictures() {
        let dir = tempfile::tempdir().expect("dir");
        let spool = Spool::open(dir.path(), Arc::new(Reverse)).expect("spool");
        let key = Spool::new_key();
        spool.create_held(input(&key), "fi-8170").expect("create");
        for (n, pdf) in [b"%PDF one".as_slice(), b"%PDF two", b"%PDF three"]
            .into_iter()
            .enumerate()
        {
            let sequence = spool.append_page(&key, pdf, &markers()).expect("page");
            let n = u8::try_from(n).expect("n");
            spool
                .store_pictures(&key, sequence, &pictures(n))
                .expect("pictures");
        }
        spool.rotate(&key, 3, 90).expect("turn");

        assert_eq!(spool.delete_page(&key, 2).expect("delete"), 2);
        let batch = spool.get(&key).expect("batch");
        let sequences: Vec<u32> = batch.pages.iter().map(|p| p.sequence).collect();
        assert_eq!(sequences, [1, 2]);
        assert_eq!(
            spool.read_page(&key, &batch.pages[0]).expect("one"),
            b"%PDF one"
        );
        assert_eq!(
            spool.read_page(&key, &batch.pages[1]).expect("three"),
            b"%PDF three"
        );
        assert_eq!(batch.pages[1].rotation, 90, "the turn moves with the page");
        assert_eq!(
            spool.picture(&key, 2, PictureSize::Thumb).expect("thumb"),
            pictures(2).thumb
        );
        assert_eq!(
            spool.picture(&key, 2, PictureSize::View).expect("view"),
            pictures(2).view
        );
        assert!(matches!(
            spool.picture(&key, 3, PictureSize::Thumb),
            Err(SpoolError::NoPage(3))
        ));
        assert!(
            !dir.path()
                .join("batches")
                .join(&key)
                .join("page-0003.bin")
                .exists()
        );
    }

    #[test]
    fn a_page_already_sent_keeps_the_batch_as_it_is() {
        let dir = tempfile::tempdir().expect("dir");
        let spool = Spool::open(dir.path(), Arc::new(Reverse)).expect("spool");
        let key = Spool::new_key();
        spool.create(input(&key), "fi-8170").expect("create");
        spool
            .append_page(&key, b"%PDF one", &markers())
            .expect("page");
        spool
            .append_page(&key, b"%PDF two", &markers())
            .expect("page");
        spool.mark_uploaded(&key, 1).expect("uploaded");
        spool.hold(&key).expect("hold");
        assert!(matches!(
            spool.delete_page(&key, 2),
            Err(SpoolError::NotHeld)
        ));
        assert!(matches!(
            spool.rotate(&key, 1, 90),
            Err(SpoolError::NoPage(1))
        ));
    }

    #[test]
    fn pictures_are_kept_encrypted_and_read_from_refused_batches_too() {
        let dir = tempfile::tempdir().expect("dir");
        let spool = Spool::open(dir.path(), Arc::new(Reverse)).expect("spool");
        let key = Spool::new_key();
        spool.create(input(&key), "fi-8170").expect("create");
        spool
            .append_page(&key, b"%PDF one", &markers())
            .expect("page");
        assert!(matches!(
            spool.picture(&key, 1, PictureSize::Thumb),
            Err(SpoolError::NoPage(1))
        ));
        spool
            .store_pictures(&key, 1, &pictures(7))
            .expect("pictures");
        let on_disk = fs::read(
            dir.path()
                .join("batches")
                .join(&key)
                .join("page-0001.thumb"),
        )
        .expect("disk");
        assert_ne!(on_disk, pictures(7).thumb, "sealed at rest");

        spool.fail(&key, "refused").expect("fail");
        assert_eq!(
            spool.picture(&key, 1, PictureSize::Thumb).expect("thumb"),
            pictures(7).thumb
        );
        let too_big = PagePictures {
            thumb: vec![0; MAX_PICTURE_BYTES + 1],
            view: Vec::new(),
        };
        let other = Spool::new_key();
        spool.create(input(&other), "fi-8170").expect("create");
        spool
            .append_page(&other, b"%PDF one", &markers())
            .expect("page");
        assert!(matches!(
            spool.store_pictures(&other, 1, &too_big),
            Err(SpoolError::PictureTooLarge)
        ));
    }

    #[test]
    fn a_printed_job_keeps_the_pictures_of_its_pages() {
        let dir = tempfile::tempdir().expect("dir");
        let spool = Spool::open(dir.path(), Arc::new(Reverse)).expect("spool");
        let key = Spool::new_key();
        let batch = spool
            .create_print(print(&key, &[pictures(1), pictures(2)], true))
            .expect("print");
        assert!(batch.held);
        assert_eq!(batch.document.as_ref().map(|d| d.pictures), Some(2));
        assert_eq!(
            spool.picture(&key, 2, PictureSize::View).expect("view"),
            pictures(2).view
        );
        assert!(matches!(
            spool.picture(&key, 3, PictureSize::View),
            Err(SpoolError::NoPage(3))
        ));
        assert!(matches!(
            spool.delete_page(&key, 1),
            Err(SpoolError::NotHeld)
        ));
        spool.discard_held(&key).expect("discard");
        assert!(spool.pending().expect("pending").is_empty());
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

    fn batch_dir(dir: &Path, key: &str) -> PathBuf {
        dir.join("batches").join(key)
    }

    fn journals(dir: &Path, key: &str) -> Vec<String> {
        let mut names: Vec<String> = fs::read_dir(batch_dir(dir, key))
            .expect("dir")
            .filter_map(Result::ok)
            .filter_map(|e| e.file_name().to_str().map(str::to_owned))
            .filter(|name| name.starts_with("journal-"))
            .collect();
        names.sort();
        names
    }

    #[test]
    fn pages_are_journaled_and_replayed_after_a_restart() {
        let dir = tempfile::tempdir().expect("dir");
        let key = Spool::new_key();
        {
            let spool = Spool::open(dir.path(), Arc::new(Reverse)).expect("spool");
            spool.create(input(&key), "fi-8170").expect("create");
            for n in 0..3u8 {
                spool
                    .append_page(&key, &[b'%', n], &markers())
                    .expect("page");
            }
            spool
                .store_pictures(&key, 2, &pictures(2))
                .expect("pictures");
            spool.mark_uploaded(&key, 1).expect("uploaded");
            spool.mark_uploaded(&key, 1).expect("uploaded twice");
            let manifest =
                fs::read_to_string(batch_dir(dir.path(), &key).join(MANIFEST)).expect("manifest");
            assert!(
                manifest.contains("\"pages\": []"),
                "adding a page does not rewrite the manifest"
            );
        }

        let spool = Spool::open(dir.path(), Arc::new(Reverse)).expect("reopen");
        let batch = spool.get(&key).expect("get");
        assert_eq!(batch.pages.len(), 3);
        assert!(batch.pages[0].uploaded);
        assert!(!batch.pages[1].uploaded);
        assert!(batch.pages[1].pictures);
        assert_eq!(
            spool.read_page(&key, &batch.pages[2]).expect("read"),
            [b'%', 2]
        );
    }

    #[test]
    fn a_line_cut_short_by_a_crash_is_dropped_and_the_journal_goes_on() {
        let dir = tempfile::tempdir().expect("dir");
        let key = Spool::new_key();
        {
            let spool = Spool::open(dir.path(), Arc::new(Reverse)).expect("spool");
            spool.create(input(&key), "fi-8170").expect("create");
            spool
                .append_page(&key, b"%PDF one", &markers())
                .expect("page");
        }
        let journal = batch_dir(dir.path(), &key).join(&journals(dir.path(), &key)[0]);
        let mut file = fs::OpenOptions::new()
            .append(true)
            .open(&journal)
            .expect("open");
        file.write_all(b"{\"op\":\"uploaded\",\"seq").expect("torn");
        drop(file);

        let spool = Spool::open(dir.path(), Arc::new(Reverse)).expect("reopen");
        let batch = spool.get(&key).expect("get");
        assert_eq!(batch.pages.len(), 1);
        assert!(!batch.pages[0].uploaded, "the cut line is not applied");
        spool
            .append_page(&key, b"%PDF two", &markers())
            .expect("page");
        drop(spool);

        let spool = Spool::open(dir.path(), Arc::new(Reverse)).expect("again");
        assert_eq!(spool.get(&key).expect("get").pages.len(), 2);
        assert_eq!(
            journals(dir.path(), &key).len(),
            1,
            "the torn journal was replaced"
        );
    }

    #[test]
    fn a_journal_the_manifest_does_not_name_is_never_read() {
        let dir = tempfile::tempdir().expect("dir");
        let key = Spool::new_key();
        let spool = Spool::open(dir.path(), Arc::new(Reverse)).expect("spool");
        spool.create_held(input(&key), "fi-8170").expect("create");
        spool
            .append_page(&key, b"%PDF one", &markers())
            .expect("page");
        spool.release(&key).expect("release");
        drop(spool);

        fs::write(
            journal_path(&batch_dir(dir.path(), &key), 0),
            b"{\"op\":\"uploaded\",\"sequence\":1}\n",
        )
        .expect("stale");
        let spool = Spool::open(dir.path(), Arc::new(Reverse)).expect("reopen");
        assert!(!spool.get(&key).expect("get").pages[0].uploaded);
    }

    #[test]
    fn a_manifest_written_before_the_journal_is_read_as_it_was() {
        let dir = tempfile::tempdir().expect("dir");
        let key = Spool::new_key();
        let batch_path = batch_dir(dir.path(), &key);
        fs::create_dir_all(&batch_path).expect("dir");
        let old = SpooledBatch {
            version: MANIFEST_VERSION,
            created_at: 1,
            label: "fi-8170".to_owned(),
            input: input(&key),
            batch_id: None,
            pages: Vec::new(),
            document: None,
            complete: false,
            held: false,
        };
        fs::write(
            batch_path.join(MANIFEST),
            serde_json::to_vec(&old).expect("json"),
        )
        .expect("write");

        let spool = Spool::open(dir.path(), Arc::new(Reverse)).expect("spool");
        assert_eq!(spool.get(&key).expect("get"), old);
        spool
            .append_page(&key, b"%PDF one", &markers())
            .expect("page");
        assert_eq!(spool.get(&key).expect("get").pages.len(), 1);
    }

    #[test]
    fn a_long_journal_is_folded_into_the_manifest() {
        let dir = tempfile::tempdir().expect("dir");
        let key = Spool::new_key();
        let pages = u32::try_from(JOURNAL_COMPACT_AFTER).expect("fits") + 5;
        {
            let spool = Spool::open(dir.path(), Arc::new(Reverse)).expect("spool");
            spool.create(input(&key), "fi-8170").expect("create");
            for n in 0..pages {
                spool
                    .append_page(&key, n.to_le_bytes().as_slice(), &PageMarkers::default())
                    .expect("page");
            }
            assert_eq!(
                journals(dir.path(), &key).len(),
                1,
                "the folded journal is removed"
            );
        }
        let spool = Spool::open(dir.path(), Arc::new(Reverse)).expect("reopen");
        let batch = spool.get(&key).expect("get");
        assert_eq!(batch.pages.len(), usize::try_from(pages).expect("fits"));
        assert!(
            batch
                .pages
                .iter()
                .enumerate()
                .all(|(i, p)| p.sequence as usize == i + 1),
            "in order"
        );
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

    fn refuse(spool: &Spool, dir: &Path, label: &str, pages: &[&[u8]], at_secs: u64) -> String {
        let key = Spool::new_key();
        spool.create(input(&key), label).expect("create");
        for page in pages {
            spool.append_page(&key, page, &markers()).expect("page");
        }
        spool.set_batch_id(&key, Id::from("cbat_old")).expect("id");
        spool.mark_uploaded(&key, 1).expect("uploaded");
        spool.complete(&key).expect("complete");
        spool.fail(&key, "the batch was ended\n").expect("fail");
        let failure = fs::File::options()
            .write(true)
            .open(dir.join("failed").join(&key).join("failure.txt"))
            .expect("failure file");
        failure
            .set_modified(UNIX_EPOCH + std::time::Duration::from_secs(at_secs))
            .expect("mtime");
        key
    }

    #[test]
    fn refused_batches_are_listed_most_recent_first_with_their_reason() {
        let dir = tempfile::tempdir().expect("dir");
        let spool = Spool::open(dir.path(), Arc::new(Reverse)).expect("spool");
        let older = refuse(&spool, dir.path(), "fi-8170", &[b"%PDF 1"], 1_000);
        let newer = refuse(
            &spool,
            dir.path(),
            "ScanSnap",
            &[b"%PDF 1", b"%PDF 2"],
            2_000,
        );

        let refused = spool.refused().expect("refused");
        assert_eq!(
            refused.iter().map(|r| r.key.as_str()).collect::<Vec<_>>(),
            [newer.as_str(), older.as_str()]
        );
        assert_eq!(refused[0].label, "ScanSnap");
        assert_eq!(refused[0].pages, 2);
        assert_eq!(refused[0].reason, "the batch was ended");
        assert_eq!(refused[0].refused_at, 2_000_000);
        assert_eq!(refused[0].source, BatchSource::Scan);
        assert!(refused[0].readable);
        assert!(spool.pending().expect("pending").is_empty());
    }

    #[test]
    fn a_retried_batch_goes_back_to_intake_as_a_new_batch_with_every_page_to_send() {
        let dir = tempfile::tempdir().expect("dir");
        let spool = Spool::open(dir.path(), Arc::new(Reverse)).expect("spool");
        let key = refuse(
            &spool,
            dir.path(),
            "fi-8170",
            &[b"%PDF 1", b"%PDF 2"],
            1_000,
        );

        let new_key = spool.retry(&key).expect("retry");

        assert_ne!(new_key, key, "the old key names a batch the server closed");
        assert!(spool.refused().expect("refused").is_empty());
        let pending = spool.pending().expect("pending");
        assert_eq!(pending.len(), 1);
        let batch = &pending[0];
        assert_eq!(batch.key(), new_key);
        assert_eq!(batch.batch_id, None);
        assert_eq!(
            batch.input.request_id, None,
            "a retried batch goes to intake"
        );
        assert!(batch.complete);
        assert_eq!(batch.pages_waiting(), 2);
        assert_eq!(
            spool.read_page(&new_key, &batch.pages[1]).expect("page"),
            b"%PDF 2"
        );
        assert!(
            !dir.path()
                .join("batches")
                .join(&new_key)
                .join("failure.txt")
                .exists(),
            "the old reason does not travel with it"
        );
    }

    #[test]
    fn a_discarded_batch_is_gone_for_good() {
        let dir = tempfile::tempdir().expect("dir");
        let spool = Spool::open(dir.path(), Arc::new(Reverse)).expect("spool");
        let key = refuse(&spool, dir.path(), "fi-8170", &[b"%PDF 1"], 1_000);

        spool.discard(&key).expect("discard");

        assert!(spool.refused().expect("refused").is_empty());
        assert!(!dir.path().join("failed").join(&key).exists());
        assert!(matches!(spool.discard(&key), Err(SpoolError::Missing(_))));
    }

    #[test]
    fn saving_a_refused_batch_writes_its_pages_and_leaves_out_one_that_changed() {
        let dir = tempfile::tempdir().expect("dir");
        let spool = Spool::open(dir.path(), Arc::new(Reverse)).expect("spool");
        let key = refuse(
            &spool,
            dir.path(),
            "fi-8170",
            &[b"%PDF 1", b"%PDF 2", b"%PDF 3"],
            1_000,
        );
        fs::write(
            dir.path().join("failed").join(&key).join("page-0002.bin"),
            b"tampered",
        )
        .expect("tamper");
        let out = tempfile::tempdir().expect("out");
        let into = out.path().join("fi-8170 scan");

        let export = spool.export(&key, &into).expect("export");

        assert_eq!(
            export.written,
            [into.join("page-0001.pdf"), into.join("page-0003.pdf")]
        );
        assert_eq!(export.unreadable, [2]);
        assert_eq!(
            fs::read(into.join("page-0003.pdf")).expect("page"),
            b"%PDF 3"
        );
        assert!(
            spool.refused().expect("refused").len() == 1,
            "saving a copy keeps the batch until it is discarded"
        );
        assert!(matches!(
            spool.export(&key, &into),
            Err(SpoolError::Exists(_))
        ));
    }

    #[test]
    fn a_saved_print_is_its_document() {
        let dir = tempfile::tempdir().expect("dir");
        let spool = Spool::open(dir.path(), Arc::new(Reverse)).expect("spool");
        let key = Spool::new_key();
        spool.create_print(print(&key, &[], false)).expect("print");
        spool.fail(&key, "ended").expect("fail");

        let refused = spool.refused().expect("refused");
        assert_eq!(refused[0].source, BatchSource::Print);
        assert_eq!(refused[0].pages, 3);

        let out = tempfile::tempdir().expect("out");
        let into = out.path().join("print");
        let export = spool.export(&key, &into).expect("export");
        assert_eq!(export.written, [into.join("document.pdf")]);
        assert_eq!(
            fs::read(into.join("document.pdf")).expect("doc"),
            b"%PDF printed"
        );
    }

    #[test]
    fn a_refused_batch_whose_manifest_is_unreadable_is_listed_but_cannot_be_sent_or_saved() {
        let dir = tempfile::tempdir().expect("dir");
        let spool = Spool::open(dir.path(), Arc::new(Reverse)).expect("spool");
        let key = refuse(&spool, dir.path(), "fi-8170", &[b"%PDF 1"], 1_000);
        fs::write(
            dir.path().join("failed").join(&key).join("manifest.json"),
            b"{",
        )
        .expect("break");

        let refused = spool.refused().expect("refused");
        assert_eq!(refused.len(), 1);
        assert!(!refused[0].readable);
        assert_eq!(refused[0].reason, "the batch was ended");
        assert!(matches!(spool.retry(&key), Err(SpoolError::Manifest(_))));
        let out = tempfile::tempdir().expect("out");
        assert!(matches!(
            spool.export(&key, &out.path().join("x")),
            Err(SpoolError::Manifest(_))
        ));
        spool
            .discard(&key)
            .expect("an unreadable batch can still be discarded");
    }

    #[test]
    fn refused_keys_cannot_escape_the_spool() {
        let dir = tempfile::tempdir().expect("dir");
        let spool = Spool::open(dir.path(), Arc::new(Reverse)).expect("spool");
        for key in ["../batches", "..", "C:\\Windows"] {
            assert!(matches!(spool.retry(key), Err(SpoolError::Missing(_))));
            assert!(matches!(spool.discard(key), Err(SpoolError::Missing(_))));
        }
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
