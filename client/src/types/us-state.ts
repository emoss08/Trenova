import { z } from "zod";
import { optionalStringSchema } from "./helpers";
import { createLimitOffsetResponse } from "./server";

export const usStateSchema = z.object({
  id: optionalStringSchema,
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
  countryName: z.string().optional(),
  countryIso3: z
    .string({
      error: "Country ISO 3 is required",
    })
    .min(1, "Country ISO 3 is required"),
});

export type UsState = z.infer<typeof usStateSchema>;

export const usStateSelectOptionResponseSchema =
  createLimitOffsetResponse(usStateSchema);

export type UsStateSelectOptionResponse = z.infer<
  typeof usStateSelectOptionResponseSchema
>;
