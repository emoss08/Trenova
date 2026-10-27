# Vendored: monty 1.3.0

Pydantic's Monty (MIT, <https://github.com/pydantic/monty>), the Python interpreter
`trenova-scripts` runs agent scripts in. It is vendored only to carry one change, and is
otherwise the crates.io release byte for byte (its tests and test corpus are left out).

## The change

`src/heap_data.rs`, `MAX_HEAP_DATA_SIZE` on 32-bit targets: 56 becomes 64.

The published 1.2.0 and 1.3.0 fail their own compile-time check on `wasm32-wasip1`: `HeapData`
is 64 bytes there, past the 56-byte budget the check holds it to, so the crate does not build
for WebAssembly at all. The budget is a memory-efficiency target, not an invariant: every heap
slot is 8 bytes wider on 32-bit, and nothing else changes.

## Dropping it

When a Monty release builds for `wasm32-wasip1` unpatched, delete this directory and the
`[patch.crates-io]` entry in `services/Cargo.toml`, bump the `monty*` versions in
`[workspace.dependencies]` together, and rebuild the module (`task rust:wasm-scripts`).
