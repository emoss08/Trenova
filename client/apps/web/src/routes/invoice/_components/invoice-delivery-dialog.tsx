import { InputField } from "@/components/fields/input-field";
import { TextareaField } from "@/components/fields/textarea-field";
import { useApiMutation } from "@/hooks/use-api-mutation";
import { invalidateInvoiceQueries, invoice as invoiceQueries } from "@/lib/queries/invoice";
import { apiService } from "@/services/api";
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
import { useT, type TranslateFn } from "@trenova/shared/i18n/use-t";
import { invalidEmailAddresses, splitEmailList } from "@trenova/shared/lib/email-list";
import type { Invoice, InvoiceSendPlan } from "@trenova/shared/types/invoice";
import { useEffect, useMemo } from "react";
import { FormProvider, useForm, type Resolver } from "react-hook-form";
import { z } from "zod";

const DELIVERY_FORM_ID = "invoice-delivery-form";
const MAX_SUBJECT_LENGTH = 998;

const RECIPIENT_FIELDS = ["to", "cc", "bcc"] as const;

function deliveryFormSchema(t: TranslateFn) {
  return z
    .object({
      to: z.string(),
      cc: z.string(),
      bcc: z.string(),
      subject: z.string().max(MAX_SUBJECT_LENGTH),
      body: z.string(),
    })
    .superRefine((values, ctx) => {
      for (const field of RECIPIENT_FIELDS) {
        const invalid = invalidEmailAddresses(values[field]);
        if (invalid.length > 0) {
          ctx.addIssue({
            code: "custom",
            path: [field],
            message: t("Not an email address: {0}", invalid.join(", ")),
          });
        }
      }
    });
}

type DeliveryFormValues = z.infer<ReturnType<typeof deliveryFormSchema>>;

type DeliveryInvoice = Pick<
  Invoice,
  | "id"
  | "number"
  | "emailToSnapshot"
  | "emailCcSnapshot"
  | "emailBccSnapshot"
  | "emailSubjectSnapshot"
  | "emailBodySnapshot"
>;

function overridesOf(invoice: DeliveryInvoice): DeliveryFormValues {
  return {
    to: (invoice.emailToSnapshot ?? []).join(", "),
    cc: (invoice.emailCcSnapshot ?? []).join(", "),
    bcc: (invoice.emailBccSnapshot ?? []).join(", "),
    subject: invoice.emailSubjectSnapshot ?? "",
    body: invoice.emailBodySnapshot ?? "",
  };
}

const NO_OVERRIDES: DeliveryFormValues = { to: "", cc: "", bcc: "", subject: "", body: "" };

/**
 * Who this one invoice is emailed to and what it says. A field left empty uses
 * the customer's email profile and the organization's invoice email template,
 * so a corrected customer address still reaches a later resend; a field filled
 * in here applies to this invoice alone.
 */
export function InvoiceDeliveryDialog({
  invoice,
  sendPlan,
  open,
  onOpenChange,
}: {
  invoice: DeliveryInvoice;
  sendPlan: InvoiceSendPlan | undefined;
  open: boolean;
  onOpenChange: (open: boolean) => void;
}) {
  const t = useT();
  const queryClient = useQueryClient();

  const schema = useMemo(() => deliveryFormSchema(t), [t]);
  const form = useForm<DeliveryFormValues>({
    resolver: zodResolver(schema) as Resolver<DeliveryFormValues>,
    defaultValues: overridesOf(invoice),
  });

  useEffect(() => {
    if (open) {
      form.reset(overridesOf(invoice));
    }
  }, [open, form, invoice]);

  const mutation = useApiMutation<Invoice, DeliveryFormValues, unknown, DeliveryFormValues>({
    form,
    resourceName: "invoice",
    mutationFn: (values) =>
      apiService.invoiceService.updateDraft(invoice.id, {
        emailTo: splitEmailList(values.to),
        emailCc: splitEmailList(values.cc),
        emailBcc: splitEmailList(values.bcc),
        emailSubject: values.subject.trim(),
        emailBody: values.body.trim(),
      }),
    onSuccess: () => {
      invalidateInvoiceQueries(queryClient);
      void queryClient.invalidateQueries(invoiceQueries.sendPlan(invoice.id));
      onOpenChange(false);
    },
  });

  const defaults = sendPlan?.recipients;
  const placeholderFor = (addresses: string[] | undefined) =>
    addresses && addresses.length > 0
      ? addresses.join(", ")
      : t("From the customer's email profile");

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent size="lg">
        <DialogHeader>
          <DialogTitle>{t("Delivery for {0}", invoice.number)}</DialogTitle>
          <DialogDescription>
            {t(
              "Leave a field empty to use the customer's email profile and your invoice email template. Anything entered here applies to this invoice only.",
            )}
          </DialogDescription>
        </DialogHeader>

        <FormProvider {...form}>
          <Form
            id={DELIVERY_FORM_ID}
            onSubmit={form.handleSubmit((values) => mutation.mutate(values))}
          >
            <FormGroup cols={1}>
              <FormControl>
                <InputField
                  control={form.control}
                  name="to"
                  label={t("To")}
                  placeholder={placeholderFor(defaults?.to)}
                  description={t("Separate multiple addresses with commas.")}
                />
              </FormControl>
              <FormControl>
                <InputField
                  control={form.control}
                  name="cc"
                  label={t("CC")}
                  placeholder={placeholderFor(defaults?.cc)}
                />
              </FormControl>
              <FormControl>
                <InputField
                  control={form.control}
                  name="bcc"
                  label={t("BCC")}
                  placeholder={placeholderFor(defaults?.bcc)}
                />
              </FormControl>
              <FormControl>
                <InputField
                  control={form.control}
                  name="subject"
                  label={t("Subject")}
                  placeholder={sendPlan?.subject || t("From your invoice email template")}
                />
              </FormControl>
              <FormControl>
                <TextareaField
                  control={form.control}
                  name="body"
                  label={t("Message")}
                  placeholder={sendPlan?.body || t("From your invoice email template")}
                  rows={5}
                />
              </FormControl>
            </FormGroup>
          </Form>
        </FormProvider>

        <DialogFooter className="sm:justify-between">
          <Button
            type="button"
            variant="ghost"
            disabled={mutation.isPending}
            onClick={() => mutation.mutate(NO_OVERRIDES)}
          >
            {t("Use customer defaults")}
          </Button>
          <div className="flex gap-2">
            <Button type="button" variant="outline" onClick={() => onOpenChange(false)}>
              {t("Cancel")}
            </Button>
            <Button type="submit" form={DELIVERY_FORM_ID} disabled={mutation.isPending}>
              {t("Save")}
            </Button>
          </div>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
