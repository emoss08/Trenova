import { describe, expect, it } from "vitest";
import { complianceStatusMeta, safetyRatingMeta, trainingHealthMetaOf } from "../worker-health";

describe("complianceStatusMeta", () => {
  it("reads only Compliant as good", () => {
    expect(complianceStatusMeta("Compliant").good).toBe(true);
    expect(complianceStatusMeta("NonCompliant").good).toBe(false);
    expect(complianceStatusMeta("Pending").good).toBe(false);
  });

  it("maps the three server values onto the badge palette", () => {
    expect(complianceStatusMeta("Compliant").badgeVariant).toBe("active");
    expect(complianceStatusMeta("NonCompliant").badgeVariant).toBe("inactive");
    expect(complianceStatusMeta("Pending").badgeVariant).toBe("warning");
  });

  it("hyphenates the label a human reads", () => {
    expect(complianceStatusMeta("NonCompliant").label).toBe("Non-compliant");
  });

  // A value the client has never heard of must still draw a row: the server
  // may add a state before the client learns the word for it.
  it("falls back to a neutral meta for an unknown value", () => {
    const meta = complianceStatusMeta("Suspended");
    expect(meta.label).toBe("Suspended");
    expect(meta.badgeVariant).toBe("secondary");
    expect(meta.good).toBe(false);
  });
});

describe("safetyRatingMeta", () => {
  it("reads Excellent and Good as good, Watch and AtRisk as not", () => {
    expect(safetyRatingMeta("Excellent").good).toBe(true);
    expect(safetyRatingMeta("Good").good).toBe(true);
    expect(safetyRatingMeta("Watch").good).toBe(false);
    expect(safetyRatingMeta("AtRisk").good).toBe(false);
  });

  it("uses the CSA label rather than the enum spelling", () => {
    expect(safetyRatingMeta("AtRisk").label).toBe("At risk");
  });
});

describe("trainingHealthMetaOf", () => {
  it("reads only Current as good", () => {
    expect(trainingHealthMetaOf("Current").good).toBe(true);
    expect(trainingHealthMetaOf("DueSoon").good).toBe(false);
    expect(trainingHealthMetaOf("Scheduled").good).toBe(false);
  });

  it("carries the blocking flag through from the typed helper", () => {
    expect(trainingHealthMetaOf("Overdue").blocks).toBe(true);
    expect(trainingHealthMetaOf("Failed").blocks).toBe(true);
    expect(trainingHealthMetaOf("Missing").blocks).toBe(true);
    expect(trainingHealthMetaOf("ExpiringSoon").blocks).toBe(false);
  });

  it("treats an unknown value as Missing rather than throwing", () => {
    const meta = trainingHealthMetaOf("Unheard");
    expect(meta.label).toBe("Missing");
    expect(meta.blocks).toBe(true);
    expect(meta.good).toBe(false);
  });
});
