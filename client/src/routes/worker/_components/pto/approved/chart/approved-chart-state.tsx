import { ErrorBoundaryUi } from "@/components/elements/error-boundary-ui";
import { LoadingSkeletonState } from "@/components/loading-skeleton";
import { QueryErrorResetBoundary } from "@tanstack/react-query";
import { Suspense } from "react";
import { ErrorBoundary } from "react-error-boundary";

export function ApprovedChartBoundary({ children }: { children: React.ReactNode }) {
  return (
    <QueryErrorResetBoundary>
      {({ reset }) => (
        <ErrorBoundary
          fallbackRender={({ error, resetErrorBoundary }) => (
            <ErrorBoundaryUi error={error as Error} resetError={resetErrorBoundary} />
          )}
          onReset={reset}
        >
          <Suspense fallback={<LoadingSkeletonState description="Loading chart component..." />}>
            {children}
          </Suspense>
        </ErrorBoundary>
      )}
    </QueryErrorResetBoundary>
  );
}
