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
import { queries } from "@/lib/queries";
import {
  HoldShipmentRequestSchema,
  holdShipmentRequestSchema,
} from "@/lib/schemas/shipment-hold-schema";
import { api } from "@/services/api";
import { zodResolver } from "@hookform/resolvers/zod";
import { useQueryClient } from "@tanstack/react-query";
import { useCallback, useEffect } from "react";
import { FormProvider, useForm } from "react-hook-form";
import { toast } from "sonner";
import { ShipmentHoldForm } from "./shipment-hold-form";

type ShipmentHoldDialogProps = {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  shipmentId?: string;
};

export function ShipmentHoldDialog({
  open,
  onOpenChange,
  shipmentId,
}: ShipmentHoldDialogProps) {
  const queryClient = useQueryClient();
  const form = useForm({
    resolver: zodResolver(holdShipmentRequestSchema),
    defaultValues: {
      shipmentId: shipmentId || "",
      holdReasonId: "",
      orgId: "",
      buId: "",
      userId: "",
    },
  });

  const {
    setError,
    formState: { isSubmitting, isSubmitSuccessful },
    handleSubmit,
    reset,
  } = form;

  const { mutateAsync } = useApiMutation({
    mutationFn: (values: HoldShipmentRequestSchema) =>
      api.shipments.applyHold(values),
    onSuccess: () => {
      toast.success("Shipment hold added successfully", {
        description: `The shipment hold has been added`,
      });

      queryClient.invalidateQueries({
        queryKey: queries.shipment.getHolds(shipmentId).queryKey,
      });
      broadcastQueryInvalidation({
        queryKey: ["shipment", "shipment-list", "stop", "assignment"],
        options: {
          correlationId: `apply-shipment-hold-${Date.now()}`,
        },
        config: {
          predicate: true,
          refetchType: "all",
        },
      });

      onOpenChange(false);
    },
    resourceName: "Shipment Hold",
    setFormError: setError,
  });

  const onSubmit = useCallback(
    async (values: HoldShipmentRequestSchema) => {
      await mutateAsync(values);
    },
    [mutateAsync],
  );

  useEffect(() => {
    if (isSubmitSuccessful) {
      reset();
    }
  }, [isSubmitSuccessful, reset]);

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>Add Hold</DialogTitle>
          <DialogDescription>Add a hold to the shipment.</DialogDescription>
        </DialogHeader>
        <FormProvider {...form}>
          <Form className="space-y-0 p-0">
            <DialogBody>
              <ShipmentHoldForm />
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
                title="shipment hold"
                text="Confirm Hold"
              />
            </DialogFooter>
          </Form>
        </FormProvider>
      </DialogContent>
    </Dialog>
  );
}
