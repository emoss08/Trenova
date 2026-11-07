import { useRealtimeStore } from "@/stores/realtime-store";
import { WifiIcon, WifiOffIcon } from "@trenova/shared/components/icons";
import { Button } from "@trenova/shared/components/ui/button";
import { useNow } from "@trenova/shared/hooks/use-now";
import { useOnlineStatus } from "@trenova/shared/hooks/use-online-status";
import { useT } from "@trenova/shared/i18n/use-t";
import { formatMinutesSpan, formatUnixTimeWithSeconds } from "@trenova/shared/lib/date";
import { cn } from "@trenova/shared/lib/utils";

export type BoardFreshness = {
  /** When the server built the board on screen, in seconds; zero before the first read. */
  generatedAt: number;
  isError: boolean;
  isFetching: boolean;
  refetch: () => void;
};

export type StatusLineState = "live" | "reconnecting" | "offline" | "failed";

/**
 * Which of the four things the line says. A failed read wins over the connection, since
 * the board on screen is then older than the last thing the dispatcher saw happen.
 */
export function statusLineState(
  freshness: Pick<BoardFreshness, "isError">,
  online: boolean,
  connection: "connecting" | "connected" | "disconnected",
): StatusLineState {
  if (freshness.isError) return "failed";
  if (!online) return "offline";
  if (connection !== "connected") return "reconnecting";
  return "live";
}

/**
 * One line carries the board's freshness for the whole page, so a dispatcher never acts on
 * a board that stopped updating without knowing it: live, reconnecting, offline, or a read
 * that failed and left the last good board on screen.
 */
export function ConsoleStatusLine({ freshness }: { freshness: BoardFreshness }) {
  const t = useT();
  const online = useOnlineStatus();
  const connection = useRealtimeStore((s) => s.connectionState);
  const nowSeconds = useNow("minute");

  const state = statusLineState(freshness, online, connection);
  const updated =
    freshness.generatedAt > 0 ? formatUnixTimeWithSeconds(freshness.generatedAt) : null;
  const age =
    freshness.generatedAt > 0
      ? formatMinutesSpan(Math.max(0, Math.round((nowSeconds - freshness.generatedAt) / 60)))
      : null;

  return (
    <span
      role="status"
      aria-live="polite"
      className="text-muted-foreground inline-flex shrink-0 items-center gap-1.5 text-xs whitespace-nowrap"
    >
      {state === "live" ? (
        <>
          <span className="bg-success size-1.5 shrink-0 rounded-full" aria-hidden />
          <span className="text-foreground font-medium">{t("Live")}</span>
          {updated ? <span>{t("· updated {0}", updated)}</span> : null}
        </>
      ) : state === "reconnecting" ? (
        <>
          <WifiIcon className="text-warning-foreground size-3.5 shrink-0" aria-hidden />
          <span className="text-warning-foreground font-medium">{t("Reconnecting")}</span>
          {updated && age ? <span>{t("· last update {0}, {1} ago", updated, age)}</span> : null}
        </>
      ) : state === "offline" ? (
        <>
          <WifiOffIcon className="text-danger-foreground size-3.5 shrink-0" aria-hidden />
          <span className="text-danger-foreground font-medium">{t("Offline")}</span>
          <span className="truncate">
            {updated
              ? t("· last update {0}. Moves others cover won't show until it's back.", updated)
              : t("· moves others cover won't show until it's back.")}
          </span>
        </>
      ) : (
        <>
          <span className="bg-destructive size-1.5 shrink-0 rounded-full" aria-hidden />
          <span className="text-danger-foreground font-medium">{t("Not updated")}</span>
          {updated ? <span>{t("· last good read {0}", updated)}</span> : null}
          <Button
            variant="link"
            size="xxs"
            className={cn("ml-1", freshness.isFetching && "pointer-events-none opacity-60")}
            disabled={freshness.isFetching}
            onClick={freshness.refetch}
          >
            {t("Try again")}
          </Button>
        </>
      )}
    </span>
  );
}
