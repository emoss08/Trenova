//! Compiles every contract in `services/proto` with protox, a pure-Rust protobuf
//! compiler, and generates prost messages and tonic clients and servers from it.

use std::path::PathBuf;

const FILES: &[&str] = &["trenova/imaging/v1/imaging.proto"];

fn main() -> Result<(), Box<dyn std::error::Error>> {
    let manifest = PathBuf::from(std::env::var("CARGO_MANIFEST_DIR")?);
    let root = manifest.join("../../proto");
    for file in FILES {
        println!("cargo:rerun-if-changed={}", root.join(file).display());
    }
    let descriptors = protox::compile(FILES, [&root])?;
    tonic_prost_build::configure()
        .build_server(true)
        .build_client(true)
        .compile_fds(descriptors)?;
    Ok(())
}
