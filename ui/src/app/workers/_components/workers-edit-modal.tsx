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
import { VisuallyHidden } from "@/components/ui/visually-hidden";
import { useUnsavedChanges } from "@/hooks/use-form";
import { broadcastQueryInvalidation } from "@/hooks/use-invalidate-query";
import { http } from "@/lib/http-client";
import { workerSchema, WorkerSchema } from "@/lib/schemas/worker-schema";
import { EditTableSheetProps } from "@/types/data-table";
import { type APIError } from "@/types/errors";
import { yupResolver } from "@hookform/resolvers/yup";
import { useMutation } from "@tanstack/react-query";
import { useCallback } from "react";
import { FormProvider, useForm } from "react-hook-form";
import { toast } from "sonner";
import { WorkerForm } from "./workers-form";

function WorkerEditForm({
  currentRecord,
  onOpenChange,
}: EditTableSheetProps<WorkerSchema>) {
  const form = useForm<WorkerSchema>({
    resolver: yupResolver(workerSchema),
    defaultValues: currentRecord,
  });

  const {
    handleSubmit,
    reset,
    setError,
    formState: { isDirty, isSubmitting },
  } = form;

  const mutation = useMutation({
    mutationFn: async (values: WorkerSchema) => {
      const response = await http.put(`/workers/${currentRecord.id}`, values);
      return response.data;
    },
    onSuccess: () => {
      toast.success("Worker updated successfully");
      onOpenChange(false);
      form.reset();

      // Invalidate the worker list query to refresh the table
      broadcastQueryInvalidation({ queryKeys: ["worker-list"] });
    },
    onError: (error: APIError) => {
      if (error.isValidationError()) {
        error.getFieldErrors().forEach((fieldError) => {
          setError(fieldError.name as keyof WorkerSchema, {
            message: fieldError.reason,
          });
        });
      }

      toast.error("Failed to create worker", {
        description: "Check the form for errors and try again.",
      });
    },
  });

  const onSubmit = useCallback(
    async (values: WorkerSchema) => {
      await mutation.mutateAsync(values);
    },
    [mutation.mutateAsync],
  );

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

  return (
    <>
      <FormProvider {...form}>
        <Form className="space-y-0 p-0" onSubmit={handleSubmit(onSubmit)}>
          <DialogBody className="p-0">
            <WorkerForm />
          </DialogBody>
          <DialogFooter>
            <Button type="button" variant="outline" onClick={onClose}>
              Cancel
            </Button>
            <Button type="submit" isLoading={isSubmitting}>
              Save Changes
            </Button>
          </DialogFooter>
        </Form>
      </FormProvider>

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

export function EditWorkerModal({
  open,
  onOpenChange,
  currentRecord,
}: EditTableSheetProps<WorkerSchema>) {
  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="md:max-w-[700px] lg:max-w-[800px]">
        <VisuallyHidden>
          <DialogHeader>
            <DialogTitle>Edit Worker</DialogTitle>
            <DialogDescription>
              Edit the details of the worker.
            </DialogDescription>
          </DialogHeader>
        </VisuallyHidden>
        <WorkerEditForm
          open={open}
          onOpenChange={onOpenChange}
          currentRecord={currentRecord}
        />
      </DialogContent>
    </Dialog>
  );
}
