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
  FleetCodeSchema,
  fleetCodeSchema,
} from "@/lib/schemas/fleet-code-schema";
import { Status } from "@/types/common";
import { type TableSheetProps } from "@/types/data-table";
import { APIError } from "@/types/errors";
import { yupResolver } from "@hookform/resolvers/yup";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { useCallback } from "react";
import { FormProvider, useForm } from "react-hook-form";
import { toast } from "sonner";
import { FleetCodeForm } from "./fleet-code-form";

export function CreateFleetCodeModal({ open, onOpenChange }: TableSheetProps) {
  const queryClient = useQueryClient();
  const { isPopout, closePopout } = usePopoutWindow();

  const form = useForm<FleetCodeSchema>({
    resolver: yupResolver(fleetCodeSchema),
    defaultValues: {
      name: "",
      status: Status.Active,
      description: "",
      managerId: undefined,
      revenueGoal: undefined,
      deadheadGoal: undefined,
      color: "",
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
    mutationFn: async (values: FleetCodeSchema) => {
      const response = await http.post("/fleet-codes", values);
      return response.data;
    },
    onSuccess: () => {
      toast.success("Fleet Code created successfully");
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
    async (values: FleetCodeSchema) => {
      await mutation.mutateAsync(values);
    },
    [mutation.mutateAsync],
  );

  return (
    <>
      <Dialog open={open} onOpenChange={onClose}>
        <DialogContent className="max-w-[450px]">
          <DialogHeader>
            <DialogTitle>Add New Fleet Code</DialogTitle>
            <DialogDescription>
              Please fill out the form below to create a new Fleet Code.
            </DialogDescription>
          </DialogHeader>
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
