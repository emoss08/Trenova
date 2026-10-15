import { useT } from "@trenova/shared/i18n/use-t";
import type { ShipmentTender } from "@/lib/graphql/tender";
import { dispatchConsoleQueries } from "@/lib/queries/dispatch-console";
import { TenderOfferStatusBadge, TenderStatusBadge } from "@trenova/shared/components/status-badge";
import { Badge } from "@trenova/shared/components/ui/badge";
import { Button } from "@trenova/shared/components/ui/button";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { formatUnixDateTime } from "@trenova/shared/lib/date";
import { cn } from "@trenova/shared/lib/utils";
import { usePermissionStore } from "@trenova/shared/stores/permission-store";
import { Operation, Resource } from "@trenova/shared/types/permission";
import {
  TENDER_CHANNEL_LABEL,
  TENDER_MODE_LABEL,
  TENDER_RESPONSE_SOURCE_LABEL,
  type BounceTenderPayload,
} from "@trenova/shared/types/tender";
import { useQuery } from "@tanstack/react-query";
import { ChevronDownIcon, ChevronRightIcon } from "@trenova/shared/components/icons";
import { useCallback, useState } from "react";
import { TenderBounceDialog } from "./tender-bounce-dialog";
import { formatOfferRate } from "./tender-vocabulary";

type BounceTender = (params: {
  tenderId: string;
  payload: BounceTenderPayload;
}) => Promise<unknown>;

function acceptedOffer(tender: ShipmentTender) {
  if (!tender.acceptedOfferId) return null;
  return tender.offers?.find((offer) => offer.id === tender.acceptedOfferId) ?? null;
}

/**
 * A carrier can back out of a tender it accepted while the acceptance still
 * stands: the tender is accepted, or parked for review, and the offer it
 * accepted has not already been walked back. The server holds the same rule.
 */
export function canRecordBackout(tender: ShipmentTender): boolean {
  if (tender.status !== "Accepted" && tender.status !== "NeedsReview") return false;
  return acceptedOffer(tender)?.status === "Accepted";
}

function TenderHistoryRow({
  tender,
  isTendering,
  onBounceTender,
}: {
  tender: ShipmentTender;
  isTendering: boolean;
  onBounceTender: BounceTender | null;
}) {
  const t = useT();

  const [expanded, setExpanded] = useState(false);
  const [bouncing, setBouncing] = useState(false);
  const offers = [...(tender.offers ?? [])].sort((a, b) => a.rank - b.rank);
  const accepted = acceptedOffer(tender);
  const acceptedCarrier = accepted?.carrier?.name ?? null;
  const showBackout = onBounceTender !== null && canRecordBackout(tender);

  const handleBounce = useCallback(
    (payload: BounceTenderPayload) => {
      if (!onBounceTender) return;
      void onBounceTender({ tenderId: tender.id, payload })
        .then(() => setBouncing(false))
        .catch(() => {
          // handled by the mutation's onError; the dialog stays open for a retry
        });
    },
    [onBounceTender, tender.id],
  );

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
        <Badge
          variant="neutral"
          appearance="outline"
          className="h-4 shrink-0 rounded-md px-1 text-3xs"
        >
          {TENDER_MODE_LABEL[tender.mode]}
        </Badge>
        <TenderStatusBadge status={tender.status} className="shrink-0 text-3xs" />
        <span className="text-muted-foreground ml-auto shrink-0 text-2xs">
          {formatUnixDateTime(tender.createdAt)}
        </span>
      </button>

      {acceptedCarrier && (
        <span className="text-muted-foreground px-2 pb-1 text-2xs">
          {t("Accepted by")} <span className="text-foreground font-medium">{acceptedCarrier}</span>
          {tender.acceptedAt != null ? ` · ${formatUnixDateTime(tender.acceptedAt)}` : ""}
          {accepted?.counterRate && accepted.agreedRate !== accepted.rate
            ? ` · ${t("at {0} after a counter-offer", formatOfferRate(accepted.agreedRate, accepted.rateMethod))}`
            : ""}
        </span>
      )}
      {(tender.bounceCount > 0 || tender.escalationRound > 0) && (
        <span className="text-muted-foreground px-2 pb-1 text-2xs">
          {[
            tender.escalationRound > 0 ? t("Escalation round {0}", tender.escalationRound) : null,
            tender.bounceCount > 0
              ? t(
                  "{0, plural, one {a carrier backed out once} other {carriers backed out # times}}",
                  tender.bounceCount,
                )
              : null,
          ]
            .filter(Boolean)
            .join(" · ")}
        </span>
      )}
      {showBackout && (
        <div className="px-2 pb-2">
          <Button
            size="sm"
            variant="outline"
            className="text-destructive hover:text-destructive h-6 px-2 text-2xs"
            disabled={isTendering}
            onClick={() => setBouncing(true)}
          >
            {t("Carrier backed out")}
          </Button>
        </div>
      )}
      {bouncing && (
        <TenderBounceDialog
          carrierName={acceptedCarrier}
          isSubmitting={isTendering}
          onOpenChange={setBouncing}
          onConfirm={handleBounce}
        />
      )}
      {tender.status === "Canceled" && tender.cancellationReason && (
        <span className="text-muted-foreground px-2 pb-1 text-2xs italic">
          “{tender.cancellationReason}”
        </span>
      )}

      {expanded && (
        <div className="flex flex-col gap-1 border-t p-2">
          {offers.map((offer) => (
            <div
              key={offer.id}
              className="bg-muted/30 flex flex-col gap-0.5 rounded-md border px-2 py-1"
            >
              <div className="flex items-center justify-between gap-2">
                <div className="flex min-w-0 items-center gap-1.5">
                  <Badge
                    variant="neutral"
                    appearance="outline"
                    className="h-4 shrink-0 rounded-md px-1 text-3xs tabular-nums"
                  >
                    #{offer.rank}
                  </Badge>
                  <span className="truncate text-xs">
                    {offer.carrier?.name ?? t("Unknown carrier")}
                  </span>
                </div>
                <TenderOfferStatusBadge status={offer.status} className="shrink-0 text-3xs" />
              </div>
              <div className="text-muted-foreground flex flex-wrap items-center gap-x-2 text-2xs">
                <span className="tabular-nums">
                  {formatOfferRate(offer.rate, offer.rateMethod)}
                </span>
                <span>· {TENDER_CHANNEL_LABEL[offer.channel]}</span>
                {offer.sentAt != null && (
                  <span>{t("· sent {0}", formatUnixDateTime(offer.sentAt))}</span>
                )}
                {offer.respondedAt != null && (
                  <span>{t("· responded {0}", formatUnixDateTime(offer.respondedAt))}</span>
                )}
                {offer.respondedAt != null && offer.responseSource && (
                  <span title={t("How the carrier's response reached us")}>
                    · {TENDER_RESPONSE_SOURCE_LABEL[offer.responseSource]}
                  </span>
                )}
              </div>
              {offer.counterRate && (
                <span className="text-muted-foreground text-2xs tabular-nums">
                  {t("Countered at {0}", formatOfferRate(offer.counterRate, offer.rateMethod))}
                </span>
              )}
              {(offer.status === "Declined" || offer.status === "CounterRejected") &&
                offer.declineReason && (
                  <span className="text-muted-foreground text-2xs italic">
                    “{offer.declineReason}”
                  </span>
                )}
              {offer.status === "Revoked" && offer.revokeReason && (
                <span className="text-muted-foreground text-2xs">
                  {t("Backed out: “{0}”", offer.revokeReason)}
                </span>
              )}
            </div>
          ))}
          {offers.length === 0 && (
            <p className="text-muted-foreground py-1 text-center text-2xs">{t("No offers.")}</p>
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
  isTendering = false,
  onBounceTender,
}: {
  shipmentId: string;
  className?: string;
  isTendering?: boolean;
  /** Records that an accepting carrier backed out; offered to users with tender:update. */
  onBounceTender?: BounceTender;
}) {
  const t = useT();
  const canUpdateTender = usePermissionStore((state) =>
    state.hasPermission(Resource.Tender, Operation.Update),
  );
  const bounce = onBounceTender && canUpdateTender ? onBounceTender : null;

  const { data, isLoading } = useQuery({
    ...dispatchConsoleQueries.shipmentTenders(shipmentId),
  });

  const tenders = [...(data ?? [])].sort((a, b) => b.createdAt - a.createdAt);

  if (!isLoading && tenders.length === 0) {
    return null;
  }

  return (
    <div className={cn("flex flex-col gap-1.5", className)}>
      <span className="text-muted-foreground text-xs">{t("Tenders")}</span>
      {isLoading ? (
        <Skeleton className="h-12 rounded-md" />
      ) : (
        tenders.map((tender) => (
          <TenderHistoryRow
            key={tender.id}
            tender={tender}
            isTendering={isTendering}
            onBounceTender={bounce}
          />
        ))
      )}
    </div>
  );
}
