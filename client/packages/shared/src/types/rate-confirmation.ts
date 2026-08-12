import { z } from "zod";
import { nullableArraySchema, nullableStringSchema, optionalStringSchema } from "./helpers";

export const rateConfirmationStatusSchema = z.enum(["Generated", "Sent", "Confirmed", "Voided"]);
export type RateConfirmationStatus = z.infer<typeof rateConfirmationStatusSchema>;

export const rateConfirmationViaSchema = z.enum([
  "Dispatcher",
  "TenderAcceptance",
  "PublicSignature",
]);
export type RateConfirmationVia = z.infer<typeof rateConfirmationViaSchema>;

export const RATE_CONFIRMATION_VIA_LABEL: Record<RateConfirmationVia, string> = {
  Dispatcher: "Confirmed by dispatcher",
  TenderAcceptance: "Signed via tender acceptance",
  PublicSignature: "Signed publicly",
};

// The Go domain serializes Via fields without omitempty, so an unset value
// arrives as "" and must normalize to null rather than fail parsing.
const nullableViaSchema = rateConfirmationViaSchema
  .nullish()
  .or(z.literal("").transform(() => null));

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
  generatedVia: nullableViaSchema,
  sourceTenderOfferId: nullableStringSchema,
  confirmedVia: nullableViaSchema,
  confirmedByTitle: nullableStringSchema,
  version: z.number().optional(),
  createdAt: z.number(),
  updatedAt: z.number(),
});
export type RateConfirmation = z.infer<typeof rateConfirmationSchema>;

export const publicRateConfirmationStopSchema = z.object({
  sequence: z.number().int(),
  type: z.string(),
  name: z.string(),
  address: z.string(),
  window: z.string(),
});
export type PublicRateConfirmationStop = z.infer<typeof publicRateConfirmationStopSchema>;

/**
 * Mirrors PublicRateConfirmationView in the TMS service — the frozen-snapshot
 * view a carrier without a login may see. Labels are pre-formatted strings and
 * may be empty; they are never omitted.
 */
export const publicRateConfirmationSchema = z.object({
  carrierName: z.string(),
  companyName: z.string(),
  shipmentProNumber: z.string(),
  revisionLabel: z.string(),
  rateMethodLabel: z.string(),
  baseRateLabel: z.string(),
  totalCost: z.string(),
  currency: z.string(),
  paymentTermsLabel: z.string(),
  stops: nullableArraySchema(publicRateConfirmationStopSchema),
  status: z.string(),
  expiresAt: z.number(),
  confirmed: z.boolean(),
});
export type PublicRateConfirmation = z.infer<typeof publicRateConfirmationSchema>;

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
