import type { WatchtowerStopPanel } from "@/lib/graphql/watchtower";
import { Badge } from "@trenova/shared/components/ui/badge";
import { useT, type TranslateFn } from "@trenova/shared/i18n/use-t";
import { cn } from "@trenova/shared/lib/utils";
import { formatFactTime } from "../../watchtower/lanes";
import { Figure, Figures, PanelBody, PanelCard } from "./panel-parts";
import { agoOrIn, spanBetween, trackShare } from "./panel-time";

type PanelTime = { now: number; timezone: string };

function verdictBadge(t: TranslateFn, stop: WatchtowerStopPanel) {
  switch (stop.verdict) {
    case "Missed":
      return (
        <Badge variant="danger" dot>
          {t("Missed")}
        </Badge>
      );
    case "ArrivedLate":
      return (
        <Badge variant="danger" dot>
          {t("Arrived late")}
        </Badge>
      );
    case "Late":
      return (
        <Badge variant="danger" dot>
          {t("Projected late")}
        </Badge>
      );
    default:
      return (
        <Badge variant="warning" dot>
          {t("At risk")}
        </Badge>
      );
  }
}

/**
 * A stop and its window, on one line of time: where the window closed, where
 * the truck is projected to arrive, and where now is. The stretch past the
 * window is what is late, and is the part drawn in colour.
 */
export function StopPanel({ stop, now, timezone }: { stop: WatchtowerStopPanel } & PanelTime) {
  const t = useT();
  const closed = stop.windowEnd;
  const eta = stop.estimatedAt;
  const arrived = stop.arrivedAt;
  // The arrival the lateness is measured to: when it arrived, else the
  // projection, else now once the window is missed.
  const reach = arrived > 0 ? arrived : stop.verdict === "Missed" || eta <= 0 ? now : eta;
  const late = closed > 0 && reach > closed;
  const marks = [closed, eta, arrived, now].filter((at) => at > 0);
  const first = Math.min(...marks);
  const last = Math.max(...marks);
  const pad = Math.max((last - first) * 0.18, 900);
  const start = first - pad;
  const end = last + pad;
  const role = stop.role === "Pickup" ? t("Pickup") : t("Delivery");

  return (
    <PanelCard
      eyebrow={role}
      title={stop.place || t("The {0} stop", role.toLocaleLowerCase())}
      badge={verdictBadge(t, stop)}
    >
      {closed > 0 && (
        <PanelBody>
          <div
            className="relative mt-1 h-9"
            role="img"
            aria-label={
              late
                ? t("{0} past the window", spanBetween(closed, reach))
                : t("{0} to spare before the window closes", spanBetween(reach, closed))
            }
          >
            <span className="absolute inset-x-0 top-2 h-1 rounded-full bg-dsk-sunken" />
            <span
              className={cn(
                "absolute top-2 h-1 rounded-full",
                late ? "bg-dsk-error" : "bg-dsk-warn",
              )}
              style={{
                left: `${trackShare(Math.min(closed, reach), start, end)}%`,
                width: `${Math.abs(trackShare(reach, start, end) - trackShare(closed, start, end))}%`,
              }}
            />
            <Mark at={closed} start={start} end={end} label={t("Window closed")} shape="bar" />
            {eta > 0 && <Mark at={eta} start={start} end={end} label={t("Arrival")} shape="ring" />}
            {arrived > 0 ? (
              <Mark at={arrived} start={start} end={end} label={t("Arrived")} shape="dot" />
            ) : (
              <Mark at={now} start={start} end={end} label={t("Now")} shape="dot" />
            )}
          </div>
        </PanelBody>
      )}
      <Figures>
        {closed > 0 && (
          <Figure
            label={t("Window closed")}
            value={formatFactTime(t, closed, now, timezone)}
            note={agoOrIn(t, closed, now)}
          />
        )}
        {eta > 0 && (
          <Figure
            label={t("Projected arrival")}
            value={formatFactTime(t, eta, now, timezone)}
            note={stop.stale ? t("From old information") : agoOrIn(t, eta, now)}
            tone={stop.stale ? "warn" : undefined}
          />
        )}
        {arrived > 0 && (
          <Figure label={t("Arrived")} value={formatFactTime(t, arrived, now, timezone)} />
        )}
        {closed > 0 && (
          <Figure
            label={late ? t("Late by") : t("To spare")}
            value={spanBetween(closed, reach)}
            tone={late ? "bad" : "warn"}
          />
        )}
        {stop.milesRemaining > 0 && (
          <Figure label={t("Miles to go")} value={stop.milesRemaining.toLocaleString()} />
        )}
        {stop.lastHeardAt > 0 && (
          <Figure label={t("Last heard")} value={agoOrIn(t, stop.lastHeardAt, now)} />
        )}
      </Figures>
    </PanelCard>
  );
}

function Mark({
  at,
  start,
  end,
  label,
  shape,
}: {
  at: number;
  start: number;
  end: number;
  label: string;
  shape: "bar" | "ring" | "dot";
}) {
  return (
    <span
      className="absolute top-0 flex -translate-x-1/2 flex-col items-center gap-1"
      style={{ left: `${trackShare(at, start, end)}%` }}
    >
      <span
        aria-hidden
        className={cn(
          "mt-0.5 block",
          shape === "bar" && "h-4 w-0.5 rounded-full bg-dsk-fg2",
          shape === "ring" && "mt-1 size-3 rounded-full border-2 border-dsk-fg bg-dsk-card",
          shape === "dot" && "mt-1 size-3 rounded-full bg-dsk-fg ring-2 ring-dsk-card",
        )}
      />
      <span className="text-2xs whitespace-nowrap text-dsk-subtle">{label}</span>
    </span>
  );
}
