import * as z from "zod";
import {
  nullablePulidSchema,
  nullableTimestampSchema,
  optionalStringSchema,
  timestampSchema,
  versionSchema,
} from "./helpers";

export const FiscalPeriodStatusSchema = z.enum(["Open", "Closed", "Locked"]);

export const FiscalPeriodTypeSchema = z.enum(["Month", "Quarter", "Year"]);

export const fiscalPeriodSchema = z
  .object({
    id: optionalStringSchema,
    version: versionSchema,
    createdAt: timestampSchema,
    updatedAt: timestampSchema,
    organizationId: optionalStringSchema,
    businessUnitId: optionalStringSchema,
    status: FiscalPeriodStatusSchema,
    periodType: FiscalPeriodTypeSchema,
    fiscalYearId: optionalStringSchema,
    periodNumber: z
      .number()
      .int()
      .positive()
      .min(1, {
        error: "Period number is required",
      })
      .max(12, {
        error: "Period number must be less than 12",
      }),
    name: z
      .string()
      .min(1, {
        error: "Name is required",
      })
      .max(100, {
        error: "Name must be less than 100 characters",
      }),
    startDate: z.number().int().positive().min(1, {
      error: "Start date is required",
    }),
    endDate: z.number().int().positive().min(1, {
      error: "End date is required",
    }),
    closedAt: nullableTimestampSchema,
    closedById: nullablePulidSchema,
  })
  .refine(
    (data) => {
      if (data.startDate >= data.endDate) {
        return false;
      }
      return true;
    },
    {
      message: "Start date must be before end date",
      path: ["endDate"],
    },
  );

export type FiscalPeriodSchema = z.infer<typeof fiscalPeriodSchema>;
