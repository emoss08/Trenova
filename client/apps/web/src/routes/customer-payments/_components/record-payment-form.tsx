import { CustomerAutocompleteField } from "@/components/autocomplete-fields";
import { AutoCompleteDateField } from "@/components/fields/date-field/date-field";
import { InputField } from "@/components/fields/input-field";
import { NumberField } from "@/components/fields/number-field";
import { SelectField } from "@/components/fields/select-field";
import { TextareaField } from "@/components/fields/textarea-field";
import { FormControl, FormGroup } from "@/components/ui/form";
import {
  allocateBudget,
  openItemToApplicationRow,
  toMinor,
} from "@/lib/cash-application";
import { paymentMethodChoices } from "@/lib/choices";
import { queries } from "@/lib/queries";
import type { RecordPaymentFormValues } from "@/types/customer-payment";
import { useQuery } from "@tanstack/react-query";
import { useEffect, useRef } from "react";
import { useFormContext, useWatch } from "react-hook-form";
import { CashApplicationEditor } from "./cash-application-editor";

export function RecordPaymentForm({
  prefilledInvoiceIds,
}: {
  prefilledInvoiceIds: string[];
}) {
  const { control, setValue, getValues } = useFormContext<RecordPaymentFormValues>();
  const customerId = useWatch({ control, name: "customerId" });
  const amount = useWatch({ control, name: "amount" });
  const budgetMinor = toMinor(amount ?? 0);

  const { data: openItems, isLoading: itemsLoading } = useQuery({
    ...queries.ar.openItems({ customerId }),
    enabled: Boolean(customerId),
  });

  const initializedForCustomer = useRef<string | null>(null);

  useEffect(() => {
    if (!customerId) {
      initializedForCustomer.current = null;
      setValue("applications", []);
      return;
    }
    if (!openItems || initializedForCustomer.current === customerId) return;

    initializedForCustomer.current = customerId;
    setValue(
      "applications",
      openItems.map((item) =>
        openItemToApplicationRow(item, prefilledInvoiceIds.includes(item.invoiceId)),
      ),
    );
  }, [customerId, openItems, prefilledInvoiceIds, setValue]);

  const handleAutoApply = () => {
    setValue("applications", allocateBudget(getValues("applications"), budgetMinor), {
      shouldDirty: true,
    });
  };

  return (
    <div className="flex flex-col gap-4">
      <FormGroup cols={2}>
        <FormControl className="col-span-2">
          <CustomerAutocompleteField
            control={control}
            name="customerId"
            label="Customer"
            placeholder="Select customer"
            rules={{ required: true }}
          />
        </FormControl>
        <FormControl>
          <NumberField
            control={control}
            name="amount"
            label="Payment Amount"
            placeholder="0.00"
            rules={{ required: true }}
            decimalScale={2}
            fixedDecimalScale
            sideText="USD"
          />
        </FormControl>
        <FormControl>
          <SelectField
            control={control}
            name="paymentMethod"
            label="Payment Method"
            rules={{ required: true }}
            options={paymentMethodChoices}
          />
        </FormControl>
        <FormControl>
          <AutoCompleteDateField
            control={control}
            name="paymentDate"
            label="Payment Date"
            rules={{ required: "Payment date is required" }}
            placeholder="Select date"
            description="The date the funds were received."
          />
        </FormControl>
        <FormControl>
          <AutoCompleteDateField
            control={control}
            name="accountingDate"
            label="Accounting Date"
            rules={{ required: "Accounting date is required" }}
            placeholder="Select date"
            description="The GL date. It must fall within an open fiscal period."
          />
        </FormControl>
        <FormControl>
          <InputField
            control={control}
            name="referenceNumber"
            label="Reference Number"
            placeholder="Check # / ACH trace"
            maxLength={100}
          />
        </FormControl>
        <FormControl>
          <TextareaField control={control} name="memo" label="Memo" placeholder="Optional note" />
        </FormControl>
      </FormGroup>

      {customerId ? (
        <CashApplicationEditor
          budgetMinor={budgetMinor}
          budgetLabel="Payment"
          onAutoApply={handleAutoApply}
          isLoadingItems={itemsLoading}
          emptyMessage="This customer has no open invoices — the full amount will post as unapplied cash."
        />
      ) : (
        <div className="flex h-24 items-center justify-center rounded-md border border-dashed text-sm text-muted-foreground">
          Select a customer to see their open invoices
        </div>
      )}
    </div>
  );
}
