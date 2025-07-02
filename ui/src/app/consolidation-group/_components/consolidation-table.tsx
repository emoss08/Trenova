import { DataTable } from "@/components/data-table/data-table";
import type { ConsolidationGroupSchema } from "@/lib/schemas/consolidation-schema";
import { Resource } from "@/types/audit-entry";
import { useMemo } from "react";
import { getColumns } from "./consolidation-columns";
import { ConsolidationCreateSheet } from "./consolidation-create-sheet";

export default function ConsolidationTable() {
  const columns = useMemo(() => getColumns(), []);

  return (
    <DataTable<ConsolidationGroupSchema>
      name="Consolidation Group"
      link="/consolidations/"
      extraSearchParams={{
        expandDetails: true,
      }}
      queryKey="consolidation-list"
      exportModelName="consolidation"
      resource={Resource.Consolidation}
      TableModal={ConsolidationCreateSheet}
      // TableEditModal={ConsolidationEditSheet}
      columns={columns}
      // getRowClassName={(row) => {
      //   return cn(getShipmentStatusRowClassName(row.original.status));
      // }}
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
      //     key: "view-available-shipments",
      //     label: "View Available Shipments",
      //     description: "View shipments that can be consolidated",
      //     icon: faBoxes,
      //     onClick: () => {
      //       console.log("View available shipments");
      //       // TODO: Open available shipments modal
      //     },
      //   },
      // ]}
    />
  );
}
