import { useT } from "@trenova/shared/i18n/use-t";
import {
  PayCodeAutocompleteField,
  WorkerAutocompleteField,
} from "@/components/autocomplete-fields";
import { AutoCompleteDateField } from "@/components/fields/date-field/date-field";
import { FormCreatePanel } from "@/components/form-create-panel";
import { FormEditPanel } from "@/components/form-edit-panel";
import { InputField } from "@/components/fields/input-field";
import { NumberField } from "@/components/fields/number-field";
import { SelectField } from "@/components/fields/select-field";
import { useSelectOption } from "@/hooks/use-select-option";
import type { SelectOption } from "@/lib/graphql/select-options";
import { selectOptionMetaBoolean, selectOptionMetaNumber } from "@/lib/select-option-meta";
import { FormControl, FormGroup } from "@trenova/shared/components/ui/form";
import { recurringEarningFrequencyChoices, recurringEarningStatusChoices } from "@/lib/choices";
import {
  createRecurringEarning,
  updateRecurringEarning,
  type RecurringEarningRow,
} from "@/lib/graphql/driver-settlement";
import { getTodayDate } from "@trenova/shared/lib/date";
import type { DataTablePanelProps } from "@trenova/shared/types/data-table";
import {
  recurringEarningFormSchema,
  type RecurringEarningFormValues,
} from "@trenova/shared/types/driver-pay";
import { zodResolver } from "@hookform/resolvers/zod";
import { useEffect } from "react";
import { useForm, useFormContext, useWatch, type Resolver } from "react-hook-form";

function buildDefaults(row?: RecurringEarningRow | null): RecurringEarningFormValues {
  if (!row) {
    return {
      workerId: "",
      payCodeId: "",
      status: "Active",
      frequency: "EverySettlement",
      description: "",
      amount: 0,
      totalCap: null,
      startDate: getTodayDate(),
      endDate: null,
    };
  }
  return {
    workerId: row.workerId,
    payCodeId: row.payCodeId,
    status: row.status,
    frequency: row.frequency,
    description: row.description,
    amount: row.amountMinor / 100,
    totalCap: row.totalCapMinor != null ? row.totalCapMinor / 100 : null,
    startDate: row.startDate,
    endDate: row.endDate ?? null,
  };
}

function toSharedInput(values: RecurringEarningFormValues) {
  return {
    workerId: values.workerId,
    payCodeId: values.payCodeId,
    frequency: values.frequency,
    description: values.description,
    amountMinor: Math.round(values.amount * 100),
    totalCapMinor: values.totalCap != null ? Math.round(values.totalCap * 100) : undefined,
    startDate: values.startDate,
    endDate: values.endDate ?? undefined,
  };
}

export function EarningPanel({
  open,
  onOpenChange,
  mode,
  row,
}: DataTablePanelProps<RecurringEarningRow>) {
  if (mode === "edit" && row) {
    return <EarningEditPanel open={open} onOpenChange={onOpenChange} row={row} />;
  }
  return <EarningCreatePanel open={open} onOpenChange={onOpenChange} />;
}

function EarningCreatePanel({
  open,
  onOpenChange,
}: {
  open: boolean;
  onOpenChange: (open: boolean) => void;
}) {
  const t = useT();

  const form = useForm<RecurringEarningFormValues>({
    resolver: zodResolver(recurringEarningFormSchema) as Resolver<RecurringEarningFormValues>,
    defaultValues: buildDefaults(null),
  });

  return (
    <FormCreatePanel<RecurringEarningFormValues, RecurringEarningRow>
      open={open}
      onOpenChange={onOpenChange}
      title={t("Recurring Earning")}
      description={t(
        "Added automatically to each qualifying settlement until its end date or cap.",
      )}
      queryKey="recurring-earning-list"
      form={form}
      formComponent={<EarningForm isEdit={false} />}
      mutationFn={async (values) => {
        await createRecurringEarning(toSharedInput(values));
        return values;
      }}
    />
  );
}

function EarningEditPanel({
  open,
  onOpenChange,
  row,
}: {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  row: RecurringEarningRow;
}) {
  const t = useT();

  const formRow = { ...row, ...buildDefaults(row) } as unknown as RecurringEarningRow &
    Record<string, unknown>;
  const form = useForm<RecurringEarningFormValues>({
    resolver: zodResolver(recurringEarningFormSchema) as Resolver<RecurringEarningFormValues>,
    defaultValues: buildDefaults(row),
  });

  return (
    <FormEditPanel<RecurringEarningFormValues, RecurringEarningRow & Record<string, unknown>>
      open={open}
      onOpenChange={onOpenChange}
      row={formRow}
      title={t("Recurring Earning")}
      fieldKey="description"
      queryKey="recurring-earning-list"
      form={form}
      formComponent={<EarningForm isEdit />}
      mutationFn={async (values) => {
        await updateRecurringEarning({
          id: row.id,
          version: row.version,
          status: values.status,
          ...toSharedInput(values),
        });
        return values;
      }}
    />
  );
}

function useDefaultAmountPrefill(selectedCode: SelectOption | null) {
  const { setValue, getValues } = useFormContext<RecurringEarningFormValues>();

  useEffect(() => {
    if (!selectedCode || getValues("amount") > 0) return;
    const defaultAmountMinor = selectOptionMetaNumber(selectedCode, "defaultAmountMinor");
    if (defaultAmountMinor != null) {
      setValue("amount", defaultAmountMinor / 100, { shouldDirty: true });
    }
  }, [selectedCode, setValue, getValues]);
}

function EarningForm({ isEdit }: { isEdit: boolean }) {
  const t = useT();

  const { control } = useFormContext<RecurringEarningFormValues>();
  const payCodeId = useWatch({ control, name: "payCodeId" });
  const { option: selectedCode } = useSelectOption("PAY_CODE", payCodeId);
  useDefaultAmountPrefill(selectedCode);

  return (
    <div className="flex flex-col gap-4">
      <FormGroup cols={2}>
        <FormControl className="col-span-2">
          <WorkerAutocompleteField
            control={control}
            name="workerId"
            label={t("Driver")}
            placeholder={t("Select driver")}
            rules={{ required: true }}
            description={t("The driver whose settlements this earning is added to.")}
          />
        </FormControl>
        <FormControl>
          <PayCodeAutocompleteField
            control={control}
            name="payCodeId"
            label={t("Pay Code")}
            placeholder={t("Select pay code")}
            direction="Earning"
            rules={{ required: true }}
            description={t(
              "Earning code that categorizes the pay and routes it to the code's GL account when one is mapped.",
            )}
          />
        </FormControl>
        <FormControl>
          <SelectField
            control={control}
            name="frequency"
            label={t("Frequency")}
            options={recurringEarningFrequencyChoices}
            rules={{ required: true }}
            description={t(
              "Every settlement pays each cycle; monthly pays only on the first settlement of each month.",
            )}
          />
        </FormControl>
        {isEdit && (
          <FormControl>
            <SelectField
              control={control}
              name="status"
              label={t("Status")}
              options={recurringEarningStatusChoices}
              rules={{ required: true }}
              description={t(
                "Pause to skip upcoming settlements without losing history; completed earnings stop permanently.",
              )}
            />
          </FormControl>
        )}
        <FormControl className={isEdit ? undefined : "col-span-2"}>
          <InputField
            control={control}
            name="description"
            label={t("Description")}
            placeholder={t("e.g. OTR per diem — IRS substantiated M&IE")}
            rules={{ required: true }}
            description={t(
              "Shown verbatim on the driver's settlement statement, so make it recognizable.",
            )}
          />
        </FormControl>
        <FormControl>
          <NumberField
            control={control}
            name="amount"
            label={t("Amount per Application")}
            decimalScale={2}
            fixedDecimalScale
            sideText={t("USD")}
            rules={{ required: true }}
            description={t("The amount added each time the earning applies to a settlement.")}
          />
        </FormControl>
        <FormControl>
          <NumberField
            control={control}
            name="totalCap"
            label={t("Total Cap")}
            decimalScale={2}
            fixedDecimalScale
            sideText={t("USD")}
            description={t("Earning stops automatically once this lifetime total is reached.")}
          />
        </FormControl>
        <FormControl>
          <AutoCompleteDateField
            control={control}
            name="startDate"
            label={t("Start Date")}
            rules={{ required: true }}
            description={t(
              "The earning begins applying to settlements whose period ends after this date.",
            )}
          />
        </FormControl>
        <FormControl>
          <AutoCompleteDateField
            control={control}
            name="endDate"
            label={t("End Date")}
            description={t("Optional last day the earning applies; leave blank for open-ended.")}
          />
        </FormControl>
      </FormGroup>
      {selectedCode != null && !selectOptionMetaBoolean(selectedCode, "taxable") && (
        <p className="text-muted-foreground text-xs">
          {t(
            "This code is non-taxable — amounts post to the settlement as reimbursements, are excluded from guaranteed-minimum checks, and post to the code's GL account (or the driver reimbursement account) instead of wages expense.",
          )}
        </p>
      )}
    </div>
  );
}
