import { mkdtemp, mkdir, rm, writeFile } from "node:fs/promises";
import { tmpdir } from "node:os";
import { join } from "node:path";
import { gzipSync } from "node:zlib";
import { afterEach, describe, expect, it } from "vitest";
import { measure, overBudget } from "../../../scripts/bundle-budget.ts";
import { startupFiles } from "../startup-files";

const HTML = `<!doctype html>
<html>
  <head>
    <script type="module" crossorigin src="/assets/js/main.abc.js"></script>
    <link rel="modulepreload" crossorigin href="/assets/js/react-vendor.def.js">
    <link rel="modulepreload" crossorigin href="/assets/js/initial.ghi.js">
    <link rel="modulepreload" crossorigin href="/assets/js/initial.ghi.js">
    <link rel="stylesheet" crossorigin href="/assets/initial-styles.jkl.css">
    <link rel="icon" href="/logo.ico">
    <script src="https://maps.googleapis.com/maps/api/js"></script>
    <link rel="modulepreload" href="https://cdn.example.com/lib.js">
  </head>
</html>`;

describe("startupFiles", () => {
  it("names the entry script and lists every preloaded module once and the linked stylesheets", () => {
    expect(startupFiles(HTML)).toEqual({
      entry: "/assets/js/main.abc.js",
      scripts: [
        "/assets/js/main.abc.js",
        "/assets/js/react-vendor.def.js",
        "/assets/js/initial.ghi.js",
      ],
      styles: ["/assets/initial-styles.jkl.css"],
    });
  });

  it("leaves out what the build did not emit: other origins, icons, classic scripts", () => {
    const { scripts, styles } = startupFiles(HTML);
    expect([...scripts, ...styles].some((url) => url.includes("example.com"))).toBe(false);
    expect([...scripts, ...styles].some((url) => url.includes("googleapis"))).toBe(false);
    expect([...scripts, ...styles]).not.toContain("/logo.ico");
  });
});

describe("measure", () => {
  let dist: string | null = null;

  afterEach(async () => {
    if (dist) await rm(dist, { recursive: true, force: true });
    dist = null;
  });

  it("totals the raw and gzipped size of each startup file the build wrote", async () => {
    dist = await mkdtemp(join(tmpdir(), "bundle-budget-"));
    await mkdir(join(dist, "assets/js"), { recursive: true });
    const files: Record<string, string> = {
      "assets/js/main.abc.js": "import './react-vendor.def.js';".repeat(40),
      "assets/js/react-vendor.def.js": "export const react = 1;".repeat(200),
      "assets/js/initial.ghi.js": "export const app = 2;".repeat(100),
      "assets/initial-styles.jkl.css": ".a{color:red}".repeat(50),
    };
    await writeFile(join(dist, "index.html"), HTML);
    for (const [file, source] of Object.entries(files)) await writeFile(join(dist, file), source);

    const measured = await measure(dist);
    const gzip = (file: string) => gzipSync(files[file], { level: 6 }).length;

    expect(measured.scripts.files).toBe(3);
    expect(measured.scripts.bytes).toBe(
      files["assets/js/main.abc.js"].length +
        files["assets/js/react-vendor.def.js"].length +
        files["assets/js/initial.ghi.js"].length,
    );
    expect(measured.scripts.gzip).toBe(
      gzip("assets/js/main.abc.js") +
        gzip("assets/js/react-vendor.def.js") +
        gzip("assets/js/initial.ghi.js"),
    );
    expect(measured.styles).toMatchObject({
      files: 1,
      bytes: files["assets/initial-styles.jkl.css"].length,
      gzip: gzip("assets/initial-styles.jkl.css"),
    });
  });
});

describe("overBudget", () => {
  const budget = {
    scripts: { files: 3, gzipBytes: 102_400 },
    styles: { files: 1, gzipBytes: 51_200 },
  };
  const within = {
    scripts: { files: 3, bytes: 400_000, gzip: 102_400, largest: [] },
    styles: { files: 1, bytes: 200_000, gzip: 51_200, largest: [] },
  };

  it("passes a build exactly at its budget", () => {
    expect(overBudget(within, budget)).toEqual([]);
  });

  it("names every figure that is over, by kind", () => {
    const over = {
      scripts: { ...within.scripts, files: 4, gzip: 104_448 },
      styles: { ...within.styles, files: 2 },
    };
    expect(overBudget(over, budget)).toEqual([
      "scripts: 102.0 kB gzip is over the 100.0 kB budget",
      "scripts: 4 files is over the budget of 3",
      "styles: 2 files is over the budget of 1",
    ]);
  });
});
