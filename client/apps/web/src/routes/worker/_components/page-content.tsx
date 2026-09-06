import { DataTableLazyComponent } from "@trenova/shared/components/error-boundary";
import { Tabs, TabsContent, TabsList, TabsTab } from "@trenova/shared/components/ui/tabs";
import { CalendarIcon, UsersIcon } from "lucide-react";
import { parseAsStringLiteral, useQueryState } from "nuqs";
import { lazy } from "react";
import { PTOBalanceSummaryCard } from "./pto/pto-balance-summary-card";
import PTODataTable from "./pto/pto-table";
import { RosterViews } from "./roster-views";

const WorkerTable = lazy(() => import("./worker-table"));

const tabValues = ["workers", "pto"] as const;

export const WORKERS_PAGE_TAB_PARAM = "pageTab";

export const workersPageTabParser = parseAsStringLiteral(tabValues)
  .withOptions({
    history: "push",
    shallow: true,
  })
  .withDefault("workers");

export default function WorkersContent() {
  const [tab, setTab] = useQueryState(WORKERS_PAGE_TAB_PARAM, workersPageTabParser);

  return (
    <Tabs
      value={tab}
      className="gap-1"
      onValueChange={(value) => setTab(value as "workers" | "pto")}
    >
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
      <TabsContent value="workers" className="flex flex-col gap-2">
        <RosterViews />
        <DataTableLazyComponent>
          <WorkerTable />
        </DataTableLazyComponent>
      </TabsContent>
      <TabsContent value="pto">
        <PTOBalanceSummaryCard />
        <DataTableLazyComponent>
          <PTODataTable />
        </DataTableLazyComponent>
      </TabsContent>
    </Tabs>
  );
}
