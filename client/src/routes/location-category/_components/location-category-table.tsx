import { DataTable } from "@/components/data-table/data-table";
import type { LocationCategory } from "@/types/location-category";
import { Resource } from "@/types/permission";
import { useMemo } from "react";
import { getColumns } from "./location-category-columns";
import { LocationCategoryPanel } from "./location-category-panel";

export default function LocationCategoryTable() {
  const columns = useMemo(() => getColumns(), []);

  return (
    <DataTable<LocationCategory>
      name="Location Category"
      link="/location-categories/"
      queryKey="location-category-list"
      exportModelName="location-category"
      resource={Resource.LocationCategory}
      columns={columns}
      TablePanel={LocationCategoryPanel}
    />
  );
}
