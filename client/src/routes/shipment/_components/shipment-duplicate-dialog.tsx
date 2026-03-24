import { NumberField } from "@/components/fields/number-field";
import { SwitchField } from "@/components/fields/switch-field";
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
import type { DuplicateShipmentRequest } from "@/types/shipment";
import { duplicateShipmentRequestSchema } from "@/types/shipment";
import { zodResolver } from "@hookform/resolvers/zod";
import { useQueryClient } from "@tanstack/react-query";
import { useCallback } from "react";
import { useForm } from "react-hook-form";
import { toast } from "sonner";

type ShipmentDuplicateDialogProps = {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  shipmentId: string;
};

export function ShipmentDuplicateDialog({
  open,
  onOpenChange,
  shipmentId,
}: ShipmentDuplicateDialogProps) {
  const queryClient = useQueryClient();

  const form = useForm({
    resolver: zodResolver(duplicateShipmentRequestSchema),
    defaultValues: {
      shipmentId,
      count: 1,
      overrideDates: false,
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
    mutationFn: (payload: DuplicateShipmentRequest) =>
      apiService.shipmentService.duplicate(payload),
    resourceName: "Shipment",
    setFormError: setError,
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ["shipment-list"] });
      toast.success("Shipment duplication started", {
        description: "The shipment will be duplicated in the background.",
      });
    },
  });

  const handleClose = useCallback(() => {
    onOpenChange(false);
    reset({ shipmentId, count: 1, overrideDates: false });
  }, [onOpenChange, reset, shipmentId]);

  const onSubmit = useCallback(
    async (values: DuplicateShipmentRequest) => {
      await mutateAsync(values);
      handleClose();
    },
    [mutateAsync, handleClose],
  );

  return (
    <Dialog open={open} onOpenChange={(nextOpen) => !nextOpen && handleClose()}>
      <DialogContent className="sm:max-w-100">
        <DialogHeader>
          <DialogTitle>Duplicate Shipment</DialogTitle>
          <DialogDescription>Create one or more copies of this shipment.</DialogDescription>
        </DialogHeader>
        <Form
          onSubmit={(e) => {
            e.stopPropagation();
            void handleSubmit(onSubmit)(e);
          }}
        >
          <FormGroup cols={1} className="pb-4">
            <FormControl>
              <NumberField
                control={control}
                name="count"
                label="Number of Copies"
                placeholder="1"
                min={1}
                max={20}
                rules={{ required: true }}
              />
            </FormControl>
            <FormControl>
              <SwitchField
                control={control}
                name="overrideDates"
                label="Override Dates"
                description="Reset planned arrival and departure times on the duplicated shipment stops."
              />
            </FormControl>
          </FormGroup>
          <DialogFooter>
            <Button type="button" variant="outline" onClick={handleClose}>
              Cancel
            </Button>
            <Button type="submit" isLoading={isSubmitting} loadingText="Duplicating...">
              Duplicate
            </Button>
          </DialogFooter>
        </Form>
      </DialogContent>
    </Dialog>
  );
}
