import assert from "node:assert/strict";
import { mkdir, mkdtemp, rm, writeFile } from "node:fs/promises";
import { tmpdir } from "node:os";
import { join, resolve } from "node:path";
import { after, before, describe, it } from "node:test";
import { fileURLToPath } from "node:url";
import { findFragments } from "./fragments.mjs";

const repoRoot = resolve(fileURLToPath(import.meta.url), "../../..");
let root;

async function scan(source) {
  const dir = join(root, "client", "apps", "web", "src", String(Math.random()).slice(2));
  await mkdir(dir, { recursive: true });
  await writeFile(join(dir, "page.tsx"), source);
  return findFragments(root, [join("client", "apps", "web", "src", dir.split("/").pop())]);
}

before(async () => {
  root = await mkdtemp(join(tmpdir(), "fragments-"));
});

after(async () => {
  await rm(root, { recursive: true, force: true });
});

describe("findFragments", () => {
  it("reports a sentence split by markup", async () => {
    const found = await scan(`const A = () => <p>{t("You've used")} <b>{pct}</b> {t("of this month's allowance")}</p>;`);
    assert.deepEqual(found.map((f) => f.kind), ["split-sentence"]);
  });

  it("accepts the same sentence written whole", async () => {
    const found = await scan(
      `const A = () => <p>{rt("You've used <b>{0}</b> of this month's allowance", { b: (c) => <b>{c}</b> }, pct)}</p>;`,
    );
    assert.deepEqual(found, []);
  });

  it("does not mistake two sentences around a block for one", async () => {
    const found = await scan(`const A = () => <div>{t("First.")}<div /> {t("Second.")}</div>;`);
    assert.deepEqual(found, []);
  });

  it("reports English handed to a message by a literal, a ternary or a helper", async () => {
    const found = await scan(`
      t("{0} {1}", n, n === 1 ? "stop" : "stops");
      t("Pickup {0}", when || "unscheduled");
      t("Send {0}", pluralize("notice", n));
      t("Sequence {0}{1}", s, path ? \` / repeats \${path}\` : "");
      rt("Open <b>{0}</b>", tags, "the record");
    `);
    assert.deepEqual(
      found.map((f) => f.detail),
      ["stop", "unscheduled", "pluralize(…)", "` / repeats `", "the record"],
    );
  });

  it("leaves variables, numbers, wire values and tokens alone", async () => {
    const found = await scan(`
      t("Open {0}", label);
      t("{0, plural, one {# stop} other {# stops}}", n);
      t("Code {0}", "SHIPMENT_ID");
      t("Replace {0}", "{loginSlug}");
    `);
    assert.deepEqual(found, []);
  });

  it("finds none in the app", async () => {
    const found = await findFragments(repoRoot);
    assert.deepEqual(found.map((f) => `${f.file}:${f.line} ${f.detail}`), []);
  });
});
