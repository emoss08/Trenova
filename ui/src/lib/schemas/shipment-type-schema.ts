/*
 * Copyright 2023-2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

import { Status } from "@/types/common";
import * as z from "zod/v4";

export const shipmentTypeSchema = z.object({
  id: z.string().optional(),
  version: z.number().optional(),
  createdAt: z.number().optional(),
  updatedAt: z.number().optional(),

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

export type ShipmentTypeSchema = z.infer<typeof shipmentTypeSchema>;
