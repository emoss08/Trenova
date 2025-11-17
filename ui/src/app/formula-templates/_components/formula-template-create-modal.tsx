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
import { VisuallyHidden } from "@/components/ui/visually-hidden";
import { usePopoutWindow } from "@/hooks/popout-window/use-popout-window";
import { useFormWithSave } from "@/hooks/use-form-with-save";
import { broadcastQueryInvalidation } from "@/hooks/use-invalidate-query";
import { http } from "@/lib/http-client";
import {
  type FormulaTemplateSchema,
  formulaTemplateSchema,
} from "@/lib/schemas/formula-template-schema";
import { type TableSheetProps } from "@/types/data-table";
import { zodResolver } from "@hookform/resolvers/zod";
import { useCallback, useEffect } from "react";
import { FormProvider } from "react-hook-form";
import { FormulaTemplateForm } from "./formula-template-form";

export function CreateFormulaTemplateModal({
  open,
  onOpenChange,
}: TableSheetProps) {
  const { isPopout, closePopout } = usePopoutWindow();

  const form = useFormWithSave({
    resourceName: "FormulaTemplate",
    formOptions: {
      resolver: zodResolver(formulaTemplateSchema),
      defaultValues: {
        name: "",
        description: "",
        category: "Custom",
        expression: "",
        variables: [],
        parameters: [],
        tags: [],
        examples: [],
        requirements: [],
        minRate: "",
        maxRate: "",
        outputUnit: "USD",
        isActive: true,
        isDefault: false,
      },
    },
    mutationFn: async (values: FormulaTemplateSchema) => {
      const response = await http.post("/formula-templates/", values);
      return response.data;
    },
    onSuccess: () => {
      onOpenChange(false);

      broadcastQueryInvalidation({
        queryKey: ["formulaTemplate", "formula-template-list"],
        options: {
          correlationId: `create-formula-template-${Date.now()}`,
        },
        config: {
          predicate: true,
          refetchType: "all",
        },
      });
    },
    onSettled: () => {
      if (isPopout) {
        closePopout();
      }
    },
  });

  const {
    formState: { isSubmitting, isSubmitSuccessful },
    handleSubmit,
    onSubmit,
    reset,
  } = form;

  // Reset the form when the mutation is successful
  useEffect(() => {
    if (isSubmitSuccessful) {
      reset();
    }
  }, [isSubmitSuccessful, reset]);

  const handleClose = useCallback(() => {
    onOpenChange(false);
    reset();
  }, [onOpenChange, reset]);

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="md:max-w-[700px] lg:max-w-[900px]">
        <VisuallyHidden>
          <DialogHeader>
            <DialogTitle>Create Formula Template</DialogTitle>
            <DialogDescription>
              Create a new formula template for calculating shipment rates.
            </DialogDescription>
          </DialogHeader>
        </VisuallyHidden>
        <FormProvider {...form}>
          <Form className="space-y-0 p-0" onSubmit={handleSubmit(onSubmit)}>
            <DialogBody className="p-0">
              <FormulaTemplateForm />
            </DialogBody>
            <DialogFooter>
              <Button type="button" variant="outline" onClick={handleClose}>
                Cancel
              </Button>
              <FormSaveButton
                isPopout={isPopout}
                isSubmitting={isSubmitting}
                title="Formula Template"
              />
            </DialogFooter>
          </Form>
        </FormProvider>
      </DialogContent>
    </Dialog>
  );
}
