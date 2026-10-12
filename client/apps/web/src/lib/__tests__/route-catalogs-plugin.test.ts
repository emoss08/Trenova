import { describe, expect, it } from "vitest";
import {
  AFTER_CATALOGS,
  dynamicImports,
  REQUIRE_CATALOG,
  routeCatalogFor,
  routeCatalogs,
} from "../../../vite/route-catalogs";

const SRC = "/repo/client/apps/web/src";
const ROUTES = `${SRC}/routes`;
const RUNTIME = '"@trenova/shared/i18n/runtime"';

type TransformResult = { code: string; map: { mappings: string } } | null;

// What Vite's resolver would answer: the app's aliases and relative paths land in src, a
// package lands in node_modules, an unknown specifier resolves to nothing.
function resolve(specifier: string, importer: string) {
  if (specifier.startsWith("@/"))
    return { id: `${SRC}/${specifier.slice(2)}.tsx`, external: false };
  if (specifier.startsWith(".")) {
    const dir = importer.slice(0, importer.lastIndexOf("/"));
    const parts = `${dir}/${specifier}`.split("/");
    const out: string[] = [];
    for (const part of parts) {
      if (part === "..") out.pop();
      else if (part !== ".") out.push(part);
    }
    return { id: `${out.join("/")}.tsx`, external: false };
  }
  if (specifier === "missing") return null;
  if (specifier === "cdn-lib") return { id: specifier, external: true };
  return { id: `/repo/node_modules/${specifier}/index.js`, external: false };
}

const plugin = routeCatalogs({ srcDir: SRC });

async function transform(code: string, id: string): Promise<TransformResult> {
  const hook = plugin.transform as unknown as (
    this: { resolve: typeof resolve },
    code: string,
    id: string,
  ) => Promise<TransformResult>;
  return hook.call({ resolve }, code, id);
}

describe("routeCatalogFor", () => {
  it("names the folder's bundle for a module anywhere inside it", () => {
    expect(routeCatalogFor(`${ROUTES}/shipment/page.tsx`, ROUTES)).toBe("routes/shipment");
    expect(routeCatalogFor(`${ROUTES}/shipment/_components/form/panel.tsx`, ROUTES)).toBe(
      "routes/shipment",
    );
    expect(routeCatalogFor(`${ROUTES}/admin/schema.ts`, ROUTES)).toBe("routes/admin");
  });

  it("leaves files directly in the routes folder to the shell's bundle", () => {
    expect(routeCatalogFor(`${ROUTES}/app-layout.tsx`, ROUTES)).toBeNull();
  });

  it("ignores code outside the routes folder, including a look-alike sibling", () => {
    expect(routeCatalogFor(`${SRC}/components/nav.tsx`, ROUTES)).toBeNull();
    expect(routeCatalogFor(`${SRC}/routes-old/x/page.tsx`, ROUTES)).toBeNull();
    expect(
      routeCatalogFor("/repo/client/packages/shared/src/routes/x/page.tsx", ROUTES),
    ).toBeNull();
  });

  it("ignores tests, stories and non-code assets", () => {
    expect(routeCatalogFor(`${ROUTES}/shipment/page.test.tsx`, ROUTES)).toBeNull();
    expect(routeCatalogFor(`${ROUTES}/shipment/__tests__/helpers.ts`, ROUTES)).toBeNull();
    expect(routeCatalogFor(`${ROUTES}/shipment/page.stories.tsx`, ROUTES)).toBeNull();
    expect(routeCatalogFor(`${ROUTES}/shipment/styles.css`, ROUTES)).toBeNull();
  });

  it("ignores another module kind of the same file", () => {
    expect(routeCatalogFor(`${ROUTES}/shipment/page.tsx?raw`, ROUTES)).toBeNull();
    expect(routeCatalogFor(`\0virtual:${ROUTES}/shipment/page.tsx`, ROUTES)).toBeNull();
  });

  it("reads Windows paths the same way", () => {
    expect(
      routeCatalogFor(
        "C:\\repo\\client\\apps\\web\\src\\routes\\shipment\\page.tsx",
        "C:\\repo\\client\\apps\\web\\src\\routes\\",
      ),
    ).toBe("routes/shipment");
  });
});

describe("dynamicImports", () => {
  it("finds lazy imports in TSX, however they are written", () => {
    const code = [
      'const A = lazy(() => import("./a"));',
      "const B = lazy(",
      "  () =>",
      '    import("@/routes/shipment/_components/comments").then((m) => ({ default: m.Comments })),',
      ");",
      "export const C = () => <Suspense><A /></Suspense>;",
    ].join("\n");

    expect(dynamicImports(code, "x.tsx").map((i) => i.specifier)).toEqual([
      "./a",
      "@/routes/shipment/_components/comments",
    ]);
  });

  it("does not mistake a type for an import", () => {
    const code = 'type M = typeof import("./a");\nconst s = "import(\\"./b\\")";';
    expect(dynamicImports(code, "x.ts")).toEqual([]);
  });

  it("leaves a computed specifier alone", () => {
    expect(dynamicImports("const m = import(`./locale/${name}`);", "x.ts")).toEqual([]);
  });

  it("points past the call even after non-ASCII text", () => {
    const code = 'const s = "运单 🚚"; const A = lazy(() => import("./a"));';
    const [found] = dynamicImports(code, "x.tsx");
    expect(code.slice(0, found.end).endsWith('import("./a")')).toBe(true);
  });
});

describe("routeCatalogs plugin", () => {
  it("makes a route module require its folder's bundle, moving no line", async () => {
    const code = 'import { useT } from "@trenova/shared/i18n/use-t";\nexport const a = 1;';
    const result = await transform(code, `${ROUTES}/shipment/page.tsx`);

    expect(result?.code.startsWith(code)).toBe(true);
    expect(result?.code.slice(code.length)).toBe(
      `\nimport { requireCatalog as ${REQUIRE_CATALOG} } from ${RUNTIME};\n` +
        `${REQUIRE_CATALOG}("routes/shipment");\n`,
    );
  });

  it("holds a lazy component borrowed from another route folder until its strings land", async () => {
    const code = 'const Comments = lazy(() => import("@/routes/shipment/_components/comments"));\n';
    const result = await transform(code, `${ROUTES}/billing-queue/_components/detail.tsx`);

    expect(result?.code).toContain(
      `import("@/routes/shipment/_components/comments").then(${AFTER_CATALOGS})`,
    );
    expect(result?.code).toContain(
      `import { requireCatalog as ${REQUIRE_CATALOG}, afterCatalogs as ${AFTER_CATALOGS} } from ${RUNTIME};`,
    );
  });

  it("holds every lazy import of app code, which may reach a folder indirectly", async () => {
    const code = [
      'const Palette = lazy(() => import("./command-palette"));',
      'const Preview = lazy(() => import("../shipment/shipment-preview").then((m) => ({ default: m.P })));',
    ].join("\n");
    const result = await transform(code, `${SRC}/components/command-palette/mount.tsx`);

    expect(result?.code).toContain(`import("./command-palette").then(${AFTER_CATALOGS})`);
    expect(result?.code).toContain(
      `import("../shipment/shipment-preview").then(${AFTER_CATALOGS}).then((m) =>`,
    );
    expect(result?.code).not.toContain(REQUIRE_CATALOG);
  });

  it("leaves imports of packages, externals and unresolvable specifiers alone", async () => {
    const code = [
      'const pdf = import("pdfjs-dist");',
      'const lib = import("cdn-lib");',
      'const gone = import("missing");',
    ].join("\n");

    expect(await transform(code, `${SRC}/components/viewer.tsx`)).toBeNull();
  });

  it("keeps the source map pointing at the original lines", async () => {
    const code = 'const A = lazy(() => import("./a"));\nexport const b = 2;\n';
    const result = await transform(code, `${SRC}/components/x.tsx`);

    const lines = result?.map.mappings.split(";") ?? [];
    expect(lines.length).toBeGreaterThanOrEqual(2);
    expect(lines[0]).not.toBe("");
    expect(lines[1]).not.toBe("");
  });

  it("leaves code outside the app, tests and assets untouched", async () => {
    const lazyCode = 'const A = lazy(() => import("./a"));';
    expect(await transform(lazyCode, "/repo/client/packages/shared/src/x.tsx")).toBeNull();
    expect(await transform(lazyCode, `${ROUTES}/shipment/page.test.tsx`)).toBeNull();
    expect(await transform(lazyCode, `${ROUTES}/shipment/page.tsx?raw`)).toBeNull();
    expect(await transform("export {}", `${SRC}/main.tsx`)).toBeNull();
  });

  it("runs before the TypeScript transform so it sees source paths", () => {
    expect(plugin.enforce).toBe("pre");
  });
});
