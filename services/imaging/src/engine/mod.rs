//! The work a child does: everything that parses a document or an image.
//!
//! The engine runs only inside a hardened child. It is handed the job's
//! header and the job's mapped input, and sends what it produces through a
//! [`Sink`], one encoded response message at a time, so a document with
//! hundreds of pages never has more than one page's output in memory.

pub mod decode;
pub mod error;
pub mod heif;
pub mod pdf;
pub mod probe;
pub mod raster;
pub mod render;

use prost::Message;
use trenova_proto::imaging::v1::{ArtifactChunk, ArtifactHeader};

use crate::protocol::{ChildInit, JobInput, JobLimits, Operation, WorkerJob};
use error::{EngineError, EngineResult};
use heif::Heif;
use pdf::Pdf;

/// Where the engine sends each message it produces.
pub trait Sink {
    fn emit(&mut self, message: Vec<u8>) -> EngineResult<()>;
}

/// The loaded libraries.
#[derive(Debug)]
pub struct Engine {
    pdf: Pdf,
    heif: Option<Heif>,
}

/// One input of a job: its slice of the mapped memfd.
#[derive(Clone, Copy, Debug)]
pub struct Input<'a> {
    pub index: u32,
    pub content_type: &'a str,
    pub bytes: &'a [u8],
}

impl Engine {
    /// Loads PDFium and, when configured, libheif. Runs before hardening.
    pub fn new(init: &ChildInit) -> EngineResult<Self> {
        let pdf = Pdf::bind(&init.pdfium_library)?;
        let heif = if init.heif_library.is_empty() {
            None
        } else {
            Some(Heif::load(&init.heif_library)?)
        };
        Ok(Self { pdf, heif })
    }

    pub fn pdf(&self) -> &Pdf {
        &self.pdf
    }

    pub fn heif(&self) -> Option<&Heif> {
        self.heif.as_ref()
    }

    /// Runs a job. Returns the encoded closing message.
    pub fn run(
        &mut self,
        job: &WorkerJob,
        input: &[u8],
        sink: &mut dyn Sink,
    ) -> EngineResult<Vec<u8>> {
        let limits = job.limits.unwrap_or(JobLimits {
            max_pixels: 25_000_000,
            max_pages: 2000,
            chunk_bytes: 256 << 10,
            max_image_pixels: 100_000_000,
        });
        let inputs = slices(&job.inputs, input)?;
        let operation = job
            .operation
            .as_ref()
            .ok_or_else(|| EngineError::request("the job names no operation"))?;
        match operation {
            Operation::Probe(header) => {
                let first = only(&inputs)?;
                Ok(probe::probe(self, header, first, limits)?.encode_to_vec())
            }
            Operation::Render(header) => {
                let first = only(&inputs)?;
                Ok(render::render(self, header, first, limits, sink)?.encode_to_vec())
            }
            Operation::InspectCapture(_)
            | Operation::Assemble(_)
            | Operation::NormalizeImage(_) => Err(EngineError::request(format!(
                "{} is not available on this worker",
                operation.name()
            ))),
        }
    }
}

fn only<'a>(inputs: &[Input<'a>]) -> EngineResult<Input<'a>> {
    match inputs {
        [one] => Ok(*one),
        _ => Err(EngineError::request(
            "the operation takes exactly one document",
        )),
    }
}

/// Cuts the job's inputs out of the mapped memfd, refusing any that runs
/// outside it.
pub fn slices<'a>(inputs: &'a [JobInput], input: &'a [u8]) -> EngineResult<Vec<Input<'a>>> {
    inputs
        .iter()
        .map(|meta| {
            let start = usize::try_from(meta.offset)
                .map_err(|_| EngineError::request("an input offset is too large"))?;
            let length = usize::try_from(meta.length)
                .map_err(|_| EngineError::request("an input length is too large"))?;
            let end = start
                .checked_add(length)
                .filter(|&end| end <= input.len())
                .ok_or_else(|| EngineError::request("an input runs past the job's data"))?;
            Ok(Input {
                index: meta.index,
                content_type: meta.content_type.as_str(),
                bytes: &input[start..end],
            })
        })
        .collect()
}

/// Sends an artifact as its header and chunks, each wrapped in the
/// operation's response message.
pub fn emit_artifact<M, H, C>(
    sink: &mut dyn Sink,
    header: ArtifactHeader,
    data: &[u8],
    chunk_bytes: u32,
    wrap_header: H,
    wrap_chunk: C,
) -> EngineResult<()>
where
    M: Message,
    H: FnOnce(ArtifactHeader) -> M,
    C: Fn(ArtifactChunk) -> M,
{
    sink.emit(wrap_header(header).encode_to_vec())?;
    let chunk = (chunk_bytes as usize).max(16 << 10);
    if data.is_empty() {
        return sink.emit(
            wrap_chunk(ArtifactChunk {
                data: Vec::new(),
                last: true,
            })
            .encode_to_vec(),
        );
    }
    let mut pieces = data.chunks(chunk).peekable();
    while let Some(piece) = pieces.next() {
        sink.emit(
            wrap_chunk(ArtifactChunk {
                data: piece.to_vec(),
                last: pieces.peek().is_none(),
            })
            .encode_to_vec(),
        )?;
    }
    Ok(())
}

/// What a document is, decided from its bytes and checked against what the
/// caller declared.
#[derive(Clone, Copy, Debug, PartialEq, Eq)]
pub enum Kind {
    Pdf,
    Image(decode::ImageKind),
}

pub fn kind_of(input: &Input<'_>) -> EngineResult<Kind> {
    let declared = input.content_type.trim().to_ascii_lowercase();
    let declared_pdf = declared == "application/pdf";
    let declared_image = declared.starts_with("image/");
    if !declared_pdf && !declared_image {
        return Err(EngineError::unsupported(format!(
            "{declared:?} is not a type this worker reads"
        )));
    }
    if decode::is_pdf(input.bytes) {
        if !declared_pdf {
            return Err(EngineError::invalid(
                "the file is a PDF but was declared as an image",
            ));
        }
        return Ok(Kind::Pdf);
    }
    if declared_pdf {
        return Err(EngineError::invalid("the file is not a PDF"));
    }
    decode::sniff(input.bytes)
        .map(Kind::Image)
        .ok_or_else(|| EngineError::unsupported("the file is not an image this worker reads"))
}

#[cfg(test)]
mod tests {
    use super::*;

    struct Collect(Vec<Vec<u8>>);

    impl Sink for Collect {
        fn emit(&mut self, message: Vec<u8>) -> EngineResult<()> {
            self.0.push(message);
            Ok(())
        }
    }

    #[test]
    fn an_input_outside_the_data_is_refused() {
        let inputs = [JobInput {
            index: 0,
            content_type: "application/pdf".into(),
            offset: 4,
            length: 10,
        }];
        assert!(slices(&inputs, &[0u8; 8]).is_err());
    }

    #[test]
    fn an_artifact_is_its_header_then_chunks_with_the_last_flagged() {
        use trenova_proto::imaging::v1::{RenderResponse, render_response::Part};
        let mut sink = Collect(Vec::new());
        emit_artifact(
            &mut sink,
            ArtifactHeader::default(),
            &vec![7u8; 40 << 10],
            16 << 10,
            |h| RenderResponse {
                part: Some(Part::Page(h)),
            },
            |c| RenderResponse {
                part: Some(Part::Chunk(c)),
            },
        )
        .expect("emits");
        assert_eq!(sink.0.len(), 4);
        let last = RenderResponse::decode(sink.0[3].as_slice()).expect("decodes");
        assert!(matches!(
            last.part,
            Some(Part::Chunk(ArtifactChunk { last: true, .. }))
        ));
    }

    #[test]
    fn a_declared_type_must_match_the_bytes() {
        let pdf = Input {
            index: 0,
            content_type: "image/png",
            bytes: b"%PDF-1.7\n",
        };
        assert!(kind_of(&pdf).is_err());
        let text = Input {
            index: 0,
            content_type: "text/plain",
            bytes: b"hello",
        };
        assert_eq!(
            kind_of(&text).expect_err("refused").reason,
            trenova_proto::imaging::v1::ErrorReason::UnsupportedFormat
        );
    }
}
