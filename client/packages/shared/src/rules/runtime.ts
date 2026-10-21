// The browser side of the rules engine (docs/engineering/rules-engine.md): the same
// rules.wasm the TMS runs in wazero, called synchronously once it has loaded.
//
// A call is one JSON request in and one JSON response out through the module's three
// exports: the host asks rules_alloc for a buffer, writes the UTF-8 request into it, calls
// rules_call, reads the response at the address and length packed into the u64 it returns,
// and hands both buffers back with rules_free. Nothing here is asynchronous, so render code
// and CodeMirror extensions can call the module directly once loadRules() has resolved.
//
// An instance that traps (a panic inside the module, always a bug) is replaced at once from
// the compiled module, as is one whose memory grew past RECYCLE_MEMORY_BYTES: linear memory
// never shrinks, and one large call should not hold its high-water mark for the session.

export const RULES_VERSION_OP = "rules.version";

/** Linear memory past which an instance is replaced after a call, as on the server. */
export const RECYCLE_MEMORY_BYTES = 64 * 1024 * 1024;

/** The module's request error codes: each one is a bug in the caller, never user input. */
export type RulesRequestErrorCode = "malformed" | "unknown_op" | "invalid_input";

/** The module refused the request. The detail is developer English: log it, never show it. */
export class RulesRequestError extends Error {
  readonly op: string;
  readonly code: RulesRequestErrorCode;
  readonly detail: string;

  constructor(op: string, code: RulesRequestErrorCode, detail: string) {
    super(`rules: ${op}: ${code}: ${detail}`);
    this.name = "RulesRequestError";
    this.op = op;
    this.code = code;
    this.detail = detail;
  }
}

/** The module trapped: an internal bug. The instance has already been replaced. */
export class RulesTrapError extends Error {
  readonly op: string;

  constructor(op: string, cause: unknown) {
    super(`rules: ${op}: the rules module trapped`, { cause });
    this.name = "RulesTrapError";
    this.op = op;
  }
}

/** rulesSync() was called before loadRules() resolved. */
export class RulesNotLoadedError extends Error {
  constructor() {
    super("rules: the rules module has not loaded yet; await loadRules() or use useRules()");
    this.name = "RulesNotLoadedError";
  }
}

/** What rules.version answers. */
export type RulesVersion = {
  version: string;
  operations: string[];
};

type RulesExports = {
  memory: WebAssembly.Memory;
  rules_alloc: (length: number) => number;
  rules_call: (pointer: number, length: number) => bigint;
  rules_free: (pointer: number, length: number) => void;
};

const OK_PREFIX = '{"ok":';
const encoder = new TextEncoder();
const decoder = new TextDecoder();

function rulesExports(instance: WebAssembly.Instance): RulesExports {
  const { memory, rules_alloc, rules_call, rules_free } = instance.exports;
  if (
    !(memory instanceof WebAssembly.Memory) ||
    typeof rules_alloc !== "function" ||
    typeof rules_call !== "function" ||
    typeof rules_free !== "function"
  ) {
    throw new TypeError("rules: the module does not export the rules ABI");
  }
  return {
    memory,
    rules_alloc: rules_alloc as RulesExports["rules_alloc"],
    rules_call: rules_call as RulesExports["rules_call"],
    rules_free: rules_free as RulesExports["rules_free"],
  };
}

function isRequestErrorCode(code: unknown): code is RulesRequestErrorCode {
  return code === "malformed" || code === "unknown_op" || code === "invalid_input";
}

/** One loaded rules module. Get it from loadRules(), rulesSync() or useRules(). */
export class RulesRuntime {
  readonly #module: WebAssembly.Module;
  #exports: RulesExports;

  constructor(module: WebAssembly.Module, instance: WebAssembly.Instance) {
    this.#module = module;
    this.#exports = rulesExports(instance);
  }

  /** Runs `op` with `input` (any JSON value; omitted means `{}`) and returns its output. */
  call(op: string, input?: unknown): unknown {
    const encoded = input === undefined ? undefined : JSON.stringify(input);
    return JSON.parse(this.callRaw(op, encoded));
  }

  /**
   * Runs `op` with input that is already JSON text, such as a cached rate table, and
   * returns the output as JSON text, so neither side is parsed twice.
   */
  callRaw(op: string, input?: string): string {
    const name = JSON.stringify(op);
    const request = encoder.encode(
      input === undefined ? `{"op":${name}}` : `{"op":${name},"input":${input}}`,
    );
    let response: string;
    try {
      response = this.#exchange(request);
    } catch (error) {
      if (error instanceof WebAssembly.RuntimeError) {
        this.#replace();
        throw new RulesTrapError(op, error);
      }
      throw error;
    }
    if (this.#exports.memory.buffer.byteLength > RECYCLE_MEMORY_BYTES) this.#replace();
    return unwrap(op, response);
  }

  /** The crate's version and every operation it answers. */
  version(): RulesVersion {
    return this.call(RULES_VERSION_OP) as RulesVersion;
  }

  #exchange(request: Uint8Array): string {
    const abi = this.#exports;
    const pointer = request.length === 0 ? 0 : abi.rules_alloc(request.length);
    if (request.length > 0) new Uint8Array(abi.memory.buffer, pointer, request.length).set(request);
    const packed = abi.rules_call(pointer, request.length);
    const address = Number(BigInt.asUintN(64, packed) >> 32n);
    const length = Number(BigInt.asUintN(32, packed));
    const text = decoder.decode(new Uint8Array(abi.memory.buffer, address, length));
    abi.rules_free(address, length);
    if (request.length > 0) abi.rules_free(pointer, request.length);
    return text;
  }

  #replace(): void {
    this.#exports = rulesExports(new WebAssembly.Instance(this.#module, {}));
  }
}

function unwrap(op: string, response: string): string {
  if (response.startsWith(OK_PREFIX) && response.endsWith("}")) {
    return response.slice(OK_PREFIX.length, -1);
  }
  const parsed = JSON.parse(response) as { error?: { code?: unknown; detail?: unknown } };
  const code = parsed.error?.code;
  if (!isRequestErrorCode(code)) {
    throw new RulesTrapError(op, new TypeError("the response is not a rules envelope"));
  }
  throw new RulesRequestError(op, code, String(parsed.error?.detail ?? ""));
}

let loaded: RulesRuntime | null = null;
const listeners = new Set<() => void>();

/** Makes `runtime` the loaded module. Called by loadRules(). */
export function publishRules(runtime: RulesRuntime): void {
  loaded = runtime;
  for (const listener of listeners) listener();
}

/** The loaded module. Throws RulesNotLoadedError before loadRules() has resolved. */
export function rulesSync(): RulesRuntime {
  if (loaded === null) throw new RulesNotLoadedError();
  return loaded;
}

export function isRulesLoaded(): boolean {
  return loaded !== null;
}

/** For useSyncExternalStore: notified once, when the module loads. */
export function subscribeRules(listener: () => void): () => void {
  listeners.add(listener);
  return () => {
    listeners.delete(listener);
  };
}

/** For useSyncExternalStore: the loaded module, or null. */
export function getRulesSnapshot(): RulesRuntime | null {
  return loaded;
}
