import type { BillingTransferRun, BillingTransferRunItem } from "@/lib/graphql/billing-transfer";
import { shipmentPanelPath } from "@/lib/shipment-utils";
import { useInfiniteQuery } from "@tanstack/react-query";
import { PlainBillingQueueStatusBadge } from "@trenova/shared/components/status-badge";
import { Badge } from "@trenova/shared/components/ui/badge";
import { Button } from "@trenova/shared/components/ui/button";
import { ScrollArea } from "@trenova/shared/components/ui/scroll-area";
import { SegmentedControl } from "@trenova/shared/components/ui/segmented-control";
import { Spinner } from "@trenova/shared/components/ui/spinner";
import { formatNumber } from "@trenova/shared/i18n/format";
import { useT } from "@trenova/shared/i18n/use-t";
import { cn } from "@trenova/shared/lib/utils";
import { AlertTriangleIcon, ExternalLinkIcon, InfoIcon } from "lucide-react";
import { useMemo, useState } from "react";
import { billingTransferRunItemsQuery } from "../../billing-queue-queries";
import { BILLING_TRANSFER_FAILURE_REASONS } from "./bulk-billing-transfer-report";
import { summarizeRun } from "./bulk-billing-transfer-run";

type ResultView = "notTransferred" | "transferred" | "notProcessed";

const VIEW_STATUSES: Record<ResultView, BillingTransferRunItem["status"][]> = {
  notTransferred: ["NotTransferred"],
  transferred: ["Transferred"],
  notProcessed: ["Skipped", "Pending"],
};

export function BulkBillingTransferResults({ run }: { run: BillingTransferRun }) {
  const t = useT();
  const summary = summarizeRun(run);

  const [preferredView, setPreferredView] = useState<ResultView | null>(null);
  const defaultView: ResultView =
    summary.notTransferred > 0
      ? "notTransferred"
      : summary.transferred > 0
        ? "transferred"
        : "notProcessed";
  const view = preferredView ?? defaultView;

  const itemsQuery = useInfiniteQuery(billingTransferRunItemsQuery(run.id, VIEW_STATUSES[view]));
  const items = useMemo<BillingTransferRunItem[]>(
    () => itemsQuery.data?.pages.flatMap((page) => page.edges.map((edge) => edge.node)) ?? [],
    [itemsQuery.data],
  );

  const viewItems = [
    {
      value: "notTransferred" as const,
      label: t("Not transferred"),
      caption: summary.notTransferred,
    },
    { value: "transferred" as const, label: t("Transferred"), caption: summary.transferred },
    ...(summary.notProcessed > 0
      ? [
          {
            value: "notProcessed" as const,
            label: t("Not processed"),
            caption: summary.notProcessed,
          },
        ]
      : []),
  ];

  return (
    <div className="flex min-h-0 flex-col gap-3">
      <dl
        role="group"
        aria-label={t("Transfer summary")}
        className="grid grid-cols-2 gap-3 sm:grid-cols-3"
      >
        <SummaryTile
          label={t("Transferred")}
          value={summary.transferred}
          hint={
            summary.markedReadyToInvoice > 0
              ? t("{0} marked Ready to Invoice first", summary.markedReadyToInvoice)
              : null
          }
          tone="success"
        />
        <SummaryTile label={t("Not transferred")} value={summary.notTransferred} tone="danger" />
        {summary.notProcessed > 0 ? (
          <SummaryTile label={t("Not processed")} value={summary.notProcessed} tone="muted" />
        ) : null}
      </dl>

      {run.status === "Failed" ? (
        <Notice tone="danger" title={t("The transfer stopped early")}>
          {run.failureMessage ?? t("The transfer stopped because of an unexpected error.")}
        </Notice>
      ) : null}
      {run.status === "Canceled" ? (
        <Notice tone="muted">
          {t("You stopped the transfer. The remaining shipments were not sent.")}
        </Notice>
      ) : null}
      {summary.unmatched > 0 ? (
        <Notice tone="muted">
          {t(
            "{1} more {0, plural, one {shipment} other {shipments}} matched than one run transfers. Run Transfer all again for the rest.",
            summary.unmatched,
            formatNumber(summary.unmatched),
          )}
        </Notice>
      ) : null}

      {summary.total === 0 ? (
        <p className="text-muted-foreground py-8 text-center text-sm">
          {t("No shipments matched, so nothing was transferred.")}
        </p>
      ) : (
        <>
          <SegmentedControl
            aria-label={t("Show outcome")}
            items={viewItems}
            value={view}
            onValueChange={setPreferredView}
          />
          <ScrollArea className="h-[min(22rem,45vh)] rounded-lg border">
            {itemsQuery.isPending ? (
              <div className="flex items-center justify-center py-10">
                <Spinner className="size-4" />
              </div>
            ) : itemsQuery.isError ? (
              <div className="flex flex-col items-center gap-2 px-3 py-8 text-center">
                <p className="text-muted-foreground text-sm">
                  {t("The transfer report could not be loaded.")}
                </p>
                <Button variant="outline" size="xs" onClick={() => void itemsQuery.refetch()}>
                  {t("Try again")}
                </Button>
              </div>
            ) : (
              <>
                <ResultList label={viewLabel(view, t)} empty={emptyLabel(view, t)}>
                  {items.map((item) =>
                    view === "notTransferred" ? (
                      <FailureItem key={item.id} item={item} />
                    ) : view === "transferred" ? (
                      <TransferredItem key={item.id} item={item} />
                    ) : (
                      <NotProcessedItem key={item.id} item={item} />
                    ),
                  )}
                </ResultList>
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
        </>
      )}
    </div>
  );
}

function viewLabel(view: ResultView, t: ReturnType<typeof useT>): string {
  switch (view) {
    case "notTransferred":
      return t("Not transferred");
    case "transferred":
      return t("Transferred");
    default:
      return t("Not processed");
  }
}

function emptyLabel(view: ResultView, t: ReturnType<typeof useT>): string {
  switch (view) {
    case "notTransferred":
      return t("Every shipment transferred.");
    case "transferred":
      return t("No shipment transferred.");
    default:
      return t("Every shipment was processed.");
  }
}

function SummaryTile({
  label,
  value,
  hint,
  tone,
}: {
  label: string;
  value: number;
  hint?: string | null;
  tone: "success" | "danger" | "muted";
}) {
  return (
    <div className="rounded-lg border px-3 py-2">
      <dt className="text-muted-foreground text-xs">{label}</dt>
      <dd
        className={cn(
          "text-xl font-semibold tabular-nums",
          tone === "success" && value > 0 && "text-success-foreground",
          tone === "danger" && value > 0 && "text-destructive",
        )}
      >
        {value}
      </dd>
      {hint ? <p className="text-muted-foreground text-xs">{hint}</p> : null}
    </div>
  );
}

function Notice({
  tone,
  title,
  children,
}: {
  tone: "danger" | "muted";
  title?: string;
  children: React.ReactNode;
}) {
  const Icon = tone === "danger" ? AlertTriangleIcon : InfoIcon;
  return (
    <div
      role={tone === "danger" ? "alert" : "status"}
      className={cn(
        "flex gap-2 rounded-lg border px-3 py-2 text-xs",
        tone === "danger" ? "border-destructive/40 bg-destructive/5" : "bg-muted/40",
      )}
    >
      <Icon
        className={cn(
          "mt-0.5 size-3.5 shrink-0",
          tone === "danger" ? "text-destructive" : "text-muted-foreground",
        )}
      />
      <div className="flex flex-col gap-0.5">
        {title ? <p className="font-medium">{title}</p> : null}
        <p className="text-muted-foreground">{children}</p>
      </div>
    </div>
  );
}

function ResultList({
  label,
  empty,
  children,
}: {
  label: string;
  empty: string;
  children: React.ReactNode[];
}) {
  if (children.length === 0) {
    return <p className="text-muted-foreground px-3 py-8 text-center text-sm">{empty}</p>;
  }
  return (
    <ul aria-label={label} className="divide-y">
      {children}
    </ul>
  );
}

function ShipmentLink({ shipmentId, proNumber }: { shipmentId: string; proNumber: string | null }) {
  const t = useT();
  const display = proNumber ?? shipmentId;

  return (
    <div className="flex min-w-0 items-center gap-1.5">
      <span className="truncate font-mono text-xs font-medium">{display}</span>
      <a
        href={shipmentPanelPath(shipmentId)}
        target="_blank"
        rel="noopener noreferrer"
        aria-label={t("Open shipment {0}", display)}
        className="text-muted-foreground hover:text-foreground shrink-0"
      >
        <ExternalLinkIcon className="size-3" />
      </a>
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
              variant="outline"
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

function FailureItem({ item }: { item: BillingTransferRunItem }) {
  const t = useT();
  const reason = BILLING_TRANSFER_FAILURE_REASONS[item.failureCode ?? "Unexpected"];

  return (
    <li className="flex flex-col gap-1.5 px-3 py-2.5">
      <div className="flex items-center justify-between gap-3">
        <ShipmentLink shipmentId={item.shipmentId} proNumber={item.proNumber} />
        <div className="flex items-center gap-1.5">
          {item.markedReadyToInvoice ? (
            <Badge variant="info" className="max-h-5 text-2xs">
              {t("Marked Ready to Invoice")}
            </Badge>
          ) : null}
          <span className="text-destructive text-xs font-medium" title={t(reason.description)}>
            {t(reason.label)}
          </span>
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
              {t("Marked Ready to Invoice")}
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
