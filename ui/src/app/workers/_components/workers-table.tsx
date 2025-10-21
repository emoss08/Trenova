import { DataTable } from "@/components/data-table/data-table";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { type WorkerSchema } from "@/lib/schemas/worker-schema";
import { Resource } from "@/types/audit-entry";
import { CalendarIcon, UsersIcon } from "lucide-react";
import { useQueryState } from "nuqs";
import { useMemo } from "react";
import { PtoDataTable } from "./pto/pto-table";
import { CreateWorkerModal } from "./workers-create-modal";
import { EditWorkerModal } from "./workers-edit-modal";
import { getColumns } from "./workers-table-columns";

export default function WorkersContent() {
  const [tab, setTab] = useQueryState("tab", {
    defaultValue: "workers",
    shallow: true, // So it doesn't trigger a full page reload
  });

  return (
    <Tabs value={tab} onValueChange={setTab}>
      <TabsList className="before:bg-border relative h-auto w-full gap-0.5 bg-transparent p-0 before:absolute before:inset-x-0 before:bottom-0 before:h-px justify-start">
        <TabsTrigger
          value="workers"
          className="bg-muted overflow-hidden rounded-b-none border-x border-t py-2 data-[state=active]:z-10 data-[state=active]:shadow-none"
        >
          <UsersIcon
            className="-ms-0.5 mb-0.5 opacity-60"
            size={16}
            aria-hidden="true"
          />
          Workers
        </TabsTrigger>
        <TabsTrigger
          value="pto"
          className="bg-muted overflow-hidden rounded-b-none border-x border-t py-2 data-[state=active]:z-10 data-[state=active]:shadow-none"
        >
          <CalendarIcon
            className="-ms-0.5 mb-0.5 opacity-60"
            size={16}
            aria-hidden="true"
          />
          Paid Time Off
        </TabsTrigger>
      </TabsList>
      <TabsContent value="workers">
        <WorkersDataTable />
      </TabsContent>
      <TabsContent value="pto">
        <PtoDataTable />
      </TabsContent>
    </Tabs>
  );
}

function WorkersDataTable() {
  const columns = useMemo(() => getColumns(), []);

  return (
    <DataTable<WorkerSchema>
      extraSearchParams={{
        includeProfile: "true",
        includePTO: "true",
      }}
      TableModal={CreateWorkerModal}
      TableEditModal={EditWorkerModal}
      queryKey="worker-list"
      name="Worker"
      link="/workers/"
      exportModelName="Worker"
      columns={columns}
      resource={Resource.Worker}
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
