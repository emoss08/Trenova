import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Form } from "@/components/ui/form";
import { useApiMutation } from "@/hooks/use-api-mutation";
import { api } from "@/lib/api";
import { cn } from "@/lib/utils";
import type { TableSheetProps } from "@/types/data-table";
import type { API_ENDPOINTS } from "@/types/server";
import { useQueryClient } from "@tanstack/react-query";
import { useCallback, useEffect } from "react";
import {
  type FieldValues,
  FormProvider,
  type UseFormReturn,
} from "react-hook-form";
import { toast } from "sonner";

type FormCreateModalProps<T extends FieldValues> = TableSheetProps & {
  url: API_ENDPOINTS;
  title: string;
  queryKey: string;
  formComponent: React.ReactNode;
  description?: string;
  form: UseFormReturn<T>;
  className?: string;
  notice?: React.ReactNode;
};

export function FormCreateModal<T extends FieldValues>({
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
}: FormCreateModalProps<T>) {
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
      return api.post<T>(url, values);
    },
    onSuccess: (data) => {
      toast.success("Changes have been saved.", {
        description: `${title} created successfully`,
      });
      onOpenChange(false);
      reset();

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
      if (
        open &&
        (event.ctrlKey || event.metaKey) &&
        event.key === "Enter" &&
        !isSubmitting
      ) {
        event.preventDefault();
        void handleSubmit(onSubmit)();
      }
    };

    document.addEventListener("keydown", handleKeyDown);
    return () => document.removeEventListener("keydown", handleKeyDown);
  }, [open, isSubmitting, handleSubmit, onSubmit]);

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className={cn("max-w-[450px]", className)}>
        <DialogHeader>
          <DialogTitle>Add New {title}</DialogTitle>
          <DialogDescription>
            {description
              ? description
              : `Please fill out the form below to create a new ${title}.`}
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
              <Button
                type="submit"
                isLoading={isSubmitting}
                loadingText="Saving..."
              >
                Save and Close
              </Button>
            </DialogFooter>
          </Form>
        </FormProvider>
      </DialogContent>
    </Dialog>
  );
}
