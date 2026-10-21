//! The supervisor: the gRPC service, which validates each call, admits it
//! against the budget, streams its input into a sealed memfd, runs it on the
//! sandbox pool and streams the results back.
//!
//! Nothing here parses a document. The supervisor checks headers and sizes,
//! counts bytes and forwards what children produce; every byte of document
//! content is read only inside a child.

pub mod cost;
pub mod errors;
pub mod metrics;

use std::sync::Arc;
use std::time::Duration;

use prost::Message;
use tokio::sync::{Semaphore, mpsc};
use tokio::time::Instant;
use tokio_stream::wrappers::ReceiverStream;
use tonic::{Request, Response, Status, Streaming};
use trenova_proto::imaging::v1::imaging_service_server::ImagingService;
use trenova_proto::imaging::v1::{
    AssembleRequest, AssembleResponse, AssembleSummary, CapturePageResult, CaptureSummary,
    DocumentHeader, ErrorReason, InspectCaptureRequest, InspectCaptureResponse,
    NormalizeImageRequest, NormalizeImageResponse, NormalizeSummary, PageError, ProbeRequest,
    ProbeResponse, RenderRequest, RenderResponse, RenderSummary, assemble_request,
    assemble_response, inspect_capture_request, inspect_capture_response, normalize_image_request,
    normalize_image_response, probe_request, render_request, render_response, render_size,
};
use trenova_sandbox::{InputWriter, Job, Outcome, Pool, SealedInput};
use trenova_service_kit::{Admission, Cost, Permit, Priority};

use crate::config::Config;
use crate::protocol::{JobInput, JobLimits, Operation, WorkerError, WorkerJob};
use errors::{from_job, from_rejection, from_worker, invalid, status};
use metrics::{CallLabels, DurationLabels, OperationLabel, ServiceMetrics, code_label};

/// The most pages one capture call may carry.
const MAX_CAPTURE_PAGES: u32 = 1000;
/// The most sources one assembly may take.
const MAX_SOURCES: u32 = 500;
/// The most outputs one assembly may build.
const MAX_OUTPUTS: usize = 1000;
/// Frames buffered between a child and the caller.
const FORWARD_BUFFER: usize = 8;

const CONTENT_TYPES: &[&str] = &[
    "application/pdf",
    "image/jpeg",
    "image/png",
    "image/webp",
    "image/tiff",
    "image/gif",
    "image/heic",
    "image/heif",
];

#[derive(Debug)]
struct Shared {
    pool: Pool,
    admission: Admission,
    config: Config,
    metrics: ServiceMetrics,
}

/// The `ImagingService` implementation.
#[derive(Clone, Debug)]
pub struct ImagingServer {
    shared: Arc<Shared>,
}

impl ImagingServer {
    pub fn new(pool: Pool, admission: Admission, config: Config, metrics: ServiceMetrics) -> Self {
        Self {
            shared: Arc::new(Shared {
                pool,
                admission,
                config,
                metrics,
            }),
        }
    }
}

/// One call's context: what it is, its class and its deadline.
#[derive(Clone, Debug)]
struct Call {
    operation: &'static str,
    priority: Priority,
    deadline: Instant,
    started: Instant,
    shared: Arc<Shared>,
}

impl Call {
    fn new<T>(shared: &Arc<Shared>, operation: &'static str, request: &Request<T>) -> Self {
        let metadata = request.metadata();
        let priority = Priority::from_metadata(
            metadata
                .get(Priority::METADATA_KEY)
                .and_then(|value| value.to_str().ok()),
        );
        let cap = shared.config.deadline(operation);
        let asked = metadata
            .get("grpc-timeout")
            .and_then(|value| value.to_str().ok())
            .and_then(parse_grpc_timeout);
        let started = Instant::now();
        Self {
            operation,
            priority,
            deadline: started + asked.map_or(cap, |asked| asked.min(cap)),
            started,
            shared: Arc::clone(shared),
        }
    }

    fn limits(&self) -> JobLimits {
        let limits = &self.shared.config.limits;
        JobLimits {
            max_pixels: limits.max_render_pixels,
            max_pages: limits.max_pages,
            chunk_bytes: u32::try_from(limits.chunk_bytes).unwrap_or(256 << 10),
            max_image_pixels: limits.max_image_pixels,
        }
    }

    async fn admit(&self, cost: Cost) -> Result<Permit, Status> {
        self.shared
            .admission
            .admit(cost, self.priority)
            .await
            .map_err(from_rejection)
    }

    fn finish(&self, code: tonic::Code) {
        let metrics = &self.shared.metrics;
        metrics
            .calls
            .get_or_create(&CallLabels {
                operation: self.operation,
                priority: self.priority.into(),
                code: code_label(code),
            })
            .inc();
        metrics
            .duration
            .get_or_create(&DurationLabels {
                operation: self.operation,
                priority: self.priority.into(),
            })
            .observe(self.started.elapsed().as_secs_f64());
    }

    fn count_in(&self, bytes: u64) {
        self.shared
            .metrics
            .bytes_in
            .get_or_create(&OperationLabel {
                operation: self.operation,
            })
            .inc_by(bytes);
    }

    fn count_out(&self, bytes: usize, pages: u64) {
        let label = OperationLabel {
            operation: self.operation,
        };
        self.shared
            .metrics
            .bytes_out
            .get_or_create(&label)
            .inc_by(bytes as u64);
        if pages > 0 {
            self.shared
                .metrics
                .pages
                .get_or_create(&label)
                .inc_by(pages);
        }
    }

    /// Runs a job, forwarding its output frames to `forward` as `R`, and
    /// returns the closing message's bytes.
    async fn run<R>(
        &self,
        operation: Operation,
        inputs: Vec<JobInput>,
        input: Option<SealedInput>,
        forward: Option<&mpsc::Sender<Result<R, Status>>>,
    ) -> Result<Vec<u8>, Status>
    where
        R: Message + Default + Send + 'static,
    {
        if Instant::now() >= self.deadline {
            return Err(status(
                ErrorReason::Timeout,
                "the deadline passed before the work began",
                None,
            ));
        }
        let job = Job {
            body: WorkerJob {
                operation: Some(operation),
                inputs,
                limits: Some(self.limits()),
            }
            .encode_to_vec(),
            input,
            deadline: self.deadline,
        };
        let (frames_tx, mut frames_rx) = mpsc::channel::<Vec<u8>>(FORWARD_BUFFER);
        let running = self.shared.pool.run(job, frames_tx);
        let forwarding = async {
            let mut failure = None;
            while let Some(frame) = frames_rx.recv().await {
                if failure.is_some() {
                    continue;
                }
                self.count_out(frame.len(), 0);
                let Some(forward) = forward else {
                    failure = Some(status(
                        ErrorReason::Internal,
                        "a child produced output for a call that takes none",
                        None,
                    ));
                    break;
                };
                let Ok(message) = R::decode(frame.as_slice()) else {
                    failure = Some(status(
                        ErrorReason::Internal,
                        "a child produced a malformed message",
                        None,
                    ));
                    break;
                };
                if forward.send(Ok(message)).await.is_err() {
                    failure = Some(Status::cancelled("the caller went away"));
                    break;
                }
            }
            failure
        };
        let (outcome, failure) = tokio::join!(running, forwarding);
        if let Some(failure) = failure {
            return Err(failure);
        }
        match outcome {
            Ok(Outcome::Done(body)) => Ok(body),
            Ok(Outcome::Failed(body)) => Err(WorkerError::decode(body.as_slice()).map_or_else(
                |_| {
                    status(
                        ErrorReason::Internal,
                        "a child reported a malformed error",
                        None,
                    )
                },
                |e| from_worker(&e),
            )),
            Err(err) => Err(from_job(&err)),
        }
    }
}

/// Parses the `grpc-timeout` header: up to eight digits and a unit.
fn parse_grpc_timeout(value: &str) -> Option<Duration> {
    let (digits, unit) = value.split_at(value.len().checked_sub(1)?);
    if digits.is_empty() || digits.len() > 8 {
        return None;
    }
    let amount: u64 = digits.parse().ok()?;
    Some(match unit {
        "H" => Duration::from_secs(amount * 3600),
        "M" => Duration::from_secs(amount * 60),
        "S" => Duration::from_secs(amount),
        "m" => Duration::from_millis(amount),
        "u" => Duration::from_micros(amount),
        "n" => Duration::from_nanos(amount),
        _ => return None,
    })
}

fn check_document(
    document: Option<&DocumentHeader>,
    max_bytes: u64,
) -> Result<DocumentHeader, Status> {
    let document = document.ok_or_else(|| invalid("the header names no document"))?;
    let content_type = document.content_type.trim().to_ascii_lowercase();
    if !CONTENT_TYPES.contains(&content_type.as_str()) {
        return Err(status(
            ErrorReason::UnsupportedFormat,
            format!("{content_type:?} is not a type this worker reads"),
            None,
        ));
    }
    if document.declared_size == 0 {
        return Err(status(
            ErrorReason::InvalidDocument,
            "the document is empty",
            None,
        ));
    }
    if document.declared_size > max_bytes {
        return Err(status(
            ErrorReason::TooLarge,
            format!(
                "a {}-byte document is over the {max_bytes}-byte limit",
                document.declared_size
            ),
            None,
        ));
    }
    Ok(DocumentHeader {
        content_type,
        declared_size: document.declared_size,
    })
}

fn writer(limit: u64) -> Result<InputWriter, Status> {
    InputWriter::new("imaging-input", limit).map_err(|_| {
        status(
            ErrorReason::Internal,
            "the worker could not hold the input",
            None,
        )
    })
}

/// Writes a chunk, mapping an overrun to `TOO_LARGE`.
fn write(writer: &mut InputWriter, chunk: &[u8]) -> Result<(), Status> {
    writer.write_chunk(chunk).map_err(|err| {
        if err.kind() == std::io::ErrorKind::FileTooLarge {
            status(
                ErrorReason::TooLarge,
                "the stream sent more bytes than declared",
                None,
            )
        } else {
            status(
                ErrorReason::Internal,
                "the worker could not hold the input",
                None,
            )
        }
    })
}

fn seal(writer: InputWriter) -> Result<SealedInput, Status> {
    writer.seal().map_err(|_| {
        status(
            ErrorReason::Internal,
            "the worker could not seal the input",
            None,
        )
    })
}

/// Reads one document's chunks, which must add up to its declared size.
async fn read_single<M, F>(
    call: &Call,
    stream: &mut Streaming<M>,
    document: &DocumentHeader,
    chunk_of: F,
) -> Result<SealedInput, Status>
where
    F: Fn(M) -> Option<Vec<u8>>,
{
    let mut input = writer(document.declared_size)?;
    loop {
        let next = tokio::time::timeout_at(call.deadline, stream.message())
            .await
            .map_err(|_| {
                status(
                    ErrorReason::Timeout,
                    "the input did not arrive before the deadline",
                    None,
                )
            })??;
        let Some(message) = next else {
            break;
        };
        let chunk =
            chunk_of(message).ok_or_else(|| invalid("only chunks may follow the header"))?;
        write(&mut input, &chunk)?;
    }
    if input.len() != document.declared_size {
        return Err(status(
            ErrorReason::InvalidDocument,
            format!(
                "the stream sent {} of {} declared bytes",
                input.len(),
                document.declared_size
            ),
            None,
        ));
    }
    call.count_in(input.len());
    seal(input)
}

fn single_input(document: &DocumentHeader) -> Vec<JobInput> {
    vec![JobInput {
        index: 0,
        content_type: document.content_type.clone(),
        offset: 0,
        length: document.declared_size,
    }]
}

async fn first<M>(call: &Call, stream: &mut Streaming<M>) -> Result<M, Status> {
    tokio::time::timeout_at(call.deadline, stream.message())
        .await
        .map_err(|_| {
            status(
                ErrorReason::Timeout,
                "no header arrived before the deadline",
                None,
            )
        })??
        .ok_or_else(|| invalid("the call sent nothing"))
}

/// Records the call's outcome and passes the result through.
fn finished<T>(call: &Call, result: Result<T, Status>) -> Result<T, Status> {
    call.finish(
        result
            .as_ref()
            .map_or_else(Status::code, |_| tonic::Code::Ok),
    );
    result
}

/// Runs a streaming call's work in its own task, sending its failure, if
/// any, as the stream's last item.
fn spawn_stream<R, F>(call: Call, tx: mpsc::Sender<Result<R, Status>>, work: F)
where
    R: Send + 'static,
    F: std::future::Future<Output = Result<R, Status>> + Send + 'static,
{
    tokio::spawn(async move {
        let result = work.await;
        call.finish(
            result
                .as_ref()
                .map_or_else(Status::code, |_| tonic::Code::Ok),
        );
        let _ = tx.send(result).await;
    });
}

type Stream<R> = ReceiverStream<Result<R, Status>>;

#[tonic::async_trait]
impl ImagingService for ImagingServer {
    async fn probe(
        &self,
        request: Request<Streaming<ProbeRequest>>,
    ) -> Result<Response<ProbeResponse>, Status> {
        let call = Call::new(&self.shared, "probe", &request);
        let result = async {
            let mut stream = request.into_inner();
            let Some(probe_request::Part::Header(header)) = first(&call, &mut stream).await?.part
            else {
                return Err(invalid("the first message must be the header"));
            };
            let document = check_document(
                header.document.as_ref(),
                call.shared.config.limits.max_input_bytes,
            )?;
            let _permit = call.admit(cost::job(document.declared_size, 0)).await?;
            let input = read_single(&call, &mut stream, &document, |m: ProbeRequest| {
                match m.part {
                    Some(probe_request::Part::Chunk(chunk)) => Some(chunk),
                    _ => None,
                }
            })
            .await?;
            let done = call
                .run::<ProbeResponse>(
                    Operation::Probe(header),
                    single_input(&document),
                    Some(input),
                    None,
                )
                .await?;
            ProbeResponse::decode(done.as_slice())
                .map(Response::new)
                .map_err(|_| {
                    status(
                        ErrorReason::Internal,
                        "a child produced a malformed probe",
                        None,
                    )
                })
        }
        .await;
        finished(&call, result)
    }

    type RenderStream = Stream<RenderResponse>;

    async fn render(
        &self,
        request: Request<Streaming<RenderRequest>>,
    ) -> Result<Response<Self::RenderStream>, Status> {
        let call = Call::new(&self.shared, "render", &request);
        let mut stream = request.into_inner();
        let header = match first(&call, &mut stream).await.map(|m| m.part) {
            Ok(Some(render_request::Part::Header(header))) => header,
            Ok(_) => return finished(&call, Err(invalid("the first message must be the header"))),
            Err(err) => return finished(&call, Err(err)),
        };
        let document = match check_document(
            header.document.as_ref(),
            call.shared.config.limits.max_input_bytes,
        ) {
            Ok(document) => document,
            Err(err) => return finished(&call, Err(err)),
        };
        let pixels = match header.size.as_ref().and_then(|s| s.size) {
            Some(render_size::Size::Dpi(dpi)) => cost::letter_pixels(dpi.min(1200)),
            Some(render_size::Size::Fit(fit)) => cost::fit_pixels(fit.max_width, fit.max_height),
            None => return finished(&call, Err(invalid("a render needs a size"))),
        }
        .min(call.shared.config.limits.max_render_pixels);
        let permit = match call.admit(cost::job(document.declared_size, pixels)).await {
            Ok(permit) => permit,
            Err(err) => return finished(&call, Err(err)),
        };

        let (tx, rx) = mpsc::channel(FORWARD_BUFFER);
        let forward = tx.clone();
        let task_call = call.clone();
        spawn_stream(call, tx, async move {
            let _permit = permit;
            let input = read_single(
                &task_call,
                &mut stream,
                &document,
                |m: RenderRequest| match m.part {
                    Some(render_request::Part::Chunk(chunk)) => Some(chunk),
                    _ => None,
                },
            )
            .await?;
            let done = task_call
                .run(
                    Operation::Render(header),
                    single_input(&document),
                    Some(input),
                    Some(&forward),
                )
                .await?;
            let summary = RenderSummary::decode(done.as_slice()).map_err(|_| {
                status(
                    ErrorReason::Internal,
                    "a child produced a malformed summary",
                    None,
                )
            })?;
            task_call.count_out(0, u64::from(summary.rendered));
            Ok(RenderResponse {
                part: Some(render_response::Part::Summary(summary)),
            })
        });
        Ok(Response::new(ReceiverStream::new(rx)))
    }

    type InspectCaptureStream = Stream<InspectCaptureResponse>;

    async fn inspect_capture(
        &self,
        request: Request<Streaming<InspectCaptureRequest>>,
    ) -> Result<Response<Self::InspectCaptureStream>, Status> {
        let call = Call::new(&self.shared, "inspect_capture", &request);
        let mut stream = request.into_inner();
        let header = match first(&call, &mut stream).await.map(|m| m.part) {
            Ok(Some(inspect_capture_request::Part::Header(header))) => header,
            Ok(_) => return finished(&call, Err(invalid("the first message must be the header"))),
            Err(err) => return finished(&call, Err(err)),
        };
        if header.page_count == 0 || header.page_count > MAX_CAPTURE_PAGES {
            return finished(
                &call,
                Err(invalid(format!(
                    "a capture batch holds 1 to {MAX_CAPTURE_PAGES} pages"
                ))),
            );
        }

        let (tx, rx) = mpsc::channel(FORWARD_BUFFER);
        let forward = tx.clone();
        let task_call = call.clone();
        spawn_stream(call, tx, async move {
            inspect_batch(task_call, header, stream, forward).await
        });
        Ok(Response::new(ReceiverStream::new(rx)))
    }

    type AssembleStream = Stream<AssembleResponse>;

    async fn assemble(
        &self,
        request: Request<Streaming<AssembleRequest>>,
    ) -> Result<Response<Self::AssembleStream>, Status> {
        let call = Call::new(&self.shared, "assemble", &request);
        let mut stream = request.into_inner();
        let header = match first(&call, &mut stream).await.map(|m| m.part) {
            Ok(Some(assemble_request::Part::Header(header))) => header,
            Ok(_) => return finished(&call, Err(invalid("the first message must be the header"))),
            Err(err) => return finished(&call, Err(err)),
        };
        let max_input = call.shared.config.limits.max_input_bytes;
        if header.source_count == 0 || header.source_count > MAX_SOURCES {
            return finished(
                &call,
                Err(invalid(format!(
                    "an assembly takes 1 to {MAX_SOURCES} sources"
                ))),
            );
        }
        if header.outputs.is_empty() || header.outputs.len() > MAX_OUTPUTS {
            return finished(
                &call,
                Err(invalid(format!(
                    "an assembly builds 1 to {MAX_OUTPUTS} outputs"
                ))),
            );
        }
        if header.total_size == 0 || header.total_size > max_input {
            return finished(
                &call,
                Err(status(
                    ErrorReason::TooLarge,
                    format!("an assembly's sources may total at most {max_input} bytes"),
                    None,
                )),
            );
        }
        let permit = match call
            .admit(cost::job(
                header.total_size.saturating_mul(2),
                cost::letter_pixels(300),
            ))
            .await
        {
            Ok(permit) => permit,
            Err(err) => return finished(&call, Err(err)),
        };

        let (tx, rx) = mpsc::channel(FORWARD_BUFFER);
        let forward = tx.clone();
        let task_call = call.clone();
        spawn_stream(call, tx, async move {
            let _permit = permit;
            let (inputs, input) = read_sources(&task_call, &mut stream, &header).await?;
            let done = task_call
                .run(
                    Operation::Assemble(header),
                    inputs,
                    Some(input),
                    Some(&forward),
                )
                .await?;
            let summary = AssembleSummary::decode(done.as_slice()).map_err(|_| {
                status(
                    ErrorReason::Internal,
                    "a child produced a malformed summary",
                    None,
                )
            })?;
            task_call.count_out(
                0,
                summary
                    .output_page_counts
                    .iter()
                    .map(|&n| u64::from(n))
                    .sum(),
            );
            Ok(AssembleResponse {
                part: Some(assemble_response::Part::Summary(summary)),
            })
        });
        Ok(Response::new(ReceiverStream::new(rx)))
    }

    type NormalizeImageStream = Stream<NormalizeImageResponse>;

    async fn normalize_image(
        &self,
        request: Request<Streaming<NormalizeImageRequest>>,
    ) -> Result<Response<Self::NormalizeImageStream>, Status> {
        let call = Call::new(&self.shared, "normalize_image", &request);
        let mut stream = request.into_inner();
        let header = match first(&call, &mut stream).await.map(|m| m.part) {
            Ok(Some(normalize_image_request::Part::Header(header))) => header,
            Ok(_) => return finished(&call, Err(invalid("the first message must be the header"))),
            Err(err) => return finished(&call, Err(err)),
        };
        let document = match check_document(
            header.document.as_ref(),
            call.shared.config.limits.max_input_bytes,
        ) {
            Ok(document) => document,
            Err(err) => return finished(&call, Err(err)),
        };
        let pixels = document
            .declared_size
            .saturating_mul(20)
            .min(call.shared.config.limits.max_image_pixels);
        let permit = match call.admit(cost::job(document.declared_size, pixels)).await {
            Ok(permit) => permit,
            Err(err) => return finished(&call, Err(err)),
        };

        let (tx, rx) = mpsc::channel(FORWARD_BUFFER);
        let forward = tx.clone();
        let task_call = call.clone();
        spawn_stream(call, tx, async move {
            let _permit = permit;
            let input = read_single(
                &task_call,
                &mut stream,
                &document,
                |m: NormalizeImageRequest| match m.part {
                    Some(normalize_image_request::Part::Chunk(chunk)) => Some(chunk),
                    _ => None,
                },
            )
            .await?;
            let done = task_call
                .run(
                    Operation::NormalizeImage(header),
                    single_input(&document),
                    Some(input),
                    Some(&forward),
                )
                .await?;
            let summary = NormalizeSummary::decode(done.as_slice()).map_err(|_| {
                status(
                    ErrorReason::Internal,
                    "a child produced a malformed summary",
                    None,
                )
            })?;
            Ok(NormalizeImageResponse {
                part: Some(normalize_image_response::Part::Summary(summary)),
            })
        });
        Ok(Response::new(ReceiverStream::new(rx)))
    }
}

/// Reads an assembly's sources, end to end, into one memfd.
async fn read_sources(
    call: &Call,
    stream: &mut Streaming<AssembleRequest>,
    header: &trenova_proto::imaging::v1::AssembleHeader,
) -> Result<(Vec<JobInput>, SealedInput), Status> {
    let mut input = writer(header.total_size)?;
    let mut inputs: Vec<JobInput> = Vec::with_capacity(header.source_count as usize);
    loop {
        let next = tokio::time::timeout_at(call.deadline, stream.message())
            .await
            .map_err(|_| {
                status(
                    ErrorReason::Timeout,
                    "the sources did not arrive before the deadline",
                    None,
                )
            })??;
        let Some(message) = next else {
            break;
        };
        match message.part {
            Some(assemble_request::Part::Source(source)) => {
                finish_source(&input, inputs.last())?;
                if source.index != u32::try_from(inputs.len()).unwrap_or(u32::MAX) {
                    return Err(invalid("sources must arrive in index order"));
                }
                let document = check_document(source.document.as_ref(), header.total_size)?;
                inputs.push(JobInput {
                    index: source.index,
                    content_type: document.content_type,
                    offset: input.len(),
                    length: document.declared_size,
                });
            }
            Some(assemble_request::Part::Chunk(chunk)) => {
                let current = inputs
                    .last()
                    .ok_or_else(|| invalid("a chunk arrived before any source"))?;
                if input.len() + chunk.len() as u64 > current.offset + current.length {
                    return Err(status(
                        ErrorReason::TooLarge,
                        "a source sent more bytes than declared",
                        None,
                    ));
                }
                write(&mut input, &chunk)?;
            }
            _ => return Err(invalid("only sources and chunks may follow the header")),
        }
    }
    finish_source(&input, inputs.last())?;
    if inputs.len() != header.source_count as usize {
        return Err(invalid(format!(
            "the header promised {} sources and {} arrived",
            header.source_count,
            inputs.len()
        )));
    }
    if input.len() != header.total_size {
        return Err(invalid(
            "the sources do not add up to the declared total size",
        ));
    }
    call.count_in(input.len());
    Ok((inputs, seal(input)?))
}

fn finish_source(input: &InputWriter, current: Option<&JobInput>) -> Result<(), Status> {
    if let Some(current) = current
        && input.len() != current.offset + current.length
    {
        return Err(status(
            ErrorReason::InvalidDocument,
            format!("source {} sent fewer bytes than declared", current.index),
            None,
        ));
    }
    Ok(())
}

/// A capture sub-batch being collected.
struct Batch {
    input: InputWriter,
    pages: Vec<JobInput>,
}

/// Runs a capture batch: pages are gathered into sub-batches as they arrive,
/// each sub-batch runs on a child while the next is read, and results go back
/// per page as they come. A sub-batch whose child dies is run again a page at
/// a time, so one hostile page fails alone.
// One state machine over the request stream: pages, their chunks, and the
// sub-batches they are gathered into.
#[allow(clippy::too_many_lines)]
async fn inspect_batch(
    call: Call,
    header: trenova_proto::imaging::v1::InspectCaptureHeader,
    mut stream: Streaming<InspectCaptureRequest>,
    forward: mpsc::Sender<Result<InspectCaptureResponse, Status>>,
) -> Result<InspectCaptureResponse, Status> {
    let limits = &call.shared.config.limits;
    let batch_size = limits.capture_batch as usize;
    let parallel = Arc::new(Semaphore::new(limits.capture_parallelism));
    let mut remaining = limits.max_input_bytes;
    let mut running = tokio::task::JoinSet::new();
    let mut batch: Option<Batch> = None;
    let mut received: u32 = 0;
    let mut skipped: u32 = 0;
    let mut skipping: u64 = 0;
    let mut seen = std::collections::BTreeSet::new();

    loop {
        let next = tokio::time::timeout_at(call.deadline, stream.message())
            .await
            .map_err(|_| {
                status(
                    ErrorReason::Timeout,
                    "the pages did not arrive before the deadline",
                    None,
                )
            })??;
        let Some(message) = next else {
            break;
        };
        match message.part {
            Some(inspect_capture_request::Part::Page(page)) => {
                if let Some(open) = &batch {
                    check_page_complete(open)?;
                    if open.pages.len() >= batch_size {
                        let full = batch.take().ok_or_else(|| invalid("no batch"))?;
                        dispatch_capture(&call, &header, full, &parallel, &forward, &mut running)
                            .await?;
                    }
                }
                if page.index >= header.page_count || !seen.insert(page.index) {
                    return Err(invalid(format!(
                        "page index {} is repeated or outside the batch",
                        page.index
                    )));
                }
                received += 1;
                skipping = 0;
                let document = match check_document(page.document.as_ref(), remaining) {
                    Ok(document) => document,
                    Err(err) => {
                        skipping = page.document.as_ref().map_or(0, |d| d.declared_size);
                        skipped += 1;
                        send_page_error(&forward, page.index, reason_of(&err), err.message())
                            .await?;
                        continue;
                    }
                };
                remaining -= document.declared_size;
                let open = match &mut batch {
                    Some(open) => open,
                    None => batch.insert(Batch {
                        input: writer(limits.max_input_bytes)?,
                        pages: Vec::with_capacity(batch_size),
                    }),
                };
                open.pages.push(JobInput {
                    index: page.index,
                    content_type: document.content_type,
                    offset: open.input.len(),
                    length: document.declared_size,
                });
            }
            Some(inspect_capture_request::Part::Chunk(chunk)) => {
                if skipping > 0 {
                    let length = chunk.len() as u64;
                    if length > skipping {
                        return Err(invalid("a refused page sent more bytes than it declared"));
                    }
                    skipping -= length;
                    continue;
                }
                let open = batch
                    .as_mut()
                    .ok_or_else(|| invalid("a chunk arrived before any page"))?;
                let current = open
                    .pages
                    .last()
                    .ok_or_else(|| invalid("a chunk arrived before any page"))?;
                if open.input.len() + chunk.len() as u64 > current.offset + current.length {
                    return Err(status(
                        ErrorReason::TooLarge,
                        "a page sent more bytes than declared",
                        None,
                    ));
                }
                write(&mut open.input, &chunk)?;
            }
            _ => return Err(invalid("only pages and chunks may follow the header")),
        }
    }
    if let Some(open) = batch.take() {
        check_page_complete(&open)?;
        dispatch_capture(&call, &header, open, &parallel, &forward, &mut running).await?;
    }
    if received != header.page_count {
        return Err(invalid(format!(
            "the header promised {} pages and {received} arrived",
            header.page_count
        )));
    }

    let mut failed = skipped;
    while let Some(joined) = running.join_next().await {
        failed +=
            joined.map_err(|_| status(ErrorReason::Internal, "a capture job panicked", None))??;
    }
    call.count_out(0, u64::from(received));
    Ok(InspectCaptureResponse {
        part: Some(inspect_capture_response::Part::Summary(CaptureSummary {
            inspected: received - failed,
            failed,
        })),
    })
}

fn check_page_complete(batch: &Batch) -> Result<(), Status> {
    if let Some(last) = batch.pages.last()
        && batch.input.len() != last.offset + last.length
    {
        return Err(status(
            ErrorReason::InvalidDocument,
            format!("page {} sent fewer bytes than declared", last.index),
            None,
        ));
    }
    Ok(())
}

async fn dispatch_capture(
    call: &Call,
    header: &trenova_proto::imaging::v1::InspectCaptureHeader,
    batch: Batch,
    parallel: &Arc<Semaphore>,
    forward: &mpsc::Sender<Result<InspectCaptureResponse, Status>>,
    running: &mut tokio::task::JoinSet<Result<u32, Status>>,
) -> Result<(), Status> {
    let slot = Arc::clone(parallel)
        .acquire_owned()
        .await
        .map_err(|_| status(ErrorReason::Internal, "the capture batch was closed", None))?;
    let bytes = batch.input.len();
    call.count_in(bytes);
    let input = seal(batch.input)?;
    let permit = call
        .admit(cost::job(
            bytes,
            cost::letter_pixels(header.render_dpi.clamp(72, 300).max(150)),
        ))
        .await?;
    let call = call.clone();
    let header = *header;
    let forward = forward.clone();
    let pages = batch.pages;
    running.spawn(async move {
        let _slot = slot;
        let _permit = permit;
        inspect_pages(&call, &header, pages, input, &forward).await
    });
    Ok(())
}

/// Runs one sub-batch; on a child failure, each page again on its own.
/// Returns how many pages failed.
async fn inspect_pages(
    call: &Call,
    header: &trenova_proto::imaging::v1::InspectCaptureHeader,
    pages: Vec<JobInput>,
    input: SealedInput,
    forward: &mpsc::Sender<Result<InspectCaptureResponse, Status>>,
) -> Result<u32, Status> {
    let alone = pages.len() == 1;
    let copy = input
        .try_clone()
        .map_err(|_| status(ErrorReason::Internal, "the input could not be shared", None))?;
    match call
        .run(
            Operation::InspectCapture(*header),
            pages.clone(),
            Some(copy),
            Some(forward),
        )
        .await
    {
        Ok(done) => Ok(CaptureSummary::decode(done.as_slice()).map_or(0, |summary| summary.failed)),
        Err(err) if err.code() == tonic::Code::Cancelled => Err(err),
        Err(err) if alone || !retryable(&err) => {
            let reason = reason_of(&err);
            for page in &pages {
                send_page_error(forward, page.index, reason, err.message()).await?;
            }
            Ok(u32::try_from(pages.len()).unwrap_or(u32::MAX))
        }
        Err(_) => {
            let mut failed = 0;
            for page in pages {
                let copy = input.try_clone().map_err(|_| {
                    status(ErrorReason::Internal, "the input could not be shared", None)
                })?;
                let index = page.index;
                match call
                    .run(
                        Operation::InspectCapture(*header),
                        vec![page],
                        Some(copy),
                        Some(forward),
                    )
                    .await
                {
                    Ok(done) => {
                        failed += CaptureSummary::decode(done.as_slice())
                            .map_or(0, |summary| summary.failed);
                    }
                    Err(err) => {
                        if err.code() == tonic::Code::Cancelled {
                            return Err(err);
                        }
                        send_page_error(forward, index, reason_of(&err), err.message()).await?;
                        failed += 1;
                    }
                }
            }
            Ok(failed)
        }
    }
}

/// Whether a sub-batch's failure is the kind a page-at-a-time rerun can
/// narrow down: its child died or ran out of time.
fn retryable(err: &Status) -> bool {
    matches!(
        reason_of(err),
        ErrorReason::WorkerCrashed | ErrorReason::Timeout | ErrorReason::Internal
    )
}

fn reason_of(err: &Status) -> ErrorReason {
    use tonic_types::StatusExt;
    err.get_details_error_info()
        .and_then(|info| ErrorReason::from_str_name(&format!("ERROR_REASON_{}", info.reason)))
        .unwrap_or(ErrorReason::Internal)
}

async fn send_page_error(
    forward: &mpsc::Sender<Result<InspectCaptureResponse, Status>>,
    index: u32,
    reason: ErrorReason,
    message: &str,
) -> Result<(), Status> {
    forward
        .send(Ok(InspectCaptureResponse {
            part: Some(inspect_capture_response::Part::Page(CapturePageResult {
                index,
                error: Some(PageError {
                    page_index: index,
                    reason: reason as i32,
                    message: message.to_owned(),
                }),
                ..CapturePageResult::default()
            })),
        }))
        .await
        .map_err(|_| Status::cancelled("the caller went away"))
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn grpc_timeouts_parse_in_every_unit() {
        assert_eq!(parse_grpc_timeout("30S"), Some(Duration::from_secs(30)));
        assert_eq!(
            parse_grpc_timeout("1500m"),
            Some(Duration::from_millis(1500))
        );
        assert_eq!(parse_grpc_timeout("2M"), Some(Duration::from_secs(120)));
        assert_eq!(parse_grpc_timeout("123456789S"), None);
        assert_eq!(parse_grpc_timeout("S"), None);
        assert_eq!(parse_grpc_timeout("10x"), None);
    }

    #[test]
    fn only_known_types_within_the_limit_are_taken() {
        let ok = DocumentHeader {
            content_type: "Application/PDF".into(),
            declared_size: 10,
        };
        assert_eq!(
            check_document(Some(&ok), 100).expect("ok").content_type,
            "application/pdf"
        );
        let big = DocumentHeader {
            declared_size: 1000,
            ..ok.clone()
        };
        assert_eq!(
            check_document(Some(&big), 100)
                .expect_err("too large")
                .code(),
            tonic::Code::OutOfRange
        );
        let text = DocumentHeader {
            content_type: "text/html".into(),
            declared_size: 10,
        };
        assert_eq!(
            check_document(Some(&text), 100)
                .expect_err("refused")
                .code(),
            tonic::Code::InvalidArgument
        );
    }
}
