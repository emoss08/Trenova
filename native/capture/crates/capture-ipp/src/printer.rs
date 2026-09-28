//! The Trenova printer: the IPP operations Windows' IPP Class Driver uses.
//!
//! It answers `Get-Printer-Attributes` with an IPP Everywhere-style
//! description (PDF and PWG raster, letter, legal and A4), takes documents by
//! `Print-Job` or `Create-Job` plus `Send-Document`, and reports jobs through
//! `Get-Job-Attributes` and `Get-Jobs`. What happens to a document is the
//! [`JobHandler`]'s business: the service attributes it to a Windows user and
//! hands it to that user's agent. A job name is never shown to anyone asking
//! about jobs, because any local process can ask.

use std::collections::VecDeque;
use std::sync::{Mutex, PoisonError};
use std::time::{Duration, Instant};

use crate::codec::{Message, decode, encode};
use crate::value::{Attribute, Group, GroupTag, Resolution, Value};

/// Operation IDs (RFC 8011 §5.4.15).
pub mod op {
    pub const PRINT_JOB: u16 = 0x0002;
    pub const VALIDATE_JOB: u16 = 0x0004;
    pub const CREATE_JOB: u16 = 0x0005;
    pub const SEND_DOCUMENT: u16 = 0x0006;
    pub const CANCEL_JOB: u16 = 0x0008;
    pub const GET_JOB_ATTRIBUTES: u16 = 0x0009;
    pub const GET_JOBS: u16 = 0x000A;
    pub const GET_PRINTER_ATTRIBUTES: u16 = 0x000B;
}

/// Status codes (RFC 8011 §B, PWG 5100.x).
pub mod status {
    pub const SUCCESSFUL_OK: u16 = 0x0000;
    pub const CLIENT_ERROR_BAD_REQUEST: u16 = 0x0400;
    pub const CLIENT_ERROR_NOT_AUTHORIZED: u16 = 0x0403;
    pub const CLIENT_ERROR_NOT_POSSIBLE: u16 = 0x0404;
    pub const CLIENT_ERROR_NOT_FOUND: u16 = 0x0406;
    pub const CLIENT_ERROR_REQUEST_ENTITY_TOO_LARGE: u16 = 0x0409;
    pub const CLIENT_ERROR_DOCUMENT_FORMAT_NOT_SUPPORTED: u16 = 0x040A;
    pub const CLIENT_ERROR_COMPRESSION_NOT_SUPPORTED: u16 = 0x040F;
    pub const CLIENT_ERROR_DOCUMENT_FORMAT_ERROR: u16 = 0x0410;
    pub const SERVER_ERROR_INTERNAL_ERROR: u16 = 0x0500;
    pub const SERVER_ERROR_OPERATION_NOT_SUPPORTED: u16 = 0x0501;
    pub const SERVER_ERROR_SERVICE_UNAVAILABLE: u16 = 0x0502;
    pub const SERVER_ERROR_VERSION_NOT_SUPPORTED: u16 = 0x0503;
    pub const SERVER_ERROR_MULTIPLE_DOCUMENT_JOBS_NOT_SUPPORTED: u16 = 0x0509;
}

/// `job-state` values (RFC 8011 §5.3.7).
mod state {
    pub const PENDING: i32 = 3;
    pub const CANCELED: i32 = 7;
    pub const ABORTED: i32 = 8;
    pub const COMPLETED: i32 = 9;
}

/// How long a job created without its document waits for it.
const PENDING_TIMEOUT: Duration = Duration::from_secs(300);
/// How many finished jobs are remembered for `Get-Jobs`.
const REMEMBERED_JOBS: usize = 64;

/// The formats this printer takes.
#[derive(Clone, Copy, Debug, PartialEq, Eq)]
pub enum DocumentFormat {
    Pdf,
    PwgRaster,
}

impl DocumentFormat {
    pub const PDF: &'static str = "application/pdf";
    pub const PWG_RASTER: &'static str = "image/pwg-raster";

    fn from_mime(mime: &str) -> Option<Self> {
        match mime.trim().to_ascii_lowercase().as_str() {
            Self::PDF => Some(Self::Pdf),
            Self::PWG_RASTER => Some(Self::PwgRaster),
            _ => None,
        }
    }

    /// For `application/octet-stream`, the format the data itself shows.
    fn sniff(data: &[u8]) -> Option<Self> {
        if data.starts_with(b"%PDF-") {
            Some(Self::Pdf)
        } else if data.starts_with(b"RaS2") {
            Some(Self::PwgRaster)
        } else {
            None
        }
    }
}

/// A document the printer received.
#[derive(Clone, Debug, PartialEq, Eq)]
pub struct IncomingJob {
    pub job_id: u32,
    /// The application's name for it, as the job-name attribute carried it.
    pub name: String,
    /// The `requesting-user-name` the client claimed. A hint only: any local
    /// process can claim anything.
    pub requesting_user: Option<String>,
    pub format: DocumentFormat,
    pub data: Vec<u8>,
}

/// Why a document was not taken.
#[derive(Clone, Debug, PartialEq, Eq)]
pub enum Refusal {
    /// It could not be matched to a job in the Windows print queue.
    NotAttributed(String),
    TooLarge,
    /// The document is not what it says it is.
    BadDocument(String),
    /// The printer cannot take documents right now.
    Unavailable(String),
    Internal(String),
}

impl Refusal {
    fn status(&self) -> (u16, &str) {
        match self {
            Self::NotAttributed(m) => (status::CLIENT_ERROR_NOT_AUTHORIZED, m),
            Self::TooLarge => (
                status::CLIENT_ERROR_REQUEST_ENTITY_TOO_LARGE,
                "The document is larger than Trenova accepts.",
            ),
            Self::BadDocument(m) => (status::CLIENT_ERROR_DOCUMENT_FORMAT_ERROR, m),
            Self::Unavailable(m) => (status::SERVER_ERROR_SERVICE_UNAVAILABLE, m),
            Self::Internal(m) => (status::SERVER_ERROR_INTERNAL_ERROR, m),
        }
    }
}

/// What the printer does with a document once it has one.
pub trait JobHandler: Send + Sync {
    fn accept(&self, job: IncomingJob) -> Result<(), Refusal>;
}

/// How the printer describes itself.
#[derive(Clone, Debug)]
pub struct PrinterConfig {
    /// `ipp://127.0.0.1:<port>/ipp/print`.
    pub uri: String,
    /// `urn:uuid:…`, stable for the installation.
    pub uuid: String,
    pub name: String,
    pub info: String,
    pub location: String,
    pub make_and_model: String,
    /// The largest document taken.
    pub max_document_bytes: usize,
}

#[derive(Clone, Debug)]
struct Pending {
    name: String,
    requesting_user: Option<String>,
    format: Option<DocumentFormat>,
    since: Instant,
}

#[derive(Clone, Debug)]
struct JobRecord {
    id: u32,
    state: i32,
    reason: &'static str,
    created: Instant,
    pending: Option<Pending>,
}

#[derive(Debug)]
struct Jobs {
    next_id: u32,
    records: VecDeque<JobRecord>,
}

impl Jobs {
    fn find(&mut self, id: u32) -> Option<&mut JobRecord> {
        self.records.iter_mut().find(|r| r.id == id)
    }

    fn add(&mut self, record: JobRecord) {
        self.records.push_back(record);
        while self.records.len() > REMEMBERED_JOBS {
            self.records.pop_front();
        }
    }

    fn allocate(&mut self) -> u32 {
        let id = self.next_id;
        self.next_id = self.next_id.checked_add(1).unwrap_or(1);
        id
    }

    /// Aborts jobs whose document never came.
    fn expire(&mut self, now: Instant) {
        for record in &mut self.records {
            if record
                .pending
                .as_ref()
                .is_some_and(|p| now.duration_since(p.since) > PENDING_TIMEOUT)
            {
                record.pending = None;
                record.state = state::ABORTED;
                record.reason = "aborted-by-system";
            }
        }
    }
}

pub struct Printer<H: JobHandler> {
    config: PrinterConfig,
    handler: H,
    jobs: Mutex<Jobs>,
    started: Instant,
}

impl<H: JobHandler> std::fmt::Debug for Printer<H> {
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        f.debug_struct("Printer")
            .field("uri", &self.config.uri)
            .finish_non_exhaustive()
    }
}

/// What a response carries besides the operation group.
struct Reply {
    status: u16,
    message: Option<String>,
    groups: Vec<Group>,
}

impl Reply {
    fn ok(groups: Vec<Group>) -> Self {
        Self {
            status: status::SUCCESSFUL_OK,
            message: None,
            groups,
        }
    }

    fn error(status: u16, message: impl Into<String>) -> Self {
        Self {
            status,
            message: Some(message.into()),
            groups: Vec::new(),
        }
    }
}

fn keyword(value: &str) -> Value {
    Value::Keyword(value.to_owned())
}

fn text(value: &str) -> Value {
    Value::Text(value.to_owned())
}

/// A media size as `media-col` describes it, in hundredths of a millimetre.
fn media_col(width: i32, height: i32) -> Value {
    Value::Collection(vec![
        Attribute::new(
            "media-size",
            vec![Value::Collection(vec![
                Attribute::new("x-dimension", vec![Value::Integer(width)]),
                Attribute::new("y-dimension", vec![Value::Integer(height)]),
            ])],
        ),
        Attribute::new("media-top-margin", vec![Value::Integer(0)]),
        Attribute::new("media-bottom-margin", vec![Value::Integer(0)]),
        Attribute::new("media-left-margin", vec![Value::Integer(0)]),
        Attribute::new("media-right-margin", vec![Value::Integer(0)]),
    ])
}

const MEDIA: [(&str, i32, i32); 3] = [
    ("na_letter_8.5x11in", 21590, 27940),
    ("na_legal_8.5x14in", 21590, 35560),
    ("iso_a4_210x297mm", 21000, 29700),
];

fn one(name: &str, value: Value) -> Attribute {
    Attribute::new(name, vec![value])
}

fn many(name: &str, values: Vec<Value>) -> Attribute {
    Attribute::new(name, values)
}

fn resolution(dpi: i32) -> Value {
    Value::Resolution(Resolution {
        x: dpi,
        y: dpi,
        units: 3,
    })
}

/// What the printer takes and how it renders it.
fn format_attributes() -> Vec<Attribute> {
    vec![
        one("charset-configured", Value::Charset("utf-8".into())),
        one("charset-supported", Value::Charset("utf-8".into())),
        one(
            "natural-language-configured",
            Value::NaturalLanguage("en".into()),
        ),
        one(
            "generated-natural-language-supported",
            Value::NaturalLanguage("en".into()),
        ),
        one(
            "document-format-default",
            Value::MimeMediaType(DocumentFormat::PDF.into()),
        ),
        many(
            "document-format-supported",
            vec![
                Value::MimeMediaType(DocumentFormat::PDF.into()),
                Value::MimeMediaType(DocumentFormat::PWG_RASTER.into()),
            ],
        ),
        one("compression-supported", keyword("none")),
        one("pdl-override-supported", keyword("attempted")),
        one("multiple-document-jobs-supported", Value::Boolean(false)),
        one("job-ids-supported", Value::Boolean(true)),
        many(
            "which-jobs-supported",
            vec![
                keyword("completed"),
                keyword("not-completed"),
                keyword("all"),
            ],
        ),
        one("color-supported", Value::Boolean(true)),
        many(
            "print-color-mode-supported",
            vec![keyword("monochrome"), keyword("color")],
        ),
        one("print-color-mode-default", keyword("monochrome")),
        many(
            "pwg-raster-document-resolution-supported",
            vec![resolution(300), resolution(600)],
        ),
        many(
            "pwg-raster-document-type-supported",
            vec![keyword("black_1"), keyword("sgray_8"), keyword("srgb_8")],
        ),
        one("pwg-raster-document-sheet-back", keyword("normal")),
        many(
            "printer-resolution-supported",
            vec![resolution(300), resolution(600)],
        ),
        one("printer-resolution-default", resolution(300)),
        many("print-quality-supported", vec![Value::Enum(4)]),
        one("print-quality-default", Value::Enum(4)),
    ]
}

/// The sheets the printer offers and what it does to them.
fn media_attributes() -> Vec<Attribute> {
    vec![
        many(
            "media-supported",
            MEDIA.iter().map(|m| keyword(m.0)).collect(),
        ),
        many("media-ready", MEDIA.iter().map(|m| keyword(m.0)).collect()),
        one("media-default", keyword(MEDIA[0].0)),
        many(
            "media-col-supported",
            [
                "media-size",
                "media-top-margin",
                "media-bottom-margin",
                "media-left-margin",
                "media-right-margin",
            ]
            .iter()
            .map(|k| keyword(k))
            .collect(),
        ),
        many(
            "media-col-database",
            MEDIA.iter().map(|m| media_col(m.1, m.2)).collect(),
        ),
        many(
            "media-col-ready",
            MEDIA.iter().map(|m| media_col(m.1, m.2)).collect(),
        ),
        one("media-col-default", media_col(MEDIA[0].1, MEDIA[0].2)),
        many("media-top-margin-supported", vec![Value::Integer(0)]),
        many("media-bottom-margin-supported", vec![Value::Integer(0)]),
        many("media-left-margin-supported", vec![Value::Integer(0)]),
        many("media-right-margin-supported", vec![Value::Integer(0)]),
        many("sides-supported", vec![keyword("one-sided")]),
        one("sides-default", keyword("one-sided")),
        one("copies-supported", Value::Range(1, 1)),
        one("copies-default", Value::Integer(1)),
        many("finishings-supported", vec![Value::Enum(3)]),
        one("finishings-default", Value::Enum(3)),
        many(
            "orientation-requested-supported",
            vec![Value::Enum(3), Value::Enum(4)],
        ),
        one("orientation-requested-default", Value::Enum(3)),
        many("output-bin-supported", vec![keyword("face-up")]),
        one("output-bin-default", keyword("face-up")),
        one("pages-per-minute", Value::Integer(60)),
        one("pages-per-minute-color", Value::Integer(60)),
        many(
            "job-creation-attributes-supported",
            [
                "copies",
                "media",
                "media-col",
                "orientation-requested",
                "print-color-mode",
                "print-quality",
                "sides",
            ]
            .iter()
            .map(|k| keyword(k))
            .collect(),
        ),
        one("printer-kind", keyword("document")),
    ]
}

impl<H: JobHandler> Printer<H> {
    pub fn new(config: PrinterConfig, handler: H) -> Self {
        Self {
            config,
            handler,
            jobs: Mutex::new(Jobs {
                next_id: 1,
                records: VecDeque::new(),
            }),
            started: Instant::now(),
        }
    }

    pub fn config(&self) -> &PrinterConfig {
        &self.config
    }

    fn jobs(&self) -> std::sync::MutexGuard<'_, Jobs> {
        self.jobs.lock().unwrap_or_else(PoisonError::into_inner)
    }

    fn job_uri(&self, id: u32) -> String {
        format!("{}/{id}", self.config.uri.trim_end_matches('/'))
    }

    /// Answers one IPP request, given its whole body.
    pub fn handle(&self, request: &[u8]) -> Vec<u8> {
        let message = match decode(request) {
            Ok(message) => message,
            Err(err) => {
                let request_id = request
                    .get(4..8)
                    .and_then(|b| b.try_into().ok())
                    .map_or(0, u32::from_be_bytes);
                return Self::encode(
                    (2, 0),
                    request_id,
                    Reply::error(status::CLIENT_ERROR_BAD_REQUEST, err.to_string()),
                );
            }
        };
        let version = match message.version {
            (1, 0 | 1) => (1, 1),
            (2, _) => (2, 0),
            _ => {
                return Self::encode(
                    (2, 0),
                    message.request_id,
                    Reply::error(
                        status::SERVER_ERROR_VERSION_NOT_SUPPORTED,
                        "Use IPP 1.1 or 2.0.",
                    ),
                );
            }
        };
        let reply = if message.operation("attributes-charset").is_none() {
            Reply::error(
                status::CLIENT_ERROR_BAD_REQUEST,
                "attributes-charset is required.",
            )
        } else {
            self.jobs().expire(Instant::now());
            match message.code {
                op::GET_PRINTER_ATTRIBUTES => self.get_printer_attributes(&message),
                op::VALIDATE_JOB => match Self::check_document(&message, &[]) {
                    Ok(_) => Reply::ok(Vec::new()),
                    Err(reply) => reply,
                },
                op::PRINT_JOB => self.print_job(&message, request),
                op::CREATE_JOB => self.create_job(&message),
                op::SEND_DOCUMENT => self.send_document(&message, request),
                op::CANCEL_JOB => self.cancel_job(&message),
                op::GET_JOB_ATTRIBUTES => self.get_job_attributes(&message),
                op::GET_JOBS => self.get_jobs(&message),
                _ => Reply::error(
                    status::SERVER_ERROR_OPERATION_NOT_SUPPORTED,
                    "This printer does not support that operation.",
                ),
            }
        };
        Self::encode(version, message.request_id, reply)
    }

    fn encode(version: (u8, u8), request_id: u32, reply: Reply) -> Vec<u8> {
        let mut operation = Group::new(GroupTag::Operation);
        operation.push("attributes-charset", Value::Charset("utf-8".into()));
        operation.push(
            "attributes-natural-language",
            Value::NaturalLanguage("en".into()),
        );
        if let Some(message) = reply.message {
            operation.push("status-message", Value::Text(message));
        }
        let mut groups = vec![operation];
        groups.extend(reply.groups);
        encode(version, reply.status, request_id, &groups)
    }

    /// The document format and compression a request names, checked against
    /// what this printer takes. `data` is sniffed for octet-stream.
    fn check_document(message: &Message, data: &[u8]) -> Result<Option<DocumentFormat>, Reply> {
        if let Some(compression) = message
            .operation("compression")
            .and_then(Attribute::first)
            .and_then(Value::as_str)
            && compression != "none"
        {
            let mut unsupported = Group::new(GroupTag::Unsupported);
            unsupported.push("compression", keyword(compression));
            return Err(Reply {
                status: status::CLIENT_ERROR_COMPRESSION_NOT_SUPPORTED,
                message: Some("Send documents uncompressed.".into()),
                groups: vec![unsupported],
            });
        }
        let Some(mime) = message
            .operation("document-format")
            .and_then(Attribute::first)
            .and_then(Value::as_str)
        else {
            return Ok(None);
        };
        if mime.eq_ignore_ascii_case("application/octet-stream") {
            return Ok(DocumentFormat::sniff(data));
        }
        let Some(format) = DocumentFormat::from_mime(mime) else {
            let mut unsupported = Group::new(GroupTag::Unsupported);
            unsupported.push("document-format", Value::MimeMediaType(mime.to_owned()));
            return Err(Reply {
                status: status::CLIENT_ERROR_DOCUMENT_FORMAT_NOT_SUPPORTED,
                message: Some("Trenova takes PDF and PWG raster.".into()),
                groups: vec![unsupported],
            });
        };
        Ok(Some(format))
    }

    fn job_name(message: &Message) -> String {
        ["job-name", "document-name"]
            .iter()
            .find_map(|name| {
                message
                    .operation(name)
                    .and_then(Attribute::first)
                    .and_then(Value::as_str)
            })
            .map(str::trim)
            .filter(|n| !n.is_empty())
            .unwrap_or("Untitled")
            .to_owned()
    }

    fn requesting_user(message: &Message) -> Option<String> {
        message
            .operation("requesting-user-name")
            .and_then(Attribute::first)
            .and_then(Value::as_str)
            .map(str::trim)
            .filter(|u| !u.is_empty())
            .map(str::to_owned)
    }

    fn job_group(&self, record: &JobRecord) -> Group {
        let mut group = Group::new(GroupTag::Job);
        group.push(
            "job-id",
            Value::Integer(i32::try_from(record.id).unwrap_or(i32::MAX)),
        );
        group.push("job-uri", Value::Uri(self.job_uri(record.id)));
        group.push("job-state", Value::Enum(record.state));
        group.push("job-state-reasons", keyword(record.reason));
        group.push("job-printer-uri", Value::Uri(self.config.uri.clone()));
        let age = record
            .created
            .saturating_duration_since(self.started)
            .as_secs();
        group.push(
            "time-at-creation",
            Value::Integer(i32::try_from(age).unwrap_or(i32::MAX)),
        );
        group
    }

    /// Hands a document to the handler and records how that went.
    fn deliver(
        &self,
        id: u32,
        pending: Pending,
        format: Option<DocumentFormat>,
        data: &[u8],
    ) -> Reply {
        let finish = |state: i32, reason: &'static str| {
            let mut jobs = self.jobs();
            let record = JobRecord {
                id,
                state,
                reason,
                created: jobs.find(id).map_or_else(Instant::now, |r| r.created),
                pending: None,
            };
            match jobs.find(id) {
                Some(existing) => *existing = record.clone(),
                None => jobs.add(record.clone()),
            }
            self.job_group(&record)
        };

        if data.is_empty() {
            finish(state::ABORTED, "aborted-by-system");
            return Reply::error(status::CLIENT_ERROR_BAD_REQUEST, "The job has no document.");
        }
        if data.len() > self.config.max_document_bytes {
            finish(state::ABORTED, "aborted-by-system");
            return Reply::error(
                status::CLIENT_ERROR_REQUEST_ENTITY_TOO_LARGE,
                "The document is larger than Trenova accepts.",
            );
        }
        let Some(format) = format
            .or(pending.format)
            .or_else(|| DocumentFormat::sniff(data))
        else {
            finish(state::ABORTED, "document-format-error");
            return Reply::error(
                status::CLIENT_ERROR_DOCUMENT_FORMAT_NOT_SUPPORTED,
                "The document is neither PDF nor PWG raster.",
            );
        };

        let job = IncomingJob {
            job_id: id,
            name: pending.name,
            requesting_user: pending.requesting_user,
            format,
            data: data.to_vec(),
        };
        match self.handler.accept(job) {
            Ok(()) => Reply::ok(vec![finish(state::COMPLETED, "job-completed-successfully")]),
            Err(refusal) => {
                let (code, message) = refusal.status();
                let message = message.to_owned();
                let group = finish(state::ABORTED, "aborted-by-system");
                Reply {
                    status: code,
                    message: Some(message),
                    groups: vec![group],
                }
            }
        }
    }

    fn print_job(&self, message: &Message, request: &[u8]) -> Reply {
        let data = &request[message.data_offset..];
        let format = match Self::check_document(message, data) {
            Ok(format) => format,
            Err(reply) => return reply,
        };
        let id = self.jobs().allocate();
        let pending = Pending {
            name: Self::job_name(message),
            requesting_user: Self::requesting_user(message),
            format,
            since: Instant::now(),
        };
        self.deliver(id, pending, format, data)
    }

    fn create_job(&self, message: &Message) -> Reply {
        let format = match Self::check_document(message, &[]) {
            Ok(format) => format,
            Err(reply) => return reply,
        };
        let mut jobs = self.jobs();
        let id = jobs.allocate();
        let record = JobRecord {
            id,
            state: state::PENDING,
            reason: "job-incoming",
            created: Instant::now(),
            pending: Some(Pending {
                name: Self::job_name(message),
                requesting_user: Self::requesting_user(message),
                format,
                since: Instant::now(),
            }),
        };
        jobs.add(record.clone());
        drop(jobs);
        Reply::ok(vec![self.job_group(&record)])
    }

    /// The job a request names, by `job-id` or `job-uri`.
    fn requested_job(message: &Message) -> Option<u32> {
        if let Some(id) = message
            .operation("job-id")
            .and_then(Attribute::first)
            .and_then(Value::as_int)
        {
            return u32::try_from(id).ok();
        }
        let uri = message
            .operation("job-uri")
            .and_then(Attribute::first)
            .and_then(Value::as_str)?;
        uri.rsplit('/').next()?.parse().ok()
    }

    fn send_document(&self, message: &Message, request: &[u8]) -> Reply {
        let Some(id) = Self::requested_job(message) else {
            return Reply::error(status::CLIENT_ERROR_BAD_REQUEST, "job-id is required.");
        };
        let last = message
            .operation("last-document")
            .and_then(Attribute::first)
            .and_then(Value::as_bool);
        if last == Some(false) {
            return Reply::error(
                status::SERVER_ERROR_MULTIPLE_DOCUMENT_JOBS_NOT_SUPPORTED,
                "Send the whole job as one document.",
            );
        }
        let data = &request[message.data_offset..];
        let format = match Self::check_document(message, data) {
            Ok(format) => format,
            Err(reply) => return reply,
        };
        let pending = {
            let mut jobs = self.jobs();
            let Some(record) = jobs.find(id) else {
                return Reply::error(status::CLIENT_ERROR_NOT_FOUND, "No such job.");
            };
            match record.pending.take() {
                Some(pending) => pending,
                None => {
                    return Reply::error(
                        status::CLIENT_ERROR_NOT_POSSIBLE,
                        "The job already has its document.",
                    );
                }
            }
        };
        self.deliver(id, pending, format, data)
    }

    fn cancel_job(&self, message: &Message) -> Reply {
        let Some(id) = Self::requested_job(message) else {
            return Reply::error(status::CLIENT_ERROR_BAD_REQUEST, "job-id is required.");
        };
        let mut jobs = self.jobs();
        let Some(record) = jobs.find(id) else {
            return Reply::error(status::CLIENT_ERROR_NOT_FOUND, "No such job.");
        };
        if record.pending.take().is_none() {
            return Reply::error(
                status::CLIENT_ERROR_NOT_POSSIBLE,
                "The job has already finished.",
            );
        }
        record.state = state::CANCELED;
        record.reason = "job-canceled-by-user";
        Reply::ok(Vec::new())
    }

    fn get_job_attributes(&self, message: &Message) -> Reply {
        let Some(id) = Self::requested_job(message) else {
            return Reply::error(status::CLIENT_ERROR_BAD_REQUEST, "job-id is required.");
        };
        let record = self.jobs().find(id).cloned();
        match record {
            Some(record) => Reply::ok(vec![self.job_group(&record)]),
            None => Reply::error(status::CLIENT_ERROR_NOT_FOUND, "No such job."),
        }
    }

    fn get_jobs(&self, message: &Message) -> Reply {
        let which = message
            .operation("which-jobs")
            .and_then(Attribute::first)
            .and_then(Value::as_str)
            .unwrap_or("not-completed")
            .to_owned();
        let limit = message
            .operation("limit")
            .and_then(Attribute::first)
            .and_then(Value::as_int)
            .and_then(|l| usize::try_from(l).ok())
            .filter(|&l| l > 0)
            .unwrap_or(usize::MAX);
        let records: Vec<JobRecord> = self.jobs().records.iter().rev().cloned().collect();
        let groups = records
            .iter()
            .filter(|r| match which.as_str() {
                "completed" => r.state >= state::CANCELED,
                "all" => true,
                _ => r.state < state::CANCELED,
            })
            .take(limit)
            .map(|r| self.job_group(r))
            .collect();
        Reply::ok(groups)
    }

    fn printer_attributes(&self) -> Vec<Attribute> {
        let c = &self.config;
        let uptime = i32::try_from(self.started.elapsed().as_secs())
            .unwrap_or(i32::MAX)
            .max(1);
        let queued = self
            .jobs()
            .records
            .iter()
            .filter(|r| r.pending.is_some())
            .count();
        let operations = [
            op::PRINT_JOB,
            op::VALIDATE_JOB,
            op::CREATE_JOB,
            op::SEND_DOCUMENT,
            op::CANCEL_JOB,
            op::GET_JOB_ATTRIBUTES,
            op::GET_JOBS,
            op::GET_PRINTER_ATTRIBUTES,
        ];
        let mut attributes = vec![
            one("printer-uri-supported", Value::Uri(c.uri.clone())),
            one("uri-security-supported", keyword("none")),
            one(
                "uri-authentication-supported",
                keyword("requesting-user-name"),
            ),
            one("printer-name", Value::Name(c.name.clone())),
            one("printer-info", text(&c.info)),
            one("printer-location", text(&c.location)),
            one("printer-make-and-model", text(&c.make_and_model)),
            one("printer-uuid", Value::Uri(c.uuid.clone())),
            one("printer-state", Value::Enum(3)),
            one("printer-state-reasons", keyword("none")),
            one("printer-state-message", text("Ready")),
            one("printer-is-accepting-jobs", Value::Boolean(true)),
            one("printer-up-time", Value::Integer(uptime)),
            one(
                "queued-job-count",
                Value::Integer(i32::try_from(queued).unwrap_or(0)),
            ),
            many(
                "ipp-versions-supported",
                vec![keyword("1.1"), keyword("2.0")],
            ),
            many(
                "operations-supported",
                operations
                    .iter()
                    .map(|o| Value::Enum(i32::from(*o)))
                    .collect(),
            ),
        ];
        attributes.extend(format_attributes());
        attributes.extend(media_attributes());
        attributes
    }

    fn get_printer_attributes(&self, message: &Message) -> Reply {
        let requested: Vec<String> = message
            .operation("requested-attributes")
            .map(|a| {
                a.values
                    .iter()
                    .filter_map(Value::as_str)
                    .map(str::to_owned)
                    .collect()
            })
            .unwrap_or_default();
        let everything = requested.is_empty()
            || requested
                .iter()
                .any(|r| matches!(r.as_str(), "all" | "printer-description" | "job-template"));
        let mut group = Group::new(GroupTag::Printer);
        group.attributes = self
            .printer_attributes()
            .into_iter()
            .filter(|a| everything || requested.iter().any(|r| r == &a.name))
            .collect();
        Reply::ok(vec![group])
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::codec::decode;
    use std::sync::Arc;

    #[derive(Default)]
    struct Recorder {
        jobs: Mutex<Vec<IncomingJob>>,
        refuse: Mutex<Option<Refusal>>,
    }

    impl JobHandler for Arc<Recorder> {
        fn accept(&self, job: IncomingJob) -> Result<(), Refusal> {
            if let Some(refusal) = self.refuse.lock().expect("lock").take() {
                return Err(refusal);
            }
            self.jobs.lock().expect("lock").push(job);
            Ok(())
        }
    }

    fn printer() -> (Printer<Arc<Recorder>>, Arc<Recorder>) {
        let recorder = Arc::new(Recorder::default());
        let printer = Printer::new(
            PrinterConfig {
                uri: "ipp://127.0.0.1:49631/ipp/print".into(),
                uuid: "urn:uuid:5c1a3a2e-0000-4000-8000-000000000001".into(),
                name: "Trenova".into(),
                info: "Sends what you print to Trenova".into(),
                location: "This computer".into(),
                make_and_model: "Trenova Capture".into(),
                max_document_bytes: 64,
            },
            Arc::clone(&recorder),
        );
        (printer, recorder)
    }

    fn request(operation: u16, extra: &[(&str, Value)], data: &[u8]) -> Vec<u8> {
        let mut group = Group::new(GroupTag::Operation);
        group.push("attributes-charset", Value::Charset("utf-8".into()));
        group.push(
            "attributes-natural-language",
            Value::NaturalLanguage("en".into()),
        );
        group.push(
            "printer-uri",
            Value::Uri("ipp://127.0.0.1:49631/ipp/print".into()),
        );
        for (name, value) in extra {
            group.push(name, value.clone());
        }
        let mut bytes = encode((2, 0), operation, 7, &[group]);
        bytes.extend_from_slice(data);
        bytes
    }

    fn reply(bytes: &[u8]) -> Message {
        decode(bytes).expect("a valid response")
    }

    fn job_attr(message: &Message, name: &str) -> Option<Value> {
        message
            .group(GroupTag::Job)
            .and_then(|g| g.get(name))
            .and_then(Attribute::first)
            .cloned()
    }

    #[test]
    fn the_printer_describes_itself_and_honours_requested_attributes() {
        let (printer, _) = printer();
        let all = reply(&printer.handle(&request(op::GET_PRINTER_ATTRIBUTES, &[], &[])));
        assert_eq!(all.code, status::SUCCESSFUL_OK);
        assert_eq!(all.request_id, 7);
        let attributes = all.group(GroupTag::Printer).expect("printer group");
        let formats: Vec<_> = attributes
            .get("document-format-supported")
            .expect("formats")
            .values
            .iter()
            .filter_map(Value::as_str)
            .collect();
        assert_eq!(formats, ["application/pdf", "image/pwg-raster"]);
        assert!(attributes.get("media-col-database").is_some());

        let some = reply(&printer.handle(&request(
            op::GET_PRINTER_ATTRIBUTES,
            &[(
                "requested-attributes",
                Value::Keyword("printer-name".into()),
            )],
            &[],
        )));
        let only = some.group(GroupTag::Printer).expect("printer group");
        assert_eq!(only.attributes.len(), 1);
        assert_eq!(
            only.attributes[0].first(),
            Some(&Value::Name("Trenova".into()))
        );
    }

    #[test]
    fn a_printed_pdf_is_handed_on_with_its_name_and_claimed_user() {
        let (printer, recorder) = printer();
        let answer = reply(&printer.handle(&request(
            op::PRINT_JOB,
            &[
                ("requesting-user-name", Value::Name("jdoe".into())),
                ("job-name", Value::Name("Rate confirmation".into())),
                (
                    "document-format",
                    Value::MimeMediaType("application/pdf".into()),
                ),
            ],
            b"%PDF-1.7 body",
        )));
        assert_eq!(answer.code, status::SUCCESSFUL_OK);
        assert_eq!(job_attr(&answer, "job-state"), Some(Value::Enum(9)));
        let jobs = recorder.jobs.lock().expect("lock");
        assert_eq!(jobs.len(), 1);
        assert_eq!(jobs[0].name, "Rate confirmation");
        assert_eq!(jobs[0].requesting_user.as_deref(), Some("jdoe"));
        assert_eq!(jobs[0].format, DocumentFormat::Pdf);
        assert_eq!(jobs[0].data, b"%PDF-1.7 body");
    }

    #[test]
    fn create_job_then_send_document_is_one_job_and_octet_stream_is_sniffed() {
        let (printer, recorder) = printer();
        let created = reply(&printer.handle(&request(
            op::CREATE_JOB,
            &[("job-name", Value::Name("Invoice 1042".into()))],
            &[],
        )));
        assert_eq!(job_attr(&created, "job-state"), Some(Value::Enum(3)));
        let Some(Value::Integer(id)) = job_attr(&created, "job-id") else {
            panic!("no job id");
        };
        let sent = reply(&printer.handle(&request(
            op::SEND_DOCUMENT,
            &[
                ("job-id", Value::Integer(id)),
                ("last-document", Value::Boolean(true)),
                (
                    "document-format",
                    Value::MimeMediaType("application/octet-stream".into()),
                ),
            ],
            b"RaS2....",
        )));
        assert_eq!(sent.code, status::SUCCESSFUL_OK);
        assert_eq!(
            recorder.jobs.lock().expect("lock")[0].format,
            DocumentFormat::PwgRaster
        );
        assert_eq!(recorder.jobs.lock().expect("lock")[0].name, "Invoice 1042");

        let again = reply(&printer.handle(&request(
            op::SEND_DOCUMENT,
            &[("job-id", Value::Integer(id))],
            b"%PDF-",
        )));
        assert_eq!(again.code, status::CLIENT_ERROR_NOT_POSSIBLE);
    }

    #[test]
    fn an_unattributed_job_is_refused_and_reported_aborted() {
        let (printer, recorder) = printer();
        *recorder.refuse.lock().expect("lock") =
            Some(Refusal::NotAttributed("No print queue job matches.".into()));
        let answer = reply(&printer.handle(&request(op::PRINT_JOB, &[], b"%PDF-1.7")));
        assert_eq!(answer.code, status::CLIENT_ERROR_NOT_AUTHORIZED);
        assert_eq!(job_attr(&answer, "job-state"), Some(Value::Enum(8)));
        let jobs = reply(&printer.handle(&request(
            op::GET_JOBS,
            &[("which-jobs", Value::Keyword("completed".into()))],
            &[],
        )));
        assert_eq!(
            jobs.groups
                .iter()
                .filter(|g| g.tag == GroupTag::Job)
                .count(),
            1
        );
    }

    #[test]
    fn formats_compression_size_and_versions_are_checked() {
        let (printer, recorder) = printer();
        let word = reply(&printer.handle(&request(
            op::PRINT_JOB,
            &[(
                "document-format",
                Value::MimeMediaType("application/msword".into()),
            )],
            b"doc",
        )));
        assert_eq!(
            word.code,
            status::CLIENT_ERROR_DOCUMENT_FORMAT_NOT_SUPPORTED
        );
        assert!(word.group(GroupTag::Unsupported).is_some());

        let gzip = reply(&printer.handle(&request(
            op::VALIDATE_JOB,
            &[("compression", Value::Keyword("gzip".into()))],
            &[],
        )));
        assert_eq!(gzip.code, status::CLIENT_ERROR_COMPRESSION_NOT_SUPPORTED);

        let large = reply(&printer.handle(&request(op::PRINT_JOB, &[], &[b'%'; 65])));
        assert_eq!(large.code, status::CLIENT_ERROR_REQUEST_ENTITY_TOO_LARGE);

        let mut v3 = request(op::GET_PRINTER_ATTRIBUTES, &[], &[]);
        v3[0] = 3;
        assert_eq!(
            reply(&printer.handle(&v3)).code,
            status::SERVER_ERROR_VERSION_NOT_SUPPORTED
        );

        let garbage = reply(&printer.handle(&[2, 0, 0, 11, 0, 0, 0, 9, 0x01, 0x47]));
        assert_eq!(garbage.code, status::CLIENT_ERROR_BAD_REQUEST);
        assert_eq!(garbage.request_id, 9);

        let unknown = reply(&printer.handle(&request(0x0010, &[], &[])));
        assert_eq!(unknown.code, status::SERVER_ERROR_OPERATION_NOT_SUPPORTED);
        assert!(recorder.jobs.lock().expect("lock").is_empty());
    }

    #[test]
    fn a_pending_job_can_be_cancelled_but_a_finished_one_cannot() {
        let (printer, _) = printer();
        let created = reply(&printer.handle(&request(op::CREATE_JOB, &[], &[])));
        let Some(Value::Integer(id)) = job_attr(&created, "job-id") else {
            panic!("no job id");
        };
        let cancel = |printer: &Printer<Arc<Recorder>>| {
            reply(&printer.handle(&request(
                op::CANCEL_JOB,
                &[("job-id", Value::Integer(id))],
                &[],
            )))
        };
        assert_eq!(cancel(&printer).code, status::SUCCESSFUL_OK);
        assert_eq!(cancel(&printer).code, status::CLIENT_ERROR_NOT_POSSIBLE);
        let looked_up = reply(&printer.handle(&request(
            op::GET_JOB_ATTRIBUTES,
            &[(
                "job-uri",
                Value::Uri(format!("ipp://127.0.0.1:49631/ipp/print/{id}")),
            )],
            &[],
        )));
        assert_eq!(job_attr(&looked_up, "job-state"), Some(Value::Enum(7)));
        assert!(
            job_attr(&looked_up, "job-name").is_none(),
            "names are never shown"
        );
    }
}
