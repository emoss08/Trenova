import assert from "node:assert/strict";
import { readdir, readFile } from "node:fs/promises";
import { join, resolve, sep } from "node:path";
import { describe, it } from "node:test";
import { fileURLToPath } from "node:url";
import { bundleFileName, catalogBundle, compareBundles } from "./bundles.mjs";
import { featureArea } from "./extract-ts.mjs";

const repoRoot = resolve(fileURLToPath(import.meta.url), "../../..");
const p = (...parts) => parts.join(sep);

describe("featureArea", () => {
  it("labels a web route folder by its folder", () => {
    assert.equal(featureArea(p("client", "apps", "web", "src", "routes", "shipment", "page.tsx")), "routes/shipment");
    assert.equal(
      featureArea(p("client", "apps", "web", "src", "routes", "shipment", "_components", "form.tsx")),
      "routes/shipment",
    );
  });

  it("puts a file directly in the web routes folder with the shell", () => {
    assert.equal(featureArea(p("client", "apps", "web", "src", "routes", "app-layout.tsx")), "apps/web/routes");
  });

  it("keeps the driver portal's routes its own", () => {
    assert.equal(featureArea(p("client", "apps", "dash", "src", "routes", "login.tsx")), "apps/dash/routes");
    assert.equal(featureArea(p("client", "apps", "dash", "src", "_components", "card.tsx")), "apps/dash/_components");
  });

  it("labels shared code by its top folder, even one called routes", () => {
    assert.equal(featureArea(p("client", "packages", "shared", "src", "components", "button.tsx")), "packages/shared/components");
    assert.equal(featureArea(p("client", "packages", "cloud", "src", "routes", "billing", "page.tsx")), "packages/cloud/routes");
  });
});

describe("catalogBundle", () => {
  it("ships a string used in one route folder with that folder", () => {
    assert.equal(catalogBundle(["routes/shipment"]), "routes/shipment");
  });

  it("ignores the Go areas a string shares with the server", () => {
    assert.equal(catalogBundle(["domain/shipment", "routes/shipment", "service/shipment"]), "routes/shipment");
  });

  it("ships a string two route folders share with the web shell", () => {
    assert.equal(catalogBundle(["routes/customer", "routes/shipment"]), "web");
  });

  it("ships a string the shell renders with the shell, even if one route does too", () => {
    assert.equal(catalogBundle(["apps/web/components"]), "web");
    assert.equal(catalogBundle(["apps/web/components", "routes/shipment"]), "web");
    assert.equal(catalogBundle(["apps/web/routes"]), "web");
  });

  it("ships shared-package strings in core for both apps", () => {
    assert.equal(catalogBundle(["packages/shared/components"]), "core");
    assert.equal(catalogBundle(["packages/shared/lib", "routes/shipment"]), "core");
  });

  it("ships a string both apps render in core", () => {
    assert.equal(catalogBundle(["apps/dash/_components", "routes/shipment"]), "core");
    assert.equal(catalogBundle(["apps/dash/routes", "apps/web/components"]), "core");
  });

  it("keeps the driver portal's own strings out of the web app", () => {
    assert.equal(catalogBundle(["apps/dash/routes"]), "dash");
    assert.equal(catalogBundle(["apps/dash/_components", "apps/dash/routes"]), "dash");
  });

  it("never strands a string with no client area", () => {
    assert.equal(catalogBundle([]), "core");
    assert.equal(catalogBundle(["domain/shipment"]), "core");
  });
});

describe("bundleFileName", () => {
  it("flattens a route bundle into one file name", () => {
    assert.equal(bundleFileName("core"), "core.json");
    assert.equal(bundleFileName("routes/carrier-settlement"), "routes.carrier-settlement.json");
  });
});

describe("compareBundles", () => {
  it("orders the startup bundles first, then routes by name", () => {
    const sorted = ["routes/b", "dash", "routes/a", "web", "core"].sort(compareBundles);
    assert.deepEqual(sorted, ["core", "web", "dash", "routes/a", "routes/b"]);
  });
});

// The web app's build requires `routes/<dir>` for modules in src/routes/<dir>; a bundle with
// no folder behind it would never be loaded, and its strings would render in English.
describe("emitted catalogs", () => {
  it("names a real route folder for every route bundle", async () => {
    const module = await readFile(join(repoRoot, "client/packages/shared/src/i18n/generated/locales.ts"), "utf8");
    const list = /export const CATALOG_BUNDLES = \[([^\]]*)\]/.exec(module);
    assert.ok(list, "CATALOG_BUNDLES not found in the generated locale module");
    const bundles = [...list[1].matchAll(/"([^"]+)"/g)].map((m) => m[1]);

    const folders = new Set(
      (await readdir(join(repoRoot, "client/apps/web/src/routes"), { withFileTypes: true }))
        .filter((entry) => entry.isDirectory())
        .map((entry) => `routes/${entry.name}`),
    );
    const orphans = bundles.filter((b) => b.startsWith("routes/") && !folders.has(b));
    assert.deepEqual(orphans, []);
    assert.ok(bundles.includes("core") && bundles.includes("web"));
  });
});
