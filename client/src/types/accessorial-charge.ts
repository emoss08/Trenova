import { z } from "zod";
import { statusSchema, tenantInfoSchema } from "./helpers";

export const rateUnitSchema = z.enum(["Mile", "Hour", "Day", "Stop"]);

export type RateUnit = z.infer<typeof rateUnitSchema>;

export const accessorialChargeMethodSchema = z.enum([
  "Flat",
  "PerUnit",
  "Percentage",
]);

export type AccessorialChargeMethod = z.infer<
  typeof accessorialChargeMethodSchema
>;

export const accessorialChargeSchema = z
  .object({
    ...tenantInfoSchema.shape,

    status: statusSchema,
    code: z
      .string({ error: "Code is required" })
      .min(3, { error: "Code must be at least 3 characters" })
      .max(10, { error: "Code must be less than 10 characters" }),
    description: z.string().min(1, { error: "Description is required" }),
    method: accessorialChargeMethodSchema,
    rateUnit: rateUnitSchema.optional(),
    amount: z.preprocess(
      (val) => {
        if (val === "" || val === null || val === undefined) return undefined;
        const parsed = parseFloat(typeof val === "string" || typeof val === "number" || typeof val === "boolean" ? String(val) : "");
        return isNaN(parsed) ? undefined : parsed;
      },
      z.number().min(0.01, { message: "Amount must be greater than zero" }),
    ),
  })
  .refine(
    (data) => {
      if (data.method === "PerUnit" && !data.rateUnit) {
        return false;
      }

      return true;
    },
    {
      path: ["rateUnit"],
      message: "Rate unit is required when method is PerUnit",
    },
  );

export type AccessorialCharge = z.infer<typeof accessorialChargeSchema>;
