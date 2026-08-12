import { describe, expect, it } from "vitest";
import {
  latestActiveRateConfirmation,
  publicRateConfirmationSchema,
  rateConfirmationSchema,
  type RateConfirmation,
} from "./rate-confirmation";

function makeRateConfirmation(overrides: Partial<RateConfirmation>): RateConfirmation {
  return rateConfirmationSchema.parse({
    id: "ratecon_01",
    carrierAssignmentId: "casn_01",
    carrierId: "car_01",
    shipmentId: "shp_01",
    shipmentMoveId: "smv_01",
    revision: 1,
    status: "Generated",
    createdAt: 1,
    updatedAt: 1,
    ...overrides,
  });
}

describe("rateConfirmationSchema", () => {
  it("parses the REST payload shape with nulled optional fields", () => {
    const parsed = rateConfirmationSchema.parse({
      id: "ratecon_01",
      organizationId: "org_01",
      businessUnitId: "bu_01",
      carrierAssignmentId: "casn_01",
      carrierId: "car_01",
      shipmentId: "shp_01",
      shipmentMoveId: "smv_01",
      revision: 2,
      status: "Sent",
      documentId: null,
      sentAt: 1720000000,
      sentToEmails: "dispatch@carrier.test",
      confirmedAt: null,
      confirmedByName: "",
      voidedAt: null,
      voidReason: null,
      payloadSnapshot: { total: "1500.00" },
      generatedById: "usr_01",
      version: 3,
      createdAt: 1719990000,
      updatedAt: 1720000000,
    });

    expect(parsed.status).toBe("Sent");
    expect(parsed.documentId).toBeNull();
    expect(parsed.confirmedByName).toBeNull();
  });

  it("rejects unknown statuses", () => {
    expect(() => makeRateConfirmation({ status: "Draft" as never })).toThrow();
  });

  // The Go domain serializes Via fields without omitempty, so an unconfirmed
  // rate confirmation arrives with confirmedVia: "" — that must normalize to
  // null, never fail parsing.
  it("parses provenance fields and normalizes an empty confirmedVia to null", () => {
    const parsed = rateConfirmationSchema.parse({
      id: "ratecon_01",
      carrierAssignmentId: "casn_01",
      carrierId: "car_01",
      shipmentId: "shp_01",
      shipmentMoveId: "smv_01",
      revision: 1,
      status: "Sent",
      generatedVia: "TenderAcceptance",
      sourceTenderOfferId: "tof_01",
      confirmedVia: "",
      confirmedByTitle: "",
      createdAt: 1,
      updatedAt: 1,
    });

    expect(parsed.generatedVia).toBe("TenderAcceptance");
    expect(parsed.sourceTenderOfferId).toBe("tof_01");
    expect(parsed.confirmedVia).toBeNull();
    expect(parsed.confirmedByTitle).toBeNull();
  });

  it("parses a publicly signed confirmation's provenance", () => {
    const parsed = makeRateConfirmation({
      status: "Confirmed",
      confirmedVia: "PublicSignature",
      confirmedByTitle: "Operations Manager",
    });

    expect(parsed.confirmedVia).toBe("PublicSignature");
    expect(parsed.confirmedByTitle).toBe("Operations Manager");
  });

  it("tolerates payloads that predate provenance tracking", () => {
    const parsed = makeRateConfirmation({});

    expect(parsed.generatedVia ?? null).toBeNull();
    expect(parsed.confirmedVia ?? null).toBeNull();
  });

  it("rejects unknown via values", () => {
    expect(() => makeRateConfirmation({ confirmedVia: "Fax" as never })).toThrow();
  });
});

describe("publicRateConfirmationSchema", () => {
  // Fixture written from PublicRateConfirmationView in
  // services/tms/internal/core/services/rateconfirmationservice/public.go:
  // all strings are always present (possibly empty), stops is a JSON array.
  it("parses the public preview payload", () => {
    const parsed = publicRateConfirmationSchema.parse({
      carrierName: "Knight Logistics LLC",
      companyName: "Acme Brokerage",
      shipmentProNumber: "PRO-10422",
      revisionLabel: "Revision 2",
      rateMethodLabel: "Flat Rate",
      baseRateLabel: "$1,500.00",
      totalCost: "$1,650.00",
      currency: "USD",
      paymentTermsLabel: "Net 30",
      stops: [
        { sequence: 1, type: "Pickup", name: "DC 12", address: "100 Main St", window: "Jun 1" },
      ],
      status: "Sent",
      expiresAt: 1760000000,
      confirmed: false,
    });

    expect(parsed.carrierName).toBe("Knight Logistics LLC");
    expect(parsed.stops).toHaveLength(1);
    expect(parsed.stops[0].sequence).toBe(1);
  });

  it("accepts empty labels and an empty stop list on a confirmed snapshot", () => {
    const parsed = publicRateConfirmationSchema.parse({
      carrierName: "Knight Logistics LLC",
      companyName: "",
      shipmentProNumber: "",
      revisionLabel: "",
      rateMethodLabel: "",
      baseRateLabel: "",
      totalCost: "$1,650.00",
      currency: "",
      paymentTermsLabel: "",
      stops: [],
      status: "Confirmed",
      expiresAt: 0,
      confirmed: true,
    });

    expect(parsed.confirmed).toBe(true);
    expect(parsed.stops).toEqual([]);
  });
});

describe("latestActiveRateConfirmation", () => {
  it("returns null for empty or missing lists", () => {
    expect(latestActiveRateConfirmation(null)).toBeNull();
    expect(latestActiveRateConfirmation(undefined)).toBeNull();
    expect(latestActiveRateConfirmation([])).toBeNull();
  });

  it("picks the highest revision among non-voided rate confirmations", () => {
    const first = makeRateConfirmation({ id: "ratecon_01", revision: 1, status: "Voided" });
    const second = makeRateConfirmation({ id: "ratecon_02", revision: 2, status: "Confirmed" });
    const third = makeRateConfirmation({ id: "ratecon_03", revision: 3, status: "Sent" });

    expect(latestActiveRateConfirmation([second, first, third])?.id).toBe("ratecon_03");
  });

  it("ignores a voided revision even when it is the newest", () => {
    const active = makeRateConfirmation({ id: "ratecon_01", revision: 1, status: "Sent" });
    const voided = makeRateConfirmation({ id: "ratecon_02", revision: 2, status: "Voided" });

    expect(latestActiveRateConfirmation([active, voided])?.id).toBe("ratecon_01");
  });

  it("falls back to the latest voided revision when everything is voided", () => {
    const older = makeRateConfirmation({ id: "ratecon_01", revision: 1, status: "Voided" });
    const newer = makeRateConfirmation({ id: "ratecon_02", revision: 2, status: "Voided" });

    expect(latestActiveRateConfirmation([older, newer])?.id).toBe("ratecon_02");
  });
});
