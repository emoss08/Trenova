import { MetaTags } from "@/components/meta-tags";
import { Loader, UserSearch } from "lucide-react";
import { lazy, memo, Suspense } from "react";

const ConsolidationTable = lazy(() =>
  import("./_components/consolidation-table").then((mod) => ({
    default: mod.default,
  })),
);

export function ConsolidationGroup() {
  return (
    <div className="flex flex-col h-full">
      <MetaTags
        title="Consolidation Management"
        description="Manage consolidation groups for optimized shipment routing"
      />
      <ConsolidationManagementHeader />
      <main className="flex-1 flex flex-col min-h-0 px-4 pb-4">
        <Suspense
          fallback={
            <div className="flex items-center justify-center h-full">
              <Loader className="animate-spin h-8 w-8 text-muted-foreground" />
            </div>
          }
        >
          <ConsolidationTable />
        </Suspense>
      </main>
    </div>
  );
}

const ConsolidationManagementHeader = memo(() => {
  return (
    <header className="bg-card/50 backdrop-blur-sm border-b border-border px-6 py-4 mb-4">
      <div className="flex justify-between items-center">
        <div className="space-y-1">
          <h1 className="text-2xl font-semibold tracking-tight flex items-center gap-2">
            <UserSearch className="h-6 w-6 text-muted-foreground" />
            Consolidation Management
          </h1>
          <p className="text-sm text-muted-foreground">
            Optimize routes by grouping shipments together for efficient
            delivery
          </p>
        </div>
      </div>
    </header>
  );
});

ConsolidationManagementHeader.displayName = "ConsolidationManagementHeader";
