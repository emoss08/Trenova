/**
 * # Copyright 2023-2025 Eric Moss
 * # Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * # Full license: https://github.com/emoss08/trenova/blob/main/LICENSE.md
 */

import { DataTable } from "@/components/data-table/data-table";
import { LocationSchema } from "@/lib/schemas/location-schema";
import { Resource } from "@/types/audit-entry";
import { useMemo } from "react";
import { getColumns } from "./location-columns";
import { CreateLocationModal } from "./location-create-modal";
import { EditLocationModal } from "./location-edit-modal";

export default function LocationsDataTable() {
  const columns = useMemo(() => getColumns(), []);

  return (
    <DataTable<LocationSchema>
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
