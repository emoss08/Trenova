/*
 * Copyright 2023-2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

import { DataTable } from "@/components/data-table/data-table";
import { type FleetCodeSchema } from "@/lib/schemas/fleet-code-schema";
import { Resource } from "@/types/audit-entry";
import { useMemo } from "react";
import { getColumns } from "./fleet-code-columns";
import { CreateFleetCodeModal } from "./fleet-code-create-modal";
import { EditFleetCodeModal } from "./fleet-code-edit-modal";

export default function FleetCodesDataTable() {
  const columns = useMemo(() => getColumns(), []);

  return (
    <DataTable<FleetCodeSchema>
      name="Fleet Code"
      link="/fleet-codes/"
      queryKey="fleet-code-list"
      extraSearchParams={{
        includeManagerDetails: true,
      }}
      exportModelName="fleet-code"
      TableModal={CreateFleetCodeModal}
      TableEditModal={EditFleetCodeModal}
      columns={columns}
      resource={Resource.FleetCode}
    />
  );
}
