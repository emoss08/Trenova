import { type CardDetail, type DeskDetention, toDeskDetention } from "@/lib/graphql/detention-card";
import { formatMessage } from "@trenova/shared/i18n/format-message";
import type { TranslateFn } from "@trenova/shared/i18n/runtime";

/** Midnight UTC on Oct 14, 2026, the day every stop here was worked. */
export const OCT14 = 1_791_936_000;
export const MINUTE = 60;
export const HOUR = 3_600;

/** A moment on Oct 14 by its UTC hour and minute. */
export function at(hour: number, minute = 0): number {
  return OCT14 + hour * HOUR + minute * MINUTE;
}

/** A translator that fills ICU arguments and plurals the way the catalog does. */
export const t: TranslateFn = (message, ...args) => formatMessage("en", message ?? "", args);

type RawOccurrence = CardDetail["occurrence"];

/**
 * The policy snapshot as the server stores it on the occurrence
 * (`detentionpolicy.PolicySnapshot`): decimals as strings, nullable decimals as
 * null, and every field the engine writes, including the ones the card does
 * not read.
 */
export function snapshotJson(overrides: Record<string, unknown> = {}): Record<string, unknown> {
  return {
    policyId: "dtp_kroger",
    policyCode: "KRG-DC",
    policyName: "Kroger DC standard",
    policyVersion: 3,
    specificityScore: 6,
    priority: 20,
    clockStartBasis: "LaterOfArrivalOrAppointment",
    lateArrivalRule: "ClockFromAppointment",
    lateArrivalGraceMinutes: 0,
    freeMinutes: 120,
    payFreeMinutes: 180,
    minimumBillableMinutes: 15,
    billingIncrementMinutes: 15,
    roundingMode: "Up",
    rateSource: "Tiers",
    accessorialChargeId: "",
    accessorialCode: "DET",
    flatRate: "0",
    flatRateUnit: "Hour",
    tiers: [{ fromMinute: 0, toMinute: null, rate: "75", rateUnit: "Hour", label: "Tier 1" }],
    maxBillableMinutesPerStop: null,
    maxChargePerStop: null,
    maxChargePerDay: null,
    maxChargePerShipment: null,
    dayBoundaryMode: "PerStop",
    convertToLayoverAtMinutes: null,
    layoverAccessorialChargeId: null,
    notificationRequirement: "Required",
    notificationLeadMinutes: 30,
    notificationDeadlineMinutes: 0,
    unnotifiedBehavior: "Flag",
    requireApprovalOverAmount: "250",
    autoApproveUnderAmount: null,
    currency: "USD",
    capturedAt: at(15, 1),
    ...overrides,
  };
}

/**
 * The calculation trace the engine writes for the Kroger stop
 * (`detentionservice.calculator`), in its own words: decimals as strings,
 * `term`, `minutes`, `amount` and `timestamp` left out when a step has none.
 */
export function traceJson(): Record<string, unknown> {
  return {
    engineVersion: "1.0.0",
    computedAt: at(15, 1),
    deterministic: true,
    steps: [
      {
        kind: "PolicyResolved",
        label: "Policy applied",
        detail: "Kroger DC standard (KRG-DC)",
        term: "specificity 6, priority 20",
        isReduction: false,
      },
      {
        kind: "ClockStart",
        label: "Clock started",
        detail: "Later of arrival or appointment start",
        term: "Clock start basis: LaterOfArrivalOrAppointment",
        timestamp: at(8, 42),
        isReduction: false,
      },
      {
        kind: "ClockStop",
        label: "Clock stopped",
        detail: "Departure recorded",
        term: "Clock stop basis: Departure",
        timestamp: at(15),
        isReduction: false,
      },
      {
        kind: "RawDwell",
        label: "Total time on site",
        detail: "6h 18m",
        minutes: 378,
        isReduction: false,
      },
      {
        kind: "FreeTime",
        label: "Free time deducted",
        detail: "2h 00m",
        term: "120 min contractual free time",
        minutes: 120,
        isReduction: true,
      },
      {
        kind: "Rounding",
        label: "Rounded 4h 18m to 4h 30m",
        detail: "Rounded up to the next 15 minutes",
        term: "Up to 15 min increments",
        minutes: 270,
        isReduction: false,
      },
      {
        kind: "TierApplied",
        label: "Tier 1",
        detail: "4h 30m at $75.00/hr",
        term: "From minute 0",
        amount: "337.5",
        isReduction: false,
      },
      {
        kind: "NotificationGate",
        label: "Notice requirement satisfied",
        detail: "Customer was notified within the contractual window",
        term: "Notification: Required",
        isReduction: false,
      },
      {
        kind: "DriverPay",
        label: "Driver detention pay",
        detail: "3h 15m beyond 3h 00m of pay free time",
        term: "Driver pay free time: 3h 00m",
        amount: "130",
        isReduction: false,
      },
      {
        kind: "Final",
        label: "Billable detention",
        detail: "4h 30m billable of 6h 18m on site",
        amount: "337.5",
        isReduction: false,
      },
    ],
  };
}

/**
 * The Kroger delivery waiting on approval, as the card's query answers it:
 * the appointment 8:00 to 9:00, arrived 8:42, free time to 10:42, notice sent
 * 10:09 inside its window, departed 3:00 PM; 6h 18m on site, 4h 18m billable
 * rounded to 4h 30m at $75.00 an hour, over the $250.00 approval line.
 */
export function rawOccurrence(overrides: Partial<RawOccurrence> = {}): RawOccurrence {
  return {
    id: "dto_kroger",
    shipmentId: "shp_240193",
    stopId: "stp_3",
    customerId: "cus_kroger",
    locationId: "loc_kroger_atl",
    policySnapshot: snapshotJson(),
    calculationTrace: traceJson(),
    stopType: "Delivery",
    scheduleType: "Appointment",
    appointmentStart: at(8),
    appointmentEnd: at(9),
    arrivedAt: at(8, 42),
    departedAt: at(15),
    clockStartAt: at(8, 42),
    clockStopAt: at(15),
    freeTimeExpiresAt: at(10, 42),
    noticeDueAt: at(10, 12),
    noticeDeadlineAt: at(10, 42),
    isOpen: false,
    arrivedLate: false,
    lateByMinutes: 0,
    freeMinutesGranted: 120,
    rawDwellMinutes: 378,
    billableMinutes: 258,
    roundedMinutes: 270,
    billableAmount: "337.5",
    driverPayMinutes: 195,
    driverPayAmount: "130",
    netMargin: "207.5",
    capApplied: "None",
    convertedToLayover: false,
    currency: "USD",
    status: "Pending",
    notificationStatus: "Sent",
    noticeSentAt: at(10, 9),
    suppressedByGate: false,
    requiresApproval: true,
    approvedAt: null,
    waiverReason: null,
    waiverNote: "",
    waivedAt: null,
    waivedAmount: "0",
    disputeNote: "",
    disputedAt: null,
    locationName: "Kroger Atlanta DC",
    customerName: "The Kroger Co.",
    shipmentProNumber: "S-240193",
    createdAt: at(8, 42),
    updatedAt: at(15, 1),
    approvedBy: null,
    waivedBy: null,
    location: {
      id: "loc_kroger_atl",
      city: "Forest Park",
      state: { id: "st_ga", abbreviation: "GA" },
    },
    ...overrides,
  };
}

/** The card's whole answer for an occurrence: evidence out of order, one notice, the score. */
export function rawDetail(overrides: Partial<CardDetail> = {}): CardDetail {
  return {
    occurrence: rawOccurrence(),
    evidence: [
      {
        id: "dte_departure",
        kind: "Departure",
        source: "Telematics",
        summary: "Truck 2207 left the geofence",
        observedAt: at(15),
      },
      {
        id: "dte_arrival",
        kind: "Arrival",
        source: "Telematics",
        summary: "Truck 2207 inside the Kroger Atlanta geofence",
        observedAt: at(8, 42),
      },
      {
        id: "dte_notice",
        kind: "Notice",
        source: "System",
        summary: "Warning notice delivered to 2 recipients",
        observedAt: at(10, 9),
      },
    ],
    notices: [
      {
        id: "dtn_warning",
        kind: "Warning",
        channel: "Email",
        deliveryStatus: "Delivered",
        recipients: ["dock-atl@kroger.com", "ap@kroger.com"],
        sentAt: at(10, 9),
        scheduledFor: at(10, 9),
        failureReason: "",
        satisfiesRequirement: true,
        wasAutomatic: true,
      },
    ],
    collectability: {
      score: 86,
      band: "Strong",
      chainValid: true,
      summary: "Strong claim",
      factors: [
        {
          key: "arrival",
          label: "Arrival timestamp",
          earned: 20,
          possible: 20,
          detail: "Telematics",
          remedy: "",
        },
        {
          key: "notice",
          label: "Customer notice",
          earned: 25,
          possible: 25,
          detail: "Sent in window",
          remedy: "",
        },
        {
          key: "document",
          label: "Supporting document",
          earned: 11,
          possible: 25,
          detail: "No signed in and out times",
          remedy: "Attach the POD with the receiver's in and out times.",
        },
      ],
    },
    currentlyBillable: null,
    ...overrides,
  };
}

/** The Kroger charge as the card reads it, waiting on approval. */
export function pendingCharge(): DeskDetention {
  return toDeskDetention(rawDetail());
}

/** The moment the accruing stop is read: 8:38 AM, the truck still at the dock. */
export const ACCRUING_NOW = at(8, 38);

/**
 * A pickup still on the clock at 8:38: arrived 5:51, the clock from the 6:00
 * appointment, free time to 8:00, notice due 7:30 and not sent, and a policy
 * that holds an unnotified charge back; 38 minutes billable so far, rounded to
 * 45 and priced at $56.25.
 */
export function accruingCharge(): DeskDetention {
  return toDeskDetention(
    rawDetail({
      occurrence: rawOccurrence({
        id: "dto_lineage",
        shipmentId: "shp_240211",
        customerId: "cus_acme",
        locationId: "loc_lineage_dal",
        policySnapshot: snapshotJson({
          policyName: "Cold storage pickups",
          policyVersion: 2,
          unnotifiedBehavior: "Suppress",
          notificationDeadlineMinutes: 15,
          requireApprovalOverAmount: null,
        }),
        calculationTrace: null,
        stopType: "Pickup",
        appointmentStart: at(6),
        appointmentEnd: null,
        arrivedAt: at(5, 51),
        departedAt: null,
        clockStartAt: at(6),
        clockStopAt: null,
        freeTimeExpiresAt: at(8),
        noticeDueAt: at(7, 30),
        noticeDeadlineAt: at(8, 45),
        isOpen: true,
        rawDwellMinutes: 0,
        billableMinutes: 0,
        roundedMinutes: 0,
        billableAmount: "0",
        driverPayMinutes: 0,
        driverPayAmount: "0",
        netMargin: "0",
        status: "Accruing",
        notificationStatus: "Pending",
        noticeSentAt: null,
        requiresApproval: false,
        locationName: "Lineage Logistics Dallas",
        customerName: "Acme Foods",
        shipmentProNumber: "S-240211",
        location: null,
      }),
      notices: null,
      evidence: null,
      collectability: {
        score: 48,
        band: "Weak",
        chainValid: true,
        summary: "Weak claim",
        factors: [
          {
            key: "notice",
            label: "Customer notice",
            earned: 0,
            possible: 25,
            detail: "Not sent",
            remedy: "Send the notice now; a late notice still earns part of this.",
          },
        ],
      },
      currentlyBillable: { billableMinutes: 45, billableAmount: "56.25", asOf: ACCRUING_NOW },
    }),
  );
}

/** The Home Depot charge waived for goodwill before anyone approved it. */
export function waivedCharge(): DeskDetention {
  return toDeskDetention(
    rawDetail({
      occurrence: rawOccurrence({
        id: "dto_hd",
        status: "Waived",
        billableAmount: "187.5",
        waivedAmount: "187.5",
        waiverReason: "CustomerGoodwill",
        waiverNote: "Waived for the Q4 volume commitment.",
        waivedAt: at(17, 12),
        waivedBy: { id: "usr_dana", name: "Dana Ruiz" },
        locationName: "Home Depot RDC Atlanta",
        customerName: "The Home Depot",
        shipmentProNumber: "S-240150",
      }),
    }),
  );
}
