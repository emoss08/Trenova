import { WorkerAutocompleteField } from "@/components/autocomplete-fields";
import { AutoCompleteDateField } from "@/components/fields/date-field/date-field";
import { FormCreatePanel } from "@/components/form-create-panel";
import { FormEditPanel } from "@/components/form-edit-panel";
import { InputField } from "@/components/fields/input-field";
import { NumberField } from "@/components/fields/number-field";
import { PayCodeSelectField, usePayCodeOptions } from "@/components/fields/pay-code-select-field";
import { SelectField } from "@/components/fields/select-field";
import { FormControl, FormGroup } from "@/components/ui/form";
import { recurringEarningFrequencyChoices, recurringEarningStatusChoices } from "@/lib/choices";
import {
  createRecurringEarning,
  updateRecurringEarning,
  type RecurringEarningRow,
} from "@/lib/graphql/driver-settlement";
import { getTodayDate } from "@/lib/date";
import type { DataTablePanelProps } from "@/types/data-table";
import { recurringEarningFormSchema, type RecurringEarningFormValues } from "@/types/driver-pay";
import { zodResolver } from "@hookform/resolvers/zod";
import { useEffect } from "react";
import { useForm, useFormContext, useWatch, type Control, type Resolver } from "react-hook-form";

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
  const form = useForm<RecurringEarningFormValues>({
    resolver: zodResolver(recurringEarningFormSchema) as Resolver<RecurringEarningFormValues>,
    defaultValues: buildDefaults(null),
  });

  return (
    <FormCreatePanel<RecurringEarningFormValues, RecurringEarningRow>
      open={open}
      onOpenChange={onOpenChange}
      title="Recurring Earning"
      description="Added automatically to each qualifying settlement until its end date or cap."
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
      title="Recurring Earning"
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

function useDefaultAmountPrefill(control: Control<RecurringEarningFormValues>) {
  const { setValue, getValues } = useFormContext<RecurringEarningFormValues>();
  const payCodeId = useWatch({ control, name: "payCodeId" });
  const { data: options } = usePayCodeOptions("Earning");

  useEffect(() => {
    if (!payCodeId || getValues("amount") > 0) return;
    const option = (options ?? []).find((code) => code.id === payCodeId);
    if (option?.defaultAmountMinor != null) {
      setValue("amount", option.defaultAmountMinor / 100, { shouldDirty: true });
    }
  }, [payCodeId, options, setValue, getValues]);
}

function EarningForm({ isEdit }: { isEdit: boolean }) {
  const { control } = useFormContext<RecurringEarningFormValues>();
  useDefaultAmountPrefill(control);
  const payCodeId = useWatch({ control, name: "payCodeId" });
  const { data: options } = usePayCodeOptions("Earning");
  const selectedCode = (options ?? []).find((code) => code.id === payCodeId);

  return (
    <div className="flex flex-col gap-4">
      <FormGroup cols={2}>
        <FormControl className="col-span-2">
          <WorkerAutocompleteField
            control={control}
            name="workerId"
            label="Driver"
            placeholder="Select driver"
            rules={{ required: true }}
            description="The driver whose settlements this earning is added to."
          />
        </FormControl>
        <FormControl>
          <PayCodeSelectField
            control={control}
            name="payCodeId"
            direction="Earning"
            description="Earning code that categorizes the pay and routes it to the code's GL account when one is mapped."
          />
        </FormControl>
        <FormControl>
          <SelectField
            control={control}
            name="frequency"
            label="Frequency"
            options={recurringEarningFrequencyChoices}
            rules={{ required: true }}
            description="Every settlement pays each cycle; monthly pays only on the first settlement of each month."
          />
        </FormControl>
        {isEdit && (
          <FormControl>
            <SelectField
              control={control}
              name="status"
              label="Status"
              options={recurringEarningStatusChoices}
              rules={{ required: true }}
              description="Pause to skip upcoming settlements without losing history; completed earnings stop permanently."
            />
          </FormControl>
        )}
        <FormControl className={isEdit ? undefined : "col-span-2"}>
          <InputField
            control={control}
            name="description"
            label="Description"
            placeholder="e.g. OTR per diem — IRS substantiated M&IE"
            rules={{ required: true }}
            description="Shown verbatim on the driver's settlement statement, so make it recognizable."
          />
        </FormControl>
        <FormControl>
          <NumberField
            control={control}
            name="amount"
            label="Amount per Application"
            decimalScale={2}
            fixedDecimalScale
            sideText="USD"
            rules={{ required: true }}
            description="The amount added each time the earning applies to a settlement."
          />
        </FormControl>
        <FormControl>
          <NumberField
            control={control}
            name="totalCap"
            label="Total Cap"
            decimalScale={2}
            fixedDecimalScale
            sideText="USD"
            description="Earning stops automatically once this lifetime total is reached."
          />
        </FormControl>
        <FormControl>
          <AutoCompleteDateField
            control={control}
            name="startDate"
            label="Start Date"
            rules={{ required: true }}
            description="The earning begins applying to settlements whose period ends after this date."
          />
        </FormControl>
        <FormControl>
          <AutoCompleteDateField
            control={control}
            name="endDate"
            label="End Date"
            description="Optional last day the earning applies; leave blank for open-ended."
          />
        </FormControl>
      </FormGroup>
      {selectedCode != null && !selectedCode.taxable && (
        <p className="text-xs text-muted-foreground">
          This code is non-taxable — amounts post to the settlement as reimbursements, are excluded
          from guaranteed-minimum checks, and post to the code&apos;s GL account (or the driver
          reimbursement account) instead of wages expense.
        </p>
      )}
    </div>
  );
}
