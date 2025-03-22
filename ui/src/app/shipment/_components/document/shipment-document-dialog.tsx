import { LazyComponent } from "@/components/error-boundary";
import {
  Dialog,
  DialogBody,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { type TableSheetProps } from "@/types/data-table";
import { type Shipment } from "@/types/shipment";
import { lazy } from "react";

type ShipmentDocumentDialogProps = {
  shipmentId: Shipment["id"];
} & TableSheetProps;

const ShipmentDocumentTable = lazy(() => import("./shipment-document-table"));

export function ShipmentDocumentDialog({
  shipmentId,
  ...props
}: ShipmentDocumentDialogProps) {
  return (
    <Dialog {...props}>
      <DialogContent className="max-w-[1300px]">
        <DialogHeader>
          <DialogTitle>Shipment Documents</DialogTitle>
          <DialogDescription>
            View and manage documents associated with this shipment.
          </DialogDescription>
        </DialogHeader>
        <DialogBody className="p-0">
          <div className="p-2">
            <LazyComponent>
              <ShipmentDocumentTable shipmentId={shipmentId} />
            </LazyComponent>
          </div>
        </DialogBody>
      </DialogContent>
    </Dialog>
  );
}
