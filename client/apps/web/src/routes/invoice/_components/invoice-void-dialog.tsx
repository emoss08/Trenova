import { SelectField } from "@/components/fields/select-field";
import { TextareaField } from "@/components/fields/textarea-field";
import { useApiMutation } from "@/hooks/use-api-mutation";
import { invoiceVoidDispositionChoices } from "@/lib/choices";
import { voidInvoice, type VoidInvoiceResult } from "@/lib/graphql/invoice";
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
import { useT } from "@trenova/shared/i18n/use-t";
import {
  voidInvoiceFormSchema,
  type Invoice,
  type VoidInvoiceFormValues,
} from "@trenova/shared/types/invoice";
import { ClockIcon } from "lucide-react";
import { useEffect, useState } from "react";
import { FormProvider, useForm, type Resolver } from "react-hook-form";
import { toast } from "sonner";

const VOID_FORM_ID = "invoice-void-form";

const DEFAULT_VALUES: VoidInvoiceFormValues = { reason: "", disposition: "DoNotRebill" };

/**
 * Takes an invoice out of circulation. A draft is voided the moment the form
 * submits; a posted invoice is voided through a full-reversal adjustment, and
 * when that reversal needs an approver the dialog stays open to say so rather
 * than closing as if the invoice were already gone.
 */
export function InvoiceVoidDialog({
  invoice,
  open,
  onOpenChange,
  onVoided,
}: {
  invoice: Pick<Invoice, "id" | "number" | "status">;
  open: boolean;
  onOpenChange: (open: boolean) => void;
  onVoided?: (result: VoidInvoiceResult) => void;
}) {
  const t = useT();
  const queryClient = useQueryClient();
  const [pending, setPending] = useState<VoidInvoiceResult | null>(null);

  const form = useForm<VoidInvoiceFormValues>({
    resolver: zodResolver(voidInvoiceFormSchema) as Resolver<VoidInvoiceFormValues>,
    defaultValues: DEFAULT_VALUES,
  });

  useEffect(() => {
    if (open) {
      form.reset(DEFAULT_VALUES);
      setPending(null);
    }
  }, [open, form]);

  const mutation = useApiMutation<
    VoidInvoiceResult,
    VoidInvoiceFormValues,
    unknown,
    VoidInvoiceFormValues
  >({
    form,
    resourceName: "invoice",
    mutationFn: (values) =>
      voidInvoice({
        invoiceId: invoice.id,
        reason: values.reason,
        disposition: values.disposition,
      }),
    onSuccess: (result) => {
      invalidateInvoiceQueries(queryClient);
      onVoided?.(result);
      if (result.pendingApproval) {
        setPending(result);
        toast.success(t("Void of {0} requested; awaiting reversal approval", invoice.number));
        return;
      }
      toast.success(t("{0} voided", invoice.number));
      onOpenChange(false);
    },
  });

  const dispositionOptions = invoiceVoidDispositionChoices.map((choice) => ({
    value: choice.value,
    label: t(choice.label),
    description: t(choice.description),
  }));

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-[520px]">
        <DialogHeader>
          <DialogTitle>{t("Void invoice {0}", invoice.number)}</DialogTitle>
          <DialogDescription>
            {invoice.status === "Posted"
              ? t(
                  "A posted invoice is voided through a full-reversal credit memo. The invoice keeps its number and stays readable, but nothing more is owed on it.",
                )
              : t(
                  "The draft is voided at once and keeps its number so the audit trail stays readable.",
                )}
          </DialogDescription>
        </DialogHeader>

        {pending ? (
          <div className="flex gap-3 rounded-md border border-amber-300 bg-amber-50/60 p-3 text-sm dark:border-amber-900 dark:bg-amber-950/30">
            <ClockIcon className="mt-0.5 size-4 shrink-0 text-amber-600 dark:text-amber-400" />
            <div>
              <p className="font-medium text-amber-800 dark:text-amber-200">
                {t("Void requested; awaiting reversal approval")}
              </p>
              <p className="mt-0.5 text-xs text-amber-800/80 dark:text-amber-200/80">
                {t(
                  "The full-reversal adjustment needs an approver. The invoice reads Voided the moment it executes.",
                )}
              </p>
            </div>
          </div>
        ) : (
          <FormProvider {...form}>
            <Form
              id={VOID_FORM_ID}
              onSubmit={form.handleSubmit((values) => mutation.mutate(values))}
            >
              <FormGroup cols={1}>
                <FormControl>
                  <TextareaField
                    control={form.control}
                    name="reason"
                    label={t("Reason")}
                    placeholder={t("Why this invoice should not stand")}
                    rows={3}
                  />
                </FormControl>
                <FormControl>
                  <SelectField
                    control={form.control}
                    name="disposition"
                    label={t("Freight disposition")}
                    description={t(
                      "What happens to the billing queue items and shipments this invoice billed.",
                    )}
                    options={dispositionOptions}
                  />
                </FormControl>
              </FormGroup>
            </Form>
          </FormProvider>
        )}

        <DialogFooter>
          <Button type="button" variant="outline" onClick={() => onOpenChange(false)}>
            {pending ? t("Close") : t("Cancel")}
          </Button>
          {pending ? null : (
            <Button
              type="submit"
              form={VOID_FORM_ID}
              variant="destructive"
              disabled={mutation.isPending}
            >
              {t("Void invoice")}
            </Button>
          )}
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
