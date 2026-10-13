import { createHash } from "node:crypto";
import { build, type Plugin } from "vite";
import type { ShellManifest } from "../src/service-worker/manifest.ts";
import { startupFiles } from "../src/lib/startup-files.ts";

// Writes /sw.js (src/service-worker/sw.ts) into the build with the build's own manifest in
// it: which files the HTML shell needs to render, which build files exist, and a version
// that changes with them. A new build therefore ships a different sw.js, which is how the
// browser learns there is one. With VITE_SERVICE_WORKER=off the worker is built with no
// manifest, and in that form it deletes its caches and unregisters itself from every
// browser that installed an earlier one.

/** The manifest for a build, from its output file names and its index.html. */
export function shellManifest({
  base,
  fileNames,
  html,
}: {
  base: string;
  fileNames: string[];
  html: string;
}): ShellManifest {
  const names = fileNames.filter((name) => name !== "sw.js").sort();
  const { entry, scripts, styles } = startupFiles(html);
  if (entry === null) {
    throw new Error("trenova:service-worker: index.html loads no entry script");
  }
  return {
    version: createHash("sha256").update(names.join("\n")).update(html).digest("hex").slice(0, 16),
    base,
    entry,
    startup: [...scripts, ...styles],
    assets: names.filter((name) => name.startsWith("assets/")).map((name) => `${base}${name}`),
  };
}

async function bundleWorker(source: string, manifest: ShellManifest | null): Promise<string> {
  const result = await build({
    configFile: false,
    logLevel: "silent",
    publicDir: false,
    define: { __SHELL__: JSON.stringify(manifest) },
    build: {
      write: false,
      minify: "oxc",
      sourcemap: false,
      reportCompressedSize: false,
      lib: {
        entry: source,
        formats: ["iife"],
        name: "trenovaServiceWorker",
        fileName: () => "sw.js",
      },
    },
  });
  for (const output of Array.isArray(result) ? result : [result]) {
    if (!("output" in output)) continue;
    for (const chunk of output.output) {
      if (chunk.type === "chunk" && chunk.isEntry) return chunk.code;
    }
  }
  throw new Error("trenova:service-worker: bundling the worker produced no script");
}

/** Writes sw.js, bundled from `source` (src/service-worker/sw.ts), into the build. */
export function serviceWorker({ source }: { source: string }): Plugin {
  let base = "/";
  let enabled = true;

  return {
    name: "trenova:service-worker",
    apply: "build",
    configResolved(config) {
      base = config.base;
      enabled = config.env.VITE_SERVICE_WORKER !== "off";
    },
    generateBundle: {
      order: "post",
      async handler(_options, bundle) {
        const html = bundle["index.html"];
        if (html?.type !== "asset") {
          this.error("trenova:service-worker: the build wrote no index.html to read");
        }
        const manifest = enabled
          ? shellManifest({ base, fileNames: Object.keys(bundle), html: String(html.source) })
          : null;
        this.emitFile({
          type: "asset",
          fileName: "sw.js",
          source: await bundleWorker(source, manifest),
        });
      },
    },
  };
}
