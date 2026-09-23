import { useT } from "@trenova/shared/i18n/use-t";
import { ErrorStateBoundary } from "@trenova/shared/components/error-boundary";
import { LoadingSkeletonState } from "@trenova/shared/components/loading-skeleton";
import { QueryErrorResetBoundary } from "@tanstack/react-query";
import { Suspense } from "react";

export function ApprovedChartBoundary({ children }: { children: React.ReactNode }) {
  const t = useT();

  return (
    <QueryErrorResetBoundary>
      <ErrorStateBoundary layout="compact">
        <Suspense fallback={<LoadingSkeletonState description={t("Loading chart component...")} />}>
          {children}
        </Suspense>
      </ErrorStateBoundary>
    </QueryErrorResetBoundary>
  );
}
