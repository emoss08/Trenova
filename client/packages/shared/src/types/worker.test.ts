import { describe, expect, it } from "vitest";
import { workerFormSchema, workerSchema, workerWithoutPersonalDetailsSchema } from "./worker";

function worker(profile: Record<string, unknown>) {
  return {
    stateId: "st_1",
    status: "Active",
    type: "Employee",
    driverType: "OTR",
    firstName: "Ana",
    lastName: "Diaz",
    addressLine1: "1 Main St",
    city: "Dallas",
    postalCode: "75201",
    phoneNumber: "214-555-0100",
    gender: "Female",
    profile: {
      cdlClass: "A",
      endorsement: "O",
      licenseExpiry: 1_893_456_000,
      hireDate: 1_609_459_200,
      complianceStatus: "Pending",
      ...profile,
    },
  };
}

function issuePaths(result: { success: boolean; error?: { issues: { path: PropertyKey[] }[] } }) {
  return (result.error?.issues ?? []).map((issue) => issue.path.join("."));
}

describe("worker personal details", () => {
  it("requires a date of birth and license number of whoever can see them", () => {
    const result = workerSchema.safeParse(worker({ dob: null, licenseNumber: null }));

    expect(result.success).toBe(false);
    expect(issuePaths(result)).toEqual(["profile.dob", "profile.licenseNumber"]);
    expect(workerSchema.safeParse(worker({ dob: 0, licenseNumber: "  " })).success).toBe(false);
    expect(
      workerSchema.safeParse(worker({ dob: 946_684_800, licenseNumber: "D1234567" })).success,
    ).toBe(true);
  });

  it("accepts the withheld values a role without personal details receives", () => {
    expect(
      workerWithoutPersonalDetailsSchema.safeParse(
        worker({ dob: null, licenseNumber: null, twicCardNumber: null, lastDrugTest: null }),
      ).success,
    ).toBe(true);
    expect(
      workerWithoutPersonalDetailsSchema.safeParse(worker({ dob: 0, licenseNumber: "" })).success,
    ).toBe(true);
  });

  it("picks the schema by whether the viewer can see personal details", () => {
    expect(workerFormSchema(true)).toBe(workerSchema);
    expect(workerFormSchema(false)).toBe(workerWithoutPersonalDetailsSchema);
  });
});
