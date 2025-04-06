import { Status } from "@/types/common";
import { boolean, type InferType, mixed, number, object, string } from "yup";

export const commoditySchema = object({
  id: string().optional(),
  organizationId: string().nullable().optional(),
  businessUnitId: string().nullable().optional(),
  hazardousMaterialId: string().nullable().optional(),
  status: mixed<Status>()
    .required("Status is required")
    .oneOf(Object.values(Status)),
  name: string().required("Name is required"),
  description: string().required("Description is required"),
  minTemperature: number().nullable().optional(),
  maxTemperature: number().nullable().optional(),
  weightPerUnit: number().optional().nullable(),
  linearFeetPerUnit: number().optional().nullable(),
  freightClass: string().optional(),
  dotClassification: string().optional(),
  stackable: boolean().required("Stackable is required"),
  fragile: boolean().required("Fragile is required"),
});

export type CommoditySchema = InferType<typeof commoditySchema>;
