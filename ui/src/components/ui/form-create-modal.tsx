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
import { cn } from "@/lib/utils";
import { type TableSheetProps } from "@/types/data-table";
import { type API_ENDPOINTS } from "@/types/server";
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
  form: UseFormReturn<T>;
  className?: string;
  notice?: React.ReactNode;
};

export function FormCreateModal<T extends FieldValues>({
  open,
  onOpenChange,
  title,
  formComponent,
  form,
  className,
  url,
  queryKey,
  notice,
}: FormCreateModalProps<T>) {
  const { isPopout, closePopout } = usePopoutWindow();
  const queryClient = useQueryClient();

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
    mutationFn: async (values: T) => {
      const response = await http.post<T>(url, values);
      return response.data;
    },
    onSuccess: (data) => {
      toast.success("Changes have been saved.", {
        description: `${title} created successfully`,
      });
      onOpenChange(false);

      // Invalidate the query to refresh the table
      broadcastQueryInvalidation({
        queryKey: [queryKey],
        options: { correlationId: `create-${queryKey}-${Date.now()}` },
        config: {
          predicate: true,
          refetchType: "all",
        },
      });

      queryClient.setQueryData([queryKey], data);

      // * If the page is a popout, close it
      if (isPopout) {
        closePopout();
      }
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

  // Reset the form when the mutation is successful
  // This is recommended by react-hook-form - https://react-hook-form.com/docs/useform/reset
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
    <>
      <Dialog open={open} onOpenChange={onOpenChange}>
        <DialogContent className={cn("max-w-[450px]", className)}>
          <DialogHeader>
            <DialogTitle>Add New {title}</DialogTitle>
            <DialogDescription>
              Please fill out the form below to create a new {title}.
            </DialogDescription>
          </DialogHeader>
          {notice ? notice : null}
          <FormProvider {...form}>
            <Form onSubmit={handleSubmit(onSubmit)}>
              <DialogBody>{formComponent}</DialogBody>
              <DialogFooter>
                <Button type="button" variant="outline" onClick={handleClose}>
                  Cancel
                </Button>
                <FormSaveButton
                  isPopout={isPopout}
                  isSubmitting={isSubmitting}
                  title={title}
                />
              </DialogFooter>
            </Form>
          </FormProvider>
        </DialogContent>
      </Dialog>
    </>
  );
}
