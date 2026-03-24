import { cn } from "@/lib/utils";
import { QueryErrorResetBoundary, type QueryKey } from "@tanstack/react-query";
import {
  AlertTriangle,
  Bug,
  ClipboardIcon,
  ExternalLink,
  RefreshCw,
  Trash2,
  WifiOff,
} from "lucide-react";
import type React from "react";
import type { ReactNode } from "react";
import { Suspense, useEffect, useMemo, useState } from "react";
import {
  ErrorBoundary,
  ErrorBoundary as ReactErrorBoundary,
  type FallbackProps,
} from "react-error-boundary";
import { isRouteErrorResponse, useRouteError } from "react-router";
import { SuspenseLoader, type ComponentLoaderProps } from "./component-loader";
import { DataTableSkeleton } from "./data-table/data-table-skeleton";
import { NotFoundPage } from "./not-found-page";
import { Button } from "./ui/button";
import { Card, CardContent, CardDescription, CardFooter, CardHeader, CardTitle } from "./ui/card";
import { ScrollArea } from "./ui/scroll-area";
import { Skeleton } from "./ui/skeleton";
import { Tooltip, TooltipContent, TooltipTrigger } from "./ui/tooltip";

type LazyLoadErrorFallbackProps = {
  error: Error;
  resetErrorBoundary: () => void;
};

function normalizeFallbackError(error: unknown): Error {
  return error instanceof Error ? error : new Error(String(error));
}

function LazyLoadErrorFallbackAdapter(props: FallbackProps) {
  return (
    <LazyLoadErrorFallback
      error={normalizeFallbackError(props.error)}
      resetErrorBoundary={props.resetErrorBoundary}
    />
  );
}

export function LazyLoadErrorFallback({ error, resetErrorBoundary }: LazyLoadErrorFallbackProps) {
  const [copied, setCopied] = useState(false);
  const [clearing, setClearing] = useState(false);
  const [online, setOnline] = useState(typeof navigator !== "undefined" ? navigator.onLine : true);

  useEffect(() => {
    const on = () => setOnline(true);
    const off = () => setOnline(false);
    window.addEventListener("online", on);
    window.addEventListener("offline", off);
    return () => {
      window.removeEventListener("online", on);
      window.removeEventListener("offline", off);
    };
  }, []);

  const diag = useMemo(() => diagnoseError(error, online), [error, online]);

  async function copyDetails() {
    const payload = [
      `Message: ${error?.message ?? "(no message)"}`,
      error?.stack ? `\nStack:\n${error.stack}` : "",
      `\nURL: ${typeof location !== "undefined" ? location.href : "(unknown)"}`,
      typeof navigator !== "undefined" ? `User-Agent: ${navigator.userAgent}` : "",
      `\nDiagnostics: ${diag.labels.join(", ")}`,
    ]
      .filter(Boolean)
      .join("\n");

    await navigator.clipboard
      .writeText(payload)
      .then(() => {
        setCopied(true);
        setTimeout(() => setCopied(false), 1200);
      })
      .catch(() => {
        // Do nothing
      });
  }

  async function clearAppCacheAndReload() {
    setClearing(true);
    await Promise.resolve()
      .then(async () => {
        if ("caches" in window) {
          const keys = await caches.keys();
          await Promise.all(keys.map((k) => caches.delete(k)));
        }
        // Unregister any service workers
        if (navigator.serviceWorker?.getRegistrations) {
          const regs = await navigator.serviceWorker.getRegistrations();
          await Promise.all(regs.map((r) => r.unregister()));
        }
        // Best-effort wipe of IndexedDB (not supported everywhere)
        const anyIDB: any = indexedDB as any;
        if (anyIDB?.databases) {
          const dbs = await anyIDB.databases();
          await Promise.all(
            dbs.map((db: any) => db?.name && indexedDB.deleteDatabase(db.name as string)),
          );
        }
      })
      .finally(() => {
        setClearing(false);
        // Hard reload
        window.location.reload();
      });
  }

  return (
    <Card className={cn("relative overflow-hidden border border-border")}>
      <CardHeader>
        <div className="flex items-start justify-between gap-3">
          <div className="flex items-center gap-2">
            <div>
              <CardTitle className="leading-tight">Component failed to load</CardTitle>
              <CardDescription>
                We couldn&apos;t load this section — usually it&apos;s a momentary hiccup.
              </CardDescription>
            </div>
          </div>
        </div>
      </CardHeader>
      <CardContent className="space-y-3">
        <div className="w-full overflow-hidden rounded-md border">
          <ScrollArea className="h-40 bg-background px-2">
            <div className="w-full font-mono text-xs leading-relaxed wrap-break-word whitespace-pre-wrap">
              {error?.message}
              {error?.stack ? "\n\n" + error.stack : ""}
            </div>
          </ScrollArea>
        </div>
      </CardContent>
      <CardFooter className="flex flex-wrap items-end justify-end gap-2">
        <div className="flex items-center gap-2">
          <Tooltip>
            <TooltipTrigger>
              <Button variant="outline" size="sm" onClick={copyDetails}>
                {copied ? <Bug className="size-4" /> : <ClipboardIcon className="size-4" />}
                {copied ? "Copied" : "Copy details"}
              </Button>
            </TooltipTrigger>
            <TooltipContent>Copy message, stack, and diagnostics</TooltipContent>
          </Tooltip>
          <Button variant="secondary" size="sm" onClick={() => window.location.reload()}>
            <ExternalLink className="size-4" />
            Reload page
          </Button>
          <Button size="sm" onClick={resetErrorBoundary}>
            <RefreshCw className="size-4" />
            Try again
          </Button>
          <Button
            size="sm"
            variant="destructive"
            className="gap-2"
            onClick={clearAppCacheAndReload}
            disabled={clearing}
            title="Clears app caches & service worker, then reloads"
          >
            <Trash2 className="size-4" />
            {clearing ? "Clearing…" : "Clear cache & reload"}
          </Button>
        </div>
      </CardFooter>
    </Card>
  );
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
    <ErrorBoundary FallbackComponent={LazyLoadErrorFallbackAdapter} onError={handleBoundaryError}>
      <Suspense fallback={<DataTableSkeleton columnCount={10} rowCount={10} />}>
        {children}
      </Suspense>
    </ErrorBoundary>
  );
}

function diagnoseError(error: Error, online: boolean) {
  const msg = (error?.message || "").toLowerCase();
  const labels: string[] = [];
  const tips: { icon: React.ReactNode; text: string }[] = [];

  const isChunk =
    /chunkload|loading chunk|dynamically imported module|import\(|failed to fetch/.test(msg);
  const isSyntax = /syntaxerror|unexpected token/.test(msg);
  const isCors = /cors/.test(msg);

  if (!online) {
    labels.push("offline");
    tips.push({
      icon: <WifiOff className="mt-0.5 size-4 text-foreground/60" />,
      text: "You appear to be offline. Reconnect to the network and try again.",
    });
  }
  if (isChunk) {
    labels.push("chunk load");
    tips.push({
      icon: <AlertTriangle className="mt-0.5 size-4 text-foreground/60" />,
      text: "A new deploy may have invalidated the chunk. Use ‘Try again’ or Reload the page.",
    });
  }
  if (isSyntax) {
    labels.push("syntax error");
    tips.push({
      icon: <Bug className="mt-0.5 size-4 text-foreground/60" />,
      text: "Looks like a script parse error. Try a hard reload. If it persists, report to the team.",
    });
  }
  if (isCors) {
    labels.push("cors");
    tips.push({
      icon: <AlertTriangle className="mt-0.5 size-4 text-foreground/60" />,
      text: "CORS might be blocking the chunk from loading. Verify CDN/origin settings.",
    });
  }

  if (labels.length === 0) {
    labels.push("unknown");
    tips.push({
      icon: <AlertTriangle className="mt-0.5 size-4 text-foreground/60" />,
      text: "Unknown error. Use ‘Try again’. If it happens repeatedly, copy details and share with the team.",
    });
  }

  return { labels, tips } as const;
}

type ErrorBoundaryProps = {
  children: ReactNode;
};

export function RootErrorBoundary({ children }: ErrorBoundaryProps) {
  return <ReactErrorBoundary FallbackComponent={ErrorFallback}>{children}</ReactErrorBoundary>;
}

export function RouteErrorBoundary() {
  const error = useRouteError();
  const isDev = import.meta.env.DEV;
  const isNotFoundRoute = isRouteErrorResponse(error) && error.status === 404;

  let title = "Something went wrong";
  let message =
    "We encountered an unexpected error. Our team has been notified and is working to resolve the issue.";
  let errorDetails: {
    name: string;
    message: string;
    stack?: string;
  } | null = null;

  if (isRouteErrorResponse(error)) {
    switch (error.status) {
      case 401:
        title = "Unauthorized";
        message = "You don't have permission to access this page.";
        break;
      case 403:
        title = "Access denied";
        message = "You don't have the required permissions to view this resource.";
        break;
      case 404:
        title = "Page not found";
        message = "The page you're looking for doesn't exist or has been moved.";
        break;
      default:
        if (error.status >= 500) {
          title = "Server error";
          message = "Something went wrong on our end. Please try again later.";
        }
    }
    errorDetails = {
      name: `${error.status}`,
      message: error.statusText,
    };
  } else if (error instanceof Error) {
    errorDetails = {
      name: error.name,
      message: error.message,
      stack: error.stack,
    };
  }

  const handleGoHome = () => {
    window.location.href = "/";
  };

  const handleGoBack = () => {
    window.history.back();
  };

  const handleReload = () => {
    window.location.reload();
  };

  if (isNotFoundRoute) {
    return (
      <NotFoundPage
        onGoHome={handleGoHome}
        isDev={isDev}
        errorName={errorDetails?.name}
        errorMessage={errorDetails?.message}
        path={typeof window !== "undefined" ? window.location.pathname : undefined}
      />
    );
  }

  return (
    <div className="flex min-h-screen items-center justify-center bg-background p-4">
      <div className="w-full max-w-lg">
        <div className="rounded-xl border border-border bg-card p-8 shadow-sm">
          <div className="flex flex-col items-center text-center">
            <div className="flex size-16 items-center justify-center rounded-full bg-destructive/10">
              <svg
                className="size-8 text-destructive"
                fill="none"
                viewBox="0 0 24 24"
                stroke="currentColor"
                strokeWidth={1.5}
              >
                <path
                  strokeLinecap="round"
                  strokeLinejoin="round"
                  d="M12 9v3.75m-9.303 3.376c-.866 1.5.217 3.374 1.948 3.374h14.71c1.73 0 2.813-1.874 1.948-3.374L13.949 3.378c-.866-1.5-3.032-1.5-3.898 0L2.697 16.126ZM12 15.75h.007v.008H12v-.008Z"
                />
              </svg>
            </div>

            <h1 className="mt-6 text-xl font-semibold text-foreground">{title}</h1>
            <p className="mt-2 text-sm text-muted-foreground">{message}</p>

            {isDev && errorDetails && (
              <div className="mt-6 w-full rounded-lg bg-muted/50 p-4 text-left">
                <p className="text-xs font-medium text-muted-foreground">
                  Error Details (Development Only)
                </p>
                <p className="mt-1 font-mono text-xs text-destructive">
                  {errorDetails.name}: {errorDetails.message}
                </p>
                {errorDetails.stack && (
                  <pre className="mt-2 max-h-32 overflow-auto font-mono text-xs text-muted-foreground">
                    {errorDetails.stack}
                  </pre>
                )}
              </div>
            )}

            <div className="mt-8 flex w-full flex-col gap-3 sm:flex-row sm:justify-center">
              <Button onClick={handleReload} variant="default" className="w-full sm:w-auto">
                Try Again
              </Button>
              <Button onClick={handleGoBack} variant="outline" className="w-full sm:w-auto">
                Go Back
              </Button>
              <Button onClick={handleGoHome} variant="ghost" className="w-full sm:w-auto">
                Go to Home
              </Button>
            </div>
          </div>
        </div>

        <p className="mt-6 text-center text-xs text-muted-foreground">
          If this problem persists, please contact support.
        </p>
      </div>
    </div>
  );
}

function ErrorFallback({ error, resetErrorBoundary }: FallbackProps) {
  const isDev = import.meta.env.DEV;

  const handleGoHome = () => {
    window.location.href = "/";
  };

  return (
    <div className="flex min-h-screen items-center justify-center bg-background p-4">
      <div className="w-full max-w-lg">
        <div className="rounded-xl border border-border bg-card p-8 shadow-sm">
          <div className="flex flex-col items-center text-center">
            <div className="flex size-16 items-center justify-center rounded-full bg-destructive/10">
              <svg
                className="size-8 text-destructive"
                fill="none"
                viewBox="0 0 24 24"
                stroke="currentColor"
                strokeWidth={1.5}
              >
                <path
                  strokeLinecap="round"
                  strokeLinejoin="round"
                  d="M12 9v3.75m-9.303 3.376c-.866 1.5.217 3.374 1.948 3.374h14.71c1.73 0 2.813-1.874 1.948-3.374L13.949 3.378c-.866-1.5-3.032-1.5-3.898 0L2.697 16.126ZM12 15.75h.007v.008H12v-.008Z"
                />
              </svg>
            </div>

            <h1 className="mt-6 text-xl font-semibold text-foreground">Something went wrong</h1>
            <p className="mt-2 text-sm text-muted-foreground">
              We encountered an unexpected error. Our team has been notified and is working to
              resolve the issue.
            </p>

            {isDev && error instanceof Error && (
              <div className="mt-6 w-full rounded-lg bg-muted/50 p-4 text-left">
                <p className="text-xs font-medium text-muted-foreground">
                  Error Details (Development Only)
                </p>
                <p className="mt-1 font-mono text-xs text-destructive">
                  {error.name}: {error.message}
                </p>
                {error.stack && (
                  <pre className="mt-2 max-h-32 overflow-auto font-mono text-xs text-muted-foreground">
                    {error.stack}
                  </pre>
                )}
              </div>
            )}

            <div className="mt-8 flex w-full flex-col gap-3 sm:flex-row sm:justify-center">
              <Button onClick={resetErrorBoundary} variant="default" className="w-full sm:w-auto">
                Try Again
              </Button>
              <Button onClick={handleGoHome} variant="outline" className="w-full sm:w-auto">
                Go to Home
              </Button>
            </div>
          </div>
        </div>

        <p className="mt-6 text-center text-xs text-muted-foreground">
          If this problem persists, please contact support.
        </p>
      </div>
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
  // Adapter for onError signature
  const handleBoundaryError = onError
    ? (error: unknown, info: React.ErrorInfo) => {
        if (error instanceof Error) {
          onError(error, info);
        }
      }
    : undefined;

  return (
    <ErrorBoundary FallbackComponent={LazyLoadErrorFallbackAdapter} onError={handleBoundaryError}>
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
