import { Metadata } from "@/components/metadata";
import { publicLinkErrorKind } from "@/components/public-page/error-kind";
import {
  dayAndTime,
  journeyOf,
  minutesOutsideWindow,
  nextStopOf,
  publicStatusAttrs,
  trackingHeadline,
  windowRange,
} from "@/lib/shipment-tracking";
import { ShipmentTrackingService } from "@/services/shipment-tracking";
import logo from "@/assets/logo.webp";
import { useCopyToClipboard } from "@/hooks/use-copy-to-clipboard";
import { useNowSeconds } from "@/hooks/use-now-seconds";
import {
  AlertTriangleIcon,
  CalendarPlus01Icon,
  ClockIcon,
  Link01Icon,
  RefreshCw01Icon,
  SlashCircle01Icon,
} from "@trenova/shared/components/icons";
import { downloadCalendarFile } from "@trenova/shared/lib/calendar-file";
import { Badge } from "@trenova/shared/components/ui/badge";
import { Button } from "@trenova/shared/components/ui/button";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { useT } from "@trenova/shared/i18n/use-t";
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
import { useQuery } from "@tanstack/react-query";
import { useParams } from "react-router";
import { RouteRibbon } from "./_components/route-ribbon";
import { StopTimeline } from "./_components/stop-timeline";
import { TrackingAside } from "./_components/tracking-aside";
import "./tracking.css";

const shipmentTrackingService = new ShipmentTrackingService();
const REFRESH_MS = 2 * 60 * 1000;

function TopBar({
  company,
  updatedAt,
  refreshing,
  onRefresh,
  now,
}: {
  company?: string;
  updatedAt?: number;
  refreshing?: boolean;
  onRefresh?: () => void;
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
      {updatedAt != null && onRefresh && (
        <div className="flex items-center gap-1 text-xs text-muted-foreground">
          <span className="relative flex size-2">
            <span className="st-halo absolute inset-0 animate-live-ping rounded-full" />
            <span className="st-dot relative size-2 rounded-full" />
          </span>
          <span className="ml-1.5 hidden tabular-nums sm:inline">
            {t("Live · updated {0}", formatSecondsAgo(now - updatedAt))}
          </span>
          <Button
            type="button"
            variant="ghost"
            size="icon-sm"
            aria-label={t("Refresh")}
            onClick={onRefresh}
            disabled={refreshing}
          >
            <RefreshCw01Icon className={cn("size-3.5", refreshing && "animate-spin")} aria-hidden />
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

function Arrival({ view, now }: { view: PublicShipmentView; now: number }) {
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
        actions={<ArrivalActions view={view} stop={first} start={first.windowStart} />}
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
        actions={<ArrivalActions view={view} stop={next} start={next.estimatedArrival} />}
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

function Tracking({
  view,
  token,
  now,
  updatedAt,
  refreshing,
  onRefresh,
}: {
  view: PublicShipmentView;
  token: string;
  now: number;
  updatedAt: number;
  refreshing: boolean;
  onRefresh: () => void;
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
      <TopBar
        company={view.companyName}
        updatedAt={updatedAt}
        refreshing={refreshing}
        onRefresh={onRefresh}
        now={now}
      />

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
            <Arrival view={view} now={now} />
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
          <TrackingAside view={view} token={token} now={now} />
        </div>
      </main>

      <footer className="relative mx-auto flex w-full max-w-6xl flex-col gap-1 border-t border-border-subtle px-5 py-8 text-xs text-muted-foreground md:flex-row md:items-center md:justify-between md:px-8">
        <span>
          {t(
            "Questions about this shipment? Reply to the email this link came in. The link works until {0}.",
            formatUnixDateMedium(view.expiresAt),
          )}
        </span>
        <span>{t("Tracking by Trenova")}</span>
      </footer>
    </div>
  );
}

function TrackingSkeleton({ now }: { now: number }) {
  return (
    <div className="st-page relative min-h-full" data-tone="queued">
      <TopBar now={now} />
      <main className="mx-auto flex w-full max-w-6xl flex-col gap-8 px-5 pt-16 md:px-8">
        <Skeleton className="h-5 w-48" />
        <Skeleton className="h-14 w-full max-w-xl" />
        <Skeleton className="h-9 w-64" />
        <Skeleton className="mt-6 h-36 w-full" />
        <div className="grid gap-12 lg:grid-cols-[minmax(0,1fr)_360px]">
          <Skeleton className="h-64" />
          <Skeleton className="h-64" />
        </div>
      </main>
    </div>
  );
}

function Message({
  icon,
  title,
  body,
  action,
}: {
  icon: React.ReactNode;
  title: string;
  body: string;
  action?: React.ReactNode;
}) {
  return (
    <div className="st-page relative flex min-h-full flex-col" data-tone="queued">
      <TopBar now={0} />
      <main className="st-in mx-auto flex w-full max-w-xl flex-1 flex-col items-start justify-center gap-4 px-5 pb-32">
        <span className="flex size-11 items-center justify-center rounded-full bg-background text-muted-foreground ring-1 ring-foreground/10">
          {icon}
        </span>
        <h1 className="font-newsreader text-4xl leading-tight tracking-[-0.02em]">{title}</h1>
        <p className="text-sm text-muted-foreground">{body}</p>
        {action}
      </main>
    </div>
  );
}

/**
 * The page a customer opens from a tracking link. It is read-only and shows
 * only what the customer's settings allow; the server decides that, so the
 * page draws whatever parts it was given.
 */
export function ShipmentTrackingPublicPage() {
  const t = useT();
  const { token = "" } = useParams();
  const now = useNowSeconds(30_000);

  const viewQuery = useQuery({
    queryKey: ["public-shipment-tracking", token] as const,
    queryFn: () => shipmentTrackingService.getPublicView(token),
    retry: false,
    refetchInterval: REFRESH_MS,
  });

  let content: React.ReactNode;
  if (viewQuery.isLoading) {
    content = <TrackingSkeleton now={now} />;
  } else if (viewQuery.isError) {
    const kind = publicLinkErrorKind(viewQuery.error);
    content =
      kind === "throttled" ? (
        <Message
          icon={<ClockIcon className="size-5" aria-hidden />}
          title={t("Too many attempts")}
          body={t("Please wait a minute and open the link again.")}
        />
      ) : kind === "unavailable" ? (
        <Message
          icon={<AlertTriangleIcon className="size-5" aria-hidden />}
          title={t("Temporarily unavailable")}
          body={t(
            "The shipment could not be loaded because of a temporary problem. Please try again in a moment.",
          )}
          action={
            <Button
              type="button"
              variant="outline"
              size="sm"
              isLoading={viewQuery.isRefetching}
              loadingText={t("Retrying...")}
              onClick={() => void viewQuery.refetch()}
            >
              {t("Try again")}
            </Button>
          }
        />
      ) : (
        <InvalidLink />
      );
  } else if (viewQuery.data) {
    content = (
      <Tracking
        view={viewQuery.data}
        token={token}
        now={now}
        updatedAt={Math.floor(viewQuery.dataUpdatedAt / 1000)}
        refreshing={viewQuery.isFetching}
        onRefresh={() => void viewQuery.refetch()}
      />
    );
  } else {
    content = <InvalidLink />;
  }

  return (
    <>
      <Metadata title={t("Shipment tracking")} description={t("Follow a shipment")} />
      <div className="fixed inset-0 h-svh w-full overflow-y-auto bg-canvas text-foreground">
        {content}
      </div>
    </>
  );
}

function InvalidLink() {
  const t = useT();
  return (
    <Message
      icon={<SlashCircle01Icon className="size-5" aria-hidden />}
      title={t("This tracking link is no longer valid")}
      body={t(
        "The link may have expired or been turned off. Ask the company that sent it for a new one.",
      )}
    />
  );
}
