import { z } from "zod";

export const releaseInfoSchema = z.object({
  version: z.string(),
  tagName: z.string(),
  publishedAt: z.number(),
  releaseNotes: z.string(),
  downloadUrl: z.string(),
  htmlUrl: z.string(),
  isPrerelease: z.boolean(),
});

export const updateStatusSchema = z.object({
  currentVersion: z.string(),
  latestVersion: z.string().optional(),
  updateAvailable: z.boolean(),
  latestRelease: releaseInfoSchema.optional().nullable(),
  lastChecked: z.number(),
});

export const versionInfoSchema = z.object({
  version: z.string(),
  environment: z.string(),
  buildDate: z.string().optional(),
  gitCommit: z.string().optional(),
});

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

export type ReleaseInfo = z.infer<typeof releaseInfoSchema>;
export type UpdateStatus = z.infer<typeof updateStatusSchema>;
export type VersionInfo = z.infer<typeof versionInfoSchema>;
export type NetworkPulse = z.infer<typeof networkPulseSchema>;
export type NetworkPulseLane = z.infer<typeof networkPulseLaneSchema>;
