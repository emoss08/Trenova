import { z } from "zod";

export const costBehaviorSchema = z.enum(["Fixed", "Variable"]);
export type CostBehavior = z.infer<typeof costBehaviorSchema>;

export const costRateSourceSchema = z.enum(["Benchmark", "Override", "GLActual"]);
export type CostRateSource = z.infer<typeof costRateSourceSchema>;

export const costCategoryTypeSchema = z.enum([
  "DriverWages",
  "DriverBenefits",
  "Fuel",
  "EquipmentPayments",
  "Maintenance",
  "Insurance",
  "Tires",
  "Tolls",
  "PermitsLicenses",
  "Overhead",
  "Custom",
]);
export type CostCategoryType = z.infer<typeof costCategoryTypeSchema>;

export const costCategoryFormSchema = z
  .object({
    id: z.string(),
    category: costCategoryTypeSchema,
    name: z.string().min(1),
    costBehavior: costBehaviorSchema,
    rateSource: costRateSourceSchema,
    benchmarkRatePerMile: z.string(),
    overrideRatePerMile: z.coerce
      .number()
      .min(0, "Rate must be 0 or greater")
      .max(100, "Rate must be 100 or less")
      .nullish(),
    isActive: z.boolean(),
    glAccountIds: z.array(z.string()),
    version: z.number(),
  })
  .superRefine((values, ctx) => {
    if (
      values.rateSource === "Override" &&
      (values.overrideRatePerMile === null || values.overrideRatePerMile === undefined)
    ) {
      ctx.addIssue({
        code: "custom",
        path: ["overrideRatePerMile"],
        message: "Override rate is required when the rate source is Override",
      });
    }
    if (values.rateSource === "GLActual" && values.glAccountIds.length === 0) {
      ctx.addIssue({
        code: "custom",
        path: ["glAccountIds"],
        message: "Map at least one GL account to use GL actuals for this category",
      });
    }
  });
export type CostCategoryFormValues = z.infer<typeof costCategoryFormSchema>;

export const costControlSchema = z
  .object({
    fuelIndexId: z.string().nullish(),
    useLiveFuelPrice: z.boolean(),
    milesPerGallon: z.coerce
      .number()
      .positive("Miles per gallon must be greater than 0")
      .max(20, "Miles per gallon must be 20 or less"),
    includeDeadheadMiles: z.boolean(),
    glActualsEnabled: z.boolean(),
    glRollingMonths: z.coerce
      .number()
      .int()
      .min(1, "GL rolling months must be at least 1")
      .max(12, "GL rolling months must be 12 or less"),
    plannedMonthlyMiles: z.coerce
      .number()
      .int()
      .positive("Planned monthly miles must be greater than 0")
      .nullish(),
    targetMarginPercent: z.coerce
      .number()
      .min(0, "Target margin must be 0 or greater")
      .max(100, "Target margin must be 100 or less")
      .nullish(),
    version: z.number(),
    categories: z.array(costCategoryFormSchema),
  })
  .superRefine((values, ctx) => {
    if (values.useLiveFuelPrice && !values.fuelIndexId) {
      ctx.addIssue({
        code: "custom",
        path: ["fuelIndexId"],
        message: "Fuel index is required when live fuel pricing is enabled",
      });
    }
  });
export type CostControlFormValues = z.infer<typeof costControlSchema>;
