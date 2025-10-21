import { LazyLoadErrorFallback } from "@/components/error-boundary";
import { Skeleton } from "@/components/ui/skeleton";
import { TextShimmer } from "@/components/ui/text-shimmer";
import { cn } from "@/lib/utils";
import { QueryErrorResetBoundary } from "@tanstack/react-query";
import { Suspense } from "react";
import { ErrorBoundary } from "react-error-boundary";

export function ApprovedChartBoundary({
  children,
}: {
  children: React.ReactNode;
}) {
  return (
    <QueryErrorResetBoundary>
      {({ reset }) => (
        <ErrorBoundary
          fallbackRender={({ error, resetErrorBoundary }) => (
            <LazyLoadErrorFallback
              error={error}
              resetErrorBoundary={resetErrorBoundary}
            />
          )}
          onReset={reset}
        >
          <Suspense
            fallback={
              <ApprovedChartLoadingState description="Loading chart component..." />
            }
          >
            {children}
          </Suspense>
        </ErrorBoundary>
      )}
    </QueryErrorResetBoundary>
  );
}

export function ApprovedChartLoadingState({
  description,
  className,
}: {
  description: string;
  className?: string;
}) {
  return (
    <div
      className={cn(
        "flex flex-col items-center justify-center w-full h-[334px] border border-border rounded-md",
        className,
      )}
    >
      <div className="relative size-full">
        <Skeleton className="size-full" />
        <span className="absolute inset-0 flex items-center justify-center text-sm text-muted-foreground">
          <TextShimmer duration={1.5}>{description}</TextShimmer>
        </span>
      </div>
    </div>
  );
}
