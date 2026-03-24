import { DataTable } from "@/components/data-table/data-table";
import type { HoldReason } from "@/types/hold-reason";
import { Resource } from "@/types/permission";
import { useMemo } from "react";
import { getColumns } from "./hold-reason-columns";
import { HoldReasonPanel } from "./hold-reason-panel";

export default function HoldReasonTable() {
  const columns = useMemo(() => getColumns(), []);

  return (
    <DataTable<HoldReason>
      name="Hold Reason"
      link="/hold-reasons/"
      queryKey="hold-reason-list"
      exportModelName="hold-reason"
      resource={Resource.HoldReason}
      columns={columns}
      TablePanel={HoldReasonPanel}
    />
  );
}
