import { describe, expect, it } from "vitest";
import {
  calculationTraceSchema,
  detentionPolicySchema,
  type DetentionPolicy,
} from "@trenova/shared/types/detention";
import { toDetentionPolicyInput } from "./detention";

function policy(overrides: Partial<DetentionPolicy> = {}): DetentionPolicy {
  return {
    name: "Standard detention terms",
    code: "DET-STD",
    description: "",
    status: "Active",
    isOrgDefault: true,
    priority: 0,
    specificityScore: 0,
    customerId: null,
    locationId: null,
    shipmentTypeIds: null,
    serviceTypeIds: null,
    commodityIds: null,
    stopTypes: null,
    appointmentStopsOnly: false,
    effectiveStartDate: null,
    effectiveEndDate: null,
    clockStartBasis: "LaterOfArrivalOrAppointment",
    lateArrivalRule: "NoEffect",
    lateArrivalGraceMinutes: 15,
    billingFreeMinutes: 120,
    pickupFreeMinutes: null,
    deliveryFreeMinutes: null,
    payFreeMinutes: null,
    minimumBillableMinutes: 15,
    billingIncrementMinutes: 15,
    roundingMode: "Up",
    rateSource: "Accessorial",
    accessorialChargeId: "acc_1",
    tiers: [],
    maxBillableMinutesPerStop: null,
    maxChargePerStop: null,
    maxChargePerDay: null,
    maxChargePerShipment: null,
    dayBoundaryMode: "PerStop",
    convertToLayoverAtMinutes: null,
    layoverAccessorialChargeId: null,
    notificationRequirement: "Advisory",
    notificationLeadMinutes: 30,
    notificationDeadlineMinutes: 0,
    unnotifiedBehavior: "Bill",
    autoSendNotice: false,
    sendDepartureSummary: true,
    requireApprovalOverAmount: null,
    autoApproveUnderAmount: null,
    currency: "USD",
    comments: "",
    ...overrides,
  } as DetentionPolicy;
}

describe("toDetentionPolicyInput", () => {
  // A policy with no rate ladder arrives from the server with tiers: null, and
  // both the preview and backtest tabs map the form values before requesting.
  it("accepts a policy whose tiers are null", () => {
    const input = toDetentionPolicyInput(
      policy({ tiers: null as unknown as DetentionPolicy["tiers"] }),
    );

    expect(input.tiers).toBeNull();
  });

  it("maps a rate ladder in order with a fallback sort order", () => {
    const input = toDetentionPolicyInput(
      policy({
        rateSource: "Tiers",
        tiers: [
          { fromMinute: 0, toMinute: 240, rate: 60, rateUnit: "Hour", label: "First 4h" },
          { fromMinute: 240, toMinute: null, rate: 85, rateUnit: "Hour", label: "" },
        ] as unknown as DetentionPolicy["tiers"],
      }),
    );

    expect(input.tiers).toEqual([
      {
        fromMinute: 0,
        toMinute: 240,
        rate: "60",
        rateUnit: "Hour",
        label: "First 4h",
        sortOrder: 0,
      },
      {
        fromMinute: 240,
        toMinute: null,
        rate: "85",
        rateUnit: "Hour",
        label: "",
        sortOrder: 1,
      },
    ]);
  });
});

describe("detention schemas", () => {
  it("normalizes a null tier list to an empty array", () => {
    const parsed = detentionPolicySchema.parse({
      ...policy(),
      tiers: null,
    });

    expect(parsed.tiers).toEqual([]);
  });

  it("normalizes a null trace step list to an empty array", () => {
    const parsed = calculationTraceSchema.parse({
      steps: null,
      engineVersion: "1.0.0",
      computedAt: 1_800_000_000,
      deterministic: true,
    });

    expect(parsed.steps).toEqual([]);
  });
});
