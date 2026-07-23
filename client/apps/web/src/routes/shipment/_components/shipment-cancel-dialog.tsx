import { TextareaField } from "@/components/fields/textarea-field";
import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Form, FormControl, FormGroup } from "@/components/ui/form";
import { useApiMutation } from "@/hooks/use-api-mutation";
import { apiService } from "@/services/api";
import { useQueryClient } from "@tanstack/react-query";
import { useCallback } from "react";
import { useForm } from "react-hook-form";
import { toast } from "sonner";

type CancelFormValues = {
  cancelReason: string;
};

type ShipmentCancelDialogProps = {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  shipmentId: string;
};

export function ShipmentCancelDialog({
  open,
  onOpenChange,
  shipmentId,
}: ShipmentCancelDialogProps) {
  const queryClient = useQueryClient();

  const form = useForm<CancelFormValues>({
    defaultValues: {
      cancelReason: "",
    },
  });

  const {
    control,
    handleSubmit,
    reset,
    setError,
    formState: { isSubmitting },
  } = form;

  const { mutateAsync } = useApiMutation({
    mutationFn: (values: CancelFormValues) =>
      apiService.shipmentService.cancel(shipmentId, values.cancelReason),
    resourceName: "Shipment",
    setFormError: setError,
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ["shipment-list"] });
      toast.success("Shipment canceled", {
        description: "The shipment has been canceled.",
      });
    },
  });

  const handleClose = useCallback(() => {
    onOpenChange(false);
    reset({ cancelReason: "" });
  }, [onOpenChange, reset]);

  const onSubmit = useCallback(
    async (values: CancelFormValues) => {
      await mutateAsync(values);
      handleClose();
    },
    [mutateAsync, handleClose],
  );

  return (
    <Dialog open={open} onOpenChange={(nextOpen) => !nextOpen && handleClose()}>
      <DialogContent className="sm:max-w-100">
        <DialogHeader>
          <DialogTitle>Cancel Shipment</DialogTitle>
          <DialogDescription>
            Are you sure you want to cancel this shipment? You can optionally provide a reason.
          </DialogDescription>
        </DialogHeader>
        <Form
          onSubmit={(e) => {
            e.stopPropagation();
            void handleSubmit(onSubmit)(e);
          }}
        >
          <FormGroup cols={1} className="pb-4">
            <FormControl>
              <TextareaField
                control={control}
                name="cancelReason"
                label="Cancel Reason"
                placeholder="Optional reason for cancellation..."
              />
            </FormControl>
          </FormGroup>
          <DialogFooter>
            <Button type="button" variant="outline" onClick={handleClose}>
              Close
            </Button>
            <Button
              type="submit"
              variant="destructive"
              isLoading={isSubmitting}
              loadingText="Canceling..."
            >
              Cancel Shipment
            </Button>
          </DialogFooter>
        </Form>
      </DialogContent>
    </Dialog>
  );
}
