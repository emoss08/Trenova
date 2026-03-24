import { LazyLoadErrorFallback } from "@/components/error-boundary";
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
            <LazyLoadErrorFallback error={error as Error} resetErrorBoundary={resetErrorBoundary} />
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
