/*
 * Copyright 2023-2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

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
import { useFormWithSave } from "@/hooks/use-form-with-save";
import { broadcastQueryInvalidation } from "@/hooks/use-invalidate-query";
import { http } from "@/lib/http-client";
import { workerSchema, WorkerSchema } from "@/lib/schemas/worker-schema";
import { Gender, Status } from "@/types/common";
import { TableSheetProps } from "@/types/data-table";
import { ComplianceStatus, Endorsement, WorkerType } from "@/types/worker";
import { zodResolver } from "@hookform/resolvers/zod";
import { useCallback, useEffect } from "react";
import { FormProvider } from "react-hook-form";
import { WorkerForm } from "./workers-form";

export function CreateWorkerModal({ open, onOpenChange }: TableSheetProps) {
  const { isPopout, closePopout } = usePopoutWindow();

  const form = useFormWithSave({
    resourceName: "Worker",
    formOptions: {
      resolver: zodResolver(workerSchema),
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
    },
    mutationFn: async (values: WorkerSchema) => {
      const response = await http.post("/workers", values);
      return response.data;
    },
    onSuccess: () => {
      onOpenChange(false);

      broadcastQueryInvalidation({
        queryKey: ["worker", "worker-list"],
        options: {
          correlationId: `create-worker-${Date.now()}`,
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
              <Button type="button" variant="outline" onClick={handleClose}>
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
      </DialogContent>
    </Dialog>
  );
}
