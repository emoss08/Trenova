import { DataTable } from "@/components/data-table/data-table";
import { LiveModePresets } from "@/lib/live-mode-utils";
import type { ShipmentSchema } from "@/lib/schemas/shipment-schema";
import { getShipmentStatusRowClassName } from "@/lib/table-styles";
import { cn } from "@/lib/utils";
import { Resource } from "@/types/audit-entry";
import { useMemo } from "react";
import { getColumns } from "./shipment-columns";
import { ShipmentCreateSheet } from "./shipment-create-sheet";
import { ShipmentEditSheet } from "./shipment-edit-sheet";

export default function ShipmentTable() {
  const columns = useMemo(() => getColumns(), []);

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
