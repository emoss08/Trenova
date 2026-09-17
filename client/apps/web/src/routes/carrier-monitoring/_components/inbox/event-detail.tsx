import { RelativeTime } from "@/components/carrier-intelligence/relative-time";
import { RiskLabel, SeverityLabel } from "@/components/carrier-intelligence/status-dot";
import type { CarrierIntelLabels } from "@/components/carrier-intelligence/use-carrier-intel-labels";
import {
  canAcknowledgeEvent,
  canResolveEvent,
  carrierIntelProviderLabel,
} from "@/lib/carrier-intelligence";
import { carrierPanelPath } from "@/lib/carrier-links";
import {
  CARRIER_INTELLIGENCE_KEY,
  type CarrierIntelEvent,
} from "@/lib/graphql/carrier-intelligence";
import { fetchCarrierIntelEventCarrierSummary } from "@/lib/graphql/carrier-monitoring-table";
import { useQuery } from "@tanstack/react-query";
import { Button } from "@trenova/shared/components/ui/button";
import { Kbd } from "@trenova/shared/components/ui/kbd";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { useT } from "@trenova/shared/i18n/use-t";
import { formatUnixDateTimeMedium } from "@trenova/shared/lib/date";
import { cn } from "@trenova/shared/lib/utils";
import { ArrowRightIcon, CheckCheckIcon, CheckIcon, ExternalLinkIcon } from "lucide-react";
import type { ReactNode } from "react";
import { Link } from "react-router";
import type { EventPresentation } from "@/components/carrier-intelligence/use-event-presenter";

export type EventDetailProps = {
  event: CarrierIntelEvent;
  presentation: EventPresentation;
  labels: CarrierIntelLabels;
  ruleDescription: string | null;
  canUpdate: boolean;
  acknowledging: boolean;
  onAcknowledge: (event: CarrierIntelEvent) => void;
  onResolve: (event: CarrierIntelEvent) => void;
  className?: string;
};

function DetailRow({ label, children }: { label: string; children: ReactNode }) {
  return (
    <>
      <dt className="text-muted-foreground text-xs">{label}</dt>
      <dd className="min-w-0 text-sm">{children}</dd>
    </>
  );
}

function Section({ title, children }: { title: string; children: ReactNode }) {
  return (
    <section className="flex flex-col gap-3 border-b border-border/60 px-5 py-4 last:border-b-0">
      <h3 className="text-muted-foreground text-xs font-medium">{title}</h3>
      {children}
    </section>
  );
}

function ActorName({
  userId,
  user,
}: {
  userId: string | null;
  user: CarrierIntelEvent["acknowledgedBy"];
}) {
  const t = useT();
  if (user?.name) {
    return <span className="text-foreground">{user.name}</span>;
  }
  return <span>{userId ? t("a teammate") : t("the system")}</span>;
}

type TimelineStep = {
  id: string;
  tone: "done" | "current";
  title: ReactNode;
  timestamp: number;
  note?: string | null;
};

function StatusTimeline({
  event,
  labels,
}: {
  event: CarrierIntelEvent;
  labels: CarrierIntelLabels;
}) {
  const t = useT();
  const steps: TimelineStep[] = [
    { id: "detected", tone: "done", title: t("Detected"), timestamp: event.detectedAt },
  ];
  if (event.acknowledgedAt) {
    steps.push({
      id: "acknowledged",
      tone: "done",
      title: (
        <>
          {t("Acknowledged by")}{" "}
          <ActorName userId={event.acknowledgedById} user={event.acknowledgedBy} />
        </>
      ),
      timestamp: event.acknowledgedAt,
    });
  }
  if (event.resolvedAt) {
    steps.push({
      id: "resolved",
      tone: "done",
      title: (
        <>
          {event.status === "Dismissed" ? t("Dismissed by") : t("Resolved by")}{" "}
          <ActorName userId={event.resolvedById} user={event.resolvedBy} />
          {event.resolution ? (
            <span className="text-muted-foreground"> · {labels.resolution[event.resolution]}</span>
          ) : null}
        </>
      ),
      timestamp: event.resolvedAt,
      note: event.resolutionNote,
    });
  }

  return (
    <ol className="relative flex flex-col gap-4">
      {steps.map((step, index) => (
        <li key={step.id} className="relative flex gap-3">
          {index < steps.length - 1 ? (
            <span aria-hidden className="bg-border absolute top-4 bottom-[-1rem] left-[3px] w-px" />
          ) : null}
          <span
            aria-hidden
            className={cn(
              "relative mt-1.5 size-[7px] shrink-0 rounded-full",
              index === steps.length - 1 ? "bg-foreground" : "bg-muted-foreground/50",
            )}
          />
          <div className="flex min-w-0 flex-1 flex-col gap-1">
            <div className="flex flex-wrap items-baseline justify-between gap-x-3 text-sm">
              <span className="min-w-0">{step.title}</span>
              <RelativeTime timestamp={step.timestamp} className="text-muted-foreground text-xs" />
            </div>
            {step.note ? (
              <p className="text-muted-foreground rounded-md bg-muted/50 px-3 py-2 text-xs whitespace-pre-line">
                {step.note}
              </p>
            ) : null}
          </div>
        </li>
      ))}
    </ol>
  );
}

function CarrierStanding({ carrierId }: { carrierId: string }) {
  const t = useT();
  const summaryQuery = useQuery({
    queryKey: [CARRIER_INTELLIGENCE_KEY, "event-carrier-summary", carrierId],
    queryFn: ({ signal }) => fetchCarrierIntelEventCarrierSummary(carrierId, { signal }),
    staleTime: 30_000,
  });

  if (summaryQuery.isPending) {
    return <Skeleton className="h-4 w-48" />;
  }
  if (summaryQuery.isError || !summaryQuery.data) {
    return (
      <p className="text-muted-foreground text-xs">{t("Current standing is not available.")}</p>
    );
  }

  const carrier = summaryQuery.data;
  return (
    <div className="text-muted-foreground flex flex-wrap items-center gap-x-3 gap-y-1 text-xs">
      <RiskLabel level={carrier.intelRiskLevel} />
      <span aria-hidden>·</span>
      <span className={cn("tabular-nums", carrier.intelBlockingCount > 0 && "text-foreground")}>
        {t(
          "{0, plural, =0 {No blockers} one {# blocker} other {# blockers}}",
          carrier.intelBlockingCount,
        )}
      </span>
      <span aria-hidden>·</span>
      <span className="tabular-nums">
        {t(
          "{0, plural, =0 {No open changes} one {# open change} other {# open changes}}",
          carrier.openIntelEventCount,
        )}
      </span>
      {carrier.intelReviewRequired ? (
        <>
          <span aria-hidden>·</span>
          <span className="text-foreground">{t("Review required")}</span>
        </>
      ) : null}
    </div>
  );
}

export function EventDetail({
  event,
  presentation,
  labels,
  ruleDescription,
  canUpdate,
  acknowledging,
  onAcknowledge,
  onResolve,
  className,
}: EventDetailProps) {
  const t = useT();
  const carrierName = event.subjectName || t("USDOT {0}", event.dotNumber);
  const showSummary =
    Boolean(event.ruleCode) && event.summary && event.summary !== presentation.title;

  return (
    <article className={cn("flex flex-col", className)} aria-label={presentation.title}>
      <header className="flex flex-col gap-3 border-b border-border/60 px-5 py-4">
        <div className="text-muted-foreground flex items-center gap-2 text-xs">
          <SeverityLabel severity={event.severity} label={labels.severity[event.severity]} />
          <span aria-hidden>·</span>
          <span>{labels.eventStatus[event.status]}</span>
        </div>
        <h2 className="text-base leading-snug font-semibold text-balance">{presentation.title}</h2>
        <div className="text-muted-foreground flex flex-wrap items-center gap-x-1.5 text-sm">
          {event.carrierId ? (
            <Link
              to={carrierPanelPath(event.carrierId, "intelligence")}
              className="text-foreground font-medium hover:underline"
            >
              {carrierName}
            </Link>
          ) : (
            <span className="text-foreground font-medium">{carrierName}</span>
          )}
          {event.subjectName ? (
            <span className="text-xs tabular-nums">· {t("USDOT {0}", event.dotNumber)}</span>
          ) : null}
        </div>
        <div className="flex flex-wrap items-center gap-2">
          {canUpdate && canAcknowledgeEvent(event.status) ? (
            <Button
              type="button"
              variant="outline"
              className="h-8 text-xs"
              isLoading={acknowledging}
              onClick={() => onAcknowledge(event)}
            >
              <CheckIcon className="size-3.5" />
              {t("Acknowledge")}
              <Kbd>E</Kbd>
            </Button>
          ) : null}
          {canUpdate && canResolveEvent(event.status) ? (
            <Button
              type="button"
              variant="outline"
              className="h-8 text-xs"
              onClick={() => onResolve(event)}
            >
              <CheckCheckIcon className="size-3.5" />
              {t("Resolve")}
            </Button>
          ) : null}
          {event.carrierId ? (
            <Button
              variant="ghost"
              className="h-8 text-xs"
              nativeButton={false}
              render={<Link to={carrierPanelPath(event.carrierId, "intelligence")} />}
            >
              <ExternalLinkIcon className="size-3.5" />
              {t("Open carrier")}
            </Button>
          ) : null}
        </div>
      </header>

      {presentation.change ? (
        <Section title={presentation.change.label}>
          <div className="flex flex-wrap items-center gap-2 text-sm">
            <span className="text-muted-foreground">{presentation.change.before}</span>
            <ArrowRightIcon className="text-muted-foreground size-3.5" aria-hidden />
            <span className="font-medium">{presentation.change.after}</span>
          </div>
        </Section>
      ) : null}

      {event.ruleCode && (ruleDescription || showSummary) ? (
        <Section title={t("Rule")}>
          {showSummary ? <p className="text-sm">{event.summary}</p> : null}
          {ruleDescription ? (
            <p className="text-muted-foreground text-xs leading-relaxed">{ruleDescription}</p>
          ) : null}
        </Section>
      ) : null}

      <Section title={t("Details")}>
        <dl className="grid grid-cols-[7.5rem_minmax(0,1fr)] items-baseline gap-x-4 gap-y-2">
          <DetailRow label={t("Detected")}>
            <span>{formatUnixDateTimeMedium(event.detectedAt)}</span>
          </DetailRow>
          {event.vendorChangedAt ? (
            <DetailRow label={t("Changed at provider")}>
              <span>{formatUnixDateTimeMedium(event.vendorChangedAt)}</span>
            </DetailRow>
          ) : null}
          <DetailRow label={t("Category")}>{labels.section[event.category]}</DetailRow>
          <DetailRow label={t("Source")}>{labels.eventSource[event.source]}</DetailRow>
          <DetailRow label={t("Provider")}>{carrierIntelProviderLabel(event.provider)}</DetailRow>
        </dl>
      </Section>

      {event.carrierId ? (
        <Section title={t("Carrier standing")}>
          <CarrierStanding carrierId={event.carrierId} />
        </Section>
      ) : null}

      <Section title={t("Activity")}>
        <StatusTimeline event={event} labels={labels} />
      </Section>
    </article>
  );
}
