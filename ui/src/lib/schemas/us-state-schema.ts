/*
 * Copyright 2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

import * as z from "zod";

export const usStateSchema = z.object({
  id: z.string().optional(),
  name: z
    .string({
      error: "Name is required",
    })
    .min(1, "Name is required"),
  abbreviation: z
    .string({
      error: "Abbreviation is required",
    })
    .min(1, "Abbreviation is required"),
  countryName: z
    .string({
      error: "Country name is required",
    })
    .min(1, "Country name is required"),
  countryIso3: z
    .string({
      error: "Country ISO 3 is required",
    })
    .min(1, "Country ISO 3 is required"),
});

export type UsStateSchema = z.infer<typeof usStateSchema>;
