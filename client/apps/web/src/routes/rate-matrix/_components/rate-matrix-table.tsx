import { DataTable } from "@/components/data-table/data-table";
import { rateMatrixTableGraphQLConfig, type RateMatrixRow } from "@/lib/graphql/rate-tables";
import { Resource } from "@trenova/shared/types/permission";
import { useMemo } from "react";
import { getColumns } from "./rate-matrix-columns";
import { RateMatrixPanel } from "./rate-matrix-panel";

export default function RateMatrixTable() {
  const columns = useMemo(() => getColumns(), []);

  return (
    <DataTable<RateMatrixRow>
      name="Rate Matrix"
      queryKey="rate-matrix-list"
      graphql={rateMatrixTableGraphQLConfig}
      resource={Resource.RateMatrix}
      columns={columns}
      TablePanel={RateMatrixPanel}
    />
  );
}
