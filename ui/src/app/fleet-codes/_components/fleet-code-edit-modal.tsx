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
import {
  fleetCodeSchema,
  type FleetCodeSchema,
} from "@/lib/schemas/fleet-code-schema";
import { type EditTableSheetProps } from "@/types/data-table";
import { type APIError } from "@/types/errors";
import { yupResolver } from "@hookform/resolvers/yup";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { useCallback } from "react";
import { FormProvider, useForm } from "react-hook-form";
import { toast } from "sonner";
import { FleetCodeForm } from "./fleet-code-form";

function FleetCodeEditForm({
  currentRecord,
  onOpenChange,
}: EditTableSheetProps<FleetCodeSchema>) {
  const queryClient = useQueryClient();
  const { isPopout, closePopout } = usePopoutWindow();

  const form = useForm<FleetCodeSchema>({
    resolver: yupResolver(fleetCodeSchema),
    defaultValues: currentRecord,
  });

  const {
    handleSubmit,
    reset,
    setError,
    formState: { isDirty, isSubmitting },
  } = form;

  const mutation = useMutation({
    mutationFn: async (values: FleetCodeSchema) => {
      const response = await http.put(
        `/fleet-codes/${currentRecord.id}`,
        values,
      );
      return response.data;
    },
    onSuccess: () => {
      toast.success("Fleet Code updated successfully");
      onOpenChange(false);
      form.reset();

      // Invalidate the fleet code list query to refresh the table
      queryClient.invalidateQueries({ queryKey: ["fleet-code-list"] });
    },
    onError: (error: APIError) => {
      if (error.isValidationError()) {
        error.getFieldErrors().forEach((fieldError) => {
          setError(fieldError.name as keyof FleetCodeSchema, {
            message: fieldError.reason,
          });
        });
      }

      toast.error("Failed to update fleet code", {
        description: "Check the form for errors and try again.",
      });
    },
    onSettled: () => {
      if (isPopout) {
        closePopout();
      }
    },
  });

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

  const onSubmit = useCallback(
    async (values: FleetCodeSchema) => {
      await mutation.mutateAsync(values);
    },
    [mutation.mutateAsync],
  );

  return (
    <>
      <FormProvider {...form}>
        <Form onSubmit={handleSubmit(onSubmit)}>
          <DialogBody>
            <FleetCodeForm />
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

export function EditFleetCodeModal({
  open,
  onOpenChange,
  currentRecord,
}: EditTableSheetProps<FleetCodeSchema>) {
  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-[450px]">
        <DialogHeader>
          <DialogTitle>Edit Fleet Code</DialogTitle>
          <DialogDescription>
            Edit the details of the fleet code.
          </DialogDescription>
        </DialogHeader>
        <FleetCodeEditForm
          open={open}
          currentRecord={currentRecord}
          onOpenChange={onOpenChange}
        />
      </DialogContent>
    </Dialog>
  );
}
