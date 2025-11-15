import { DataTable } from "@/components/data-table/data-table";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { VariableSchema } from "@/lib/schemas/variable-schema";
import { Resource } from "@/types/audit-entry";
import { PiIcon, VariableIcon } from "lucide-react";
import { useQueryState } from "nuqs";
import { useMemo } from "react";
import { FormatDataTable } from "./format/format-table";
import { getColumns } from "./variable-columns";
import { CreateVariableModal } from "./variable-create-modal";
import { EditVariableModal } from "./variable-edit-modal";

export default function VariableContent() {
  const [tab, setTab] = useQueryState("tab", {
    defaultValue: "variables",
    shallow: true,
  });

  return (
    <Tabs value={tab} onValueChange={setTab}>
      <TabsList className="relative h-auto w-full justify-start gap-0.5 bg-transparent p-0 before:absolute before:inset-x-0 before:bottom-0 before:h-px before:bg-border">
        <TabsTrigger
          value="variables"
          className="overflow-hidden rounded-b-none border-x border-t bg-muted py-2 data-[state=active]:z-10 data-[state=active]:shadow-none"
        >
          <VariableIcon
            className="-ms-0.5 mb-0.5 opacity-60"
            size={16}
            aria-hidden="true"
          />
          Variables
        </TabsTrigger>
        <TabsTrigger
          value="formats"
          className="overflow-hidden rounded-b-none border-x border-t bg-muted py-2 data-[state=active]:z-10 data-[state=active]:shadow-none"
        >
          <PiIcon
            className="-ms-0.5 mb-0.5 opacity-60"
            size={16}
            aria-hidden="true"
          />
          Formats
        </TabsTrigger>
      </TabsList>
      <TabsContent value="variables">
        <VariablesDataTable />
      </TabsContent>
      <TabsContent value="formats">
        <FormatDataTable />
      </TabsContent>
    </Tabs>
  );
}

function VariablesDataTable() {
  const columns = useMemo(() => getColumns(), []);

  return (
    <DataTable<VariableSchema>
      extraSearchParams={{
        includeFormat: "true",
      }}
      queryKey="variable-list"
      name="Variable"
      link="/variables/"
      exportModelName="Variable"
      columns={columns}
      resource={Resource.Variable}
      TableModal={CreateVariableModal}
      TableEditModal={EditVariableModal}
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
      defaultSort={[{ field: "createdAt", direction: "desc" }]}
    />
  );
}
