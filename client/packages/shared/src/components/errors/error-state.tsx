import { Button } from "@trenova/shared/components/ui/button";
import { useOnlineStatus } from "@trenova/shared/hooks/use-online-status";
import { useT } from "@trenova/shared/i18n/use-t";
import { type ErrorDescription, describeError } from "@trenova/shared/lib/error-presentation";
import { cn } from "@trenova/shared/lib/utils";
import { LogInIcon, RefreshCwIcon, RotateCcwIcon } from "lucide-react";
import { type ReactNode, useEffect, useId, useMemo, useRef, useState } from "react";
import { ERROR_ICONS, ERROR_TONE_WELL, errorCopy } from "./error-copy";
import { ErrorDetails, ErrorReference } from "./error-details";

export type ErrorStateLayout = "section" | "compact" | "screen";

export type ErrorStateProps = {
  error: unknown;
  // onRetry re-runs whatever failed (an error boundary's reset, a query's refetch). Without
  // it the state offers a page reload, which is the only retry left.
  onRetry?: () => void;
  componentStack?: string | null;
  layout?: ErrorStateLayout;
  // title and description override the classified copy for a caller that knows better what
  // failed ("Couldn't load the fleet map"). The classification still picks the icon, the
  // actions and the details.
  title?: string;
  description?: string;
  // actions render after the built-in ones, for a way forward only the caller knows.
  actions?: ReactNode;
  className?: string;
};

function reloadPage() {
  window.location.reload();
}

function isConnectivityKind(kind: ErrorDescription["kind"]): boolean {
  return kind === "offline" || kind === "network";
}

// useRetryOnReconnect retries once when the browser comes back online after a connectivity
// failure. The person asked for this view; making them press a button after their Wi-Fi
// returns is busywork. Without a retry of its own the view reloads, which is what the
// offline copy promises.
function useRetryOnReconnect(description: ErrorDescription, onRetry: (() => void) | undefined) {
  const online = useOnlineStatus();
  const wasOffline = useRef(!online);

  useEffect(() => {
    if (!online) {
      wasOffline.current = true;
      return;
    }
    if (wasOffline.current && isConnectivityKind(description.kind)) {
      wasOffline.current = false;
      (onRetry ?? reloadPage)();
    }
  }, [online, onRetry, description.kind]);

  return online;
}

type ErrorActionsProps = {
  description: ErrorDescription;
  onRetry?: () => void;
  size: "sm" | "default";
  extra?: ReactNode;
};

function ErrorActions({ description, onRetry, size, extra }: ErrorActionsProps) {
  const t = useT();

  if (description.kind === "stale-build") {
    return (
      <>
        <Button size={size} onClick={reloadPage}>
          <RefreshCwIcon />
          {t("Reload page")}
        </Button>
        {extra}
      </>
    );
  }

  if (description.kind === "session-expired") {
    return (
      <>
        <Button size={size} onClick={reloadPage}>
          <LogInIcon />
          {t("Sign in again")}
        </Button>
        {extra}
      </>
    );
  }

  if (!onRetry && !description.retryable) {
    return <>{extra}</>;
  }

  if (!onRetry) {
    return (
      <>
        <Button size={size} onClick={reloadPage}>
          <RefreshCwIcon />
          {t("Reload page")}
        </Button>
        {extra}
      </>
    );
  }

  return (
    <>
      <Button size={size} variant={description.retryable ? "default" : "outline"} onClick={onRetry}>
        <RotateCcwIcon />
        {t("Try again")}
      </Button>
      {description.retryable ? (
        <Button size={size} variant="outline" onClick={reloadPage}>
          {t("Reload page")}
        </Button>
      ) : null}
      {extra}
    </>
  );
}

// ErrorState is the one way Trenova says something failed. It classifies the error, says what
// happened and what to do in plain words, and keeps the machinery (status, reference, stack)
// behind a disclosure a person can copy to support in one click.
//
// It is drawn in ink on the surface it replaces. Only the icon well carries the tone, so a
// failed panel reads as a state of that panel rather than as an alarm across the screen.
export function ErrorState({
  error,
  onRetry,
  componentStack,
  layout = "section",
  title,
  description: descriptionOverride,
  actions,
  className,
}: ErrorStateProps) {
  const t = useT();
  const titleId = useId();
  const [occurredAt] = useState(() => new Date());

  const description = useMemo(() => describeError(error), [error]);
  const online = useRetryOnReconnect(description, onRetry);
  const kind = description.kind === "network" && !online ? "offline" : description.kind;
  const resolved = kind === description.kind ? description : { ...description, kind };

  const copy = errorCopy(t, resolved);
  const heading = title ?? copy.title;
  const body = descriptionOverride ?? copy.description;
  const Icon = ERROR_ICONS[resolved.kind];
  const well = ERROR_TONE_WELL[resolved.tone];

  if (layout === "compact") {
    return (
      <section
        role="alert"
        aria-labelledby={titleId}
        data-slot="error-state"
        data-kind={resolved.kind}
        className={cn(
          "animate-rise border-border bg-card flex w-full flex-col gap-3 rounded-lg border p-3",
          className,
        )}
      >
        <div className="flex items-start gap-3">
          <span
            aria-hidden
            className={cn(
              "flex size-7 shrink-0 items-center justify-center rounded-md border",
              well,
            )}
          >
            <Icon className="size-3.5" />
          </span>
          <div className="flex min-w-0 flex-1 flex-col gap-0.5">
            <h3 id={titleId} className="text-sm font-semibold">
              {heading}
            </h3>
            <p className="text-foreground-muted text-xs text-pretty">{body}</p>
          </div>
          <div className="flex shrink-0 items-center gap-1.5">
            <ErrorActions description={resolved} onRetry={onRetry} size="sm" extra={actions} />
          </div>
        </div>
        <ErrorDetails
          description={resolved}
          title={heading}
          occurredAt={occurredAt}
          componentStack={componentStack}
          className="pl-9"
        />
      </section>
    );
  }

  const isScreen = layout === "screen";

  return (
    <section
      role="alert"
      aria-labelledby={titleId}
      data-slot="error-state"
      data-kind={resolved.kind}
      className={cn(
        "animate-rise flex w-full flex-1 flex-col items-center justify-center px-4 text-center",
        isScreen ? "py-16" : "py-12",
        className,
      )}
    >
      <div className="flex w-full max-w-md flex-col items-center">
        <span
          aria-hidden
          className={cn(
            "flex items-center justify-center rounded-lg border",
            isScreen ? "size-12" : "size-10",
            well,
          )}
        >
          <Icon className={isScreen ? "size-5" : "size-4.5"} />
        </span>
        <h2
          id={titleId}
          className={cn("mt-4 font-semibold text-balance", isScreen ? "text-2xl" : "text-lg")}
        >
          {heading}
        </h2>
        <p
          className={cn(
            "text-foreground-muted mt-1.5 max-w-[52ch] text-pretty",
            isScreen ? "text-base" : "text-sm",
          )}
        >
          {body}
        </p>
        {resolved.traceId ? <ErrorReference traceId={resolved.traceId} className="mt-3" /> : null}
        <div className="mt-5 flex flex-wrap items-center justify-center gap-2">
          <ErrorActions
            description={resolved}
            onRetry={onRetry}
            size={isScreen ? "default" : "sm"}
            extra={actions}
          />
        </div>
      </div>
      <ErrorDetails
        description={resolved}
        title={heading}
        occurredAt={occurredAt}
        componentStack={componentStack}
        defaultOpen={import.meta.env.DEV && resolved.kind === "unexpected"}
        className="mt-8 max-w-lg"
      />
    </section>
  );
}
