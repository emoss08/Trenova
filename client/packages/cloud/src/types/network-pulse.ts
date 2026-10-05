import { z } from "zod";

export const networkPulseLaneSchema = z.object({
  from: z.string(),
  to: z.string(),
  // Raw shipment status; the sign-in panel owns the wording.
  status: z.string(),
  count: z.number(),
});

// Instance-wide sign-in figures. sampleSize is what separates "no deliveries in the
// window" from "nothing arrived on time" — the percentage alone cannot say which.
export const networkPulseSchema = z.object({
  loadsInMotion: z.number(),
  onTimePercent: z.number(),
  sampleSize: z.number(),
  windowDays: z.number(),
  lanes: z
    .array(networkPulseLaneSchema)
    .nullish()
    .transform((value) => value ?? []),
});

export type NetworkPulse = z.infer<typeof networkPulseSchema>;
export type NetworkPulseLane = z.infer<typeof networkPulseLaneSchema>;
