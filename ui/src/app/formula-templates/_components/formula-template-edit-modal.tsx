/* eslint-disable react-hooks/exhaustive-deps */
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
import { VisuallyHidden } from "@/components/ui/visually-hidden";
import { usePopoutWindow } from "@/hooks/popout-window/use-popout-window";
import { useApiMutation } from "@/hooks/use-api-mutation";
import { searchParamsParser } from "@/hooks/use-data-table-state";
import { broadcastQueryInvalidation } from "@/hooks/use-invalidate-query";
import { http } from "@/lib/http-client";
import {
  formulaTemplateSchema,
  type FormulaTemplateSchema,
} from "@/lib/schemas/formula-template-schema";
import { type EditTableSheetProps } from "@/types/data-table";
import { zodResolver } from "@hookform/resolvers/zod";
import { useQueryClient } from "@tanstack/react-query";
import { useQueryStates } from "nuqs";
import { useCallback, useEffect, useRef } from "react";
import { FormProvider, useForm } from "react-hook-form";
import { toast } from "sonner";
import { FormulaTemplateForm } from "./formula-template-form";

export function FormulaTemplateEditForm({
  currentRecord,
}: EditTableSheetProps<FormulaTemplateSchema>) {
  const { table } = useDataTable();
  const queryClient = useQueryClient();
  const [, setSearchParams] = useQueryStates(searchParamsParser);
  const { isPopout, closePopout } = usePopoutWindow();
  const previousRecordIdRef = useRef<string | number | null>(null);

  const form = useForm({
    resolver: zodResolver(formulaTemplateSchema),
    defaultValues: currentRecord,
    mode: "onChange",
  });

  const {
    setError,
    reset,
    handleSubmit,
    formState: { isSubmitting },
  } = form;

  const { mutateAsync } = useApiMutation({
    mutationFn: async (values: FormulaTemplateSchema) => {
      const response = await http.put<FormulaTemplateSchema>(
        `/formula-templates/${currentRecord?.id}/`,
        values,
      );
      return response.data;
    },
    onMutate: async (newValues) => {
      await queryClient.cancelQueries({
        queryKey: ["formulaTemplate", currentRecord?.id],
      });

      const previousTemplate = queryClient.getQueryData([
        "formulaTemplate",
        currentRecord?.id,
      ]);

      queryClient.setQueryData(
        ["formulaTemplate", currentRecord?.id],
        newValues,
      );

      return { previousTemplate, newValues };
    },
    onSuccess: (newValues) => {
      toast.success("Changes have been saved", {
        description: `Formula template updated successfully`,
      });

      broadcastQueryInvalidation({
        queryKey: ["formulaTemplate", "formula-template-list"],
        options: {
          correlationId: `update-formula-template-${Date.now()}`,
        },
        config: {
          predicate: true,
          refetchType: "all",
        },
      });

      reset(newValues);
      table.resetRowSelection();
    },
    onError: (error: any) => {
      const errorMessage =
        error?.response?.data?.message || "Failed to update formula template";

      if (error?.response?.data?.errors) {
        const errors = error.response.data.errors;
        Object.keys(errors).forEach((key) => {
          setError(key as keyof FormulaTemplateSchema, {
            type: "manual",
            message: errors[key],
          });
        });
      }

      toast.error("Error", { description: errorMessage });
    },
  });

  const onSubmit = async (values: FormulaTemplateSchema) => {
    await mutateAsync(values);

    if (isPopout) {
      closePopout();
    }
  };

  const handleCancel = useCallback(() => {
    setSearchParams({ id: null, edit: null });
    reset();
  }, [setSearchParams, reset]);

  // Update form when record changes (navigation between rows)
  useEffect(() => {
    if (
      currentRecord &&
      previousRecordIdRef.current !== currentRecord.id &&
      !form.formState.isDirty
    ) {
      reset(currentRecord);
      previousRecordIdRef.current = currentRecord.id || null;
    }
  }, [currentRecord, reset]);

  return (
    <FormProvider {...form}>
      <Form className="space-y-0 p-0" onSubmit={handleSubmit(onSubmit)}>
        <DialogBody className="p-0">
          <FormulaTemplateForm />
        </DialogBody>
        <DialogFooter>
          <Button type="button" variant="outline" onClick={handleCancel}>
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
  );
}

export function EditFormulaTemplateModal(
  props: EditTableSheetProps<FormulaTemplateSchema>,
) {
  const [searchParams] = useQueryStates(searchParamsParser);

  return (
    <Dialog
      open={searchParams.edit === "true"}
      onOpenChange={(value) => {
        if (!value) {
          props.onClose?.();
        }
      }}
    >
      <DialogContent className="md:max-w-[700px] lg:max-w-[900px]">
        <VisuallyHidden>
          <DialogHeader>
            <DialogTitle>Edit Formula Template</DialogTitle>
            <DialogDescription>
              Make changes to the formula template. Click save when you're done.
            </DialogDescription>
          </DialogHeader>
        </VisuallyHidden>
        <FormulaTemplateEditForm {...props} />
      </DialogContent>
    </Dialog>
  );
}
