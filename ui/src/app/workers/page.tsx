/*
 * Copyright 2023-2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

import { LazyComponent, QueryLazyComponent } from "@/components/error-boundary";
import { FormSaveProvider } from "@/components/form";
import { MetaTags } from "@/components/meta-tags";
import { queries } from "@/lib/queries";
import { lazy } from "react";

const WorkersDataTable = lazy(() => import("./_components/workers-table"));
const UpcomingPTOContent = lazy(
  () => import("./_components/pto/upcoming-pto-content"),
);

export function Workers() {
  return (
    <>
      <MetaTags title="Workers" description="Workers" />
      <FormSaveProvider>
        <QueryLazyComponent queryKey={queries.worker.listUpcomingPTO._def}>
          <UpcomingPTOContent />
        </QueryLazyComponent>
        <LazyComponent>
          <WorkersDataTable />
        </LazyComponent>
      </FormSaveProvider>
    </>
  );
}
