// Loads edi.wasm once per page: the EDI module the TMS reads and writes X12 and runs Lading
// with (docs/engineering/lading.md), here for the Lading studio's editor, tests and runs. The
// module is a content-hashed build asset fetched on first use rather than at startup. A load
// that fails is forgotten, so the next caller tries again.

import { instantiateModule, WasmModuleRuntime } from "@trenova/shared/lib/wasm-module";

/** The module could not be fetched. */
export class EdiLoadError extends Error {
  constructor(message: string, options?: ErrorOptions) {
    super(message, options);
    this.name = "EdiLoadError";
  }
}

/** One loaded EDI module. */
export type EdiRuntime = WasmModuleRuntime;

let pending: Promise<EdiRuntime> | null = null;
let loaded: EdiRuntime | null = null;
const listeners = new Set<() => void>();

/**
 * Loads the EDI module and resolves with it. Every call returns the same promise until it
 * settles; a rejected load is not kept. `bytes` loads from memory instead of the network
 * (tests read edi.wasm from disk); it is ignored once a load has started.
 */
export function loadEdi(bytes?: BufferSource): Promise<EdiRuntime> {
  if (loaded !== null) return Promise.resolve(loaded);
  if (pending !== null) return pending;
  const loading: Promise<EdiRuntime> = instantiateModule(
    new URL("./edi.wasm", import.meta.url),
    bytes,
    (message, cause) => new EdiLoadError(`edi: ${message}`, { cause }),
  )
    .then(({ module, instance }) => {
      const runtime = new WasmModuleRuntime("edi", module, instance);
      loaded = runtime;
      for (const listener of listeners) listener();
      return runtime;
    })
    .catch((error: unknown) => {
      if (pending === loading) pending = null;
      throw error;
    });
  pending = loading;
  return loading;
}

/** For useSyncExternalStore: calls `listener` when the module has loaded. */
export function subscribeEdi(listener: () => void): () => void {
  listeners.add(listener);
  return () => listeners.delete(listener);
}

/** For useSyncExternalStore: the loaded module, or null. */
export function getEdiSnapshot(): EdiRuntime | null {
  return loaded;
}
