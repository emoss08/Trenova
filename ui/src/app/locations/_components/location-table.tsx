import { DataTable } from "@/components/data-table/data-table";
import { LocationSchema } from "@/lib/schemas/location-schema";
import { useMemo } from "react";
import { getColumns } from "./location-columns";
import { CreateLocationModal } from "./location-create-modal";
import { EditLocationModal } from "./location-edit-modal";

export default function LocationsDataTable() {
  const columns = useMemo(() => getColumns(), []);

  return (
    <DataTable<LocationSchema>
      name="Location"
      link="/locations/"
      extraSearchParams={{
        includeCategory: true,
        includeState: true,
      }}
      queryKey={["location"]}
      exportModelName="location"
      TableModal={CreateLocationModal}
      TableEditModal={EditLocationModal}
      columns={columns}
    />
  );
}
