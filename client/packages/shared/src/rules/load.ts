// Loads rules.wasm once per page. The module is a content-hashed build asset (Vite emits
// `new URL("./rules.wasm", import.meta.url)` as assets/rules.<hash>.wasm), fetched on first
// use rather than at startup, compiled while it streams in, and kept by the service worker
// like any other hashed asset. A load that fails is forgotten, so the next caller tries again.

import { publishRules, RulesRuntime } from "./runtime";

/** The module could not be fetched. */
export class RulesLoadError extends Error {
  constructor(message: string, options?: ErrorOptions) {
    super(message, options);
    this.name = "RulesLoadError";
  }
}

type Instantiated = { module: WebAssembly.Module; instance: WebAssembly.Instance };

let pending: Promise<RulesRuntime> | null = null;

/**
 * Loads the rules module and resolves with it. Every call returns the same promise until it
 * settles; a rejected load is not kept, so the next call fetches again. `bytes` loads from
 * memory instead of the network (tests read rules.wasm from disk); it is ignored once a load
 * has started.
 */
export function loadRules(bytes?: BufferSource): Promise<RulesRuntime> {
  if (pending !== null) return pending;
  const loading: Promise<RulesRuntime> = instantiate(bytes)
    .then(({ module, instance }) => {
      const runtime = new RulesRuntime(module, instance);
      publishRules(runtime);
      return runtime;
    })
    .catch((error: unknown) => {
      if (pending === loading) pending = null;
      throw error;
    });
  pending = loading;
  return loading;
}

async function instantiate(bytes: BufferSource | undefined): Promise<Instantiated> {
  if (bytes !== undefined) return WebAssembly.instantiate(bytes, {});

  const url = new URL("./rules.wasm", import.meta.url);
  let response: Response;
  try {
    response = await fetch(url, { credentials: "same-origin" });
  } catch (error) {
    throw new RulesLoadError("rules: could not fetch rules.wasm", { cause: error });
  }
  if (!response.ok) {
    throw new RulesLoadError(`rules: fetching rules.wasm answered ${response.status}`);
  }
  const type = response.headers.get("content-type") ?? "";
  if (
    typeof WebAssembly.instantiateStreaming === "function" &&
    type.startsWith("application/wasm")
  ) {
    try {
      return await WebAssembly.instantiateStreaming(response, {});
    } catch (error) {
      // A host that refuses the response before reading it (a TypeError, the body untouched)
      // can still compile the bytes; anything else is a real failure.
      if (!(error instanceof TypeError) || response.bodyUsed) throw error;
    }
  }
  return WebAssembly.instantiate(await response.arrayBuffer(), {});
}
