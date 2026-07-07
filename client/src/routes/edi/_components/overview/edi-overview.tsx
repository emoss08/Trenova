import { ComponentLoader } from "@/components/component-loader";
import { HoverCardTimestamp } from "@/components/hover-card-timestamp";
import { Badge } from "@/components/ui/badge";
import type { EdiSummaryDocument } from "@/graphql/generated/graphql";
import type { ResultOf } from "@graphql-typed-document-node/core";
import { AlertTriangleIcon } from "lucide-react";
import { Link } from "react-router";
import { InfoTile } from "../panel/edi-panel-primitives";
import { useEDISummary } from "./use-edi-summary";

type EDISummaryResult = ResultOf<typeof EdiSummaryDocument>["ediSummary"];
type EDISummaryAttentionItem = EDISummaryResult["attentionItems"][number];

export function EDIOverview() {
  const { data, isLoading, isError } = useEDISummary();

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
      <section className="flex flex-col gap-3">
        <h2 className="text-sm font-semibold">Needs attention</h2>
        <div className="grid grid-cols-2 gap-3 md:grid-cols-4">
          <Link to="/edi/messages">
            <InfoTile
              label="Dead-lettered messages"
              value={deadLettered}
              hint="Outbound deliveries that exhausted retries"
            />
          </Link>
          <Link to="/edi/inbound-files">
            <InfoTile
              label="Quarantined files"
              value={quarantined}
              hint="Inbound files that failed processing"
            />
          </Link>
          <Link to="/edi/transfers/inbound">
            <InfoTile
              label="Stuck transfers"
              value={mappingRequired}
              hint="Inbound tenders waiting on mappings"
            />
          </Link>
          <Link to="/edi/messages">
            <InfoTile
              label="Overdue acknowledgments"
              value={summary.overdueAckCount}
              hint="Pending 997/999 past the expected window"
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
            />
          </Link>
          <Link to="/edi/inbound-files">
            <InfoTile
              label="Partially processed files"
              value={partiallyProcessed}
              hint="Processed with warnings or failures"
            />
          </Link>
          <Link to="/edi/transfers/inbound">
            <InfoTile
              label="Pending approval"
              value={pendingApproval}
              hint="Inbound tenders awaiting review"
            />
          </Link>
          <Link to="/edi/messages">
            <InfoTile
              label="Rejected acknowledgments"
              value={rejectedAcks}
              hint="Partners rejected our documents"
            />
          </Link>
        </div>
      </section>
      <section className="flex flex-col gap-3">
        <h2 className="text-sm font-semibold">Recent failures</h2>
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
