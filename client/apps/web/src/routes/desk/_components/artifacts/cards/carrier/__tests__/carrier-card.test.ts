import type { CarrierIntelLabels } from "@/components/carrier-intelligence/use-carrier-intel-labels";
import type {
  CarrierEligibility,
  CarrierIntel,
  CarrierSettlementRow,
  DeskCarrier,
} from "@/lib/graphql/carrier-card";
import type { TranslateFn } from "@trenova/shared/i18n/use-t";
import { describe, expect, it } from "vitest";
import {
  authorityStats,
  basicMeters,
  carrierFacts,
  carrierKicker,
  carrierSubtitle,
  complianceStanding,
  contactLine,
  eligibilityChecks,
  eligibilityVerdict,
  insuranceChecks,
  orderedContacts,
  riskReading,
  settlementAmount,
  settlementLine,
} from "../carrier-card";

const t = ((message: string, ...args: unknown[]) =>
  message
    .replace(
      /\{(\d+), plural, one \{([^}]*)\} other \{([^}]*)\}\}/gu,
      (_, index, one: string, other: string) => {
        const count = Number(args[Number(index)]);
        return (count === 1 ? one : other).replace("#", String(count));
      },
    )
    .replace(/\{(\d+)\}/gu, (_, index) => String(args[Number(index)]))) as TranslateFn;

const NOW = 1_791_720_000;
const DAY = 86_400;

const labels = {
  authorityStatus: {
    Active: "Active",
    Inactive: "Inactive",
    Revoked: "Revoked",
    None: "None",
    Unknown: "Unknown",
  },
} as CarrierIntelLabels;

function carrier(overrides: Partial<DeskCarrier> = {}): DeskCarrier {
  return {
    id: "car_1",
    status: "Active",
    code: "RRF",
    name: "Red River Freight",
    dbaName: null,
    carrierType: "Common",
    dotNumber: "3141592",
    mcNumber: "MC-271828",
    scac: "RRFT",
    complianceStatus: "Qualified",
    safetyRating: "Satisfactory",
    qualifiedAt: NOW - 200 * DAY,
    disqualifiedReason: null,
    w9OnFile: true,
    paymentMethod: "ACHManual",
    paymentTermDays: 30,
    remitToName: null,
    addressLine1: "400 Industrial Blvd",
    addressLine2: null,
    city: "Shreveport",
    postalCode: "71101",
    phone: "(318) 555-0190",
    email: "dispatch@redriver.example",
    externalId: null,
    notes: null,
    intelRiskLevel: "Low",
    intelReviewRequired: false,
    createdAt: NOW - 900 * DAY,
    state: { abbreviation: "LA" },
    contacts: [],
    insurancePolicies: [],
    ...overrides,
  } as DeskCarrier;
}

type Policy = NonNullable<DeskCarrier["insurancePolicies"]>[number];

function policy(overrides: Partial<Policy> = {}): Policy {
  return {
    id: "pol_1",
    policyType: "CargoLiability",
    policyNumber: "CG-1182",
    providerName: "Great West Casualty",
    coverageAmount: "100000",
    effectiveDate: NOW - 200 * DAY,
    expirationDate: NOW + 200 * DAY,
    isVerified: true,
    ...overrides,
  } as Policy;
}

function intel(overrides: Partial<CarrierIntel["profile"]> = {}): CarrierIntel {
  return {
    id: "snap_1",
    provider: "CarrierOK",
    notFound: false,
    riskLevel: "Low",
    fetchedAt: NOW - 3_600,
    effectiveAsOf: NOW - 3_600,
    profile: {
      authority: {
        common: { status: "Active", pending: false, revocationPending: false, grantedAt: 1 },
        contract: null,
        broker: { status: "Revoked", pending: false, revocationPending: false, grantedAt: 1 },
        totalRevocations: 1,
        lastRevocationAt: NOW - 400 * DAY,
      },
      safety: {
        rating: "Conditional",
        ratingDate: null,
        outOfServiceOrder: false,
        outOfServiceAt: null,
      },
      basics: [],
      ...overrides,
    },
  } as CarrierIntel;
}

describe("complianceStanding", () => {
  it("reads each compliance status as the pill says it, and an inactive carrier as muted", () => {
    expect(complianceStanding(carrier(), t)).toEqual({
      label: "Qualified",
      tone: "success",
      mark: "check",
    });
    expect(complianceStanding(carrier({ complianceStatus: "Pending" }), t).tone).toBe("progress");
    expect(complianceStanding(carrier({ complianceStatus: "Disqualified" }), t)).toMatchObject({
      tone: "warn",
      mark: "pause",
    });
    expect(complianceStanding(carrier({ complianceStatus: "Expired" }), t).label).toBe(
      "Qualification expired",
    );
    expect(complianceStanding(carrier({ status: "Inactive" }), t).label).toBe("Inactive");
  });
});

describe("carrierKicker", () => {
  it("reads the code, DOT, MC and SCAC, leaving out what is not on file", () => {
    expect(carrierKicker(carrier(), t)).toEqual(["RRF", "DOT 3141592", "MC 271828", "SCAC RRFT"]);
    expect(carrierKicker(carrier({ mcNumber: "271828" }), t)).toContain("MC 271828");
    expect(carrierKicker(carrier({ mcNumber: "mc# 99" }), t)).toContain("MC 99");
    expect(carrierKicker(carrier({ mcNumber: " ", scac: null }), t)).toEqual([
      "RRF",
      "DOT 3141592",
    ]);
  });
});

describe("carrierSubtitle", () => {
  it("reads the name the carrier trades under, or else what kind of carrier and where", () => {
    expect(carrierSubtitle(carrier({ dbaName: " Red River Logistics " }), t)).toBe(
      "Doing business as Red River Logistics",
    );
    expect(carrierSubtitle(carrier(), t)).toBe("Common carrier · Shreveport, LA");
    expect(carrierSubtitle(carrier({ city: null, state: null }), t)).toBe("Common carrier");
  });
});

describe("riskReading", () => {
  it("marks an elevated risk and a high one, and reads nothing for an unknown or unvetted carrier", () => {
    expect(riskReading("Low")).toEqual({ label: "Low", state: "ok" });
    expect(riskReading("Elevated")?.state).toBe("warn");
    expect(riskReading("VeryHigh")).toEqual({ label: "Very high", state: "bad" });
    expect(riskReading("Unknown")).toBeNull();
    expect(riskReading(null)).toBeNull();
  });
});

describe("eligibility", () => {
  const eligibility = {
    blockers: ["Carrier compliance status is Disqualified"],
    warnings: ["Cargo insurance expires in 9 days"],
    advisories: ["Authority granted 4 months ago"],
    findings: [
      {
        code: "intel.authority_age",
        source: "Intelligence",
        severity: "Advisory",
        message: "Authority granted 4 months ago",
        requiresOverride: false,
      },
      {
        code: "record.insurance_expiring",
        source: "Record",
        severity: "Warning",
        message: "Cargo insurance expires in 9 days",
        requiresOverride: true,
      },
      {
        code: "record.compliance_status",
        source: "Record",
        severity: "Blocker",
        message: "Carrier compliance status is Disqualified",
        requiresOverride: false,
      },
    ],
  } as CarrierEligibility;

  it("reads a blocked carrier as blocked, one with warnings as needing an override, else clear", () => {
    expect(eligibilityVerdict(eligibility, t)).toEqual({ label: "Blocked", state: "bad" });
    expect(eligibilityVerdict({ ...eligibility, blockers: [] }, t).label).toBe("Needs an override");
    expect(eligibilityVerdict({ ...eligibility, blockers: [], warnings: [] }, t)).toEqual({
      label: "Clear to tender",
      state: "ok",
    });
  });

  it("lists what stands in the way worst first, saying where each comes from", () => {
    expect(
      eligibilityChecks(eligibility, t).map((check) => [check.state, check.title, check.detail]),
    ).toEqual([
      ["bad", "Carrier compliance status is Disqualified", "On the carrier's record"],
      ["warn", "Cargo insurance expires in 9 days", "On the carrier's record · Needs an override"],
      ["info", "Authority granted 4 months ago", "From intelligence"],
    ]);
  });
});

describe("insuranceChecks", () => {
  it("reads each policy by its expiry, worst first, and an unverified one as worth a look", () => {
    const checks = insuranceChecks(
      [
        policy(),
        policy({ id: "pol_2", policyType: "AutoLiability", expirationDate: NOW + 9 * DAY }),
        policy({ id: "pol_3", policyType: "GeneralLiability", expirationDate: NOW - DAY }),
        policy({ id: "pol_4", policyType: "Umbrella", isVerified: false }),
      ],
      NOW,
      t,
    );
    expect(checks.map((check) => [check.state, check.title])).toEqual([
      ["bad", "General liability expired"],
      ["warn", "Auto liability expires in 9 days"],
      ["warn", "Umbrella"],
      ["ok", "Cargo liability"],
    ]);
    expect(checks[2]?.detail).toMatch(/· Not verified$/u);
    expect(checks[3]?.detail).toMatch(/^Great West Casualty · \$100,000\.00 · Through /u);
  });
});

describe("authorityStats", () => {
  it("reads each authority and the safety rating, marking a revoked grant and a conditional rating", () => {
    expect(
      authorityStats(intel(), labels, t).map((stat) => [stat.label, stat.value, stat.state]),
    ).toEqual([
      ["Common authority", "Active", "ok"],
      ["Contract authority", "None", undefined],
      ["Broker authority", "Revoked", "bad"],
      ["Safety rating", "Conditional", "warn"],
    ]);
  });

  it("says an authority pending or about to be revoked, and an out-of-service order above any rating", () => {
    const stats = authorityStats(
      intel({
        authority: {
          common: { status: "Active", pending: false, revocationPending: true, grantedAt: 1 },
          contract: { status: "Active", pending: true, revocationPending: false, grantedAt: 1 },
          broker: null,
          totalRevocations: 0,
          lastRevocationAt: null,
        },
        safety: {
          rating: "Satisfactory",
          ratingDate: null,
          outOfServiceOrder: true,
          outOfServiceAt: NOW,
        },
      }),
      labels,
      t,
    );
    expect(stats.map((stat) => [stat.value, stat.state])).toEqual([
      ["Active, revocation pending", "bad"],
      ["Active, pending", "warn"],
      ["None", undefined],
      ["Out-of-service order", "bad"],
    ]);
  });
});

describe("basicMeters", () => {
  it("orders the BASICs the FMCSA's way, each read against its alert threshold", () => {
    const meters = basicMeters(
      [
        { basic: "VehicleMaintenance", measure: 4.1, percentile: 72, threshold: 80, alert: false },
        { basic: "UnsafeDriving", measure: 2.2, percentile: 41, threshold: 65, alert: false },
        { basic: "HOSCompliance", measure: 6.4, percentile: 81, threshold: 65, alert: true },
        { basic: "DriverFitness", measure: null, percentile: null, threshold: 80, alert: false },
      ],
      t,
    );
    expect(meters.map((meter) => [meter.label, meter.reading, meter.state])).toEqual([
      ["Unsafe Driving", "41 of 65", "ok"],
      ["Hours of Service", "81 of 65", "bad"],
      ["Driver Fitness", "Not enough data", "ok"],
      ["Vehicle Maintenance", "72 of 80", "warn"],
    ]);
    expect(meters[0]?.share).toBeCloseTo(0.41);
    expect(meters[2]?.share).toBe(0);
  });
});

describe("settlements and contacts", () => {
  it("reads a settlement's period, pay date, loads and amount in its own currency", () => {
    const row = {
      id: "cs_1",
      settlementNumber: "CS-0042",
      status: "Paid",
      periodStart: NOW - 14 * DAY,
      periodEnd: NOW - DAY,
      payDate: NOW,
      netPayableMinor: 1_245_050,
      shipmentCount: 1,
      currencyCode: "USD",
    } as CarrierSettlementRow;
    expect(settlementLine(row, t)).toMatch(/ to .* · Pays .* · 1 shipment$/u);
    expect(settlementAmount(row)).toBe("$12,450.50");
  });

  it("puts the primary contact first and reads how to reach each", () => {
    const contacts = orderedContacts([
      {
        id: "c2",
        name: "Zoe",
        title: null,
        email: "zoe@x.example",
        phone: null,
        isPrimary: false,
        receivesRateConfirmations: false,
      },
      {
        id: "c1",
        name: "Ana",
        title: "Dispatch",
        email: null,
        phone: "555",
        isPrimary: false,
        receivesRateConfirmations: true,
      },
      {
        id: "c3",
        name: "Mo",
        title: "Owner",
        email: null,
        phone: null,
        isPrimary: true,
        receivesRateConfirmations: false,
      },
    ] as NonNullable<DeskCarrier["contacts"]>);
    expect(contacts.map((contact) => contact.name)).toEqual(["Mo", "Ana", "Zoe"]);
    expect(contactLine(contacts[1]!)).toBe("Dispatch · 555");
  });
});

describe("carrierFacts", () => {
  it("lists the type, payment, remit-to and how to reach the carrier, saying what is not on file", () => {
    const facts = Object.fromEntries(
      carrierFacts(carrier({ phone: null, w9OnFile: false, qualifiedAt: null }), t).map((fact) => [
        fact.key,
        fact.value,
      ]),
    );
    expect(facts).toMatchObject({
      type: "Common",
      payment: "ACH (manual), net 30",
      remit: "Red River Freight",
      w9: "Not on file",
      phone: "None on file",
      address: "400 Industrial Blvd, Shreveport, LA 71101",
      qualified: "Not yet",
    });
  });
});
