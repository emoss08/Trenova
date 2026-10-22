// rules.wasm and its .sha256 sidecar for tests, read from disk by Vite when the test is
// transformed, so a Vitest setup file can load the module without the network or node:fs.
// Only test setups and tests import this; the app fetches rules.wasm through load.ts.

import inlined from "./rules.wasm?inline";
import sidecar from "./rules.wasm.sha256?raw";

function decodeDataUrl(url: string): Uint8Array<ArrayBuffer> {
  const binary = atob(url.slice(url.indexOf(",") + 1));
  const bytes = new Uint8Array(binary.length);
  for (let index = 0; index < binary.length; index += 1) bytes[index] = binary.charCodeAt(index);
  return bytes;
}

/** The committed rules.wasm. */
export const RULES_WASM_BYTES = decodeDataUrl(inlined);

/** The committed rules.wasm.sha256, in sha256sum format. */
export const RULES_WASM_SHA256 = sidecar;
