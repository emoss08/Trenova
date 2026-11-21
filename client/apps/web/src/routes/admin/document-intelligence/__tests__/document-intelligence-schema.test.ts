import { describe, expect, it } from "vitest";
import { documentControlSchema, type DocumentControl } from "@/types/document-control";
import { savedDocumentControl } from "./document-intelligence-fixture";

const issues = (changes: Partial<DocumentControl>) => {
  const result = documentControlSchema.safeParse({ ...savedDocumentControl, ...changes });
  return result.success ? [] : result.error.issues.map((issue) => issue.path.join("."));
};

describe("documentControlSchema", () => {
  it("accepts the record the server sends", () => {
    expect(issues({})).toEqual([]);
  });

  it("accepts every kind of record the server lets drafts come from", () => {
    expect(
      issues({
        enableShipmentDraftExtraction: true,
        shipmentDraftAllowedResources: ["shipment", "trailer", "tractor", "worker"],
      }),
    ).toEqual([]);
  });

  it("refuses a kind of record the server does not draft from", () => {
    expect(
      issues({
        shipmentDraftAllowedResources: [
          "location",
        ] as unknown as DocumentControl["shipmentDraftAllowedResources"],
      }),
    ).toEqual(["shipmentDraftAllowedResources.0"]);
  });

  it("needs a record to draft from while drafting is on", () => {
    expect(
      issues({ enableShipmentDraftExtraction: true, shipmentDraftAllowedResources: [] }),
    ).toEqual(["shipmentDraftAllowedResources"]);
  });

  it("keeps an empty list while drafting is off", () => {
    expect(
      issues({ enableShipmentDraftExtraction: false, shipmentDraftAllowedResources: [] }),
    ).toEqual([]);
  });

  it("holds intake retention between a day and a year", () => {
    expect(issues({ captureRetentionDays: 0 })).toEqual(["captureRetentionDays"]);
    expect(issues({ captureRetentionDays: 366 })).toEqual(["captureRetentionDays"]);
    expect(issues({ captureRetentionDays: 1 })).toEqual([]);
    expect(issues({ captureRetentionDays: 365 })).toEqual([]);
  });

  it("takes a minimum agent version only as major.minor.patch", () => {
    expect(issues({ captureMinAgentVersion: "v1.4.0" })).toEqual([]);
    expect(issues({ captureMinAgentVersion: "1.4" })).toEqual(["captureMinAgentVersion"]);
  });
});
