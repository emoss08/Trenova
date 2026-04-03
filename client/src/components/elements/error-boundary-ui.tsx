"use client";

import * as React from "react";

import {
  AlertTriangle,
  ChevronDown,
  ChevronUp,
  Copy,
  RefreshCw,
} from "lucide-react";

import { cn } from "@/lib/utils";

interface ErrorBoundaryUiProps {
  error: Error;
  resetError?: () => void;
  componentStack?: string | null;
  isDev?: boolean;
  className?: string;
}

function parseStackTrace(
  stack: string,
): { file: string; line: string; column: string; fn: string }[] {
  const lines = stack.split("\n").slice(1);
  return lines
    .map((line) => {
      const match =
        line.match(/at\s+(.+?)\s+\((.+?):(\d+):(\d+)\)/) ||
        line.match(/at\s+(.+?):(\d+):(\d+)/);
      if (match) {
        if (match.length === 5) {
          return {
            fn: match[1],
            file: match[2],
            line: match[3],
            column: match[4],
          };
        }
        return {
          fn: "anonymous",
          file: match[1],
          line: match[2],
          column: match[3],
        };
      }
      return null;
    })
    .filter((x): x is NonNullable<typeof x> => x !== null);
}

export function ErrorBoundaryUi({
  error,
  resetError,
  componentStack,
  isDev = import.meta.env.DEV,
  className,
}: ErrorBoundaryUiProps) {
  const [showStack, setShowStack] = React.useState(isDev);
  const [showComponentStack, setShowComponentStack] = React.useState(false);
  const [copied, setCopied] = React.useState(false);

  const stackFrames = React.useMemo(
    () => (error.stack ? parseStackTrace(error.stack) : []),
    [error.stack],
  );

  const handleCopy = React.useCallback(async () => {
    const errorText = [
      `Error: ${error.message}`,
      "",
      "Stack Trace:",
      error.stack,
      componentStack ? `\nComponent Stack:\n${componentStack}` : "",
    ].join("\n");

    await navigator.clipboard.writeText(errorText);
    setCopied(true);
    setTimeout(() => setCopied(false), 1500);
  }, [error.message, error.stack, componentStack]);

  const handleToggleStack = React.useCallback(() => {
    setShowStack((prev) => !prev);
  }, []);

  const handleToggleComponentStack = React.useCallback(() => {
    setShowComponentStack((prev) => !prev);
  }, []);

  return (
    <div
      data-slot="error-boundary-ui"
      role="alert"
      aria-live="assertive"
      aria-atomic="true"
      className={cn(
        "overflow-hidden rounded-lg border border-red-200 bg-red-50 dark:border-red-900 dark:bg-red-950/30",
        className,
      )}
    >
      <div className="flex items-start gap-3 p-4">
        <div className="mt-0.5 shrink-0">
          <AlertTriangle className="h-5 w-5 text-red-500" />
        </div>
        <div className="min-w-0 flex-1">
          <h3 className="font-semibold text-red-700 dark:text-red-300">
            {isDev ? error.name || "Error" : "Something went wrong"}
          </h3>
          <p className="mt-1 text-sm break-words text-red-600 dark:text-red-400">
            {isDev
              ? error.message
              : "An unexpected error occurred. Please try again."}
          </p>
        </div>
      </div>

      <div className="flex items-center gap-2 px-4 pb-4">
        {resetError && (
          <button
            type="button"
            onClick={resetError}
            aria-label="Try again"
            className="flex items-center gap-1.5 rounded bg-red-100 px-3 py-1.5 text-sm font-medium text-red-700 transition-colors hover:bg-red-200 dark:bg-red-900/50 dark:text-red-300 dark:hover:bg-red-900"
          >
            <RefreshCw className="h-3.5 w-3.5" />
            Try again
          </button>
        )}
        <button
          type="button"
          onClick={handleCopy}
          aria-label={copied ? "Copied to clipboard" : "Copy error details"}
          className="flex items-center gap-1.5 rounded bg-red-100 px-3 py-1.5 text-sm font-medium text-red-700 transition-colors hover:bg-red-200 dark:bg-red-900/50 dark:text-red-300 dark:hover:bg-red-900"
        >
          <Copy className="h-3.5 w-3.5" />
          {copied ? "Copied!" : "Copy error"}
        </button>
      </div>

      {isDev && error.stack && (
        <div className="border-t border-red-200 dark:border-red-900">
          <button
            type="button"
            onClick={handleToggleStack}
            aria-expanded={showStack}
            aria-controls="stack-trace-content"
            aria-label="Toggle stack trace"
            className="flex w-full items-center justify-between px-4 py-2 text-sm text-red-600 transition-colors hover:bg-red-100 dark:text-red-400 dark:hover:bg-red-900/30"
          >
            <span className="font-medium">Stack Trace</span>
            {showStack ? (
              <ChevronUp className="h-4 w-4" />
            ) : (
              <ChevronDown className="h-4 w-4" />
            )}
          </button>
          {showStack && (
            <div
              id="stack-trace-content"
              className="overflow-auto px-4 pb-4"
              aria-label="Error stack trace"
            >
              <div className="space-y-1 font-mono text-xs">
                {stackFrames.map((frame, idx) => (
                  <div
                    key={idx}
                    className="flex gap-2 text-red-600 dark:text-red-400"
                  >
                    <span className="shrink-0 text-red-400 dark:text-red-600">
                      at
                    </span>
                    <span className="text-red-700 dark:text-red-300">
                      {frame.fn}
                    </span>
                    <span className="truncate text-red-500">
                      ({frame.file}:{frame.line}:{frame.column})
                    </span>
                  </div>
                ))}
              </div>
            </div>
          )}
        </div>
      )}

      {isDev && componentStack && (
        <div className="border-t border-red-200 dark:border-red-900">
          <button
            type="button"
            onClick={handleToggleComponentStack}
            aria-expanded={showComponentStack}
            aria-controls="component-stack-content"
            aria-label="Toggle component stack"
            className="flex w-full items-center justify-between px-4 py-2 text-sm text-red-600 transition-colors hover:bg-red-100 dark:text-red-400 dark:hover:bg-red-900/30"
          >
            <span className="font-medium">Component Stack</span>
            {showComponentStack ? (
              <ChevronUp className="h-4 w-4" />
            ) : (
              <ChevronDown className="h-4 w-4" />
            )}
          </button>
          {showComponentStack && (
            <div
              id="component-stack-content"
              className="overflow-auto px-4 pb-4"
              aria-label="Component stack trace"
            >
              <pre className="font-mono text-xs whitespace-pre-wrap text-red-600 dark:text-red-400">
                {componentStack}
              </pre>
            </div>
          )}
        </div>
      )}
    </div>
  );
}

export type { ErrorBoundaryUiProps };
