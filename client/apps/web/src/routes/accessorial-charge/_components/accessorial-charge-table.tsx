import { useT } from "@trenova/shared/i18n/use-t";
import { DataTable } from "@/components/data-table/data-table";
import {
  accessorialChargeTableGraphQLConfig,
  type AccessorialChargeRow,
} from "@/lib/graphql/accessorial-charge-table";
import { Resource } from "@trenova/shared/types/permission";
import { useMemo } from "react";
import { getColumns } from "./accessorial-charge-columns";
import { AccessorialChargePanel } from "@/components/accessorial-charge/accessorial-charge-panel";

export default function AccessorialChargeTable() {
  const t = useT();

  const columns = useMemo(() => getColumns(t), [t]);

  return (
    <DataTable<AccessorialChargeRow>
      name="Accessorial Charge"
      emptyTitle={t("No accessorial charges yet")}
      queryKey="accessorial-charge-list"
      graphql={accessorialChargeTableGraphQLConfig}
      resource={Resource.AccessorialCharge}
      columns={columns}
      enableRowSelection
      TablePanel={AccessorialChargePanel}
    />
  );
}
