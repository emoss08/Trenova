import { z } from "zod";
import {
  decimalNumberSchema,
  decimalStringSchema,
  nullableStringSchema,
  optionalStringSchema,
  tenantInfoSchema,
} from "./helpers";
import { carrierRateMethodSchema } from "./shipment";

export const MIN_OFFER_TTL_SECONDS = 300;
export const MAX_OFFER_TTL_SECONDS = 604800;
export const DEFAULT_OFFER_TTL_SECONDS = 7200;

export const tenderModeSchema = z.enum(["Waterfall", "SpotBroadcast", "SpotSequential"]);
export type TenderMode = z.infer<typeof tenderModeSchema>;

export const tenderStatusSchema = z.enum([
  "Active",
  "Accepted",
  "Exhausted",
  "Canceled",
  "NeedsReview",
]);
export type TenderStatus = z.infer<typeof tenderStatusSchema>;

export const tenderOfferStatusSchema = z.enum([
  "Pending",
  "Sent",
  "Accepted",
  "Declined",
  "Expired",
  "Withdrawn",
  "Superseded",
  "Skipped",
  "DeliveryFailed",
]);
export type TenderOfferStatus = z.infer<typeof tenderOfferStatusSchema>;

export const tenderChannelSchema = z.enum(["Email", "EDI"]);
export type TenderChannel = z.infer<typeof tenderChannelSchema>;

export const tenderResponseSourceSchema = z.enum(["Email", "EDI", "Manual"]);
export type TenderResponseSource = z.infer<typeof tenderResponseSourceSchema>;

export const TENDER_MODE_LABEL: Record<TenderMode, string> = {
  Waterfall: "Waterfall",
  SpotBroadcast: "Spot broadcast",
  SpotSequential: "Spot sequential",
};

export const TENDER_CHANNEL_LABEL: Record<TenderChannel, string> = {
  Email: "Email",
  EDI: "EDI",
};

export const tenderCarrierSummarySchema = z.object({
  id: optionalStringSchema,
  name: optionalStringSchema,
  scac: nullableStringSchema,
});

export const tenderOfferSchema = z.object({
  ...tenantInfoSchema.shape,
  tenderId: optionalStringSchema,
  carrierId: optionalStringSchema,
  rank: z.number().int(),
  rateMethod: carrierRateMethodSchema,
  rate: decimalStringSchema,
  offerTtlSeconds: z.number().int(),
  channel: tenderChannelSchema,
  status: tenderOfferStatusSchema,
  recipientEmail: nullableStringSchema,
  sentAt: z.number().nullish(),
  expiresAt: z.number().nullish(),
  respondedAt: z.number().nullish(),
  responseSource: tenderResponseSourceSchema.nullish().or(z.literal("").transform(() => null)),
  declineReason: nullableStringSchema,
  deliveryError: nullableStringSchema,
  lateResponseAction: nullableStringSchema,
  lateResponseAt: z.number().nullish(),
  carrier: tenderCarrierSummarySchema.nullish(),
});
export type TenderOffer = z.infer<typeof tenderOfferSchema>;

export const tenderSchema = z.object({
  ...tenantInfoSchema.shape,
  shipmentId: optionalStringSchema,
  shipmentMoveId: optionalStringSchema,
  routingGuideId: nullableStringSchema,
  mode: tenderModeSchema,
  status: tenderStatusSchema,
  currentRank: z.number().int(),
  cancellationReason: nullableStringSchema,
  acceptedOfferId: nullableStringSchema,
  acceptedAt: z.number().nullish(),
  exhaustedAt: z.number().nullish(),
  canceledAt: z.number().nullish(),
  offers: z.array(tenderOfferSchema).nullish(),
});
export type Tender = z.infer<typeof tenderSchema>;

export const waterfallTenderPayloadSchema = z.object({
  shipmentMoveId: z.string().min(1, { error: "Shipment move is required" }),
  routingGuideId: nullableStringSchema,
});
export type WaterfallTenderPayload = z.infer<typeof waterfallTenderPayloadSchema>;

export const spotTenderModeSchema = z.enum(["SpotBroadcast", "SpotSequential"]);
export type SpotTenderMode = z.infer<typeof spotTenderModeSchema>;

const emailFormatSchema = z.email();

/**
 * The email field only renders while the line's channel is Email, but
 * react-hook-form keeps the unmounted value. Validating it unconditionally
 * would fail a submit on an error the dispatcher can no longer see, so the
 * format check applies only to non-EDI lines and EDI lines have the leftover
 * value stripped on parse.
 */
export const spotTenderLinePayloadSchema = z
  .object({
    carrierId: z.string().min(1, { error: "Carrier is required" }),
    rateMethod: carrierRateMethodSchema,
    rate: decimalNumberSchema("Rate is required", "Rate cannot be negative"),
    offerTtlSeconds: z
      .number({ error: "Offer expiry is required" })
      .int()
      .min(MIN_OFFER_TTL_SECONDS, { error: "Offer expiry must be at least 5 minutes" })
      .max(MAX_OFFER_TTL_SECONDS, { error: "Offer expiry cannot exceed 7 days" }),
    channel: tenderChannelSchema,
    email: z.string().default("").optional(),
  })
  .superRefine((line, ctx) => {
    if (line.channel === "EDI" || !line.email) return;
    if (!emailFormatSchema.safeParse(line.email).success) {
      ctx.addIssue({
        code: "custom",
        path: ["email"],
        message: "Enter a valid email address",
      });
    }
  })
  .transform((line) => (line.channel === "EDI" ? { ...line, email: "" } : line));
export type SpotTenderLinePayload = z.infer<typeof spotTenderLinePayloadSchema>;

export const spotTenderPayloadSchema = z
  .object({
    shipmentMoveId: z.string().min(1, { error: "Shipment move is required" }),
    mode: spotTenderModeSchema,
    lines: z
      .array(spotTenderLinePayloadSchema)
      .min(1, { error: "At least one carrier line is required" }),
    overrideInsuranceWarnings: z.boolean().default(false),
  })
  .superRefine((payload, ctx) => {
    const seen = new Set<string>();
    payload.lines.forEach((line, index) => {
      if (!line.carrierId) return;
      if (seen.has(line.carrierId)) {
        ctx.addIssue({
          code: "custom",
          path: ["lines", index, "carrierId"],
          message: "Carrier is already on this tender",
        });
      }
      seen.add(line.carrierId);
    });
  });
export type SpotTenderPayload = z.infer<typeof spotTenderPayloadSchema>;
export type SpotTenderPayloadInput = z.input<typeof spotTenderPayloadSchema>;

export const emptySpotTenderLine: SpotTenderLinePayload = {
  carrierId: "",
  rateMethod: "Flat",
  rate: 0,
  offerTtlSeconds: DEFAULT_OFFER_TTL_SECONDS,
  channel: "Email",
  email: "",
};

export const cancelTenderPayloadSchema = z.object({
  reason: z
    .string()
    .min(1, { error: "A cancellation reason is required" })
    .max(500, { error: "Reason must be at most 500 characters" }),
});
export type CancelTenderPayload = z.infer<typeof cancelTenderPayloadSchema>;

export const tenderResponseActionSchema = z.enum(["Accept", "Decline"]);
export type TenderResponseAction = z.infer<typeof tenderResponseActionSchema>;

export const recordTenderResponsePayloadSchema = z
  .object({
    action: tenderResponseActionSchema,
    declineReason: z
      .string()
      .max(500, { error: "Reason must be at most 500 characters" })
      .default(""),
  })
  .superRefine((payload, ctx) => {
    if (payload.action === "Decline" && !payload.declineReason.trim()) {
      ctx.addIssue({
        code: "custom",
        path: ["declineReason"],
        message: "A decline reason is required",
      });
    }
  });
export type RecordTenderResponsePayload = z.infer<typeof recordTenderResponsePayloadSchema>;

export const publicTenderOfferSchema = z.object({
  carrierName: z.string(),
  shipmentProNumber: z.string(),
  originSummary: z.string(),
  destinationSummary: z.string(),
  pickupWindow: z.string(),
  deliveryWindow: z.string(),
  equipmentSummary: z.string(),
  weightSummary: z.string(),
  rateAmount: z.string(),
  rateMethodLabel: z.string(),
  expiresAt: z.number(),
  responded: z.boolean(),
});
export type PublicTenderOffer = z.infer<typeof publicTenderOfferSchema>;
