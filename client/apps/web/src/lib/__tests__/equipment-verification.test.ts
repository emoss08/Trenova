import { describe, expect, it } from "vitest";
import {
  canOverrideEquipmentVerification,
  emptyVerifyEquipmentForm,
  equipmentOverrideFormSchema,
  toVerifyCarrierEquipmentInput,
  verifyEquipmentFormSchema,
  vinProblem,
  type VerifyEquipmentFormValues,
} from "../equipment-verification";

const VALID_VIN = "1FUJGLDR5CLBP8834";

function values(overrides: Partial<VerifyEquipmentFormValues>): VerifyEquipmentFormValues {
  return { ...emptyVerifyEquipmentForm, ...overrides };
}

function messages(input: VerifyEquipmentFormValues): string[] {
  const result = verifyEquipmentFormSchema.safeParse(input);
  return result.success ? [] : result.error.issues.map((issue) => issue.message);
}

describe("vinProblem", () => {
  it("accepts a 17-character VIN and ignores case and spacing", () => {
    expect(vinProblem(VALID_VIN)).toBeNull();
    expect(vinProblem(" 1fujgldr5clbp8834 ")).toBeNull();
    expect(vinProblem("1FUJ GLDR5 CLBP8834")).toBeNull();
  });

  it("rejects a VIN that is not exactly 17 characters", () => {
    expect(vinProblem("1FUJGLDR5CLBP883")).toBe("VIN must be exactly 17 characters");
    expect(vinProblem("1FUJGLDR5CLBP88345")).toBe("VIN must be exactly 17 characters");
  });

  it("rejects the letters I, O and Q", () => {
    expect(vinProblem("1FUJGLDR5CLBP883I")).toBe("VIN cannot contain the letters I, O or Q");
    expect(vinProblem("1FUJGLDR5CLBP883O")).toBe("VIN cannot contain the letters I, O or Q");
    expect(vinProblem("1FUJGLDR5CLBP883Q")).toBe("VIN cannot contain the letters I, O or Q");
  });

  it("rejects punctuation and blanks", () => {
    expect(vinProblem("1FUJGLDR5CLBP883-")).toBe("VIN can only contain letters and digits");
    expect(vinProblem("")).toBe("Enter the VIN");
  });
});

describe("verifyEquipmentFormSchema", () => {
  it("validates only the identifier that was chosen", () => {
    expect(messages(values({ identifyBy: "vin", vin: VALID_VIN, unitNumber: "" }))).toEqual([]);
    expect(messages(values({ identifyBy: "unit", vin: "bad", unitNumber: "T-4521" }))).toEqual([]);
  });

  it("reports an invalid VIN on the vin field", () => {
    const result = verifyEquipmentFormSchema.safeParse(
      values({ identifyBy: "vin", vin: "1FUJGLDR5CLBP883Q" }),
    );
    expect(result.success).toBe(false);
    expect(result.error?.issues[0]?.path).toEqual(["vin"]);
  });

  it("requires a plate state alongside a plate number", () => {
    expect(messages(values({ identifyBy: "plate", plateNumber: "ABC-1234" }))).toContain(
      "Choose the state that issued the plate",
    );
    expect(
      messages(
        values({
          identifyBy: "plate",
          plateNumber: "ABC-1234",
          plateStateId: "us_tx",
          plateState: "TX",
        }),
      ),
    ).toEqual([]);
  });

  it("requires a unit number when identifying by unit", () => {
    expect(messages(values({ identifyBy: "unit", unitNumber: "   " }))).toContain(
      "Enter the unit number",
    );
  });
});

describe("toVerifyCarrierEquipmentInput", () => {
  it("sends only the chosen identifier, normalized", () => {
    expect(
      toVerifyCarrierEquipmentInput(
        "ca_1",
        values({
          unitType: "Trailer",
          identifyBy: "vin",
          vin: " 1fujgldr5clbp8834",
          unitNumber: "x",
        }),
      ),
    ).toEqual({ carrierAssignmentId: "ca_1", unitType: "Trailer", vin: VALID_VIN });

    expect(
      toVerifyCarrierEquipmentInput(
        "ca_1",
        values({ identifyBy: "plate", plateNumber: " abc-1234 ", plateState: "tx" }),
      ),
    ).toEqual({
      carrierAssignmentId: "ca_1",
      unitType: "Tractor",
      plateNumber: "ABC-1234",
      plateState: "TX",
    });
  });
});

describe("equipment verification overrides", () => {
  it("can override a mismatch that has not been overridden", () => {
    expect(canOverrideEquipmentVerification({ result: "Mismatch", overriddenAt: null })).toBe(true);
  });

  it("cannot override a match or an already overridden verification", () => {
    expect(canOverrideEquipmentVerification({ result: "Match", overriddenAt: null })).toBe(false);
    expect(
      canOverrideEquipmentVerification({ result: "Mismatch", overriddenAt: 1_800_000_000 }),
    ).toBe(false);
  });

  it("requires a reason", () => {
    const result = equipmentOverrideFormSchema.safeParse({ reason: "   " });
    expect(result.success).toBe(false);
    expect(result.error?.issues[0]?.message).toBe(
      "A reason is required to override a verification",
    );
  });
});
