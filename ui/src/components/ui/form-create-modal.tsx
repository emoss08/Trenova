import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from "@/components/ui/alert-dialog";
import { Button } from "@/components/ui/button";
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
import { useUnsavedChanges } from "@/hooks/use-form";
import { http } from "@/lib/http-client";
import { cn } from "@/lib/utils";
import { type TableSheetProps } from "@/types/data-table";
import { type APIError } from "@/types/errors";
import { type API_ENDPOINTS } from "@/types/server";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { useCallback } from "react";
import {
  type FieldValues,
  FormProvider,
  type Path,
  type UseFormReturn,
} from "react-hook-form";
import { toast } from "sonner";
import { type ObjectSchema } from "yup";

type FormCreateModalProps<T extends FieldValues> = TableSheetProps & {
  url: API_ENDPOINTS;
  title: string;
  queryKey: string;
  formComponent: React.ReactNode;
  form: UseFormReturn<T>;
  schema: ObjectSchema<T>;
  className?: string;
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
}: FormCreateModalProps<T>) {
  const queryClient = useQueryClient();
  const { isPopout, closePopout } = usePopoutWindow();

  const {
    setError,
    formState: { isDirty, isSubmitting },
    handleSubmit,
    reset,
  } = form;

  const handleClose = useCallback(() => {
    onOpenChange(false);
    reset();
  }, [onOpenChange, reset]);

  const {
    showWarning,
    handleClose: onClose,
    handleConfirmClose,
    handleCancelClose,
  } = useUnsavedChanges({
    isDirty,
    onClose: handleClose,
  });

  const mutation = useMutation({
    mutationFn: async (values: T) => {
      const response = await http.post(url, values);
      return response.data;
    },
    onSuccess: () => {
      toast.success(`${title} created successfully`);
      onOpenChange(false);
      form.reset();

      // Invalidate the equipment type list query to refresh the table
      queryClient.invalidateQueries({ queryKey: [queryKey] });
    },
    onError: (error: APIError) => {
      if (error.isValidationError()) {
        error.getFieldErrors().forEach((fieldError) => {
          setError(fieldError.name as Path<T>, {
            message: fieldError.reason,
          });
        });
      }

      if (error.isRateLimitError()) {
        toast.error("Rate limit exceeded", {
          description:
            "You have exceeded the rate limit. Please try again later.",
        });
      }
    },
    onSettled: () => {
      if (isPopout) {
        closePopout();
      }
    },
  });

  const onSubmit = useCallback(
    async (values: T) => {
      await mutation.mutateAsync(values);
    },
    [mutation.mutateAsync],
  );

  return (
    <>
      <Dialog open={open} onOpenChange={onClose}>
        <DialogContent className={cn("max-w-[450px]", className)}>
          <DialogHeader>
            <DialogTitle>Add New {title}</DialogTitle>
            <DialogDescription>
              Please fill out the form below to create a new {title}.
            </DialogDescription>
          </DialogHeader>
          <FormProvider {...form}>
            <Form onSubmit={handleSubmit(onSubmit)}>
              <DialogBody>{formComponent}</DialogBody>
              <DialogFooter>
                <Button type="button" variant="outline" onClick={onClose}>
                  Cancel
                </Button>
                <Button type="submit" isLoading={isSubmitting}>
                  Save {isPopout ? "and Close" : "Changes"}
                </Button>
              </DialogFooter>
            </Form>
          </FormProvider>
        </DialogContent>
      </Dialog>

      {showWarning && (
        <AlertDialog open={showWarning} onOpenChange={handleCancelClose}>
          <AlertDialogContent>
            <AlertDialogHeader>
              <AlertDialogTitle>Unsaved Changes</AlertDialogTitle>
              <AlertDialogDescription>
                You have unsaved changes. Are you sure you want to close this
                form? All changes will be lost.
              </AlertDialogDescription>
            </AlertDialogHeader>
            <AlertDialogFooter>
              <AlertDialogCancel onClick={handleCancelClose}>
                Continue Editing
              </AlertDialogCancel>
              <AlertDialogAction onClick={handleConfirmClose}>
                Discard Changes
              </AlertDialogAction>
            </AlertDialogFooter>
          </AlertDialogContent>
        </AlertDialog>
      )}
    </>
  );
}
