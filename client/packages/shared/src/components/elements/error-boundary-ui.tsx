"use client";

import { useT } from "@trenova/shared/i18n/use-t";
import * as React from "react";

import { AlertTriangle, ChevronDown, ChevronUp, Copy, RefreshCw } from "lucide-react";

import { cn } from "@trenova/shared/lib/utils";

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
        line.match(/at\s+(.+?)\s+\((.+?):(\d+):(\d+)\)/) || line.match(/at\s+(.+?):(\d+):(\d+)/);
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
  const t = useT();

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
        "overflow-hidden rounded-lg border border-danger-border bg-danger-subtle dark:border-danger-border dark:bg-danger-subtle/30",
        className,
      )}
    >
      <div className="flex items-start gap-3 p-4">
        <div className="mt-0.5 shrink-0">
          <AlertTriangle className="h-5 w-5 text-danger-foreground" />
        </div>
        <div className="min-w-0 flex-1">
          <h3 className="font-semibold text-danger-foreground">
            {isDev ? error.name || t("Error") : t("Something went wrong")}
          </h3>
          <p className="mt-1 text-sm break-words text-danger-foreground">
            {isDev ? error.message : t("An unexpected error occurred. Please try again.")}
          </p>
        </div>
      </div>

      <div className="flex items-center gap-2 px-4 pb-4">
        {resetError && (
          <button
            type="button"
            onClick={resetError}
            aria-label={t("Try again")}
            className="flex items-center gap-1.5 rounded bg-danger-subtle px-3 py-1.5 text-sm font-medium text-danger-foreground transition-colors hover:bg-danger-subtle/50 dark:text-danger-foreground dark:hover:bg-danger-subtle"
          >
            <RefreshCw className="h-3.5 w-3.5" />
            {t("Try again")}
          </button>
        )}
        <button
          type="button"
          onClick={handleCopy}
          aria-label={copied ? "Copied to clipboard" : "Copy error details"}
          className="flex items-center gap-1.5 rounded bg-danger-subtle px-3 py-1.5 text-sm font-medium text-danger-foreground transition-colors hover:bg-danger-subtle/50 dark:text-danger-foreground dark:hover:bg-danger-subtle"
        >
          <Copy className="h-3.5 w-3.5" />
          {copied ? t("Copied!") : t("Copy error")}
        </button>
      </div>

      {isDev && error.stack && (
        <div className="border-t border-danger-border">
          <button
            type="button"
            onClick={handleToggleStack}
            aria-expanded={showStack}
            aria-controls="stack-trace-content"
            aria-label={t("Toggle stack trace")}
            className="flex w-full items-center justify-between px-4 py-2 text-sm text-danger-foreground transition-colors hover:bg-danger-subtle dark:text-danger-foreground dark:hover:bg-danger-subtle/30"
          >
            <span className="font-medium">{t("Stack Trace")}</span>
            {showStack ? <ChevronUp className="h-4 w-4" /> : <ChevronDown className="h-4 w-4" />}
          </button>
          {showStack && (
            <div
              id="stack-trace-content"
              className="overflow-auto px-4 pb-4"
              aria-label={t("Error stack trace")}
            >
              <div className="space-y-1 font-mono text-xs">
                {stackFrames.map((frame, idx) => (
                  <div key={idx} className="flex gap-2 text-danger-foreground">
                    <span className="shrink-0 text-danger-foreground">at</span>
                    <span className="text-danger-foreground">{frame.fn}</span>
                    <span className="truncate text-danger-foreground">
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
        <div className="border-t border-danger-border">
          <button
            type="button"
            onClick={handleToggleComponentStack}
            aria-expanded={showComponentStack}
            aria-controls="component-stack-content"
            aria-label={t("Toggle component stack")}
            className="flex w-full items-center justify-between px-4 py-2 text-sm text-danger-foreground transition-colors hover:bg-danger-subtle dark:text-danger-foreground dark:hover:bg-danger-subtle/30"
          >
            <span className="font-medium">{t("Component Stack")}</span>
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
              aria-label={t("Component stack trace")}
            >
              <pre className="font-mono text-xs whitespace-pre-wrap text-danger-foreground">
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
