import { translate } from "@trenova/shared/i18n/runtime";
import type {
  ShipmentBillingAction,
  ShipmentBillingActions,
} from "@/hooks/use-shipment-billing-actions";
import { isEligibleTenderStatus } from "@/lib/shipment-utils";
import type { RowAction, Row } from "@trenova/shared/types/data-table";
import type { Shipment } from "@trenova/shared/types/shipment";
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
  onUncancel: (row: Row<Shipment>) => Promise<unknown>;
  onTransferOwnership: (row: Row<Shipment>) => void;
  billingActions: ShipmentBillingActions;
  onSendEDI: (row: Row<Shipment>) => void;
  canSendEDI: boolean;
};

function toRowAction(action: ShipmentBillingAction): RowAction<Shipment> {
  return {
    id: action.id,
    label: action.label,
    icon: action.icon,
    onClick: (row) => (row.original.id ? action.run(row.original.id) : undefined),
    hidden: (row) => !action.isAvailable(row.original),
  };
}

export function buildShipmentRowActions(
  handlers: ShipmentRowActionHandlers,
): RowAction<Shipment>[] {
  return [
    {
      id: "edit",
      label: translate("Edit"),
      icon: PencilIcon,
      onClick: handlers.onEdit,
    },
    {
      id: "duplicate",
      label: translate("Duplicate"),
      icon: CopyIcon,
      onClick: handlers.onDuplicate,
    },
    {
      id: "transfer-ownership",
      label: translate("Transfer Ownership"),
      icon: ArrowRightLeftIcon,
      onClick: handlers.onTransferOwnership,
      hidden: (row) => row.original.status === "Canceled",
    },
    {
      id: "send-edi-load-tender",
      label: translate("Send EDI Load Tender"),
      icon: SendIcon,
      onClick: handlers.onSendEDI,
      hidden: (row) => {
        const shipment = row.original;
        const tenderStatus = shipment.tenderStatus;
        const isEligible = isEligibleTenderStatus(tenderStatus);

        return (
          !handlers.canSendEDI ||
          shipment.status !== "New" ||
          !isEligible ||
          !shipment.customer?.ediPartner
        );
      },
    },
    toRowAction(handlers.billingActions.markReadyToBill),
    toRowAction(handlers.billingActions.markReadyAndTransferToBilling),
    toRowAction(handlers.billingActions.transferToBilling),
    {
      id: "cancel",
      label: translate("Cancel"),
      icon: BanIcon,
      variant: "destructive",
      onClick: handlers.onCancel,
      hidden: (row) => row.original.status === "Canceled",
    },
    {
      id: "uncancel",
      label: translate("Uncancel"),
      icon: UndoIcon,
      onClick: handlers.onUncancel,
      hidden: (row) => row.original.status !== "Canceled",
    },
  ];
}
