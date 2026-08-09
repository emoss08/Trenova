import { describe, expect, it } from "vitest";
import {
  latestActiveRateConfirmation,
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
