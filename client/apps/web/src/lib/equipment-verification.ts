import type {
  CarrierEquipmentVerificationResult,
  CarrierIntelUnitType,
  VerifyCarrierEquipmentInput,
} from "@trenova/graphql/generated/graphql";
import { z } from "zod";

export const VIN_LENGTH = 17;
export const PLATE_NUMBER_MAX_LENGTH = 15;
export const UNIT_NUMBER_MAX_LENGTH = 50;
export const EQUIPMENT_OVERRIDE_REASON_MAX_LENGTH = 2000;

const VIN_PATTERN = /^[A-HJ-NPR-Z0-9]{17}$/;
const PLATE_NUMBER_PATTERN = /^[A-Z0-9 -]{1,15}$/;
const PLATE_STATE_PATTERN = /^[A-Z]{2}$/;

export const EQUIPMENT_UNIT_TYPES = [
  "Tractor",
  "Trailer",
  "Straight",
] as const satisfies readonly CarrierIntelUnitType[];

export const EQUIPMENT_IDENTIFIERS = ["vin", "plate", "unit"] as const;
export type EquipmentIdentifier = (typeof EQUIPMENT_IDENTIFIERS)[number];

export function normalizeVin(value: string): string {
  return value.replace(/\s+/g, "").toUpperCase();
}

export function normalizePlate(value: string): string {
  return value.trim().toUpperCase();
}

export function vinProblem(value: string): string | null {
  const vin = normalizeVin(value);
  if (vin === "") {
    return "Enter the VIN";
  }
  if (vin.length !== VIN_LENGTH) {
    return "VIN must be exactly 17 characters";
  }
  if (/[IOQ]/.test(vin)) {
    return "VIN cannot contain the letters I, O or Q";
  }
  if (!VIN_PATTERN.test(vin)) {
    return "VIN can only contain letters and digits";
  }
  return null;
}

export const verifyEquipmentFormSchema = z
  .object({
    unitType: z.enum(EQUIPMENT_UNIT_TYPES, { error: "Choose the unit type" }),
    identifyBy: z.enum(EQUIPMENT_IDENTIFIERS),
    vin: z.string(),
    plateNumber: z.string(),
    plateStateId: z.string().nullable(),
    plateState: z.string(),
    unitNumber: z.string(),
  })
  .superRefine((values, ctx) => {
    switch (values.identifyBy) {
      case "vin": {
        const problem = vinProblem(values.vin);
        if (problem) {
          ctx.addIssue({ code: "custom", path: ["vin"], message: problem });
        }
        break;
      }
      case "plate": {
        const plate = normalizePlate(values.plateNumber);
        if (plate === "") {
          ctx.addIssue({
            code: "custom",
            path: ["plateNumber"],
            message: "Enter the plate number",
          });
        } else if (!PLATE_NUMBER_PATTERN.test(plate)) {
          ctx.addIssue({
            code: "custom",
            path: ["plateNumber"],
            message: "Plate number must be 1-15 letters, digits, spaces or dashes",
          });
        }
        if (!PLATE_STATE_PATTERN.test(values.plateState.trim().toUpperCase())) {
          ctx.addIssue({
            code: "custom",
            path: ["plateStateId"],
            message: "Choose the state that issued the plate",
          });
        }
        break;
      }
      case "unit": {
        const unit = values.unitNumber.trim();
        if (unit === "") {
          ctx.addIssue({ code: "custom", path: ["unitNumber"], message: "Enter the unit number" });
        } else if (unit.length > UNIT_NUMBER_MAX_LENGTH) {
          ctx.addIssue({
            code: "custom",
            path: ["unitNumber"],
            message: "Unit number cannot exceed 50 characters",
          });
        }
        break;
      }
    }
  });

export type VerifyEquipmentFormValues = z.infer<typeof verifyEquipmentFormSchema>;

export const emptyVerifyEquipmentForm: VerifyEquipmentFormValues = {
  unitType: "Tractor",
  identifyBy: "vin",
  vin: "",
  plateNumber: "",
  plateStateId: null,
  plateState: "",
  unitNumber: "",
};

export function toVerifyCarrierEquipmentInput(
  carrierAssignmentId: string,
  values: VerifyEquipmentFormValues,
): VerifyCarrierEquipmentInput {
  const base: VerifyCarrierEquipmentInput = { carrierAssignmentId, unitType: values.unitType };
  switch (values.identifyBy) {
    case "vin":
      return { ...base, vin: normalizeVin(values.vin) };
    case "plate":
      return {
        ...base,
        plateNumber: normalizePlate(values.plateNumber),
        plateState: values.plateState.trim().toUpperCase(),
      };
    case "unit":
      return { ...base, unitNumber: values.unitNumber.trim() };
  }
}

export const equipmentOverrideFormSchema = z.object({
  reason: z
    .string()
    .trim()
    .min(1, "A reason is required to override a verification")
    .max(EQUIPMENT_OVERRIDE_REASON_MAX_LENGTH, "Reason cannot exceed 2000 characters"),
});

export type EquipmentOverrideFormValues = z.infer<typeof equipmentOverrideFormSchema>;

export function canOverrideEquipmentVerification(verification: {
  result: CarrierEquipmentVerificationResult;
  overriddenAt: number | null;
}): boolean {
  return verification.result !== "Match" && verification.overriddenAt === null;
}
