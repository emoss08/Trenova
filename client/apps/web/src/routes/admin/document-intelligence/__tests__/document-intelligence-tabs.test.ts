import { describe, expect, it } from "vitest";
import { captureSettingFields, documentControlSchema } from "@/types/document-control";
import { documentIntelligenceTabs } from "../document-intelligence-tabs";

const RECORD_KEYS = new Set([
  "id",
  "version",
  "createdAt",
  "updatedAt",
  "organizationId",
  "businessUnitId",
]);

describe("document extraction tabs", () => {
  it("puts every setting the page edits on exactly one tab, leaving capture to its own page", () => {
    const placed = documentIntelligenceTabs().flatMap((tab) => tab.fields);
    const editable = Object.keys(documentControlSchema.shape).filter(
      (key) => !RECORD_KEYS.has(key) && !(captureSettingFields as readonly string[]).includes(key),
    );

    expect([...placed].sort()).toEqual([...editable].sort());
    expect(new Set(placed).size).toBe(placed.length);
  });

  it("names exactly the capture settings the record carries", () => {
    const capture = Object.keys(documentControlSchema.shape).filter(
      (key) => key.startsWith("capture") || key === "enableCapture",
    );

    expect([...captureSettingFields].sort()).toEqual(capture.sort());
  });

  it("offers reading, filing and shipment drafts, in that order", () => {
    expect(documentIntelligenceTabs().map((tab) => [tab.value, tab.label])).toEqual([
      ["reading", "Reading"],
      ["filing", "Filing"],
      ["drafts", "Shipment drafts"],
    ]);
  });
});
