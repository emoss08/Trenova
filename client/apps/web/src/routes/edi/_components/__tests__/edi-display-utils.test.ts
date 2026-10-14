import type { EDIMappingResolution } from "@trenova/shared/types/edi";
import { describe, expect, it } from "vitest";
import { findMapping } from "../edi-display-utils";

const row = (overrides: Partial<EDIMappingResolution>): EDIMappingResolution => ({
  entityType: "Location",
  sourceId: "ACME-DAL",
  resolved: true,
  ...overrides,
});

describe("findMapping", () => {
  it("finds the mapping for a stop's location code", () => {
    const rows = [row({ sourceId: "KR-HOU" }), row({ sourceId: "ACME-DAL" })];
    expect(findMapping(rows, "Location", "ACME-DAL")).toBe(rows[1]);
  });

  it("finds nothing for a stop the partner sent without a location code", () => {
    const rows = [row({ sourceId: "" })];
    expect(findMapping(rows, "Location", null)).toBeUndefined();
    expect(findMapping(rows, "Location", undefined)).toBeUndefined();
  });
});
