import { DataTable } from "@/components/data-table/data-table";
import { rateZoneTableGraphQLConfig } from "@/lib/graphql/rate-tables";
import { Resource } from "@trenova/shared/types/permission";
import type { RateZone } from "@trenova/shared/types/rate";
import { useMemo } from "react";
import { getColumns } from "./rate-zone-columns";
import { RateZonePanel } from "./rate-zone-panel";

export default function RateZoneTable() {
  const columns = useMemo(() => getColumns(), []);

  return (
    <DataTable<RateZone>
      name="Rate Zone"
      queryKey="rate-zone-list"
      graphql={rateZoneTableGraphQLConfig}
      resource={Resource.RateZone}
      columns={columns}
      TablePanel={RateZonePanel}
    />
  );
}
