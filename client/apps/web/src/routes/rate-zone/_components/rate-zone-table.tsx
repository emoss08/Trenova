import { useT } from "@trenova/shared/i18n/use-t";
import { DataTable } from "@/components/data-table/data-table";
import { rateZoneTableGraphQLConfig, type RateZoneRow } from "@/lib/graphql/rate-tables";
import { Resource } from "@trenova/shared/types/permission";
import { useMemo } from "react";
import { getColumns } from "./rate-zone-columns";
import { RateZonePanel } from "./rate-zone-panel";

export default function RateZoneTable() {
  const t = useT();

  const columns = useMemo(() => getColumns(t), [t]);

  return (
    <DataTable<RateZoneRow>
      name="Rate Zone"
      queryKey="rate-zone-list"
      graphql={rateZoneTableGraphQLConfig}
      resource={Resource.RateZone}
      columns={columns}
      TablePanel={RateZonePanel}
    />
  );
}
