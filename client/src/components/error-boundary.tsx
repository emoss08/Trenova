import { QueryErrorResetBoundary, type QueryKey } from "@tanstack/react-query";
import type React from "react";
import type { ReactNode } from "react";
import { Suspense } from "react";
import {
  ErrorBoundary,
  ErrorBoundary as ReactErrorBoundary,
  type FallbackProps,
} from "react-error-boundary";
import { isRouteErrorResponse, useRouteError } from "react-router";
import { SuspenseLoader, type ComponentLoaderProps } from "./component-loader";
import { DataTableSkeleton } from "./data-table/data-table-skeleton";
import { ErrorBoundaryUi } from "./elements/error-boundary-ui";
import { NotFoundPage } from "./not-found-page";
import { Skeleton } from "./ui/skeleton";

function normalizeFallbackError(error: unknown): Error {
  return error instanceof Error ? error : new Error(String(error));
}

function ErrorBoundaryFallbackAdapter(props: FallbackProps) {
  const error = normalizeFallbackError(props.error);
  return <ErrorBoundaryUi error={error} resetError={props.resetErrorBoundary} />;
}

export function DataTableLazyComponent({
  children,
  onError,
}: {
  children: React.ReactNode;
  onError?: (error: Error, info: React.ErrorInfo) => void;
}) {
  // Adapter for onError signature
  const handleBoundaryError = onError
    ? (error: unknown, info: React.ErrorInfo) => {
        if (error instanceof Error) {
          onError(error, info);
        }
      }
    : undefined;

  return (
    <ErrorBoundary FallbackComponent={ErrorBoundaryFallbackAdapter} onError={handleBoundaryError}>
      <Suspense fallback={<DataTableSkeleton columnCount={10} rowCount={10} />}>
        {children}
      </Suspense>
    </ErrorBoundary>
  );
}

type ErrorBoundaryProps = {
  children: ReactNode;
};

export function RootErrorBoundary({ children }: ErrorBoundaryProps) {
  return (
    <ReactErrorBoundary FallbackComponent={ErrorBoundaryFallbackAdapter}>
      {children}
    </ReactErrorBoundary>
  );
}

export function RouteErrorBoundary() {
  const error = useRouteError();
  const isDev = import.meta.env.DEV;
  const isNotFoundRoute = isRouteErrorResponse(error) && error.status === 404;

  if (isNotFoundRoute) {
    const handleGoHome = () => {
      window.location.href = "/";
    };

    return (
      <NotFoundPage
        onGoHome={handleGoHome}
        isDev={isDev}
        errorName={`${(error as { status: number }).status}`}
        errorMessage={(error as { statusText: string }).statusText}
        path={typeof window !== "undefined" ? window.location.pathname : undefined}
      />
    );
  }

  const routeError =
    error instanceof Error
      ? error
      : isRouteErrorResponse(error)
        ? new Error(`${error.status}: ${error.statusText}`)
        : new Error("An unexpected error occurred");

  return (
    <div className="flex min-h-screen items-center justify-center bg-background p-4">
      <ErrorBoundaryUi error={routeError} resetError={() => window.location.reload()} />
    </div>
  );
}

export function LazyComponent({
  children,
  onError,
}: {
  children: React.ReactNode;
  onError?: (error: Error, info: React.ErrorInfo) => void;
}) {
  const handleBoundaryError = onError
    ? (error: unknown, info: React.ErrorInfo) => {
        if (error instanceof Error) {
          onError(error, info);
        }
      }
    : undefined;

  return (
    <ErrorBoundary FallbackComponent={ErrorBoundaryFallbackAdapter} onError={handleBoundaryError}>
      <Suspense fallback={<Skeleton className="size-full animate-pulse rounded-md" />}>
        {children}
      </Suspense>
    </ErrorBoundary>
  );
}

type QueryLazyComponentProps = {
  children: React.ReactNode;
  queryKey: QueryKey;
  componentLoaderProps?: ComponentLoaderProps;
};

/**
 * QueryLazyComponent is a wrapper component that allows for lazy loading of components that
 * use react-query.
 * It also resets the query cache when the error boundary is reset.
 */
export function QueryLazyComponent({ children, componentLoaderProps }: QueryLazyComponentProps) {
  return (
    <QueryErrorResetBoundary>
      {() => (
        <SuspenseLoader componentLoaderProps={componentLoaderProps}>{children}</SuspenseLoader>
      )}
    </QueryErrorResetBoundary>
  );
}

export { useErrorBoundary } from "react-error-boundary";
