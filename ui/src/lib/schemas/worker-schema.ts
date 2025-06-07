import { Gender, Status } from "@/types/common";
import {
  ComplianceStatus,
  Endorsement,
  PTOStatus,
  PTOType,
  WorkerType,
} from "@/types/worker";
import { z } from "zod";

/* Worker Profile Schema */
const workerProfileSchema = z.object({
  id: z.string().optional(),
  version: z.number().optional(),
  createdAt: z.number().optional(),
  updatedAt: z.number().optional(),
  workerId: z.string().optional(),
  organizationId: z.string().optional(),
  businessUnitId: z.string().optional(),

  // * Core Fields
  dob: z.number(),
  licenseNumber: z.string(),
  endorsement: z.nativeEnum(Endorsement),
  hazmatExpiry: z.number().optional(),
  complianceStatus: z.nativeEnum(ComplianceStatus),
  isQualified: z.boolean(),
  licenseExpiry: z.number(),
  hireDate: z.number(),
  licenseStateId: z.string(),
  terminationDate: z.number().nullable().optional(),
  physicalDueDate: z.number().nullable().optional(),
  mvrDueDate: z.number().nullable().optional(),
  lastMvrCheck: z.number(),
  lastDrugTest: z.number(),
});

/* Worker PTO Schema */
const workerPTOSchema = z.object({
  id: z.string().optional(),
  version: z.number().optional(),
  createdAt: z.number().optional(),
  updatedAt: z.number().optional(),

  // * Core Fields
  status: z.nativeEnum(PTOStatus),
  type: z.nativeEnum(PTOType),
  startDate: z.number().min(1, "Start date is required"),
  endDate: z.number().min(1, "End date is required"),
  reason: z.string().optional(),
});

/* Worker Schema */
export const workerSchema = z
  .object({
    id: z.string().optional(),
    version: z.number().optional(),
    createdAt: z.number().optional(),
    updatedAt: z.number().optional(),

    // * Core Fields
    profilePictureUrl: z.string().optional(),
    status: z.nativeEnum(Status),
    type: z.nativeEnum(WorkerType),
    firstName: z.string(),
    lastName: z.string(),
    addressLine1: z.string(),
    addressLine2: z.string().optional(),
    city: z.string(),
    stateId: z.string(),
    fleetCodeId: z.string().nullable().optional(),
    gender: z.nativeEnum(Gender),
    postalCode: z.string(),
    profile: workerProfileSchema.nullable().optional(),
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
