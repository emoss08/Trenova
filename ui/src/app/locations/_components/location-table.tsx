import { DataTableV2 } from "@/components/data-table/data-table-v2";
import { LocationSchema } from "@/lib/schemas/location-schema";
import { Resource } from "@/types/audit-entry";
import { useMemo } from "react";
import { getColumns } from "./location-columns";
import { CreateLocationModal } from "./location-create-modal";
import { EditLocationModal } from "./location-edit-modal";

export default function LocationsDataTable() {
  const columns = useMemo(() => getColumns(), []);

  return (
    <DataTableV2<LocationSchema>
      resource={Resource.Location}
      name="Location"
      link="/locations/"
      extraSearchParams={{
        includeCategory: true,
        includeState: true,
      }}
      queryKey="location-list"
      exportModelName="location"
      TableModal={CreateLocationModal}
      TableEditModal={EditLocationModal}
      columns={columns}
    />
  );
}
