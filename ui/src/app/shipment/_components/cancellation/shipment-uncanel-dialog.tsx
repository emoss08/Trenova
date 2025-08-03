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
import { useApiMutation } from "@/hooks/use-api-mutation";
import { broadcastQueryInvalidation } from "@/hooks/use-invalidate-query";
import {
  shipmentUncancelSchema,
  type ShipmentUncancelSchema,
} from "@/lib/schemas/shipment-cancellation-schema";
import { api } from "@/services/api";
import { Resource } from "@/types/audit-entry";
import { zodResolver } from "@hookform/resolvers/zod";
import { useCallback } from "react";
import { FormProvider, useForm } from "react-hook-form";
import { toast } from "sonner";
import { ShipmentUncancelForm } from "./shipment-uncancel-form";

type ShipmentCancellationDialogProps = {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  shipmentId?: string;
};

export function UnCancelShipmentDialog({
  open,
  onOpenChange,
  shipmentId,
}: ShipmentCancellationDialogProps) {
  const form = useForm({
    resolver: zodResolver(shipmentUncancelSchema),
    defaultValues: {
      shipmentId: shipmentId || "",
      updateAppointments: false,
    },
  });

  const {
    setError,
    formState: { isSubmitting },
    handleSubmit,
    reset,
  } = form;

  const { mutateAsync } = useApiMutation({
    mutationFn: async (values: ShipmentUncancelSchema) => {
      return await api.shipments.uncancel(values);
    },
    onSuccess: () => {
      toast.success("Shipment cancelled successfully", {
        description: `The shipment has been cancelled`,
      });
      onOpenChange(false);
      reset();

      // Invalidate the query to refresh the table
      broadcastQueryInvalidation({
        queryKey: ["assignment-list", "shipment", "stop", "shipment-list"],
        options: {
          correlationId: `create-shipment-move-assignment-${Date.now()}`,
        },
        config: {
          predicate: true,
          refetchType: "all",
        },
      });
    },
    setFormError: setError,
    resourceName: Resource.Shipment,
  });

  const onSubmit = useCallback(
    async (values: ShipmentUncancelSchema) => {
      await mutateAsync(values);
    },
    [mutateAsync],
  );

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-[400px]">
        <DialogHeader>
          <DialogTitle>Cancel Shipment</DialogTitle>
          <DialogDescription>
            Cancel the shipment and all associated moves.
          </DialogDescription>
        </DialogHeader>
        <FormProvider {...form}>
          <Form
            className="space-y-0 p-0"
            onSubmit={(e) => {
              e.preventDefault();
              e.stopPropagation();
              handleSubmit(onSubmit)(e);
            }}
          >
            <DialogBody>
              <ShipmentUncancelForm />
            </DialogBody>
            <DialogFooter>
              <Button
                type="button"
                variant="outline"
                onClick={() => onOpenChange(!open)}
              >
                Cancel
              </Button>
              <FormSaveButton
                type="button"
                onClick={() => handleSubmit(onSubmit)()}
                isSubmitting={isSubmitting}
                title="shipment uncancellation"
                text="Confirm"
              />
            </DialogFooter>
          </Form>
        </FormProvider>
      </DialogContent>
    </Dialog>
  );
}
