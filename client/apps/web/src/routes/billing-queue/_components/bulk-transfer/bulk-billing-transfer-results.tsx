import { KpiStrip, KpiStripItem } from "@/components/kpi/kpi-strip";
import type {
  BillingTransferFailureCode,
  BillingTransferRun,
  BillingTransferRunItem,
} from "@/lib/graphql/billing-transfer";
import { shipmentPanelPath } from "@/lib/shipment-utils";
import { useInfiniteQuery } from "@tanstack/react-query";
import {
  AlertCircleIcon,
  InfoCircleIcon,
  LinkExternal01Icon,
} from "@trenova/shared/components/icons";
import { PlainBillingQueueStatusBadge } from "@trenova/shared/components/status-badge";
import { Alert, AlertDescription, AlertTitle } from "@trenova/shared/components/ui/alert";
import { Badge } from "@trenova/shared/components/ui/badge";
import { Button } from "@trenova/shared/components/ui/button";
import { ScrollArea } from "@trenova/shared/components/ui/scroll-area";
import { SegmentedControl } from "@trenova/shared/components/ui/segmented-control";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { useCanReadKind } from "@trenova/shared/hooks/use-can-read-kind";
import { formatDuration, formatNumber } from "@trenova/shared/i18n/format";
import { useT, type TranslateFn } from "@trenova/shared/i18n/use-t";
import { cn } from "@trenova/shared/lib/utils";
import { m, type Variants } from "motion/react";
import { useMemo, useState } from "react";
import { billingTransferRunItemsQuery } from "../../billing-queue-queries";
import { BILLING_TRANSFER_FAILURE_REASONS } from "./bulk-billing-transfer-report";
import { runElapsedSeconds, runOutcome, type RunOutcome } from "./bulk-billing-transfer-run";

type ResultView = "notTransferred" | "transferred" | "notProcessed";

type Selection = { view: ResultView; reason: BillingTransferFailureCode | null };

const ALL_REASONS = "all";

const VIEW_STATUSES: Record<ResultView, BillingTransferRunItem["status"][]> = {
  notTransferred: ["NotTransferred"],
  transferred: ["Transferred"],
  notProcessed: ["Skipped", "Pending"],
};

function defaultSelection(run: BillingTransferRun): Selection {
  if (run.notTransferredCount > 0) return { view: "notTransferred", reason: null };
  if (run.transferredCount > 0) return { view: "transferred", reason: null };
  return { view: "notProcessed", reason: null };
}

const RISE: Variants = {
  hidden: { opacity: 0, y: 10 },
  shown: { opacity: 1, y: 0, transition: { duration: 0.52, ease: [0.16, 1, 0.3, 1] } },
};

const STAGGER: Variants = {
  hidden: {},
  shown: { transition: { staggerChildren: 0.08, delayChildren: 0.12 } },
};

/**
 * A finished run's report: what it amounted to, the figures behind it, then the shipments
 * themselves. Each figure is also the filter for the list under it, and it opens on the
 * shipments that did not move, which can be narrowed to one reason at a time: each reason is
 * a piece of work left to do. Arriving from the transfer stage (`animateIn`), its parts rise
 * in one after another; opened on a run that had already finished, it is simply there.
 */
export function BulkBillingTransferResults({
  run,
  animateIn = false,
}: {
  run: BillingTransferRun;
  animateIn?: boolean;
}) {
  const t = useT();
  const [preferred, setPreferred] = useState<Selection | null>(null);
  const selection = preferred ?? defaultSelection(run);
  const select = (view: ResultView) => setPreferred({ view, reason: null });

  return (
    <m.div
      className="flex min-h-0 flex-col gap-4"
      variants={STAGGER}
      initial={animateIn ? "hidden" : false}
      animate="shown"
    >
      <m.div variants={RISE}>
        <OutcomeHeader run={run} />
      </m.div>

      {run.status === "Failed" ? (
        <m.div variants={RISE}>
          <Alert variant="destructive" size="sm">
            <AlertCircleIcon />
            <AlertTitle>{t("The transfer stopped because of an error")}</AlertTitle>
            <AlertDescription>
              {run.failureMessage ??
                t("Something unexpected stopped the run. Retry it to continue.")}
            </AlertDescription>
          </Alert>
        </m.div>
      ) : null}
      {run.unmatchedCount > 0 ? (
        <m.div variants={RISE}>
          <Alert variant="info" size="sm">
            <InfoCircleIcon />
            <AlertDescription>
              {t(
                "{1} more {0, plural, one {shipment} other {shipments}} matched than one run transfers. Run Transfer all again for the rest.",
                run.unmatchedCount,
                formatNumber(run.unmatchedCount),
              )}
            </AlertDescription>
          </Alert>
        </m.div>
      ) : null}

      {run.totalCount > 0 ? (
        <>
          <m.div variants={RISE}>
            <KpiStrip aria-label={t("Transfer outcome")} minItemWidth="8rem">
              <KpiStripItem
                label={t("Transferred")}
                tone="success"
                value={formatNumber(run.transferredCount)}
                active={selection.view === "transferred"}
                onClick={() => select("transferred")}
              />
              <KpiStripItem
                label={t("Needs attention")}
                tone="warning"
                value={formatNumber(run.notTransferredCount)}
                active={selection.view === "notTransferred"}
                onClick={() => select("notTransferred")}
              />
              {run.skippedCount > 0 ? (
                <KpiStripItem
                  label={t("Not processed")}
                  tone="muted"
                  value={formatNumber(run.skippedCount)}
                  active={selection.view === "notProcessed"}
                  onClick={() => select("notProcessed")}
                />
              ) : null}
            </KpiStrip>
          </m.div>
          <m.div variants={RISE}>
            <ReportList run={run} selection={selection} onSelect={setPreferred} />
          </m.div>
        </>
      ) : null}
    </m.div>
  );
}

function OutcomeHeader({ run }: { run: BillingTransferRun }) {
  const t = useT();
  const outcome = runOutcome(run);
  const elapsed = runElapsedSeconds(run, run.completedAt ?? 0);
  const facts = outcomeFacts(run, outcome, t);
  if (elapsed !== null && elapsed > 0) {
    facts.push(t("Took {0}", formatDuration(elapsed)));
  }

  return (
    <header className="flex flex-col gap-1" role="status">
      <h3 className="text-base leading-tight font-semibold">{outcomeHeadline(run, outcome, t)}</h3>
      {facts.length > 0 ? (
        <ul className="text-muted-foreground flex flex-wrap items-center gap-x-1.5 gap-y-0.5 text-xs">
          {facts.map((fact, index) => (
            <li key={fact} className="flex items-center gap-1.5">
              {index > 0 ? <span aria-hidden>·</span> : null}
              {fact}
            </li>
          ))}
        </ul>
      ) : null}
    </header>
  );
}

function outcomeHeadline(run: BillingTransferRun, outcome: RunOutcome, t: TranslateFn): string {
  switch (outcome) {
    case "transferred":
      return t(
        "{0, plural, one {# shipment moved to billing} other {# shipments moved to billing}}",
        run.transferredCount,
      );
    case "partial":
      return t(
        "{0} of {1} shipments moved to billing",
        formatNumber(run.transferredCount),
        formatNumber(run.totalCount),
      );
    case "blocked":
      return t("No shipments moved to billing");
    case "canceled":
      return t("Transfer stopped");
    case "failed":
      return t("The transfer stopped early");
    default:
      return t("Nothing to transfer");
  }
}

function outcomeFacts(run: BillingTransferRun, outcome: RunOutcome, t: TranslateFn): string[] {
  const facts: string[] = [];
  if (outcome === "empty") {
    facts.push(t("No shipments matched, so nothing was transferred."));
    return facts;
  }
  if (outcome === "canceled" || outcome === "failed") {
    facts.push(
      t("{0} of {1} checked", formatNumber(run.processedCount), formatNumber(run.totalCount)),
    );
  }
  if (run.notTransferredCount > 0) {
    facts.push(
      t("{0, plural, one {# needs attention} other {# need attention}}", run.notTransferredCount),
    );
  }
  if (run.markedReadyToInvoiceCount > 0) {
    facts.push(
      t(
        "{0, plural, one {# marked Ready to Invoice first} other {# marked Ready to Invoice first}}",
        run.markedReadyToInvoiceCount,
      ),
    );
  }

  return facts;
}

function ReportList({
  run,
  selection,
  onSelect,
}: {
  run: BillingTransferRun;
  selection: Selection;
  onSelect: (selection: Selection) => void;
}) {
  const t = useT();
  const itemsQuery = useInfiniteQuery(
    billingTransferRunItemsQuery(
      run.id,
      VIEW_STATUSES[selection.view],
      selection.reason ? [selection.reason] : undefined,
    ),
  );
  const items = useMemo<BillingTransferRunItem[]>(
    () => itemsQuery.data?.pages.flatMap((page) => page.edges.map((edge) => edge.node)) ?? [],
    [itemsQuery.data],
  );
  const title = sectionTitle(selection, t);
  const filterable = selection.view === "notTransferred" && run.failureCounts.length > 1;

  return (
    <section
      aria-label={title}
      className="bg-card flex min-w-0 flex-col overflow-hidden rounded-lg border"
    >
      <header className="flex flex-col gap-2 border-b px-3 py-2">
        <div className="flex flex-col gap-0.5">
          <h4 className="truncate text-sm font-semibold">{title}</h4>
          <SectionNote run={run} selection={selection} />
        </div>
        {filterable ? (
          <div className="-mx-0.5 overflow-x-auto px-0.5">
            <SegmentedControl
              aria-label={t("Reasons")}
              value={selection.reason ?? ALL_REASONS}
              onValueChange={(value) =>
                onSelect({
                  view: "notTransferred",
                  reason: value === ALL_REASONS ? null : (value as BillingTransferFailureCode),
                })
              }
              items={[
                {
                  value: ALL_REASONS,
                  label: t("All reasons"),
                  caption: formatNumber(run.notTransferredCount),
                },
                ...run.failureCounts.map((reason) => ({
                  value: reason.code as string,
                  label: t(BILLING_TRANSFER_FAILURE_REASONS[reason.code].label),
                  caption: formatNumber(reason.count),
                })),
              ]}
            />
          </div>
        ) : null}
      </header>
      <ScrollArea className="h-[min(18rem,36vh)]">
        {itemsQuery.isPending ? (
          <ReportRowsSkeleton />
        ) : itemsQuery.isError ? (
          <div className="flex flex-col items-center gap-2 px-3 py-8 text-center">
            <p className="text-muted-foreground text-sm">
              {t("The transfer report could not be loaded.")}
            </p>
            <Button variant="outline" size="xs" onClick={() => void itemsQuery.refetch()}>
              {t("Try again")}
            </Button>
          </div>
        ) : items.length === 0 ? (
          <p className="text-muted-foreground px-3 py-10 text-center text-sm">
            {emptyLabel(selection.view, t)}
          </p>
        ) : (
          <>
            <ul aria-label={title} className="divide-y">
              {items.map((item) =>
                selection.view === "notTransferred" ? (
                  <FailureItem key={item.id} item={item} showReason={selection.reason === null} />
                ) : selection.view === "transferred" ? (
                  <TransferredItem key={item.id} item={item} />
                ) : (
                  <NotProcessedItem key={item.id} item={item} />
                ),
              )}
            </ul>
            {itemsQuery.hasNextPage ? (
              <div className="flex justify-center py-2">
                <Button
                  variant="ghost"
                  size="xs"
                  disabled={itemsQuery.isFetchingNextPage}
                  onClick={() => void itemsQuery.fetchNextPage()}
                >
                  {itemsQuery.isFetchingNextPage ? t("Loading...") : t("Load more")}
                </Button>
              </div>
            ) : null}
          </>
        )}
      </ScrollArea>
    </section>
  );
}

function SectionNote({ run, selection }: { run: BillingTransferRun; selection: Selection }) {
  const t = useT();

  if (selection.reason) {
    const reason = run.failureCounts.find((entry) => entry.code === selection.reason);
    return (
      <p className="text-muted-foreground text-xs">
        {t(BILLING_TRANSFER_FAILURE_REASONS[selection.reason].description)}{" "}
        {reason && !reason.retryable ? t("Retrying won't move these.") : null}
      </p>
    );
  }
  if (selection.view === "notTransferred" && run.retryableCount > 0) {
    return (
      <p className="text-muted-foreground text-xs">
        {t(
          "{0, plural, one {# of these can be retried once its reason is dealt with.} other {# of these can be retried once their reasons are dealt with.}}",
          run.retryableCount,
        )}
      </p>
    );
  }
  if (selection.view === "notProcessed") {
    return (
      <p className="text-muted-foreground text-xs">
        {t("The run ended before it reached these. Retry sends them again.")}
      </p>
    );
  }

  return null;
}

function sectionTitle(selection: Selection, t: TranslateFn): string {
  if (selection.reason) return t(BILLING_TRANSFER_FAILURE_REASONS[selection.reason].label);
  switch (selection.view) {
    case "notTransferred":
      return t("Needs attention");
    case "transferred":
      return t("Transferred");
    default:
      return t("Not processed");
  }
}

function emptyLabel(view: ResultView, t: TranslateFn): string {
  switch (view) {
    case "notTransferred":
      return t("Every shipment transferred.");
    case "transferred":
      return t("No shipment transferred.");
    default:
      return t("Every shipment was processed.");
  }
}

const SKELETON_ROWS = ["w-24", "w-20", "w-28", "w-16", "w-24"] as const;

function ReportRowsSkeleton() {
  const t = useT();

  return (
    <ul className="divide-y" aria-busy aria-label={t("Loading the transfer report")}>
      {SKELETON_ROWS.map((width, index) => (
        <li key={index} className="flex flex-col gap-1.5 px-3 py-2.5" aria-hidden>
          <div className="flex items-center justify-between gap-3">
            <Skeleton className={cn("h-3", width)} />
            <Skeleton className="h-3 w-28" />
          </div>
          <Skeleton className="h-2.5 w-48" />
        </li>
      ))}
    </ul>
  );
}

function ShipmentLink({ shipmentId, proNumber }: { shipmentId: string; proNumber: string | null }) {
  const t = useT();
  const canReadShipment = useCanReadKind("shipment");
  const display = proNumber ?? shipmentId;

  return (
    <div className="flex min-w-0 items-center gap-1.5">
      <span className="truncate font-mono text-xs font-medium">{display}</span>
      {canReadShipment ? (
        <a
          href={shipmentPanelPath(shipmentId)}
          target="_blank"
          rel="noopener noreferrer"
          aria-label={t("Open shipment {0}", display)}
          className="text-muted-foreground hover:text-foreground shrink-0"
        >
          <LinkExternal01Icon className="size-3" />
        </a>
      ) : null}
    </div>
  );
}

function DocumentChips({ item }: { item: BillingTransferRunItem }) {
  const t = useT();

  if (item.missingRequirements.length === 0 && item.validationFailures.length === 0) {
    return null;
  }

  return (
    <div className="flex flex-col gap-1">
      {item.missingRequirements.length > 0 ? (
        <div className="flex flex-wrap items-center gap-1">
          <span className="text-muted-foreground text-xs">{t("Missing documents:")}</span>
          {item.missingRequirements.map((requirement) => (
            <Badge
              key={requirement.documentTypeId}
              variant="neutral"
              appearance="outline"
              className="max-h-5 text-2xs"
            >
              {requirement.documentTypeName}
            </Badge>
          ))}
        </div>
      ) : null}
      {item.validationFailures.length > 0 ? (
        <ul className="text-muted-foreground list-disc pl-4 text-xs">
          {item.validationFailures.map((failure) => (
            <li key={`${failure.field}-${failure.code}`}>{failure.message}</li>
          ))}
        </ul>
      ) : null}
    </div>
  );
}

function FailureItem({ item, showReason }: { item: BillingTransferRunItem; showReason: boolean }) {
  const t = useT();
  const reason = BILLING_TRANSFER_FAILURE_REASONS[item.failureCode ?? "Unexpected"];

  return (
    <li className="flex flex-col gap-1.5 px-3 py-2.5">
      <div className="flex items-center justify-between gap-3">
        <ShipmentLink shipmentId={item.shipmentId} proNumber={item.proNumber} />
        <div className="flex items-center gap-1.5">
          {item.markedReadyToInvoice ? (
            <Badge variant="info" className="max-h-5 text-2xs">
              {t("Marked ready to invoice")}
            </Badge>
          ) : null}
          {showReason ? (
            <span
              className="text-warning-foreground text-xs font-medium"
              title={t(reason.description)}
            >
              {t(reason.label)}
            </span>
          ) : null}
        </div>
      </div>
      {item.errorMessage ? (
        <p className="text-muted-foreground text-xs">{item.errorMessage}</p>
      ) : null}
      <DocumentChips item={item} />
    </li>
  );
}

function TransferredItem({ item }: { item: BillingTransferRunItem }) {
  const t = useT();
  const hasOpenItems = item.missingRequirements.length > 0 || item.validationFailures.length > 0;

  return (
    <li className="flex flex-col gap-1.5 px-3 py-2.5">
      <div className="flex items-center justify-between gap-3">
        <ShipmentLink shipmentId={item.shipmentId} proNumber={item.proNumber} />
        <div className="flex items-center gap-1.5">
          {item.markedReadyToInvoice ? (
            <Badge variant="info" className="max-h-5 text-2xs">
              {t("Marked ready to invoice")}
            </Badge>
          ) : null}
          {item.billingQueueNumber ? (
            <span className="text-muted-foreground font-mono text-xs">
              {item.billingQueueNumber}
            </span>
          ) : null}
          {item.billingQueueStatus ? (
            <PlainBillingQueueStatusBadge status={item.billingQueueStatus} />
          ) : null}
        </div>
      </div>
      {hasOpenItems ? (
        <>
          <p className="text-muted-foreground text-xs">
            {t("Transferred for billing review with open items:")}
          </p>
          <DocumentChips item={item} />
        </>
      ) : null}
    </li>
  );
}

function NotProcessedItem({ item }: { item: BillingTransferRunItem }) {
  const t = useT();

  return (
    <li className="flex items-center justify-between gap-3 px-3 py-2.5">
      <ShipmentLink shipmentId={item.shipmentId} proNumber={item.proNumber} />
      <span className="text-muted-foreground text-xs">{t("Not sent")}</span>
    </li>
  );
}
