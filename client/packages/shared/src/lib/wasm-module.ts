// A WebAssembly module built on services/crates/module-kit, called synchronously from the
// browser: the same bytes the TMS runs in wazero. A call is one JSON request in and one JSON
// response out through the module's three exports, `<prefix>_alloc`, `<prefix>_call` and
// `<prefix>_free`. The host writes the UTF-8 request into a buffer from alloc, calls call, reads
// the response at the address and length packed into the u64 it returns, and frees both.
//
// An instance that traps (a panic inside the module, always a bug) is replaced at once from the
// compiled module, as is one whose memory grew past RECYCLE_MEMORY_BYTES: linear memory never
// shrinks, and one large call should not hold its high-water mark for the session.

/** Linear memory past which an instance is replaced after a call, as on the server. */
export const MODULE_RECYCLE_MEMORY_BYTES = 64 * 1024 * 1024;

/** The module's request error codes: each one is a bug in the caller, never user input. */
export type ModuleRequestErrorCode = "malformed" | "unknown_op" | "invalid_input";

/** The module refused the request. The detail is developer English: log it, never show it. */
export class ModuleRequestError extends Error {
  readonly op: string;
  readonly code: ModuleRequestErrorCode;
  readonly detail: string;

  constructor(op: string, code: ModuleRequestErrorCode, detail: string) {
    // i18n-ignore: developer English for logs; a request error is a caller bug, never shown
    super(`${op}: ${code}: ${detail}`);
    this.name = "ModuleRequestError";
    this.op = op;
    this.code = code;
    this.detail = detail;
  }
}

/** The module trapped: an internal bug. The instance has already been replaced. */
export class ModuleTrapError extends Error {
  readonly op: string;

  constructor(op: string, cause: unknown) {
    // i18n-ignore: developer English for logs; a trap is an internal bug, never shown
    super(`${op}: the module trapped`, { cause });
    this.name = "ModuleTrapError";
    this.op = op;
  }
}

type ModuleExports = {
  memory: WebAssembly.Memory;
  alloc: (length: number) => number;
  call: (pointer: number, length: number) => bigint;
  free: (pointer: number, length: number) => void;
};

const OK_PREFIX = '{"ok":';
const encoder = new TextEncoder();
const decoder = new TextDecoder();

function moduleExports(prefix: string, instance: WebAssembly.Instance): ModuleExports {
  const memory = instance.exports.memory;
  const alloc = instance.exports[`${prefix}_alloc`];
  const call = instance.exports[`${prefix}_call`];
  const free = instance.exports[`${prefix}_free`];
  if (
    !(memory instanceof WebAssembly.Memory) ||
    typeof alloc !== "function" ||
    typeof call !== "function" ||
    typeof free !== "function"
  ) {
    throw new TypeError(`the module does not export the ${prefix} ABI`);
  }
  return {
    memory,
    alloc: alloc as ModuleExports["alloc"],
    call: call as ModuleExports["call"],
    free: free as ModuleExports["free"],
  };
}

function isRequestErrorCode(code: unknown): code is ModuleRequestErrorCode {
  return code === "malformed" || code === "unknown_op" || code === "invalid_input";
}

/** One loaded module, called synchronously. */
export class WasmModuleRuntime {
  readonly prefix: string;
  readonly #module: WebAssembly.Module;
  #exports: ModuleExports;

  constructor(prefix: string, module: WebAssembly.Module, instance: WebAssembly.Instance) {
    this.prefix = prefix;
    this.#module = module;
    this.#exports = moduleExports(prefix, instance);
  }

  /** Runs `op` with `input` (any JSON value; omitted means `{}`) and returns its output. */
  call(op: string, input?: unknown): unknown {
    const encoded = input === undefined ? undefined : JSON.stringify(input);
    return JSON.parse(this.callRaw(op, encoded));
  }

  /** Runs `op` with input that is already JSON text and returns the output as JSON text. */
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
        throw new ModuleTrapError(op, error);
      }
      throw error;
    }
    if (this.#exports.memory.buffer.byteLength > MODULE_RECYCLE_MEMORY_BYTES) this.#replace();
    return unwrap(op, response);
  }

  #exchange(request: Uint8Array): string {
    const abi = this.#exports;
    const pointer = request.length === 0 ? 0 : abi.alloc(request.length);
    if (request.length > 0) new Uint8Array(abi.memory.buffer, pointer, request.length).set(request);
    const packed = abi.call(pointer, request.length);
    const address = Number(BigInt.asUintN(64, packed) >> 32n);
    const length = Number(BigInt.asUintN(32, packed));
    const text = decoder.decode(new Uint8Array(abi.memory.buffer, address, length));
    abi.free(address, length);
    if (request.length > 0) abi.free(pointer, request.length);
    return text;
  }

  #replace(): void {
    this.#exports = moduleExports(this.prefix, new WebAssembly.Instance(this.#module, {}));
  }
}

function unwrap(op: string, response: string): string {
  if (response.startsWith(OK_PREFIX) && response.endsWith("}")) {
    return response.slice(OK_PREFIX.length, -1);
  }
  const parsed = JSON.parse(response) as { error?: { code?: unknown; detail?: unknown } };
  const code = parsed.error?.code;
  if (!isRequestErrorCode(code)) {
    throw new ModuleTrapError(op, new TypeError("the response is not a module envelope"));
  }
  throw new ModuleRequestError(op, code, String(parsed.error?.detail ?? ""));
}

/** Instantiates a module from a URL, compiling it while it streams in when the host allows. */
export async function instantiateModule(
  url: URL,
  bytes: BufferSource | undefined,
  onFetchError: (message: string, cause?: unknown) => Error,
): Promise<{ module: WebAssembly.Module; instance: WebAssembly.Instance }> {
  if (bytes !== undefined) return WebAssembly.instantiate(bytes, {});

  let response: Response;
  try {
    response = await fetch(url, { credentials: "same-origin" });
  } catch (error) {
    throw onFetchError(`could not fetch ${url.pathname}`, error);
  }
  if (!response.ok) {
    throw onFetchError(`fetching ${url.pathname} answered ${response.status}`);
  }
  const type = response.headers.get("content-type") ?? "";
  if (
    typeof WebAssembly.instantiateStreaming === "function" &&
    type.startsWith("application/wasm")
  ) {
    try {
      return await WebAssembly.instantiateStreaming(response, {});
    } catch (error) {
      if (!(error instanceof TypeError) || response.bodyUsed) throw error;
    }
  }
  return WebAssembly.instantiate(await response.arrayBuffer(), {});
}
