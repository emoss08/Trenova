/*
 * Copyright 2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

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
    <div className="flex h-full flex-col">
      <MetaTags
        title="Consolidation Management"
        description="Manage consolidation groups for optimized shipment routing"
      />
      <ConsolidationManagementHeader />
      <main className="flex min-h-0 flex-1 flex-col px-4 pb-4">
        <Suspense
          fallback={
            <div className="flex h-full items-center justify-center">
              <Loader className="h-8 w-8 animate-spin text-muted-foreground" />
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
    <header className="mb-4 border-b border-border bg-card/50 px-6 py-4 backdrop-blur-sm">
      <div className="flex items-center justify-between">
        <div className="space-y-1">
          <h1 className="flex items-center gap-2 text-2xl font-semibold tracking-tight">
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
