import { describe, expect, it } from "vitest";
import { carrierSchema } from "@trenova/shared/types/carrier";

/**
 * `Carrier` in carrier.graphqls declares `contacts: [CarrierContact!]` and
 * `insurancePolicies: [CarrierInsurancePolicy!]` as nullable lists, and the server sends
 * a carrier saved with none of either as `null`. Opening that carrier to edit it must not
 * fail validation, and the form's contact and policy lists must start empty.
 */
const savedCarrier = {
  id: "car_1",
  status: "Active",
  code: "ACME",
  name: "Acme Freight",
  dbaName: null,
  carrierType: "Common",
  dotNumber: "1234567",
  mcNumber: "765432",
  scac: "ACMF",
  complianceStatus: "Pending",
  safetyRating: "NotRated",
  qualifiedAt: null,
  disqualifiedReason: null,
  taxId: null,
  taxIdType: null,
  w9OnFile: false,
  is1099Eligible: false,
  paymentMethod: "Check",
  paymentTermDays: 30,
  remitToName: null,
  remitAddressLine1: null,
  remitAddressLine2: null,
  remitCity: null,
  remitStateId: null,
  remitPostalCode: null,
  addressLine1: null,
  addressLine2: null,
  city: null,
  stateId: null,
  postalCode: null,
  phone: null,
  email: null,
  externalId: null,
  notes: null,
};

describe("carrierSchema", () => {
  it("reads a carrier saved with no contacts and no insurance policies", () => {
    const result = carrierSchema.safeParse({
      ...savedCarrier,
      contacts: null,
      insurancePolicies: null,
    });

    expect(result.success).toBe(true);
    expect(result.data?.contacts).toEqual([]);
    expect(result.data?.insurancePolicies).toEqual([]);
  });

  it("reads a carrier whose lists the server leaves out the same as empty ones", () => {
    const result = carrierSchema.safeParse(savedCarrier);

    expect(result.success).toBe(true);
    expect(result.data?.contacts).toEqual([]);
    expect(result.data?.insurancePolicies).toEqual([]);
  });

  it("keeps the contacts a carrier has", () => {
    const result = carrierSchema.safeParse({
      ...savedCarrier,
      contacts: [
        {
          name: "Dana Ortiz",
          title: "Dispatch",
          email: "dispatch@acme.example",
          phone: null,
          isPrimary: true,
          receivesRateConfirmations: true,
        },
        {
          name: "Lee Park",
          title: null,
          email: null,
          phone: "555-0100",
          isPrimary: false,
          receivesRateConfirmations: false,
        },
      ],
      insurancePolicies: null,
    });

    expect(result.success).toBe(true);
    expect(result.data?.contacts.map((contact) => contact.name)).toEqual([
      "Dana Ortiz",
      "Lee Park",
    ]);
    expect(result.data?.insurancePolicies).toEqual([]);
  });
});
