import { AccessorialChargeMethod } from "@/types/billing";
import { Status } from "@/types/common";
import { z } from "zod";

export const accessorialChargeSchema = z.object({
  id: z.string().optional(),
  organizationId: z.string().optional(),
  businessUnitId: z.string().optional(),
  version: z.number().optional(),
  createdAt: z.number().optional(),
  updatedAt: z.number().optional(),

  // * Core Fields
  status: z.nativeEnum(Status),
  code: z
    .string()
    .min(3, "Code must be at least 3 characters")
    .max(10, "Code must be less than 10 characters"),
  description: z.string().min(1, "Description is required"),
  unit: z.preprocess(
    (val) => {
      if (val === "" || val === null || val === undefined) {
        return undefined;
      }
      const parsed = parseInt(String(val), 10);
      return isNaN(parsed) ? undefined : parsed;
    },
    z.number({
      required_error: "Unit is required",
    }),
  ),
  method: z.nativeEnum(AccessorialChargeMethod, {
    message: "Method is required",
  }),
  amount: z.preprocess(
    (val) => {
      if (val === "" || val === null || val === undefined) {
        return undefined;
      }
      const parsed = parseFloat(String(val));
      return isNaN(parsed) ? undefined : parsed;
    },
    z.number().min(1, "Amount is required"),
  ),
});

export type AccessorialChargeSchema = z.infer<typeof accessorialChargeSchema>;
