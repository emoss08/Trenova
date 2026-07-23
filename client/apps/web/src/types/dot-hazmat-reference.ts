import { z } from "zod";
import { optionalStringSchema } from "./helpers";

export const dotHazmatReferenceSchema = z.object({
  id: optionalStringSchema,
  unNumber: z.string(),
  properShippingName: z.string(),
  hazardClass: z.string(),
  subsidiaryHazard: optionalStringSchema,
  packingGroup: optionalStringSchema,
  specialProvisions: optionalStringSchema,
  ergGuide: optionalStringSchema,
  symbols: optionalStringSchema,
});

export type DotHazmatReference = z.infer<typeof dotHazmatReferenceSchema>;
