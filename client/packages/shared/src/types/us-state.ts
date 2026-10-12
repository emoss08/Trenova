import { z } from "zod";
import { optionalStringSchema, relationSchema } from "./helpers";
import { createLimitOffsetResponse } from "./server";
import { translate } from "@trenova/shared/i18n/runtime";

export const usStateSchema = z.object({
  id: optionalStringSchema,
  name: z
    .string({
      error: () => translate("Name is required"),
    })
    .min(1, { error: () => translate("Name is required") }),
  abbreviation: z
    .string({
      error: () => translate("Abbreviation is required"),
    })
    .min(1, { error: () => translate("Abbreviation is required") }),
  countryName: z.string().optional(),
  countryIso3: z
    .string({
      error: () => translate("Country ISO 3 is required"),
    })
    .min(1, { error: () => translate("Country ISO 3 is required") }),
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
