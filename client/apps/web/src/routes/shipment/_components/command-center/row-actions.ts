import { isEligibleTenderStatus } from "@/lib/shipment-utils";
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
  onSendEDI: (row: Row<Shipment>) => void;
  canSendEDI: boolean;
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
      id: "transfer-ownership",
      label: "Transfer Ownership",
      icon: ArrowRightLeftIcon,
      onClick: handlers.onTransferOwnership,
      hidden: (row) => row.original.status === "Canceled",
    },
    {
      id: "send-edi-load-tender",
      label: "Send EDI Load Tender",
      icon: SendIcon,
      onClick: handlers.onSendEDI,
      hidden: (row) => {
        const shipment = row.original;
        const tenderStatus = shipment.tenderStatus;
        const isEligible = isEligibleTenderStatus(tenderStatus);

        return !handlers.canSendEDI || shipment.status !== "New" || !isEligible;
      },
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
  ];
}
