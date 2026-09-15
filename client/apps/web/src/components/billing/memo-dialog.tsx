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
import { memoFormSchema, memoFormTotal, type MemoFormValues } from "@trenova/shared/types/invoice";
import { PlusIcon, Trash2Icon } from "lucide-react";
import { useEffect, useMemo } from "react";
import { FormProvider, useFieldArray, useForm, useWatch, type Resolver } from "react-hook-form";
import { useNavigate } from "react-router";
import { toast } from "sonner";

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

  const lines = useWatch({ control: form.control, name: "lines" }) ?? [];
  const chosenBillType = useWatch({ control: form.control, name: "billType" });
  const total = useMemo(() => memoFormTotal(lines), [lines]);
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
      toast.success(t("{0} created", memo.number));
      onOpenChange(false);
      void navigate(invoicePanelPath(memo.id));
    },
  });

  const submitLabel =
    chosenBillType === "DebitMemo" ? t("Create debit memo") : t("Create credit memo");

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-h-[calc(100dvh-2rem)] sm:max-w-[680px]">
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
                  />
                </FormControl>
                <FormControl>
                  <SelectField
                    control={form.control}
                    name="billType"
                    label={t("Memo type")}
                    options={billTypeOptions}
                  />
                </FormControl>
                <FormControl>
                  <DateField
                    control={form.control}
                    name="invoiceDate"
                    label={t("Memo date")}
                    placeholder={t("Today")}
                    clearable
                  />
                </FormControl>
                <FormControl>
                  <SwitchField
                    control={form.control}
                    name="autoPost"
                    label={t("Post immediately")}
                    description={t("Post the memo to the ledger in the same step.")}
                  />
                </FormControl>
                <FormControl cols="full">
                  <TextareaField
                    control={form.control}
                    name="reason"
                    label={t("Reason")}
                    placeholder={t("Why the memo is being raised")}
                    rows={2}
                  />
                </FormControl>
                <FormControl cols="full">
                  <TextareaField
                    control={form.control}
                    name="memo"
                    label={t("Memo text")}
                    placeholder={t("Printed on the memo, optional")}
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
                      hideLabel={index > 0}
                      aria-label={t("Description")}
                    />
                    <NumberField
                      control={form.control}
                      name={`lines.${index}.amount`}
                      label={t("Amount")}
                      aria-label={t("Amount")}
                      decimalScale={2}
                      fixedDecimalScale
                    />
                    <NumberField
                      control={form.control}
                      name={`lines.${index}.quantity`}
                      label={t("Qty")}
                      aria-label={t("Qty")}
                      decimalScale={2}
                    />
                    <Button
                      type="button"
                      size="icon-sm"
                      variant="ghost"
                      className="mt-5"
                      aria-label={t("Remove line {0}", index + 1)}
                      disabled={fields.length <= 1}
                      onClick={() => remove(index)}
                    >
                      <Trash2Icon className="size-3.5" />
                    </Button>
                  </div>
                ))}
              </div>
              <div className="mt-3 flex items-center justify-end gap-3 border-t pt-3">
                <span className="text-muted-foreground text-sm">{t("Total")}</span>
                <span className="text-base font-semibold tabular-nums">
                  {formatCurrency(total, "USD")}
                </span>
              </div>
            </ScrollArea>
          </Form>
        </FormProvider>

        <DialogFooter>
          <Button type="button" variant="outline" onClick={() => onOpenChange(false)}>
            {t("Cancel")}
          </Button>
          <Button type="submit" form={MEMO_FORM_ID} disabled={mutation.isPending}>
            {submitLabel}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
