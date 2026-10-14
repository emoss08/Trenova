import assert from "node:assert/strict";
import { mkdirSync, mkdtempSync, rmSync, writeFileSync } from "node:fs";
import { tmpdir } from "node:os";
import { dirname, join } from "node:path";
import { after, describe, it } from "node:test";

import { auditLoadingSkeletons } from "./loading-skeletons.mjs";

const roots = [];

after(() => {
  for (const root of roots) {
    rmSync(root, { recursive: true, force: true });
  }
});

const DATA_TABLE = `export function DataTable() { return null; }\n`;

const TABLE = `import { DataTable } from "@/components/data-table/data-table";
export default function Table() {
  return <DataTable<Row> resource="x" />;
}
`;

const DASHBOARD = `export function Dashboard() {
  return <section>figures</section>;
}
`;

/** A client root holding these files, each path relative to the root. */
function client(files) {
  const root = mkdtempSync(join(tmpdir(), "loading-skeletons-"));
  roots.push(root);
  const all = {
    "apps/web/src/components/data-table/data-table.tsx": DATA_TABLE,
    "apps/web/src/routes/x/_components/table.tsx": TABLE,
    "apps/web/src/routes/x/_components/dashboard.tsx": DASHBOARD,
    ...files,
  };
  for (const [path, text] of Object.entries(all)) {
    mkdirSync(dirname(join(root, path)), { recursive: true });
    writeFileSync(join(root, path), text);
  }
  return root;
}

function page(body, imports = "") {
  return `import { DataTableLazyComponent, LazyComponent } from "@trenova/shared/components/error-boundary";
import { lazy, Suspense } from "react";
${imports}
export function Page() {
  return (
    <>
${body}
    </>
  );
}
`;
}

async function rules(files) {
  const { violations } = await auditLoadingSkeletons(client(files));
  return violations.map(({ file, line, rule }) => `${file}:${line} ${rule}`);
}

describe("auditLoadingSkeletons", () => {
  it("passes a table lazy-loaded into the table boundary", async () => {
    const found = await rules({
      "apps/web/src/routes/x/page.tsx": page(
        `      <DataTableLazyComponent>
        <Table />
      </DataTableLazyComponent>`,
        `const Table = lazy(() => import("./_components/table"));`,
      ),
    });

    assert.deepEqual(found, []);
  });

  it("flags a dashboard in the table boundary, through .then, lazyComponent and conditionals", async () => {
    const found = await rules({
      "apps/web/src/routes/x/page.tsx": page(
        `      <DataTableLazyComponent>
        {open && <Board />}
        {open ? <Panel.Component /> : null}
        <Table />
      </DataTableLazyComponent>`,
        `import { lazyComponent } from "@/lib/lazy-component";
const Table = lazy(() => import("./_components/table"));
const Board = lazy(() =>
  import("./_components/dashboard").then((module) => ({ default: module.Dashboard })),
);
const Panel = lazyComponent(
  () => import("./_components/dashboard"),
  (module) => module.Dashboard,
);`,
      ),
    });

    assert.deepEqual(found, [
      "apps/web/src/routes/x/page.tsx:16 table-skeleton-on-non-table",
      "apps/web/src/routes/x/page.tsx:17 table-skeleton-on-non-table",
    ]);
  });

  it("follows a table re-exported through an index module", async () => {
    const found = await rules({
      "apps/web/src/routes/x/_components/index.ts": `export { default } from "./table";\n`,
      "apps/web/src/routes/x/page.tsx": page(
        `      <DataTableLazyComponent>
        <Table />
      </DataTableLazyComponent>`,
        `import Table from "./_components";`,
      ),
    });

    assert.deepEqual(found, []);
  });

  it("flags a child it cannot follow to a module", async () => {
    const found = await rules({
      "apps/web/src/routes/x/page.tsx": page(
        `      <DataTableLazyComponent>
        <Suspense fallback={null}>
          <Missing />
        </Suspense>
      </DataTableLazyComponent>`,
      ),
    });

    assert.deepEqual(found, ["apps/web/src/routes/x/page.tsx:8 unresolved-lazy-child"]);
  });

  it("flags a Suspense without a fallback but accepts an explicit null", async () => {
    const found = await rules({
      "apps/web/src/routes/x/page.tsx": page(
        `      <Suspense>
        <Dashboard />
      </Suspense>
      <Suspense fallback={null}>
        <Dashboard />
      </Suspense>`,
        `import { Dashboard } from "./_components/dashboard";`,
      ),
    });

    assert.deepEqual(found, ["apps/web/src/routes/x/page.tsx:7 suspense-without-fallback"]);
  });

  it("flags a spinner as a fallback, in either branch of a conditional", async () => {
    const found = await rules({
      "apps/web/src/routes/x/page.tsx": page(
        `      <LazyComponent fallback={designer ? <ComponentLoader /> : <DashboardSkeleton />}>
        <Dashboard />
      </LazyComponent>
      <Suspense fallback={<Spinner />}>
        <Dashboard />
      </Suspense>`,
        `import { Dashboard } from "./_components/dashboard";`,
      ),
    });

    assert.deepEqual(found, [
      "apps/web/src/routes/x/page.tsx:7 spinner-fallback",
      "apps/web/src/routes/x/page.tsx:10 spinner-fallback",
    ]);
  });

  it("honours an ignore comment on or above the line", async () => {
    const found = await rules({
      "apps/web/src/routes/x/page.tsx": page(
        `      {/* loading-skeleton-ignore: the overlay draws its own placeholder */}
      <Suspense>
        <Dashboard />
      </Suspense>`,
        `import { Dashboard } from "./_components/dashboard";`,
      ),
    });

    assert.deepEqual(found, []);
  });

  it("skips tests and stories", async () => {
    const found = await rules({
      "apps/web/src/routes/x/__tests__/page.test.tsx": page(
        `      <Suspense>
        <p />
      </Suspense>`,
      ),
      "apps/web/src/routes/x/page.stories.tsx": page(
        `      <Suspense>
        <p />
      </Suspense>`,
      ),
    });

    assert.deepEqual(found, []);
  });
});
