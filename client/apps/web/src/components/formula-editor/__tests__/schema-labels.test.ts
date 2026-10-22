import { describe, expect, it } from "vitest";
import { categoryLabel, CATEGORY_LABELS, FUNCTION_CATEGORY_LABELS } from "../schema-labels";

describe("categoryLabel", () => {
  it("falls back to the raw category or Other", () => {
    expect(categoryLabel("computed", CATEGORY_LABELS)).toBe("Computed rollups");
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
    expect(CATEGORY_LABELS.serviceType).toBe("Service type");
    expect(CATEGORY_LABELS.shipmentType).toBe("Shipment type");
  });
});

describe("CATEGORY_LABELS for provided context", () => {
  it("names the market-data group fed by integrations", () => {
    expect(CATEGORY_LABELS.context).toBe("Market data");
  });
});
