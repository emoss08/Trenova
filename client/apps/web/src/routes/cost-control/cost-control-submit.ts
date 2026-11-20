import { mapFieldErrorPaths } from "@/lib/form-errors";
import type { CostingControl } from "@/lib/graphql/cost-control";
import type { CostCategoryFormValues, CostControlFormValues } from "@/types/cost-control";
import type {
  CostCategoryUpdateInput,
  CostingControlInput,
  CostingControlSaveInput,
} from "@trenova/graphql/generated/graphql";
import type { FieldPath } from "@trenova/shared/types/errors";

const decimalOrNull = (value: string | null | undefined): number | null =>
  value === null || value === undefined ? null : Number(value);

const textOrNull = (value: number | null | undefined): string | null =>
  value === null || value === undefined ? null : String(value);

/** The cost control as the form edits it, from the record the server sent. */
export function toCostControlFormValues(control: CostingControl): CostControlFormValues {
  return {
    fuelIndexId: control.fuelIndexId ?? null,
    useLiveFuelPrice: control.useLiveFuelPrice,
    milesPerGallon: Number(control.milesPerGallon),
    includeDeadheadMiles: control.includeDeadheadMiles,
    glActualsEnabled: control.glActualsEnabled,
    glRollingMonths: control.glRollingMonths,
    plannedMonthlyMiles: control.plannedMonthlyMiles ?? null,
    targetMarginPercent: decimalOrNull(control.targetMarginPercent),
    version: control.version,
    categories: control.categories.map((category) => ({
      id: category.id,
      category: category.category,
      name: category.name,
      costBehavior: category.costBehavior,
      rateSource: category.rateSource,
      benchmarkRatePerMile: category.benchmarkRatePerMile,
      overrideRatePerMile: decimalOrNull(category.overrideRatePerMile),
      isActive: category.isActive,
      glAccountIds: category.glAccounts.map((link) => link.glAccountId),
      version: category.version,
    })),
  };
}

/**
 * The record as it is saved. The GL actuals settings go as they are, since the server checks
 * them only while GL actuals are on. An inactive category's rate is hidden, and an override
 * with no rate is refused, so an inactive category left on an override without one goes back
 * to the benchmark.
 */
export function prepareCostControl(values: CostControlFormValues): CostControlFormValues {
  return {
    ...values,
    categories: values.categories.map((category): CostCategoryFormValues =>
      !category.isActive &&
      category.rateSource === "Override" &&
      (category.overrideRatePerMile === null || category.overrideRatePerMile === undefined)
        ? { ...category, rateSource: "Benchmark", overrideRatePerMile: null }
        : category,
    ),
  };
}

function controlInput(values: CostControlFormValues): CostingControlInput {
  return {
    fuelIndexId: values.fuelIndexId || null,
    useLiveFuelPrice: values.useLiveFuelPrice,
    milesPerGallon: String(values.milesPerGallon),
    includeDeadheadMiles: values.includeDeadheadMiles,
    glActualsEnabled: values.glActualsEnabled,
    glRollingMonths: values.glRollingMonths,
    plannedMonthlyMiles: values.plannedMonthlyMiles ?? null,
    targetMarginPercent: textOrNull(values.targetMarginPercent),
    version: values.version,
  };
}

function categoryInput(category: CostCategoryFormValues): CostCategoryUpdateInput {
  return {
    id: category.id,
    rateSource: category.rateSource,
    overrideRatePerMile: textOrNull(category.overrideRatePerMile),
    isActive: category.isActive,
    glAccountIds: category.glAccountIds,
    version: category.version,
  };
}

function sameAccounts(a: readonly string[], b: readonly string[]): boolean {
  return a.length === b.length && a.every((id) => b.includes(id));
}

function categoryChanged(category: CostCategoryFormValues, before?: CostCategoryFormValues) {
  if (!before) return true;
  return (
    category.rateSource !== before.rateSource ||
    (category.overrideRatePerMile ?? null) !== (before.overrideRatePerMile ?? null) ||
    category.isActive !== before.isActive ||
    !sameAccounts(category.glAccountIds, before.glAccountIds)
  );
}

export type CostControlSave = {
  /** The one save the server takes: the control when one of its settings changed, and each changed category. */
  input: CostingControlSaveInput;
  /** Where each category in `input.categories` sits in the form, to place a refusal. */
  categoryIndexes: number[];
};

/**
 * What a save sends, or null when nothing changed: the control only when one of its settings
 * changed, since every write moves its version, and each category that changed. Unchanged
 * categories are not sent.
 */
export function costControlSave(
  values: CostControlFormValues,
  original: CostControlFormValues,
): CostControlSave | null {
  const next = controlInput(values);
  const settings = (input: CostingControlInput) => JSON.stringify({ ...input, version: 0 });
  const controlChanged = settings(next) !== settings(controlInput(original));

  const categories: CostCategoryUpdateInput[] = [];
  const categoryIndexes: number[] = [];
  values.categories.forEach((category, index) => {
    const before = original.categories.find((item) => item.id === category.id);
    if (!categoryChanged(category, before)) return;
    categories.push(categoryInput(category));
    categoryIndexes.push(index);
  });

  if (!controlChanged && categories.length === 0) return null;
  return { input: { control: controlChanged ? next : null, categories }, categoryIndexes };
}

/**
 * The server names a field by where it sits in the save (`["control", "milesPerGallon"]`,
 * `["categories", 0, "glAccountIds"]` for the first category sent); the form holds the
 * control's settings at its root and every category in order.
 */
export function costControlFieldPath(
  path: readonly (string | number)[],
  categoryIndexes: readonly number[],
): FieldPath {
  const [head, index, ...rest] = path;
  if (head === "control") return path.slice(1);
  if (head === "categories" && typeof index === "number") {
    const at = categoryIndexes[index];
    if (at !== undefined) return ["categories", at, ...rest];
  }
  return [...path];
}

/** A refused save with each field error moved to where the form shows it. */
export function costControlRefusal(error: unknown, categoryIndexes: readonly number[]): unknown {
  return mapFieldErrorPaths(error, (path) => costControlFieldPath(path, categoryIndexes));
}
