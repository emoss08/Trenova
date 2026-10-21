//! What the supervisor and a child say to each other inside sandbox frames.
//!
//! A job body is a [`WorkerJob`]: the operation's public header from the
//! contract, and where each input sits in the job's one sealed memfd (inputs
//! are laid end to end, so a capture sub-batch of many pages or an assembly
//! of many sources still passes one descriptor). Output frames carry the
//! operation's public response messages, encoded, so the supervisor forwards
//! them as they are; the done frame carries the closing message (a summary,
//! or the whole `ProbeResponse`); a failed frame carries a [`WorkerError`].

use prost::Message;
use trenova_proto::imaging::v1::{
    AssembleHeader, ErrorReason, InspectCaptureHeader, NormalizeImageHeader, ProbeHeader,
    RenderHeader,
};

/// One input inside the job's memfd.
#[derive(Clone, PartialEq, Eq, Message)]
pub struct JobInput {
    /// The page index (capture) or source index (assembly) the caller gave.
    #[prost(uint32, tag = "1")]
    pub index: u32,
    #[prost(string, tag = "2")]
    pub content_type: String,
    #[prost(uint64, tag = "3")]
    pub offset: u64,
    #[prost(uint64, tag = "4")]
    pub length: u64,
}

/// The limits a child applies to the job, from the worker's configuration.
#[derive(Clone, Copy, PartialEq, Eq, Message)]
pub struct JobLimits {
    #[prost(uint64, tag = "1")]
    pub max_pixels: u64,
    #[prost(uint32, tag = "2")]
    pub max_pages: u32,
    #[prost(uint32, tag = "3")]
    pub chunk_bytes: u32,
    #[prost(uint64, tag = "4")]
    pub max_image_pixels: u64,
}

#[derive(Clone, PartialEq, Message)]
pub struct WorkerJob {
    #[prost(oneof = "Operation", tags = "1, 2, 3, 4, 5")]
    pub operation: Option<Operation>,
    #[prost(message, repeated, tag = "10")]
    pub inputs: Vec<JobInput>,
    #[prost(message, optional, tag = "11")]
    pub limits: Option<JobLimits>,
}

#[derive(Clone, PartialEq, prost::Oneof)]
pub enum Operation {
    #[prost(message, tag = "1")]
    Probe(ProbeHeader),
    #[prost(message, tag = "2")]
    Render(RenderHeader),
    #[prost(message, tag = "3")]
    InspectCapture(InspectCaptureHeader),
    #[prost(message, tag = "4")]
    Assemble(AssembleHeader),
    #[prost(message, tag = "5")]
    NormalizeImage(NormalizeImageHeader),
}

impl Operation {
    pub fn name(&self) -> &'static str {
        match self {
            Self::Probe(_) => "probe",
            Self::Render(_) => "render",
            Self::InspectCapture(_) => "inspect_capture",
            Self::Assemble(_) => "assemble",
            Self::NormalizeImage(_) => "normalize_image",
        }
    }
}

/// A job's typed failure.
#[derive(Clone, PartialEq, Eq, Message)]
pub struct WorkerError {
    #[prost(enumeration = "ErrorReason", tag = "1")]
    pub reason: i32,
    #[prost(string, tag = "2")]
    pub message: String,
}

impl WorkerError {
    pub fn new(reason: ErrorReason, message: impl Into<String>) -> Self {
        Self {
            reason: reason as i32,
            message: message.into(),
        }
    }
}

/// What the supervisor tells every child at start.
#[derive(Clone, PartialEq, Eq, Message)]
pub struct ChildInit {
    /// Where libpdfium is.
    #[prost(string, tag = "1")]
    pub pdfium_library: String,
    /// Where libheif is; empty to refuse HEIC and HEIF.
    #[prost(string, tag = "2")]
    pub heif_library: String,
    #[prost(uint32, tag = "3")]
    pub max_frame_bytes: u32,
}

pub fn encode<M: Message>(message: &M) -> Vec<u8> {
    message.encode_to_vec()
}
