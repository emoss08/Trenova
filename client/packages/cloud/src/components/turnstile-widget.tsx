import { useResolvedTheme } from "@/hooks/use-resolved-theme";
import { loadTurnstile, type TurnstileApi } from "../lib/turnstile";
import { useT } from "@trenova/shared/i18n/use-t";
import { cn } from "@trenova/shared/lib/utils";
import { useEffect, useImperativeHandle, useRef, useState, type Ref } from "react";

export type TurnstileHandle = {
  /** Discards the current token and asks for a fresh challenge. Tokens are single use. */
  reset: () => void;
};

type TurnstileWidgetProps = {
  siteKey: string;
  /** Names the form in Cloudflare's analytics and is checked server-side. */
  action: string;
  onTokenChange: (token: string | null) => void;
  ref?: Ref<TurnstileHandle>;
  className?: string;
};

type LoadState = "loading" | "ready" | "failed";

/**
 * Cloudflare Turnstile, rendered explicitly into its own container. The token it
 * produces is reported upward and cleared whenever it stops being usable — expired,
 * timed out, errored or reset — so the form can never submit a spent token.
 */
export function TurnstileWidget({
  siteKey,
  action,
  onTokenChange,
  ref,
  className,
}: TurnstileWidgetProps) {
  const t = useT();
  const theme = useResolvedTheme();
  const containerRef = useRef<HTMLDivElement>(null);
  const widgetIdRef = useRef<string | null>(null);
  const apiRef = useRef<TurnstileApi | null>(null);
  const onTokenChangeRef = useRef(onTokenChange);
  const [loadState, setLoadState] = useState<LoadState>("loading");
  const [attempt, setAttempt] = useState(0);

  useEffect(() => {
    onTokenChangeRef.current = onTokenChange;
  });

  useImperativeHandle(
    ref,
    () => ({
      reset: () => {
        onTokenChangeRef.current(null);
        const widgetId = widgetIdRef.current;
        if (widgetId && apiRef.current) {
          apiRef.current.reset(widgetId);
        }
      },
    }),
    [],
  );

  useEffect(() => {
    let cancelled = false;
    const container = containerRef.current;
    if (!container || siteKey === "") {
      return;
    }

    loadTurnstile()
      .then((api) => {
        if (cancelled) {
          return;
        }
        apiRef.current = api;
        const widgetId = api.render(container, {
          sitekey: siteKey,
          action,
          theme,
          size: "flexible",
          "response-field": false,
          "refresh-expired": "auto",
          callback: (token) => onTokenChangeRef.current(token),
          "expired-callback": () => onTokenChangeRef.current(null),
          "timeout-callback": () => onTokenChangeRef.current(null),
          "error-callback": () => {
            onTokenChangeRef.current(null);
          },
        });
        widgetIdRef.current = widgetId ?? null;
        setLoadState("ready");
      })
      .catch(() => {
        if (!cancelled) {
          setLoadState("failed");
        }
      });

    return () => {
      cancelled = true;
      const widgetId = widgetIdRef.current;
      widgetIdRef.current = null;
      if (widgetId && apiRef.current) {
        apiRef.current.remove(widgetId);
      }
      onTokenChangeRef.current(null);
    };
  }, [siteKey, action, theme, attempt]);

  if (siteKey === "") {
    return (
      <p role="alert" className="text-auth-danger m-0 text-xs">
        {t("Bot protection is not configured, so signup is unavailable right now.")}
      </p>
    );
  }

  return (
    <div className={cn("flex flex-col gap-1.5", className)}>
      <div ref={containerRef} className="min-h-[65px] w-full" data-testid="turnstile-container" />
      {loadState === "loading" ? (
        <span className="text-subtle-foreground text-xs">{t("Loading security check…")}</span>
      ) : null}
      {loadState === "failed" ? (
        <span role="alert" className="text-auth-danger text-xs">
          {t("The security check could not load. Check your connection or content blocker.")}{" "}
          <button
            type="button"
            onClick={() => {
              setLoadState("loading");
              setAttempt((current) => current + 1);
            }}
            className="text-foreground cursor-pointer underline underline-offset-[3px]"
          >
            {t("Try again")}
          </button>
        </span>
      ) : null}
    </div>
  );
}
