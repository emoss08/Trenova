//! The helper's one job, on Windows.

use std::io::{self, Read, Stdout, Write};
use std::process::ExitCode;
use std::sync::atomic::{AtomicBool, Ordering};
use std::sync::{Arc, Mutex, PoisonError};

use capture_imaging::scan::{ScanEnd, ScanSettings, ScannedPage};
use capture_imaging::{PixelFormat, binarize, encode_page};
use capture_protocol::api::{PixelType, RequestFailureCode, Settings, SourceInfo, SourceProtocol};
use capture_protocol::helper::{
    Frame, HelperCommand, HelperEvent, PageMeta, ScanCondition, ScanJob, read_frame, write_bytes,
    write_message,
};
use capture_twain::win::{Canceller, LibraryDsm, WindowPump};
use capture_twain::{AppIdentity, Manager, TwainError};

const BITNESS: u8 = if cfg!(target_pointer_width = "64") {
    64
} else {
    32
};

/// The pipe back to the agent. Frames are written whole under the lock, so a
/// page's meta and bytes are never split by another message.
#[derive(Clone)]
struct Pipe(Arc<Mutex<Stdout>>);

impl Pipe {
    fn send(&self, event: &HelperEvent) -> Result<(), String> {
        let mut out = self.0.lock().unwrap_or_else(PoisonError::into_inner);
        write_message(&mut *out, event).map_err(|e| e.to_string())
    }

    fn send_page(&self, meta: PageMeta, pdf: &[u8]) -> Result<(), String> {
        let mut out = self.0.lock().unwrap_or_else(PoisonError::into_inner);
        write_message(&mut *out, &HelperEvent::Page(meta)).map_err(|e| e.to_string())?;
        write_bytes(&mut *out, pdf).map_err(|e| e.to_string())?;
        out.flush().map_err(|e| e.to_string())
    }

    fn fail(&self, code: RequestFailureCode, message: impl Into<String>) {
        let message = message.into();
        tracing::error!(?code, message, "the scan failed");
        if let Err(err) = self.send(&HelperEvent::Failed { code, message }) {
            tracing::error!(error = %err, "could not tell the agent");
        }
    }
}

fn app() -> AppIdentity {
    let version = env!("CARGO_PKG_VERSION");
    let mut parts = version.split('.').map(|p| p.parse::<u16>().unwrap_or(0));
    AppIdentity {
        version: version.to_owned(),
        major: parts.next().unwrap_or(1),
        minor: parts.next().unwrap_or(0),
    }
}

pub fn run() -> ExitCode {
    let pipe = Pipe(Arc::new(Mutex::new(io::stdout())));
    let mut stdin = io::stdin();
    let command = match read_frame::<_, HelperCommand>(&mut stdin) {
        Ok(Some(Frame::Message(command))) => command,
        Ok(_) => {
            tracing::error!("the agent sent no command");
            return ExitCode::from(2);
        }
        Err(err) => {
            tracing::error!(error = %err, "unreadable command");
            return ExitCode::from(2);
        }
    };

    match command {
        HelperCommand::Enumerate => enumerate(&pipe),
        HelperCommand::Scan(job) => scan(&job, &pipe, stdin),
        HelperCommand::Cancel => {}
    }
    ExitCode::SUCCESS
}

fn enumerate(pipe: &Pipe) {
    let mut sources = match twain_sources() {
        Ok(sources) => sources,
        Err(err) => {
            tracing::warn!(error = %err, "no TWAIN sources");
            Vec::new()
        }
    };
    if BITNESS == 64 {
        match capture_wia::sources() {
            Ok(wia) => sources.extend(wia),
            Err(err) => tracing::warn!(error = %err, "no WIA sources"),
        }
    }
    if let Err(err) = pipe.send(&HelperEvent::Sources { sources }) {
        tracing::error!(error = %err, "could not send the sources");
    }
}

/// TWAIN sources as the DSM lists them. What each can do is learned when it
/// is opened for a scan, because opening an offline source can show its
/// driver's error dialog.
fn twain_sources() -> Result<Vec<SourceInfo>, TwainError> {
    let pump = WindowPump::new().map_err(|e| TwainError::Load(e.message()))?;
    let manager = Manager::open(LibraryDsm::load()?, &app(), pump.parent())?;
    Ok(manager
        .sources()?
        .into_iter()
        .map(|entry| SourceInfo {
            name: entry.name,
            protocol: SourceProtocol::Twain,
            bitness: BITNESS,
            is_default: entry.is_default,
            duplex: false,
            feeder: false,
            patch_codes: false,
            barcodes: false,
            blank_discard: false,
            resolutions: Vec::new(),
            pixel_types: Vec::new(),
        })
        .collect())
}

/// Watches standard input for a cancel (or the agent going away) while the
/// scan runs.
fn watch_for_cancel(
    mut stdin: impl Read + Send + 'static,
    cancel: Arc<AtomicBool>,
    canceller: Option<Canceller>,
) {
    std::thread::spawn(move || {
        loop {
            match read_frame::<_, HelperCommand>(&mut stdin) {
                Ok(Some(Frame::Message(HelperCommand::Cancel)) | None) | Err(_) => break,
                Ok(Some(_)) => {}
            }
        }
        cancel.store(true, Ordering::Release);
        if let Some(canceller) = canceller {
            canceller.cancel();
        }
    });
}

/// Encodes and sends one page, reducing it to black and white first when the
/// profile asked for that and the source sent more.
fn deliver(
    pipe: &Pipe,
    page: ScannedPage,
    index: u32,
    want: &ScanSettings,
    quality: u8,
) -> Result<(), String> {
    let raster = if want.pixel_type == PixelType::BlackWhite
        && !matches!(page.raster.format, PixelFormat::Bilevel { .. })
    {
        binarize(&page.raster.as_raster().map_err(|e| e.to_string())?)
    } else {
        page.raster
    };
    let view = raster.as_raster().map_err(|e| e.to_string())?;
    let encoded = encode_page(&view, page.resolution, quality).map_err(|e| e.to_string())?;
    let meta = PageMeta {
        index,
        width_px: encoded.width_px,
        height_px: encoded.height_px,
        dpi: encoded.resolution.x,
        pixel_type: if encoded.bilevel {
            PixelType::BlackWhite
        } else if matches!(raster.format, PixelFormat::Gray8) {
            PixelType::Grayscale
        } else {
            PixelType::Color
        },
        patch_code: page.patch_code.map(str::to_owned),
        barcodes: page.barcodes,
    };
    pipe.send_page(meta, &encoded.pdf)
}

fn condition_message(condition: ScanCondition) -> &'static str {
    match condition {
        ScanCondition::PaperJam => "The scanner reported a paper jam.",
        ScanCondition::DoubleFeed => "The scanner picked up more than one sheet at once.",
        ScanCondition::CoverOpen => "The scanner's cover is open.",
        ScanCondition::FeederEmpty => "There is no paper in the scanner.",
        ScanCondition::CanceledByOperator => "Scanning was cancelled at the scanner.",
    }
}

fn report_end(pipe: &Pipe, end: ScanEnd) -> Result<(), String> {
    match end {
        ScanEnd::Finished { pages } | ScanEnd::Canceled { pages } => {
            pipe.send(&HelperEvent::Finished { pages })
        }
        ScanEnd::Stopped { condition, .. } => pipe.send(&HelperEvent::Condition {
            condition,
            message: condition_message(condition).to_owned(),
        }),
    }
}

fn scan(job: &ScanJob, pipe: &Pipe, stdin: io::Stdin) {
    let want = ScanSettings::from(job);
    let cancel = Arc::new(AtomicBool::new(false));
    let result = match job.source.protocol {
        SourceProtocol::Twain => scan_twain(job, &want, pipe, stdin, &cancel),
        SourceProtocol::Wia => scan_wia(job, &want, pipe, stdin, &cancel),
        SourceProtocol::Unknown => Err((
            RequestFailureCode::SourceUnavailable,
            "an unknown kind of scanner".to_owned(),
        )),
    };
    if let Err((code, message)) = result {
        pipe.fail(code, message);
    }
}

type Failure = (RequestFailureCode, String);

fn twain_failure(err: &TwainError) -> Failure {
    (err.failure_code(), err.to_string())
}

fn scan_twain(
    job: &ScanJob,
    want: &ScanSettings,
    pipe: &Pipe,
    stdin: io::Stdin,
    cancel: &Arc<AtomicBool>,
) -> Result<(), Failure> {
    let mut pump = WindowPump::new().map_err(|e| (RequestFailureCode::Internal, e.message()))?;
    watch_for_cancel(stdin, Arc::clone(cancel), Some(pump.canceller()));

    let dsm = LibraryDsm::load().map_err(|e| twain_failure(&e))?;
    let mut manager = Manager::open(dsm, &app(), pump.parent()).map_err(|e| twain_failure(&e))?;
    let mut source = manager
        .open_source(&job.source.name)
        .map_err(|e| twain_failure(&e))?;

    let described = source.describe(false, BITNESS);
    pipe.send(&HelperEvent::Described { source: described })
        .map_err(|e| (RequestFailureCode::Internal, e))?;
    let negotiated = source
        .negotiate(want, BITNESS)
        .map_err(|e| twain_failure(&e))?;
    pipe.send(&HelperEvent::Started {
        settings: negotiated.settings,
    })
    .map_err(|e| (RequestFailureCode::Internal, e))?;

    let mut index = 0u32;
    let end = source
        .scan(want, &mut pump, cancel, &mut |page| {
            index += 1;
            deliver(pipe, page, index, want, job.jpeg_quality)
        })
        .map_err(|e| twain_failure(&e))?;
    report_end(pipe, end).map_err(|e| (RequestFailureCode::Internal, e))
}

fn scan_wia(
    job: &ScanJob,
    want: &ScanSettings,
    pipe: &Pipe,
    stdin: io::Stdin,
    cancel: &Arc<AtomicBool>,
) -> Result<(), Failure> {
    watch_for_cancel(stdin, Arc::clone(cancel), None);
    let sink_pipe = pipe.clone();
    let sink_want = *want;
    let quality = job.jpeg_quality;
    let mut index = 0u32;
    let sink = Box::new(move |page: ScannedPage| {
        index += 1;
        deliver(&sink_pipe, page, index, &sink_want, quality)
    });
    let mut started = |settings: &Settings| {
        pipe.send(&HelperEvent::Started {
            settings: settings.clone(),
        })
    };
    let end = capture_wia::scan(&job.source.name, want, cancel, &mut started, sink)
        .map_err(|e| (e.failure_code(), e.to_string()))?;
    report_end(pipe, end).map_err(|e| (RequestFailureCode::Internal, e))
}
