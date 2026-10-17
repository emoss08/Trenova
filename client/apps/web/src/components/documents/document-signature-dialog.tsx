import { useT } from "@trenova/shared/i18n/use-t";
import { DatePickerField } from "@/components/fields/date-picker-field";
import { SegmentedField } from "@/components/fields/segmented-field";
import { useApiMutation } from "@/hooks/use-api-mutation";
import {
  DOCUMENT_SIGNATURE_LABEL,
  DOCUMENT_SIGNATURE_STATUSES,
  documentSignatureFormSchema,
  documentSignatureFormValues,
  type DocumentSignatureFormValues,
} from "@/lib/document-signature";
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
import type { Document, DocumentSignatureStatus } from "@trenova/shared/types/document";
import { useEffect } from "react";
import { FormProvider, useForm, useWatch, type Resolver } from "react-hook-form";

export type DocumentSignatureDialogProps = {
  document: Document | null;
  open: boolean;
  onOpenChange: (open: boolean) => void;
  onSaved?: (document: Document) => void;
};

const EMPTY_VALUES: DocumentSignatureFormValues = { status: "Unknown", signedAt: null };

export function DocumentSignatureDialog({
  document,
  open,
  onOpenChange,
  onSaved,
}: DocumentSignatureDialogProps) {
  const t = useT();
  const queryClient = useQueryClient();

  const form = useForm<DocumentSignatureFormValues>({
    resolver: zodResolver(documentSignatureFormSchema) as Resolver<DocumentSignatureFormValues>,
    defaultValues: document ? documentSignatureFormValues(document) : EMPTY_VALUES,
  });
  const { control, handleSubmit, reset, setValue } = form;
  const status = useWatch({ control, name: "status" });

  useEffect(() => {
    if (open) {
      reset(document ? documentSignatureFormValues(document) : EMPTY_VALUES);
    }
  }, [open, document, reset]);

  useEffect(() => {
    if (status !== "Signed") {
      setValue("signedAt", null);
    }
  }, [status, setValue]);

  const { mutateAsync, isPending } = useApiMutation<
    Document,
    DocumentSignatureFormValues,
    unknown,
    DocumentSignatureFormValues
  >({
    form,
    resourceName: "Document",
    mutationFn: (values) => {
      if (!document) {
        throw new Error("No document selected");
      }
      return apiService.documentService.setSignature(document.id, {
        status: values.status,
        signedAt: values.status === "Signed" ? values.signedAt : null,
        version: document.version,
      });
    },
    onSuccess: (saved) => {
      void queryClient.invalidateQueries({ queryKey: ["documents"] });
      void queryClient.invalidateQueries({ queryKey: ["document-versions"] });
      void queryClient.invalidateQueries({ queryKey: ["shipment", "billingReadiness"] });
      void queryClient.invalidateQueries({ queryKey: ["billingQueue"] });
      void queryClient.invalidateQueries({ queryKey: ["billing-queue-list"] });
      onSaved?.(saved);
      onOpenChange(false);
    },
  });

  const options = DOCUMENT_SIGNATURE_STATUSES.map((value: DocumentSignatureStatus) => ({
    value,
    label: DOCUMENT_SIGNATURE_LABEL[value],
  }));

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent size="md">
        <DialogHeader>
          <DialogTitle>{t("Record signature")}</DialogTitle>
          <DialogDescription>
            {t(
              "Say whether the receiver signed this document. Your answer is final: reading the document again won't change it, and billing checks the proof of delivery against it.",
            )}
          </DialogDescription>
        </DialogHeader>
        {document ? (
          <div className="border-y py-3 text-sm">
            <span className="font-medium">{document.originalName}</span>
          </div>
        ) : null}
        <FormProvider {...form}>
          <Form
            className="flex flex-col gap-4"
            onSubmit={(submitEvent) => {
              submitEvent.preventDefault();
              submitEvent.stopPropagation();
              void handleSubmit((values) => mutateAsync(values))(submitEvent);
            }}
          >
            <FormGroup cols={1}>
              <FormControl>
                <SegmentedField<DocumentSignatureFormValues, DocumentSignatureStatus>
                  control={control}
                  name="status"
                  label={t("Signature")}
                  options={options}
                  rules={{ required: true }}
                />
              </FormControl>
              {status === "Signed" ? (
                <FormControl>
                  <DatePickerField<DocumentSignatureFormValues>
                    control={control}
                    name="signedAt"
                    label={t("Signed on")}
                    description={t("Leave empty when the date beside the signature can't be read.")}
                    placeholder={t("Date not legible")}
                  />
                </FormControl>
              ) : null}
            </FormGroup>
            <DialogFooter>
              <Button type="button" variant="outline" onClick={() => onOpenChange(false)}>
                {t("Cancel")}
              </Button>
              <Button type="submit" isLoading={isPending} disabled={!document}>
                {t("Save")}
              </Button>
            </DialogFooter>
          </Form>
        </FormProvider>
      </DialogContent>
    </Dialog>
  );
}
