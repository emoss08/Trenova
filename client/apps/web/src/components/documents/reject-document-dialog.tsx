import { useT } from "@trenova/shared/i18n/use-t";
import { TextareaField } from "@/components/fields/textarea-field";
import { useApiMutation } from "@/hooks/use-api-mutation";
import {
  DOCUMENT_REJECTION_REASON_MAX_LENGTH,
  rejectDocumentFormSchema,
  type RejectDocumentFormValues,
} from "@/lib/document-review";
import { apiService } from "@/services/api";
import { zodResolver } from "@hookform/resolvers/zod";
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
import type { Document } from "@trenova/shared/types/document";
import { useEffect } from "react";
import { FormProvider, useForm, type Resolver } from "react-hook-form";
import { toast } from "sonner";

export type RejectDocumentDialogProps = {
  document: Document | null;
  open: boolean;
  onOpenChange: (open: boolean) => void;
  onRejected?: (document: Document) => void;
};

const DEFAULT_VALUES: RejectDocumentFormValues = { reason: "" };

export function RejectDocumentDialog({
  document,
  open,
  onOpenChange,
  onRejected,
}: RejectDocumentDialogProps) {
  const t = useT();

  const form = useForm<RejectDocumentFormValues>({
    resolver: zodResolver(rejectDocumentFormSchema) as Resolver<RejectDocumentFormValues>,
    defaultValues: DEFAULT_VALUES,
  });
  const { control, handleSubmit, reset } = form;

  useEffect(() => {
    if (open) {
      reset(DEFAULT_VALUES);
    }
  }, [open, reset]);

  const { mutateAsync, isPending } = useApiMutation<
    Document,
    RejectDocumentFormValues,
    unknown,
    RejectDocumentFormValues
  >({
    form,
    resourceName: "Document",
    mutationFn: (values) => {
      if (!document) {
        throw new Error("No document selected");
      }
      return apiService.documentService.reject(document.id, values.reason);
    },
    onSuccess: (rejected) => {
      toast.success(t("Document rejected"));
      onRejected?.(rejected);
      onOpenChange(false);
    },
  });

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent size="md">
        <DialogHeader>
          <DialogTitle>{t("Reject document")}</DialogTitle>
          <DialogDescription>
            {t(
              "A rejected document no longer counts toward the shipment's billing requirements. Upload a replacement or approve it later to restore it.",
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
                <TextareaField<RejectDocumentFormValues>
                  control={control}
                  name="reason"
                  label={t("Reason")}
                  placeholder={t("What is wrong with it, such as an illegible signature or the wrong load")}
                  rules={{ required: true }}
                  maxLength={DOCUMENT_REJECTION_REASON_MAX_LENGTH}
                />
              </FormControl>
            </FormGroup>
            <DialogFooter>
              <Button type="button" variant="outline" onClick={() => onOpenChange(false)}>
                {t("Cancel")}
              </Button>
              <Button
                type="submit"
                variant="destructive"
                isLoading={isPending}
                disabled={!document}
              >
                {t("Reject")}
              </Button>
            </DialogFooter>
          </Form>
        </FormProvider>
      </DialogContent>
    </Dialog>
  );
}
