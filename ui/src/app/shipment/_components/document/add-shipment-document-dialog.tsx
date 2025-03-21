import { LazyComponent } from "@/components/error-boundary";
import { BetaTag } from "@/components/ui/beta-tag";
import {
  Dialog,
  DialogBody,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { shipmentDocumentTypes } from "@/lib/choices";
import { Resource } from "@/types/audit-entry";
import { type TableSheetProps } from "@/types/data-table";
import { type Shipment } from "@/types/shipment";
import { lazy } from "react";

const DocumentUpload = lazy(
  () => import("@/components/file-uploader/file-uploader"),
);

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
          <DialogTitle>
            Add Shipment Document <BetaTag />
          </DialogTitle>
          <DialogDescription>Add a document to the shipment.</DialogDescription>
        </DialogHeader>
        <DialogBody>
          <LazyComponent
            componentLoaderProps={{
              message: "Loading document uploader...",
              description: "This may take a few seconds.",
            }}
          >
            <DocumentUpload
              resourceType={Resource.Shipment}
              resourceId={shipmentId}
              documentTypes={shipmentDocumentTypes}
              allowMultiple
              maxFileSizeMB={100}
            />
          </LazyComponent>
        </DialogBody>
      </DialogContent>
    </Dialog>
  );
}
