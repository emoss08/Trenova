import type { ShipmentTender } from "@/lib/graphql/tender";
import { dispatchConsoleQueries } from "@/lib/queries/dispatch-console";
import { TenderOfferStatusBadge, TenderStatusBadge } from "@trenova/shared/components/status-badge";
import { Badge } from "@trenova/shared/components/ui/badge";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { formatUnixDateTime } from "@trenova/shared/lib/date";
import { cn } from "@trenova/shared/lib/utils";
import {
  TENDER_CHANNEL_LABEL,
  TENDER_MODE_LABEL,
  TENDER_RESPONSE_SOURCE_LABEL,
} from "@trenova/shared/types/tender";
import { useQuery } from "@tanstack/react-query";
import { ChevronDownIcon, ChevronRightIcon } from "lucide-react";
import { useState } from "react";
import { formatOfferRate } from "./tender-vocabulary";

function acceptedCarrierName(tender: ShipmentTender): string | null {
  if (!tender.acceptedOfferId) return null;
  const accepted = tender.offers?.find((offer) => offer.id === tender.acceptedOfferId);
  return accepted?.carrier?.name ?? null;
}

function TenderHistoryRow({ tender }: { tender: ShipmentTender }) {
  const [expanded, setExpanded] = useState(false);
  const offers = [...(tender.offers ?? [])].sort((a, b) => a.rank - b.rank);
  const acceptedCarrier = acceptedCarrierName(tender);

  return (
    <div className="bg-card flex flex-col rounded-md border">
      <button
        type="button"
        className="flex items-center gap-1.5 p-2 text-left"
        onClick={() => setExpanded((value) => !value)}
        aria-expanded={expanded}
      >
        {expanded ? (
          <ChevronDownIcon className="text-muted-foreground size-3 shrink-0" aria-hidden />
        ) : (
          <ChevronRightIcon className="text-muted-foreground size-3 shrink-0" aria-hidden />
        )}
        <Badge variant="outline" className="h-4 shrink-0 rounded px-1 text-[9px]">
          {TENDER_MODE_LABEL[tender.mode]}
        </Badge>
        <TenderStatusBadge status={tender.status} className="shrink-0 text-[9px]" />
        <span className="text-muted-foreground ml-auto shrink-0 text-[10px]">
          {formatUnixDateTime(tender.createdAt)}
        </span>
      </button>

      {acceptedCarrier && (
        <span className="text-muted-foreground px-2 pb-1 text-[10px]">
          Accepted by <span className="text-foreground font-medium">{acceptedCarrier}</span>
          {tender.acceptedAt != null ? ` · ${formatUnixDateTime(tender.acceptedAt)}` : ""}
        </span>
      )}
      {tender.status === "Canceled" && tender.cancellationReason && (
        <span className="text-muted-foreground px-2 pb-1 text-[10px] italic">
          “{tender.cancellationReason}”
        </span>
      )}

      {expanded && (
        <div className="flex flex-col gap-1 border-t p-2">
          {offers.map((offer) => (
            <div
              key={offer.id}
              className="bg-muted/30 flex flex-col gap-0.5 rounded border px-2 py-1"
            >
              <div className="flex items-center justify-between gap-2">
                <div className="flex min-w-0 items-center gap-1.5">
                  <Badge
                    variant="outline"
                    className="h-4 shrink-0 rounded px-1 text-[9px] tabular-nums"
                  >
                    #{offer.rank}
                  </Badge>
                  <span className="truncate text-[11px]">
                    {offer.carrier?.name ?? "Unknown carrier"}
                  </span>
                </div>
                <TenderOfferStatusBadge status={offer.status} className="shrink-0 text-[9px]" />
              </div>
              <div className="text-muted-foreground flex flex-wrap items-center gap-x-2 text-[10px]">
                <span className="tabular-nums">
                  {formatOfferRate(offer.rate, offer.rateMethod)}
                </span>
                <span>· {TENDER_CHANNEL_LABEL[offer.channel]}</span>
                {offer.sentAt != null && <span>· sent {formatUnixDateTime(offer.sentAt)}</span>}
                {offer.respondedAt != null && (
                  <span>· responded {formatUnixDateTime(offer.respondedAt)}</span>
                )}
                {offer.respondedAt != null && offer.responseSource && (
                  <span title="How the carrier's response reached us">
                    · {TENDER_RESPONSE_SOURCE_LABEL[offer.responseSource]}
                  </span>
                )}
              </div>
              {offer.status === "Declined" && offer.declineReason && (
                <span className="text-muted-foreground text-[10px] italic">
                  “{offer.declineReason}”
                </span>
              )}
            </div>
          ))}
          {offers.length === 0 && (
            <p className="text-muted-foreground py-1 text-center text-[10px]">No offers.</p>
          )}
        </div>
      )}
    </div>
  );
}

/**
 * Every tender the shipment has seen, newest first — the audit trail a
 * dispatcher reads before tendering the same freight again.
 */
export function TenderHistory({
  shipmentId,
  className,
}: {
  shipmentId: string;
  className?: string;
}) {
  const { data, isLoading } = useQuery({
    ...dispatchConsoleQueries.shipmentTenders(shipmentId),
  });

  const tenders = [...(data ?? [])].sort((a, b) => b.createdAt - a.createdAt);

  if (!isLoading && tenders.length === 0) {
    return null;
  }

  return (
    <div className={cn("flex flex-col gap-1.5", className)}>
      <span className="text-muted-foreground text-[10px] tracking-wide uppercase">Tenders</span>
      {isLoading ? (
        <Skeleton className="h-12 rounded-md" />
      ) : (
        tenders.map((tender) => <TenderHistoryRow key={tender.id} tender={tender} />)
      )}
    </div>
  );
}
