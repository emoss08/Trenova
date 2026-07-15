import { ControlledShipmentAutocompleteField } from "@/components/autocomplete-fields";
import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { FormControl, FormGroup } from "@/components/ui/form";
import { attachOrderShipments } from "@/lib/graphql/order";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { useCallback, useState } from "react";
import { toast } from "sonner";

type AddLegDialogProps = {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  orderId: string;
  customerId?: string;
};

export function AddLegDialog({ open, onOpenChange, orderId, customerId }: AddLegDialogProps) {
  const queryClient = useQueryClient();
  const [shipmentId, setShipmentId] = useState("");

  const handleClose = useCallback(() => {
    onOpenChange(false);
    setShipmentId("");
  }, [onOpenChange]);

  const { mutate, isPending } = useMutation({
    mutationFn: () => attachOrderShipments(orderId, [shipmentId]),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ["order-detail", orderId] });
      toast.success("Leg added", {
        description: "The shipment has been attached to this order.",
      });
      handleClose();
    },
    onError: () => {
      toast.error("Failed to add leg");
    },
  });

  return (
    <Dialog open={open} onOpenChange={(nextOpen) => !nextOpen && handleClose()}>
      <DialogContent className="sm:max-w-100">
        <DialogHeader>
          <DialogTitle>Add Leg</DialogTitle>
          <DialogDescription>
            Attach a shipment to this order as an additional leg.
          </DialogDescription>
        </DialogHeader>
        <FormGroup cols={1} className="pb-4">
          <FormControl>
            <ControlledShipmentAutocompleteField
              value={shipmentId}
              onValueChange={setShipmentId}
              description="Only shipments for this order's customer are shown."
              extraSearchParams={customerId ? { customerId } : undefined}
            />
          </FormControl>
        </FormGroup>
        <DialogFooter>
          <Button type="button" variant="outline" onClick={handleClose}>
            Cancel
          </Button>
          <Button
            type="button"
            disabled={!shipmentId}
            isLoading={isPending}
            loadingText="Adding..."
            onClick={() => mutate()}
          >
            Add Leg
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
