import "@testing-library/jest-dom/vitest";
import { APP_CATALOGS } from "@/lib/app-catalogs";
import { requireCatalog } from "@trenova/shared/i18n/runtime";
import { cleanup } from "@testing-library/react";
import { afterEach } from "vitest";

// What the app's provider requires before its first frame; a route folder's own bundle is
// required by its modules as they load, exactly as in the app (vite/route-catalogs.ts).
requireCatalog(...APP_CATALOGS);

afterEach(() => {
  cleanup();
});

if (typeof Element.prototype.getAnimations === "undefined") {
  Element.prototype.getAnimations = () => [];
}
