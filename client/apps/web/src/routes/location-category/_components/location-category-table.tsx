import { useT } from "@trenova/shared/i18n/use-t";
import { DataTable } from "@/components/data-table/data-table";
import { locationCategoryTableGraphQLConfig } from "@/lib/graphql/location-category-table";
import type { LocationCategory } from "@/types/location-category";
import { Resource } from "@trenova/shared/types/permission";
import { useMemo } from "react";
import { getColumns } from "./location-category-columns";
import { LocationCategoryPanel } from "./location-category-panel";

export default function LocationCategoryTable() {
  const t = useT();

  const columns = useMemo(() => getColumns(t), [t]);

  return (
    <DataTable<LocationCategory>
      name="Location Category"
      queryKey="location-category-list"
      graphql={locationCategoryTableGraphQLConfig}
      resource={Resource.LocationCategory}
      columns={columns}
      TablePanel={LocationCategoryPanel}
    />
  );
}
