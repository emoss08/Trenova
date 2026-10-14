import { translate } from "@trenova/shared/i18n/runtime";
import { z } from "zod";
import {
  decimalNumberSchema,
  decimalStringSchema,
  nullableStringSchema,
  optionalStringSchema,
  tenantInfoSchema,
} from "./helpers";
import { carrierRateMethodSchema } from "./shipment";
import { defineLabels } from "@trenova/shared/i18n/labels";

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
  "Countered",
  "CounterRejected",
  "Revoked",
]);
export type TenderOfferStatus = z.infer<typeof tenderOfferStatusSchema>;

export const tenderChannelSchema = z.enum(["Email", "EDI"]);
export type TenderChannel = z.infer<typeof tenderChannelSchema>;

export const tenderResponseSourceSchema = z.enum(["Email", "EDI", "Manual"]);
export type TenderResponseSource = z.infer<typeof tenderResponseSourceSchema>;

export const TENDER_MODE_LABEL: Record<TenderMode, string> = defineLabels({
  Waterfall: "Waterfall",
  SpotBroadcast: "Spot broadcast",
  SpotSequential: "Spot sequential",
});

export const TENDER_CHANNEL_LABEL: Record<TenderChannel, string> = defineLabels({
  Email: "Email",
  EDI: "EDI",
});

export const TENDER_RESPONSE_SOURCE_LABEL: Record<TenderResponseSource, string> = {
  Email: "via email link",
  EDI: "via EDI 990",
  Manual: "recorded by dispatcher",
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
  counterRate: decimalStringSchema,
  counterNote: nullableStringSchema,
  counteredAt: z.number().nullish(),
  counterMarginPercent: decimalStringSchema,
  counterNeedsApproval: z.boolean().default(false),
  counterDecidedAt: z.number().nullish(),
  counterDecidedById: nullableStringSchema,
  revokedAt: z.number().nullish(),
  revokeReason: nullableStringSchema,
  revokedById: nullableStringSchema,
  reofferedFromId: nullableStringSchema,
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
  bounceCount: z.number().int().default(0),
  lastBouncedAt: z.number().nullish(),
  escalatedFromId: nullableStringSchema,
  escalationRound: z.number().int().default(0),
  offers: z.array(tenderOfferSchema).nullish(),
});
export type Tender = z.infer<typeof tenderSchema>;

export const guideEntryScreeningSchema = z.object({
  carrierName: z.string(),
  rank: z.number().int(),
  reasons: z
    .array(z.string())
    .nullish()
    .transform((value) => value ?? []),
});
export type GuideEntryScreening = z.infer<typeof guideEntryScreeningSchema>;

export const guideScreeningSummarySchema = z.object({
  skipped: z
    .array(guideEntryScreeningSchema)
    .nullish()
    .transform((value) => value ?? []),
  warned: z
    .array(guideEntryScreeningSchema)
    .nullish()
    .transform((value) => value ?? []),
});
export type GuideScreeningSummary = z.infer<typeof guideScreeningSummarySchema>;

/**
 * The waterfall endpoint embeds the tender, so the response body is a superset of
 * a plain tender: the screening summary is absent when every routing-guide entry
 * was offered.
 */
export const waterfallTenderResultSchema = tenderSchema.extend({
  screening: guideScreeningSummarySchema.nullish().transform((value) => value ?? null),
});
export type WaterfallTenderResult = z.infer<typeof waterfallTenderResultSchema>;

export const waterfallTenderPayloadSchema = z.object({
  shipmentMoveId: z.string().min(1, { error: () => translate("Shipment move is required") }),
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
    carrierId: z.string().min(1, { error: () => translate("Carrier is required") }),
    rateMethod: carrierRateMethodSchema,
    rate: decimalNumberSchema("Rate is required", "Rate cannot be negative"),
    offerTtlSeconds: z
      .number({ error: () => translate("Offer expiry is required") })
      .int()
      .min(MIN_OFFER_TTL_SECONDS, {
        error: () => translate("Offer expiry must be at least 5 minutes"),
      })
      .max(MAX_OFFER_TTL_SECONDS, { error: () => translate("Offer expiry cannot exceed 7 days") }),
    channel: tenderChannelSchema,
    email: z.string().default("").optional(),
  })
  .superRefine((line, ctx) => {
    if (line.channel === "EDI" || !line.email) return;
    if (!emailFormatSchema.safeParse(line.email).success) {
      ctx.addIssue({
        code: "custom",
        path: ["email"],
        message: translate("Enter a valid email address"),
      });
    }
  })
  .transform((line) => (line.channel === "EDI" ? { ...line, email: "" } : line));
export type SpotTenderLinePayload = z.infer<typeof spotTenderLinePayloadSchema>;

export const spotTenderPayloadSchema = z
  .object({
    shipmentMoveId: z.string().min(1, { error: () => translate("Shipment move is required") }),
    mode: spotTenderModeSchema,
    lines: z
      .array(spotTenderLinePayloadSchema)
      .min(1, { error: () => translate("At least one carrier line is required") }),
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
          message: translate("Carrier is already on this tender"),
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
    .min(1, { error: () => translate("A cancellation reason is required") })
    .max(500, { error: () => translate("Reason must be at most 500 characters") }),
});
export type CancelTenderPayload = z.infer<typeof cancelTenderPayloadSchema>;

export const MAX_COUNTER_NOTE_LENGTH = 500;
export const MAX_COUNTER_RATE = 10_000_000;
const MAX_REASON_LENGTH = 500;

const reasonSchema = z
  .string()
  .max(MAX_REASON_LENGTH, { error: () => translate("Reason must be at most 500 characters") });

const counterNoteSchema = z
  .string()
  .max(MAX_COUNTER_NOTE_LENGTH, {
    error: () => translate("Note must be at most 500 characters"),
  })
  .default("");

/**
 * The counter rate is entered as text so the field can be empty while the
 * dispatcher types; the server takes the same bounds (positive, below ten
 * million).
 */
const counterRateSchema = z
  .string()
  .trim()
  .default("")
  .superRefine((value, ctx) => {
    if (value === "") {
      ctx.addIssue({ code: "custom", message: translate("Counter rate is required") });
      return;
    }
    const rate = Number(value);
    if (!Number.isFinite(rate) || rate <= 0) {
      ctx.addIssue({
        code: "custom",
        message: translate("Counter rate must be a positive amount"),
      });
      return;
    }
    if (rate >= MAX_COUNTER_RATE) {
      ctx.addIssue({ code: "custom", message: translate("Counter rate is too large") });
    }
  });

export const tenderResponseActionSchema = z.enum(["Accept", "Decline", "Counter"]);
export type TenderResponseAction = z.infer<typeof tenderResponseActionSchema>;

export const recordTenderResponsePayloadSchema = z
  .object({
    action: tenderResponseActionSchema,
    declineReason: reasonSchema.default(""),
    counterRate: z.string().default(""),
    counterNote: counterNoteSchema,
  })
  .superRefine((payload, ctx) => {
    if (payload.action === "Decline" && !payload.declineReason.trim()) {
      ctx.addIssue({
        code: "custom",
        path: ["declineReason"],
        message: translate("A decline reason is required"),
      });
    }
    if (payload.action !== "Counter") return;
    const rate = counterRateSchema.safeParse(payload.counterRate);
    if (!rate.success) {
      rate.error.issues.forEach((issue) =>
        ctx.addIssue({ code: "custom", path: ["counterRate"], message: issue.message }),
      );
    }
  });
export type RecordTenderResponsePayload = z.infer<typeof recordTenderResponsePayloadSchema>;
export type RecordTenderResponsePayloadInput = z.input<typeof recordTenderResponsePayloadSchema>;

export const bounceTenderPayloadSchema = z.object({
  reason: reasonSchema.trim().min(1, { error: () => translate("Say why the carrier backed out") }),
});
export type BounceTenderPayload = z.infer<typeof bounceTenderPayloadSchema>;

export const publicTenderCounterPayloadSchema = z.object({
  rate: counterRateSchema,
  note: counterNoteSchema,
});
export type PublicTenderCounterPayload = z.infer<typeof publicTenderCounterPayloadSchema>;

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
  rateValue: z.string().default(""),
  expiresAt: z.number(),
  responded: z.boolean(),
  counterAllowed: z.boolean().default(false),
});
export type PublicTenderOffer = z.infer<typeof publicTenderOfferSchema>;
