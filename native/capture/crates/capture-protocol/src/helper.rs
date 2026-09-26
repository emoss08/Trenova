//! The pipe between the agent and a scan helper.
//!
//! A helper is started once per job, in the bitness of the data source it
//! drives, with its standard input and output as the two ends of the pipe. It
//! reads one [`HelperCommand`] (and possibly a later
//! [`HelperCommand::Cancel`]) and writes [`HelperEvent`]s back until it exits.
//! A page travels as a [`HelperEvent::Page`] followed immediately by one
//! binary frame holding its single-page PDF, so raw bitmaps never cross the
//! process boundary and the helper never touches the network.
//!
//! Every frame is a one-byte kind, a little-endian `u32` length and that many
//! bytes. A JSON frame is at most [`MAX_JSON_FRAME`] bytes and a page frame at
//! most [`MAX_PAGE_BYTES`], so a confused or hostile peer cannot make the
//! other side allocate without bound.

use std::io::{self, Read, Write};

use serde::{Deserialize, Serialize, de::DeserializeOwned};

use crate::api::{
    MAX_PAGE_BYTES, PixelType, RequestFailureCode, Settings, SourceInfo, SourceProtocol,
};

/// The largest JSON frame either side accepts.
pub const MAX_JSON_FRAME: usize = 1 << 20;

const KIND_JSON: u8 = 1;
const KIND_BYTES: u8 = 2;
const HEADER_LEN: usize = 5;

/// Which scanner a job is for, as enumeration reported it.
#[derive(Clone, Debug, PartialEq, Eq, Hash, Serialize, Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct SourceRef {
    pub name: String,
    pub protocol: SourceProtocol,
}

/// Everything a helper needs to run one scan. It is resolved by the agent from
/// the profile and what the source said it can do.
#[derive(Clone, Debug, PartialEq, Eq, Serialize, Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct ScanJob {
    pub source: SourceRef,
    pub dpi: u32,
    pub pixel_type: PixelType,
    pub duplex: bool,
    pub use_feeder: bool,
    pub discard_blank_pages: bool,
    /// 30 to 95; used for grayscale and colour pages.
    pub jpeg_quality: u8,
    pub show_driver_ui: bool,
    pub detect_patch_codes: bool,
    pub detect_barcodes: bool,
}

/// What the agent asks of a helper.
#[derive(Clone, Debug, PartialEq, Eq, Serialize, Deserialize)]
#[serde(tag = "type", rename_all = "camelCase")]
pub enum HelperCommand {
    /// List every source this helper's bitness can reach.
    Enumerate,
    /// Scan with these settings.
    Scan(ScanJob),
    /// Stop the scan at the next page boundary. Pages already sent are kept.
    Cancel,
}

/// A condition that stopped a scan partway. The pages sent before it are
/// good, and the agent offers to continue the batch.
#[derive(Clone, Copy, Debug, PartialEq, Eq, Serialize, Deserialize)]
#[serde(rename_all = "camelCase")]
pub enum ScanCondition {
    PaperJam,
    DoubleFeed,
    CoverOpen,
    FeederEmpty,
    /// The operator closed or cancelled the driver's own window.
    CanceledByOperator,
}

impl ScanCondition {
    /// How the condition reads on a failed request, when nothing was scanned.
    pub fn failure_code(self) -> RequestFailureCode {
        match self {
            Self::PaperJam | Self::DoubleFeed | Self::CoverOpen => RequestFailureCode::PaperJam,
            Self::FeederEmpty => RequestFailureCode::FeederEmpty,
            Self::CanceledByOperator => RequestFailureCode::CanceledByUser,
        }
    }
}

/// One page, sent just before its bytes.
#[derive(Clone, Debug, PartialEq, Eq, Serialize, Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct PageMeta {
    /// 1-based position within this run of the helper.
    pub index: u32,
    pub width_px: u32,
    pub height_px: u32,
    pub dpi: u32,
    pub pixel_type: PixelType,
    #[serde(default)]
    pub patch_code: Option<String>,
    #[serde(default)]
    pub barcodes: Vec<String>,
}

/// What a helper reports.
#[derive(Clone, Debug, PartialEq, Eq, Serialize, Deserialize)]
#[serde(tag = "type", rename_all = "camelCase")]
pub enum HelperEvent {
    /// The answer to [`HelperCommand::Enumerate`].
    Sources { sources: Vec<SourceInfo> },
    /// What the source said it can do once opened. Enumeration cannot ask a
    /// TWAIN source without opening it, so this is where its capabilities
    /// are first learned.
    Described { source: SourceInfo },
    /// The source is open and negotiated; these are the settings in force.
    Started { settings: Settings },
    /// A page follows as one binary frame.
    Page(PageMeta),
    /// The scan stopped early; the pages already sent stand.
    Condition {
        condition: ScanCondition,
        message: String,
    },
    /// The scan ended normally.
    Finished { pages: u32 },
    /// The scan could not run, or broke in a way that keeps nothing further.
    Failed {
        code: RequestFailureCode,
        message: String,
    },
}

/// A frame as read off the pipe.
#[derive(Clone, Debug, PartialEq, Eq)]
pub enum Frame<T> {
    Message(T),
    Bytes(Vec<u8>),
}

#[derive(Debug, thiserror::Error)]
pub enum FrameError {
    #[error("pipe: {0}")]
    Io(#[from] io::Error),
    #[error("the pipe ended in the middle of a frame")]
    Truncated,
    #[error("unknown frame kind {0}")]
    UnknownKind(u8),
    #[error("a {kind} frame of {len} bytes is over the {max} byte limit")]
    TooLarge {
        kind: &'static str,
        len: usize,
        max: usize,
    },
    #[error("unreadable message: {0}")]
    Json(#[from] serde_json::Error),
}

/// Writes one message frame.
pub fn write_message<W: Write, T: Serialize>(
    writer: &mut W,
    message: &T,
) -> Result<(), FrameError> {
    let payload = serde_json::to_vec(message)?;
    write_frame(writer, KIND_JSON, &payload, MAX_JSON_FRAME, "message")
}

/// Writes one binary frame.
pub fn write_bytes<W: Write>(writer: &mut W, bytes: &[u8]) -> Result<(), FrameError> {
    write_frame(writer, KIND_BYTES, bytes, MAX_PAGE_BYTES, "page")
}

fn write_frame<W: Write>(
    writer: &mut W,
    kind: u8,
    payload: &[u8],
    max: usize,
    label: &'static str,
) -> Result<(), FrameError> {
    if payload.len() > max {
        return Err(FrameError::TooLarge {
            kind: label,
            len: payload.len(),
            max,
        });
    }
    let len = u32::try_from(payload.len()).map_err(|_| FrameError::TooLarge {
        kind: label,
        len: payload.len(),
        max,
    })?;
    let mut header = [0u8; HEADER_LEN];
    header[0] = kind;
    header[1..].copy_from_slice(&len.to_le_bytes());
    writer.write_all(&header)?;
    writer.write_all(payload)?;
    writer.flush()?;
    Ok(())
}

/// Reads the next frame. `Ok(None)` is the peer closing the pipe cleanly,
/// between frames.
pub fn read_frame<R: Read, T: DeserializeOwned>(
    reader: &mut R,
) -> Result<Option<Frame<T>>, FrameError> {
    let mut header = [0u8; HEADER_LEN];
    match read_full(reader, &mut header)? {
        0 => return Ok(None),
        HEADER_LEN => {}
        _ => return Err(FrameError::Truncated),
    }

    let len = u32::from_le_bytes([header[1], header[2], header[3], header[4]]) as usize;
    let (max, label) = match header[0] {
        KIND_JSON => (MAX_JSON_FRAME, "message"),
        KIND_BYTES => (MAX_PAGE_BYTES, "page"),
        other => return Err(FrameError::UnknownKind(other)),
    };
    if len > max {
        return Err(FrameError::TooLarge {
            kind: label,
            len,
            max,
        });
    }

    let mut payload = vec![0u8; len];
    if read_full(reader, &mut payload)? != len {
        return Err(FrameError::Truncated);
    }

    if header[0] == KIND_JSON {
        Ok(Some(Frame::Message(serde_json::from_slice(&payload)?)))
    } else {
        Ok(Some(Frame::Bytes(payload)))
    }
}

/// Fills `buf` unless the stream ends first, returning how much was read.
fn read_full<R: Read>(reader: &mut R, buf: &mut [u8]) -> io::Result<usize> {
    let mut filled = 0;
    while filled < buf.len() {
        match reader.read(&mut buf[filled..]) {
            Ok(0) => break,
            Ok(n) => filled += n,
            Err(err) if err.kind() == io::ErrorKind::Interrupted => {}
            Err(err) => return Err(err),
        }
    }
    Ok(filled)
}

#[cfg(test)]
mod tests {
    use super::*;

    fn job() -> ScanJob {
        ScanJob {
            source: SourceRef {
                name: "fi-8170".into(),
                protocol: SourceProtocol::Twain,
            },
            dpi: 300,
            pixel_type: PixelType::BlackWhite,
            duplex: true,
            use_feeder: true,
            discard_blank_pages: true,
            jpeg_quality: 80,
            show_driver_ui: false,
            detect_patch_codes: true,
            detect_barcodes: false,
        }
    }

    #[test]
    fn a_page_travels_as_its_meta_then_its_bytes() {
        let mut pipe = Vec::new();
        let meta = PageMeta {
            index: 1,
            width_px: 2550,
            height_px: 3300,
            dpi: 300,
            pixel_type: PixelType::BlackWhite,
            patch_code: Some("T".into()),
            barcodes: vec![],
        };
        write_message(&mut pipe, &HelperEvent::Page(meta.clone())).expect("meta");
        write_bytes(&mut pipe, b"%PDF-1.7 one page").expect("bytes");
        write_message(&mut pipe, &HelperEvent::Finished { pages: 1 }).expect("finished");

        let mut reader = pipe.as_slice();
        assert_eq!(
            read_frame::<_, HelperEvent>(&mut reader).expect("frame"),
            Some(Frame::Message(HelperEvent::Page(meta)))
        );
        assert_eq!(
            read_frame::<_, HelperEvent>(&mut reader).expect("frame"),
            Some(Frame::Bytes(b"%PDF-1.7 one page".to_vec()))
        );
        assert_eq!(
            read_frame::<_, HelperEvent>(&mut reader).expect("frame"),
            Some(Frame::Message(HelperEvent::Finished { pages: 1 }))
        );
        assert_eq!(
            read_frame::<_, HelperEvent>(&mut reader).expect("eof"),
            None
        );
    }

    #[test]
    fn commands_are_tagged_json() {
        let mut pipe = Vec::new();
        write_message(&mut pipe, &HelperCommand::Scan(job())).expect("scan");
        let json: serde_json::Value = serde_json::from_slice(&pipe[HEADER_LEN..]).expect("json");
        assert_eq!(json["type"], "scan");
        assert_eq!(json["source"]["protocol"], "TWAIN");
        assert_eq!(json["pixelType"], "BlackWhite");

        let mut reader = pipe.as_slice();
        assert_eq!(
            read_frame::<_, HelperCommand>(&mut reader).expect("frame"),
            Some(Frame::Message(HelperCommand::Scan(job())))
        );
    }

    #[test]
    fn a_frame_cut_short_is_an_error_not_an_end() {
        let mut pipe = Vec::new();
        write_bytes(&mut pipe, &[7u8; 64]).expect("bytes");
        let cut = &pipe[..pipe.len() - 1];
        assert!(matches!(
            read_frame::<_, HelperEvent>(&mut &cut[..]),
            Err(FrameError::Truncated)
        ));
        let header_only = &pipe[..3];
        assert!(matches!(
            read_frame::<_, HelperEvent>(&mut &header_only[..]),
            Err(FrameError::Truncated)
        ));
    }

    #[test]
    fn an_oversized_or_unknown_frame_is_refused_before_allocating() {
        let mut header = vec![KIND_JSON];
        header.extend_from_slice(
            &u32::try_from(MAX_JSON_FRAME + 1)
                .expect("fits")
                .to_le_bytes(),
        );
        assert!(matches!(
            read_frame::<_, HelperEvent>(&mut header.as_slice()),
            Err(FrameError::TooLarge { .. })
        ));

        let unknown = [9u8, 0, 0, 0, 0];
        assert!(matches!(
            read_frame::<_, HelperEvent>(&mut &unknown[..]),
            Err(FrameError::UnknownKind(9))
        ));

        let mut sink = Vec::new();
        assert!(matches!(
            write_bytes(&mut sink, &vec![0u8; MAX_PAGE_BYTES + 1]),
            Err(FrameError::TooLarge { .. })
        ));
        assert!(sink.is_empty());
    }

    #[test]
    fn conditions_map_to_the_failure_the_server_knows() {
        assert_eq!(
            ScanCondition::DoubleFeed.failure_code(),
            RequestFailureCode::PaperJam
        );
        assert_eq!(
            ScanCondition::CanceledByOperator.failure_code(),
            RequestFailureCode::CanceledByUser
        );
    }
}
