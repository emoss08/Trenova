import { z } from "zod";
import {
  decimalNumberSchema,
  decimalStringSchema,
  nullableStringSchema,
  optionalStringSchema,
  statusSchema,
  tenantInfoSchema,
  versionSchema,
} from "./helpers";
import { carrierRateMethodSchema } from "./shipment";
import {
  DEFAULT_OFFER_TTL_SECONDS,
  MAX_OFFER_TTL_SECONDS,
  MIN_OFFER_TTL_SECONDS,
  tenderChannelSchema,
} from "./tender";

const routingGuideCarrierSummarySchema = z.object({
  id: optionalStringSchema,
  name: optionalStringSchema,
  scac: nullableStringSchema,
});

export const routingGuideEntrySchema = z.object({
  ...tenantInfoSchema.shape,
  routingGuideId: optionalStringSchema,
  carrierId: z.string(),
  rank: z.number().int(),
  rateMethod: carrierRateMethodSchema,
  rate: decimalStringSchema,
  offerTtlSeconds: z.number().int(),
  channel: tenderChannelSchema,
  carrier: routingGuideCarrierSummarySchema.nullish(),
});
export type RoutingGuideEntry = z.infer<typeof routingGuideEntrySchema>;

export const routingGuideSchema = z.object({
  ...tenantInfoSchema.shape,
  name: z.string(),
  description: nullableStringSchema,
  status: statusSchema,
  originLocationId: nullableStringSchema,
  destinationLocationId: nullableStringSchema,
  originCity: nullableStringSchema,
  originState: nullableStringSchema,
  destinationCity: nullableStringSchema,
  destinationState: nullableStringSchema,
  specificity: z.number().int().optional(),
  entries: z.array(routingGuideEntrySchema).nullish(),
});
export type RoutingGuide = z.infer<typeof routingGuideSchema>;

/**
 * Lane match tiers mirror the server's specificity model: an exact location
 * pair beats a city+state pair, which beats a state-only pair. Both sides of
 * the lane must be defined at the same tier.
 */
export const LANE_MATCH_MODES = ["locations", "cityState", "stateOnly"] as const;
export type LaneMatchMode = (typeof LANE_MATCH_MODES)[number];

const routingGuideEntryPayloadSchema = z.object({
  id: optionalStringSchema,
  version: versionSchema,
  carrierId: z.string().min(1, { error: "Carrier is required" }),
  rank: z.number({ error: "Rank is required" }).int().min(1, { error: "Rank must be at least 1" }),
  rateMethod: carrierRateMethodSchema,
  rate: decimalNumberSchema("Rate is required", "Rate cannot be negative"),
  offerTtlSeconds: z
    .number({ error: "Offer expiry is required" })
    .int()
    .min(MIN_OFFER_TTL_SECONDS, { error: "Offer expiry must be at least 5 minutes" })
    .max(MAX_OFFER_TTL_SECONDS, { error: "Offer expiry cannot exceed 7 days" }),
  channel: tenderChannelSchema,
});
export type RoutingGuideEntryPayload = z.infer<typeof routingGuideEntryPayloadSchema>;

const laneSideTier = (
  locationId: string | null | undefined,
  city: string | null | undefined,
  state: string | null | undefined,
): number => {
  if (locationId) return 3;
  if (city && state) return 2;
  if (state) return 1;
  return 0;
};

export const routingGuidePayloadSchema = z
  .object({
    id: optionalStringSchema,
    version: versionSchema,
    name: z
      .string()
      .min(1, { error: "Name is required" })
      .max(255, { error: "Name must be at most 255 characters" }),
    description: z.string().nullish(),
    status: statusSchema,
    originLocationId: nullableStringSchema,
    destinationLocationId: nullableStringSchema,
    originCity: z.string().nullish(),
    originState: z.string().nullish(),
    destinationCity: z.string().nullish(),
    destinationState: z.string().nullish(),
    entries: z
      .array(routingGuideEntryPayloadSchema)
      .min(1, { error: "At least one carrier entry is required" }),
  })
  .superRefine((payload, ctx) => {
    const origin = laneSideTier(payload.originLocationId, payload.originCity, payload.originState);
    const destination = laneSideTier(
      payload.destinationLocationId,
      payload.destinationCity,
      payload.destinationState,
    );

    if (origin === 0) {
      ctx.addIssue({
        code: "custom",
        path: ["originLocationId"],
        message: "Origin requires a location, a city and state, or a state",
      });
    }
    if (destination === 0) {
      ctx.addIssue({
        code: "custom",
        path: ["destinationLocationId"],
        message: "Destination requires a location, a city and state, or a state",
      });
    }
    if (origin !== 0 && destination !== 0 && origin !== destination) {
      ctx.addIssue({
        code: "custom",
        path: ["originLocationId"],
        message: "Origin and destination must be defined at the same level of detail",
      });
    }

    if (payload.originLocationId && (payload.originCity || payload.originState)) {
      ctx.addIssue({
        code: "custom",
        path: ["originCity"],
        message: "An origin location cannot be combined with city or state criteria",
      });
    }
    if (payload.destinationLocationId && (payload.destinationCity || payload.destinationState)) {
      ctx.addIssue({
        code: "custom",
        path: ["destinationCity"],
        message: "A destination location cannot be combined with city or state criteria",
      });
    }

    const ranks = new Set<number>();
    payload.entries.forEach((entry, index) => {
      if (ranks.has(entry.rank)) {
        ctx.addIssue({
          code: "custom",
          path: ["entries", index, "rank"],
          message: "Rank must be unique within the guide",
        });
      }
      ranks.add(entry.rank);
    });
  });
export type RoutingGuidePayload = z.infer<typeof routingGuidePayloadSchema>;
export type RoutingGuidePayloadInput = z.input<typeof routingGuidePayloadSchema>;

export const ROUTING_GUIDE_TIER_LABEL: Record<number, string> = {
  3: "Exact locations",
  2: "City to city",
  1: "State to state",
};

type LaneLike = {
  originLocationId?: string | null;
  destinationLocationId?: string | null;
  originCity?: string | null;
  originState?: string | null;
  destinationCity?: string | null;
  destinationState?: string | null;
};

const laneSideLabel = (
  locationId: string | null | undefined,
  city: string | null | undefined,
  state: string | null | undefined,
): string => {
  if (locationId) return "Exact location";
  if (city && state) return `${city}, ${state}`;
  if (state) return state;
  return "—";
};

export function formatRoutingGuideLane(guide: LaneLike): string {
  const origin = laneSideLabel(guide.originLocationId, guide.originCity, guide.originState);
  const destination = laneSideLabel(
    guide.destinationLocationId,
    guide.destinationCity,
    guide.destinationState,
  );
  return `${origin} → ${destination}`;
}

export function deriveLaneMatchMode(guide: {
  originLocationId?: string | null;
  destinationLocationId?: string | null;
  originCity?: string | null;
  destinationCity?: string | null;
}): LaneMatchMode {
  if (guide.originLocationId || guide.destinationLocationId) return "locations";
  if (guide.originCity || guide.destinationCity) return "cityState";
  return "stateOnly";
}

export const emptyRoutingGuideEntry = (rank: number): RoutingGuideEntryPayload => ({
  id: undefined,
  version: undefined,
  carrierId: "",
  rank,
  rateMethod: "Flat",
  rate: 0,
  offerTtlSeconds: DEFAULT_OFFER_TTL_SECONDS,
  channel: "Email",
});

export const emptyRoutingGuidePayload: RoutingGuidePayloadInput = {
  name: "",
  description: "",
  status: "Active",
  originLocationId: null,
  destinationLocationId: null,
  originCity: "",
  originState: "",
  destinationCity: "",
  destinationState: "",
  entries: [],
};
