import { InputField } from "@/components/fields/input-field";
import { TextareaField } from "@/components/fields/textarea-field";
import { useApiMutation } from "@/hooks/use-api-mutation";
import {
  requestMissingFactoringDocuments,
  type FactoringDocumentRequestResult,
} from "@/lib/graphql/factoring";
import { invalidateFactoringQueries } from "@/lib/queries/factoring";
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
import { translate } from "@trenova/shared/i18n/runtime";
import { useT } from "@trenova/shared/i18n/use-t";
import { useEffect } from "react";
import { FormProvider, useForm, type Resolver } from "react-hook-form";
import { toast } from "@trenova/shared/components/ui/toast";
import { z } from "zod";

const FORM_ID = "request-factoring-documents-form";
const MAX_MESSAGE = 2000;

const requestSchema = z.object({
  recipient: z
    .string()
    .trim()
    .max(500, { error: () => translate("Recipients must be at most 500 characters") }),
  message: z
    .string()
    .max(MAX_MESSAGE, { error: () => translate("The message must be at most 2,000 characters") }),
});

type RequestValues = z.infer<typeof requestSchema>;

const DEFAULT_VALUES: RequestValues = { recipient: "", message: "" };

/**
 * Asks whoever holds the missing paperwork for it. With no recipient the
 * server picks the driver, then the carrier, then the customer's invoice
 * contacts; with no document named it asks for everything the invoice lacks.
 */
export function RequestFactoringDocumentsDialog({
  invoiceId,
  invoiceNumber,
  documentTypeIds,
  documentNames,
  open,
  onOpenChange,
}: {
  invoiceId: string;
  invoiceNumber: string;
  documentTypeIds?: string[];
  documentNames: string[];
  open: boolean;
  onOpenChange: (open: boolean) => void;
}) {
  const t = useT();
  const queryClient = useQueryClient();
  const form = useForm<RequestValues>({
    resolver: zodResolver(requestSchema) as Resolver<RequestValues>,
    defaultValues: DEFAULT_VALUES,
  });

  useEffect(() => {
    if (open) {
      form.reset(DEFAULT_VALUES);
    }
  }, [open, form]);

  const mutation = useApiMutation<
    FactoringDocumentRequestResult,
    RequestValues,
    unknown,
    RequestValues
  >({
    form,
    resourceName: "factoring document request",
    mutationFn: (values) =>
      requestMissingFactoringDocuments({
        invoiceId,
        documentTypeIds: documentTypeIds && documentTypeIds.length > 0 ? documentTypeIds : null,
        recipient: values.recipient.trim() === "" ? null : values.recipient.trim(),
        message: values.message.trim() === "" ? null : values.message.trim(),
      }),
    onSuccess: (result) => {
      invalidateFactoringQueries(queryClient);
      toast.success(t("Documents requested"), {
        description: t("Asked {0} for {1}", result.recipient, result.documentTypes.join(", ")),
      });
      onOpenChange(false);
    },
  });

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent size="md">
        <DialogHeader>
          <DialogTitle>{t("Request documents for {0}", invoiceNumber)}</DialogTitle>
          <DialogDescription>
            {documentNames.length > 0
              ? t("Asks for {0}.", documentNames.join(", "))
              : t("Asks for every document this invoice is missing.")}
          </DialogDescription>
        </DialogHeader>
        <FormProvider {...form}>
          <Form
            id={FORM_ID}
            onSubmit={(event) => {
              event.stopPropagation();
              void form.handleSubmit((values) => mutation.mutate(values))(event);
            }}
          >
            <FormGroup cols={1}>
              <FormControl>
                <InputField
                  control={form.control}
                  name="recipient"
                  label={t("Recipient")}
                  placeholder={t("Leave blank to ask the driver, carrier or customer")}
                  description={t(
                    "Email addresses, separated by commas. When blank, the driver is asked first, then the carrier, then the customer's invoice contacts.",
                  )}
                />
              </FormControl>
              <FormControl>
                <TextareaField
                  control={form.control}
                  name="message"
                  label={t("Message")}
                  placeholder={t("Optional note added to the request")}
                  rows={3}
                />
              </FormControl>
            </FormGroup>
          </Form>
        </FormProvider>
        <DialogFooter>
          <Button type="button" variant="outline" onClick={() => onOpenChange(false)}>
            {t("Cancel")}
          </Button>
          <Button type="submit" form={FORM_ID} isLoading={mutation.isPending}>
            {t("Send request")}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
