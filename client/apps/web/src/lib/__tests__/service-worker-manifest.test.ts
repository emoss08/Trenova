import { describe, expect, it } from "vitest";
import { shellManifest } from "../../../vite/service-worker.ts";

const HTML = `<!doctype html><html><head>
<script type="module" crossorigin src="/assets/js/main.abc.js"></script>
<link rel="modulepreload" crossorigin href="/assets/js/react-vendor.def.js">
<link rel="stylesheet" crossorigin href="/assets/initial-styles.ghi.css">
</head></html>`;

const FILES = [
  "index.html",
  "assets/js/main.abc.js",
  "assets/js/react-vendor.def.js",
  "assets/js/page.lazy.js",
  "assets/initial-styles.ghi.css",
  "assets/fonts/inter.woff2",
  "logo.ico",
  "sw.js",
];

describe("shellManifest", () => {
  it("lists what the shell loads before rendering, and every build file under assets", () => {
    const manifest = shellManifest({ base: "/", fileNames: FILES, html: HTML });

    expect(manifest.entry).toBe("/assets/js/main.abc.js");
    expect(manifest.startup).toEqual([
      "/assets/js/main.abc.js",
      "/assets/js/react-vendor.def.js",
      "/assets/initial-styles.ghi.css",
    ]);
    expect(manifest.assets).toEqual([
      "/assets/fonts/inter.woff2",
      "/assets/initial-styles.ghi.css",
      "/assets/js/main.abc.js",
      "/assets/js/page.lazy.js",
      "/assets/js/react-vendor.def.js",
    ]);
  });

  it("names a version that changes with the build and not with file order", () => {
    const version = shellManifest({ base: "/", fileNames: FILES, html: HTML }).version;

    expect(shellManifest({ base: "/", fileNames: [...FILES].reverse(), html: HTML }).version).toBe(
      version,
    );
    expect(
      shellManifest({
        base: "/",
        fileNames: [...FILES, "assets/js/new-route.xyz.js"],
        html: HTML,
      }).version,
    ).not.toBe(version);
    expect(
      shellManifest({ base: "/", fileNames: FILES, html: HTML.replace("main.abc", "main.abd") })
        .version,
    ).not.toBe(version);
  });

  it("leaves the worker itself out, so writing it cannot change its own version", () => {
    const withWorker = shellManifest({ base: "/", fileNames: FILES, html: HTML });
    const without = shellManifest({
      base: "/",
      fileNames: FILES.filter((file) => file !== "sw.js"),
      html: HTML,
    });
    expect(withWorker).toEqual(without);
  });

  it("refuses a build whose HTML loads no entry script", () => {
    expect(() =>
      shellManifest({ base: "/", fileNames: FILES, html: "<html><head></head></html>" }),
    ).toThrow(/loads no entry script/);
  });

  it("prefixes paths with the app's mount path", () => {
    const manifest = shellManifest({ base: "/app/", fileNames: FILES, html: HTML });
    expect(manifest.base).toBe("/app/");
    expect(manifest.assets[0]).toBe("/app/assets/fonts/inter.woff2");
  });
});
