import { findChoice, serviceFailureStatusChoices } from "@/lib/choices";
import { nullableTextSchema } from "@/types/helpers";
import { serviceFailureManualCreateSchema } from "@/types/service-failure";
import { serviceFailureReasonCodeSchema } from "@/types/service-failure-reason-code";
import { describe, expect, it } from "vitest";

describe("service failure shared schemas", () => {
  it("normalizes nullish text to empty strings", () => {
    expect(nullableTextSchema.parse(null)).toBe("");
    expect(nullableTextSchema.parse(undefined)).toBe("");
    expect(nullableTextSchema.parse("notes")).toBe("notes");
  });

  it("uses shared nullable text schema in service failure inputs", () => {
    const parsed = serviceFailureManualCreateSchema.parse({
      shipmentId: "sp_123",
      shipmentMoveId: "sm_123",
      stopId: "stp_123",
      reasonCodeId: "sfrc_123",
      type: "LateDelivery",
      notes: null,
      internalNotes: undefined,
    });

    expect(parsed.notes).toBe("");
    expect(parsed.internalNotes).toBe("");
  });

  it("uses shared nullable text schema in reason code inputs", () => {
    const parsed = serviceFailureReasonCodeSchema.parse({
      id: "sfrc_123",
      organizationId: "org_123",
      businessUnitId: "bu_123",
      code: "LATE",
      label: "Late delivery",
      description: null,
      defaultNote: undefined,
    });

    expect(parsed.description).toBe("");
    expect(parsed.defaultNote).toBe("");
  });
});

describe("findChoice", () => {
  it("returns the matching choice by value", () => {
    expect(findChoice(serviceFailureStatusChoices, "Reviewed")?.label).toBe("Reviewed");
    expect(findChoice(serviceFailureStatusChoices, "Open")?.color).toBe("#dc2626");
  });
});
