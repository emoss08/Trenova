import { describe, expect, it } from "vitest";
import { workerSchema } from "../worker";

// Shaped after the `WorkerTableRowFields` projection the edit panel resets the
// form with: relations are selected down to the columns the table renders, the
// Go zero value of an unset enum column reaches the wire as "", and a nil
// custom-field map serializes as null.
function workerTableRow() {
  return {
    id: "wrk_01JAV1TCH0000000000000000",
    businessUnitId: "bu_01JAV1TCH0000000000000000",
    organizationId: "org_01JAV1TCH0000000000000000",
    stateId: "us_01JAV1TCH0000000000000000",
    fleetCodeId: "fc_01JAV1TCH0000000000000000",
    managerId: null,
    status: "Active",
    type: "Employee",
    driverType: "OTR",
    leaveType: "",
    profilePicUrl: null,
    firstName: "Ada",
    lastName: "Lovelace",
    wholeName: "Ada Lovelace",
    addressLine1: "1 Analytical Way",
    addressLine2: null,
    city: "Trenton",
    postalCode: "08608",
    email: "ada@example.com",
    phoneNumber: "555-0100",
    emergencyContactName: null,
    emergencyContactPhone: null,
    externalId: null,
    assignmentBlocked: null,
    gender: "Female",
    canBeAssigned: true,
    availableForDispatch: true,
    version: 3,
    createdAt: 1_756_000_000,
    updatedAt: 1_756_000_100,
    customFields: null,
    fleetCode: {
      id: "fc_01JAV1TCH0000000000000000",
      code: "EAST",
      color: "#0ea5e9",
    },
    state: {
      id: "us_01JAV1TCH0000000000000000",
      name: "New Jersey",
      abbreviation: "NJ",
    },
    profile: {
      id: "wp_01JAV1TCH0000000000000000",
      workerId: "wrk_01JAV1TCH0000000000000000",
      businessUnitId: "bu_01JAV1TCH0000000000000000",
      organizationId: "org_01JAV1TCH0000000000000000",
      licenseStateId: "us_01JAV1TCH0000000000000000",
      dob: 631_152_000,
      licenseNumber: "D1234567",
      cdlClass: "A",
      cdlRestrictions: null,
      endorsement: "N",
      hazmatExpiry: null,
      licenseExpiry: 1_800_000_000,
      medicalCardExpiry: null,
      medicalExaminerName: null,
      medicalExaminerNpi: null,
      twicCardNumber: null,
      twicExpiry: null,
      hireDate: 1_700_000_000,
      terminationDate: null,
      physicalDueDate: null,
      mvrDueDate: null,
      complianceStatus: "Compliant",
      trainingHealth: "Current",
      safetyRating: "Good",
      safetyScore: 0,
      nextCredentialExpiry: null,
      nextTrainingDue: null,
      drugAlcoholStatus: "Clear",
      isQualified: true,
      disqualificationReason: null,
      lastComplianceCheck: 0,
      lastMvrCheck: 0,
      lastDrugTest: 0,
      eldExempt: false,
      shortHaulExempt: false,
      version: 1,
      createdAt: 1_756_000_000,
      updatedAt: 1_756_000_100,
      licenseState: {
        id: "us_01JAV1TCH0000000000000000",
        name: "New Jersey",
        abbreviation: "NJ",
      },
    },
  };
}

describe("workerSchema", () => {
  it("accepts a worker exactly as the table query returns it", () => {
    const result = workerSchema.safeParse(workerTableRow());

    expect(result.error?.issues ?? []).toEqual([]);
    expect(result.success).toBe(true);
  });

  it("normalizes an unset leave type to null instead of failing the enum", () => {
    const parsed = workerSchema.parse(workerTableRow());

    expect(parsed.leaveType).toBeNull();
  });

  it("still keeps a real leave type", () => {
    const parsed = workerSchema.parse({ ...workerTableRow(), leaveType: "FMLA" });

    expect(parsed.leaveType).toBe("FMLA");
  });

  it("rejects a leave type outside the enum", () => {
    const result = workerSchema.safeParse({ ...workerTableRow(), leaveType: "Sabbatical" });

    expect(result.success).toBe(false);
  });

  it("accepts a null custom-field map", () => {
    const parsed = workerSchema.parse(workerTableRow());

    expect(parsed.customFields).toBeNull();
  });

  it("accepts relation projections that omit columns the table never selects", () => {
    const parsed = workerSchema.parse(workerTableRow());

    expect(parsed.state?.abbreviation).toBe("NJ");
    expect(parsed.fleetCode?.code).toBe("EAST");
    expect(parsed.profile?.licenseState?.abbreviation).toBe("NJ");
  });

  // REST marshals the whole Go struct, so a relation column the repository never
  // selected arrives as its zero value rather than being absent.
  it("accepts a state relation whose unselected columns arrive as zero values", () => {
    const row = workerTableRow();
    const result = workerSchema.safeParse({
      ...row,
      state: { ...row.state, countryName: "", countryIso3: "" },
      profile: {
        ...row.profile,
        licenseState: { ...row.profile.licenseState, countryName: "", countryIso3: "" },
      },
    });

    expect(result.error?.issues ?? []).toEqual([]);
    expect(result.data?.state?.abbreviation).toBe("NJ");
    expect(result.data?.state?.countryIso3).toBeUndefined();
    expect(result.data?.profile?.licenseState?.countryIso3).toBeUndefined();
  });

  it("accepts a fleet code relation whose unselected columns arrive as zero values", () => {
    const row = workerTableRow();
    const result = workerSchema.safeParse({
      ...row,
      fleetCode: {
        ...row.fleetCode,
        organizationId: null,
        businessUnitId: null,
        status: "",
        managerId: null,
      },
    });

    expect(result.error?.issues ?? []).toEqual([]);
    expect(result.data?.fleetCode?.code).toBe("EAST");
    expect(result.data?.fleetCode?.status).toBeUndefined();
  });
});
