import { AssignmentStatus } from "@/types/assignment";
import * as z from "zod/v4";
import {
  nullableStringSchema,
  optionalStringSchema,
  timestampSchema,
  versionSchema,
} from "./helpers";
import { tractorSchema } from "./tractor-schema";
import { trailerSchema } from "./trailer-schema";
import { workerSchema } from "./worker-schema";

export const assignmentSchema = z.object({
  id: optionalStringSchema,
  version: versionSchema,
  createdAt: timestampSchema,
  updatedAt: timestampSchema,

  // * Core Fields
  status: z.enum(AssignmentStatus),
  shipmentMoveId: optionalStringSchema,
  primaryWorkerId: z
    .string({ error: "Primary Worker is required" })
    .min(1, { error: "Primary Worker is required" }),
  secondaryWorkerId: nullableStringSchema,
  trailerId: z.string().min(1, { error: "Trailer is required" }),
  tractorId: z.string().min(1, { error: "Tractor is required" }),

  // * Relationships
  tractor: tractorSchema.nullish(),
  trailer: trailerSchema.nullish(),
  primaryWorker: workerSchema.nullish(),
  secondaryWorker: workerSchema.nullish(),
});

export type AssignmentSchema = z.infer<typeof assignmentSchema>;
