//! The WebAssembly boundary: three exports and no imports.
//!
//! A host writes a UTF-8 JSON request into memory it got from `edi_alloc`, calls `edi_call` with
//! its address and length, and gets back one `u64`: the response's address in the high 32 bits
//! and its length in the low 32. It reads the response, then hands both buffers back with
//! `edi_free`. The Go host (`services/tms/pkg/x12`, on the shared `services/tms/pkg/wasmmodule`)
//! does exactly this and nothing else. The exports are `trenova_module_kit::export_abi!`, shared
//! with every module.

trenova_module_kit::export_abi!(edi_alloc, edi_call, edi_free, crate::ops::dispatch);
