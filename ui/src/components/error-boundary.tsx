import { QueryErrorResetBoundary, QueryKey } from "@tanstack/react-query";
import { ErrorInfo } from "react";
import { ErrorBoundary, ErrorBoundaryProps } from "react-error-boundary";
import { useRouteError } from "react-router";
import { Button } from "./ui/button";
import { Card, CardDescription, CardTitle } from "./ui/card";
import { SuspenseLoader } from "./ui/component-loader";

export function RootErrorBoundary() {
  const error = useRouteError() as Error;
  return (
    <div>
      <h1>Uh oh, something went terribly wrong 😩</h1>
      <pre>{error.message || JSON.stringify(error)}</pre>
      <button onClick={() => (window.location.href = "/")}>
        Click here to reload the app
      </button>
    </div>
  );
}

// Specific error fallback for lazy-loaded components
function LazyLoadErrorFallback({
  error,
  resetErrorBoundary,
}: {
  error: Error;
  resetErrorBoundary: () => void;
}) {
  return (
    <Card className="m-4">
      <CardTitle>Component Failed to Load</CardTitle>
      <CardDescription className="mt-2">
        <p>This section of the application failed to load.</p>
        <pre className="mt-2 rounded bg-red-50 p-2 text-sm">
          {error.message}
        </pre>
        <Button variant="outline" className="mt-4" onClick={resetErrorBoundary}>
          Try Again
        </Button>
      </CardDescription>
    </Card>
  );
}

/**
 * LazyComponent is a wrapper component that allows for lazy loading of components
 * with error handling.
 */
export function LazyComponent({ children, onError }: ErrorBoundaryProps) {
  return (
    <ErrorBoundary FallbackComponent={LazyLoadErrorFallback} onError={onError}>
      <SuspenseLoader>{children}</SuspenseLoader>
    </ErrorBoundary>
  );
}

type QueryLazyComponentProps = {
  children: React.ReactNode;
  onError?: (error: Error, info: ErrorInfo) => void;
  queryKey: QueryKey;
};

/**
 * QueryLazyComponent is a wrapper component that allows for lazy loading of components that
 * use react-query.
 * It also resets the query cache when the error boundary is reset.
 */
export function QueryLazyComponent({
  children,
  onError,
  queryKey,
}: QueryLazyComponentProps) {
  return (
    <QueryErrorResetBoundary>
      {({ reset }) => (
        <ErrorBoundary
          FallbackComponent={LazyLoadErrorFallback}
          onReset={reset}
          onError={onError}
          resetKeys={queryKey as any}
        >
          <SuspenseLoader>{children}</SuspenseLoader>
        </ErrorBoundary>
      )}
    </QueryErrorResetBoundary>
  );
}
