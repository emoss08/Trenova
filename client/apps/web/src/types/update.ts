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

export type ReleaseInfo = z.infer<typeof releaseInfoSchema>;
export type UpdateStatus = z.infer<typeof updateStatusSchema>;
