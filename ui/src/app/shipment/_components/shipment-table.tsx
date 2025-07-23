/**
 * # Copyright 2023-2025 Eric Moss
 * # Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * # Full license: https://github.com/emoss08/trenova/blob/main/LICENSE.md
 */

import { DataTable } from "@/components/data-table/data-table";
import { searchParamsParser } from "@/hooks/use-data-table-state";
import { shipmentActionsParser } from "@/hooks/use-shipment-actions-state";
import { LiveModePresets } from "@/lib/live-mode-utils";
import {
  ShipmentStatus,
  type ShipmentSchema,
} from "@/lib/schemas/shipment-schema";
import { getShipmentStatusRowClassName } from "@/lib/table-styles";
import { cn } from "@/lib/utils";
import { Resource } from "@/types/audit-entry";
import type { ContextMenuAction } from "@/types/data-table";
import { useQueryStates } from "nuqs";
import { useMemo } from "react";
import { toast } from "sonner";
import { getColumns } from "./shipment-columns";
import { ShipmentCreateSheet } from "./shipment-create-sheet";
import { ShipmentEditSheet } from "./shipment-edit-sheet";

export default function ShipmentTable() {
  const columns = useMemo(() => getColumns(), []);
  const [, setSearchParams] = useQueryStates(searchParamsParser);
  const [, setShipmentMenuParams] = useQueryStates(shipmentActionsParser);

  const contextMenuActions: ContextMenuAction<ShipmentSchema>[] = useMemo(
    () => [
      {
        id: "edit",
        label: "Edit Shipment",
        shortcut: "⌘E",
        onClick: (row) => {
          setSearchParams({
            modalType: "edit",
            entityId: row.original.id,
          });
        },
      },
      {
        id: "duplicate",
        label: "Duplicate",
        onClick: (row) => {
          toast.info(`Duplicate shipment: ${row.original.id}`);
        },
        separator: "after",
      },
      {
        id: "export",
        label: "Export",
        subActions: [
          {
            id: "export-pdf",
            label: "Export as PDF",
            onClick: (row) => {
              toast.info(`Export PDF for shipment: ${row.original.id}`);
            },
          },
          {
            id: "export-excel",
            label: "Export as Excel",
            onClick: (row) => {
              toast.info(`Export Excel for shipment: ${row.original.id}`);
            },
          },
        ],
      },
      {
        id: "print",
        label: "Print",
        shortcut: "⌘P",
        onClick: (row) => {
          toast.info(`Print shipment: ${row.original.id}`);
        },
        separator: "after",
      },
      {
        id: "cancel",
        label: (row) => {
          return row.original.status === ShipmentStatus.enum.Canceled
            ? "Un-Cancel"
            : "Cancel";
        },
        variant: "destructive",
        onClick: (row) => {
          setSearchParams({
            modalType: "edit",
            entityId: row.original.id,
          }).then(() => {
            // Open the cancellation dialog based on the current status;
            setShipmentMenuParams({
              cancellationDialogOpen:
                row.original.status !== ShipmentStatus.enum.Canceled,
              unCancelDialogOpen:
                row.original.status === ShipmentStatus.enum.Canceled,
            });
          });
        },
        hidden: (row) => row.original.status === "Completed",
      },
    ],
    [setSearchParams, setShipmentMenuParams],
  );

  return (
    <DataTable<ShipmentSchema>
      name="Shipment"
      link="/shipments/"
      extraSearchParams={{
        expandShipmentDetails: true,
      }}
      queryKey="shipment-list"
      exportModelName="shipment"
      resource={Resource.Shipment}
      TableModal={ShipmentCreateSheet}
      TableEditModal={ShipmentEditSheet}
      columns={columns}
      getRowClassName={(row) => {
        return cn(getShipmentStatusRowClassName(row.original.status));
      }}
      liveMode={LiveModePresets.shipments()}
      config={{
        enableFiltering: true,
        enableSorting: true,
        enableMultiSort: true,
        maxFilters: 5,
        maxSorts: 3,
        searchDebounce: 300,
        showFilterUI: true,
        showSortUI: true,
      }}
      useEnhancedBackend={true}
      contextMenuActions={contextMenuActions}
      // extraActions={[
      //   {
      //     key: "import-from-rate",
      //     label: "Import from Rate Conf.",
      //     description: "Import shipment from rate confirmation",
      //     icon: faFileImport,
      //     onClick: () => {
      //       console.log("Import from Rate Conf.");
      //     },
      //     endContent: <BetaTag label="Preview" />,
      //   },
      // ]}
    />
  );
}
