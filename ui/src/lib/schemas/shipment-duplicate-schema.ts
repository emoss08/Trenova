import { z } from "zod";

export const shipmentDuplicateSchema = z.object({
  count: z.preprocess(
    (val) => {
      if (val === "" || val === null || val === undefined) {
        return undefined;
      }
      const parsed = parseInt(String(val), 10);
      return isNaN(parsed) ? undefined : parsed;
    },
    z
      .number({
        required_error: "Count is required",
      })
      .min(1, "Count is required")
      .max(20, "Count must be less than 20"),
  ),
  overrideDates: z.boolean(),
  includeCommodities: z.boolean(),
  includeAdditionalCharges: z.boolean(),
  shipmentID: z.string().min(1, "Shipment ID is required"),
});

export type ShipmentDuplicateSchema = z.infer<typeof shipmentDuplicateSchema>;
