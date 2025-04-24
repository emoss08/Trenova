import { Status } from "@/types/common";
import {
  HazardousClassChoiceProps,
  PackingGroupChoiceProps,
} from "@/types/hazardous-material";
import { z } from "zod";

export const hazardousMaterialSchema = z.object({
  id: z.string().optional(),
  version: z.number().optional(),
  createdAt: z.number().optional(),
  updatedAt: z.number().optional(),

  // * Core Fields
  status: z.nativeEnum(Status),
  code: z.string().min(1, "Code is required"),
  name: z.string().min(1, "Name is required"),
  description: z.string().min(1, "Description is required"),
  class: z.nativeEnum(HazardousClassChoiceProps),
  unNumber: z
    .string()
    .max(4, "UN Number must be 4 characters or less")
    .optional(),
  casNumber: z
    .string()
    .max(10, "CAS Number must be 10 characters or less")
    .optional(),
  packingGroup: z.nativeEnum(PackingGroupChoiceProps),
  properShippingName: z.string().optional(),
  handlingInstructions: z.string().optional(),
  emergencyContact: z.string().optional(),
  emergencyContactPhoneNumber: z.string().optional(),
  placardRequired: z.boolean().optional(),
  isReportableQuantity: z.boolean().optional(),
});

export type HazardousMaterialSchema = z.infer<typeof hazardousMaterialSchema>;
