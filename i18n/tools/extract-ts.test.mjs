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
});
