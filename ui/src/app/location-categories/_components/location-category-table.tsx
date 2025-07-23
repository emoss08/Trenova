/**
 * # Copyright 2023-2025 Eric Moss
 * # Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * # Full license: https://github.com/emoss08/trenova/blob/main/LICENSE.md
 */

import { DataTable } from "@/components/data-table/data-table";
import { LocationCategorySchema } from "@/lib/schemas/location-category-schema";
import { Resource } from "@/types/audit-entry";
import { useMemo } from "react";
import { getColumns } from "./location-category-columns";
import { CreateLocationCategoryModal } from "./location-category-create-modal";
import { EditLocationCategoryModal } from "./location-category-edit-modal";

export default function LocationCategoryTable() {
  const columns = useMemo(() => getColumns(), []);

  return (
    <DataTable<LocationCategorySchema>
      resource={Resource.LocationCategory}
      name="Location Category"
      link="/location-categories/"
      queryKey="location-category-list"
      exportModelName="location-category"
      TableModal={CreateLocationCategoryModal}
      TableEditModal={EditLocationCategoryModal}
      columns={columns}
    />
  );
}
