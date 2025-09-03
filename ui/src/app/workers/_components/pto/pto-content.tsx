/*
 * Copyright 2023-2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

import { LazyLoader } from "@/components/error-boundary";
import { Skeleton } from "@/components/ui/skeleton";
import { lazy } from "react";

const ApprovePTOOverview = lazy(() => import("./approved-pto-overview"));
const RequestedPTOOverview = lazy(() => import("./requested-pto-overview"));

export default function PTOContent() {
  return (
    <PTOContentInner>
      <LazyLoader fallback={<Skeleton className="h-[300px]" />}>
        <ApprovePTOOverview />
      </LazyLoader>
      <LazyLoader fallback={<Skeleton className="h-[300px]" />}>
        <RequestedPTOOverview />
      </LazyLoader>
    </PTOContentInner>
  );
}

function PTOContentInner({ children }: { children: React.ReactNode }) {
  return <div className="grid grid-cols-12 gap-4">{children}</div>;
}
