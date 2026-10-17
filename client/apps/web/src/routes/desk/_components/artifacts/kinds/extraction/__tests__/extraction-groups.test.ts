import type { TranslateFn } from "@trenova/shared/i18n/use-t";
import { describe, expect, it } from "vitest";
import { checkCount, checkNames, checkQuestion } from "../extraction-asks";
import {
  fieldState,
  groupExtraction,
  reviewTally,
  rowsInOrder,
  type FieldRow,
} from "../extraction-groups";
import { extractionFrom } from "../extraction-payload";
import { currentPayload, legacyPayload } from "./extraction-fixtures";

const t = ((source: string, ...args: unknown[]) =>
  source.replace(/\{(\d+)\}/g, (_, index: string) => String(args[Number(index)]))) as TranslateFn;

function labels(rows: readonly FieldRow[]) {
  return rows.map((row) => row.label);
}

describe("groupExtraction", () => {
  const groups = groupExtraction(extractionFrom(currentPayload()));
  const group = (id: string) => groups.find((candidate) => candidate.id === id);

  it("orders the groups the way a dispatcher reads a load", () => {
    expect(groups.map((candidate) => candidate.id)).toEqual([
      "parties",
      "stops",
      "references",
      "freight",
      "charges",
      "other",
    ]);
  });

  it("lists who before what, in the catalog's order, labelled in sentence case", () => {
    expect(labels(group("parties")?.rows ?? [])).toEqual(["Shipper", "Bill to"]);
    expect(labels(group("references")?.rows ?? [])).toEqual(["Load #", "Pro #"]);
    expect(labels(group("freight")?.rows ?? [])).toEqual(["Commodity", "Pieces"]);
    expect(labels(group("charges")?.rows ?? [])).toEqual(["Rate"]);
  });

  it("puts each stop in sequence with its parts, and a window read apart joins its stop", () => {
    const stops = group("stops")?.stops ?? [];

    expect(stops.map((stop) => [stop.sequence, stop.role, stop.doubtful])).toEqual([
      [1, "pickup", false],
      [2, "delivery", true],
      [3, "delivery", true],
    ]);
    expect(stops[0].rows.map((row) => row.key)).toEqual(["stop1.name", "stop1.window"]);
    expect(stops[1].rows.map((row) => row.key)).toEqual(["stop2.name", "deliveryWindow"]);
  });

  it("leaves out a settled window that only repeats its stop's own", () => {
    const keys = rowsInOrder(groups).map((row) => row.key);

    expect(keys).not.toContain("pickupWindow");
    expect(keys).toContain("deliveryWindow");
  });

  it("stands a missing field where it would have been read, and an unknown one under Other", () => {
    const references = group("references")?.rows ?? [];
    const other = group("other")?.rows ?? [];

    expect(references[1]).toMatchObject({
      kind: "missing",
      key: "missing:proNumber",
      state: "missing",
    });
    expect(other.map((row) => [row.kind, row.label])).toEqual([
      ["read", "Custom Flag"],
      ["missing", "Temperature"],
    ]);
  });

  it("finds an older card's missing field by its label", () => {
    const references = groupExtraction(extractionFrom(legacyPayload())).find(
      (candidate) => candidate.id === "references",
    );

    expect(references?.rows.map((row) => row.key)).toEqual(["loadNumber", "missing:proNumber"]);
  });

  it("walks the rows stops first within their group, in the order they are drawn", () => {
    expect(rowsInOrder(groups).map((row) => row.key)).toEqual([
      "shipper",
      "billTo",
      "stop1.name",
      "stop1.window",
      "stop2.name",
      "deliveryWindow",
      "stop3.name",
      "loadNumber",
      "missing:proNumber",
      "commodity",
      "missing:pieceCount",
      "rate",
      "customFlag",
      "missing:temperature",
    ]);
  });
});

describe("fieldState", () => {
  it("settles a sure reading, shows a fair one's figure, and doubts one that needs a look", () => {
    const fields = new Map(
      extractionFrom(currentPayload()).fields.map((field) => [field.key, field]),
    );

    expect(fieldState(fields.get("loadNumber")!)).toBe("settled");
    expect(fieldState(fields.get("rate")!)).toBe("fair");
    expect(fieldState(fields.get("billTo")!)).toBe("doubtful");
    expect(fieldState(fields.get("deliveryWindow")!)).toBe("doubtful");
  });
});

describe("reviewTally", () => {
  const groups = groupExtraction(extractionFrom(currentPayload()));
  const tally = reviewTally(groups);

  it("counts a stop under review once, the missing fields, and what was read", () => {
    expect(
      tally.check.map((item) =>
        item.kind === "stop" ? `stop ${item.stop.sequence}` : item.row.key,
      ),
    ).toEqual(["billTo", "stop 2", "stop 3", "commodity"]);
    expect(tally.missing.map((row) => row.key)).toEqual([
      "missing:proNumber",
      "missing:pieceCount",
      "missing:temperature",
    ]);
    expect(tally.read).toBe(7);
  });

  it("names every field Check asks about, with why, and those to look for", () => {
    const names = checkNames(tally, t);

    expect(checkCount(tally)).toBe(7);
    expect(names.shown.check).toEqual([
      "Bill to",
      "Delivery · stop 2",
      "Delivery · stop 3",
      "Commodity",
    ]);
    expect(names.check[0]).toBe("Bill to (Marked for review · 71%)");
    expect(names.check[1]).toBe("Delivery · stop 2: Delivery window (Two readings disagree)");
    expect(names.missing).toEqual(["Pro #", "Pieces", "Temperature"]);
    expect(
      checkQuestion({ fileName: "Northline_RateCon.pdf", documentId: "doc_1" }, names, t),
    ).toContain("On Northline_RateCon.pdf (document doc_1), check these fields");
  });
});
