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
import { type TableSheetProps } from "@/types/data-table";
import { zodResolver } from "@hookform/resolvers/zod";
import { useQueryClient } from "@tanstack/react-query";
import { useCallback, useEffect } from "react";
import { FormProvider, useForm } from "react-hook-form";
import { toast } from "sonner";
import { DocumentTemplateForm } from "./document-template-form";

export function DocumentTemplateCreateDialog({
  open,
  onOpenChange,
}: TableSheetProps) {
  const { isPopout, closePopout } = usePopoutWindow();
  const queryClient = useQueryClient();

  const form = useForm<DocumentTemplateSchema>({
    resolver: zodResolver(documentTemplateSchema),
    defaultValues: {
      code: "",
      name: "",
      description: "",
      documentTypeId: undefined,
      htmlContent: "",
      cssContent: "",
      headerHtml: "",
      footerHtml: "",
      pageSize: "Letter",
      orientation: "Portrait",
      marginTop: 20,
      marginBottom: 20,
      marginLeft: 20,
      marginRight: 20,
      status: "Draft",
      isDefault: false,
      isSystem: false,
    },
  });

  const {
    setError,
    formState: { isSubmitting, isSubmitSuccessful },
    handleSubmit,
    reset,
  } = form;

  const handleClose = useCallback(() => {
    onOpenChange(false);
    reset();
  }, [onOpenChange, reset]);

  const { mutateAsync } = useApiMutation({
    mutationFn: async (values: DocumentTemplateSchema) => {
      const response = await http.post<DocumentTemplateSchema>(
        "/document-templates/",
        values,
      );
      return response.data;
    },
    onSuccess: (data) => {
      toast.success("Changes have been saved.", {
        description: "Document template created successfully",
      });
      onOpenChange(false);

      broadcastQueryInvalidation({
        queryKey: ["document-template-list"],
        options: {
          correlationId: `create-document-template-${Date.now()}`,
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
      reset();
    }
  }, [isSubmitSuccessful, reset]);

  useEffect(() => {
    const handleKeyDown = (event: KeyboardEvent) => {
      if (
        open &&
        (event.ctrlKey || event.metaKey) &&
        event.key === "Enter" &&
        !isSubmitting
      ) {
        event.preventDefault();
        handleSubmit(onSubmit)();
      }
    };

    document.addEventListener("keydown", handleKeyDown);
    return () => document.removeEventListener("keydown", handleKeyDown);
  }, [open, isSubmitting, handleSubmit, onSubmit]);

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-[95vw]">
        <DialogHeader>
          <DialogTitle>Create Document Template</DialogTitle>
          <DialogDescription>
            Design a new document template using HTML, CSS, and Go template
            syntax
          </DialogDescription>
        </DialogHeader>
        <FormProvider {...form}>
          <Form onSubmit={handleSubmit(onSubmit)} className="flex flex-col">
            <DialogBody className="p-0">
              <DocumentTemplateForm />
            </DialogBody>
            <DialogFooter>
              <Button type="button" variant="outline" onClick={handleClose}>
                Cancel
              </Button>
              <FormSaveButton
                isPopout={isPopout}
                isSubmitting={isSubmitting}
                title="Document Template"
                text="Save Template"
              />
            </DialogFooter>
          </Form>
        </FormProvider>
      </DialogContent>
    </Dialog>
  );
}
