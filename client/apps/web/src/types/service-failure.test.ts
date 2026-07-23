import { findChoice, serviceFailureStatusChoices } from "@/lib/choices";
import { nullableTextSchema } from "@trenova/shared/types/helpers";
import {
  serviceFailureEvaluationResultSchema,
  serviceFailureSourceSchema,
  serviceFailureTypeSchema,
  serviceFailureUpdateSchema,
} from "@/types/service-failure";
import {
  serviceFailureReasonCategorySchema,
  serviceFailureReasonCodeAppliesToSchema,
  serviceFailureReasonCodeSchema,
} from "@/types/service-failure-reason-code";
import { describe, expect, it } from "vitest";

describe("service failure shared schemas", () => {
  it("normalizes nullish text to empty strings", () => {
    expect(nullableTextSchema.parse(null)).toBe("");
    expect(nullableTextSchema.parse(undefined)).toBe("");
    expect(nullableTextSchema.parse("notes")).toBe("notes");
  });

  it("uses shared nullable text schema in service failure updates", () => {
    const parsed = serviceFailureUpdateSchema.parse({
      id: "sf_123",
      shipmentId: "sp_123",
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

  it("accepts expanded service failure enum values", () => {
    expect(serviceFailureSourceSchema.parse("EDI")).toBe("EDI");
    expect(serviceFailureSourceSchema.parse("Integration")).toBe("Integration");
    expect(serviceFailureTypeSchema.parse("MissedPickup")).toBe("MissedPickup");
    expect(serviceFailureTypeSchema.parse("MissedDelivery")).toBe("MissedDelivery");
    expect(serviceFailureTypeSchema.parse("AppointmentMissed")).toBe("AppointmentMissed");
    expect(serviceFailureReasonCategorySchema.parse("Driver")).toBe("Driver");
    expect(serviceFailureReasonCategorySchema.parse("Shipper")).toBe("Shipper");
    expect(serviceFailureReasonCategorySchema.parse("Consignee")).toBe("Consignee");
    expect(serviceFailureReasonCategorySchema.parse("Appointment")).toBe("Appointment");
    expect(serviceFailureReasonCodeAppliesToSchema.parse("All")).toBe("All");
  });

  it("normalizes nullish service failure evaluation results", () => {
    const parsed = serviceFailureEvaluationResultSchema.parse({
      createdIds: null,
      updatedIds: null,
      skippedStops: null,
      skipped: null,
    });

    expect(parsed.createdIds).toEqual([]);
    expect(parsed.updatedIds).toEqual([]);
    expect(parsed.createdStops).toEqual([]);
    expect(parsed.updatedStops).toEqual([]);
    expect(parsed.skippedStops).toEqual([]);
    expect(parsed.skipped).toBe(0);
  });

  it("accepts service failure evaluation skipped stop details", () => {
    const parsed = serviceFailureEvaluationResultSchema.parse({
      createdIds: [],
      updatedIds: [],
      skipped: 1,
      skippedStops: [
        {
          shipmentId: "shp_123",
          stopId: "stp_123",
          stopSequence: 2,
          stopType: "Delivery",
          locationId: "loc_123",
          locationName: "Warehouse 12",
          locationCode: "WH12",
          city: "Austin",
          stateCode: "TX",
          scheduledCutoff: 1_700_000_000,
          actualArrival: 1_700_000_900,
          gracePeriodMinutes: 10,
          lateMinutes: 5,
          reason: "missing actual arrival",
        },
      ],
    });

    expect(parsed.skippedStops[0]?.stopSequence).toBe(2);
    expect(parsed.skippedStops[0]?.locationName).toBe("Warehouse 12");
    expect(parsed.skippedStops[0]?.stateCode).toBe("TX");
    expect(parsed.skippedStops[0]?.scheduledCutoff).toBe(1_700_000_000);
    expect(parsed.skippedStops[0]?.reason).toBe("missing actual arrival");
  });

  it("accepts service failure evaluation created and updated stop details", () => {
    const parsed = serviceFailureEvaluationResultSchema.parse({
      createdIds: ["sf_created"],
      updatedIds: ["sf_updated"],
      createdStops: [
        {
          shipmentId: "sp_123",
          shipmentMoveId: "sm_123",
          stopId: "stp_created",
          stopSequence: 1,
          stopType: "Pickup",
          locationName: "Austin Yard",
          serviceFailureId: "sf_created",
        },
      ],
      updatedStops: [
        {
          shipmentId: "sp_123",
          shipmentMoveId: "sm_123",
          stopId: "stp_updated",
          stopSequence: 2,
          stopType: "Delivery",
          locationName: "Dallas Yard",
          serviceFailureId: "sf_updated",
        },
      ],
      skippedStops: [],
      skipped: 0,
    });

    expect(parsed.createdStops[0]?.serviceFailureId).toBe("sf_created");
    expect(parsed.updatedStops[0]?.locationName).toBe("Dallas Yard");
  });
});

describe("findChoice", () => {
  it("returns the matching choice by value", () => {
    expect(findChoice(serviceFailureStatusChoices, "Reviewed")?.label).toBe("Reviewed");
    expect(findChoice(serviceFailureStatusChoices, "Open")?.color).toBe("#dc2626");
  });
});
