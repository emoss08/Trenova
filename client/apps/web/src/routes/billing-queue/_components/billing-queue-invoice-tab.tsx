import { useT } from "@trenova/shared/i18n/use-t";
import { DatePickerField } from "@/components/fields/date-picker-field";
import { InputField } from "@/components/fields/input-field";
import { SelectField } from "@/components/fields/select-field";
import { paymentTermLabel } from "@/lib/billing-queue-checks";
import { paymentTermChoices } from "@/lib/choices";
import { Form } from "@trenova/shared/components/ui/form";
import { cn } from "@trenova/shared/lib/utils";
import { useDebounce } from "@trenova/shared/hooks/use-debounce";
import { formatUnixDate } from "@trenova/shared/lib/date";
import type { BillingQueueItem } from "@trenova/shared/types/billing-queue";
import { zodResolver } from "@hookform/resolvers/zod";
import { useEffect } from "react";
import { useForm, useWatch } from "react-hook-form";
import { z } from "zod";
import { useInvoiceDetails } from "./use-invoice-details";

const CUSTOMER_DEFAULT = "";
const AUTOSAVE_MS = 700;

const invoiceDetailsFormSchema = z.object({
  poNumber: z.string().trim().max(100),
  paymentTerm: z.string(),
  invoiceDate: z.number().nullable(),
  invoiceMemo: z.string().trim().max(1000),
});
type InvoiceDetailsForm = z.infer<typeof invoiceDetailsFormSchema>;

function formValuesOf(item: BillingQueueItem): InvoiceDetailsForm {
  return {
    poNumber: item.poNumber ?? "",
    paymentTerm: item.paymentTerm ?? CUSTOMER_DEFAULT,
    invoiceDate: item.invoiceDate ?? null,
    invoiceMemo: item.invoiceMemo ?? "",
  };
}

/**
 * What the invoice will say beyond its charges: the customer's PO, the terms and
 * invoice date when they differ from the customer's defaults, and a note printed on
 * it. Each change saves on its own after a pause, so there is no save button to miss.
 */
export function BillingQueueInvoiceTab({ item }: { item: BillingQueueItem }) {
  const t = useT();
  const { save, isSaving, editable } = useInvoiceDetails(item);
  const customerTerm = item.shipment?.customer?.billingProfile?.paymentTerm;

  const form = useForm<InvoiceDetailsForm>({
    resolver: zodResolver(invoiceDetailsFormSchema),
    defaultValues: formValuesOf(item),
  });
  const values = useWatch({ control: form.control });
  const settled = useDebounce(values, AUTOSAVE_MS);

  useEffect(() => {
    if (!editable || !form.formState.isDirty) {
      return;
    }
    void form.handleSubmit(async (next) => {
      await save({
        poNumber: next.poNumber,
        paymentTerm: next.paymentTerm,
        invoiceDate: next.invoiceDate,
        invoiceMemo: next.invoiceMemo,
      });
      form.reset(next, { keepValues: true });
    })();
  }, [settled, editable, form, save]);

  const termOptions = [
    {
      value: CUSTOMER_DEFAULT,
      label: customerTerm
        ? t("Customer default · {0}", paymentTermLabel(customerTerm, t))
        : t("Customer default"),
    },
    ...paymentTermChoices.map((choice) => ({ value: choice.value, label: t(choice.label) })),
  ];

  const termCode = values.paymentTerm || item.paymentTerm || customerTerm || "";
  const netDays =
    termCode === "DueOnReceipt" ? 0 : Number(/^Net(\d+)$/u.exec(termCode)?.[1] ?? NaN);
  const dueHint =
    Number.isFinite(netDays) && termCode
      ? t(
          "Due {0}",
          formatUnixDate((values.invoiceDate ?? Math.floor(Date.now() / 1000)) + netDays * 86400),
        )
      : null;
  const poRequired = (item.review?.issues ?? []).some(
    (issue) => issue.code === "PONumberMissing" && !issue.resolvedAt,
  );
  const payerName = item.billToCustomer?.name ?? item.shipment?.customer?.name ?? "—";
  const reference = item.shipment?.bol || item.shipment?.externalReference || "—";

  return (
    <section aria-label={t("Invoice details")} className="flex flex-col gap-2.5 text-sm">
      <div className="flex items-baseline gap-2">
        <h3 className="text-foreground font-semibold">{t("Invoice")}</h3>
        <span className="text-foreground-muted text-xs" aria-live="polite">
          {!editable
            ? t("Set when it was approved")
            : isSaving
              ? t("Saving…")
              : t("Draft · saves as you edit")}
        </span>
      </div>
      <Form onSubmit={(event) => event.preventDefault()}>
        <div className="grid grid-cols-3 gap-x-3.5 gap-y-2.5">
          <ReadOnlyField label={t("Bill to")} value={payerName} />
          <div className="flex flex-col gap-1">
            <InputField
              control={form.control}
              name="poNumber"
              label={t("Customer PO")}
              placeholder={t("PO-55821")}
              maxLength={100}
              disabled={!editable}
            />
            {poRequired ? (
              <span className="text-warning-foreground text-2xs">
                {t("Required by {0}", payerName)}
              </span>
            ) : null}
          </div>
          <ReadOnlyField label={t("BOL / reference")} value={reference} mono />
          {editable ? (
            <DatePickerField
              control={form.control}
              name="invoiceDate"
              label={t("Invoice date")}
              placeholder={t("The day it's approved")}
            />
          ) : (
            <ReadOnlyField
              label={t("Invoice date")}
              value={
                item.invoiceDate ? formatUnixDate(item.invoiceDate) : t("The day it's approved")
              }
            />
          )}
          <div className="flex flex-col gap-1">
            <SelectField
              control={form.control}
              name="paymentTerm"
              label={t("Terms")}
              options={termOptions}
              isReadOnly={!editable}
            />
            {dueHint ? <span className="text-foreground-muted text-2xs">{dueHint}</span> : null}
          </div>
        </div>
        <div className="mt-2.5">
          <InputField
            control={form.control}
            name="invoiceMemo"
            label={t("Note on invoice")}
            placeholder={t("Shown under the charges on the customer's invoice")}
            maxLength={1000}
            disabled={!editable}
          />
        </div>
      </Form>
    </section>
  );
}

function ReadOnlyField({
  label,
  value,
  mono = false,
}: {
  label: string;
  value: string;
  mono?: boolean;
}) {
  return (
    <div className="flex min-w-0 flex-col gap-1">
      <span className="text-foreground-muted text-xs font-medium">{label}</span>
      <span
        className={cn(
          "ui-field bg-muted text-foreground-muted flex h-7 items-center truncate px-2",
          mono && "font-mono",
        )}
      >
        {value}
      </span>
    </div>
  );
}
