/*
 * Copyright 2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

import { Status } from "@/types/common";
import * as z from "zod";
import {
    optionalStringSchema,
    timestampSchema,
    versionSchema,
} from "./helpers";

export const serviceTypeSchema = z.object({
  id: optionalStringSchema,
  version: versionSchema,
  createdAt: timestampSchema,
  updatedAt: timestampSchema,
  organizationId: optionalStringSchema,
  businessUnitId: optionalStringSchema,

  // * Core Fields
  status: z.enum(Status),
  code: z
    .string({
      error: "Code is required",
    })
    .min(1, "Code is required"),
  description: z.string().optional(),
  color: z.string().optional(),
});

export type ServiceTypeSchema = z.infer<typeof serviceTypeSchema>;
