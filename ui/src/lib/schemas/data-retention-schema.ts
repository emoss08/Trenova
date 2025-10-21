/*
 * Copyright 2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

import * as z from "zod/v4";
import {
  optionalStringSchema,
  timestampSchema,
  versionSchema,
} from "./helpers";
export const dataRetentionSchema = z.object({
  id: optionalStringSchema,
  version: versionSchema,
  createdAt: timestampSchema,
  updatedAt: timestampSchema,
  businessUnitId: optionalStringSchema,
  organizationId: optionalStringSchema,

  // * Core Fields
  auditRetentionPeriod: z.number().int().positive().min(30, {
    message: "Audit retention period must be at least 30 days",
  }),
});

export type DataRetentionSchema = z.infer<typeof dataRetentionSchema>;
