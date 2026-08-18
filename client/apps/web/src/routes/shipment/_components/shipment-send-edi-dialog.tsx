import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from "@trenova/shared/components/ui/alert-dialog";
import { apiService } from "@/services/api";
import type { Shipment } from "@trenova/shared/types/shipment";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";

type ShipmentSendEDIDialogProps = {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  shipment: Shipment;
};

export function ShipmentSendEDIDialog({
  open,
  onOpenChange,
  shipment,
}: ShipmentSendEDIDialogProps) {
  const queryClient = useQueryClient();
  const ediPartner = shipment.customer?.ediPartner;
  const mutation = useMutation({
    mutationFn: () =>
      apiService.ediService.submitLoadTender({ sourceShipmentId: shipment.id ?? "" }),
    onSuccess: async () => {
      toast.success("EDI load tender submitted");
      await Promise.all([
        queryClient.invalidateQueries({ queryKey: ["edi-outbound-transfer-list"] }),
        queryClient.invalidateQueries({ queryKey: ["shipment-list"] }),
      ]);
    },
    onError: () => toast.error("Failed to submit EDI load tender"),
  });

  return (
    <AlertDialog open={open} onOpenChange={onOpenChange}>
      <AlertDialogContent>
        <AlertDialogHeader>
          <AlertDialogTitle>Send EDI Load Tender</AlertDialogTitle>
          <AlertDialogDescription>
            {shipment.proNumber ?? "This shipment"} will be tendered to{" "}
            <span className="text-foreground font-medium">
              {ediPartner
                ? `${ediPartner.name} (${ediPartner.code})`
                : "the customer's EDI partner"}
            </span>{" "}
            for approval by the receiving organization.
          </AlertDialogDescription>
        </AlertDialogHeader>
        <AlertDialogFooter>
          <AlertDialogCancel>Cancel</AlertDialogCancel>
          <AlertDialogAction disabled={mutation.isPending} onClick={() => mutation.mutate()}>
            Send Tender
          </AlertDialogAction>
        </AlertDialogFooter>
      </AlertDialogContent>
    </AlertDialog>
  );
}
