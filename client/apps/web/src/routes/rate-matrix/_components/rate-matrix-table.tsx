import { DataTable } from "@/components/data-table/data-table";
import { rateMatrixTableGraphQLConfig } from "@/lib/graphql/rate-tables";
import { Resource } from "@trenova/shared/types/permission";
import type { RateMatrix } from "@trenova/shared/types/rate";
import { useMemo } from "react";
import { getColumns } from "./rate-matrix-columns";
import { RateMatrixPanel } from "./rate-matrix-panel";

export default function RateMatrixTable() {
  const columns = useMemo(() => getColumns(), []);

  return (
    <DataTable<RateMatrix>
      name="Rate Matrix"
      queryKey="rate-matrix-list"
      graphql={rateMatrixTableGraphQLConfig}
      resource={Resource.RateMatrix}
      columns={columns}
      TablePanel={RateMatrixPanel}
    />
  );
}
