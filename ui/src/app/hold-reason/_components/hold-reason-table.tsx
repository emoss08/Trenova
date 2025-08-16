import { DataTable } from "@/components/data-table/data-table";
import { HoldReasonSchema } from "@/lib/schemas/hold-reason-schema";
import { Resource } from "@/types/audit-entry";
import { useMemo } from "react";
import { getColumns } from "./hold-reason-columns";
import { CreateHoldReasonModal } from "./hold-reason-create-modal";
import { EditHoldReasonModal } from "./hold-reason-edit-modal";

export default function HoldReasonTable() {
  const columns = useMemo(() => getColumns(), []);

  return (
    <DataTable<HoldReasonSchema>
      resource={Resource.HoldReason}
      name="Hold Reason"
      link="/hold-reasons/"
      exportModelName="hold-reason"
      queryKey="hold-reason-list"
      columns={columns}
      TableModal={CreateHoldReasonModal}
      TableEditModal={EditHoldReasonModal}
      config={{
        enableFiltering: true,
        enableSorting: true,
        enableMultiSort: true,
        maxFilters: 5,
        maxSorts: 3,
        searchDebounce: 300,
        showFilterUI: true,
        showSortUI: true,
      }}
      useEnhancedBackend={true}
    />
  );
}
