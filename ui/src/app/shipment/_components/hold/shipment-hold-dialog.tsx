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
import { http } from "@/lib/http-client";
import {
  HoldShipmentRequestSchema,
  holdShipmentRequestSchema,
} from "@/lib/schemas/shipment-hold-schema";
import { zodResolver } from "@hookform/resolvers/zod";
import { useMutation } from "@tanstack/react-query";
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

  const { mutateAsync } = useMutation({
    mutationFn: async (values: HoldShipmentRequestSchema) => {
      const response = await http.post(`/shipment-holds/`, values);
      return response.data;
    },
    onSuccess: () => {
      toast.success("Shipment hold added successfully", {
        description: `The shipment hold has been added`,
      });
    },
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
                variant="destructive"
              />
            </DialogFooter>
          </Form>
        </FormProvider>
      </DialogContent>
    </Dialog>
  );
}
