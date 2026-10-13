import { z } from "zod";
import { nullableArraySchema } from "./helpers";

/** Mirrors PublicTrackingStop in the TMS service. */
export const publicTrackingStopSchema = z.object({
  id: z.string(),
  sequence: z.number().int(),
  type: z.string(),
  locationName: z.string(),
  city: z.string(),
  stateCode: z.string(),
  windowStart: z.number(),
  windowEnd: z.number().nullable(),
  arrivedAt: z.number().nullable(),
  departedAt: z.number().nullable(),
});
export type PublicTrackingStop = z.infer<typeof publicTrackingStopSchema>;

/**
 * Mirrors PublicTrackingView in the TMS service: what a carrier without a login
 * sees of the load it is hauling. Nothing about the customer or the rate.
 */
export const publicTrackingViewSchema = z.object({
  companyName: z.string(),
  carrierName: z.string(),
  shipmentProNumber: z.string(),
  closed: z.boolean(),
  expiresAt: z.number(),
  lastCheckCallAt: z.number().nullable(),
  stops: nullableArraySchema(publicTrackingStopSchema),
});
export type PublicTrackingView = z.infer<typeof publicTrackingViewSchema>;

export const publicCheckCallKindSchema = z.enum(["Update", "Arrival", "Departure"]);
export type PublicCheckCallKind = z.infer<typeof publicCheckCallKindSchema>;

export type PublicCheckCallPayload = {
  kind: PublicCheckCallKind;
  stopId?: string;
  reportedAt?: number;
  latitude?: number;
  longitude?: number;
  city?: string;
  stateCode?: string;
  delayed?: boolean;
  delayReason?: string;
  estimatedArrival?: number;
  note?: string;
  contactName?: string;
  clientEventId: string;
};

export const publicCheckCallResultSchema = z.object({
  outcome: z.string().optional(),
  reason: z.string().optional(),
  replayed: z.boolean(),
});
export type PublicCheckCallResult = z.infer<typeof publicCheckCallResultSchema>;
