import { z } from "zod";
import {
  decimalStringSchema,
  nullableIntegerSchema,
  nullableStringSchema,
  statusSchema,
  tenantInfoSchema,
} from "./helpers";

export const freightClassSchema = z.enum([
  "Class50",
  "Class55",
  "Class60",
  "Class65",
  "Class70",
  "Class77_5",
  "Class85",
  "Class92_5",
  "Class100",
  "Class110",
  "Class125",
  "Class150",
  "Class175",
  "Class200",
  "Class250",
  "Class300",
  "Class400",
  "Class500",
]);

export type FreightClass = z.infer<typeof freightClassSchema>;

export const commoditySchema = z.object({
  ...tenantInfoSchema.shape,
  status: statusSchema,
  name: z.string().min(1, { error: "Name is required" }),
  description: z.string().min(1, { error: "Description is required" }),
  hazardousMaterialId: nullableStringSchema,
  minTemperature: nullableIntegerSchema,
  maxTemperature: nullableIntegerSchema,
  weightPerUnit: decimalStringSchema,
  linearFeetPerUnit: decimalStringSchema,
  maxQuantityPerShipment: decimalStringSchema,
  freightClass: nullableStringSchema,
  loadingInstructions: z.string().optional(),
  stackable: z.boolean().default(false),
  fragile: z.boolean().default(false),
});

export type Commodity = z.infer<typeof commoditySchema>;

export const bulkUpdateCommodityStatusRequestSchema = z.object({
  commodityIds: z.array(z.string()),
  status: statusSchema,
});

export type BulkUpdateCommodityStatusRequest = z.infer<
  typeof bulkUpdateCommodityStatusRequestSchema
>;

export const bulkUpdateCommodityStatusResponseSchema = z.array(commoditySchema);

export type BulkUpdateCommodityStatusResponse = z.infer<
  typeof bulkUpdateCommodityStatusResponseSchema
>;
