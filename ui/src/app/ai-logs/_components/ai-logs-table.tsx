import { DataTable } from "@/components/data-table/data-table";
import { AILogSchema } from "@/lib/schemas/ai-log-schema";
import { Resource } from "@/types/audit-entry";
import { useMemo } from "react";
import { getColumns } from "./ai-logs-columns";

export default function AILogsTable() {
  const columns = useMemo(() => getColumns(), []);

  return (
    <DataTable<AILogSchema>
      resource={Resource.AILog}
      name="AI Logs"
      link="/ai-logs/"
      extraSearchParams={{
        includeUser: true,
      }}
      queryKey="location-category-list"
      exportModelName="location-category"
      //   TableModal={CreateLocationCategoryModal}
      //   TableEditModal={EditLocationCategoryModal}
      columns={columns}
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
