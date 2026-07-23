import { Button } from "@trenova/shared/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@trenova/shared/components/ui/dialog";
import { Form } from "@trenova/shared/components/ui/form";
import { useApiMutation } from "@/hooks/use-api-mutation";
import { api } from "@trenova/shared/lib/api";
import { cn } from "@trenova/shared/lib/utils";
import type { TableSheetProps } from "@trenova/shared/types/data-table";
import type { API_ENDPOINTS } from "@trenova/shared/types/server";
import { useQueryClient } from "@tanstack/react-query";
import { useCallback, useEffect } from "react";
import { type FieldValues, FormProvider, type UseFormReturn } from "react-hook-form";
import { toast } from "sonner";

type FormCreateModalProps<T extends FieldValues, TResponse = unknown> = TableSheetProps & {
  url: API_ENDPOINTS;
  title: string;
  queryKey: string;
  formComponent: React.ReactNode;
  description?: string;
  form: UseFormReturn<T>;
  className?: string;
  notice?: React.ReactNode;
  onSuccess?: (data: TResponse, values: T) => void | Promise<void>;
  submitText?: string;
  loadingText?: string;
};

export function FormCreateModal<T extends FieldValues, TResponse = unknown>({
  open,
  onOpenChange,
  description,
  title,
  formComponent,
  form,
  className,
  url,
  queryKey,
  notice,
  onSuccess,
  submitText = "Save and Close",
  loadingText = "Saving...",
}: FormCreateModalProps<T, TResponse>) {
  const queryClient = useQueryClient();

  const {
    setError,
    formState: { isSubmitting },
    handleSubmit,
    reset,
  } = form;

  const handleClose = useCallback(() => {
    onOpenChange(false);
    reset();
  }, [onOpenChange, reset]);

  const { mutateAsync } = useApiMutation({
    mutationFn: async (values: T) => {
      return api.post<TResponse>(url, values);
    },
    onSuccess: async (data, values) => {
      toast.success("Changes have been saved.", {
        description: `${title} created successfully`,
      });
      onOpenChange(false);
      reset();
      await onSuccess?.(data, values);

      queryClient.setQueryData([queryKey], data);
    },
    setFormError: setError,
    resourceName: title,
  });

  const onSubmit = useCallback(
    async (values: T) => {
      await mutateAsync(values);
    },
    [mutateAsync],
  );

  useEffect(() => {
    const handleKeyDown = (event: KeyboardEvent) => {
      if (open && (event.ctrlKey || event.metaKey) && event.key === "Enter" && !isSubmitting) {
        event.preventDefault();
        void handleSubmit(onSubmit)();
      }
    };

    document.addEventListener("keydown", handleKeyDown);
    return () => document.removeEventListener("keydown", handleKeyDown);
  }, [open, isSubmitting, handleSubmit, onSubmit]);

  return (
    <Dialog
      open={open}
      onOpenChange={(nextOpen) => {
        if (nextOpen) {
          onOpenChange(true);
          return;
        }
        handleClose();
      }}
    >
      <DialogContent className={cn("max-w-[450px]", className)}>
        <DialogHeader>
          <DialogTitle>Add New {title}</DialogTitle>
          <DialogDescription>
            {description ? description : `Please fill out the form below to create a new ${title}.`}
          </DialogDescription>
        </DialogHeader>
        {notice ? notice : null}
        <FormProvider {...form}>
          <Form onSubmit={handleSubmit(onSubmit)}>
            {formComponent}
            <DialogFooter>
              <Button type="button" variant="outline" onClick={handleClose}>
                Cancel
              </Button>
              <Button type="submit" isLoading={isSubmitting} loadingText={loadingText}>
                {submitText}
              </Button>
            </DialogFooter>
          </Form>
        </FormProvider>
      </DialogContent>
    </Dialog>
  );
}
