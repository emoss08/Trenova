//! Arbitrary job bodies and input layouts, as a child receives them: a job
//! decodes and its inputs are cut from the data, or the job is refused.

#![no_main]

use libfuzzer_sys::fuzz_target;
use prost::Message;
use trenova_imaging::engine::slices;
use trenova_imaging::protocol::WorkerJob;

fuzz_target!(|data: &[u8]| {
    let (body, input) = data.split_at(data.len() / 2);
    if let Ok(job) = WorkerJob::decode(body)
        && let Ok(inputs) = slices(&job.inputs, input)
    {
        for slice in inputs {
            assert!(slice.bytes.len() <= input.len());
        }
    }
});
