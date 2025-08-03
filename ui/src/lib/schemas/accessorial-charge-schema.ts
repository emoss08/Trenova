/*
 * Copyright 2023-2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

import { AccessorialChargeMethod } from "@/types/billing";
import { Status } from "@/types/common";
import * as z from "zod/v4";

export const accessorialChargeSchema = z.object({
  id: z.string().optional(),
  version: z.number().optional(),
  createdAt: z.number().optional(),
  updatedAt: z.number().optional(),

  // * Core Fields
  status: z.enum(Status),
  code: z
    .string()
    .min(3, { error: "Code must be at least 3 characters" })
    .max(10, { error: "Code must be less than 10 characters" }),
  description: z.string().min(1, "Description is required"),
  unit: z.preprocess(
    (val) => {
      if (val === "" || val === null || val === undefined) {
        return undefined;
      }
      const parsed = parseInt(String(val), 10);
      return isNaN(parsed) ? undefined : parsed;
    },
    z
      .number()
      .int({ error: "Unit must be a whole number" })
      .min(1, { error: "Unit is required" }),
  ),
  method: z.enum(AccessorialChargeMethod, {
    error: "Method is required",
  }),
  amount: z.preprocess(
    (val) => {
      if (val === "" || val === null || val === undefined) {
        return undefined;
      }
      const parsed = parseInt(String(val), 10);
      return isNaN(parsed) ? undefined : parsed;
    },
    z.number().min(1, { error: "Amount is required" }),
  ),
});

export type AccessorialChargeSchema = z.infer<typeof accessorialChargeSchema>;
