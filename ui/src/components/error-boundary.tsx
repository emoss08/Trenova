import {
  faChevronRight,
  faExclamationTriangle,
  faHome,
  faRefresh,
} from "@fortawesome/pro-regular-svg-icons";
import { QueryErrorResetBoundary, QueryKey } from "@tanstack/react-query";
import { ErrorInfo } from "react";
import { ErrorBoundary } from "react-error-boundary";
import { useRouteError } from "react-router";
import SuperJSON from "superjson";
import { NotFoundPage } from "./boundaries/not-found";
import { Alert, AlertDescription, AlertTitle } from "./ui/alert";
import { Button } from "./ui/button";
import {
  Card,
  CardContent,
  CardFooter,
  CardHeader,
  CardTitle,
} from "./ui/card";
import { ComponentLoaderProps, SuspenseLoader } from "./ui/component-loader";
import { Icon } from "./ui/icons";
import { Separator } from "./ui/separator";

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
    (typeof error === "string" ? error : SuperJSON.stringify(error));

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
            <Icon icon={faRefresh} className="h-4 w-4" />
            Retry Current Page
          </Button>

          <Button
            onClick={handleReload}
            size="sm"
            className="flex items-center gap-2"
          >
            <Icon icon={faHome} className="h-4 w-4" />
            Return Home
            <Icon icon={faChevronRight} className="h-4 w-4" />
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
interface LazyLoadErrorFallbackProps {
  error: Error;
  resetErrorBoundary: () => void;
}

export function LazyLoadErrorFallback({
  error,
  resetErrorBoundary,
}: LazyLoadErrorFallbackProps) {
  return (
    <Card className="shadow-md overflow-hidden">
      <CardHeader>
        <div className="flex items-center gap-2">
          <Icon icon={faExclamationTriangle} className="size-5" />
          <CardTitle>Component Failed to Load</CardTitle>
        </div>
      </CardHeader>

      <CardContent className="pt-4">
        <p className="text-muted-foreground">
          This section of the application couldn&apos;t be loaded. This may be
          due to a network issue or a recent deployment.
        </p>

        <div className="mt-4 mb-2">
          <h4 className="text-xs font-medium text-muted-foreground mb-1">
            Technical details:
          </h4>
          <div className="bg-muted rounded-md p-2 text-sm font-mono overflow-auto max-h-28">
            {error.message}
          </div>
        </div>
      </CardContent>

      <CardFooter className="py-3 flex justify-end">
        <Button
          variant="outline"
          size="sm"
          onClick={resetErrorBoundary}
          className="flex items-center gap-2"
        >
          <Icon icon={faRefresh} className="h-4 w-4" />
          Try Again
        </Button>
      </CardFooter>
    </Card>
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
