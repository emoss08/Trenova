import { DataTable } from "@/components/data-table/data-table";
import { rateTableTableGraphQLConfig } from "@/lib/graphql/rate-table-table";
import { Resource } from "@/types/permission";
import type { RateTableRow } from "@/types/rate-table";
import { useMemo } from "react";
import { getColumns } from "./rate-table-columns";
import { RateTablePanel } from "./rate-table-panel";

export default function RateTableTable() {
  const columns = useMemo(() => getColumns(), []);

  return (
    <DataTable<RateTableRow>
      name="Rate Table"
      queryKey="rate-table-list"
      graphql={rateTableTableGraphQLConfig}
      resource={Resource.RateTable}
      columns={columns}
      TablePanel={RateTablePanel}
    />
  );
}
