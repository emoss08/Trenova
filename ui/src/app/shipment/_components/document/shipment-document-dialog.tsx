/*
 * Copyright 2023-2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

import { LazyComponent } from "@/components/error-boundary";
import {
  Dialog,
  DialogBody,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import type { ShipmentSchema } from "@/lib/schemas/shipment-schema";
import { type TableSheetProps } from "@/types/data-table";
import { lazy } from "react";

type ShipmentDocumentDialogProps = {
  shipmentId: ShipmentSchema["id"];
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
