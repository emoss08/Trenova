import { z } from "zod";
import { optionalStringSchema, relationSchema } from "./helpers";
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

/**
 * The `state` on another record is a display projection — queries select the
 * columns they render (id, name, abbreviation) and leave the country columns
 * out — so it is validated with the relaxed relation schema.
 */
export const usStateRelationSchema = relationSchema(usStateSchema);

export type UsStateRelation = z.infer<typeof usStateRelationSchema>;

export const usStateSelectOptionResponseSchema = createLimitOffsetResponse(usStateSchema);

export type UsStateSelectOptionResponse = z.infer<typeof usStateSelectOptionResponseSchema>;
