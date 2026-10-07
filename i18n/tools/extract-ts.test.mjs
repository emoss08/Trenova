import assert from "node:assert/strict";
import { mkdir, mkdtemp, rm, writeFile } from "node:fs/promises";
import { tmpdir } from "node:os";
import { join } from "node:path";
import { after, before, describe, it } from "node:test";
import { extractTypeScript } from "./extract-ts.mjs";

let root;

before(async () => {
  root = await mkdtemp(join(tmpdir(), "extract-ts-"));
  const dir = join(root, "client", "apps", "web", "src", "routes", "desk");
  await mkdir(dir, { recursive: true });
  await writeFile(
    join(dir, "usage.tsx"),
    `export function Usage({ pct }) {
  const t = useT();
  const rt = useRichT();
  return (
    <p title={t("Usage")}>
      {rt("You've used <b>{0}</b> of this month's AI allowance", { b: (c) => <b>{c}</b> }, pct)}
    </p>
  );
}
`,
  );
  await writeFile(
    join(dir, "labels.ts"),
    `export const DELIVERY_LABELS = defineLabels({
  Online: "Online",
  OnTheJob: "On the job",
  Units: "minutes",
} as const satisfies Record<string, string>);

export const WIRE_VALUES = { Online: "ONLINE_DELIVERY", OnTheJob: "on_the_job" };
`,
  );
});

after(async () => {
  await rm(root, { recursive: true, force: true });
});

describe("extractTypeScript", () => {
  it("records a rich-text sentence whole, tags and all", async () => {
    const { entries } = await extractTypeScript(root, ["client/apps/web/src"]);
    const messages = entries.map((e) => e.message);

    assert.ok(messages.includes("You've used <b>{0}</b> of this month's AI allowance"));
    assert.ok(messages.includes("Usage"));
    const rich = entries.find((e) => e.message.startsWith("You've used"));
    assert.equal(rich.kind, "t-call");
    assert.equal(rich.area, "routes/desk");
  });

  it("records every caption a label map declares, a lone lowercase word included", async () => {
    const { entries } = await extractTypeScript(root, ["client/apps/web/src"]);
    const labels = entries.filter((e) => e.kind === "label-map").map((e) => e.message);

    assert.deepEqual(labels.sort(), ["On the job", "Online", "minutes"]);
    assert.ok(!entries.some((e) => e.message === "ONLINE_DELIVERY" || e.message === "on_the_job"));
  });
});
