import type { DeskServiceFailure } from "@/lib/graphql/service-failure-card";

export const NOW = 1_791_720_000;
export const HOUR = 3_600;

/** A failure as the DeskServiceFailureCard query returns it, open and unreviewed. */
export function serviceFailure(overrides: Partial<DeskServiceFailure> = {}): DeskServiceFailure {
  return {
    id: "sf_1",
    shipmentId: "shp_1",
    number: "SF-2610-0042",
    type: "LateDelivery",
    source: "Detected",
    status: "Open",
    stopType: "Delivery",
    scheduledCutoff: NOW - 4 * HOUR,
    actualArrival: NOW - 2 * HOUR + 15 * 60,
    gracePeriodMinutes: 30,
    lateMinutes: 105,
    notes: "",
    internalNotes: "",
    x12StatusCodeOverride: "",
    x12ReasonCodeOverride: "",
    x12ExceptionCode: "",
    detectedAt: NOW - 2 * HOUR,
    reviewedAt: null,
    resolvedAt: null,
    voidedAt: null,
    voidReason: "",
    createdAt: NOW - 2 * HOUR,
    updatedAt: NOW - HOUR,
    createdBy: null,
    reviewedBy: null,
    resolvedBy: null,
    voidedBy: null,
    shipment: {
      id: "shp_1",
      proNumber: "S2610-1366",
      bol: "BOL-55120",
      customer: { id: "cus_1", code: "TYSON", name: "Tyson Foods" },
    },
    stop: {
      id: "stp_3",
      type: "Delivery",
      sequence: 3,
      location: {
        id: "loc_1",
        code: "TYSDC",
        name: "Tyson Foods DC",
        addressLine1: "2200 W Don Tyson Pkwy",
        city: "Springdale",
        postalCode: "72762",
        latitude: 36.1867,
        longitude: -94.1288,
        state: { abbreviation: "AR" },
      },
    },
    reasonCode: null,
    ...overrides,
  } as DeskServiceFailure;
}
