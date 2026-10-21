//! Crash containment: whatever a job does to its process, only that job fails
//! and the pool carries on with a fresh child.

#![cfg(target_os = "linux")]

use std::path::PathBuf;
use std::time::Duration;

use prometheus_client::registry::Registry;
use tokio::sync::mpsc;
use tokio::time::Instant;
use trenova_sandbox::{
    ChildSpec, HardeningPolicy, InputWriter, Job, JobError, Limits, Outcome, Pool, PoolConfig,
    SandboxError,
};

fn policy() -> HardeningPolicy {
    HardeningPolicy {
        read_only: ["/usr", "/lib", "/lib64"]
            .iter()
            .map(PathBuf::from)
            .collect(),
        read_write: Vec::new(),
        landlock: true,
        seccomp: true,
    }
}

fn config(size: usize, max_jobs: u32) -> PoolConfig {
    PoolConfig {
        size,
        max_jobs_per_child: max_jobs,
        rss_high_water_bytes: 1 << 30,
        limits: Limits {
            address_space_bytes: 1 << 30,
            cpu_seconds: 120,
            file_size_bytes: 1 << 20,
            open_files: 64,
        },
        max_frame_bytes: 1 << 20,
        start_timeout: Duration::from_secs(10),
        policy: policy(),
        require_hardening: true,
    }
}

fn spec() -> ChildSpec {
    ChildSpec {
        program: PathBuf::from(env!("CARGO_BIN_EXE_trenova-sandbox-test-child")),
        args: Vec::new(),
        init: Vec::new(),
    }
}

async fn pool(size: usize, max_jobs: u32) -> Pool {
    Pool::start(config(size, max_jobs), spec(), &mut Registry::default())
        .await
        .expect("the pool starts hardened")
}

async fn run(
    pool: &Pool,
    body: &str,
    timeout: Duration,
) -> (Result<Outcome, JobError>, Vec<Vec<u8>>) {
    let (tx, mut rx) = mpsc::channel(4);
    let job = Job {
        body: body.as_bytes().to_vec(),
        input: None,
        deadline: Instant::now() + timeout,
    };
    let collector = tokio::spawn(async move {
        let mut frames = Vec::new();
        while let Some(frame) = rx.recv().await {
            frames.push(frame);
        }
        frames
    });
    let result = pool.run(job, tx).await;
    (result, collector.await.expect("collects"))
}

async fn still_works(pool: &Pool) {
    let (result, _) = run(pool, "pid", Duration::from_secs(10)).await;
    assert!(matches!(result, Ok(Outcome::Done(_))), "{result:?}");
}

#[tokio::test]
async fn input_arrives_through_a_sealed_memfd_and_output_streams_back() {
    let pool = pool(1, 100).await;
    let mut writer = InputWriter::new("job", 1 << 20).expect("memfd");
    writer.write_chunk(b"a page of bytes").expect("write");
    let (tx, mut rx) = mpsc::channel(1);
    let result = pool
        .run(
            Job {
                body: b"echo".to_vec(),
                input: Some(writer.seal().expect("seal")),
                deadline: Instant::now() + Duration::from_secs(10),
            },
            tx,
        )
        .await;
    assert_eq!(result, Ok(Outcome::Done(b"echoed".to_vec())));
    assert_eq!(rx.recv().await.as_deref(), Some(&b"a page of bytes"[..]));
}

#[tokio::test]
async fn a_slow_consumer_holds_the_child_back_rather_than_buffering() {
    let pool = pool(1, 100).await;
    let (tx, mut rx) = mpsc::channel(1);
    let consumer = tokio::spawn(async move {
        let mut count = 0;
        while rx.recv().await.is_some() {
            count += 1;
            if count % 100 == 0 {
                tokio::time::sleep(Duration::from_millis(1)).await;
            }
        }
        count
    });
    let result = pool
        .run(
            Job {
                body: b"outputs:2000".to_vec(),
                input: None,
                deadline: Instant::now() + Duration::from_secs(30),
            },
            tx,
        )
        .await;
    assert_eq!(result, Ok(Outcome::Done(b"streamed".to_vec())));
    assert_eq!(consumer.await.expect("consumer"), 2000);
}

#[tokio::test]
async fn a_typed_failure_keeps_the_child() {
    let pool = pool(1, 100).await;
    let (first, _) = run(&pool, "pid", Duration::from_secs(10)).await;
    let (failed, _) = run(&pool, "fail", Duration::from_secs(10)).await;
    let (second, _) = run(&pool, "pid", Duration::from_secs(10)).await;
    assert_eq!(failed, Ok(Outcome::Failed(b"typed failure".to_vec())));
    assert_eq!(first, second, "the same child served both jobs");
}

#[tokio::test]
async fn an_abort_or_a_segfault_fails_only_its_job() {
    let pool = pool(1, 100).await;
    let (aborted, _) = run(&pool, "abort", Duration::from_secs(10)).await;
    assert!(
        matches!(&aborted, Err(JobError::Crashed(status)) if status == "signal 6"),
        "{aborted:?}"
    );
    still_works(&pool).await;
    let (segv, _) = run(&pool, "segv", Duration::from_secs(10)).await;
    assert!(
        matches!(&segv, Err(JobError::Crashed(status)) if status == "signal 11"),
        "{segv:?}"
    );
    still_works(&pool).await;
    assert!(pool.healthy());
}

#[tokio::test]
async fn a_hang_is_killed_at_its_deadline() {
    let pool = pool(1, 100).await;
    let started = std::time::Instant::now();
    let (hung, _) = run(&pool, "hang", Duration::from_millis(300)).await;
    assert_eq!(hung, Err(JobError::DeadlineExceeded));
    assert!(started.elapsed() < Duration::from_secs(5));
    still_works(&pool).await;
}

#[tokio::test]
async fn exhausting_memory_hits_the_address_space_limit() {
    let pool = pool(1, 100).await;
    let (bomb, _) = run(&pool, "alloc", Duration::from_secs(30)).await;
    assert!(matches!(bomb, Err(JobError::Crashed(_))), "{bomb:?}");
    still_works(&pool).await;
}

#[tokio::test]
async fn opening_a_socket_kills_the_child() {
    let pool = pool(1, 100).await;
    let (attempt, _) = run(&pool, "socket", Duration::from_secs(10)).await;
    assert!(
        matches!(&attempt, Err(JobError::Crashed(status)) if status == "signal 31"),
        "seccomp kills with SIGSYS: {attempt:?}"
    );
    still_works(&pool).await;
}

#[tokio::test]
async fn landlock_hides_everything_the_policy_does_not_name() {
    let pool = pool(1, 100).await;
    let (outside, _) = run(&pool, "read:/etc/hostname", Duration::from_secs(10)).await;
    assert_eq!(outside, Ok(Outcome::Failed(b"denied".to_vec())));
    let (inside, _) = run(&pool, "read:/usr/lib/os-release", Duration::from_secs(10)).await;
    assert!(
        matches!(inside, Ok(Outcome::Done(_)))
            || !std::path::Path::new("/usr/lib/os-release").exists(),
        "{inside:?}"
    );
}

#[tokio::test]
async fn a_child_is_replaced_after_its_job_allowance() {
    let pool = pool(1, 2).await;
    let (a, _) = run(&pool, "pid", Duration::from_secs(10)).await;
    let (b, _) = run(&pool, "pid", Duration::from_secs(10)).await;
    let (c, _) = run(&pool, "pid", Duration::from_secs(10)).await;
    assert_eq!(a, b);
    assert_ne!(b, c, "the third job ran on a fresh child");
}

#[tokio::test]
async fn dropping_the_caller_stops_the_job() {
    let pool = pool(1, 100).await;
    let (tx, _rx) = mpsc::channel(1);
    let job = Job {
        body: b"hang".to_vec(),
        input: None,
        deadline: Instant::now() + Duration::from_secs(600),
    };
    let abandoned = tokio::time::timeout(Duration::from_millis(200), pool.run(job, tx)).await;
    assert!(abandoned.is_err(), "the job was still running");
    let started = std::time::Instant::now();
    still_works(&pool).await;
    assert!(
        started.elapsed() < Duration::from_secs(5),
        "the slot came back promptly"
    );
}

#[tokio::test]
async fn a_pool_that_cannot_harden_its_children_does_not_start() {
    let mut config = config(1, 100);
    config.policy.read_write = vec![PathBuf::from("/this/path/does/not/exist")];
    let started = Pool::start(config, spec(), &mut Registry::default()).await;
    assert!(
        matches!(started, Err(SandboxError::Unhardened(_))),
        "{started:?}"
    );
}

#[tokio::test]
async fn concurrent_jobs_spread_over_the_pool() {
    let pool = pool(3, 100).await;
    let mut handles = Vec::new();
    for _ in 0..12 {
        let pool = pool.clone();
        handles.push(tokio::spawn(async move {
            run(&pool, "pid", Duration::from_secs(10)).await.0
        }));
    }
    let mut pids = std::collections::BTreeSet::new();
    for handle in handles {
        let Ok(Outcome::Done(pid)) = handle.await.expect("joins") else {
            panic!("a job failed");
        };
        pids.insert(pid);
    }
    assert!(pids.len() > 1 && pids.len() <= 3, "{pids:?}");
    pool.close().await;
}
