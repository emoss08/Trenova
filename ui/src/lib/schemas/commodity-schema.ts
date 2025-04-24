import { Status } from "@/types/common";
import { object, z } from "zod";

export const commoditySchema = object({
  id: z.string().optional(),
  version: z.number().optional(),
  createdAt: z.number().optional(),
  updatedAt: z.number().optional(),

  // * Core Fields
  hazardousMaterialId: z.string().nullable().optional(),
  status: z.nativeEnum(Status),
  name: z.string().min(1, "Name is required"),
  description: z.string().min(1, "Description is required"),
  minTemperature: z.number().nullable().optional(),
  maxTemperature: z.number().nullable().optional(),
  weightPerUnit: z.preprocess((val) => {
    if (val === "" || val === null || val === undefined) {
      return undefined;
    }
    const parsed = parseFloat(String(val));
    return isNaN(parsed) ? undefined : parsed;
  }, z.number().optional()),
  linearFeetPerUnit: z.preprocess((val) => {
    if (val === "" || val === null || val === undefined) {
      return undefined;
    }
    const parsed = parseFloat(String(val));
    return isNaN(parsed) ? undefined : parsed;
  }, z.number().optional()),
  freightClass: z.string().optional(),
  dotClassification: z.string().optional(),
  stackable: z.boolean(),
  fragile: z.boolean(),
});

export type CommoditySchema = z.infer<typeof commoditySchema>;
