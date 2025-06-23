import { z } from "zod";

export const patternConfigSchema = z.object({
  id: z.string(),
  version: z.number(),
  createdAt: z.number(),
  updatedAt: z.number().optional().nullable(),

  enabled: z.boolean(),
  minFrequency: z.preprocess((val) => {
    if (val === "" || val === null || val === undefined) {
      return undefined;
    }
    const parsed = parseInt(String(val));
    return isNaN(parsed) ? undefined : parsed;
  }, z.number().default(3)),
  analysisWindowDays: z.preprocess((val) => {
    if (val === "" || val === null || val === undefined) {
      return undefined;
    }
    const parsed = parseInt(String(val));
    return isNaN(parsed) ? undefined : parsed;
  }, z.number().default(90)),
  minConfidenceScore: z.preprocess((val) => {
    if (val === "" || val === null || val === undefined) {
      return undefined;
    }
    const parsed = parseFloat(String(val));
    return isNaN(parsed) ? undefined : parsed;
  }, z.number().default(0.7)),
  suggestionTtlDays: z.preprocess((val) => {
    if (val === "" || val === null || val === undefined) {
      return undefined;
    }
    const parsed = parseInt(String(val));
    return isNaN(parsed) ? undefined : parsed;
  }, z.number().default(30)),
  requireExactMatch: z.boolean().default(false),
  weightRecentShipments: z.boolean().default(true),
});

export type PatternConfigSchema = z.infer<typeof patternConfigSchema>;
