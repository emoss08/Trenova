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
import { useFormWithSave } from "@/hooks/use-form-with-save";
import { broadcastQueryInvalidation } from "@/hooks/use-invalidate-query";
import { queries } from "@/lib/queries";
import { WorkerPTOSchema, workerPTOSchema } from "@/lib/schemas/worker-schema";
import { api } from "@/services/api";
import { TableSheetProps } from "@/types/data-table";
import { PTOStatus, PTOType } from "@/types/worker";
import { zodResolver } from "@hookform/resolvers/zod";
import { useCallback, useEffect } from "react";
import { FormProvider } from "react-hook-form";
import { PTOForm } from "./pto-form";

export function PTOCreateModal({ open, onOpenChange }: TableSheetProps) {
  const { isPopout, closePopout } = usePopoutWindow();

  const form = useFormWithSave({
    resourceName: "PTO",
    formOptions: {
      resolver: zodResolver(workerPTOSchema),
      defaultValues: {
        status: PTOStatus.Requested,
        type: PTOType.Vacation,
        startDate: undefined,
        endDate: undefined,
        workerId: undefined,
        reason: undefined,
        approverId: undefined,
        rejectorId: undefined,
      },
    },
    mutationFn: async (values: WorkerPTOSchema) => {
      const response = await api.worker.createPTO(values);
      return response;
    },
    onSuccess: () => {
      onOpenChange(false);

      broadcastQueryInvalidation({
        queryKey: [
          "worker-pto-list",
          "worker-list",
          "upcoming-pto",
          ...queries.worker.listUpcomingPTO._def,
        ],
        options: {
          correlationId: `create-pto-${Date.now()}`,
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
    formState: { isSubmitting, isSubmitSuccessful },
    handleSubmit,
    onSubmit,
    reset,
  } = form;

  const handleClose = useCallback(() => {
    onOpenChange(false);
    reset();
  }, [onOpenChange, reset]);

  // Reset the form when the mutation is successful
  // This is recommended by react-hook-form - https://react-hook-form.com/docs/useform/reset
  useEffect(() => {
    reset();
  }, [isSubmitSuccessful, reset]);

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>Add New PTO Request</DialogTitle>
          <DialogDescription>
            Add a new PTO request for a worker.
          </DialogDescription>
        </DialogHeader>
        <FormProvider {...form}>
          <Form onSubmit={handleSubmit(onSubmit)}>
            <DialogBody>
              <PTOForm />
            </DialogBody>
            <DialogFooter>
              <Button type="button" variant="outline" onClick={handleClose}>
                Cancel
              </Button>
              <FormSaveButton
                isPopout={isPopout}
                isSubmitting={isSubmitting}
                title="PTO"
              />
            </DialogFooter>
          </Form>
        </FormProvider>
      </DialogContent>
    </Dialog>
  );
}
