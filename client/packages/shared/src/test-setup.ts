import * as matchers from "@testing-library/jest-dom/matchers";
import { loadRules } from "@trenova/shared/rules/load";
import { RULES_WASM_BYTES } from "@trenova/shared/rules/test-module";
import { expect } from "vitest";

expect.extend(matchers);

// The committed rules module, loaded before any test so tests can call the synchronous API
// (rulesSync(), useRules()) without waiting on a load.
await loadRules(RULES_WASM_BYTES);

if (typeof Element.prototype.getAnimations === "undefined") {
  Element.prototype.getAnimations = () => [];
}
