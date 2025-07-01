import { Gender, Status } from "@/types/common";
import {
  ComplianceStatus,
  Endorsement,
  PTOStatus,
  PTOType,
  WorkerType,
} from "@/types/worker";
import * as z from "zod/v4";
import {
  nullableIntegerSchema,
  nullableStringSchema,
  nullableTimestampSchema,
  optionalStringSchema,
  timestampSchema,
  versionSchema,
} from "./helpers";

/* Worker Profile Schema */
const workerProfileSchema = z.object({
  id: optionalStringSchema,
  version: versionSchema,
  createdAt: timestampSchema,
  updatedAt: timestampSchema,
  organizationId: optionalStringSchema,
  businessUnitId: optionalStringSchema,
  workerId: optionalStringSchema,

  // * Core Fields
  dob: nullableIntegerSchema,
  licenseNumber: z.string(),
  endorsement: z.enum(Endorsement),
  hazmatExpiry: nullableIntegerSchema,
  complianceStatus: z.enum(ComplianceStatus),
  isQualified: z.boolean(),
  licenseExpiry: nullableIntegerSchema,
  hireDate: nullableIntegerSchema,
  licenseStateId: z.string(),
  terminationDate: nullableIntegerSchema,
  physicalDueDate: nullableTimestampSchema,
  mvrDueDate: nullableTimestampSchema,
  lastMvrCheck: z.number(),
  lastDrugTest: z.number(),
});

/* Worker PTO Schema */
const workerPTOSchema = z.object({
  id: optionalStringSchema,
  version: versionSchema,
  createdAt: timestampSchema,
  updatedAt: timestampSchema,
  organizationId: optionalStringSchema,
  businessUnitId: optionalStringSchema,
  workerId: optionalStringSchema,

  // * Core Fields
  status: z.enum(PTOStatus),
  type: z.enum(PTOType),
  startDate: z.number().min(1, { error: "Start date is required" }),
  endDate: z.number().min(1, { error: "End date is required" }),
  reason: z.string().optional(),
});

/* Worker Schema */
export const workerSchema = z
  .object({
    id: optionalStringSchema,
    version: versionSchema,
    createdAt: timestampSchema,
    updatedAt: timestampSchema,
    organizationId: optionalStringSchema,
    businessUnitId: optionalStringSchema,

    // * Core Fields
    profilePictureUrl: optionalStringSchema,
    status: z.enum(Status),
    type: z.enum(WorkerType),
    firstName: z.string(),
    lastName: z.string(),
    addressLine1: z.string(),
    addressLine2: optionalStringSchema,
    city: z.string(),
    stateId: z.string(),
    fleetCodeId: nullableStringSchema,
    gender: z.enum(Gender),
    postalCode: z.string(),
    profile: workerProfileSchema.nullable(),
    pto: z.array(workerPTOSchema).optional(),
  })
  .refine(
    (data) => {
      if (!data.profile) {
        return true;
      }

      const hasHazmatEndorsement =
        data.profile.endorsement === Endorsement.Hazmat ||
        data.profile.endorsement === Endorsement.TankerHazmat;

      if (hasHazmatEndorsement && !data.profile.hazmatExpiry) {
        return false;
      }
      return true;
    },
    {
      message: "Hazmat expiry is required when endorsement includes Hazmat",
      path: ["profile", "hazmatExpiry"],
    },
  );

export type WorkerSchema = z.infer<typeof workerSchema>;
