import {
  Dialog,
  DialogBody,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { DocumentUpload } from "@/components/ui/file-uploader";
import { shipmentDocumentTypes } from "@/lib/choices";
import { Resource } from "@/types/audit-entry";
import { type TableSheetProps } from "@/types/data-table";
import { type Shipment } from "@/types/shipment";

type AddShipmentDocumentDialogProps = {
  shipmentId: Shipment["id"];
} & TableSheetProps;

export function AddShipmentDocumentDialog({
  shipmentId,
  ...props
}: AddShipmentDocumentDialogProps) {
  return (
    <Dialog {...props}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>Add Shipment Document</DialogTitle>
          <DialogDescription>Add a document to the shipment.</DialogDescription>
        </DialogHeader>
        <DialogBody>
          <DocumentUpload
            resourceType={Resource.Shipment}
            resourceId={shipmentId}
            documentTypes={shipmentDocumentTypes}
            allowMultiple
          />
        </DialogBody>
      </DialogContent>
    </Dialog>
  );
}
