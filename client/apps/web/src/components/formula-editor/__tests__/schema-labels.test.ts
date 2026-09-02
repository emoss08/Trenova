import { describe, expect, it } from "vitest";
import {
  categoryLabel,
  CATEGORY_LABELS,
  FUNCTION_CATEGORY_LABELS,
  sampleInputKind,
} from "../schema-labels";

describe("sampleInputKind", () => {
  it("maps server and fallback type spellings to the same control", () => {
    expect(sampleInputKind("number")).toBe("number");
    expect(sampleInputKind("Number")).toBe("number");
    expect(sampleInputKind("integer")).toBe("number");
    expect(sampleInputKind("Boolean")).toBe("boolean");
    expect(sampleInputKind("string")).toBe("text");
  });

  it("prefers an enum select when values are known", () => {
    expect(sampleInputKind("string", ["New", "InTransit"])).toBe("enum");
    expect(sampleInputKind("string", [])).toBe("text");
  });
});

describe("categoryLabel", () => {
  it("falls back to the raw category or Other", () => {
    expect(categoryLabel("computed", CATEGORY_LABELS)).toBe("Computed Rollups");
    expect(categoryLabel("weird", CATEGORY_LABELS)).toBe("weird");
    expect(categoryLabel("", CATEGORY_LABELS)).toBe("Other");
  });
});

describe("FUNCTION_CATEGORY_LABELS", () => {
  it("names the string category for the reference pane", () => {
    expect(FUNCTION_CATEGORY_LABELS.string).toBe("Text");
  });
});

describe("CATEGORY_LABELS for typed relations", () => {
  it("names the service and shipment type groups", () => {
    expect(CATEGORY_LABELS.serviceType).toBe("Service Type");
    expect(CATEGORY_LABELS.shipmentType).toBe("Shipment Type");
  });
});

describe("CATEGORY_LABELS for provided context", () => {
  it("names the market-data group fed by integrations", () => {
    expect(CATEGORY_LABELS.context).toBe("Market Data");
  });
});
