import type { RowAction } from "@/types/data-table";
import type { Shipment } from "@/types/shipment";
import type { Row } from "@tanstack/react-table";
import {
  ArrowRightLeftIcon,
  BanIcon,
  CopyIcon,
  PencilIcon,
  SendIcon,
  UndoIcon,
} from "lucide-react";

export type ShipmentRowActionHandlers = {
  onEdit: (row: Row<Shipment>) => void;
  onDuplicate: (row: Row<Shipment>) => void;
  onCancel: (row: Row<Shipment>) => void;
  onUncancel: (row: Row<Shipment>) => void;
  onTransferOwnership: (row: Row<Shipment>) => void;
  onTransferToBilling: (row: Row<Shipment>) => void;
};

export function buildShipmentRowActions(
  handlers: ShipmentRowActionHandlers,
): RowAction<Shipment>[] {
  return [
    {
      id: "edit",
      label: "Edit",
      icon: PencilIcon,
      onClick: handlers.onEdit,
    },
    {
      id: "duplicate",
      label: "Duplicate",
      icon: CopyIcon,
      onClick: handlers.onDuplicate,
    },
    {
      id: "cancel",
      label: "Cancel",
      icon: BanIcon,
      variant: "destructive",
      onClick: handlers.onCancel,
      hidden: (row) => row.original.status === "Canceled",
    },
    {
      id: "uncancel",
      label: "Uncancel",
      icon: UndoIcon,
      onClick: handlers.onUncancel,
      hidden: (row) => row.original.status !== "Canceled",
    },
    {
      id: "transfer-ownership",
      label: "Transfer Ownership",
      icon: ArrowRightLeftIcon,
      onClick: handlers.onTransferOwnership,
      hidden: (row) => row.original.status === "Canceled",
    },
    {
      id: "transfer-to-billing",
      label: "Transfer to Billing",
      icon: SendIcon,
      onClick: handlers.onTransferToBilling,
      hidden: (row) => {
        const s = row.original;
        if (s.status !== "ReadyToInvoice") return true;
        const bts = s.billingTransferStatus;
        return !!bts && bts !== "SentBackToOps";
      },
    },
  ];
}
