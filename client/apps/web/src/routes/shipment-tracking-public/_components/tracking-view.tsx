import {
  dayAndTime,
  journeyOf,
  minutesOutsideWindow,
  nextStopOf,
  publicStatusAttrs,
  trackingHeadline,
  windowRange,
} from "@/lib/shipment-tracking";
import logo from "@/assets/logo.webp";
import { useCopyToClipboard } from "@/hooks/use-copy-to-clipboard";
import {
  CalendarPlus01Icon,
  EyeIcon,
  Link01Icon,
  RefreshCw01Icon,
} from "@trenova/shared/components/icons";
import { Badge } from "@trenova/shared/components/ui/badge";
import { Button } from "@trenova/shared/components/ui/button";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { useT } from "@trenova/shared/i18n/use-t";
import { downloadCalendarFile } from "@trenova/shared/lib/calendar-file";
import {
  formatDurationFromSeconds,
  formatSecondsAgo,
  formatUnixDateMedium,
} from "@trenova/shared/lib/date";
import { phaseTone } from "@trenova/shared/lib/status-phase";
import { stopTypeLabel } from "@trenova/shared/lib/stop-type";
import { cn } from "@trenova/shared/lib/utils";
import type {
  PublicShipmentStop,
  PublicShipmentView,
} from "@trenova/shared/types/shipment-tracking";
import { RouteRibbon } from "./route-ribbon";
import { StopTimeline } from "./stop-timeline";
import { TrackingAside } from "./tracking-aside";
import "../tracking.css";

/**
 * How the page is being read: by the customer through its link, live and
 * refreshing; or by staff previewing what a link sent now would show, which
 * has no link to copy, download through, or refresh.
 */
export type TrackingMode =
  | {
      kind: "link";
      token: string;
      updatedAt: number;
      refreshing: boolean;
      onRefresh: () => void;
    }
  | { kind: "preview"; audience: string };

export function TrackingTopBar({
  company,
  mode,
  now,
}: {
  company?: string;
  mode?: TrackingMode;
  now: number;
}) {
  const t = useT();

  return (
    <header className="relative mx-auto flex h-16 w-full max-w-6xl items-center justify-between gap-4 px-5 md:px-8">
      <div className="flex min-w-0 items-center gap-2.5">
        <img src={logo} alt="" className="size-6 object-contain" />
        {company ? (
          <span className="truncate text-sm font-semibold">{company}</span>
        ) : (
          <Skeleton className="h-4 w-32" />
        )}
      </div>
      {mode?.kind === "preview" && (
        <div className="flex min-w-0 items-center gap-2 text-xs text-muted-foreground">
          <Badge variant="info">
            <EyeIcon className="size-3" aria-hidden />
            {t("Preview")}
          </Badge>
          <span className="hidden truncate sm:inline">
            {t("What {0} would see through a link sent now", mode.audience)}
          </span>
        </div>
      )}
      {mode?.kind === "link" && (
        <div className="flex items-center gap-1 text-xs text-muted-foreground">
          <span className="relative flex size-2">
            <span className="st-halo absolute inset-0 animate-live-ping rounded-full" />
            <span className="st-dot relative size-2 rounded-full" />
          </span>
          <span className="ml-1.5 hidden tabular-nums sm:inline">
            {t("Live · updated {0}", formatSecondsAgo(now - mode.updatedAt))}
          </span>
          <Button
            type="button"
            variant="ghost"
            size="icon-sm"
            aria-label={t("Refresh")}
            onClick={mode.onRefresh}
            disabled={mode.refreshing}
          >
            <RefreshCw01Icon
              className={cn("size-3.5", mode.refreshing && "animate-spin")}
              aria-hidden
            />
          </Button>
        </div>
      )}
    </header>
  );
}

/** Puts the arrival in the reader's calendar, and copies this page's address to pass on. */
function ArrivalActions({
  view,
  stop,
  start,
}: {
  view: PublicShipmentView;
  stop: PublicShipmentStop;
  start: number;
}) {
  const t = useT();
  const { copy, isCopied } = useCopyToClipboard();
  const place = stop.locationName ?? stop.city;

  return (
    <div className="mt-2 flex flex-wrap gap-2">
      <Button
        type="button"
        variant="outline"
        size="sm"
        onClick={() =>
          downloadCalendarFile(
            {
              uid: `${view.shipmentProNumber}-${stop.sequence}@trenova`,
              title: t("{0} of load {1} at {2}", stopTypeLabel(stop.type), view.shipmentProNumber, place),
              start,
              end: stop.windowEnd ?? undefined,
              location: [stop.locationName, stop.addressLine, [stop.city, stop.stateCode].filter(Boolean).join(", ")]
                .filter(Boolean)
                .join(", "),
              description: t("Follow the load: {0}", window.location.href),
              url: window.location.href,
            },
            `${view.shipmentProNumber}-${stop.sequence}`,
          )
        }
      >
        <CalendarPlus01Icon className="size-3.5" aria-hidden />
        {t("Add to calendar")}
      </Button>
      <Button
        type="button"
        variant="ghost"
        size="sm"
        onClick={() => void copy(window.location.href, { withToast: false })}
      >
        <Link01Icon className="size-3.5" aria-hidden />
        {isCopied ? t("Link copied") : t("Copy link")}
      </Button>
    </div>
  );
}

function Arrival({
  view,
  now,
  withActions,
}: {
  view: PublicShipmentView;
  now: number;
  withActions: boolean;
}) {
  const t = useT();
  const final = view.stops.at(-1);
  const next = nextStopOf(view);

  if (view.status === "Canceled" || !final) {
    return null;
  }

  if (view.status === "Delivered" && final.arrivedAt != null) {
    const late = minutesOutsideWindow(final, final.arrivedAt);
    return (
      <ArrivalFigure
        label={t("Delivered to {0}", final.locationName ?? final.city)}
        value={dayAndTime(t, final.arrivedAt, now)}
        note={
          late == null ? undefined : late <= 0 ? (
            <Badge variant="success">{t("On time")}</Badge>
          ) : (
            <Badge variant="warning">
              {t("{0} after the window", formatDurationFromSeconds(late * 60))}
            </Badge>
          )
        }
      />
    );
  }

  if (view.status === "Planned") {
    const first = view.stops[0];
    const range = windowRange(t, first);
    if (first.windowStart == null) return null;
    return (
      <ArrivalFigure
        label={t("Pickup at {0}", first.locationName ?? first.city)}
        value={dayAndTime(t, first.windowStart, now)}
        note={
          range ? (
            <span className="text-sm text-muted-foreground">{t("Appointment {0}", range)}</span>
          ) : undefined
        }
        actions={
          withActions ? (
            <ArrivalActions view={view} stop={first} start={first.windowStart} />
          ) : undefined
        }
      />
    );
  }

  if (next?.estimatedArrival != null) {
    const outside = minutesOutsideWindow(next, next.estimatedArrival);
    const range = windowRange(t, next);
    const until = next.estimatedArrival - now;
    return (
      <ArrivalFigure
        label={t("Expected at {0}", next.locationName ?? next.city)}
        value={dayAndTime(t, next.estimatedArrival, now)}
        sub={until > 0 ? t("in {0}", formatDurationFromSeconds(until)) : undefined}
        note={
          <div className="flex flex-wrap items-center gap-2">
            {outside != null &&
              (outside > 0 ? (
                <Badge variant="warning">
                  {t("About {0} after the appointment", formatDurationFromSeconds(outside * 60))}
                </Badge>
              ) : (
                <Badge variant="success">
                  {outside < 0 ? t("Ahead of the appointment") : t("Within the appointment")}
                </Badge>
              ))}
            {range && (
              <span className="text-sm text-muted-foreground">{t("Appointment {0}", range)}</span>
            )}
          </div>
        }
        actions={
          withActions ? (
            <ArrivalActions view={view} stop={next} start={next.estimatedArrival} />
          ) : undefined
        }
      />
    );
  }

  if (next?.windowStart != null) {
    return (
      <ArrivalFigure
        label={t("Appointment at {0}", next.locationName ?? next.city)}
        value={dayAndTime(t, next.windowStart, now)}
      />
    );
  }

  return null;
}

function ArrivalFigure({
  label,
  value,
  sub,
  note,
  actions,
}: {
  label: string;
  value: string;
  sub?: string;
  note?: React.ReactNode;
  actions?: React.ReactNode;
}) {
  return (
    <div className="flex flex-col gap-2">
      <span className="text-sm text-muted-foreground">{label}</span>
      <div className="flex flex-wrap items-baseline gap-x-3 gap-y-1">
        <span className="text-3xl font-semibold tracking-tight tabular-nums">{value}</span>
        {sub && <span className="text-sm text-muted-foreground tabular-nums">{sub}</span>}
      </div>
      {note}
      {actions}
    </div>
  );
}

/** The whole tracking page for one view, as the customer reads it or as staff preview it. */
export function TrackingView({
  view,
  mode,
  now,
}: {
  view: PublicShipmentView;
  mode: TrackingMode;
  now: number;
}) {
  const t = useT();
  const status = view.status ? publicStatusAttrs(t)[view.status] : null;
  const journey = journeyOf(view, now);
  const kicker = [t("Load {0}", view.shipmentProNumber), view.customerReference]
    .filter(Boolean)
    .join("  ·  ");

  return (
    <div className="st-page relative min-h-full" data-tone={status?.phase ?? "queued"}>
      <div className="st-glow" aria-hidden />
      <TrackingTopBar company={view.companyName} mode={mode} now={now} />

      <main className="relative mx-auto flex w-full max-w-6xl flex-col px-5 md:px-8">
        <section className="st-in grid gap-10 pt-10 pb-12 md:pt-16 lg:grid-cols-[minmax(0,1fr)_minmax(0,400px)] lg:items-end lg:gap-16">
          <div className="flex flex-col gap-6">
            <div className="flex flex-wrap items-center gap-3">
              {status && (
                <Badge variant={phaseTone(status.phase)} title={status.description}>
                  {status.text}
                </Badge>
              )}
              <span className="font-mono text-xs text-muted-foreground">{kicker}</span>
            </div>
            <h1 className="max-w-3xl font-newsreader text-4xl leading-[1.04] font-normal tracking-[-0.024em] text-balance md:text-7xl">
              {trackingHeadline(t, view)}
            </h1>
          </div>
          <div className="lg:border-l lg:border-border-subtle lg:pl-10">
            <Arrival view={view} now={now} withActions={mode.kind === "link"} />
          </div>
        </section>

        {view.stops.length > 1 && (
          <section className="st-in border-y border-border-subtle py-8" style={{ animationDelay: "120ms" }}>
            <RouteRibbon view={view} journey={journey} now={now} />
          </section>
        )}

        <div
          className="st-in grid gap-12 py-12 lg:grid-cols-[minmax(0,1fr)_360px] lg:gap-16"
          style={{ animationDelay: "220ms" }}
        >
          <section className="flex flex-col gap-5">
            <h2 className="text-sm font-semibold">{t("Stops")}</h2>
            <StopTimeline view={view} journey={journey} now={now} />
          </section>
          <TrackingAside
            view={view}
            token={mode.kind === "link" ? mode.token : null}
            now={now}
          />
        </div>
      </main>

      <footer className="relative mx-auto flex w-full max-w-6xl flex-col gap-1 border-t border-border-subtle px-5 py-8 text-xs text-muted-foreground md:flex-row md:items-center md:justify-between md:px-8">
        <span>
          {mode.kind === "link"
            ? t(
                "Questions about this shipment? Reply to the email this link came in. The link works until {0}.",
                formatUnixDateMedium(view.expiresAt),
              )
            : t("A link sent now would work until {0}.", formatUnixDateMedium(view.expiresAt))}
        </span>
        <span>{t("Tracking by Trenova")}</span>
      </footer>
    </div>
  );
}


