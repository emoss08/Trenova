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
import { queries } from "@/lib/queries";
import { Resource } from "@/types/audit-entry";
import { type TableSheetProps } from "@/types/data-table";
import { type Shipment } from "@/types/shipment";
import { useQueryClient } from "@tanstack/react-query";
import { lazy } from "react";
import { toast } from "sonner";

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
  const queryClient = useQueryClient();

  const onUploadComplete = async () => {
    await queryClient.invalidateQueries({
      queryKey: queries.document.documentsByResourceID._def,
    });

    toast.success("Document uploaded successfully");
  };
  return (
    <Dialog {...props}>
      <DialogContent className="sm:max-w-screen-sm">
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
              onUploadComplete={onUploadComplete}
            />
          </LazyComponent>
        </DialogBody>
      </DialogContent>
    </Dialog>
  );
}
