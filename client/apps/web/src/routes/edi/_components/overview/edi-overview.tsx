import { useT } from "@trenova/shared/i18n/use-t";
import { ComponentLoader } from "@trenova/shared/components/component-loader";
import { HoverCardTimestamp } from "@/components/hover-card-timestamp";
import { ediWindowHasTraffic } from "@/lib/edi-summary";
import { Badge } from "@trenova/shared/components/ui/badge";
import { Button } from "@trenova/shared/components/ui/button";
import type { EdiSummaryDocument } from "@trenova/graphql/generated/graphql";
import type { ResultOf } from "@graphql-typed-document-node/core";
import { AlertTriangleIcon } from "lucide-react";
import { useState } from "react";
import { Link } from "react-router";
import { InfoTile } from "../panel/edi-panel-primitives";
import { EDIOverviewEmpty } from "./edi-overview-empty";
import { EDIPartnerScorecards } from "./edi-partner-scorecards";
import { EDITrendCharts } from "./edi-trend-charts";
import { useEDIPartnerScorecards, useEDISummary, useEDIVolumeSeries } from "./use-edi-summary";

type EDISummaryResult = ResultOf<typeof EdiSummaryDocument>["ediSummary"];
type EDISummaryAttentionItem = EDISummaryResult["attentionItems"][number];

const TIME_RANGE_OPTIONS: { label: string; sinceHours?: number }[] = [
  { label: "4h", sinceHours: 4 },
  { label: "24h", sinceHours: 24 },
  { label: "7d", sinceHours: 168 },
  { label: "30d", sinceHours: 720 },
  { label: "All", sinceHours: undefined },
];

const HOURS_PER_DAY = 24;

function windowInWords(sinceHours: number): string {
  if (sinceHours < HOURS_PER_DAY) return `last ${sinceHours} hours`;
  if (sinceHours === HOURS_PER_DAY) return "last 24 hours";
  return `last ${sinceHours / HOURS_PER_DAY} days`;
}

export function EDIOverview() {
  const t = useT();

  const [sinceHours, setSinceHours] = useState<number | undefined>(24);
  const { data, isLoading, isError } = useEDISummary(sinceHours);
  const scorecardsQuery = useEDIPartnerScorecards(sinceHours);
  const volumeQuery = useEDIVolumeSeries(sinceHours);

  if (isLoading) {
    return <ComponentLoader message={t("Loading EDI operations summary")} />;
  }
  if (isError || !data) {
    return (
      <div className="bg-background text-muted-foreground rounded-md border p-6 text-sm">
        {t("The EDI operations summary could not be loaded. Retry shortly or check the API logs.")}
      </div>
    );
  }

  const summary = data.ediSummary;
  // A window nothing moved through is drawn as the overview it will become,
  // not as eight zeros; the range control stays so the reader can widen it.
  const quiet = !ediWindowHasTraffic(summary);
  const deadLettered = countFor(summary.deliveryStatusCounts, "DeadLettered");
  const failedDeliveries = countFor(summary.deliveryStatusCounts, "Failed");
  const quarantined = countFor(summary.inboundFileStatusCounts, "Quarantined");
  const partiallyProcessed = countFor(summary.inboundFileStatusCounts, "PartiallyProcessed");
  const mappingRequired = countFor(summary.inboundTransferStatusCounts, "MappingRequired");
  const pendingApproval = countFor(summary.inboundTransferStatusCounts, "PendingApproval");
  const rejectedAcks = countFor(summary.ackStatusCounts, "Rejected");

  return (
    <div className="flex flex-col gap-6 p-3">
      <div className="flex items-center justify-between gap-3">
        <p className="text-muted-foreground text-xs">
          {t("Counts, trends, and partner scorecards for the selected window.")}
        </p>
        <div className="bg-background flex items-center gap-1 rounded-md border p-0.5">
          {TIME_RANGE_OPTIONS.map((option) => (
            <Button
              key={option.label}
              type="button"
              size="sm"
              variant={option.sinceHours === sinceHours ? "secondary" : "ghost"}
              className="h-6 px-2 text-xs"
              onClick={() => setSinceHours(option.sinceHours)}
            >
              {t(option.label)}
            </Button>
          ))}
        </div>
      </div>
      {quiet ? (
        <EDIOverviewEmpty
          title={sinceHours === undefined ? "Nothing yet" : "Nothing in this window"}
          description={
            sinceHours === undefined
              ? "No document has moved through EDI for this organization. Set up a trading partner and the first tender, invoice or acknowledgment fills this in."
              : `No document moved through EDI in the ${windowInWords(sinceHours)}. Look at everything to see older traffic, or wait for the next document to arrive.`
          }
          onWiden={sinceHours === undefined ? undefined : () => setSinceHours(undefined)}
        />
      ) : (
        <>
          <section className="flex flex-col gap-3">
            <h2 className="text-sm font-semibold">{t("Needs attention")}</h2>
            <div className="grid grid-cols-2 gap-3 md:grid-cols-4">
              <Link to="/edi/messages">
                <InfoTile
                  label={t("Dead-lettered messages")}
                  value={deadLettered}
                  hint={t("Outbound deliveries that exhausted retries")}
                  size="kpi"
                  emphasizeWhenPositive
                />
              </Link>
              <Link to="/edi/inbound-files">
                <InfoTile
                  label={t("Quarantined files")}
                  value={quarantined}
                  hint={t("Inbound files that failed processing")}
                  size="kpi"
                  emphasizeWhenPositive
                />
              </Link>
              <Link to="/edi/transfers/inbound">
                <InfoTile
                  label={t("Stuck transfers")}
                  value={mappingRequired}
                  hint={t("Inbound tenders waiting on mappings")}
                  size="kpi"
                  emphasizeWhenPositive
                />
              </Link>
              <Link to="/edi/messages">
                <InfoTile
                  label={t("Overdue acknowledgments")}
                  value={summary.overdueAckCount}
                  hint={t("Pending 997/999 past the expected window")}
                  size="kpi"
                  emphasizeWhenPositive
                />
              </Link>
            </div>
          </section>
          <section className="flex flex-col gap-3">
            <h2 className="text-sm font-semibold">{t("Pipeline state")}</h2>
            <div className="grid grid-cols-2 gap-3 md:grid-cols-4">
              <Link to="/edi/messages">
                <InfoTile
                  label={t("Failed deliveries")}
                  value={failedDeliveries}
                  hint={t("Retrying with backoff")}
                  size="kpi"
                />
              </Link>
              <Link to="/edi/inbound-files">
                <InfoTile
                  label={t("Partially processed files")}
                  value={partiallyProcessed}
                  hint={t("Processed with warnings or failures")}
                  size="kpi"
                />
              </Link>
              <Link to="/edi/transfers/inbound">
                <InfoTile
                  label={t("Pending approval")}
                  value={pendingApproval}
                  hint={t("Inbound tenders awaiting review")}
                  size="kpi"
                />
              </Link>
              <Link to="/edi/messages">
                <InfoTile
                  label={t("Rejected acknowledgments")}
                  value={rejectedAcks}
                  hint={t("Partners rejected our documents")}
                  size="kpi"
                />
              </Link>
            </div>
          </section>
          <section className="flex flex-col gap-3">
            <h2 className="text-sm font-semibold">{t("Trends")}</h2>
            {volumeQuery.isError ? (
              <div className="bg-background text-muted-foreground rounded-md border p-6 text-sm">
                {t("The volume trend could not be loaded.")}
              </div>
            ) : (
              <EDITrendCharts points={volumeQuery.data?.ediVolumeSeries ?? []} />
            )}
          </section>
          <section className="flex flex-col gap-3">
            <h2 className="text-sm font-semibold">{t("Partner scorecards")}</h2>
            {scorecardsQuery.isError ? (
              <div className="bg-background text-muted-foreground rounded-md border p-6 text-sm">
                {t("Partner scorecards could not be loaded.")}
              </div>
            ) : (
              <EDIPartnerScorecards scorecards={scorecardsQuery.data?.ediPartnerScorecards ?? []} />
            )}
          </section>
          <section className="flex flex-col gap-3">
            <div className="flex items-center justify-between gap-2">
              <h2 className="text-sm font-semibold">
                {t("Recent failures")}
                {summary.attentionItems.length > 0 && (
                  <span className="text-muted-foreground ml-2 text-xs font-normal">
                    {t("showing the {0} most recent", summary.attentionItems.length)}
                  </span>
                )}
              </h2>
              <div className="flex items-center gap-3 text-xs">
                <Link to="/edi/messages" className="text-muted-foreground hover:underline">
                  {t("View all messages")}
                </Link>
                <Link to="/edi/inbound-files" className="text-muted-foreground hover:underline">
                  {t("View all inbound files")}
                </Link>
              </div>
            </div>
            {summary.attentionItems.length === 0 ? (
              <div className="bg-background text-muted-foreground rounded-md border p-6 text-sm">
                {t("No dead-lettered messages or quarantined files. The pipeline is healthy.")}
              </div>
            ) : (
              <div className="bg-background flex flex-col divide-y rounded-md border">
                {summary.attentionItems.map((item) => (
                  <AttentionRow key={`${item.kind}-${item.id}`} item={item} />
                ))}
              </div>
            )}
          </section>
        </>
      )}
    </div>
  );
}

function AttentionRow({ item }: { item: EDISummaryAttentionItem }) {
  const t = useT();

  const isMessage = item.kind === "Message";
  const target = isMessage
    ? `/edi/messages?panelType=edit&panelEntityId=${item.id}`
    : `/edi/inbound-files?panelType=edit&panelEntityId=${item.id}`;

  return (
    <Link to={target} className="hover:bg-muted/40 flex items-start gap-3 p-3 transition-colors">
      <AlertTriangleIcon className="text-muted-foreground mt-0.5 size-4 shrink-0" />
      <div className="min-w-0 flex-1">
        <div className="flex flex-wrap items-center gap-2">
          <Badge variant="outline">
            {isMessage ? t("Dead-lettered message") : t("Quarantined file")}
          </Badge>
          {item.reference && <span className="truncate text-sm font-medium">{item.reference}</span>}
          {item.partnerName && (
            <span className="text-muted-foreground truncate text-xs">
              {item.partnerCode ? `${item.partnerCode} — ` : ""}
              {item.partnerName}
            </span>
          )}
        </div>
        {item.error && (
          <div className="text-muted-foreground mt-1 line-clamp-2 text-xs">{item.error}</div>
        )}
      </div>
      <div className="text-muted-foreground shrink-0 text-xs">
        <HoverCardTimestamp timestamp={item.occurredAt} />
      </div>
    </Link>
  );
}

function countFor(counts: { status: string; count: number }[], status: string) {
  return counts.find((entry) => entry.status === status)?.count ?? 0;
}
