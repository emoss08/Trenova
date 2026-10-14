import { recordPath } from "@/config/record-links";
import { counterparty, type TransferDirection, transferDirection } from "@/lib/edi/transfer-sides";
import { edi } from "@/lib/queries/edi";
import { keepPreviousData, useQuery } from "@tanstack/react-query";
import { formatNumber } from "@trenova/shared/i18n/format";
import { useT } from "@trenova/shared/i18n/use-t";
import { useAuthStore } from "@trenova/shared/stores/auth-store";
import type { EDITransfer, EDITransferStatus } from "@trenova/shared/types/edi";
import { useMemo } from "react";
import { Link } from "react-router";
import { useCardAccess } from "../../record-card/card-access";
import {
  CardBar,
  CardBarHint,
  CardFacts,
  CardHead,
  CardIcon,
  CardRecordRow,
  CardSection,
  CardStatus,
  CardStepper,
  CardUnreadable,
  RecordCard,
} from "../../record-card/record-card";
import { cardMoment, neighborsIn } from "../../record-card/record-card-values";
import { useListedRecords, type RecordArtifactProps } from "../registry";
import {
  decisionGates,
  mappingLedger,
  reasonNote,
  tenderCharges,
  tenderFreight,
  tenderRoute,
  tenderSlip,
  type TransferLink,
  transferLinks,
} from "./edi-transfer-card";
import { transferStanding } from "../../record-card/edi-transfer-standing";
import {
  DecisionGates,
  FreightAndRate,
  MappingLedgerView,
  ReasonSlip,
  TenderRoute,
  TenderTicket,
} from "./edi-transfer-parts";
import { EDI_TRANSFER_SECTIONS } from "./edi-transfer-sections";
import { DeskEdiTransferSkeleton } from "./edi-transfer-skeleton";

/**
 * What the card is given: the tender, and when it was opened from a table or
 * from a card that lists it, the rows in the order shown, how to step to
 * another, and the way back.
 */
export type EdiTransferCardProps = {
  recordId: string;
  order?: readonly string[];
  onSelect?: (recordId: string) => void;
  onBack?: () => void;
  /** What the way back says; the table, unless the tender was opened from another card. */
  backLabel?: string;
};

const NO_ROWS: readonly string[] = [];

/** The statuses a tender can still be accepted or declined in. */
const OPEN_TO_DECISION: ReadonlySet<EDITransferStatus> = new Set([
  "Submitted",
  "MappingRequired",
  "PendingApproval",
]);

/** The page a tender we sent is listed on; the record link opens the inbound one. */
const OUTBOUND_TRANSFERS_PAGE = "/edi/transfers/outbound";

/**
 * One load tender as the EDI coordinator or dispatcher deciding on it reads
 * it, opened from a table's row or looked up by the agent: who tendered it to
 * whom, how far it has come (received, mapped, decided, built) and why it
 * stopped where it did, its stops with their windows, the freight and what it
 * pays, which of the partner's codes are mapped to our records, and the 204
 * and shipments it names. It reads; accepting and declining stay on the
 * transfers page, one click away.
 *
 * The card steps through the rows it was opened from in the order shown.
 */
export function DeskEdiTransfer({
  recordId,
  order: rowOrder = NO_ROWS,
  onSelect,
  onBack,
  backLabel,
}: EdiTransferCardProps) {
  const t = useT();
  const query = useQuery({
    ...edi.transfer(recordId),
    retry: false,
    placeholderData: keepPreviousData,
  });
  const transfer = query.data;

  if (!transfer) {
    return query.isError ? (
      <CardUnreadable
        kind={t("EDI transfer")}
        title={t("This EDI transfer could not be read")}
        error={query.error}
        onBack={onBack}
        backLabel={backLabel}
      />
    ) : (
      <DeskEdiTransferSkeleton stepper={onSelect !== undefined && rowOrder.length > 1} />
    );
  }

  return (
    <EdiTransferCard
      key={transfer.id}
      transfer={transfer}
      rowOrder={rowOrder}
      onSelect={onSelect}
      onBack={onBack}
      backLabel={backLabel}
    />
  );
}

/**
 * The tender as an artifact of its own, when the agent looked it up: the same
 * card, with nothing to step through and nowhere to go back to.
 */
export function DeskEdiTransferArtifact({ recordId }: RecordArtifactProps) {
  return <DeskEdiTransfer recordId={recordId} />;
}

/** What the bar says of where the tender's decision is made. */
function barHint(
  transfer: EDITransfer,
  direction: TransferDirection,
  partner: string,
  t: ReturnType<typeof useT>,
): string {
  if (!OPEN_TO_DECISION.has(transfer.status)) {
    return transfer.targetShipmentId
      ? t("Its history and the shipment link are on the transfers page")
      : t("Its history is on the transfers page");
  }
  if (direction === "outbound") {
    return partner ? t("{0} decides on their side", partner) : t("The other side decides");
  }
  return transfer.status === "MappingRequired"
    ? t("Map its codes and decide on the inbound transfers page")
    : t("Accept or decline it on the inbound transfers page");
}

function EdiTransferCard({
  transfer,
  rowOrder,
  onSelect,
  onBack,
  backLabel,
}: Omit<EdiTransferCardProps, "order" | "recordId"> & {
  transfer: EDITransfer;
  rowOrder: readonly string[];
}) {
  const t = useT();
  const access = useCardAccess(EDI_TRANSFER_SECTIONS);
  const organizationId = useAuthStore((state) => state.user?.currentOrganizationId);
  const id = transfer.id;
  const direction = transferDirection(transfer, organizationId);
  const open = OPEN_TO_DECISION.has(transfer.status);
  const preview = useQuery({
    ...edi.mappingPreview(id),
    retry: false,
    enabled: direction === "inbound" && open,
  }).data;

  const partner = counterparty(transfer, direction);
  const standing = transferStanding(transfer, direction, t);
  const slip = tenderSlip(transfer, t);
  const neighbors = neighborsIn(rowOrder, id);
  const route = tenderRoute(transfer, preview, t);
  const freight = tenderFreight(transfer, t);
  const charges = tenderCharges(transfer, t);
  const ledger = mappingLedger(transfer, preview);
  const reason = reasonNote(transfer, t);
  const links = transferLinks(transfer, direction, t).filter(
    (link) => link.entity !== "shipment" || access.shipment,
  );

  const backToTender = t("Back to {0}", slip.title);
  const shipmentIds = useIds(links, "shipment");
  const messageIds = useIds(links, "edi_message");
  const listedShipments = useListedRecords({
    entity: "shipment",
    order: shipmentIds,
    backLabel: backToTender,
  });
  const listedMessages = useListedRecords({
    entity: "edi_message",
    order: messageIds,
    backLabel: backToTender,
  });

  // A record opened from the card takes the card's place, and its way back
  // returns here.
  const opened = listedShipments.opened ?? listedMessages.opened;
  if (opened) {
    return opened;
  }

  const openLink = (link: TransferLink) =>
    link.entity === "shipment"
      ? listedShipments.open && (() => listedShipments.open?.(link.id))
      : listedMessages.open && (() => listedMessages.open?.(link.id));

  const stops = route.stops.length;

  return (
    <RecordCard className={`dk-tdr dk-tdr-${direction}`}>
      <CardHead
        onBack={onBack}
        backLabel={backLabel ?? t("Back to the table")}
        kicker={[
          direction === "inbound" ? t("Inbound tender") : t("Outbound tender"),
          partner?.code,
          slip.purpose,
        ].filter(Boolean)}
        stepper={
          onSelect &&
          rowOrder.length > 1 && (
            <CardStepper
              neighbors={neighbors}
              onStep={onSelect}
              previousLabel={t("Previous tender")}
              nextLabel={t("Next tender")}
            />
          )
        }
        lead={<TenderTicket direction={direction} />}
        title={slip.title}
        subtitle={
          partner
            ? direction === "inbound"
              ? t("From {0}", partner.name)
              : t("To {0}", partner.name)
            : t("Partner not found")
        }
        figure={charges.total || undefined}
        meta={
          <>
            <CardStatus key={standing.label} tone={standing.tone} mark={standing.mark}>
              {standing.label}
            </CardStatus>
            <CardFacts>
              {[
                cardMoment(transfer.submittedAt),
                stops > 0 ? t("{0, plural, one {# stop} other {# stops}}", stops) : "",
                route.distance,
              ]}
            </CardFacts>
          </>
        }
      />

      <CardSection title={t("Decision")}>
        <DecisionGates gates={decisionGates(transfer, direction, t, preview)} />
        {reason && <ReasonSlip note={reason} />}
      </CardSection>

      <CardSection
        title={t("Route")}
        aside={route.distance ? <span className="dk-tdr-aside">{route.distance}</span> : undefined}
      >
        {stops > 0 ? (
          <TenderRoute stops={route.stops} />
        ) : (
          <p className="dk-rec-note">{t("The tender names no stops.")}</p>
        )}
      </CardSection>

      <CardSection title={t("Freight and rate")}>
        <FreightAndRate
          facts={freight.facts}
          commodities={freight.commodities}
          lines={charges.lines}
          total={charges.total}
        />
      </CardSection>

      {ledger.total > 0 && (
        <CardSection
          title={t("Mappings")}
          aside={
            <span className="dk-tdr-aside">
              {t("{0} of {1} mapped", formatNumber(ledger.resolved), formatNumber(ledger.total))}
            </span>
          }
        >
          <MappingLedgerView ledger={ledger} />
        </CardSection>
      )}

      {links.length > 0 && (
        <CardSection title={t("Records it names")}>
          <div className="dk-rec-list">
            {links.map((link) => (
              <CardRecordRow
                key={link.key}
                to={recordPath(link.entity, link.id)}
                onOpen={openLink(link)}
              >
                <span className="dk-rec-main">
                  <b>{link.title}</b>
                  <span className="dk-rec-sub">{link.line}</span>
                </span>
                <CardIcon name="forward" size={12} />
              </CardRecordRow>
            ))}
          </div>
        </CardSection>
      )}

      <CardBar>
        <CardBarHint>{barHint(transfer, direction, partner?.name ?? "", t)}</CardBarHint>
        {direction === "inbound" ? (
          <Link className="dk-ax-btn dk-ink" to={recordPath("edi_transfer", id)}>
            <CardIcon name="ext" size={13} />
            {t("Open tender")}
          </Link>
        ) : (
          <Link className="dk-ax-btn dk-ink" to={OUTBOUND_TRANSFERS_PAGE}>
            <CardIcon name="ext" size={13} />
            {t("Open outbound transfers")}
          </Link>
        )}
      </CardBar>
    </RecordCard>
  );
}

/** The ids of the listed records of one kind, held still while the list is. */
function useIds(links: readonly TransferLink[], entity: TransferLink["entity"]): readonly string[] {
  return useMemo(() => {
    const ids = links.filter((link) => link.entity === entity).map((link) => link.id);
    return ids.length > 0 ? ids : NO_ROWS;
  }, [links, entity]);
}
