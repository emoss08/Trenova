/*
 * Copyright 2023-2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

import { LazyLoader, QueryLazyComponent } from "@/components/error-boundary";
import { FormSaveProvider } from "@/components/form";
import { MetaTags } from "@/components/meta-tags";
import { Skeleton } from "@/components/ui/skeleton";
import { queries } from "@/lib/queries";
import { lazy } from "react";

const WorkersDataTable = lazy(() => import("./_components/workers-table"));
const PTOContent = lazy(() => import("./_components/pto/pto-content"));

export function Workers() {
  return (
    <>
      <MetaTags title="Workers" description="Workers" />
      <FormSaveProvider>
        <WorkersContent>
          <QueryLazyComponent queryKey={queries.worker.listUpcomingPTO._def}>
            <PTOContent />
          </QueryLazyComponent>
          <LazyLoader fallback={<Skeleton className="h-[300px]" />}>
            <WorkersDataTable />
          </LazyLoader>
        </WorkersContent>
      </FormSaveProvider>
    </>
  );
}

function WorkersContent({ children }: { children: React.ReactNode }) {
  return <div className="flex flex-col gap-4">{children}</div>;
}
