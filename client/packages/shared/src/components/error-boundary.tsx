import { useQueryErrorResetBoundary } from "@tanstack/react-query";
import type React from "react";
import type { ReactElement, ReactNode } from "react";
import { createContext, Suspense, useCallback, useContext, useEffect, useState } from "react";
import { ErrorBoundary as ReactErrorBoundary, type FallbackProps } from "react-error-boundary";
import {
  isNotFound,
  useCanGoBack,
  useLocation,
  useRouter,
  type ErrorComponentProps,
} from "@tanstack/react-router";
import { DataTableSkeleton } from "./data-table/data-table-skeleton";
import { ErrorState, type ErrorStateLayout } from "./errors/error-state";
import { NotFoundPage, NotFoundState } from "./errors/not-found";
import { StatusScreen } from "./errors/status-screen";
import { Button } from "./ui/button";
import { useT } from "@trenova/shared/i18n/use-t";
import { Home02Icon } from "@trenova/shared/components/icons";
import { describeError } from "@trenova/shared/lib/error-presentation";
import { isRouteRefusal } from "@trenova/shared/lib/router/refusal";
import { reportFailure } from "@trenova/shared/lib/failure-reporter";

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
      reportFailure(error, { source: "boundary" });
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

/**
 * The boundary around a data table: a failure draws ErrorState in place, and while the table's
 * code or first page loads it draws the table's own skeleton. It is for tables only;
 * `bun run lint:skeletons` fails when anything inside it does not render a `DataTable`. Anything
 * else takes `LazyComponent` with a skeleton of its own layout.
 */
export function DataTableLazyComponent({
  children,
  onError,
  columnCount = 10,
  rowCount = 10,
}: {
  children: React.ReactNode;
  onError?: (error: Error, info: React.ErrorInfo) => void;
  columnCount?: number;
  rowCount?: number;
}) {
  return (
    <ErrorStateBoundary onError={onError}>
      <Suspense fallback={<DataTableSkeleton columnCount={columnCount} rowCount={rowCount} />}>
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

/**
 * Whether a route error draws inside a frame that keeps the navigation on screen (the app
 * shell, the Desk) or as the whole window. A layout that keeps its frame standing around a
 * failed page provides true around its outlet.
 */
export const RouteErrorEmbedding = createContext(false);

type RouteErrorBoundaryProps = Partial<ErrorComponentProps> & {
  // homePath is where "Go to dashboard" leads; each app has its own front door.
  homePath?: string;
  // embedded draws the state inside the app shell rather than as the whole window. A route
  // error below the shell leaves the navigation standing, so the person can go elsewhere.
  // Read from RouteErrorEmbedding when not given.
  embedded?: boolean;
};

function useHomeNavigation(homePath: string) {
  const router = useRouter();
  const pathname = useLocation({ select: (location) => location.pathname });
  // Only the entry the app was opened at has nothing inside the app before it, and "Go back"
  // there would leave Trenova.
  const canGoBack = useCanGoBack();

  const goHome = useCallback(() => {
    void router.navigate({ href: homePath });
  }, [router, homePath]);

  const goBack = useCallback(() => {
    router.history.back();
  }, [router]);

  return {
    goHome,
    goBack: canGoBack ? goBack : undefined,
    path: pathname,
  };
}

/**
 * A route's error component: what a gate refused or a page threw while it loaded or rendered.
 * It is the router's default, so every route has one; inside a frame that provides
 * RouteErrorEmbedding it draws in the page's place and the navigation stays.
 */
export function RouteErrorBoundary({ error, homePath = "/", embedded }: RouteErrorBoundaryProps) {
  const t = useT();
  const router = useRouter();
  const { reset: resetQueryErrors } = useQueryErrorResetBoundary();
  const inFrame = useContext(RouteErrorEmbedding);
  const inPlace = embedded ?? inFrame;
  const { goHome, goBack, path } = useHomeNavigation(homePath);

  // A suspense query that failed stays failed until its reset boundary is told otherwise, so
  // the next render, a retry or a visit after navigating away, would re-throw the cached error.
  useEffect(() => {
    resetQueryErrors();
  }, [resetQueryErrors]);

  // Retrying re-runs the route's gates and loaders and resets the router's error boundary.
  const retry = useCallback(() => {
    void router.invalidate();
  }, [router]);

  if (isNotFound(error) || (isRouteRefusal(error) && error.status === 404)) {
    return inPlace ? (
      <NotFoundState path={path} onGoHome={goHome} onGoBack={goBack} className="min-h-full" />
    ) : (
      <NotFoundPage path={path} onGoHome={goHome} onGoBack={goBack} />
    );
  }

  // A page the organization's plan leaves out is not a failure: going home is the way on, so
  // it is the primary action, and no status code sits at the foot.
  const description = describeError(error);
  const isPlanRestricted = description.kind === "plan-restricted";
  // A refused permission, a plan restriction or a missing page gives the same answer again, so
  // only what can succeed on a second run offers the retry.
  const onRetry = description.retryable ? retry : undefined;

  const homeAction = (
    <Button
      variant={isPlanRestricted ? "default" : "outline"}
      size={inPlace ? "sm" : "default"}
      onClick={goHome}
    >
      <Home02Icon />
      {t("Go to dashboard")}
    </Button>
  );

  if (inPlace) {
    return (
      <ErrorState
        error={error}
        layout="section"
        onRetry={onRetry}
        actions={homeAction}
        className="min-h-full"
      />
    );
  }

  return (
    <StatusScreen
      meta={isRouteRefusal(error) && !isPlanRestricted ? String(error.status) : undefined}
    >
      <ErrorState error={error} layout="screen" onRetry={onRetry} actions={homeAction} />
    </StatusScreen>
  );
}

// NotFoundRoute is the catch-all inside an app's shell, so an unknown address keeps the
// navigation on screen instead of dropping the person onto a bare page.
export function NotFoundRoute({ homePath = "/" }: { homePath?: string }) {
  const { goHome, goBack, path } = useHomeNavigation(homePath);
  return <NotFoundState path={path} onGoHome={goHome} onGoBack={goBack} className="min-h-full" />;
}

/**
 * The boundary around anything that loads and is not a data table: a failure draws ErrorState in
 * place, and while its code or suspense queries load it draws `fallback`. The fallback is
 * required and is the skeleton of what loads, laid out where the real thing lands, so the first
 * paint is the page's own shape rather than a generic block that jumps when the answer arrives.
 * `null` is the right fallback only for what is not on screen yet, such as a closed dialog.
 */
export function LazyComponent({
  children,
  fallback,
  onError,
}: {
  children: React.ReactNode;
  fallback: ReactElement | null;
  onError?: (error: Error, info: React.ErrorInfo) => void;
}) {
  return (
    <ErrorStateBoundary onError={onError}>
      <Suspense fallback={fallback}>{children}</Suspense>
    </ErrorStateBoundary>
  );
}

export { useErrorBoundary } from "react-error-boundary";
