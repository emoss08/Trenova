import { ComponentLoader } from "@/components/component-loader";
import { HoverCardTimestamp } from "@/components/hover-card-timestamp";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import type { EdiSummaryDocument } from "@/graphql/generated/graphql";
import type { ResultOf } from "@graphql-typed-document-node/core";
import { AlertTriangleIcon } from "lucide-react";
import { useState } from "react";
import { Link } from "react-router";
import { InfoTile } from "../panel/edi-panel-primitives";
import { EDIPartnerScorecards } from "./edi-partner-scorecards";
import { EDITrendCharts } from "./edi-trend-charts";
import { useEDIPartnerScorecards, useEDISummary, useEDIVolumeSeries } from "./use-edi-summary";

type EDISummaryResult = ResultOf<typeof EdiSummaryDocument>["ediSummary"];
type EDISummaryAttentionItem = EDISummaryResult["attentionItems"][number];

const TIME_RANGE_OPTIONS: { label: string; sinceHours: number | null }[] = [
  { label: "4h", sinceHours: 4 },
  { label: "24h", sinceHours: 24 },
  { label: "7d", sinceHours: 168 },
  { label: "30d", sinceHours: 720 },
  { label: "All", sinceHours: null },
];

export function EDIOverview() {
  const [sinceHours, setSinceHours] = useState<number | null>(24);
  const { data, isLoading, isError } = useEDISummary(sinceHours);
  const scorecardsQuery = useEDIPartnerScorecards(sinceHours);
  const volumeQuery = useEDIVolumeSeries(sinceHours);

  if (isLoading) {
    return <ComponentLoader message="Loading EDI operations summary" />;
  }
  if (isError || !data) {
    return (
      <div className="rounded-md border bg-background p-6 text-sm text-muted-foreground">
        The EDI operations summary could not be loaded. Retry shortly or check the API logs.
      </div>
    );
  }

  const summary = data.ediSummary;
  const deadLettered = countFor(summary.deliveryStatusCounts, "DeadLettered");
  const failedDeliveries = countFor(summary.deliveryStatusCounts, "Failed");
  const quarantined = countFor(summary.inboundFileStatusCounts, "Quarantined");
  const partiallyProcessed = countFor(summary.inboundFileStatusCounts, "PartiallyProcessed");
  const mappingRequired = countFor(summary.inboundTransferStatusCounts, "MappingRequired");
  const pendingApproval = countFor(summary.inboundTransferStatusCounts, "PendingApproval");
  const rejectedAcks = countFor(summary.ackStatusCounts, "Rejected");

  return (
    <div className="flex flex-col gap-6 px-3">
      <div className="flex items-center justify-between gap-3">
        <p className="text-xs text-muted-foreground">
          Counts, trends, and partner scorecards for the selected window.
        </p>
        <div className="flex items-center gap-1 rounded-md border bg-background p-0.5">
          {TIME_RANGE_OPTIONS.map((option) => (
            <Button
              key={option.label}
              type="button"
              size="sm"
              variant={option.sinceHours === sinceHours ? "secondary" : "ghost"}
              className="h-6 px-2 text-xs"
              onClick={() => setSinceHours(option.sinceHours)}
            >
              {option.label}
            </Button>
          ))}
        </div>
      </div>
      <section className="flex flex-col gap-3">
        <h2 className="text-sm font-semibold">Needs attention</h2>
        <div className="grid grid-cols-2 gap-3 md:grid-cols-4">
          <Link to="/edi/messages">
            <InfoTile
              label="Dead-lettered messages"
              value={deadLettered}
              hint="Outbound deliveries that exhausted retries"
              size="kpi"
              emphasizeWhenPositive
            />
          </Link>
          <Link to="/edi/inbound-files">
            <InfoTile
              label="Quarantined files"
              value={quarantined}
              hint="Inbound files that failed processing"
              size="kpi"
              emphasizeWhenPositive
            />
          </Link>
          <Link to="/edi/transfers/inbound">
            <InfoTile
              label="Stuck transfers"
              value={mappingRequired}
              hint="Inbound tenders waiting on mappings"
              size="kpi"
              emphasizeWhenPositive
            />
          </Link>
          <Link to="/edi/messages">
            <InfoTile
              label="Overdue acknowledgments"
              value={summary.overdueAckCount}
              hint="Pending 997/999 past the expected window"
              size="kpi"
              emphasizeWhenPositive
            />
          </Link>
        </div>
      </section>
      <section className="flex flex-col gap-3">
        <h2 className="text-sm font-semibold">Pipeline state</h2>
        <div className="grid grid-cols-2 gap-3 md:grid-cols-4">
          <Link to="/edi/messages">
            <InfoTile
              label="Failed deliveries"
              value={failedDeliveries}
              hint="Retrying with backoff"
              size="kpi"
            />
          </Link>
          <Link to="/edi/inbound-files">
            <InfoTile
              label="Partially processed files"
              value={partiallyProcessed}
              hint="Processed with warnings or failures"
              size="kpi"
            />
          </Link>
          <Link to="/edi/transfers/inbound">
            <InfoTile
              label="Pending approval"
              value={pendingApproval}
              hint="Inbound tenders awaiting review"
              size="kpi"
            />
          </Link>
          <Link to="/edi/messages">
            <InfoTile
              label="Rejected acknowledgments"
              value={rejectedAcks}
              hint="Partners rejected our documents"
              size="kpi"
            />
          </Link>
        </div>
      </section>
      <section className="flex flex-col gap-3">
        <h2 className="text-sm font-semibold">Trends</h2>
        {volumeQuery.isError ? (
          <div className="rounded-md border bg-background p-6 text-sm text-muted-foreground">
            The volume trend could not be loaded.
          </div>
        ) : (
          <EDITrendCharts points={volumeQuery.data?.ediVolumeSeries ?? []} />
        )}
      </section>
      <section className="flex flex-col gap-3">
        <h2 className="text-sm font-semibold">Partner scorecards</h2>
        {scorecardsQuery.isError ? (
          <div className="rounded-md border bg-background p-6 text-sm text-muted-foreground">
            Partner scorecards could not be loaded.
          </div>
        ) : (
          <EDIPartnerScorecards scorecards={scorecardsQuery.data?.ediPartnerScorecards ?? []} />
        )}
      </section>
      <section className="flex flex-col gap-3">
        <div className="flex items-center justify-between gap-2">
          <h2 className="text-sm font-semibold">
            Recent failures
            {summary.attentionItems.length > 0 && (
              <span className="ml-2 text-xs font-normal text-muted-foreground">
                showing the {summary.attentionItems.length} most recent
              </span>
            )}
          </h2>
          <div className="flex items-center gap-3 text-xs">
            <Link to="/edi/messages" className="text-muted-foreground hover:underline">
              View all messages
            </Link>
            <Link to="/edi/inbound-files" className="text-muted-foreground hover:underline">
              View all inbound files
            </Link>
          </div>
        </div>
        {summary.attentionItems.length === 0 ? (
          <div className="rounded-md border bg-background p-6 text-sm text-muted-foreground">
            No dead-lettered messages or quarantined files. The pipeline is healthy.
          </div>
        ) : (
          <div className="flex flex-col divide-y rounded-md border bg-background">
            {summary.attentionItems.map((item) => (
              <AttentionRow key={`${item.kind}-${item.id}`} item={item} />
            ))}
          </div>
        )}
      </section>
    </div>
  );
}

function AttentionRow({ item }: { item: EDISummaryAttentionItem }) {
  const isMessage = item.kind === "Message";
  const target = isMessage
    ? `/edi/messages?panelType=edit&panelEntityId=${item.id}`
    : `/edi/inbound-files?panelType=edit&panelEntityId=${item.id}`;

  return (
    <Link to={target} className="flex items-start gap-3 p-3 transition-colors hover:bg-muted/40">
      <AlertTriangleIcon className="mt-0.5 size-4 shrink-0 text-muted-foreground" />
      <div className="min-w-0 flex-1">
        <div className="flex flex-wrap items-center gap-2">
          <Badge variant="outline">
            {isMessage ? "Dead-lettered message" : "Quarantined file"}
          </Badge>
          {item.reference && <span className="truncate text-sm font-medium">{item.reference}</span>}
          {item.partnerName && (
            <span className="truncate text-xs text-muted-foreground">
              {item.partnerCode ? `${item.partnerCode} — ` : ""}
              {item.partnerName}
            </span>
          )}
        </div>
        {item.error && (
          <div className="mt-1 line-clamp-2 text-xs text-muted-foreground">{item.error}</div>
        )}
      </div>
      <div className="shrink-0 text-xs text-muted-foreground">
        <HoverCardTimestamp timestamp={item.occurredAt} />
      </div>
    </Link>
  );
}

function countFor(counts: { status: string; count: number }[], status: string) {
  return counts.find((entry) => entry.status === status)?.count ?? 0;
}
