//! The WebAssembly boundary: three exports and no imports.
//!
//! A host writes a UTF-8 JSON request into memory it got from `rules_alloc`, calls `rules_call`
//! with its address and length, and gets back one `u64`: the response's address in the high 32
//! bits and its length in the low 32. It reads the response, then hands both buffers back with
//! `rules_free`. The Go host (`services/tms/pkg/rules`) and the browser loader
//! (`client/packages/shared/src/rules`) do exactly this and nothing else, so one artifact serves
//! both. The exports are `trenova_module_kit::export_abi!`, shared with every module.

trenova_module_kit::export_abi!(rules_alloc, rules_call, rules_free, crate::ops::dispatch);
