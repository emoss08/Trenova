import { OrganizationType } from "@/types/organization";
import { z } from "zod";
import { usStateSchema } from "./us-state-schema";

const organizationMetadataSchema = z.object({
  objectID: z.string().optional(),
});

export const organizationSchema = z.object({
  id: z.string().optional(),
  bucketName: z.string().optional(),
  businessUnitId: z.string().optional(),
  version: z.number().optional(),
  createdAt: z.number().optional(),
  updatedAt: z.number().optional(),

  // * Core Fields
  name: z.string().min(1, "Name is required"),
  scacCode: z.string().min(1, "SCAC code is required"),
  dotNumber: z.string().min(1, "DOT number is required"),
  logoUrl: z.string().optional(),
  orgType: z.nativeEnum(OrganizationType),
  addressLine1: z.string().min(1, "Address line 1 is required"),
  addressLine2: z.string().optional(),
  city: z.string().min(1, "City is required"),
  stateId: z.string().min(1, "State is required"),
  postalCode: z.string().min(1, "Postal code is required"),
  timezone: z.string().min(1, "Timezone is required"),
  taxId: z.string().optional(),
  state: usStateSchema.optional(),
  metadata: organizationMetadataSchema.optional(),
});

export type OrganizationSchema = z.infer<typeof organizationSchema>;
