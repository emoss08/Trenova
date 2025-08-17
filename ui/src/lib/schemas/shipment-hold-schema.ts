import * as z from "zod/v4";
import { optionalStringSchema, timestampSchema } from "./helpers";
import { userSchema } from "./user-schema";

export const HoldType = z.enum([
  "OperationalHold",
  "ComplianceHold",
  "CustomerHold",
  "FinanceHold",
]);

export const HoldSeverity = z.enum(["Informational", "Advisory", "Blocking"]);

export const HoldSource = z.enum(["User", "Rule", "API", "ELD", "EDI"]);

export const shipmentHoldSchema = z.object({
  id: optionalStringSchema,
  shipmentId: optionalStringSchema,
  businessUnitId: optionalStringSchema,
  organizationId: optionalStringSchema,

  type: HoldType,
  severity: HoldSeverity,
  reasonCode: optionalStringSchema,
  notes: optionalStringSchema,
  source: HoldSource,
  blocksDispatch: z.boolean(),
  blocksDelivery: z.boolean(),
  blocksBilling: z.boolean(),
  visibleToCustomer: z.boolean(),
  metadata: z.record(z.string(), z.any()).nullish(),
  startedAt: timestampSchema,
  releasedAt: timestampSchema.nullish(),
  createdById: optionalStringSchema,
  releasedById: optionalStringSchema,

  // Relationships
  createdBy: userSchema.nullish(),
  releasedBy: userSchema.nullish(),
});

export type ShipmentHoldSchema = z.infer<typeof shipmentHoldSchema>;
