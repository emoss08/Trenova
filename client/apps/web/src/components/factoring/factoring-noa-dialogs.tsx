import { DocumentUploadZone } from "@/components/documents/document-upload-zone";
import { DateField } from "@/components/fields/date-field/date-field";
import { InputField } from "@/components/fields/input-field";
import { SelectField } from "@/components/fields/select-field";
import { TextareaField } from "@/components/fields/textarea-field";
import { useApiMutation } from "@/hooks/use-api-mutation";
import {
  acknowledgeFactoringNoa,
  releaseFactoringNoa,
  sendFactoringNoa,
  type FactoringNoa,
} from "@/lib/graphql/factoring";
import { invalidateFactoringQueries } from "@/lib/queries/factoring";
import { resourceDocumentsQuery } from "@/lib/resource-documents";
import { apiService } from "@/services/api";
import { zodResolver } from "@hookform/resolvers/zod";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
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
import { Label } from "@trenova/shared/components/ui/label";
import { translate } from "@trenova/shared/i18n/runtime";
import { useT } from "@trenova/shared/i18n/use-t";
import { dateToUnixTimestamp } from "@trenova/shared/lib/date";
import { useEffect } from "react";
import { FormProvider, useForm, useWatch, type Resolver } from "react-hook-form";
import { toast } from "sonner";
import { z } from "zod";

/** The resource type a notice's own documents (its letter, the signed copy) are filed under. */
export const FACTORING_NOA_DOCUMENT_RESOURCE = "factoring_noa";

const MAX_RECIPIENTS = 500;
const MAX_RELEASE_REASON = 500;

function noaCustomerName(noa: FactoringNoa): string {
  return noa.customer?.name ?? noa.customerId;
}

const sendSchema = z.object({
  recipient: z
    .string()
    .trim()
    .max(MAX_RECIPIENTS, { error: () => translate("Recipients must be at most 500 characters") }),
});
type SendValues = z.infer<typeof sendSchema>;

const SEND_FORM_ID = "send-factoring-noa-form";

/**
 * Emails the customer the notice letter. With no recipient the customer's
 * invoice email contacts receive it; sending the same version twice does not
 * send twice.
 */
export function SendFactoringNoaDialog({
  noa,
  onOpenChange,
  onSent,
}: {
  noa: FactoringNoa | null;
  onOpenChange: (open: boolean) => void;
  onSent?: (noa: FactoringNoa) => void;
}) {
  const t = useT();
  const queryClient = useQueryClient();
  const form = useForm<SendValues>({
    resolver: zodResolver(sendSchema) as Resolver<SendValues>,
    defaultValues: { recipient: "" },
  });
  const { reset } = form;

  useEffect(() => {
    if (noa) reset({ recipient: noa.sentTo ?? "" });
  }, [noa, reset]);

  const mutation = useApiMutation<FactoringNoa, SendValues, unknown, SendValues>({
    form,
    resourceName: "notice of assignment",
    mutationFn: (values) => {
      if (!noa) throw new Error("No notice selected");
      const recipient = values.recipient.trim();
      return sendFactoringNoa({
        id: noa.id,
        version: noa.version,
        recipient: recipient === "" ? null : recipient,
      });
    },
    onSuccess: (sent) => {
      invalidateFactoringQueries(queryClient);
      toast.success(t("Notice sent"), {
        description: sent.sentTo ? t("Emailed to {0}", sent.sentTo) : undefined,
      });
      onSent?.(sent);
      onOpenChange(false);
    },
  });

  return (
    <Dialog open={noa !== null} onOpenChange={onOpenChange}>
      <DialogContent size="md">
        <DialogHeader>
          <DialogTitle>{t("Send notice of assignment")}</DialogTitle>
          <DialogDescription>
            {noa
              ? t(
                  "Emails {0} the letter telling them to pay {1} for invoices from now on.",
                  noaCustomerName(noa),
                  noa.factoringCompany?.name ?? noa.factoringCompanyId,
                )
              : null}
          </DialogDescription>
        </DialogHeader>
        <FormProvider {...form}>
          <Form
            id={SEND_FORM_ID}
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
                  placeholder={t("Leave blank to use the customer's invoice contacts")}
                  description={t(
                    "Email addresses, separated by commas. When blank, the notice goes to the customer's invoice email recipients.",
                  )}
                />
              </FormControl>
            </FormGroup>
          </Form>
        </FormProvider>
        <DialogFooter>
          <Button type="button" variant="outline" onClick={() => onOpenChange(false)}>
            {t("Cancel")}
          </Button>
          <Button type="submit" form={SEND_FORM_ID} isLoading={mutation.isPending}>
            {t("Send notice")}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}

const acknowledgeSchema = z
  .object({
    acknowledgedAt: z.number({ error: () => translate("Acknowledgement date is required") }),
    signedDocumentId: z
      .string()
      .min(1, { error: () => translate("Attach the customer's signed copy of the notice") }),
    expiresAt: z.number().nullable(),
  })
  .superRefine((values, ctx) => {
    if (values.expiresAt != null && values.expiresAt <= values.acknowledgedAt) {
      ctx.addIssue({
        code: "custom",
        path: ["expiresAt"],
        message: translate("The notice must expire after it was acknowledged"),
      });
    }
  });
type AcknowledgeValues = z.infer<typeof acknowledgeSchema>;

const ACKNOWLEDGE_FORM_ID = "acknowledge-factoring-noa-form";

/**
 * Records that the customer signed the notice. The signed copy is a document
 * on the notice itself: pick one already uploaded there, or upload it here.
 */
export function AcknowledgeFactoringNoaDialog({
  noa,
  onOpenChange,
}: {
  noa: FactoringNoa | null;
  onOpenChange: (open: boolean) => void;
}) {
  const t = useT();
  const queryClient = useQueryClient();
  const form = useForm<AcknowledgeValues>({
    resolver: zodResolver(acknowledgeSchema) as Resolver<AcknowledgeValues>,
    defaultValues: { acknowledgedAt: 0, signedDocumentId: "", expiresAt: null },
  });
  const { reset, setValue } = form;
  const documentsQuery = resourceDocumentsQuery(FACTORING_NOA_DOCUMENT_RESOURCE, noa?.id ?? "");
  const documents = useQuery({ ...documentsQuery, enabled: noa !== null });

  useEffect(() => {
    if (noa) {
      reset({
        acknowledgedAt: dateToUnixTimestamp(new Date()),
        signedDocumentId: noa.signedDocumentId ?? "",
        expiresAt: noa.expiresAt ?? null,
      });
    }
  }, [noa, reset]);

  const upload = useMutation({
    mutationFn: async (file: File) => {
      if (!noa) throw new Error("No notice selected");
      return apiService.documentService.upload({
        file,
        resourceId: noa.id,
        resourceType: FACTORING_NOA_DOCUMENT_RESOURCE,
        description: translate("Signed notice of assignment"),
      });
    },
    onSuccess: (document) => {
      void queryClient.invalidateQueries({ queryKey: documentsQuery.queryKey });
      setValue("signedDocumentId", document.id, { shouldValidate: true, shouldDirty: true });
    },
    onError: (error) =>
      toast.error(t("Upload failed"), {
        description: error instanceof Error ? error.message : undefined,
      }),
  });

  const mutation = useApiMutation<FactoringNoa, AcknowledgeValues, unknown, AcknowledgeValues>({
    form,
    resourceName: "notice of assignment",
    mutationFn: (values) => {
      if (!noa) throw new Error("No notice selected");
      return acknowledgeFactoringNoa({
        id: noa.id,
        version: noa.version,
        acknowledgedAt: values.acknowledgedAt,
        signedDocumentId: values.signedDocumentId,
        expiresAt: values.expiresAt,
      });
    },
    onSuccess: () => {
      invalidateFactoringQueries(queryClient);
      onOpenChange(false);
    },
  });

  const signedDocumentId = useWatch({ control: form.control, name: "signedDocumentId" });
  const documentOptions = (documents.data ?? []).map((document) => ({
    value: document.id,
    label: document.originalName || document.fileName,
  }));

  return (
    <Dialog open={noa !== null} onOpenChange={onOpenChange}>
      <DialogContent size="md">
        <DialogHeader>
          <DialogTitle>{t("Record acknowledgement")}</DialogTitle>
          <DialogDescription>
            {noa
              ? t(
                  "Record that {0} signed the notice. Its invoices for this factor can be ready once it is acknowledged.",
                  noaCustomerName(noa),
                )
              : null}
          </DialogDescription>
        </DialogHeader>
        <FormProvider {...form}>
          <Form
            id={ACKNOWLEDGE_FORM_ID}
            onSubmit={(event) => {
              event.stopPropagation();
              void form.handleSubmit((values) => mutation.mutate(values))(event);
            }}
          >
            <FormGroup cols={2}>
              <FormControl>
                <DateField
                  control={form.control}
                  name="acknowledgedAt"
                  label={t("Acknowledged on")}
                  rules={{ required: true }}
                />
              </FormControl>
              <FormControl>
                <DateField
                  control={form.control}
                  name="expiresAt"
                  label={t("Expires")}
                  placeholder={t("Does not expire")}
                  clearable
                />
              </FormControl>
              <FormControl cols="full">
                <SelectField
                  control={form.control}
                  name="signedDocumentId"
                  label={t("Signed copy")}
                  placeholder={
                    documents.isLoading
                      ? t("Loading documents...")
                      : documentOptions.length > 0
                        ? t("Choose a document on this notice")
                        : t("Upload the signed copy below")
                  }
                  options={documentOptions}
                  rules={{ required: true }}
                />
              </FormControl>
              <FormControl cols="full">
                <div className="flex flex-col gap-1.5">
                  <Label>{t("Upload the signed copy")}</Label>
                  <DocumentUploadZone
                    disabled={upload.isPending}
                    accept=".pdf,.jpg,.jpeg,.png,.webp"
                    onFilesSelected={(files) => {
                      const [file] = files;
                      if (file) upload.mutate(file);
                    }}
                  />
                  {signedDocumentId && upload.isSuccess ? (
                    <p className="text-muted-foreground text-xs">
                      {t("The uploaded copy is selected.")}
                    </p>
                  ) : null}
                </div>
              </FormControl>
            </FormGroup>
          </Form>
        </FormProvider>
        <DialogFooter>
          <Button type="button" variant="outline" onClick={() => onOpenChange(false)}>
            {t("Cancel")}
          </Button>
          <Button
            type="submit"
            form={ACKNOWLEDGE_FORM_ID}
            isLoading={mutation.isPending}
            disabled={upload.isPending}
          >
            {t("Record acknowledgement")}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}

const releaseSchema = z.object({
  reason: z
    .string()
    .trim()
    .min(1, { error: () => translate("Say why the notice is released") })
    .max(MAX_RELEASE_REASON, {
      error: () => translate("Keep the reason under 500 characters"),
    }),
});
type ReleaseValues = z.infer<typeof releaseSchema>;

const RELEASE_FORM_ID = "release-factoring-noa-form";

/** Tells the customer to pay the organization again; the factor no longer directs payment. */
export function ReleaseFactoringNoaDialog({
  noa,
  onOpenChange,
}: {
  noa: FactoringNoa | null;
  onOpenChange: (open: boolean) => void;
}) {
  const t = useT();
  const queryClient = useQueryClient();
  const form = useForm<ReleaseValues>({
    resolver: zodResolver(releaseSchema) as Resolver<ReleaseValues>,
    defaultValues: { reason: "" },
  });
  const { reset } = form;

  useEffect(() => {
    if (noa) reset({ reason: "" });
  }, [noa, reset]);

  const mutation = useApiMutation<FactoringNoa, ReleaseValues, unknown, ReleaseValues>({
    form,
    resourceName: "notice of assignment",
    mutationFn: (values) => {
      if (!noa) throw new Error("No notice selected");
      return releaseFactoringNoa({
        id: noa.id,
        version: noa.version,
        reason: values.reason.trim(),
      });
    },
    onSuccess: () => {
      invalidateFactoringQueries(queryClient);
      onOpenChange(false);
    },
  });

  return (
    <Dialog open={noa !== null} onOpenChange={onOpenChange}>
      <DialogContent size="md">
        <DialogHeader>
          <DialogTitle>{t("Release notice of assignment")}</DialogTitle>
          <DialogDescription>
            {noa
              ? t(
                  "{0} stops paying {1}. Its invoices are no longer ready to factor until a new notice is acknowledged.",
                  noaCustomerName(noa),
                  noa.factoringCompany?.name ?? noa.factoringCompanyId,
                )
              : null}
          </DialogDescription>
        </DialogHeader>
        <FormProvider {...form}>
          <Form
            id={RELEASE_FORM_ID}
            onSubmit={(event) => {
              event.stopPropagation();
              void form.handleSubmit((values) => mutation.mutate(values))(event);
            }}
          >
            <FormGroup cols={1}>
              <FormControl>
                <TextareaField
                  control={form.control}
                  name="reason"
                  label={t("Reason")}
                  placeholder={t("Why the factor no longer directs this customer's payments")}
                  rows={3}
                  rules={{ required: true }}
                />
              </FormControl>
            </FormGroup>
          </Form>
        </FormProvider>
        <DialogFooter>
          <Button type="button" variant="outline" onClick={() => onOpenChange(false)}>
            {t("Cancel")}
          </Button>
          <Button
            type="submit"
            form={RELEASE_FORM_ID}
            variant="destructive"
            isLoading={mutation.isPending}
          >
            {t("Release notice")}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
