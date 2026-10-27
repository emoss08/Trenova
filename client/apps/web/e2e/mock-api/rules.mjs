// The rules module in Node, for the mock API: the same rules.wasm the client loads, called
// through its ABI (`rules_alloc`, `rules_call`, `rules_free`). Every check, test, preview and
// scaffold the mock serves is the module's answer, so the Studio, the tests panel and the
// server agree the way they do against the real TMS.
import { readFileSync } from "node:fs";
import { dirname, resolve } from "node:path";
import { fileURLToPath } from "node:url";

const HERE = dirname(fileURLToPath(import.meta.url));
const WASM = resolve(HERE, "../../../../packages/shared/src/rules/rules.wasm");

const { instance } = await WebAssembly.instantiate(readFileSync(WASM), {});
const { memory, rules_alloc: alloc, rules_call: invoke, rules_free: free } = instance.exports;

/** A request the module refused: its code (`invalid_input`, `check`, …) and detail. */
export class RulesError extends Error {
  constructor(op, code, detail) {
    super(`${op}: ${code}: ${detail}`);
    this.op = op;
    this.code = code;
    this.detail = detail;
  }
}

/** Calls one operation and answers its output, or throws the module's refusal. */
export function rules(op, input) {
  const bytes = new TextEncoder().encode(JSON.stringify({ op, input }));
  const ptr = alloc(bytes.length);
  new Uint8Array(memory.buffer, ptr, bytes.length).set(bytes);
  const packed = invoke(ptr, bytes.length);
  const outPtr = Number(packed >> 32n);
  const outLen = Number(packed & 0xffffffffn);
  const text = new TextDecoder().decode(new Uint8Array(memory.buffer, outPtr, outLen));
  free(ptr, bytes.length);
  free(outPtr, outLen);
  const response = JSON.parse(text);
  if (response.error) throw new RulesError(op, response.error.code, response.error.detail);
  return response.ok;
}

const ROOTS = ["shipment", "lane", "customer", "fuel", "rating"];

/**
 * The module's sample record in the wire form, read back through a trace: the record a test or
 * a preview with no shipment rates. A shipment's record is this with its own values over it.
 */
export const SAMPLE_RECORD = (() => {
  const source = `tariff "record"\n${ROOTS.map((root) => `let ${root}Root = ${root}`).join("\n")}\n`;
  const { results } = rules("tariff.evaluate", {
    context: {},
    record: null,
    programs: [{ id: "record", source, inputs: {} }],
    trace: true,
  });
  const binds = new Map(
    (results[0]?.trace ?? [])
      .filter((event) => event.kind === "bind")
      .map((event) => [event.name, event.value]),
  );
  return Object.fromEntries(ROOTS.map((root) => [root, binds.get(`${root}Root`)]));
})();

function merge(base, overlay) {
  if (overlay === null || typeof overlay !== "object" || Array.isArray(overlay)) return overlay;
  if (base === null || typeof base !== "object" || Array.isArray(base))
    return structuredClone(overlay);
  const out = { ...base };
  for (const [key, value] of Object.entries(overlay)) out[key] = merge(base[key], value);
  return out;
}

/** A stop in the wire form, the sample's fields with `extra` over them. */
export function stop(kind, sequence, extra = {}) {
  const base = structuredClone(SAMPLE_RECORD.shipment.stops[0]);
  return merge({ ...base, kind, sequence: { $n: String(sequence) } }, extra);
}

/**
 * A record from the sample with `overlay` over it. A stop list sets `pickup` and `delivery` to
 * its first and last stop, the way the TMS builds a shipment's record.
 */
export function recordFrom(overlay) {
  const record = merge(SAMPLE_RECORD, overlay);
  const stops = record.shipment.stops;
  if (overlay?.shipment?.stops && stops.length > 0) {
    record.shipment.pickup = stops[0];
    record.shipment.delivery = stops.at(-1);
  }
  return record;
}
