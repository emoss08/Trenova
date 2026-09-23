import {
  QueryErrorResetBoundary,
  type QueryKey,
  useQueryErrorResetBoundary,
} from "@tanstack/react-query";
import type React from "react";
import type { ReactNode } from "react";
import { Suspense, useCallback, useState } from "react";
import { ErrorBoundary as ReactErrorBoundary, type FallbackProps } from "react-error-boundary";
import { isRouteErrorResponse, useLocation, useNavigate, useRouteError } from "react-router";
import { SuspenseLoader, type ComponentLoaderProps } from "./component-loader";
import { DataTableSkeleton } from "./data-table/data-table-skeleton";
import { ErrorState, type ErrorStateLayout } from "./errors/error-state";
import { NotFoundPage, NotFoundState } from "./errors/not-found";
import { StatusScreen } from "./errors/status-screen";
import { Button } from "./ui/button";
import { Skeleton } from "./ui/skeleton";
import { useT } from "@trenova/shared/i18n/use-t";
import { HouseIcon } from "lucide-react";

type ErrorStateBoundaryProps = {
  children: ReactNode;
  layout?: ErrorStateLayout;
  onError?: (error: Error, info: React.ErrorInfo) => void;
};

// ErrorStateBoundary catches a render or suspense failure below it and draws ErrorState in
// its place. Retrying resets TanStack Query's error state as well as the boundary: a suspense
// query that failed stays failed until its reset boundary is told otherwise, so resetting
// only the React boundary re-throws the same cached error and "Try again" does nothing.
export function ErrorStateBoundary({
  children,
  layout = "section",
  onError,
}: ErrorStateBoundaryProps) {
  const { reset } = useQueryErrorResetBoundary();
  const [componentStack, setComponentStack] = useState<string | null>(null);

  const handleError = useCallback(
    (error: unknown, info: React.ErrorInfo) => {
      setComponentStack(info.componentStack ?? null);
      if (onError && error instanceof Error) {
        onError(error, info);
      }
    },
    [onError],
  );

  const handleReset = useCallback(() => {
    setComponentStack(null);
    reset();
  }, [reset]);

  const renderFallback = useCallback(
    ({ error, resetErrorBoundary }: FallbackProps) => (
      <ErrorState
        error={error}
        onRetry={resetErrorBoundary}
        componentStack={componentStack}
        layout={layout}
      />
    ),
    [componentStack, layout],
  );

  return (
    <ReactErrorBoundary fallbackRender={renderFallback} onError={handleError} onReset={handleReset}>
      {children}
    </ReactErrorBoundary>
  );
}

export function DataTableLazyComponent({
  children,
  onError,
  fallback,
  columnCount = 10,
  rowCount = 10,
}: {
  children: React.ReactNode;
  onError?: (error: Error, info: React.ErrorInfo) => void;
  fallback?: ReactNode;
  columnCount?: number;
  rowCount?: number;
}) {
  return (
    <ErrorStateBoundary onError={onError}>
      <Suspense
        fallback={fallback ?? <DataTableSkeleton columnCount={columnCount} rowCount={rowCount} />}
      >
        {children}
      </Suspense>
    </ErrorStateBoundary>
  );
}

type ErrorBoundaryProps = {
  children: ReactNode;
};

// RootErrorBoundary is the last net, above the router and the providers. What reaches it has
// taken the whole app down, so it draws the full-window frame and offers a reload.
export function RootErrorBoundary({ children }: ErrorBoundaryProps) {
  return (
    <ReactErrorBoundary
      fallbackRender={({ error }) => (
        <StatusScreen>
          <ErrorState error={error} layout="screen" />
        </StatusScreen>
      )}
    >
      {children}
    </ReactErrorBoundary>
  );
}

type RouteErrorBoundaryProps = {
  // homePath is where "Go to dashboard" leads; each app has its own front door.
  homePath?: string;
  // embedded draws the state inside the app shell rather than as the whole window. A route
  // error below the shell leaves the navigation standing, so the person can go elsewhere.
  embedded?: boolean;
};

function useHomeNavigation(homePath: string) {
  const navigate = useNavigate();
  const location = useLocation();

  const goHome = useCallback(() => {
    void navigate(homePath);
  }, [navigate, homePath]);

  // location.key is "default" only on the entry the app was opened at, so there is nothing
  // inside the app to go back to and "Go back" would leave Trenova.
  const goBack = useCallback(() => {
    void navigate(-1);
  }, [navigate]);

  return {
    goHome,
    goBack: location.key === "default" ? undefined : goBack,
    path: location.pathname,
  };
}

export function RouteErrorBoundary({ homePath = "/", embedded = false }: RouteErrorBoundaryProps) {
  const t = useT();
  const error = useRouteError();
  const { goHome, goBack, path } = useHomeNavigation(homePath);

  if (isRouteErrorResponse(error) && error.status === 404) {
    return embedded ? (
      <NotFoundState path={path} onGoHome={goHome} onGoBack={goBack} className="min-h-full" />
    ) : (
      <NotFoundPage path={path} onGoHome={goHome} onGoBack={goBack} />
    );
  }

  const homeAction = (
    <Button variant="outline" size={embedded ? "sm" : "default"} onClick={goHome}>
      <HouseIcon />
      {t("Go to dashboard")}
    </Button>
  );

  if (embedded) {
    return (
      <ErrorState error={error} layout="section" actions={homeAction} className="min-h-full" />
    );
  }

  return (
    <StatusScreen meta={isRouteErrorResponse(error) ? String(error.status) : undefined}>
      <ErrorState error={error} layout="screen" actions={homeAction} />
    </StatusScreen>
  );
}

// NotFoundRoute is the catch-all inside an app's shell, so an unknown address keeps the
// navigation on screen instead of dropping the person onto a bare page.
export function NotFoundRoute({ homePath = "/" }: { homePath?: string }) {
  const { goHome, goBack, path } = useHomeNavigation(homePath);
  return <NotFoundState path={path} onGoHome={goHome} onGoBack={goBack} className="min-h-full" />;
}

export function LazyComponent({
  children,
  onError,
}: {
  children: React.ReactNode;
  onError?: (error: Error, info: React.ErrorInfo) => void;
}) {
  return (
    <ErrorStateBoundary onError={onError}>
      <Suspense fallback={<Skeleton className="size-full rounded-md" />}>{children}</Suspense>
    </ErrorStateBoundary>
  );
}

type QueryLazyComponentProps = {
  children: React.ReactNode;
  queryKey: QueryKey;
  componentLoaderProps?: ComponentLoaderProps;
};

/**
 * QueryLazyComponent is a wrapper component that allows for lazy loading of components that
 * use react-query. A failure draws ErrorState in place, and retrying resets the failed
 * queries inside it along with the boundary.
 */
export function QueryLazyComponent({ children, componentLoaderProps }: QueryLazyComponentProps) {
  return (
    <QueryErrorResetBoundary>
      {() => (
        <ErrorStateBoundary>
          <SuspenseLoader componentLoaderProps={componentLoaderProps}>{children}</SuspenseLoader>
        </ErrorStateBoundary>
      )}
    </QueryErrorResetBoundary>
  );
}

export { useErrorBoundary } from "react-error-boundary";
