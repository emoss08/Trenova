/*
 * Copyright 2023-2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

import * as z from "zod/v4";
import {
  decimalStringSchema,
  nullableBigIntegerSchema,
  optionalStringSchema,
  timestampSchema,
  versionSchema,
} from "./helpers";

export const patternConfigSchema = z.object({
  id: optionalStringSchema,
  version: versionSchema,
  createdAt: timestampSchema,
  updatedAt: timestampSchema,
  organizationId: optionalStringSchema,
  businessUnitId: optionalStringSchema,

  enabled: z.boolean().default(true),
  minFrequency: nullableBigIntegerSchema,
  analysisWindowDays: nullableBigIntegerSchema,
  minConfidenceScore: decimalStringSchema,
  suggestionTtlDays: nullableBigIntegerSchema,
  requireExactMatch: z.boolean().default(false),
  weightRecentShipments: z.boolean().default(true),
});

export type PatternConfigSchema = z.infer<typeof patternConfigSchema>;
