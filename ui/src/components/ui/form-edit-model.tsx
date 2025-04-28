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
import { usePopoutWindow } from "@/hooks/popout-window/use-popout-window";
import { useApiMutation } from "@/hooks/use-api-mutation";
import { broadcastQueryInvalidation } from "@/hooks/use-invalidate-query";
import { formatToUserTimezone } from "@/lib/date";
import { http } from "@/lib/http-client";
import { cn } from "@/lib/utils";
import { useUser } from "@/stores/user-store";
import { type EditTableSheetProps } from "@/types/data-table";
import { type API_ENDPOINTS } from "@/types/server";
import { useQueryClient } from "@tanstack/react-query";
import { useCallback, useEffect, useRef } from "react";
import {
  FormProvider,
  type FieldValues,
  type UseFormReturn,
} from "react-hook-form";
import { toast } from "sonner";
import { ComponentLoader } from "./component-loader";
import { Form } from "./form";

type FormEditModalProps<T extends FieldValues> = EditTableSheetProps<T> & {
  url: API_ENDPOINTS;
  title: string;
  queryKey: string;
  formComponent: React.ReactNode;
  form: UseFormReturn<T>;
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
  isLoading,
  error,
}: FormEditModalProps<T>) {
  const queryClient = useQueryClient();

  const { isPopout, closePopout } = usePopoutWindow();
  const user = useUser();
  const previousRecordIdRef = useRef<string | number | null>(null);

  const {
    setError,
    formState: { isSubmitting },
    handleSubmit,
    reset,
  } = form;

  // Update form values when currentRecord changes and is not loading
  useEffect(() => {
    if (
      !isLoading &&
      currentRecord &&
      currentRecord.id !== previousRecordIdRef.current
    ) {
      reset(currentRecord);
      previousRecordIdRef.current = currentRecord.id;
    }
  }, [currentRecord, isLoading, reset]);

  const handleClose = useCallback(() => {
    onOpenChange(false);
    reset();
  }, [onOpenChange, reset]);

  const { mutateAsync } = useApiMutation<
    T, // The response data type
    T, // The variables type
    unknown, // The context type
    T // The form values type for error handling
  >({
    mutationFn: async (values: T) => {
      const response = await http.put<T>(`${url}${currentRecord?.id}`, values);
      return response.data;
    },
    onMutate: async (newValues) => {
      // * Cancel any outgoing refetches so they don't overwrite our optmistic update
      await queryClient.cancelQueries({
        queryKey: [queryKey],
      });

      // * Snapshot the previous value
      const previousRecord = queryClient.getQueryData([queryKey]);

      // * Optimistically update to the new value
      queryClient.setQueryData([queryKey], newValues);

      return { previousRecord, newValues };
    },
    onSuccess: (newValues) => {
      toast.success("Changes have been saved", {
        description: `${title} updated successfully`,
      });

      // * Close the modal on success
      onOpenChange(false);

      // * Invalidate the query
      broadcastQueryInvalidation({
        queryKey: [queryKey],
        options: {
          correlationId: `update-${queryKey}-${Date.now()}`,
        },
        config: {
          predicate: true,
          refetchType: "all",
        },
      });

      // * Reset the form to the new values
      reset(newValues);

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

  // If there's an error, show a toast and close the modal
  useEffect(() => {
    if (error) {
      toast.error("Failed to load record", {
        description: "Please try again in a moment",
      });
      onOpenChange(false);
    }
  }, [error, onOpenChange]);

  const dialogContent = (
    <DialogContent
      withClose={!isLoading}
      className={cn("max-w-[450px]", className)}
    >
      <DialogHeader>
        <DialogTitle>
          <div>
            {isLoading
              ? "Loading record..."
              : fieldKey && currentRecord
                ? currentRecord[fieldKey]
                : title}
          </div>
        </DialogTitle>
        {!isLoading && currentRecord && (
          <DialogDescription>
            Last updated on{" "}
            {formatToUserTimezone(currentRecord.updatedAt, {
              timezone: user?.timezone,
              timeFormat: user?.timeFormat,
            })}
          </DialogDescription>
        )}
      </DialogHeader>
      <FormProvider {...form}>
        <Form onSubmit={handleSubmit(onSubmit)}>
          <DialogBody>
            {isLoading ? (
              <ComponentLoader message={`Loading ${title}...`} />
            ) : (
              formComponent
            )}
          </DialogBody>
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
  );

  return (
    <>
      <Dialog open={open} onOpenChange={onOpenChange}>
        {dialogContent}
      </Dialog>

      {/* {showWarning && (
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
      )} */}
    </>
  );
}
