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
import { useUnsavedChanges } from "@/hooks/use-form";
import { useFormWithSave } from "@/hooks/use-form-with-save";
import { broadcastQueryInvalidation } from "@/hooks/use-invalidate-query";
import { http } from "@/lib/http-client";
import { workerSchema, WorkerSchema } from "@/lib/schemas/worker-schema";
import { EditTableSheetProps } from "@/types/data-table";
import { zodResolver } from "@hookform/resolvers/zod";
import { useCallback, useEffect } from "react";
import { FormProvider } from "react-hook-form";
import { WorkerForm } from "./workers-form";

function WorkerEditForm({
  currentRecord,
  onOpenChange,
}: EditTableSheetProps<WorkerSchema>) {
  const { isPopout, closePopout } = usePopoutWindow();

  const form = useFormWithSave({
    resourceName: "Worker",
    formOptions: {
      resolver: zodResolver(workerSchema),
      defaultValues: currentRecord,
      mode: "onChange",
    },
    mutationFn: async (values: WorkerSchema) => {
      const response = await http.put(`/workers/${currentRecord?.id}`, values);
      return response.data;
    },
    onSuccess: () => {
      onOpenChange(false);
      broadcastQueryInvalidation({
        queryKey: ["worker", "worker-list"],
        options: {
          correlationId: `update-worker-${Date.now()}`,
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
    handleSubmit,
    reset,
    onSubmit,
    formState: { isDirty, isSubmitting, isSubmitSuccessful },
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

  // Make sure we populate the form with the current record
  useEffect(() => {
    if (currentRecord) {
      reset(currentRecord);
    }
  }, [currentRecord, reset]);

  // Reset the form when the mutation is successful
  // This is recommended by react-hook-form - https://react-hook-form.com/docs/useform/reset
  useEffect(() => {
    reset();
  }, [isSubmitSuccessful, currentRecord, reset, onOpenChange]);

  useEffect(() => {
    const handleKeyDown = (event: KeyboardEvent) => {
      if (
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
  }, [isSubmitting, handleSubmit, onSubmit]);

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
            <FormSaveButton
              isPopout={isPopout}
              isSubmitting={isSubmitting}
              title="Worker"
            />
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
