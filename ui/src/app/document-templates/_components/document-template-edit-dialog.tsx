import { useDataTable } from "@/components/data-table/data-table-provider";
import { Button, FormSaveButton } from "@/components/ui/button";
import {
  Dialog,
  DialogBody,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Form } from "@/components/ui/form";
import { usePopoutWindow } from "@/hooks/popout-window/use-popout-window";
import { useApiMutation } from "@/hooks/use-api-mutation";
import { broadcastQueryInvalidation } from "@/hooks/use-invalidate-query";
import { http } from "@/lib/http-client";
import {
  documentTemplateSchema,
  DocumentTemplateSchema,
} from "@/lib/schemas/document-template-schema";
import { type EditTableSheetProps } from "@/types/data-table";
import { zodResolver } from "@hookform/resolvers/zod";
import { useQueryClient } from "@tanstack/react-query";
import { useCallback, useEffect } from "react";
import { FormProvider, useForm } from "react-hook-form";
import { toast } from "sonner";
import { DocumentTemplateForm } from "./document-template-form";

export function DocumentTemplateEditDialog({
  currentRecord,
}: EditTableSheetProps<DocumentTemplateSchema>) {
  const { isPopout, closePopout } = usePopoutWindow();
  const queryClient = useQueryClient();
  const { table, rowSelection } = useDataTable();

  const selectedRowKey = Object.keys(rowSelection)[0];

  const form = useForm<DocumentTemplateSchema>({
    resolver: zodResolver(documentTemplateSchema),
    defaultValues: currentRecord,
  });

  const {
    setError,
    formState: { isSubmitting, isSubmitSuccessful, isDirty },
    handleSubmit,
    reset,
  } = form;

  useEffect(() => {
    if (currentRecord) {
      reset(currentRecord);
    }
  }, [currentRecord, reset]);

  const { mutateAsync } = useApiMutation({
    mutationFn: async (values: DocumentTemplateSchema) => {
      const response = await http.put<DocumentTemplateSchema>(
        `/document-templates/${currentRecord?.id}/`,
        values,
      );
      return response.data;
    },
    onSuccess: (data) => {
      toast.success("Changes have been saved.", {
        description: `Document template "${data.name}" updated successfully`,
      });

      broadcastQueryInvalidation({
        queryKey: ["document-template-list"],
        options: {
          correlationId: `update-document-template-${Date.now()}`,
        },
        config: {
          predicate: true,
          refetchType: "all",
        },
      });

      queryClient.setQueryData(["document-template-list"], data);

      if (isPopout) {
        closePopout();
      }
    },
    setFormError: setError,
    resourceName: "Document Template",
  });

  const onSubmit = useCallback(
    async (values: DocumentTemplateSchema) => {
      await mutateAsync(values);
    },
    [mutateAsync],
  );

  useEffect(() => {
    if (isSubmitSuccessful) {
      reset(currentRecord);
    }
  }, [isSubmitSuccessful, reset, currentRecord]);

  return (
    <Dialog
      open={!!selectedRowKey}
      onOpenChange={(open) => {
        if (!open) {
          const el = selectedRowKey
            ? document.getElementById(selectedRowKey)
            : null;
          table.resetRowSelection();
          if (el) {
            setTimeout(() => el.focus(), 0);
          }
        }
      }}
    >
      <DialogContent className="sm:max-w-[95vw]">
        <DialogHeader>
          <DialogTitle>Edit Document Template</DialogTitle>
          <DialogDescription>
            Editing &quot;{currentRecord?.name}&quot; ({currentRecord?.code})
          </DialogDescription>
        </DialogHeader>
        <FormProvider {...form}>
          <Form onSubmit={handleSubmit(onSubmit)}>
            <DialogBody className="p-0">
              <DocumentTemplateForm />
            </DialogBody>
            <DialogFooter>
              <Button
                type="button"
                variant="outline"
                onClick={() => {
                  table.resetRowSelection();
                }}
              >
                Cancel
              </Button>
              <FormSaveButton
                isPopout={isPopout}
                isSubmitting={isSubmitting}
                title="Document Template"
                text="Save Changes"
                disabled={!isDirty}
              />
            </DialogFooter>
          </Form>
        </FormProvider>
      </DialogContent>
    </Dialog>
  );
}
