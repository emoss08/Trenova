import { describe, expect, it } from "vitest";
import { dataEntryControlSchema } from "@/types/data-entry-control";
import { dataEntryControlTabs } from "../data-entry-control-tabs";

const RECORD_KEYS = new Set([
  "id",
  "version",
  "createdAt",
  "updatedAt",
  "organizationId",
  "businessUnitId",
]);

describe("data entry control tabs", () => {
  it("puts every setting the form edits on exactly one tab", () => {
    const placed = dataEntryControlTabs().flatMap((tab) => tab.fields);
    const editable = Object.keys(dataEntryControlSchema.shape).filter(
      (key) => !RECORD_KEYS.has(key),
    );

    expect([...placed].sort()).toEqual([...editable].sort());
    expect(new Set(placed).size).toBe(placed.length);
  });

  it("has one tab, case formatting, for every organization", () => {
    expect(dataEntryControlTabs().map((tab) => tab.value)).toEqual(["formatting"]);
  });
});
