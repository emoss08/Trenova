/*
 * Copyright 2023-2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

import { cn } from "@/lib/utils";
import {
  faChevronRight,
  faExclamationTriangle,
  faHome,
  faRefresh,
} from "@fortawesome/pro-regular-svg-icons";
import { QueryErrorResetBoundary, QueryKey } from "@tanstack/react-query";
import {
  AlertTriangle,
  Bug,
  ChevronDown,
  ChevronUp,
  ClipboardIcon,
  ExternalLink,
  RefreshCw,
  Sparkles,
  Trash2,
  WifiOff,
} from "lucide-react";
import React, {
  ErrorInfo,
  Suspense,
  useEffect,
  useMemo,
  useState,
} from "react";
import { ErrorBoundary } from "react-error-boundary";
import { useRouteError } from "react-router";
import { NotFoundPage } from "./boundaries/not-found";
import { Alert, AlertDescription, AlertTitle } from "./ui/alert";
import { Badge } from "./ui/badge";
import { Button } from "./ui/button";
import {
  Card,
  CardContent,
  CardDescription,
  CardFooter,
  CardHeader,
  CardTitle,
} from "./ui/card";
import { ComponentLoaderProps, SuspenseLoader } from "./ui/component-loader";
import { Icon } from "./ui/icons";
import { ScrollArea } from "./ui/scroll-area";
import { Separator } from "./ui/separator";
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from "./ui/tooltip";

interface ErrorDetails {
  status?: number;
  statusText?: string;
  message?: string;
  stack?: string;
}

/**
 * Root Error Boundary - Handles application-wide errors
 *
 * This component is used as a fallback when an error occurs at the route level.
 * It provides clear error information and navigation options for users.
 */
export function RootErrorBoundary() {
  const error = useRouteError() as Error & ErrorDetails;
  const isHttpError = !!(error?.status && error?.statusText);

  const errorMessage =
    error?.message ||
    (typeof error === "string" ? error : JSON.stringify(error));

  const errorTitle = isHttpError
    ? `${error.status} - ${error.statusText}`
    : "Application Error";

  const handleReload = () => {
    window.location.href = "/";
  };

  const handleRetry = () => {
    window.location.reload();
  };

  if (error.status === 404) {
    return <NotFoundPage />;
  }

  return (
    <div className="min-h-screen bg-background flex flex-col items-center justify-center p-4">
      <Card className="w-full max-w-md shadow-lg">
        <CardHeader>
          <div className="flex items-center gap-3">
            <Icon
              icon={faExclamationTriangle}
              className="h-6 w-6 text-red-500"
            />
            <CardTitle className="text-red-700">{errorTitle}</CardTitle>
          </div>
        </CardHeader>

        <CardContent className="pt-6">
          <Alert variant="destructive" className="mb-4">
            <AlertTitle>Something went wrong</AlertTitle>
            <AlertDescription>
              We&apos;ve encountered an unexpected error. Our team has been
              notified.
            </AlertDescription>
          </Alert>

          <div className="mt-4">
            <h3 className="text-sm font-medium text-muted-foreground mb-2">
              Error Details:
            </h3>
            <div className="bg-muted rounded-md p-3 text-sm font-mono overflow-auto max-h-32">
              {errorMessage}
            </div>
          </div>
        </CardContent>

        <Separator className="my-2" />

        <CardFooter className="flex justify-between pt-4">
          <Button
            variant="outline"
            size="sm"
            onClick={handleRetry}
            className="flex items-center gap-2"
          >
            <Icon icon={faRefresh} className="size-4" />
            Retry Current Page
          </Button>

          <Button
            onClick={handleReload}
            size="sm"
            className="flex items-center gap-2"
          >
            <Icon icon={faHome} className="size-4" />
            Return Home
            <Icon icon={faChevronRight} className="size-4" />
          </Button>
        </CardFooter>
      </Card>

      <p className="text-gray-500 text-sm mt-8">
        If this issue persists, please contact support.
      </p>
    </div>
  );
}

/**
 * LazyLoadErrorFallback - Handles errors in lazy-loaded components
 *
 * This component is used specifically when a dynamically imported component
 * fails to load, providing a scoped error display with retry functionality.
 */
type LazyLoadErrorFallbackProps = {
  error: Error;
  resetErrorBoundary: () => void;
};

export function LazyLoadErrorFallback({
  error,
  resetErrorBoundary,
}: LazyLoadErrorFallbackProps) {
  const [detailsOpen, setDetailsOpen] = useState(false);
  const [copied, setCopied] = useState(false);
  const [clearing, setClearing] = useState(false);
  const [online, setOnline] = useState(
    typeof navigator !== "undefined" ? navigator.onLine : true,
  );

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
      typeof navigator !== "undefined"
        ? `User-Agent: ${navigator.userAgent}`
        : "",
      `\nDiagnostics: ${diag.labels.join(", ")}`,
    ]
      .filter(Boolean)
      .join("\n");

    try {
      await navigator.clipboard.writeText(payload);
      setCopied(true);
      setTimeout(() => setCopied(false), 1200);
    } catch {
      // Do nothing
    }
  }

  async function clearAppCacheAndReload() {
    setClearing(true);
    try {
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
          dbs.map(
            (db: any) =>
              db?.name && indexedDB.deleteDatabase(db.name as string),
          ),
        );
      }
    } finally {
      setClearing(false);
      // Hard reload
      window.location.reload();
    }
  }

  return (
    <Card
      className={cn(
        "relative overflow-hidden border shadow-sm",
        "bg-gradient-to-br from-background to-muted/40 dark:from-muted/10 dark:to-background/20 m-2",
      )}
    >
      <div
        aria-hidden
        className="pointer-events-none absolute -top-16 -right-16 h-48 w-48 rounded-full bg-primary/10 blur-3xl"
      />

      <CardHeader className="pb-3">
        <div className="flex items-start justify-between gap-3">
          <div className="flex items-center gap-2">
            <span className="inline-flex h-7 w-7 items-center justify-center rounded-full bg-amber-500/20 text-amber-700 dark:text-amber-300">
              <AlertTriangle className="size-4" />
            </span>
            <div>
              <CardTitle className="leading-tight">
                Component failed to load
              </CardTitle>
              <CardDescription>
                We couldn&apos;t load this section — usually it&apos;s a
                momentary hiccup.
              </CardDescription>
            </div>
          </div>
          <Badge
            withDot={false}
            variant={online ? "active" : "inactive"}
            className="shrink-0"
          >
            {online ? "online" : "offline"}
          </Badge>
        </div>

        <div className="mt-3 flex flex-wrap items-center gap-2 text-xs">
          {diag.labels.map((l) => (
            <Badge
              key={l}
              withDot={false}
              variant="secondary"
              className="capitalize"
            >
              {l}
            </Badge>
          ))}
        </div>
      </CardHeader>

      <CardContent className="space-y-3">
        <ul className="text-sm text-muted-foreground space-y-1">
          {diag.tips.map((t, i) => (
            <li key={i} className="flex items-start gap-2">
              {t.icon}
              <span>{t.text}</span>
            </li>
          ))}
        </ul>

        <Separator />

        <div>
          <button
            type="button"
            onClick={() => setDetailsOpen((v) => !v)}
            className="group inline-flex items-center gap-2 text-xs text-muted-foreground hover:text-foreground"
            aria-expanded={detailsOpen}
          >
            {detailsOpen ? (
              <ChevronUp className="size-4" />
            ) : (
              <ChevronDown className="size-4" />
            )}
            Technical details
          </button>
          {detailsOpen && (
            <div className="mt-2 rounded-md border bg-muted/60 w-full">
              <ScrollArea className="h-40 p-2">
                <div className="text-xs leading-relaxed font-mono whitespace-pre-wrap break-words w-full">
                  {error?.message}
                  {error?.stack ? "\n\n" + error.stack : ""}
                </div>
              </ScrollArea>
            </div>
          )}
        </div>
      </CardContent>

      <CardFooter className="flex flex-wrap items-center justify-between gap-2">
        <div className="flex items-center gap-2 text-xs text-muted-foreground">
          <Sparkles className="size-3.5" />
          <span>Pro tip: in dev, a hard reload often fixes chunk errors.</span>
        </div>
        <div className="flex items-center gap-2">
          <TooltipProvider delayDuration={200}>
            <Tooltip>
              <TooltipTrigger asChild>
                <Button variant="outline" size="sm" onClick={copyDetails}>
                  {copied ? (
                    <Bug className="size-4" />
                  ) : (
                    <ClipboardIcon className="size-4" />
                  )}
                  {copied ? "Copied" : "Copy details"}
                </Button>
              </TooltipTrigger>
              <TooltipContent>
                Copy message, stack, and diagnostics
              </TooltipContent>
            </Tooltip>
          </TooltipProvider>

          <Button
            variant="secondary"
            size="sm"
            onClick={() => window.location.reload()}
          >
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

export function LazyLoader({
  children,
  fallback,
  onError,
  onReset,
}: {
  children: React.ReactNode;
  fallback: React.ReactNode;
  onError?: (error: Error, info: ErrorInfo) => void;
  onReset?: () => void;
}) {
  return (
    <ErrorBoundary
      FallbackComponent={LazyLoadErrorFallback}
      onError={onError}
      onReset={onReset}
    >
      <Suspense fallback={fallback}>{children}</Suspense>
    </ErrorBoundary>
  );
}

type LazyComponentProps = {
  children: React.ReactNode;
  onError?: (error: Error, info: ErrorInfo) => void;
  componentLoaderProps?: ComponentLoaderProps;
};

/**
 * LazyComponent is a wrapper component that allows for lazy loading of components
 * with error handling.
 */
export function LazyComponent({
  children,
  onError,
  componentLoaderProps,
}: LazyComponentProps) {
  return (
    <ErrorBoundary FallbackComponent={LazyLoadErrorFallback} onError={onError}>
      <SuspenseLoader componentLoaderProps={componentLoaderProps}>
        {children}
      </SuspenseLoader>
    </ErrorBoundary>
  );
}

type QueryLazyComponentProps = {
  children: React.ReactNode;
  onError?: (error: Error, info: ErrorInfo) => void;
  queryKey: QueryKey;
  componentLoaderProps?: ComponentLoaderProps;
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
  componentLoaderProps,
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
          <SuspenseLoader componentLoaderProps={componentLoaderProps}>
            {children}
          </SuspenseLoader>
        </ErrorBoundary>
      )}
    </QueryErrorResetBoundary>
  );
}

function diagnoseError(error: Error, online: boolean) {
  const msg = (error?.message || "").toLowerCase();
  const labels: string[] = [];
  const tips: { icon: React.ReactNode; text: string }[] = [];

  const isChunk =
    /chunkload|loading chunk|dynamically imported module|import\(|failed to fetch/.test(
      msg,
    );
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
