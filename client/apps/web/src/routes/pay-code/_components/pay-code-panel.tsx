import { GLAccountAutocompleteField } from "@/components/autocomplete-fields";
import { FormCreatePanel } from "@/components/form-create-panel";
import { FormEditPanel } from "@/components/form-edit-panel";
import { InputField } from "@/components/fields/input-field";
import { NumberField } from "@/components/fields/number-field";
import { SelectField } from "@/components/fields/select-field";
import { SwitchField } from "@/components/fields/switch-field";
import { FormControl, FormGroup } from "@trenova/shared/components/ui/form";
import { payCodeDirectionChoices, statusChoices } from "@/lib/choices";
import { createPayCode, updatePayCode, type PayCodeRow } from "@/lib/graphql/driver-settlement";
import type { DataTablePanelProps } from "@trenova/shared/types/data-table";
import { payCodeFormSchema, type PayCodeFormValues } from "@trenova/shared/types/driver-pay";
import { zodResolver } from "@hookform/resolvers/zod";
import { useForm, useFormContext, useWatch, type Resolver } from "react-hook-form";

function buildDefaults(row?: PayCodeRow | null): PayCodeFormValues {
  if (!row) {
    return {
      direction: "Earning",
      code: "",
      name: "",
      description: null,
      status: "Active",
      taxable: true,
      countsTowardGuarantee: true,
      glAccountId: null,
      defaultAmount: null,
    };
  }
  return {
    direction: row.direction,
    code: row.code,
    name: row.name,
    description: row.description || null,
    status: row.status as "Active" | "Inactive",
    taxable: row.taxable,
    countsTowardGuarantee: row.countsTowardGuarantee,
    glAccountId: row.glAccountId ?? null,
    defaultAmount: row.defaultAmountMinor != null ? row.defaultAmountMinor / 100 : null,
  };
}

function toSharedInput(values: PayCodeFormValues) {
  return {
    code: values.code.toUpperCase(),
    name: values.name,
    description: values.description ?? undefined,
    taxable: values.taxable,
    countsTowardGuarantee: values.countsTowardGuarantee,
    glAccountId: values.glAccountId ?? undefined,
    defaultAmountMinor:
      values.defaultAmount != null ? Math.round(values.defaultAmount * 100) : undefined,
  };
}

export function PayCodePanel({ open, onOpenChange, mode, row }: DataTablePanelProps<PayCodeRow>) {
  if (mode === "edit" && row) {
    return <PayCodeEditPanel open={open} onOpenChange={onOpenChange} row={row} />;
  }
  return <PayCodeCreatePanel open={open} onOpenChange={onOpenChange} />;
}

function PayCodeCreatePanel({
  open,
  onOpenChange,
}: {
  open: boolean;
  onOpenChange: (open: boolean) => void;
}) {
  const form = useForm<PayCodeFormValues>({
    resolver: zodResolver(payCodeFormSchema) as Resolver<PayCodeFormValues>,
    defaultValues: buildDefaults(null),
  });

  return (
    <FormCreatePanel<PayCodeFormValues, PayCodeRow>
      open={open}
      onOpenChange={onOpenChange}
      title="Pay Code"
      description="Define a carrier-specific earning or deduction code, its settlement behavior, and where it posts in the GL."
      queryKey="pay-code-list"
      form={form}
      formComponent={<PayCodeForm isEdit={false} isSystem={false} />}
      mutationFn={async (values) => {
        await createPayCode({
          direction: values.direction,
          ...toSharedInput(values),
        });
        return values;
      }}
    />
  );
}

function PayCodeEditPanel({
  open,
  onOpenChange,
  row,
}: {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  row: PayCodeRow;
}) {
  const formRow = { ...row, ...buildDefaults(row) } as unknown as PayCodeRow &
    Record<string, unknown>;
  const form = useForm<PayCodeFormValues>({
    resolver: zodResolver(payCodeFormSchema) as Resolver<PayCodeFormValues>,
    defaultValues: buildDefaults(row),
  });

  return (
    <FormEditPanel<PayCodeFormValues, PayCodeRow & Record<string, unknown>>
      open={open}
      onOpenChange={onOpenChange}
      row={formRow}
      title="Pay Code"
      fieldKey="code"
      queryKey="pay-code-list"
      form={form}
      formComponent={<PayCodeForm isEdit isSystem={row.isSystem} />}
      mutationFn={async (values) => {
        await updatePayCode({
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

function PayCodeForm({ isEdit, isSystem }: { isEdit: boolean; isSystem: boolean }) {
  const { control } = useFormContext<PayCodeFormValues>();
  const direction = useWatch({ control, name: "direction" });

  return (
    <div className="flex flex-col gap-4">
      <FormGroup cols={2}>
        <FormControl>
          <SelectField
            control={control}
            name="direction"
            label="Direction"
            options={payCodeDirectionChoices}
            rules={{ required: true }}
            isReadOnly={isEdit}
            description="Earning codes add pay to settlements; deduction codes withhold it. Fixed after creation."
          />
        </FormControl>
        <FormControl>
          <InputField
            control={control}
            name="code"
            label="Code"
            placeholder="e.g. CHAINPAY"
            rules={{ required: true }}
            disabled={isSystem}
            description="Short unique identifier shown on statements and reports; uppercase letters, digits, dashes, or underscores."
          />
        </FormControl>
        <FormControl className={isEdit ? undefined : "col-span-2"}>
          <InputField
            control={control}
            name="name"
            label="Name"
            placeholder="e.g. Chain-Up Pay"
            rules={{ required: true }}
            description="Human-readable label displayed next to the code throughout the app."
          />
        </FormControl>
        {isEdit && (
          <FormControl>
            <SelectField
              control={control}
              name="status"
              label="Status"
              options={statusChoices}
              rules={{ required: true }}
              description="Inactive codes stay on historical records but disappear from new-entry dropdowns."
            />
          </FormControl>
        )}
        <FormControl className="col-span-2">
          <InputField
            control={control}
            name="description"
            label="Description"
            placeholder="Optional note about when this code applies"
            description="Optional internal note explaining when and how the code should be used."
          />
        </FormControl>
        <FormControl className="col-span-2">
          <GLAccountAutocompleteField
            control={control}
            name="glAccountId"
            label="GL Account"
            placeholder="Select GL account"
            description="Settlement lines carrying this code post to this account; leave blank to use the accounting control defaults."
            clearable
          />
        </FormControl>
        <FormControl>
          <NumberField
            control={control}
            name="defaultAmount"
            label="Default Amount"
            decimalScale={2}
            fixedDecimalScale
            sideText="USD"
            description="Prefills the amount when creating recurring earnings or deductions with this code."
          />
        </FormControl>
        {direction === "Earning" && (
          <>
            <FormControl>
              <SwitchField
                control={control}
                name="taxable"
                label="Taxable"
                description="Taxable amounts post as earnings; non-taxable amounts (per diem, stipends) post as reimbursements."
                position="left"
              />
            </FormControl>
            <FormControl className="col-span-2">
              <SwitchField
                control={control}
                name="countsTowardGuarantee"
                label="Counts Toward Guaranteed Minimum"
                description="When off, pay under this code is ignored when checking a driver's guaranteed period minimum."
                position="left"
              />
            </FormControl>
          </>
        )}
      </FormGroup>
    </div>
  );
}
