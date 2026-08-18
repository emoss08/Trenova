import { DataTable } from "@/components/data-table/data-table";
import { rateAgreementTableGraphQLConfig } from "@/lib/graphql/rate-tables";
import { Resource } from "@trenova/shared/types/permission";
import type { RateAgreement } from "@trenova/shared/types/rate";
import { useMemo } from "react";
import { getColumns } from "./rate-agreement-columns";
import { RateAgreementPanel } from "./rate-agreement-panel";

export default function RateAgreementTable() {
  const columns = useMemo(() => getColumns(), []);

  return (
    <DataTable<RateAgreement>
      name="Rate Agreement"
      queryKey="rate-agreement-list"
      graphql={rateAgreementTableGraphQLConfig}
      resource={Resource.RateAgreement}
      columns={columns}
      TablePanel={RateAgreementPanel}
    />
  );
}
