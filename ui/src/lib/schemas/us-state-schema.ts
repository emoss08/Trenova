import { type InferType, object, string } from "yup";

export const usStateSchema = object({
  id: string().optional(),
  name: string().required("Name is required"),
  abbreviation: string().required("Abbreviation is required"),
  countryName: string().required("Country name is required"),
  countryIso3: string().required("Country ISO 3 is required"),
});

export type UsStateSchema = InferType<typeof usStateSchema>;
