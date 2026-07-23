import { WorkerAutocompleteField } from "@/components/autocomplete-fields";
import { AutoCompleteDateField } from "@/components/fields/date-field/date-field";
import { FormCreatePanel } from "@/components/form-create-panel";
import { FormEditPanel } from "@/components/form-edit-panel";
import { InputField } from "@/components/fields/input-field";
import { NumberField } from "@/components/fields/number-field";
import { PayCodeSelectField, usePayCodeOptions } from "@/components/fields/pay-code-select-field";
import { SelectField } from "@/components/fields/select-field";
import { SwitchField } from "@/components/fields/switch-field";
import { FormControl, FormGroup } from "@trenova/shared/components/ui/form";
import { recurringDeductionFrequencyChoices, recurringDeductionStatusChoices } from "@/lib/choices";
import {
  createRecurringDeduction,
  updateRecurringDeduction,
  type RecurringDeductionRow,
} from "@/lib/graphql/driver-settlement";
import { getTodayDate } from "@trenova/shared/lib/date";
import type { DataTablePanelProps } from "@trenova/shared/types/data-table";
import {
  recurringDeductionFormSchema,
  type RecurringDeductionFormValues,
} from "@trenova/shared/types/driver-pay";
import { zodResolver } from "@hookform/resolvers/zod";
import { useEffect } from "react";
import { useForm, useFormContext, useWatch, type Control, type Resolver } from "react-hook-form";

function buildDefaults(row?: RecurringDeductionRow | null): RecurringDeductionFormValues {
  if (!row) {
    return {
      workerId: "",
      payCodeId: "",
      escrowContribution: false,
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
    escrowContribution: row.escrowAccountId != null,
    status: row.status,
    frequency: row.frequency,
    description: row.description,
    amount: row.amountMinor / 100,
    totalCap: row.totalCapMinor != null ? row.totalCapMinor / 100 : null,
    startDate: row.startDate,
    endDate: row.endDate ?? null,
  };
}

function toSharedInput(values: RecurringDeductionFormValues) {
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

function useDefaultAmountPrefill(control: Control<RecurringDeductionFormValues>) {
  const { setValue, getValues } = useFormContext<RecurringDeductionFormValues>();
  const payCodeId = useWatch({ control, name: "payCodeId" });
  const { data: options } = usePayCodeOptions("Deduction");

  useEffect(() => {
    if (!payCodeId || getValues("amount") > 0) return;
    const option = (options ?? []).find((code) => code.id === payCodeId);
    if (option?.defaultAmountMinor != null) {
      setValue("amount", option.defaultAmountMinor / 100, { shouldDirty: true });
    }
  }, [payCodeId, options, setValue, getValues]);
}

export function DeductionPanel({
  open,
  onOpenChange,
  mode,
  row,
}: DataTablePanelProps<RecurringDeductionRow>) {
  if (mode === "edit" && row) {
    return <DeductionEditPanel open={open} onOpenChange={onOpenChange} row={row} />;
  }
  return <DeductionCreatePanel open={open} onOpenChange={onOpenChange} />;
}

function DeductionCreatePanel({
  open,
  onOpenChange,
}: {
  open: boolean;
  onOpenChange: (open: boolean) => void;
}) {
  const form = useForm<RecurringDeductionFormValues>({
    resolver: zodResolver(recurringDeductionFormSchema) as Resolver<RecurringDeductionFormValues>,
    defaultValues: buildDefaults(null),
  });

  return (
    <FormCreatePanel<RecurringDeductionFormValues, RecurringDeductionRow>
      open={open}
      onOpenChange={onOpenChange}
      title="Recurring Deduction"
      description="Applied automatically to each qualifying settlement until its end date or cap."
      queryKey="recurring-deduction-list"
      form={form}
      formComponent={<DeductionForm isEdit={false} />}
      mutationFn={async (values) => {
        await createRecurringDeduction({
          ...toSharedInput(values),
          escrowContribution: values.escrowContribution,
        });
        return values;
      }}
    />
  );
}

function DeductionEditPanel({
  open,
  onOpenChange,
  row,
}: {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  row: RecurringDeductionRow;
}) {
  const formRow = { ...row, ...buildDefaults(row) } as unknown as RecurringDeductionRow &
    Record<string, unknown>;
  const form = useForm<RecurringDeductionFormValues>({
    resolver: zodResolver(recurringDeductionFormSchema) as Resolver<RecurringDeductionFormValues>,
    defaultValues: buildDefaults(row),
  });

  return (
    <FormEditPanel<RecurringDeductionFormValues, RecurringDeductionRow & Record<string, unknown>>
      open={open}
      onOpenChange={onOpenChange}
      row={formRow}
      title="Recurring Deduction"
      fieldKey="description"
      queryKey="recurring-deduction-list"
      form={form}
      formComponent={<DeductionForm isEdit />}
      mutationFn={async (values) => {
        await updateRecurringDeduction({
          id: row.id,
          version: row.version,
          status: values.status,
          escrowAccountId: row.escrowAccountId ?? undefined,
          ...toSharedInput(values),
        });
        return values;
      }}
    />
  );
}

function DeductionForm({ isEdit }: { isEdit: boolean }) {
  const { control } = useFormContext<RecurringDeductionFormValues>();
  useDefaultAmountPrefill(control);
  const escrowContribution = useWatch({ control, name: "escrowContribution" });

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
            description="The driver whose settlements this deduction is withheld from."
          />
        </FormControl>
        <FormControl>
          <PayCodeSelectField
            control={control}
            name="payCodeId"
            direction="Deduction"
            description="Deduction code that categorizes the withholding and routes it to the code's GL account when one is mapped."
          />
        </FormControl>
        <FormControl>
          <SelectField
            control={control}
            name="frequency"
            label="Frequency"
            options={recurringDeductionFrequencyChoices}
            rules={{ required: true }}
            description="Every settlement withholds each cycle; monthly withholds only on the first settlement of each month."
          />
        </FormControl>
        {isEdit && (
          <FormControl>
            <SelectField
              control={control}
              name="status"
              label="Status"
              options={recurringDeductionStatusChoices}
              rules={{ required: true }}
              description="Pause to skip upcoming settlements without losing history; completed deductions stop permanently."
            />
          </FormControl>
        )}
        <FormControl className={isEdit ? undefined : "col-span-2"}>
          <InputField
            control={control}
            name="description"
            label="Description"
            placeholder="e.g. Occupational accident insurance"
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
            description="The amount withheld each time the deduction applies to a settlement."
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
            description="Deduction stops automatically once this lifetime total is reached."
          />
        </FormControl>
        <FormControl>
          <AutoCompleteDateField
            control={control}
            name="startDate"
            label="Start Date"
            rules={{ required: true }}
            description="The deduction begins applying to settlements whose period ends after this date."
          />
        </FormControl>
        <FormControl>
          <AutoCompleteDateField
            control={control}
            name="endDate"
            label="End Date"
            description="Optional last day the deduction applies; leave blank for open-ended."
          />
        </FormControl>
        <FormControl className="col-span-2">
          <SwitchField
            control={control}
            name="escrowContribution"
            label="Contribute to Escrow Account"
            disabled={isEdit}
            description="Routes the withheld amount into the driver's active escrow account and stops automatically at the account's funding target."
            position="left"
          />
        </FormControl>
      </FormGroup>
      {escrowContribution && !isEdit && (
        <p className="text-xs text-muted-foreground">
          The deduction links to the driver&apos;s active escrow account when saved. Open an escrow
          account for the driver first if one doesn&apos;t exist.
        </p>
      )}
    </div>
  );
}
