import type { BulkBillingTransferResult } from "@/lib/graphql/billing-transfer";
import { shipmentPanelPath } from "@/lib/shipment-utils";
import { PlainBillingQueueStatusBadge } from "@trenova/shared/components/status-badge";
import { Badge } from "@trenova/shared/components/ui/badge";
import { ScrollArea } from "@trenova/shared/components/ui/scroll-area";
import { SegmentedControl } from "@trenova/shared/components/ui/segmented-control";
import { formatNumber } from "@trenova/shared/i18n/format";
import { useT } from "@trenova/shared/i18n/use-t";
import { cn } from "@trenova/shared/lib/utils";
import { AlertTriangleIcon, ExternalLinkIcon, InfoIcon } from "lucide-react";
import { useState } from "react";
import {
  BILLING_TRANSFER_FAILURE_REASONS,
  summarizeBulkBillingTransfer,
  type BulkBillingTransferOutcome,
} from "./bulk-billing-transfer-report";

type ResultView = "notTransferred" | "transferred" | "notProcessed";

type BulkBillingTransferResultsProps = {
  outcome: BulkBillingTransferOutcome;
  stopped: boolean;
  error: unknown;
  unmatchedCount: number;
  proNumbers: ReadonlyMap<string, string>;
};

function errorMessage(error: unknown): string | null {
  return error instanceof Error && error.message ? error.message : null;
}

export function BulkBillingTransferResults({
  outcome,
  stopped,
  error,
  unmatchedCount,
  proNumbers,
}: BulkBillingTransferResultsProps) {
  const t = useT();
  const summary = summarizeBulkBillingTransfer(outcome);

  const [preferredView, setPreferredView] = useState<ResultView | null>(null);
  const defaultView: ResultView =
    summary.notTransferred > 0
      ? "notTransferred"
      : summary.transferred > 0
        ? "transferred"
        : "notProcessed";
  const view = preferredView ?? defaultView;

  const transferred = outcome.results.filter((result) => result.success);
  const notTransferred = outcome.results.filter((result) => !result.success);

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

      {error ? (
        <Notice tone="danger" title={t("The transfer stopped early")}>
          {errorMessage(error) ? `${errorMessage(error)} ` : ""}
          {t(
            "Shipments in the batch that was running may still have transferred. Their queue entries appear once the list refreshes.",
          )}
        </Notice>
      ) : null}
      {stopped ? (
        <Notice tone="muted">
          {t("You stopped the transfer. The remaining shipments were not sent.")}
        </Notice>
      ) : null}
      {unmatchedCount > 0 ? (
        <Notice tone="muted">
          {t(
            "{1} more {0, plural, one {shipment} other {shipments}} matched than one run transfers. Run Transfer all again for the rest.",
            unmatchedCount,
            formatNumber(unmatchedCount),
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
            {view === "notTransferred" ? (
              <ResultList label={t("Not transferred")} empty={t("Every shipment transferred.")}>
                {notTransferred.map((result) => (
                  <FailureItem key={result.shipmentId} result={result} />
                ))}
              </ResultList>
            ) : view === "transferred" ? (
              <ResultList label={t("Transferred")} empty={t("No shipment transferred.")}>
                {transferred.map((result) => (
                  <TransferredItem key={result.shipmentId} result={result} />
                ))}
              </ResultList>
            ) : (
              <ResultList label={t("Not processed")} empty={t("Every shipment was processed.")}>
                {outcome.notProcessedIds.map((shipmentId) => (
                  <li
                    key={shipmentId}
                    className="flex items-center justify-between gap-3 px-3 py-2.5"
                  >
                    <ShipmentLink
                      shipmentId={shipmentId}
                      proNumber={proNumbers.get(shipmentId) ?? null}
                    />
                    <span className="text-muted-foreground text-xs">{t("Not sent")}</span>
                  </li>
                ))}
              </ResultList>
            )}
          </ScrollArea>
        </>
      )}
    </div>
  );
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

function DocumentChips({ result }: { result: BulkBillingTransferResult }) {
  const t = useT();

  if (result.missingRequirements.length === 0 && result.validationFailures.length === 0) {
    return null;
  }

  return (
    <div className="flex flex-col gap-1">
      {result.missingRequirements.length > 0 ? (
        <div className="flex flex-wrap items-center gap-1">
          <span className="text-muted-foreground text-xs">{t("Missing documents:")}</span>
          {result.missingRequirements.map((requirement) => (
            <Badge
              key={requirement.documentTypeId}
              variant="neutral" appearance="outline"
              className="max-h-5 text-2xs"
            >
              {requirement.documentTypeName}
            </Badge>
          ))}
        </div>
      ) : null}
      {result.validationFailures.length > 0 ? (
        <ul className="text-muted-foreground list-disc pl-4 text-xs">
          {result.validationFailures.map((failure) => (
            <li key={`${failure.field}-${failure.code}`}>{failure.message}</li>
          ))}
        </ul>
      ) : null}
    </div>
  );
}

function FailureItem({ result }: { result: BulkBillingTransferResult }) {
  const t = useT();
  const reason = BILLING_TRANSFER_FAILURE_REASONS[result.failureCode ?? "Unexpected"];

  return (
    <li className="flex flex-col gap-1.5 px-3 py-2.5">
      <div className="flex items-center justify-between gap-3">
        <ShipmentLink shipmentId={result.shipmentId} proNumber={result.proNumber} />
        <div className="flex items-center gap-1.5">
          {result.markedReadyToInvoice ? (
            <Badge variant="info" className="max-h-5 text-2xs">
              {t("Marked Ready to Invoice")}
            </Badge>
          ) : null}
          <span className="text-destructive text-xs font-medium" title={t(reason.description)}>
            {t(reason.label)}
          </span>
        </div>
      </div>
      {result.error ? <p className="text-muted-foreground text-xs">{result.error}</p> : null}
      <DocumentChips result={result} />
    </li>
  );
}

function TransferredItem({ result }: { result: BulkBillingTransferResult }) {
  const t = useT();
  const hasOpenItems =
    result.missingRequirements.length > 0 || result.validationFailures.length > 0;

  return (
    <li className="flex flex-col gap-1.5 px-3 py-2.5">
      <div className="flex items-center justify-between gap-3">
        <ShipmentLink shipmentId={result.shipmentId} proNumber={result.proNumber} />
        <div className="flex items-center gap-1.5">
          {result.markedReadyToInvoice ? (
            <Badge variant="info" className="max-h-5 text-2xs">
              {t("Marked Ready to Invoice")}
            </Badge>
          ) : null}
          {result.billingQueueItem ? (
            <>
              <span className="text-muted-foreground font-mono text-xs">
                {result.billingQueueItem.number}
              </span>
              <PlainBillingQueueStatusBadge status={result.billingQueueItem.status} />
            </>
          ) : null}
        </div>
      </div>
      {hasOpenItems ? (
        <>
          <p className="text-muted-foreground text-xs">
            {t("Transferred for billing review with open items:")}
          </p>
          <DocumentChips result={result} />
        </>
      ) : null}
    </li>
  );
}
