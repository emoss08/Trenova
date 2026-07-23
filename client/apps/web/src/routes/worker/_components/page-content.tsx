import { DataTableLazyComponent } from "@/components/error-boundary";
import { Tabs, TabsContent, TabsList, TabsTab } from "@/components/ui/tabs";
import { CalendarIcon, UsersIcon } from "lucide-react";
import { parseAsStringLiteral, useQueryState } from "nuqs";
import { lazy } from "react";
import PTODataTable from "./pto/pto-table";

const WorkerTable = lazy(() => import("./worker-table"));

const tabValues = ["workers", "pto"] as const;

export default function WorkersContent() {
  const [tab, setTab] = useQueryState(
    "pageTab",
    parseAsStringLiteral(tabValues)
      .withOptions({
        history: "push",
        shallow: true,
      })
      .withDefault("workers"),
  );

  return (
    <Tabs value={tab} className="gap-1" onValueChange={(value) => setTab(value as "workers" | "pto")}>
      <TabsList variant="underline">
        <TabsTab value="workers">
          <UsersIcon size={16} aria-hidden="true" />
          Workers
        </TabsTab>
        <TabsTab value="pto">
          <CalendarIcon size={16} aria-hidden="true" />
          Paid Time Off
        </TabsTab>
      </TabsList>
      <TabsContent value="workers">
        <DataTableLazyComponent>
          <WorkerTable />
        </DataTableLazyComponent>
      </TabsContent>
      <TabsContent value="pto">
        <DataTableLazyComponent>
          <PTODataTable />
        </DataTableLazyComponent>
      </TabsContent>
    </Tabs>
  );
}
