//! Running scans in helper processes.
//!
//! Every scan, and every listing of scanners, runs in a
//! `trenova-capture-scan` process of the driver's bitness: the 64-bit helper
//! for 64-bit TWAIN sources and WIA, the 32-bit one for 32-bit TWAIN
//! sources, which a 64-bit process cannot load. A driver that crashes takes
//! the helper with it; the agent keeps every page it already received.

use std::future::Future;
use std::io::{BufRead, BufReader, Read};
use std::path::{Path, PathBuf};
use std::pin::Pin;
use std::process::{Child, ChildStdin, Command, Stdio};
use std::sync::{Arc, Mutex, PoisonError};
use std::time::Duration;

use capture_protocol::api::{RequestFailureCode, Settings, SourceInfo, SourceProtocol};
use capture_protocol::helper::{
    Frame, HelperCommand, HelperEvent, PageMeta, ScanCondition, ScanJob, read_frame, write_message,
};
use tokio::sync::mpsc;
use tokio_util::sync::CancellationToken;

/// How long a listing may take before the helper is stopped. A driver that
/// hangs on enumeration must not stall the agent.
const ENUMERATE_TIMEOUT: Duration = Duration::from_secs(60);
/// How long a helper asked to cancel gets before it is stopped.
const CANCEL_GRACE: Duration = Duration::from_secs(15);

/// What a running scan reports.
#[derive(Debug)]
pub enum ScanUpdate {
    Described(SourceInfo),
    Started(Settings),
    Page { meta: PageMeta, pdf: Vec<u8> },
    End(ScanOutcome),
}

/// How a scan ended.
#[derive(Clone, Debug, PartialEq, Eq)]
pub enum ScanOutcome {
    Finished,
    /// The scanner stopped partway; the pages before stand.
    Stopped {
        condition: ScanCondition,
        message: String,
    },
    Failed {
        code: RequestFailureCode,
        message: String,
    },
}

/// A scan in progress.
#[derive(Debug)]
pub struct ScanRun {
    pub updates: mpsc::Receiver<ScanUpdate>,
    /// Cancel to stop at the next page boundary.
    pub cancel: CancellationToken,
}

pub type SourcesFuture<'a> = Pin<Box<dyn Future<Output = Vec<SourceInfo>> + Send + 'a>>;

/// Something that can list scanners and scan from them.
pub trait ScannerHost: Send + Sync {
    fn enumerate(&self) -> SourcesFuture<'_>;
    fn scan(&self, source: &SourceInfo, job: ScanJob) -> ScanRun;
}

/// Scans through the helper executables next to the agent.
#[derive(Clone, Debug)]
pub struct HelperHost {
    dir: PathBuf,
}

/// Which helper runs a source.
pub fn helper_bitness(source: &SourceInfo) -> u8 {
    match source.protocol {
        SourceProtocol::Twain if source.bitness == 32 => 32,
        _ => 64,
    }
}

impl HelperHost {
    /// Helpers are looked for in `dir`, which is the agent's own directory.
    pub fn new(dir: impl Into<PathBuf>) -> Self {
        Self { dir: dir.into() }
    }

    /// The helper for a bitness. An installed agent has
    /// `trenova-capture-scan-x64.exe` and `-x86.exe` beside it; a development
    /// build has one `trenova-capture-scan` of its own bitness.
    fn helper(&self, bitness: u8) -> Option<PathBuf> {
        let arch = if bitness == 32 { "x86" } else { "x64" };
        let installed = self.dir.join(format!(
            "trenova-capture-scan-{arch}{}",
            std::env::consts::EXE_SUFFIX
        ));
        if installed.is_file() {
            return Some(installed);
        }
        let native = if cfg!(target_pointer_width = "64") {
            64
        } else {
            32
        };
        let development = self.dir.join(format!(
            "trenova-capture-scan{}",
            std::env::consts::EXE_SUFFIX
        ));
        (bitness == native && development.is_file()).then_some(development)
    }

    fn spawn(path: &Path, command: &HelperCommand) -> std::io::Result<Child> {
        let mut process = Command::new(path);
        process
            .stdin(Stdio::piped())
            .stdout(Stdio::piped())
            .stderr(Stdio::piped());
        #[cfg(windows)]
        {
            use std::os::windows::process::CommandExt;
            const CREATE_NO_WINDOW: u32 = 0x0800_0000;
            process.creation_flags(CREATE_NO_WINDOW);
        }
        let mut child = process.spawn()?;
        let mut stdin = child
            .stdin
            .take()
            .ok_or_else(|| std::io::Error::other("no helper input"))?;
        write_message(&mut stdin, command).map_err(std::io::Error::other)?;
        child.stdin = Some(stdin);
        if let Some(stderr) = child.stderr.take() {
            forward_log(stderr);
        }
        Ok(child)
    }

    fn list(path: &Path) -> Vec<SourceInfo> {
        let mut child = match Self::spawn(path, &HelperCommand::Enumerate) {
            Ok(child) => child,
            Err(err) => {
                tracing::error!(helper = %path.display(), error = %err, "could not start the scan helper");
                return Vec::new();
            }
        };
        let Some(mut stdout) = child.stdout.take() else {
            return Vec::new();
        };
        let child = Arc::new(Mutex::new(child));
        let watchdog = {
            let child = Arc::clone(&child);
            let (done, finished) = std::sync::mpsc::channel::<()>();
            std::thread::spawn(move || {
                if finished.recv_timeout(ENUMERATE_TIMEOUT).is_err() {
                    tracing::warn!("listing scanners took too long; stopping the helper");
                    let _ = child.lock().unwrap_or_else(PoisonError::into_inner).kill();
                }
            });
            done
        };
        let mut sources = Vec::new();
        loop {
            match read_frame::<_, HelperEvent>(&mut stdout) {
                Ok(Some(Frame::Message(HelperEvent::Sources { sources: listed }))) => {
                    sources = listed;
                    break;
                }
                Ok(Some(_)) => {}
                Ok(None) => break,
                Err(err) => {
                    tracing::warn!(error = %err, "unreadable scanner list");
                    break;
                }
            }
        }
        let _ = watchdog.send(());
        let _ = child.lock().unwrap_or_else(PoisonError::into_inner).wait();
        sources
    }
}

/// Copies a helper's log lines into the agent's log.
fn forward_log(stderr: impl Read + Send + 'static) {
    std::thread::spawn(move || {
        for line in BufReader::new(stderr).lines().map_while(Result::ok) {
            tracing::info!(target: "trenova_capture::helper", "{line}");
        }
    });
}

/// Merges the lists from both helpers, keeping one entry per scanner. A
/// TWAIN driver installed in both bitnesses is used through the 64-bit one.
pub fn merge_sources(lists: impl IntoIterator<Item = Vec<SourceInfo>>) -> Vec<SourceInfo> {
    let mut merged: Vec<SourceInfo> = Vec::new();
    for source in lists.into_iter().flatten() {
        match merged
            .iter_mut()
            .find(|s| s.name == source.name && s.protocol == source.protocol)
        {
            Some(existing) if source.bitness > existing.bitness => *existing = source,
            Some(_) => {}
            None => merged.push(source),
        }
    }
    merged
}

impl ScannerHost for HelperHost {
    fn enumerate(&self) -> SourcesFuture<'_> {
        let helpers: Vec<PathBuf> = [64, 32]
            .into_iter()
            .filter_map(|b| self.helper(b))
            .collect();
        Box::pin(async move {
            let mut lists = Vec::new();
            for path in helpers {
                match tokio::task::spawn_blocking(move || Self::list(&path)).await {
                    Ok(list) => lists.push(list),
                    Err(err) => tracing::error!(error = %err, "the scanner listing panicked"),
                }
            }
            merge_sources(lists)
        })
    }

    fn scan(&self, source: &SourceInfo, job: ScanJob) -> ScanRun {
        let (tx, updates) = mpsc::channel(8);
        let cancel = CancellationToken::new();
        let run = ScanRun {
            updates,
            cancel: cancel.clone(),
        };

        let Some(path) = self.helper(helper_bitness(source)) else {
            let _ = tx.try_send(ScanUpdate::End(ScanOutcome::Failed {
                code: RequestFailureCode::SourceUnavailable,
                message: "The scan helper for this scanner is not installed.".into(),
            }));
            return run;
        };
        let mut child = match Self::spawn(&path, &HelperCommand::Scan(job)) {
            Ok(child) => child,
            Err(err) => {
                let _ = tx.try_send(ScanUpdate::End(ScanOutcome::Failed {
                    code: RequestFailureCode::Internal,
                    message: format!("The scan helper could not start: {err}"),
                }));
                return run;
            }
        };
        let stdout = child.stdout.take();
        let stdin: Arc<Mutex<Option<ChildStdin>>> = Arc::new(Mutex::new(child.stdin.take()));
        let child = Arc::new(Mutex::new(child));

        spawn_canceller(cancel, Arc::clone(&stdin), Arc::clone(&child));
        std::thread::spawn(move || {
            let outcome = match stdout {
                Some(stdout) => relay(stdout, &tx),
                None => None,
            };
            let outcome = outcome.unwrap_or_else(|| ScanOutcome::Failed {
                code: RequestFailureCode::DriverError,
                message: "The scanner driver stopped unexpectedly.".into(),
            });
            drop(stdin.lock().unwrap_or_else(PoisonError::into_inner).take());
            let _ = child.lock().unwrap_or_else(PoisonError::into_inner).wait();
            let _ = tx.blocking_send(ScanUpdate::End(outcome));
        });
        run
    }
}

/// On cancel, asks the helper to stop, and stops it if it has not within the
/// grace period.
fn spawn_canceller(
    cancel: CancellationToken,
    stdin: Arc<Mutex<Option<ChildStdin>>>,
    child: Arc<Mutex<Child>>,
) {
    tokio::spawn(async move {
        cancel.cancelled().await;
        if let Some(input) = stdin
            .lock()
            .unwrap_or_else(PoisonError::into_inner)
            .as_mut()
        {
            let _ = write_message(input, &HelperCommand::Cancel);
        }
        tokio::time::sleep(CANCEL_GRACE).await;
        let mut child = child.lock().unwrap_or_else(PoisonError::into_inner);
        if matches!(child.try_wait(), Ok(None)) {
            tracing::warn!("the scan helper did not stop; ending it");
            let _ = child.kill();
        }
    });
}

/// Passes a helper's events on, returning how the scan ended, or `None` if
/// the helper went away without saying.
fn relay(mut stdout: impl Read, tx: &mpsc::Sender<ScanUpdate>) -> Option<ScanOutcome> {
    loop {
        let event = match read_frame::<_, HelperEvent>(&mut stdout) {
            Ok(Some(Frame::Message(event))) => event,
            Ok(Some(Frame::Bytes(_))) => {
                tracing::warn!("the scan helper sent page bytes without a page");
                continue;
            }
            Ok(None) => return None,
            Err(err) => {
                tracing::error!(error = %err, "the scan helper's output is unreadable");
                return None;
            }
        };
        let update = match event {
            HelperEvent::Page(meta) => match read_frame::<_, HelperEvent>(&mut stdout) {
                Ok(Some(Frame::Bytes(pdf))) => ScanUpdate::Page { meta, pdf },
                _ => return None,
            },
            HelperEvent::Described { source } => ScanUpdate::Described(source),
            HelperEvent::Started { settings } => ScanUpdate::Started(settings),
            HelperEvent::Finished { .. } => return Some(ScanOutcome::Finished),
            HelperEvent::Condition { condition, message } => {
                return Some(ScanOutcome::Stopped { condition, message });
            }
            HelperEvent::Failed { code, message } => {
                return Some(ScanOutcome::Failed { code, message });
            }
            HelperEvent::Sources { .. } => continue,
        };
        if tx.blocking_send(update).is_err() {
            return None;
        }
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use capture_protocol::api::PixelType;
    use capture_protocol::helper::write_bytes;

    fn source(name: &str, protocol: SourceProtocol, bitness: u8) -> SourceInfo {
        SourceInfo {
            name: name.into(),
            protocol,
            bitness,
            is_default: false,
            duplex: false,
            feeder: false,
            patch_codes: false,
            barcodes: false,
            blank_discard: false,
            resolutions: Vec::new(),
            pixel_types: Vec::new(),
        }
    }

    #[test]
    fn one_entry_per_scanner_preferring_the_64_bit_driver() {
        let merged = merge_sources([
            vec![
                source("fi-8170", SourceProtocol::Twain, 64),
                source("fi-8170", SourceProtocol::Wia, 64),
            ],
            vec![
                source("fi-8170", SourceProtocol::Twain, 32),
                source("DR-C225", SourceProtocol::Twain, 32),
            ],
        ]);
        assert_eq!(merged.len(), 3);
        assert_eq!(merged[0].bitness, 64);
        assert_eq!(helper_bitness(&merged[2]), 32);
        assert_eq!(helper_bitness(&merged[1]), 64);
    }

    #[test]
    fn a_helpers_output_is_relayed_page_by_page_to_its_end() {
        let mut pipe = Vec::new();
        write_message(
            &mut pipe,
            &HelperEvent::Started {
                settings: Settings::default(),
            },
        )
        .expect("frame");
        let meta = PageMeta {
            index: 1,
            width_px: 10,
            height_px: 10,
            dpi: 300,
            pixel_type: PixelType::BlackWhite,
            patch_code: None,
            barcodes: Vec::new(),
        };
        write_message(&mut pipe, &HelperEvent::Page(meta.clone())).expect("frame");
        write_bytes(&mut pipe, b"%PDF").expect("frame");
        write_message(
            &mut pipe,
            &HelperEvent::Condition {
                condition: ScanCondition::PaperJam,
                message: "jam".into(),
            },
        )
        .expect("frame");

        let (tx, mut rx) = mpsc::channel(8);
        let outcome = std::thread::spawn(move || relay(pipe.as_slice(), &tx))
            .join()
            .expect("relay");
        assert!(matches!(rx.try_recv(), Ok(ScanUpdate::Started(_))));
        assert!(matches!(rx.try_recv(), Ok(ScanUpdate::Page { pdf, .. }) if pdf == b"%PDF"));
        assert_eq!(
            outcome,
            Some(ScanOutcome::Stopped {
                condition: ScanCondition::PaperJam,
                message: "jam".into()
            })
        );
    }

    #[test]
    fn a_helper_that_dies_mid_page_reports_nothing_it_did_not_finish() {
        let mut pipe = Vec::new();
        write_message(
            &mut pipe,
            &HelperEvent::Page(PageMeta {
                index: 1,
                width_px: 1,
                height_px: 1,
                dpi: 300,
                pixel_type: PixelType::BlackWhite,
                patch_code: None,
                barcodes: Vec::new(),
            }),
        )
        .expect("frame");
        let (tx, mut rx) = mpsc::channel(8);
        let outcome = std::thread::spawn(move || relay(pipe.as_slice(), &tx))
            .join()
            .expect("relay");
        assert_eq!(outcome, None);
        assert!(rx.try_recv().is_err());
    }
}
