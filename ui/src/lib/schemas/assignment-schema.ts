import { AssignmentStatus } from "@/types/assignment";
import { z } from "zod";

export const assignmentSchema = z.object({
  id: z.string().optional(),
  version: z.number().optional(),
  createdAt: z.number().optional(),
  updatedAt: z.number().optional(),

  // * Core Fields
  status: z.nativeEnum(AssignmentStatus),
  shipmentMoveId: z.string().optional(),
  primaryWorkerId: z
    .string({ required_error: "Primary Worker is required" })
    .min(1, "Primary Worker is required"),
  secondaryWorkerId: z.string().nullable().optional(),
  trailerId: z.string().min(1, "Trailer is required"),
  tractorId: z.string().min(1, "Tractor is required"),
});

export type AssignmentSchema = z.infer<typeof assignmentSchema>;
