import { CustomerAutocompleteField } from "@/components/autocomplete-fields";
import { DateField } from "@/components/fields/date-field/date-field";
import { InputField } from "@/components/fields/input-field";
import { NumberField } from "@/components/fields/number-field";
import { SelectField } from "@/components/fields/select-field";
import { SwitchField } from "@/components/fields/switch-field";
import { TextareaField } from "@/components/fields/textarea-field";
import { useApiMutation } from "@/hooks/use-api-mutation";
import { memoBillTypeChoices } from "@/lib/choices";
import { createMemo, type CreatedMemo } from "@/lib/graphql/invoice";
import { invoicePanelPath } from "@/lib/invoice-links";
import { invalidateInvoiceQueries } from "@/lib/queries/invoice";
import { zodResolver } from "@hookform/resolvers/zod";
import { useQueryClient } from "@tanstack/react-query";
import { PlusIcon, Trash01Icon } from "@trenova/shared/components/icons";
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
import { ScrollArea } from "@trenova/shared/components/ui/scroll-area";
import { useT } from "@trenova/shared/i18n/use-t";
import { formatCurrency } from "@trenova/shared/lib/utils";
import {
  MAX_MEMO_LINE_AMOUNT,
  MAX_MEMO_LINES,
  MAX_MEMO_QUANTITY,
  MIN_MEMO_QUANTITY,
  memoFormSchema,
  memoFormTotal,
  type MemoFormValues,
} from "@trenova/shared/types/invoice";
import { useEffect, useMemo } from "react";
import { FormProvider, useFieldArray, useForm, useWatch, type Resolver } from "react-hook-form";
import { useNavigate } from "react-router";

const MEMO_FORM_ID = "memo-form";

type MemoBillType = "CreditMemo" | "DebitMemo";

const BLANK_LINE = { description: "", amount: 0, quantity: 1, accessorialChargeId: null };

function defaultsFor(billType: MemoBillType, customerId: string | undefined): MemoFormValues {
  return {
    customerId: customerId ?? "",
    billType,
    referenceInvoiceId: null,
    reason: "",
    invoiceDate: null,
    memo: "",
    autoPost: false,
    lines: [{ ...BLANK_LINE }],
  };
}

function decimalString(value: number): string {
  return Number.isInteger(value) ? String(value) : value.toFixed(4).replace(/0+$/, "");
}

/**
 * Raises a standalone credit or debit memo for a customer: a goodwill credit,
 * a returned-cheque fee, a correction with no shipment behind it. Crediting a
 * specific invoice is the adjustment engine's job (Adjust Invoice), which
 * checks eligibility and approval; this dialog never references one. The
 * created memo opens in the invoice workspace.
 */
export function MemoDialog({
  open,
  onOpenChange,
  billType,
  customerId,
}: {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  billType: MemoBillType;
  customerId?: string;
}) {
  const t = useT();
  const navigate = useNavigate();
  const queryClient = useQueryClient();

  const form = useForm<MemoFormValues>({
    resolver: zodResolver(memoFormSchema) as Resolver<MemoFormValues>,
    defaultValues: defaultsFor(billType, customerId),
  });
  const { fields, append, remove } = useFieldArray({
    control: form.control,
    name: "lines",
    keyName: "fieldId",
  });

  useEffect(() => {
    if (open) {
      form.reset(defaultsFor(billType, customerId));
    }
  }, [open, billType, customerId, form]);

  const lines = useWatch({ control: form.control, name: "lines" });
  const chosenBillType = useWatch({ control: form.control, name: "billType" });
  const total = formatCurrency(memoFormTotal(lines ?? []), "USD");
  const billTypeOptions = useMemo(
    () => memoBillTypeChoices.map((choice) => ({ value: choice.value, label: t(choice.label) })),
    [t],
  );

  const mutation = useApiMutation<CreatedMemo, MemoFormValues, unknown, MemoFormValues>({
    form,
    resourceName: "memo",
    mutationFn: (values) =>
      createMemo({
        customerId: values.customerId,
        billType: values.billType,
        referenceInvoiceId: values.referenceInvoiceId ?? null,
        reason: values.reason,
        invoiceDate: values.invoiceDate ?? null,
        memo: values.memo ?? "",
        autoPost: values.autoPost,
        lines: values.lines.map((line) => ({
          description: line.description,
          amount: decimalString(line.amount),
          quantity: decimalString(line.quantity),
          accessorialChargeId: line.accessorialChargeId || null,
        })),
      }),
    onSuccess: (memo) => {
      invalidateInvoiceQueries(queryClient);
      onOpenChange(false);
      void navigate(invoicePanelPath(memo.id));
    },
  });

  const submitLabel =
    chosenBillType === "DebitMemo" ? t("Create debit memo") : t("Create credit memo");

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent size="lg" className="max-h-[calc(100dvh-2rem)]">
        <DialogHeader>
          <DialogTitle>
            {chosenBillType === "DebitMemo" ? t("New debit memo") : t("New credit memo")}
          </DialogTitle>
          <DialogDescription>
            {t(
              "A memo with no shipment behind it: a goodwill credit, a returned-cheque fee, a correction the customer agreed to. To credit or rebill an invoice, use Adjust Invoice on that invoice.",
            )}
          </DialogDescription>
        </DialogHeader>

        <FormProvider {...form}>
          <Form id={MEMO_FORM_ID} onSubmit={form.handleSubmit((values) => mutation.mutate(values))}>
            <ScrollArea className="max-h-[60dvh]" viewportClassName="pr-3">
              <FormGroup cols={2}>
                <FormControl>
                  <CustomerAutocompleteField
                    control={form.control}
                    name="customerId"
                    label={t("Customer")}
                    placeholder={t("Select customer")}
                    description={t("Who the memo is issued to.")}
                  />
                </FormControl>
                <FormControl>
                  <SelectField
                    control={form.control}
                    name="billType"
                    label={t("Memo type")}
                    placeholder={t("Select type")}
                    description={t("Credit lowers the balance; debit raises it.")}
                    options={billTypeOptions}
                  />
                </FormControl>
                <FormControl cols="full">
                  <DateField
                    control={form.control}
                    name="invoiceDate"
                    label={t("Memo date")}
                    placeholder={t("Today")}
                    description={t("Leave blank to use today.")}
                    clearable
                  />
                </FormControl>
                <FormControl cols="full">
                  <SwitchField
                    control={form.control}
                    name="autoPost"
                    label={t("Post immediately")}
                    outlined
                    description={t("Post to the ledger on create.")}
                  />
                </FormControl>
                <FormControl cols="full">
                  <TextareaField
                    control={form.control}
                    name="reason"
                    label={t("Reason")}
                    placeholder={t("e.g. Goodwill credit for a late delivery")}
                    description={t("Why the memo is being raised.")}
                    rows={2}
                  />
                </FormControl>
                <FormControl cols="full">
                  <TextareaField
                    control={form.control}
                    name="memo"
                    label={t("Memo text")}
                    placeholder={t("Optional note for the customer")}
                    description={t("Printed on the memo.")}
                    rows={2}
                  />
                </FormControl>
              </FormGroup>

              <div className="mt-4 flex items-center justify-between">
                <p className="text-sm font-medium">{t("Lines")}</p>
                <Button
                  type="button"
                  size="sm"
                  variant="outline"
                  disabled={fields.length >= MAX_MEMO_LINES}
                  onClick={() => append({ ...BLANK_LINE })}
                >
                  <PlusIcon className="size-3.5" />
                  {t("Add line")}
                </Button>
              </div>
              <div className="mt-2 flex flex-col gap-2">
                {fields.map((field, index) => (
                  <div
                    key={field.fieldId}
                    className="grid grid-cols-[minmax(0,1fr)_7rem_5rem_auto] items-start gap-2 rounded-md border p-2"
                  >
                    <InputField
                      control={form.control}
                      name={`lines.${index}.description`}
                      label={t("Description")}
                      placeholder={t("e.g. Returned cheque fee")}
                      description={index === 0 ? t("What this line covers.") : undefined}
                      hideLabel={index > 0}
                      aria-label={t("Description")}
                    />
                    <NumberField
                      control={form.control}
                      name={`lines.${index}.amount`}
                      label={t("Amount")}
                      placeholder={t("0.00")}
                      description={index === 0 ? t("Line total.") : undefined}
                      hideLabel={index > 0}
                      aria-label={t("Amount")}
                      decimalScale={2}
                      fixedDecimalScale
                      max={MAX_MEMO_LINE_AMOUNT}
                    />
                    <NumberField
                      control={form.control}
                      name={`lines.${index}.quantity`}
                      label={t("Qty")}
                      placeholder={t("1")}
                      description={index === 0 ? t("Units.") : undefined}
                      hideLabel={index > 0}
                      aria-label={t("Qty")}
                      decimalScale={4}
                      min={MIN_MEMO_QUANTITY}
                      max={MAX_MEMO_QUANTITY}
                    />
                    <Button
                      type="button"
                      size="icon-sm"
                      variant="ghost"
                      className={index === 0 ? "mt-5" : undefined}
                      aria-label={t("Remove line {0}", index + 1)}
                      disabled={fields.length <= 1}
                      onClick={() => remove(index)}
                    >
                      <Trash01Icon className="size-3.5" />
                    </Button>
                  </div>
                ))}
              </div>
            </ScrollArea>
          </Form>
        </FormProvider>

        <DialogFooter className="sm:items-center sm:justify-between">
          <div className="flex min-w-0 items-baseline gap-2" aria-live="polite">
            <span className="text-muted-foreground shrink-0 text-sm">{t("Total")}</span>
            <span className="min-w-0 truncate text-base font-semibold tabular-nums" title={total}>
              {total}
            </span>
          </div>
          <div className="flex shrink-0 flex-col-reverse gap-2 sm:flex-row">
            <Button type="button" variant="outline" onClick={() => onOpenChange(false)}>
              {t("Cancel")}
            </Button>
            <Button type="submit" form={MEMO_FORM_ID} disabled={mutation.isPending}>
              {submitLabel}
            </Button>
          </div>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
