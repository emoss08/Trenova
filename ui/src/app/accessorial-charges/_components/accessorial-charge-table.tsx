import { DataTable } from "@/components/data-table/data-table";
import { type AccessorialChargeSchema } from "@/lib/schemas/accessorial-charge-schema";
import { Resource } from "@/types/audit-entry";
import { useMemo } from "react";
import { getColumns } from "./accessorial-charge-columns";
import { CreateAccessorialChargeModal } from "./accessorial-charge-create-modal";
import { EditAccessorialChargeModal } from "./accessorial-charge-edit-modal";

export default function AccessorialChargeTable() {
  const columns = useMemo(() => getColumns(), []);

  return (
    <DataTable<AccessorialChargeSchema>
      resource={Resource.AccessorialCharge}
      name="Accessorial Charge"
      link="/accessorial-charges/"
      exportModelName="accessorial-charge"
      queryKey="accessorial-charge-list"
      TableModal={CreateAccessorialChargeModal}
      TableEditModal={EditAccessorialChargeModal}
      columns={columns}
      defaultSort={[{ field: "createdAt", direction: "desc" }]}
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
    />
  );
}
