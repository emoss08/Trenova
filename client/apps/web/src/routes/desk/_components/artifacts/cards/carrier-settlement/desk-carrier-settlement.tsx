import { recordPath } from "@/config/record-links";
import { useApiMutation } from "@/hooks/use-api-mutation";
import {
  approveCarrierSettlement,
  invalidateCarrierWorkspace,
  postCarrierSettlement,
  recalculateCarrierSettlement,
  submitCarrierSettlement,
} from "@/lib/graphql/carrier-settlement";
import type { DeskCarrierSettlement as DeskCarrierSettlementRecord } from "@/lib/graphql/settlement-card";
import { settlement } from "@/lib/queries/settlement";
import {
  MarkPaidDialog,
  ReasonDialog,
  type ReasonAction,
} from "@/routes/carrier-settlement/_components/settlement-detail";
import { keepPreviousData, useQuery, useQueryClient } from "@tanstack/react-query";
import { useT } from "@trenova/shared/i18n/use-t";
import { Resource } from "@trenova/shared/types/permission";
import { useMemo, useState } from "react";
import { toast } from "sonner";
import { CardActivity } from "../../record-card/card-activity";
import { useCardAccess } from "../../record-card/card-access";
import {
  SettlementBar,
  SettlementSignoffs,
  SettlementStatement,
  SettlementStub,
  SettlementVoidSlip,
  useSettlementSteps,
} from "../../record-card/card-settlement";
import { CardShipments } from "../../record-card/card-shipments";
import { CardTrack } from "../../record-card/card-track";
import {
  CardFacts,
  CardHead,
  CardOpenLink,
  CardSection,
  CardStatus,
  CardStepper,
  CardUnreadable,
  RecordCard,
} from "../../record-card/record-card";
import { cardMoment, neighborsIn } from "../../record-card/record-card-values";
import { useRecordActivity } from "../../record-card/record-activity";
import { useRecordShipments } from "../../record-card/record-shipments";
import {
  minor,
  reachedStep,
  settlementActionLabel,
  settlementActions,
  type SettlementAction,
  settlementStanding,
  settlementTrack,
  signoffs,
  trackNote,
} from "../../record-card/settlement-values";
import { useListedRecords, type RecordArtifactProps } from "../registry";
import {
  carrierName,
  costGroups,
  headFacts,
  lineTotal,
  moveRows,
  moveTally,
  offBy,
  remittance,
  settlementFigure,
  settlementKicker,
  settlementSubtitle,
} from "./carrier-settlement-card";
import { InvoiceReconciliation, Remittance } from "./carrier-settlement-parts";
import { CARRIER_SETTLEMENT_SECTIONS } from "./carrier-settlement-sections";
import {
  CARRIER_SETTLEMENT_ACTIVITY_ROWS,
  CARRIER_SETTLEMENT_SHIPMENT_ROWS,
  DeskCarrierSettlementSkeleton,
} from "./carrier-settlement-skeleton";

/**
 * What the card is given: the settlement, and when it was opened from a table
 * or from a card that lists it, the rows in the order shown, how to step to
 * another, and the way back.
 */
export type CarrierSettlementCardProps = {
  recordId: string;
  order?: readonly string[];
  onSelect?: (recordId: string) => void;
  onBack?: () => void;
  /** What the way back says; the table, unless the settlement was opened from another card. */
  backLabel?: string;
};

const NO_ROWS: readonly string[] = [];

/**
 * One carrier's settlement as the person who approves or pays the payable
 * reads it, opened from a table's row or looked up by the agent: the carrier
 * and what it is owed, where the payable stands on its way from draft to paid
 * and who moved it, each move's invoice against the buy rate (matched, off it,
 * waiting to be accepted, or not in yet), the costs summed by kind over the
 * adjustments, where the money goes, the loads it pays with each one's share,
 * and its activity. A voided settlement says why.
 *
 * Its buttons are the settlement's own steps, taken by the person, never by
 * the agent, each under the permission the server checks: submit a draft,
 * approve or send back one waiting on approval, post an approved one to the
 * ledger, mark a posted one paid, recalculate a draft and void one not yet
 * paid. Accepting an invoice's variance and adding an adjustment stay on the
 * settlement's page.
 *
 * The card steps through the rows it was opened from in the order shown.
 */
export function DeskCarrierSettlement({
  recordId,
  order: rowOrder = NO_ROWS,
  onSelect,
  onBack,
  backLabel,
}: CarrierSettlementCardProps) {
  const t = useT();
  const query = useQuery({
    ...settlement.carrierCard(recordId),
    retry: false,
    placeholderData: keepPreviousData,
  });
  const record = query.data;

  if (!record) {
    return query.isError ? (
      <CardUnreadable
        kind={t("Carrier settlement")}
        title={t("This settlement could not be read")}
        error={query.error}
        onBack={onBack}
        backLabel={backLabel}
      />
    ) : (
      <DeskCarrierSettlementSkeleton stepper={onSelect !== undefined && rowOrder.length > 1} />
    );
  }

  return (
    <CarrierSettlementCard
      key={record.id}
      record={record}
      rowOrder={rowOrder}
      onSelect={onSelect}
      onBack={onBack}
      backLabel={backLabel}
    />
  );
}

/**
 * The settlement as an artifact of its own, when the agent looked it up: the
 * same card, with nothing to step through and nowhere to go back to.
 */
export function DeskCarrierSettlementArtifact({ recordId }: RecordArtifactProps) {
  return <DeskCarrierSettlement recordId={recordId} />;
}

function CarrierSettlementCard({
  record,
  rowOrder,
  onSelect,
  onBack,
  backLabel,
}: Omit<CarrierSettlementCardProps, "order" | "recordId"> & {
  record: DeskCarrierSettlementRecord;
  rowOrder: readonly string[];
}) {
  const t = useT();
  const queryClient = useQueryClient();
  const access = useCardAccess(CARRIER_SETTLEMENT_SECTIONS);
  const may = useSettlementSteps(Resource.CarrierSettlement);
  const id = record.id;
  const [reason, setReason] = useState<ReasonAction | null>(null);
  const [paying, setPaying] = useState(false);

  const shipments = useRecordShipments({
    scope: { kind: "carrier_settlement", id },
    pageSize: CARRIER_SETTLEMENT_SHIPMENT_ROWS,
    enabled: access.shipments && record.shipmentCount > 0,
  });
  const activity = useRecordActivity({
    resourceId: id,
    pageSize: CARRIER_SETTLEMENT_ACTIVITY_ROWS,
    enabled: access.activity,
  });

  const changed = () => invalidateCarrierWorkspace(queryClient);
  const step = useApiMutation({
    resourceName: "settlement",
    mutationFn: (action: "submit" | "approve" | "post" | "recalculate") => {
      const input = { settlementId: id };
      switch (action) {
        case "submit":
          return submitCarrierSettlement(input);
        case "approve":
          return approveCarrierSettlement(input);
        case "post":
          return postCarrierSettlement(input);
        case "recalculate":
          return recalculateCarrierSettlement(input);
      }
    },
    onSuccess: (_result, action) => {
      changed();
      toast.success(
        {
          submit: t("{0} submitted for approval", record.settlementNumber),
          approve: t("{0} approved", record.settlementNumber),
          post: t("{0} posted to the ledger", record.settlementNumber),
          recalculate: t("{0} recalculated from its cost events", record.settlementNumber),
        }[action],
      );
    },
  });

  const name = carrierName(record, t);
  const backToSettlement = t("Back to {0}", record.settlementNumber);
  const listedShipments = useListedRecords({
    entity: "shipment",
    order: shipments.ids,
    backLabel: backToSettlement,
  });
  const carrierId = record.carrier && access.carrier ? record.carrier.id : null;
  const carrierIds = useMemo(() => (carrierId ? [carrierId] : NO_ROWS), [carrierId]);
  const listedCarrier = useListedRecords({
    entity: "carrier",
    order: carrierIds,
    backLabel: backToSettlement,
  });
  const opened = listedShipments.opened ?? listedCarrier.opened;
  if (opened) {
    return opened;
  }

  const standing = settlementStanding(record.status, t);
  const figure = settlementFigure(record, t);
  const steps = settlementTrack(record.status, t);
  const done = record.status === "Paid" || record.status === "Voided";
  const voided = record.status === "Voided";
  const groups = costGroups(record, t);
  const moves = record.moveInvoices ?? [];
  const tally = moveTally(record);
  const rows = done ? [] : moveRows(record, t);
  const remit = remittance(record, t);
  const shares = new Map((record.loadShares ?? []).map((share) => [share.shipmentId, share]));
  const signed =
    record.status === "Draft" || record.status === "PendingApproval" ? [] : signoffs(record, t);
  const actions = settlementActions(record.status);
  const net = minor(record.netPayableMinor, record.currencyCode);

  const act = (action: SettlementAction) => {
    switch (action) {
      case "reject":
      case "void":
        setReason(action);
        return;
      case "pay":
        setPaying(true);
        return;
      default:
        step.mutate(action);
    }
  };

  return (
    <RecordCard className="dk-cset">
      <CardHead
        onBack={onBack}
        backLabel={backLabel ?? t("Back to the table")}
        kicker={settlementKicker(record, t)}
        stepper={
          onSelect &&
          rowOrder.length > 1 && (
            <CardStepper
              neighbors={neighborsIn(rowOrder, id)}
              onStep={onSelect}
              previousLabel={t("Previous settlement")}
              nextLabel={t("Next settlement")}
            />
          )
        }
        lead={<SettlementStub mark={t("AP")} />}
        title={name}
        subtitle={settlementSubtitle(record, t)}
        figure={
          <>
            <b>{figure.struck ? <s>{figure.amount}</s> : figure.amount}</b>
            <span>{figure.caption}</span>
          </>
        }
        meta={
          <>
            <CardStatus key={standing.label} tone={standing.tone} mark={standing.mark}>
              {standing.label}
            </CardStatus>
            <CardFacts>{headFacts(record, t)}</CardFacts>
          </>
        }
      />

      {voided && (
        <CardSection title={t("Why it was voided")}>
          <SettlementVoidSlip
            reason={record.voidReason}
            by={record.voidedBy?.name || t("Someone")}
            when={cardMoment(record.voidedAt)}
            reached={reachedStep(record, t)}
          />
        </CardSection>
      )}

      <CardSection
        title={voided ? t("Where it stood") : t("Where it stands")}
        aside={
          record.carrier && access.carrier ? (
            <CardOpenLink
              to={recordPath("carrier", record.carrier.id)}
              onOpen={listedCarrier.open && (() => listedCarrier.open?.(record.carrier?.id ?? ""))}
            >
              {t("Open carrier")}
            </CardOpenLink>
          ) : undefined
        }
      >
        <CardTrack
          steps={steps}
          label={t("Settlement lifecycle")}
          off={voided}
          note={trackNote(record, t)}
        />
        <SettlementSignoffs rows={signed} />
      </CardSection>

      {moves.length > 0 && (
        <CardSection
          title={t("Carrier invoices")}
          aside={<span>{t("{0} of {1} matched", tally.matched, moves.length)}</span>}
        >
          <InvoiceReconciliation tally={tally} rows={rows} offBy={offBy(record, t)} />
        </CardSection>
      )}

      <CardSection
        title={t("Costs")}
        aside={
          groups.length > 0 ? (
            <span>{t("{0, plural, one {# line} other {# lines}}", lineTotal(record))}</span>
          ) : undefined
        }
      >
        {groups.length > 0 ? (
          <SettlementStatement
            groups={groups}
            foot={{ label: t("Net payable"), amount: net }}
            note={
              record.shipmentCount > 1
                ? t(
                    "Each row totals every cost of the same kind. The statement lists each cost by load.",
                  )
                : undefined
            }
          />
        ) : (
          <div className="dk-rec-empty-slip">
            <b>{t("Nothing payable")}</b>
            <span>
              {t("Costs on the loads {0} hauls land here once they are delivered and rated.", name)}
            </span>
          </div>
        )}
      </CardSection>

      {!voided && (
        <CardSection title={t("Where the money goes")}>
          <Remittance lines={remit.lines} method={remit.method} terms={remit.terms} />
        </CardSection>
      )}

      {access.shipments && record.shipmentCount > 0 && (
        <CardShipments
          title={t("Loads it pays")}
          shipments={shipments}
          onOpen={listedShipments.open}
          emptyText={t("No load is paid on this settlement.")}
          layout="route"
          figureOf={(shipmentId) => {
            const share = shares.get(shipmentId);
            return share ? t("pays {0}", minor(share.amountMinor, record.currencyCode)) : null;
          }}
        />
      )}

      {access.activity && (
        <CardActivity
          activity={activity}
          emptyText={t("Nothing has been done to this settlement yet.")}
          showChanges
        />
      )}

      <SettlementBar
        primary={actions.primary}
        more={actions.more}
        labelOf={(action) => settlementActionLabel(action, net, t)}
        may={may}
        busy={step.isPending}
        onAction={act}
        hint={
          done
            ? t("See the statement and its journal on its page")
            : t("Accept invoice variances and add adjustments on its page")
        }
        pageTo={recordPath("carrier_settlement", id)}
        pageLabel={t("Open settlement")}
      />

      <ReasonDialog
        action={reason}
        settlementId={id}
        onOpenChange={(open) => !open && setReason(null)}
        onChanged={changed}
      />
      <MarkPaidDialog
        open={paying}
        settlement={record}
        onOpenChange={setPaying}
        onChanged={changed}
      />
    </RecordCard>
  );
}
