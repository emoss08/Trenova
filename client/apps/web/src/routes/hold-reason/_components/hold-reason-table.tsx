import { useT } from "@trenova/shared/i18n/use-t";
import { DataTable } from "@/components/data-table/data-table";
import { holdReasonTableGraphQLConfig } from "@/lib/graphql/hold-reason-table";
import type { HoldReason } from "@/types/hold-reason";
import { Resource } from "@trenova/shared/types/permission";
import { useMemo } from "react";
import { getColumns } from "./hold-reason-columns";
import { HoldReasonPanel } from "./hold-reason-panel";

export default function HoldReasonTable() {
  const t = useT();

  const columns = useMemo(() => getColumns(t), [t]);

  return (
    <DataTable<HoldReason>
      name="Hold Reason"
      queryKey="hold-reason-list"
      graphql={holdReasonTableGraphQLConfig}
      resource={Resource.HoldReason}
      columns={columns}
      TablePanel={HoldReasonPanel}
    />
  );
}
