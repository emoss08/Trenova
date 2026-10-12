import { recordPath } from "@/config/record-links";
import { useCopyToClipboard } from "@/hooks/use-copy-to-clipboard";
import { describeApiError } from "@/lib/api-error-message";
import { queries } from "@/lib/queries";
import { keepPreviousData, useQuery } from "@tanstack/react-query";
import { Button } from "@trenova/shared/components/ui/button";
import { formatNumber } from "@trenova/shared/i18n/format";
import { useT } from "@trenova/shared/i18n/use-t";
import { cn, formatPercent } from "@trenova/shared/lib/utils";
import { useAuthStore } from "@trenova/shared/stores/auth-store";
import type { Shipment, ShipmentStage } from "@trenova/shared/types/shipment";
import { useMemo, useState } from "react";
import { Link } from "react-router";
import {
  CardBar,
  CardBarHint,
  CardCheck,
  CardChecks,
  CardFacts,
  CardHead,
  CardIcon,
  CardLink,
  CardSection,
  CardStatus,
  CardStepper,
  CardTimeline,
  RecordCard,
} from "../../record-card/record-card";
import { CardDocuments } from "../../record-card/card-documents";
import { cardDay, cardMoment, money, neighborsIn } from "../../record-card/record-card-values";
import type { RecordArtifactProps } from "../registry";
import {
  STAGE_TONE,
  billingPhase,
  chargesOf,
  coverageOf,
  etaReading,
  freightOf,
  journeyProgress,
  journeyStops,
  laneOf,
  marginOf,
  nowReading,
  readinessChecks,
  shipmentStage,
  stageLabel,
  totalMiles,
  type ReadinessCheck,
  stopOffsets,
  tenderStatusLabel,
  truckOffset,
} from "./shipment-card";
import {
  ChargesBlock,
  CoverageRow,
  FreightBlock,
  MarginMeter,
  NowStrip,
  StopRail,
} from "./shipment-parts";
import { DeskShipmentSkeleton } from "./shipment-skeleton";

/** The latest events the card lists; the shipment's own page holds the rest. */
const ACTIVITY_ROWS = 5;

/** Stages the truck is still to finish, when the ETA belongs under the head. */
const UNDER_WAY: ReadonlySet<ShipmentStage> = new Set<ShipmentStage>([
  "Moving",
  "Scheduled",
  "Late",
]);

/**
 * One shipment as a dispatcher reads it, opened from a table's row or looked
 * up by the agent: the lane and who it is for, what it bills and earns, where
 * the truck is now against the stops still ahead, each stop's window and how
 * the truck kept it, who covers every move, the freight, the charges and the
 * margin, what stands between a delivered load and its invoice, and what
 * happened last. It reads; assigning, editing and tracking stay on the
 * shipment board, one click away.
 *
 * The card steps through the table's rows in the order the table shows them.
 */
export type ShipmentCardProps = {
  recordId: string;
  order?: readonly string[];
  onSelect?: (recordId: string) => void;
  onBack?: () => void;
  /** What the way back says; the table, unless the shipment was opened from another card. */
  backLabel?: string;
};

const NO_ROWS: readonly string[] = [];

export function DeskShipment({
  recordId,
  order = NO_ROWS,
  onSelect,
  onBack,
  backLabel,
}: ShipmentCardProps) {
  const t = useT();
  const query = useQuery({
    ...queries.shipment.get(recordId),
    retry: false,
    placeholderData: keepPreviousData,
  });
  const shipment = query.data;

  if (!shipment) {
    return query.isError ? (
      <RecordCard>
        <CardHead
          onBack={onBack}
          backLabel={backLabel ?? t("Back to the table")}
          kicker={[t("Shipment")]}
          title={t("This shipment could not be read")}
          subtitle={describeApiError(
            query.error,
            t("It may have been removed, or you may not be allowed to see it."),
          )}
        />
      </RecordCard>
    ) : (
      <DeskShipmentSkeleton stepper={onSelect !== undefined && order.length > 1} />
    );
  }

  return (
    <ShipmentCard
      key={shipment.id ?? recordId}
      shipment={shipment}
      recordId={shipment.id ?? recordId}
      order={order}
      onSelect={onSelect}
      onBack={onBack}
      backLabel={backLabel}
    />
  );
}

/**
 * The shipment as an artifact of its own, when the agent looked it up: the
 * same card, with nothing to step through and nowhere to go back to.
 */
export function DeskShipmentArtifact({ recordId }: RecordArtifactProps) {
  return <DeskShipment recordId={recordId} />;
}

function ShipmentCard({
  shipment,
  recordId,
  order,
  onSelect,
  onBack,
  backLabel,
}: Required<Pick<ShipmentCardProps, "order">> &
  Omit<ShipmentCardProps, "order"> & { shipment: Shipment }) {
  const t = useT();
  const userId = useAuthStore((state) => state.user?.id);
  const [focus, setFocus] = useState<string | null>(null);
  const { copy, isCopied } = useCopyToClipboard();

  const moves = shipment.moves;
  const stops = useMemo(() => journeyStops(moves), [moves]);
  const offsets = useMemo(() => stopOffsets(moves, stops), [moves, stops]);
  const position = truckOffset(stops, offsets, journeyProgress(moves, stops));
  const lane = laneOf(stops);
  const stage = shipmentStage(shipment);
  const eta = etaReading(shipment.eta, t);
  const reading = nowReading(shipment, stops, t);
  const coverage = useMemo(() => coverageOf(moves), [moves]);
  const freight = freightOf(shipment);
  const charges = chargesOf(shipment, t);
  const margin = marginOf(shipment);
  const miles = totalMiles(moves);
  const billing = billingPhase(shipment);
  const shipmentPath = recordPath("shipment", recordId);
  const neighbors = neighborsIn(order, recordId);

  const { data: readiness } = useQuery({
    ...queries.shipment.billingReadiness(recordId),
    enabled: billing === "readiness",
  });
  const { data: events } = useQuery(queries.shipment.events(recordId, ACTIVITY_ROWS));

  const title =
    lane.from && lane.to
      ? t("{0} → {1}", lane.from.label, lane.to.label)
      : (lane.from?.label ?? lane.to?.label ?? "");
  const customer = shipment.customer?.name ?? "";
  const billTo = shipment.billToCustomer;
  const billsElsewhere =
    !!billTo?.name &&
    (shipment.customer?.id ? billTo.id !== shipment.customer.id : billTo.name !== customer);
  const subtitle = [customer, billsElsewhere ? t("bills to {0}", billTo?.name ?? "") : ""]
    .filter(Boolean)
    .join(" · ");
  const milesText = miles !== null ? t("{0} mi", formatNumber(Math.round(miles))) : "";
  const hasFreight = freight.lines.length > 0 || freight.pieces !== null || freight.weight !== null;
  const history = useMemo(
    () => [...(events ?? [])].sort((a, b) => a.occurredAt - b.occurredAt),
    [events],
  );

  return (
    <RecordCard className="dk-sh">
      <CardHead
        onBack={onBack}
        backLabel={backLabel ?? t("Back to the table")}
        kicker={[
          shipment.proNumber || t("Shipment"),
          ...(shipment.bol ? [t("BOL {0}", shipment.bol)] : []),
          ...(shipment.entryMethod === "EDI" ? [t("EDI")] : []),
        ]}
        stepper={
          onSelect &&
          order.length > 1 && (
            <CardStepper
              neighbors={neighbors}
              onStep={onSelect}
              previousLabel={t("Previous shipment")}
              nextLabel={t("Next shipment")}
            />
          )
        }
        title={title || shipment.proNumber || t("Shipment")}
        subtitle={subtitle || undefined}
        figure={
          <>
            <b key={String(shipment.totalChargeAmount)}>{money(shipment.totalChargeAmount)}</b>
            <span>
              {margin && margin.percent !== null
                ? t("{0} margin", formatPercent(margin.percent))
                : t("total")}
            </span>
          </>
        }
        meta={
          <>
            <CardStatus
              key={stage}
              tone={STAGE_TONE[stage]}
              mark={
                stage === "Delivered"
                  ? "check"
                  : stage === "Late" || stage === "NeedsCoverage"
                    ? "pause"
                    : "dot"
              }
            >
              {stageLabel(stage, t)}
            </CardStatus>
            <CardFacts>
              {[
                eta && UNDER_WAY.has(stage) ? (
                  <span key="eta" className={cn("dk-sh-etaf", `dk-${eta.state}`)}>
                    {eta.detail ? t("{0}, {1}", eta.label, eta.detail) : eta.label}
                  </span>
                ) : null,
                milesText,
                freight.weight !== null ? t("{0} lb", formatNumber(freight.weight)) : "",
                shipment.owner?.name ?? "",
              ]}
            </CardFacts>
          </>
        }
      />

      <div className="dk-sh-nowsec">
        <NowStrip
          stops={stops}
          offsets={offsets}
          position={position}
          from={lane.from}
          to={lane.to}
          live={stage === "Moving"}
          closed={stage === "Canceled"}
          reading={reading}
          eta={eta}
          focus={focus}
          onFocus={setFocus}
        />
      </div>

      {stops.length > 0 && (
        <CardSection
          title={t("Route")}
          aside={
            <span>
              {[t("{0, plural, one {# stop} other {# stops}}", stops.length), milesText]
                .filter(Boolean)
                .join(" · ")}
            </span>
          }
        >
          <StopRail stops={stops} coverage={coverage} focus={focus} onFocus={setFocus} />
        </CardSection>
      )}

      {coverage.length > 0 && (
        <CardSection
          title={t("Coverage")}
          aside={
            shipment.tenderStatus ? (
              <span>{tenderStatusLabel(shipment.tenderStatus, t)}</span>
            ) : undefined
          }
        >
          <div className="dk-sh-cover">
            {coverage.map((entry) => (
              <CoverageRow key={entry.moveId} coverage={entry} numbered={coverage.length > 1} />
            ))}
          </div>
        </CardSection>
      )}

      {hasFreight && (
        <CardSection title={t("Freight")}>
          <FreightBlock freight={freight} />
        </CardSection>
      )}

      <CardSection title={t("Charges")}>
        <ChargesBlock charges={charges} />
        {margin && <MarginMeter margin={margin} />}
      </CardSection>

      {billing === "readiness" && readiness && (
        <BillingReadiness
          checks={readinessChecks(readiness, t)}
          ready={readiness.canMarkReadyToInvoice}
        />
      )}

      {billing === "invoiced" && (
        <CardSection title={t("Billing")}>
          <p className="dk-sh-note dk-sh-billed">
            <CardIcon name="check" size={12} stroke={2.6} />
            {shipment.billedAt ? t("Invoiced {0}", cardDay(shipment.billedAt)) : t("Invoiced")}
          </p>
        </CardSection>
      )}

      {history.length > 0 && (
        <CardSection
          title={t("Activity")}
          aside={<CardLink to={shipmentPath}>{t("All activity")}</CardLink>}
        >
          <CardTimeline
            entries={history.map((event) => ({
              id: event.id,
              text: event.summary,
              meta: [event.actor?.name || event.actorLabel, cardMoment(event.occurredAt)]
                .filter(Boolean)
                .join(" · "),
              mine: userId !== undefined && event.actorId === userId,
            }))}
          />
        </CardSection>
      )}

      <CardDocuments resourceType="shipment" resourceId={recordId} manageTo={shipmentPath} />

      <CardBar>
        <CardBarHint>{t("Assign, edit and track it on the shipment board")}</CardBarHint>
        {shipment.proNumber && (
          <Button
            variant="quiet"
            size="sm"
            className="text-dsk-subtle"
            onClick={() => void copy(shipment.proNumber ?? "")}
          >
            <CardIcon name={isCopied ? "check" : "copy"} size={12} />
            {isCopied ? t("Copied") : t("Copy PRO")}
          </Button>
        )}
        <Link className="dk-ax-btn dk-ink" to={shipmentPath}>
          <CardIcon name="ext" size={13} />
          {t("Open shipment")}
        </Link>
      </CardBar>
    </RecordCard>
  );
}

/**
 * What stands between a delivered shipment and its invoice: each document the
 * customer needs, each rule the charges break, and anything left to settle.
 */
function BillingReadiness({
  checks,
  ready,
}: {
  checks: readonly ReadinessCheck[];
  ready: boolean;
}) {
  const t = useT();
  const blocking = checks.filter((check) => check.state === "bad").length;
  const open = blocking || checks.filter((check) => check.state !== "ok").length;
  return (
    <CardSection
      title={t("Billing")}
      aside={
        ready ? (
          <span className="dk-ok">{t("Ready to bill")}</span>
        ) : open > 0 ? (
          <span className="dk-warn">{t("{0} to clear", open)}</span>
        ) : (
          <span className="dk-warn">{t("Not ready to bill")}</span>
        )
      }
    >
      {checks.length > 0 ? (
        <CardChecks>
          {checks.map((check) => (
            <CardCheck
              key={check.key}
              state={check.state}
              title={check.title}
              detail={check.detail}
            />
          ))}
        </CardChecks>
      ) : (
        <p className="dk-sh-note">{t("Nothing is required before it bills.")}</p>
      )}
    </CardSection>
  );
}
