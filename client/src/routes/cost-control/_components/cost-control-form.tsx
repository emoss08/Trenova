import {
  FuelIndexAutocompleteField,
  GLAccountMultiSelectAutocompleteField,
} from "@/components/autocomplete-fields";
import { NumberField } from "@/components/fields/number-field";
import { SwitchField } from "@/components/fields/switch-field";
import { FormSaveDock } from "@/components/form-save-dock";
import { Badge } from "@/components/ui/badge";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Form, FormControl, FormGroup } from "@/components/ui/form";
import { Label } from "@/components/ui/label";
import { Separator } from "@/components/ui/separator";
import { Switch } from "@/components/ui/switch";
import { useOptimisticMutation } from "@/hooks/use-optimistic-mutation";
import {
  getCostingControlGraphQL,
  updateCostCategoryGraphQL,
  updateCostingControlGraphQL,
  type CostingControl,
} from "@/lib/graphql/cost-control";
import { queries } from "@/lib/queries";
import { formatPerMile } from "@/lib/utils";
import {
  costControlSchema,
  type CostBehavior,
  type CostControlFormValues,
} from "@/types/cost-control";
import { zodResolver } from "@hookform/resolvers/zod";
import { useSuspenseQuery } from "@tanstack/react-query";
import { useCallback } from "react";
import {
  FormProvider,
  type Resolver,
  useForm,
  useFormContext,
  useWatch,
} from "react-hook-form";

function toFormValues(control: CostingControl): CostControlFormValues {
  return {
    fuelIndexId: control.fuelIndexId ?? null,
    useLiveFuelPrice: control.useLiveFuelPrice,
    milesPerGallon: Number(control.milesPerGallon),
    includeDeadheadMiles: control.includeDeadheadMiles,
    glActualsEnabled: control.glActualsEnabled,
    glRollingMonths: control.glRollingMonths,
    plannedMonthlyMiles: control.plannedMonthlyMiles ?? null,
    targetMarginPercent:
      control.targetMarginPercent !== null && control.targetMarginPercent !== undefined
        ? Number(control.targetMarginPercent)
        : null,
    version: control.version,
    categories: control.categories.map((category) => ({
      id: category.id,
      category: category.category,
      name: category.name,
      costBehavior: category.costBehavior,
      rateSource: category.rateSource,
      benchmarkRatePerMile: category.benchmarkRatePerMile,
      overrideRatePerMile:
        category.overrideRatePerMile !== null && category.overrideRatePerMile !== undefined
          ? Number(category.overrideRatePerMile)
          : null,
      isActive: category.isActive,
      glAccountIds: category.glAccounts.map((link) => link.glAccountId),
      version: category.version,
    })),
  };
}

function categoryInputsChanged(
  values: CostControlFormValues,
  original: CostControlFormValues,
): CostControlFormValues["categories"] {
  return values.categories.filter((category) => {
    const before = original.categories.find((item) => item.id === category.id);
    if (!before) return true;
    return (
      category.rateSource !== before.rateSource ||
      category.overrideRatePerMile !== before.overrideRatePerMile ||
      category.isActive !== before.isActive ||
      category.glAccountIds.length !== before.glAccountIds.length ||
      category.glAccountIds.some((id) => !before.glAccountIds.includes(id))
    );
  });
}

async function submitCostControl(values: CostControlFormValues, original: CostControlFormValues) {
  await updateCostingControlGraphQL({
    fuelIndexId: values.fuelIndexId || null,
    useLiveFuelPrice: values.useLiveFuelPrice,
    milesPerGallon: String(values.milesPerGallon),
    includeDeadheadMiles: values.includeDeadheadMiles,
    glActualsEnabled: values.glActualsEnabled,
    glRollingMonths: values.glRollingMonths,
    plannedMonthlyMiles: values.plannedMonthlyMiles ?? null,
    targetMarginPercent:
      values.targetMarginPercent !== null && values.targetMarginPercent !== undefined
        ? String(values.targetMarginPercent)
        : null,
    version: values.version,
  });

  const changedCategories = categoryInputsChanged(values, original);
  for (const category of changedCategories) {
    await updateCostCategoryGraphQL({
      id: category.id,
      rateSource: category.rateSource,
      overrideRatePerMile:
        category.overrideRatePerMile !== null && category.overrideRatePerMile !== undefined
          ? String(category.overrideRatePerMile)
          : null,
      isActive: category.isActive,
      glAccountIds: category.glAccountIds,
      version: category.version,
    });
  }

  return getCostingControlGraphQL();
}

export default function CostControlForm() {
  const { data } = useSuspenseQuery({
    ...queries.costControl.get(),
  });

  const defaultValues = toFormValues(data);

  const form = useForm<CostControlFormValues>({
    resolver: zodResolver(costControlSchema) as Resolver<CostControlFormValues>,
    defaultValues,
    values: defaultValues,
  });

  const { handleSubmit, setError, reset } = form;

  const { mutateAsync } = useOptimisticMutation({
    queryKey: queries.costControl.get._def,
    mutationFn: async (values: CostControlFormValues) => submitCostControl(values, defaultValues),
    resourceName: "Cost Control",
    resetForm: reset,
    setFormError: setError,
    invalidateQueries: [
      queries.costControl.get._def,
      queries.costControl.resolvedProfile._def,
      queries.shipment._def,
      ["analytics"],
    ],
  });

  const onSubmit = useCallback(
    async (values: CostControlFormValues) => {
      await mutateAsync(values);
    },
    [mutateAsync],
  );

  return (
    <FormProvider {...form}>
      <Form onSubmit={handleSubmit(onSubmit)}>
        <div className="flex flex-col gap-4 pb-14">
          <CostBasisCard />
          <CategoryRatesCard
        title="Variable Costs"
        description="Per-mile costs that scale with miles driven. Each category uses its industry benchmark unless you override it or map it to GL actuals."
            behavior="Variable"
          />
          <CategoryRatesCard
            title="Fixed Costs"
            description="Ownership and overhead costs normalized to a per-mile rate. When GL actuals are enabled, fixed categories divide by planned monthly miles when set."
            behavior="Fixed"
          />
          <GLActualsCard />
          <FormSaveDock saveButtonContent="Save Changes" />
        </div>
      </Form>
    </FormProvider>
  );
}

function CostBasisCard() {
  const { control } = useFormContext<CostControlFormValues>();
  const useLiveFuelPrice = useWatch({ control, name: "useLiveFuelPrice" });

  return (
    <Card>
      <CardHeader>
        <CardTitle>Cost Basis</CardTitle>
        <CardDescription>
          Core assumptions behind the cost-per-mile estimate: fuel pricing, fleet efficiency, and
          how deadhead miles are attributed to shipment cost.
        </CardDescription>
      </CardHeader>
      <CardContent className="max-w-prose">
        <FormGroup cols={1}>
          <FormControl>
            <SwitchField
              control={control}
              name="useLiveFuelPrice"
              label="Use Live Fuel Price"
              description="Derives the fuel cost per mile from the latest fuel index price divided by fleet MPG instead of the static benchmark."
              position="left"
            />
          </FormControl>
          {useLiveFuelPrice && (
            <FormControl className="max-w-[420px]">
              <FuelIndexAutocompleteField
                control={control}
                name="fuelIndexId"
                label="Fuel Index"
                placeholder="Select fuel index"
                description="Diesel price index used to resolve the live fuel cost per mile."
                clearable
              />
            </FormControl>
          )}
          <FormControl className="max-w-[420px]">
            <NumberField
              control={control}
              name="milesPerGallon"
              label="Fleet Miles Per Gallon"
              description="Average fleet fuel efficiency used to convert diesel price per gallon into cost per mile."
              rules={{ required: true }}
            />
          </FormControl>
          <FormControl>
            <SwitchField
              control={control}
              name="includeDeadheadMiles"
              label="Include Deadhead Miles"
              description="Charges empty repositioning miles to the shipment cost estimate. Industry practice is to include them."
              position="left"
            />
          </FormControl>
          <FormControl className="max-w-[420px]">
            <NumberField
              control={control}
              name="targetMarginPercent"
              label="Target Margin Percent"
              description="Margin threshold used to color-code shipment profitability. Margins below this show as thin; defaults to 10% when unset."
            />
          </FormControl>
        </FormGroup>
      </CardContent>
    </Card>
  );
}

const rateSourceBadge: Record<string, string> = {
  Benchmark: "Benchmark",
  Override: "Override",
  GLActual: "GL Actual",
};

function CategoryRatesCard({
  title,
  description,
  behavior,
}: {
  title: string;
  description: string;
  behavior: CostBehavior;
}) {
  const { control } = useFormContext<CostControlFormValues>();
  const categories = useWatch({ control, name: "categories" }) ?? [];

  return (
    <Card>
      <CardHeader>
        <CardTitle>{title}</CardTitle>
        <CardDescription>{description}</CardDescription>
      </CardHeader>
      <CardContent>
        <div className="flex flex-col">
          {categories.map((category, index) => {
            if (category.costBehavior !== behavior) return null;
            return (
              <CategoryRow
                key={category.id}
                index={index}
                isLast={
                  categories.filter((item) => item.costBehavior === behavior).at(-1)?.id ===
                  category.id
                }
              />
            );
          })}
        </div>
      </CardContent>
    </Card>
  );
}

function CategoryRow({ index, isLast }: { index: number; isLast: boolean }) {
  const { control, setValue, getValues } = useFormContext<CostControlFormValues>();
  const category = useWatch({ control, name: `categories.${index}` });
  const glActualsEnabled = useWatch({ control, name: "glActualsEnabled" });

  const isOverride = category.rateSource === "Override";
  const isGLActual = category.rateSource === "GLActual";

  const setRateSource = useCallback(
    (rateSource: CostControlFormValues["categories"][number]["rateSource"]) => {
      setValue(`categories.${index}.rateSource`, rateSource, {
        shouldDirty: true,
        shouldValidate: true,
      });
      if (rateSource !== "Override" && getValues(`categories.${index}.overrideRatePerMile`)) {
        setValue(`categories.${index}.overrideRatePerMile`, null, { shouldDirty: true });
      }
    },
    [getValues, index, setValue],
  );

  return (
    <div className="flex flex-col gap-3 py-4">
      <div className="flex flex-wrap items-center justify-between gap-2">
        <div className="flex items-center gap-2">
          <span className="text-sm font-medium">{category.name}</span>
          <Badge variant="secondary" className="text-2xs">
            {rateSourceBadge[category.rateSource]}
          </Badge>
        </div>
        <span className="text-sm text-muted-foreground tabular-nums">
          Benchmark {formatPerMile(Number(category.benchmarkRatePerMile))}
        </span>
      </div>
      <FormGroup cols={2}>
        <FormControl>
          <SwitchField
            control={control}
            name={`categories.${index}.isActive`}
            label="Active"
            description="Inactive categories are excluded from the cost-per-mile total."
            position="left"
          />
        </FormControl>
        <FormControl>
          <DerivedSwitchRow
            id={`override-${category.id}`}
            checked={isOverride}
            onCheckedChange={(checked) => setRateSource(checked ? "Override" : "Benchmark")}
            label="Override Rate"
            description="Replace the industry benchmark with your own per-mile rate."
          />
        </FormControl>
        {isOverride && (
          <FormControl className="max-w-[280px]">
            <NumberField
              control={control}
              name={`categories.${index}.overrideRatePerMile`}
              label="Override Rate Per Mile"
              placeholder="0.00"
              rules={{ required: true }}
            />
          </FormControl>
        )}
        {glActualsEnabled && (
          <FormControl>
            <DerivedSwitchRow
              id={`gl-actual-${category.id}`}
              checked={isGLActual}
              onCheckedChange={(checked) => setRateSource(checked ? "GLActual" : "Benchmark")}
              label="Use GL Actuals"
              description="Derive this rate from posted GL expenses divided by fleet miles over the rolling window."
            />
          </FormControl>
        )}
      </FormGroup>
      <FormControl className="max-w-[560px]">
        <GLAccountMultiSelectAutocompleteField
          control={control}
          name={`categories.${index}.glAccountIds`}
          label="Mapped GL Accounts"
          placeholder="Select GL accounts"
          description="Expense accounts whose postings feed this category when GL actuals are enabled."
          clearable
        />
      </FormControl>
      {!isLast && <Separator />}
    </div>
  );
}

function DerivedSwitchRow({
  id,
  checked,
  onCheckedChange,
  label,
  description,
}: {
  id: string;
  checked: boolean;
  onCheckedChange: (checked: boolean) => void;
  label: string;
  description: string;
}) {
  return (
    <div className="group relative flex w-full items-start gap-2 rounded-md border border-transparent p-2.5">
      <Switch id={id} checked={checked} onCheckedChange={onCheckedChange} />
      <div className="flex grow flex-col gap-0.5">
        <Label htmlFor={id} className="text-sm font-medium">
          {label}
        </Label>
        <p className="text-sm text-muted-foreground">{description}</p>
      </div>
    </div>
  );
}

function GLActualsCard() {
  const { control } = useFormContext<CostControlFormValues>();
  const glActualsEnabled = useWatch({ control, name: "glActualsEnabled" });

  return (
    <Card>
      <CardHeader>
        <CardTitle>GL Actuals</CardTitle>
        <CardDescription>
          When enabled, categories mapped to GL accounts derive their per-mile rate from real
          posted expenses over a rolling window, replacing benchmarks as your ledger fills in.
        </CardDescription>
      </CardHeader>
      <CardContent className="max-w-prose">
        <FormGroup cols={1}>
          <FormControl>
            <SwitchField
              control={control}
              name="glActualsEnabled"
              label="Enable GL Actuals"
              description="Allow categories with mapped GL accounts to use posted expense actuals."
              position="left"
            />
          </FormControl>
          {glActualsEnabled && (
            <>
              <FormControl className="max-w-[420px]">
                <NumberField
                  control={control}
                  name="glRollingMonths"
                  label="Rolling Window (Months)"
                  description="Number of trailing months of GL activity used to compute actual rates. Three months smooths lumpy maintenance spend."
                  rules={{ required: true }}
                />
              </FormControl>
              <FormControl className="max-w-[420px]">
                <NumberField
                  control={control}
                  name="plannedMonthlyMiles"
                  label="Planned Monthly Miles"
                  description="Divisor floor for fixed categories so low-mileage months do not inflate fixed cost per mile. Leave empty to always divide by actual fleet miles."
                />
              </FormControl>
            </>
          )}
        </FormGroup>
      </CardContent>
    </Card>
  );
}
