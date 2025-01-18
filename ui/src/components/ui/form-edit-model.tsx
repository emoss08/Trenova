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
import { usePopoutWindow } from "@/hooks/popout-window/use-popout-window";
import { useUnsavedChanges } from "@/hooks/use-form";
import { formatToUserTimezone } from "@/lib/date";
import { http } from "@/lib/http-client";
import { cn } from "@/lib/utils";
import { type EditTableSheetProps } from "@/types/data-table";
import { APIError } from "@/types/errors";
import { type API_ENDPOINTS } from "@/types/server";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { useCallback } from "react";
import {
  FormProvider,
  Path,
  type FieldValues,
  type UseFormReturn,
} from "react-hook-form";
import { toast } from "sonner";
import { type ObjectSchema } from "yup";
import { Badge } from "./badge";
import { Form } from "./form";

type FormEditModalProps<T extends FieldValues> = EditTableSheetProps<T> & {
  url: API_ENDPOINTS;
  title: string;
  queryKey: string;
  formComponent: React.ReactNode;
  form: UseFormReturn<T>;
  schema: ObjectSchema<T>;
  className?: string;
  fieldKey?: keyof T;
};

export function FormEditModal<T extends FieldValues>({
  open,
  onOpenChange,
  currentRecord,
  url,
  title,
  formComponent,
  queryKey,
  fieldKey,
  form,
  className,
}: FormEditModalProps<T>) {
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

  const mutation = useMutation({
    mutationFn: async (values: T) => {
      const response = await http.put(`${url}${currentRecord.id}`, values);
      return response.data;
    },
    onSuccess: () => {
      toast.success(`${title} Updated Successfully`);
      onOpenChange(false);
      reset();

      // Invalidate the query to refresh the table
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

  const {
    showWarning,
    handleClose: onClose,
    handleConfirmClose,
    handleCancelClose,
  } = useUnsavedChanges({
    isDirty,
    onClose: handleClose,
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
            <DialogTitle>
              <div>{fieldKey ? currentRecord[fieldKey] : title}</div>
              <Badge variant="purple" withDot={false} className="text-3xs">
                {currentRecord.id}
              </Badge>
            </DialogTitle>
            <DialogDescription>
              Last updated on {formatToUserTimezone(currentRecord.updatedAt)}
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
