// The EDI module in Node, for the mock API: the same edi.wasm the Lading studio loads, called
// through its ABI (`edi_alloc`, `edi_call`, `edi_free`). Every check, test, run and replay the
// mock serves is the module's answer, so the studio and the mock agree the way the studio and the
// TMS do.
import { readFileSync } from "node:fs";
import { dirname, resolve } from "node:path";
import { fileURLToPath } from "node:url";

const HERE = dirname(fileURLToPath(import.meta.url));
const WASM = resolve(HERE, "../../../../packages/shared/src/edi/edi.wasm");

const { instance } = await WebAssembly.instantiate(readFileSync(WASM), {});
const { memory, edi_alloc: alloc, edi_call: invoke, edi_free: free } = instance.exports;

/** A request the module refused: its code (`invalid_input`, …) and detail. */
export class EdiError extends Error {
  constructor(op, code, detail) {
    super(`${op}: ${code}: ${detail}`);
    this.op = op;
    this.code = code;
    this.detail = detail;
  }
}

/** Calls one operation and answers its output, or throws the module's refusal. */
export function edi(op, input) {
  const bytes = new TextEncoder().encode(JSON.stringify({ op, input }));
  const ptr = alloc(bytes.length);
  new Uint8Array(memory.buffer, ptr, bytes.length).set(bytes);
  const packed = BigInt.asUintN(64, invoke(ptr, bytes.length));
  const outPtr = Number(packed >> 32n);
  const outLen = Number(packed & 0xffffffffn);
  const text = new TextDecoder().decode(new Uint8Array(memory.buffer, outPtr, outLen));
  free(outPtr, outLen);
  free(ptr, bytes.length);
  const response = JSON.parse(text);
  if (response.error) throw new EdiError(op, response.error.code, response.error.detail);
  return response.ok;
}
