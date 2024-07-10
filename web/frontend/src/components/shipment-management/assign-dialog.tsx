import type { Tractor } from "@/types/equipment";
import type { Shipment } from "@/types/shipment";
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from "../ui/alert-dialog";

export function ShipmentConfirmDialog({
  open,
  onOpenChange,
  handleAssignTractor,
  selectedTractor,
  selectedShipment,
}: {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  handleAssignTractor?: () => void;
  selectedTractor: Tractor;
  selectedShipment: Shipment;
}) {
  return (
    <AlertDialog open={open} onOpenChange={onOpenChange}>
      <AlertDialogContent>
        <AlertDialogHeader>
          <AlertDialogTitle>Assign Tractor to Shipment</AlertDialogTitle>
          <AlertDialogDescription>
            Are you sure you want to assign {selectedTractor?.code} to shipment{" "}
            {selectedShipment?.proNumber}?
          </AlertDialogDescription>
        </AlertDialogHeader>
        <AlertDialogFooter>
          <AlertDialogCancel onClick={() => onOpenChange(false)}>
            Cancel
          </AlertDialogCancel>
          <AlertDialogAction onClick={handleAssignTractor}>
            Confirm
          </AlertDialogAction>
        </AlertDialogFooter>
      </AlertDialogContent>
    </AlertDialog>
  );
}
