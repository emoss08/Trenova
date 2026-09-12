import { useT } from "@trenova/shared/i18n/use-t";
import { InputField } from "@/components/fields/input-field";
import { MultiCheckboxField } from "@/components/fields/multi-checkbox-field";
import { NumberField } from "@/components/fields/number-field";
import { SelectField } from "@/components/fields/select-field";
import { SwitchField } from "@/components/fields/switch-field";
import { TextareaField } from "@/components/fields/textarea-field";
import { useApiMutation } from "@/hooks/use-api-mutation";
import { driverTypeChoices, statusChoices } from "@/lib/choices";
import type { DriverType } from "@trenova/shared/types/worker";
import {
  createDotRandomPool,
  DOT_RANDOM_POOLS_KEY,
  updateDotRandomPool,
  type RandomPoolRow,
} from "@/lib/graphql/worker-drug-alcohol";
import { zodResolver } from "@hookform/resolvers/zod";
import { useQueryClient } from "@tanstack/react-query";
import { Alert, AlertDescription } from "@trenova/shared/components/ui/alert";
import { Button } from "@trenova/shared/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@trenova/shared/components/ui/dialog";
import { Form, FormControl, FormGroup } from "@trenova/shared/components/ui/form";
import {
  DOT_MINIMUM_ALCOHOL_RATE,
  DOT_MINIMUM_DRUG_RATE,
  projectedRoundTarget,
  randomPeriodLabel,
} from "@trenova/shared/lib/drug-alcohol";
import {
  randomPeriodSchema,
  randomPoolFormSchema,
  type RandomPoolFormValues,
} from "@trenova/shared/types/worker-drug-alcohol";
import { useEffect } from "react";
import { FormProvider, useForm, useWatch, type Resolver } from "react-hook-form";
import { toast } from "sonner";

const PERIOD_OPTIONS = randomPeriodSchema.options.map((value) => ({
  value,
  label: randomPeriodLabel(value),
}));

/** A worked example, so the rates read as collections rather than percentages. */
const EXAMPLE_POOL_SIZE = 100;

export type RandomPoolDialogProps = {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  pool: RandomPoolRow | null;
};

function defaultsFor(pool: RandomPoolRow | null): RandomPoolFormValues {
  if (!pool) {
    return {
      code: "",
      name: "",
      description: null,
      status: "Active",
      period: "Quarterly",
      drugRatePercent: DOT_MINIMUM_DRUG_RATE,
      alcoholRatePercent: DOT_MINIMUM_ALCOHOL_RATE,
      includedDriverTypes: null,
      isDefault: false,
    };
  }
  return {
    code: pool.code,
    name: pool.name,
    description: pool.description ?? null,
    status: pool.status as "Active" | "Inactive",
    period: pool.period,
    drugRatePercent: pool.drugRatePercent,
    alcoholRatePercent: pool.alcoholRatePercent,
    includedDriverTypes: pool.includedDriverTypes.length > 0 ? [...pool.includedDriverTypes] : null,
    isDefault: pool.isDefault,
  };
}

export function RandomPoolDialog({ open, onOpenChange, pool }: RandomPoolDialogProps) {
  const t = useT();

  const queryClient = useQueryClient();
  const isEdit = Boolean(pool);
  const form = useForm<RandomPoolFormValues>({
    resolver: zodResolver(randomPoolFormSchema) as Resolver<RandomPoolFormValues>,
    defaultValues: defaultsFor(pool),
  });
  const { control, handleSubmit, reset } = form;

  useEffect(() => {
    if (!open) return;
    reset(defaultsFor(pool));
  }, [open, pool, reset]);

  const [period, drugRate, alcoholRate] = useWatch({
    control,
    name: ["period", "drugRatePercent", "alcoholRatePercent"],
  });
  const belowMinimum = drugRate < DOT_MINIMUM_DRUG_RATE || alcoholRate < DOT_MINIMUM_ALCOHOL_RATE;
  const exampleDrug = projectedRoundTarget(EXAMPLE_POOL_SIZE, drugRate, period);
  const exampleAlcohol = projectedRoundTarget(EXAMPLE_POOL_SIZE, alcoholRate, period);

  const { mutateAsync, isPending } = useApiMutation<
    RandomPoolRow,
    RandomPoolFormValues,
    unknown,
    RandomPoolFormValues
  >({
    form,
    resourceName: "Pool",
    mutationFn: (values) => {
      const input = {
        code: values.code,
        name: values.name,
        description: values.description ?? undefined,
        status: values.status,
        period: values.period,
        drugRatePercent: values.drugRatePercent,
        alcoholRatePercent: values.alcoholRatePercent,
        includedDriverTypes: values.includedDriverTypes ?? [],
        isDefault: values.isDefault,
      };
      return pool ? updateDotRandomPool(pool.id, pool.version, input) : createDotRandomPool(input);
    },
    onSuccess: () => {
      toast.success(isEdit ? "Pool updated" : "Pool created");
      void queryClient.invalidateQueries({ queryKey: [DOT_RANDOM_POOLS_KEY] });
      onOpenChange(false);
    },
  });

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-2xl">
        <DialogHeader>
          <DialogTitle>{isEdit ? t("Edit pool") : t("New random testing pool")}</DialogTitle>
          <DialogDescription>
            {t("The rates are annual. Each round draws its share of them, rounded up so a year of rounds cannot finish under the minimum.")}
          </DialogDescription>
        </DialogHeader>
        <FormProvider {...form}>
          <Form
            onSubmit={(submitEvent) => {
              submitEvent.preventDefault();
              submitEvent.stopPropagation();
              void handleSubmit((values) => mutateAsync(values))(submitEvent);
            }}
          >
            <FormGroup className="pb-2" cols={2}>
              <FormControl>
                <InputField<RandomPoolFormValues>
                  control={control}
                  name="code"
                  label={t("Code")}
                  placeholder={t("DOT")}
                  description={t("A short identifier that must be unique across your pools; it is saved in upper case.")}
                  rules={{ required: true }}
                />
              </FormControl>
              <FormControl>
                <InputField<RandomPoolFormValues>
                  control={control}
                  name="name"
                  label={t("Name")}
                  placeholder={t("e.g. DOT safety-sensitive drivers")}
                  description={t("How the pool is referred to on rounds and reports.")}
                  rules={{ required: true }}
                />
              </FormControl>
              <FormControl cols="full">
                <TextareaField<RandomPoolFormValues>
                  control={control}
                  name="description"
                  label={t("Description")}
                  placeholder={t("e.g. Every CDL holder who drives for the company")}
                  description={t("Optional notes on who the pool covers and why.")}
                  maxLength={2000}
                />
              </FormControl>
              <FormControl>
                <SelectField<RandomPoolFormValues>
                  control={control}
                  name="status"
                  label={t("Status")}
                  options={statusChoices}
                  placeholder={t("Pick a status")}
                  description={t("Whether the pool is in use; an inactive pool stays on record with its past rounds.")}
                  rules={{ required: true }}
                />
              </FormControl>
              <FormControl>
                <SelectField<RandomPoolFormValues>
                  control={control}
                  name="period"
                  label={t("Draw every")}
                  options={PERIOD_OPTIONS}
                  placeholder={t("Pick a period")}
                  description={t("How often a round is drawn; each round takes its share of the annual rate.")}
                  rules={{ required: true }}
                />
              </FormControl>
              <FormControl>
                <NumberField<RandomPoolFormValues>
                  control={control}
                  name="drugRatePercent"
                  label={t("Drug rate (% a year)")}
                  placeholder="50"
                  description={`Annual rate as a percentage of the pool; FMCSA requires at least ${DOT_MINIMUM_DRUG_RATE}% for drugs (49 CFR 382.305).`}
                  rules={{ required: true }}
                />
              </FormControl>
              <FormControl>
                <NumberField<RandomPoolFormValues>
                  control={control}
                  name="alcoholRatePercent"
                  label={t("Alcohol rate (% a year)")}
                  placeholder="10"
                  description={`Annual rate as a percentage of the pool; FMCSA requires at least ${DOT_MINIMUM_ALCOHOL_RATE}% for alcohol (49 CFR 382.305).`}
                  rules={{ required: true }}
                />
              </FormControl>
              <FormControl cols="full">
                <MultiCheckboxField<RandomPoolFormValues, DriverType>
                  control={control}
                  name="includedDriverTypes"
                  label={t("Driver types")}
                  options={driverTypeChoices}
                  description={t("Leave every box clear to include all safety-sensitive drivers.")}
                />
              </FormControl>
              <FormControl cols="full">
                <SwitchField<RandomPoolFormValues>
                  control={control}
                  name="isDefault"
                  label={t("Default pool")}
                  description={t("The pool a draw runs against when none is named; turning this on takes the default off whichever pool had it.")}
                />
              </FormControl>
              <FormControl cols="full">
                <Alert variant={belowMinimum ? "destructive" : "default"}>
                  <AlertDescription>
                    {belowMinimum
                      ? t("Below the FMCSA minimums of {0}% drug and {1}% alcohol (49 CFR 382.305). This pool is usable but is not evidence of DOT compliance.", DOT_MINIMUM_DRUG_RATE, DOT_MINIMUM_ALCOHOL_RATE)
                      : t("Over {0} drivers, each round would draw about {1} for drug testing and {2} for alcohol.", EXAMPLE_POOL_SIZE, exampleDrug, exampleAlcohol)}
                  </AlertDescription>
                </Alert>
              </FormControl>
            </FormGroup>
            <DialogFooter>
              <Button type="button" variant="outline" onClick={() => onOpenChange(false)}>
                {t("Cancel")}
              </Button>
              <Button type="submit" isLoading={isPending}>
                {isEdit ? t("Save") : t("Create")}
              </Button>
            </DialogFooter>
          </Form>
        </FormProvider>
      </DialogContent>
    </Dialog>
  );
}
