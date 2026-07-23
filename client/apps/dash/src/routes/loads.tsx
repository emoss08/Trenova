import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { Tabs, TabsList, TabsPanel, TabsTab } from "@trenova/shared/components/ui/tabs";
import type { PortalLoadScope } from "@trenova/graphql/generated/graphql";
import { TruckIcon } from "lucide-react";
import { useState } from "react";
import { LoadCard } from "../_components/load-card";
import { useMyLoads } from "../_components/use-loads";

function LoadList({ scope }: { scope: PortalLoadScope }) {
  const loads = useMyLoads(scope);

  if (loads.isPending) {
    return (
      <div className="flex flex-col gap-3">
        <Skeleton className="h-36 w-full rounded-2xl" />
        <Skeleton className="h-36 w-full rounded-2xl" />
      </div>
    );
  }

  if (!loads.data || loads.data.length === 0) {
    return (
      <div className="flex flex-col items-center gap-2 rounded-2xl border border-dashed border-border p-10 text-center">
        <TruckIcon className="size-6 text-muted-foreground" />
        <p className="text-sm text-muted-foreground">
          {scope === "Active" ? "No active or upcoming loads." : "No completed loads yet."}
        </p>
      </div>
    );
  }

  return (
    <div className="flex flex-col gap-3">
      {loads.data.map((load, index) => (
        <LoadCard key={load.assignmentId} load={load} index={index} />
      ))}
    </div>
  );
}

export function DashLoadsPage() {
  const [tab, setTab] = useState("active");

  return (
    <div className="flex flex-col gap-4">
      <h1 className="text-xl font-semibold tracking-tight">Loads</h1>
      <Tabs value={tab} onValueChange={(value) => setTab(value as string)}>
        <TabsList className="grid w-full grid-cols-2">
          <TabsTab value="active">Active</TabsTab>
          <TabsTab value="history">History</TabsTab>
        </TabsList>
        <TabsPanel value="active" className="mt-3">
          <LoadList scope="Active" />
        </TabsPanel>
        <TabsPanel value="history" className="mt-3">
          <LoadList scope="History" />
        </TabsPanel>
      </Tabs>
    </div>
  );
}
