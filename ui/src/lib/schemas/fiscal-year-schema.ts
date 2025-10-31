import * as z from "zod";
import {
  nullablePulidSchema,
  nullableTimestampSchema,
  optionalIntegerSchema,
  optionalStringSchema,
  timestampSchema,
  versionSchema,
} from "./helpers";

export const FiscalYearStatusSchema = z.enum([
  "Draft",
  "Open",
  "Closed",
  "Locked",
]);

export const fiscalYearSchema = z
  .object({
    id: optionalStringSchema,
    version: versionSchema,
    createdAt: timestampSchema,
    updatedAt: timestampSchema,
    organizationId: optionalStringSchema,
    businessUnitId: optionalStringSchema,

    status: FiscalYearStatusSchema,
    year: z
      .number()
      .int()
      .positive()
      .min(1900, {
        error: "Year must be between 1900 and 2100",
      })
      .max(2100, {
        error: "Year must be between 1900 and 2100",
      }),
    name: z
      .string()
      .min(1, {
        error: "Name is required",
      })
      .max(100, {
        error: "Name must be less than 100 characters",
      }),
    description: optionalStringSchema,
    startDate: z.number().int().positive().min(1, {
      error: "Start date is required",
    }),
    endDate: z.number().int().positive().min(1, {
      error: "End date is required",
    }),
    taxYear: optionalIntegerSchema,
    budgetAmount: optionalIntegerSchema,
    adjustmentDeadline: optionalIntegerSchema,
    isCurrent: z.boolean(),
    isCalendarYear: z.boolean(),
    allowAdjustingEntries: z.boolean(),
    closedAt: nullableTimestampSchema,
    lockedAt: nullableTimestampSchema,
    closedById: nullablePulidSchema,
    lockedById: nullablePulidSchema,
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
  )
  .refine(
    (data) => {
      // Cannot create fiscal years more than 5 years in the future
      if (data.year > new Date().getFullYear() + 5) {
        return false;
      }
      return true;
    },
    {
      message: "Cannot create fiscal years more than 5 years in the future",
      path: ["year"],
    },
  );

export type FiscalYearSchema = z.infer<typeof fiscalYearSchema>;
