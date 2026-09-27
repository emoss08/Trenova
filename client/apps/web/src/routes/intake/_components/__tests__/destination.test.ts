import type { CaptureItem } from "@/lib/graphql/capture";
import { describe, expect, it } from "vitest";
import {
  destinationKey,
  filingEntry,
  initialDestination,
  isFileable,
  withKind,
} from "../destination";

function item(overrides: Partial<CaptureItem>): CaptureItem {
  return {
    id: "citm_1",
    version: 3,
    suggestedType: "",
    suggestedId: null,
    suggestedDocumentTypeId: null,
    ...overrides,
  } as CaptureItem;
}

const noTarget = { targetType: "", targetId: null, documentTypeId: null };

describe("initialDestination", () => {
  it("takes the document's own suggestion first", () => {
    const destination = initialDestination(
      item({ suggestedType: "worker", suggestedId: "wrk_1", suggestedDocumentTypeId: "dt_mvr" }),
      { targetType: "shipment", targetId: "shp_1", documentTypeId: "dt_pod" },
    );
    expect(destination).toEqual({ kind: "worker", recordId: "wrk_1", documentTypeId: "dt_mvr" });
  });

  it("ignores a suggestion onto a kind of record captures cannot be filed onto", () => {
    const destination = initialDestination(
      item({ suggestedType: "invoice", suggestedId: "inv_1" }),
      {
        targetType: "shipment",
        targetId: "shp_1",
        documentTypeId: "dt_pod",
      },
    );
    expect(destination).toEqual({ kind: "shipment", recordId: "shp_1", documentTypeId: "dt_pod" });
  });

  it("keeps a suggested document type when only the stack names a record", () => {
    const destination = initialDestination(item({ suggestedDocumentTypeId: "dt_bol" }), {
      targetType: "tractor",
      targetId: "trac_1",
      documentTypeId: null,
    });
    expect(destination).toEqual({ kind: "tractor", recordId: "trac_1", documentTypeId: "dt_bol" });
  });

  it("starts blank on a shipment when nothing says where it goes", () => {
    expect(initialDestination(undefined, noTarget)).toEqual({
      kind: "shipment",
      recordId: "",
      documentTypeId: "",
    });
  });
});

describe("editing and filing a destination", () => {
  it("clears the record when the kind changes and keeps it when it does not", () => {
    const destination = { kind: "shipment" as const, recordId: "shp_1", documentTypeId: "dt_pod" };
    expect(withKind(destination, "shipment")).toBe(destination);
    expect(withKind(destination, "carrier")).toEqual({
      kind: "carrier",
      recordId: "",
      documentTypeId: "dt_pod",
    });
  });

  it("needs a record to file", () => {
    expect(isFileable({ kind: "shipment", recordId: "", documentTypeId: "dt" })).toBe(false);
    expect(isFileable({ kind: "shipment", recordId: "shp_1", documentTypeId: "" })).toBe(true);
  });

  it("files with the item's version and no blank document type", () => {
    expect(
      filingEntry(item({}), { kind: "customer", recordId: "cus_1", documentTypeId: "" }),
    ).toEqual({
      itemId: "citm_1",
      targetType: "customer",
      targetId: "cus_1",
      documentTypeId: null,
      version: 3,
    });
  });

  it("keys a destination by the document's first page", () => {
    expect(destinationKey(["cpg_2", "cpg_3"])).toBe("cpg_2");
    expect(destinationKey([])).toBe("");
  });
});
