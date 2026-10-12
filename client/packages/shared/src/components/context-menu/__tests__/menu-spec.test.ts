import { describe, expect, it } from "vitest";
import { normalizeEntries, type ContextMenuEntry } from "../menu-spec";

function labels(entries: ContextMenuEntry[]): string[] {
  return entries.map((entry) =>
    entry.type === "separator" ? "—" : entry.type === "header" ? `# ${entry.label}` : entry.label,
  );
}

describe("normalizeEntries", () => {
  it("drops the rows a condition turned off", () => {
    const busy = false;
    expect(labels(normalizeEntries([{ label: "Open" }, busy && { label: "Wait" }, null, undefined]))).toEqual([
      "Open",
    ]);
  });

  it("moves destructive rows to the foot, after a separator, keeping their order", () => {
    const entries = normalizeEntries([
      { label: "Delete view", destructive: true },
      { label: "Open" },
      { type: "separator" },
      { label: "Unlink", destructive: true },
      { label: "Rename" },
    ]);

    expect(labels(entries)).toEqual(["Open", "—", "Rename", "—", "Delete view", "Unlink"]);
  });

  it("never draws a separator at either end or two in a row", () => {
    const entries = normalizeEntries([
      { type: "separator" },
      { label: "Open" },
      { type: "separator" },
      { type: "separator" },
      { label: "Copy link" },
      { type: "separator" },
    ]);

    expect(labels(entries)).toEqual(["Open", "—", "Copy link"]);
  });

  it("drops a heading with nothing under it", () => {
    const entries = normalizeEntries([
      { label: "Later today" },
      { type: "separator" },
      { type: "header", label: "Until something happens" },
      { type: "separator" },
      { label: "Copy link" },
      { type: "header", label: "Nova should…" },
    ]);

    expect(labels(entries)).toEqual(["Later today", "—", "Copy link"]);
  });
});
