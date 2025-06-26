import { Status } from "@/types/common";
import * as z from "zod/v4";

export const serviceTypeSchema = z.object({
  id: z.string().optional(),
  version: z.number().optional(),
  createdAt: z.number().optional(),
  updatedAt: z.number().optional(),

  // * Core Fields
  status: z.enum(Status),
  code: z
    .string({
      error: "Code is required",
    })
    .min(1, "Code is required"),
  description: z.string().optional(),
  color: z.string().optional(),
});

export type ServiceTypeSchema = z.infer<typeof serviceTypeSchema>;
