import { DataTableLazyComponent } from "@/components/error-boundary";
import { MetaTags } from "@/components/meta-tags";
import { lazy, memo } from "react";

const AILogsTable = lazy(() => import("./_components/ai-logs-table"));

export function AILogs() {
  return (
    <div className="flex flex-col space-y-6">
      <MetaTags title="AI Logs" description="AI Logs" />
      <Header />
      <DataTableLazyComponent>
        <AILogsTable />
      </DataTableLazyComponent>
    </div>
  );
}

const Header = memo(() => {
  return (
    <div className="flex justify-between items-center">
      <div>
        <h1 className="text-3xl font-bold tracking-tight">AI Logs</h1>
        <p className="text-muted-foreground">
          Monitor, track, and analyze AI activities and user actions
        </p>
      </div>
    </div>
  );
});
Header.displayName = "Header";
