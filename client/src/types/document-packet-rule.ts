import { z } from "zod";
import { tenantInfoSchema } from "./helpers";

export const resourceTypeSchema = z.enum(["Shipment", "Trailer", "Tractor", "Worker"]);

export type ResourceType = z.infer<typeof resourceTypeSchema>;

export const documentPacketRuleSchema = z.object({
  ...tenantInfoSchema.shape,
  resourceType: resourceTypeSchema,
  documentTypeId: z.string().min(1, { message: "Document type is required" }),
  required: z.boolean(),
  allowMultiple: z.boolean(),
  displayOrder: z.number().int().min(0),
  expirationRequired: z.boolean(),
  expirationWarningDays: z.number().int().min(0),
});

export type DocumentPacketRule = z.infer<typeof documentPacketRuleSchema>;
