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

export type ReleaseInfo = z.infer<typeof releaseInfoSchema>;
export type UpdateStatus = z.infer<typeof updateStatusSchema>;
export type VersionInfo = z.infer<typeof versionInfoSchema>;
