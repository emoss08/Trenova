//! Whether anything a child runs maps a file as code once the child has
//! hardened.
//!
//! The sandbox's seccomp filter still lets `mmap` map a file executable,
//! because the dynamic loader does that. Refusing it once the child has
//! hardened would also stop a compromised child from running a later job's
//! input as code, but only if PDFium, libheif and its plugins, and the codecs
//! never load code after start (a lazy `dlopen`, a glibc module, a plugin found
//! on first use).
//!
//! This test answers that. It re-runs itself as a child that loads the
//! libraries exactly as `trenova-imaging child` does, installs a filter that
//! kills the process on any executable mapping of a file, applies the
//! sandbox's own hardening, and runs every operation over the fixtures,
//! HEIC included when `TRENOVA_HEIF_LIBRARY` is set. A pass proves the
//! operations load no code after hardening, so the sandbox may refuse
//! file-backed `PROT_EXEC`; a death by `SIGSYS` names the rule as still
//! needed. A load landlock already refuses never reaches `mmap` and is not
//! counted, as it would not be in a child either.

#![allow(clippy::print_stderr, clippy::print_stdout)]

use std::process::ExitCode;

#[cfg(not(target_os = "linux"))]
fn main() -> ExitCode {
    eprintln!("skipped: the sandbox runs only on Linux");
    ExitCode::SUCCESS
}

#[cfg(target_os = "linux")]
fn main() -> ExitCode {
    linux::main()
}

#[cfg(target_os = "linux")]
mod linux {
    use std::os::unix::process::ExitStatusExt;
    use std::path::PathBuf;
    use std::process::{Command, ExitCode};

    use trenova_imaging::config::{Config, ENV_PREFIX};
    use trenova_imaging::engine::error::EngineResult;
    use trenova_imaging::engine::{Engine, Sink};
    use trenova_imaging::protocol::{ChildInit, JobInput, Operation, WorkerJob};
    use trenova_imaging::supervisor::pool_config;
    use trenova_proto::imaging::v1::{
        AssembleHeader, AssembleOutput, DocumentHeader, FitBox, ImageFormat, InspectCaptureHeader,
        NormalizeImageHeader, PageRange, ProbeHeader, RenderHeader, RenderSize, render_size,
    };
    use trenova_service_kit::config::{Source, load_from};

    const PROBE_ENV: &str = "TRENOVA_IMAGING_CODE_AFTER_HARDENING";
    const FINISHED: &str = "every operation ran with executable file mappings refused";

    pub fn main() -> ExitCode {
        let Some(pdfium) = std::env::var("TRENOVA_PDFIUM_LIBRARY")
            .ok()
            .filter(|path| std::path::Path::new(path).is_file())
        else {
            eprintln!("skipped: TRENOVA_PDFIUM_LIBRARY is not set");
            return ExitCode::SUCCESS;
        };
        if std::env::var_os(PROBE_ENV).is_some() {
            return probe(&pdfium);
        }

        let output = Command::new(std::env::current_exe().expect("the test's own path"))
            .env(PROBE_ENV, "1")
            .output()
            .expect("the probe runs");
        let stdout = String::from_utf8_lossy(&output.stdout);
        let stderr = String::from_utf8_lossy(&output.stderr);
        if output.status.signal() == Some(libc::SIGSYS) {
            eprintln!(
                "refuted: a library mapped a file as code after the child hardened, so the \
                 sandbox must keep allowing file-backed PROT_EXEC\n{stdout}{stderr}"
            );
            return ExitCode::FAILURE;
        }
        if !output.status.success() || !stdout.contains(FINISHED) {
            eprintln!(
                "the probe did not finish ({}), which proves nothing either way\n{stdout}{stderr}",
                output.status
            );
            return ExitCode::FAILURE;
        }
        println!(
            "proven: no operation maps a file as code after hardening, so the sandbox may \
             refuse file-backed PROT_EXEC"
        );
        ExitCode::SUCCESS
    }

    fn probe(pdfium: &str) -> ExitCode {
        let heif = std::env::var("TRENOVA_HEIF_LIBRARY").unwrap_or_default();
        let config: Config = load_from(
            &Source {
                env_prefix: ENV_PREFIX.into(),
                file: None,
            },
            vec![
                ("TRENOVA_IMAGING__GRPC__LISTEN".into(), "127.0.0.1:0".into()),
                (
                    "TRENOVA_IMAGING__NATIVE__PDFIUM_LIBRARY".into(),
                    format!("\"{pdfium}\""),
                ),
                (
                    "TRENOVA_IMAGING__NATIVE__HEIF_LIBRARY".into(),
                    format!("\"{heif}\""),
                ),
            ],
        )
        .expect("the probe's configuration is valid");
        let fixtures = Fixtures::read();

        let mut engine = Engine::new(&ChildInit {
            pdfium_library: pdfium.to_owned(),
            heif_library: heif.clone(),
            max_frame_bytes: 8 << 20,
        })
        .expect("the libraries load");
        if let Err(err) = refuse_executable_file_mappings() {
            eprintln!("the strict filter could not be installed: {err}");
            return ExitCode::from(3);
        }
        let report = trenova_sandbox::harden::harden(&pool_config(&config).policy);
        if !report.no_new_privs || !report.seccomp {
            eprintln!("the sandbox's hardening is not in force: {report:?}");
            return ExitCode::from(4);
        }

        for (name, job, input) in fixtures.jobs(!heif.is_empty()) {
            let mut sink = Discard;
            if let Err(err) = engine.run(&job, &input, &mut sink) {
                eprintln!("{name} failed: {err}");
                return ExitCode::from(5);
            }
            println!("{name}: ran");
        }
        println!("{FINISHED}");
        ExitCode::SUCCESS
    }

    struct Discard;

    impl Sink for Discard {
        fn emit(&mut self, _message: Vec<u8>) -> EngineResult<()> {
            Ok(())
        }
    }

    struct Fixtures {
        native: Vec<u8>,
        rate_confirmation: Vec<u8>,
        cover: Vec<u8>,
        text: Vec<u8>,
        blank: Vec<u8>,
        webp: Vec<u8>,
        heic: Vec<u8>,
    }

    impl Fixtures {
        /// Read before hardening: landlock leaves the fixtures unreadable.
        fn read() -> Self {
            let read = |name: &str| {
                let path = PathBuf::from(env!("CARGO_MANIFEST_DIR"))
                    .join("tests/fixtures")
                    .join(name);
                std::fs::read(&path).unwrap_or_else(|err| panic!("read {}: {err}", path.display()))
            };
            Self {
                native: read("native-10p.pdf"),
                rate_confirmation: read("rate-confirmation.pdf"),
                cover: read("capture-cover-2qr.pdf"),
                text: read("capture-text-0.pdf"),
                blank: read("capture-blank.pdf"),
                webp: read("photo.webp"),
                heic: read("photo.heic"),
            }
        }

        fn jobs(&self, heif: bool) -> Vec<(&'static str, WorkerJob, Vec<u8>)> {
            const PDF: &str = "application/pdf";
            const WEBP: &str = "image/webp";
            const HEIC: &str = "image/heic";
            let mut jobs = vec![
                job(
                    "probe a PDF with its text profile",
                    Operation::Probe(ProbeHeader {
                        document: Some(document(PDF, &self.native)),
                        text_profile: true,
                    }),
                    &[(PDF, &self.native)],
                ),
                job(
                    "render every page of a PDF as WebP",
                    Operation::Render(render(
                        PDF,
                        &self.native,
                        fit(1400, 1800),
                        ImageFormat::Webp,
                    )),
                    &[(PDF, &self.native)],
                ),
                job(
                    "render a PDF at 300 DPI as JPEG",
                    Operation::Render(render(
                        PDF,
                        &self.rate_confirmation,
                        render_size::Size::Dpi(300),
                        ImageFormat::Jpeg,
                    )),
                    &[(PDF, &self.rate_confirmation)],
                ),
                job(
                    "render a PDF thumbnail as PNG",
                    Operation::Render(render(PDF, &self.native, fit(240, 0), ImageFormat::Png)),
                    &[(PDF, &self.native)],
                ),
                job(
                    "inspect captured pages and read their codes",
                    Operation::InspectCapture(InspectCaptureHeader {
                        page_count: 4,
                        ..InspectCaptureHeader::default()
                    }),
                    &[
                        (PDF, &self.cover),
                        (PDF, &self.text),
                        (PDF, &self.blank),
                        (WEBP, &self.webp),
                    ],
                ),
                job(
                    "assemble PDFs and an image, merged and turned",
                    Operation::Assemble(AssembleHeader {
                        source_count: 3,
                        outputs: vec![
                            AssembleOutput {
                                pages: vec![
                                    range(0, 0, u32::MAX, 0),
                                    range(1, 0, u32::MAX, 90),
                                    range(2, 0, 0, 0),
                                ],
                            },
                            AssembleOutput {
                                pages: vec![range(0, 2, 4, 180)],
                            },
                        ],
                        total_size: (self.native.len()
                            + self.rate_confirmation.len()
                            + self.webp.len()) as u64,
                    }),
                    &[
                        (PDF, &self.native),
                        (PDF, &self.rate_confirmation),
                        (WEBP, &self.webp),
                    ],
                ),
                job(
                    "normalize a WebP to JPEG",
                    Operation::NormalizeImage(normalize(WEBP, &self.webp, ImageFormat::Jpeg)),
                    &[(WEBP, &self.webp)],
                ),
            ];
            if heif {
                jobs.extend([
                    job(
                        "probe a HEIC image",
                        Operation::Probe(ProbeHeader {
                            document: Some(document(HEIC, &self.heic)),
                            text_profile: false,
                        }),
                        &[(HEIC, &self.heic)],
                    ),
                    job(
                        "render a HEIC image",
                        Operation::Render(render(HEIC, &self.heic, fit(240, 0), ImageFormat::Webp)),
                        &[(HEIC, &self.heic)],
                    ),
                    job(
                        "normalize a HEIC image to PNG",
                        Operation::NormalizeImage(normalize(HEIC, &self.heic, ImageFormat::Png)),
                        &[(HEIC, &self.heic)],
                    ),
                ]);
            }
            jobs
        }
    }

    fn document(content_type: &str, bytes: &[u8]) -> DocumentHeader {
        DocumentHeader {
            content_type: content_type.into(),
            declared_size: bytes.len() as u64,
        }
    }

    fn fit(max_width: u32, max_height: u32) -> render_size::Size {
        render_size::Size::Fit(FitBox {
            max_width,
            max_height,
        })
    }

    fn render(
        content_type: &str,
        bytes: &[u8],
        size: render_size::Size,
        format: ImageFormat,
    ) -> RenderHeader {
        RenderHeader {
            document: Some(document(content_type, bytes)),
            size: Some(RenderSize { size: Some(size) }),
            format: format as i32,
            quality: 80,
            ..RenderHeader::default()
        }
    }

    fn normalize(content_type: &str, bytes: &[u8], format: ImageFormat) -> NormalizeImageHeader {
        NormalizeImageHeader {
            document: Some(document(content_type, bytes)),
            format: format as i32,
            quality: 0,
            max_width: 1000,
            max_height: 1000,
        }
    }

    fn range(source: u32, first: u32, last: u32, rotation: u32) -> PageRange {
        PageRange {
            source,
            first,
            last,
            rotation,
        }
    }

    /// A job as the supervisor builds one: its inputs laid end to end.
    fn job(
        name: &'static str,
        operation: Operation,
        inputs: &[(&str, &[u8])],
    ) -> (&'static str, WorkerJob, Vec<u8>) {
        let mut data = Vec::new();
        let mut placed = Vec::with_capacity(inputs.len());
        for (index, (content_type, bytes)) in inputs.iter().enumerate() {
            placed.push(JobInput {
                index: u32::try_from(index).expect("index"),
                content_type: (*content_type).to_owned(),
                offset: data.len() as u64,
                length: bytes.len() as u64,
            });
            data.extend_from_slice(bytes);
        }
        let job = WorkerJob {
            operation: Some(operation),
            inputs: placed,
            limits: None,
        };
        (name, job, data)
    }

    #[cfg(target_arch = "x86_64")]
    const AUDIT_ARCH: u32 = 0xC000_003E;
    #[cfg(target_arch = "aarch64")]
    const AUDIT_ARCH: u32 = 0xC000_00B7;

    const LOAD_WORD: u16 = 0x20;
    const JUMP_EQUAL: u16 = 0x15;
    const JUMP_SET: u16 = 0x45;
    const RETURN: u16 = 0x06;
    const KILL_PROCESS: u32 = 0x8000_0000;
    const ALLOW: u32 = 0x7fff_0000;
    const OFFSET_NR: u32 = 0;
    const OFFSET_ARCH: u32 = 4;
    const OFFSET_PROT: u32 = 16 + 2 * 8;
    const OFFSET_FLAGS: u32 = 16 + 3 * 8;
    const SET_MODE_FILTER: libc::c_ulong = 1;
    const FILTER_FLAG_TSYNC: libc::c_ulong = 1;

    fn statement(code: u16, jt: u8, jf: u8, k: u32) -> libc::sock_filter {
        libc::sock_filter { code, jt, jf, k }
    }

    /// Kills the process, every thread of it, on `mmap` of a file with
    /// `PROT_EXEC`. Anonymous memory is left to the sandbox's own filter,
    /// which refuses it executable with `EPERM`, so a library that tries and
    /// falls back is not mistaken for one that loads code.
    fn refuse_executable_file_mappings() -> std::io::Result<()> {
        let prot_exec = u32::try_from(libc::PROT_EXEC).unwrap_or(4);
        let map_anonymous = u32::try_from(libc::MAP_ANONYMOUS).unwrap_or(0x20);
        let mmap = u32::try_from(libc::SYS_mmap).unwrap_or(u32::MAX);
        let mut program = [
            statement(LOAD_WORD, 0, 0, OFFSET_ARCH),
            statement(JUMP_EQUAL, 1, 0, AUDIT_ARCH),
            statement(RETURN, 0, 0, KILL_PROCESS),
            statement(LOAD_WORD, 0, 0, OFFSET_NR),
            statement(JUMP_EQUAL, 0, 5, mmap),
            statement(LOAD_WORD, 0, 0, OFFSET_PROT),
            statement(JUMP_SET, 0, 3, prot_exec),
            statement(LOAD_WORD, 0, 0, OFFSET_FLAGS),
            statement(JUMP_SET, 1, 0, map_anonymous),
            statement(RETURN, 0, 0, KILL_PROCESS),
            statement(RETURN, 0, 0, ALLOW),
        ];
        let filter = libc::sock_fprog {
            len: u16::try_from(program.len()).unwrap_or(u16::MAX),
            filter: program.as_mut_ptr(),
        };
        // SAFETY: `PR_SET_NO_NEW_PRIVS` takes plain integers and only
        // restricts this process.
        if unsafe { libc::prctl(libc::PR_SET_NO_NEW_PRIVS, 1, 0, 0, 0) } != 0 {
            return Err(std::io::Error::last_os_error());
        }
        // SAFETY: `filter` points at `program`, a valid classic BPF program
        // that outlives the call; the kernel copies it.
        let installed = unsafe {
            libc::syscall(
                libc::SYS_seccomp,
                SET_MODE_FILTER,
                FILTER_FLAG_TSYNC,
                &raw const filter,
            )
        };
        if installed != 0 {
            return Err(std::io::Error::last_os_error());
        }
        Ok(())
    }
}
