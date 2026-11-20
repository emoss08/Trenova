import { translate } from "@trenova/shared/i18n/runtime";
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
      .min(0, { error: () => translate("Rate must be 0 or greater") })
      .max(100, { error: () => translate("Rate must be 100 or less") })
      .nullish(),
    isActive: z.boolean(),
    glAccountIds: z.array(z.string()),
    version: z.number(),
  })
  .superRefine((values, ctx) => {
    // An inactive category is left out of the total and its rate is not shown; the page
    // sends it back to the benchmark rather than refuse a rate nobody can see.
    if (
      values.isActive &&
      values.rateSource === "Override" &&
      (values.overrideRatePerMile === null || values.overrideRatePerMile === undefined)
    ) {
      ctx.addIssue({
        code: "custom",
        path: ["overrideRatePerMile"],
        message: translate("Enter your rate per mile"),
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
      .positive({ error: () => translate("Miles per gallon must be greater than 0") })
      .max(20, { error: () => translate("Miles per gallon must be 20 or less") }),
    includeDeadheadMiles: z.boolean(),
    glActualsEnabled: z.boolean(),
    glRollingMonths: z.coerce.number().int(),
    plannedMonthlyMiles: z.coerce.number().int().nullish(),
    targetMarginPercent: z.coerce
      .number()
      .min(0, { error: () => translate("Target margin must be 0 or greater") })
      .max(100, { error: () => translate("Target margin must be 100 or less") })
      .nullish(),
    version: z.number(),
    categories: z.array(costCategoryFormSchema),
  })
  .superRefine((values, ctx) => {
    if (values.useLiveFuelPrice && !values.fuelIndexId) {
      ctx.addIssue({
        code: "custom",
        path: ["fuelIndexId"],
        message: translate("Fuel index is required when live fuel pricing is enabled"),
      });
    }
    // The GL actuals settings are shown only while GL actuals are on; while they are off
    // the page sends values the server accepts (cost-control-submit.ts).
    if (!values.glActualsEnabled) return;
    if (values.glRollingMonths < 1 || values.glRollingMonths > 12) {
      ctx.addIssue({
        code: "custom",
        path: ["glRollingMonths"],
        message: translate("GL rolling months must be between 1 and 12"),
      });
    }
    if (values.plannedMonthlyMiles != null && values.plannedMonthlyMiles < 1) {
      ctx.addIssue({
        code: "custom",
        path: ["plannedMonthlyMiles"],
        message: translate("Planned monthly miles must be greater than 0"),
      });
    }
    // A category reads its rate from the ledger only through the accounts mapped to it; with
    // none mapped it would quietly fall back to the benchmark.
    values.categories.forEach((category, index) => {
      if (
        category.isActive &&
        category.rateSource === "GLActual" &&
        category.glAccountIds.length === 0
      ) {
        ctx.addIssue({
          code: "custom",
          path: ["categories", index, "glAccountIds"],
          message: translate("Map at least one GL account to use GL actuals for this category"),
        });
      }
    });
  });
export type CostControlFormValues = z.infer<typeof costControlSchema>;
