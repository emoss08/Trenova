import * as matchers from "@testing-library/jest-dom/matchers";
import { APP_CATALOGS } from "@/lib/app-catalogs";
import { requireCatalog } from "@trenova/shared/i18n/runtime";
import { loadRules } from "@trenova/shared/rules/load";
import { RULES_WASM_BYTES } from "@trenova/shared/rules/test-module";
import { cleanup } from "@testing-library/react";
import { afterEach, expect } from "vitest";

expect.extend(matchers);

// The committed rules module, loaded before any test so tests can call the synchronous API
// (rulesSync(), useRules()) without waiting on a load.
await loadRules(RULES_WASM_BYTES);

// What the app's provider requires before its first frame; a route folder's own bundle is
// required by its modules as they load, exactly as in the app (vite/route-catalogs.ts).
requireCatalog(...APP_CATALOGS);

afterEach(() => {
  cleanup();
});

if (typeof Element.prototype.getAnimations === "undefined") {
  Element.prototype.getAnimations = () => [];
}
