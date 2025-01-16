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
import { usePopoutWindow } from "@/hooks/popout-window/use-popout-window";
import { useUnsavedChanges } from "@/hooks/use-form";
import { http } from "@/lib/http-client";
import { workerSchema, WorkerSchema } from "@/lib/schemas/worker-schema";
import { Gender, Status } from "@/types/common";
import { TableSheetProps } from "@/types/data-table";
import { APIError } from "@/types/errors";
import { ComplianceStatus, Endorsement, WorkerType } from "@/types/worker";
import { yupResolver } from "@hookform/resolvers/yup";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { useCallback } from "react";
import { FormProvider, useForm } from "react-hook-form";
import { toast } from "sonner";
import { WorkerForm } from "./workers-form";

export function CreateWorkerModal({ open, onOpenChange }: TableSheetProps) {
  const queryClient = useQueryClient();
  const { isPopout, closePopout } = usePopoutWindow();

  const form = useForm<WorkerSchema>({
    resolver: yupResolver(workerSchema),
    defaultValues: {
      status: Status.Active,
      type: WorkerType.Employee,
      gender: Gender.Male,
      firstName: "",
      lastName: "",
      addressLine1: "",
      addressLine2: "",
      city: "",
      stateId: "",
      postalCode: "",
      profile: {
        licenseNumber: "",
        licenseStateId: "",
        complianceStatus: ComplianceStatus.Pending,
        isQualified: true,
        endorsement: Endorsement.None,
        terminationDate: undefined,
        physicalDueDate: undefined,
        mvrDueDate: undefined,
        dob: undefined,
        hazmatExpiry: undefined,
        hireDate: undefined,
        licenseExpiry: undefined,
        lastMvrCheck: undefined,
        lastDrugTest: undefined,
      },
    },
  });

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
    mutationFn: async (values: WorkerSchema) => {
      const response = await http.post("/workers", values);
      return response.data;
    },
    onSuccess: () => {
      toast.success("Worker created successfully");
      onOpenChange(false);
      form.reset();

      // Invalidate the worker list query to refresh the table
      queryClient.invalidateQueries({ queryKey: ["worker-list"] });
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
    onSettled: () => {
      if (isPopout) {
        closePopout();
      }
    },
  });

  const onSubmit = useCallback(
    async (values: WorkerSchema) => {
      await mutation.mutateAsync(values);
    },
    [mutation.mutateAsync],
  );

  return (
    <>
      <Dialog open={open} onOpenChange={onClose}>
        <DialogContent className="md:max-w-[700px] lg:max-w-[800px]">
          <VisuallyHidden>
            <DialogHeader>
              <DialogTitle>Create Worker</DialogTitle>
              <DialogDescription>
                Create a new worker to manage their time and attendance.
              </DialogDescription>
            </DialogHeader>
          </VisuallyHidden>
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
