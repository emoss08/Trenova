import { z } from "zod";

export const usStateSchema = z.object({
  id: z.string().optional(),
  name: z.string().min(1, "Name is required"),
  abbreviation: z.string().min(1, "Abbreviation is required"),
  countryName: z.string().min(1, "Country name is required"),
  countryIso3: z.string().min(1, "Country ISO 3 is required"),
});

export type UsStateSchema = z.infer<typeof usStateSchema>;
