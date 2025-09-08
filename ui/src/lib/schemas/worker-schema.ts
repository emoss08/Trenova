/*
 * Copyright 2023-2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

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
  nullablePulidSchema,
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
export const workerPTOSchema = z
  .object({
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
    startDate: z
      .number({ error: "Start date is required" })
      .min(1, { error: "Start date is required" }),
    endDate: z
      .number({ error: "End date is required" })
      .min(1, { error: "End date is required" }),
    reason: z.string().optional(),
    approverId: nullablePulidSchema,
    rejectorId: nullablePulidSchema,
    get worker() {
      return workerSchema.nullish();
    },
  })
  .refine(
    (data) => {
      return (
        data.startDate < data.endDate && data.startDate > 0 && data.endDate > 0
      );
    },
    {
      message: "Start date must be before end date",
      path: ["endDate"],
    },
  );

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
    profile: workerProfileSchema.nullish(),
    pto: z.array(workerPTOSchema).nullish(),
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

export const ptoRejectionRequestSchema = z.object({
  ptoId: z.string().min(1, { message: "PTO ID is required" }),
  reason: z.string(),
});

export const ptoFilterSchema = z
  .object({
    type: z.string().optional(),
    startDate: z.number().min(1, { error: "Start date is required" }),
    endDate: z.number().min(1, { error: "End date is required" }),
    workerId: z.string().optional(),
    fleetCodeId: z.string().optional(),
  })
  .refine((data) => data.startDate <= data.endDate, {
    message: "Start date must be before end date",
    path: ["endDate"],
  })
  .refine(
    (data) => {
      const diffInMs = (data.endDate - data.startDate) * 1000;
      const ninetyDaysInMs = 90 * 24 * 60 * 60 * 1000;
      return diffInMs <= ninetyDaysInMs;
    },
    {
      message: "Date range cannot exceed 3 months",
      path: ["endDate"],
    },
  );

export type WorkerSchema = z.infer<typeof workerSchema>;
export type WorkerPTOSchema = z.infer<typeof workerPTOSchema>;

export type PTORejectionRequestSchema = z.infer<
  typeof ptoRejectionRequestSchema
>;
export type PTOFilterSchema = z.infer<typeof ptoFilterSchema>;
