import { z } from "zod";
import { nullableStringSchema, optionalStringSchema } from "./helpers";

export const rateConfirmationStatusSchema = z.enum(["Generated", "Sent", "Confirmed", "Voided"]);
export type RateConfirmationStatus = z.infer<typeof rateConfirmationStatusSchema>;

export const rateConfirmationSchema = z.object({
  id: z.string(),
  organizationId: optionalStringSchema,
  businessUnitId: optionalStringSchema,
  carrierAssignmentId: z.string(),
  carrierId: z.string(),
  shipmentId: z.string(),
  shipmentMoveId: z.string(),
  revision: z.number().int().min(1),
  status: rateConfirmationStatusSchema,
  documentId: nullableStringSchema,
  sentAt: z.number().nullish(),
  sentToEmails: nullableStringSchema,
  confirmedAt: z.number().nullish(),
  confirmedByName: nullableStringSchema,
  voidedAt: z.number().nullish(),
  voidReason: nullableStringSchema,
  payloadSnapshot: z.record(z.string(), z.unknown()).nullish(),
  generatedById: nullableStringSchema,
  version: z.number().optional(),
  createdAt: z.number(),
  updatedAt: z.number(),
});
export type RateConfirmation = z.infer<typeof rateConfirmationSchema>;

export const rateConfirmationListSchema = z.array(rateConfirmationSchema);

export function latestActiveRateConfirmation(
  rateConfirmations: RateConfirmation[] | null | undefined,
): RateConfirmation | null {
  if (!rateConfirmations || rateConfirmations.length === 0) return null;
  const active = rateConfirmations.filter((rateCon) => rateCon.status !== "Voided");
  const pool = active.length > 0 ? active : rateConfirmations;
  return pool.reduce((latest, candidate) =>
    candidate.revision > latest.revision ? candidate : latest,
  );
}
