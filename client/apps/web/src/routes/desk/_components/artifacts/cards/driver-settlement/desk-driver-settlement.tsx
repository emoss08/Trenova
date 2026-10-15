import { recordPath } from "@/config/record-links";
import { useApiMutation } from "@/hooks/use-api-mutation";
import { useNowSeconds } from "@/hooks/use-now-seconds";
import {
  approveDriverSettlement,
  invalidateDriverWorkspace,
  postDriverSettlement,
  recalculateDriverSettlement,
  submitDriverSettlement,
} from "@/lib/graphql/driver-settlement";
import type { DeskDriverSettlement as DeskDriverSettlementRecord } from "@/lib/graphql/settlement-card";
import { settlement } from "@/lib/queries/settlement";
import {
  MarkPaidDialog,
  ReasonDialog,
  type ReasonAction,
} from "@/routes/driver-settlement/_components/settlement-detail";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { useT } from "@trenova/shared/i18n/use-t";
import { cn } from "@trenova/shared/lib/utils";
import { Resource } from "@trenova/shared/types/permission";
import { useMemo, useState } from "react";
import { toast } from "sonner";
import { CardActivity } from "../../record-card/card-activity";
import { useCardAccess } from "../../record-card/card-access";
import {
  SettlementBar,
  SettlementSignoffs,
  useSettlementSteps,
  SettlementStatement,
  SettlementStub,
  SettlementVoidSlip,
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
  disputeCases,
  headFacts,
  lineTotal,
  settlementFigure,
  settlementFlags,
  settlementKicker,
  settlementSubtitle,
  statementGroups,
  waterfall,
  workerName,
} from "./driver-settlement-card";
import { PayWaterfall, SettlementDisputes, SettlementFlags } from "./driver-settlement-parts";
import { DRIVER_SETTLEMENT_SECTIONS } from "./driver-settlement-sections";
import {
  DeskDriverSettlementSkeleton,
  DRIVER_SETTLEMENT_ACTIVITY_ROWS,
  DRIVER_SETTLEMENT_SHIPMENT_ROWS,
} from "./driver-settlement-skeleton";

/**
 * What the card is given: the settlement, and when it was opened from a table
 * or from a card that lists it, the rows in the order shown, how to step to
 * another, and the way back.
 */
export type DriverSettlementCardProps = {
  recordId: string;
  order?: readonly string[];
  onSelect?: (recordId: string) => void;
  onBack?: () => void;
  /** What the way back says; the table, unless the settlement was opened from another card. */
  backLabel?: string;
};

const NO_ROWS: readonly string[] = [];

/**
 * One worker's settlement as the person who approves or pays it reads it,
 * opened from a table's row or looked up by the agent: who it pays and what it
 * comes to, where it stands on its way from draft to paid and who moved it,
 * the flags the run raised, how earnings, reimbursements and deductions add up
 * to the net pay (a shortfall carried to the next settlement hanging below
 * zero), the statement summed by how each line was paid, the worker's
 * disputes, the loads it pays with each one's share, and its activity. A
 * voided settlement says why.
 *
 * Its buttons are the settlement's own steps, taken by the person, never by
 * the agent, each under the permission the server checks: submit a draft,
 * approve or send back one waiting on approval, post an approved one to the
 * ledger, mark a posted one paid, recalculate a draft and void one not yet
 * paid. Adding pay by hand and the line-by-line statement stay on the
 * settlement's page.
 *
 * The card steps through the rows it was opened from in the order shown. A
 * step draws the next settlement's skeleton until it is read, never the
 * settlement it left with its buttons still live.
 */
export function DeskDriverSettlement({
  recordId,
  order: rowOrder = NO_ROWS,
  onSelect,
  onBack,
  backLabel,
}: DriverSettlementCardProps) {
  const t = useT();
  const query = useQuery({ ...settlement.driverCard(recordId), retry: false });
  const record = query.data;

  if (!record) {
    return query.isError ? (
      <CardUnreadable
        kind={t("Worker settlement")}
        title={t("This settlement could not be read")}
        error={query.error}
        onBack={onBack}
        backLabel={backLabel}
      />
    ) : (
      <DeskDriverSettlementSkeleton stepper={onSelect !== undefined && rowOrder.length > 1} />
    );
  }

  return (
    <DriverSettlementCard
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
export function DeskDriverSettlementArtifact({ recordId }: RecordArtifactProps) {
  return <DeskDriverSettlement recordId={recordId} />;
}

function DriverSettlementCard({
  record,
  rowOrder,
  onSelect,
  onBack,
  backLabel,
}: Omit<DriverSettlementCardProps, "order" | "recordId"> & {
  record: DeskDriverSettlementRecord;
  rowOrder: readonly string[];
}) {
  const t = useT();
  const now = useNowSeconds();
  const queryClient = useQueryClient();
  const access = useCardAccess(DRIVER_SETTLEMENT_SECTIONS);
  const may = useSettlementSteps(Resource.DriverSettlement);
  const id = record.id;
  const [reason, setReason] = useState<ReasonAction | null>(null);
  const [paying, setPaying] = useState(false);

  const shipments = useRecordShipments({
    scope: { kind: "driver_settlement", id },
    pageSize: DRIVER_SETTLEMENT_SHIPMENT_ROWS,
    enabled: access.shipments && record.shipmentCount > 0,
  });
  const activity = useRecordActivity({
    resourceId: id,
    pageSize: DRIVER_SETTLEMENT_ACTIVITY_ROWS,
    enabled: access.activity,
  });

  const changed = () => invalidateDriverWorkspace(queryClient);
  const step = useApiMutation({
    resourceName: "settlement",
    mutationFn: (action: "submit" | "approve" | "post" | "recalculate") => {
      const input = { settlementId: id };
      switch (action) {
        case "submit":
          return submitDriverSettlement(input);
        case "approve":
          return approveDriverSettlement(input);
        case "post":
          return postDriverSettlement(input);
        case "recalculate":
          return recalculateDriverSettlement(input);
      }
    },
    onSuccess: (_result, action) => {
      changed();
      toast.success(
        {
          submit: t("{0} submitted for approval", record.settlementNumber),
          approve: t("{0} approved", record.settlementNumber),
          post: t("{0} posted to the ledger", record.settlementNumber),
          recalculate: t("{0} recalculated from its pay events", record.settlementNumber),
        }[action],
      );
    },
  });

  const name = workerName(record, t);
  const backToSettlement = t("Back to {0}", record.settlementNumber);
  const listedShipments = useListedRecords({
    entity: "shipment",
    order: shipments.ids,
    backLabel: backToSettlement,
  });
  const workerId = record.worker && access.worker ? record.worker.id : null;
  const workerIds = useMemo(() => (workerId ? [workerId] : NO_ROWS), [workerId]);
  const listedWorker = useListedRecords({
    entity: "worker",
    order: workerIds,
    backLabel: backToSettlement,
  });
  const opened = listedShipments.opened ?? listedWorker.opened;
  if (opened) {
    return opened;
  }

  const standing = settlementStanding(record.status, t);
  const figure = settlementFigure(record, t);
  const steps = settlementTrack(record.status, t);
  const done = record.status === "Paid" || record.status === "Voided";
  const voided = record.status === "Voided";
  const flags = done ? [] : settlementFlags(record, t);
  const groups = statementGroups(record, t);
  const cases = access.disputes ? disputeCases(record, now, t) : [];
  const shares = new Map((record.loadShares ?? []).map((share) => [share.shipmentId, share]));
  const signed =
    record.status === "Draft" || record.status === "PendingApproval" ? [] : signoffs(record, t);
  const actions = settlementActions(record.status);
  const net = minor(record.netPayMinor, record.currencyCode);
  const busy = step.isPending;

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
    <RecordCard className="dk-pay">
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
        lead={<SettlementStub mark={t("PAY")} />}
        title={name}
        subtitle={settlementSubtitle(record, t)}
        figure={
          <>
            <b className={cn(figure.tone === "zero" && "dk-pay-zero")}>
              {figure.tone === "struck" ? <s>{figure.amount}</s> : figure.amount}
            </b>
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
          record.worker && access.worker ? (
            <CardOpenLink
              to={recordPath("worker", record.worker.id)}
              onOpen={listedWorker.open && (() => listedWorker.open?.(record.worker?.id ?? ""))}
            >
              {t("Open worker")}
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

      {flags.length > 0 && (
        <CardSection
          title={t("Review before approving")}
          aside={<span>{t("{0, plural, one {# flag} other {# flags}}", flags.length)}</span>}
        >
          <SettlementFlags flags={flags} />
        </CardSection>
      )}

      {groups.length > 0 && (
        <CardSection title={t("How it adds up")}>
          <PayWaterfall steps={waterfall(record, t)} currency={record.currencyCode} />
        </CardSection>
      )}

      <CardSection
        title={t("Pay")}
        aside={
          groups.length > 0 ? (
            <span>{t("{0, plural, one {# line} other {# lines}}", lineTotal(record))}</span>
          ) : undefined
        }
      >
        {groups.length > 0 ? (
          <SettlementStatement
            groups={groups}
            note={
              record.shipmentCount > 1
                ? t(
                    "Each row totals every line paid the same way. The statement lists each line by load.",
                  )
                : undefined
            }
          />
        ) : (
          <div className="dk-rec-empty-slip">
            <b>{t("Nothing to pay")}</b>
            <span>
              {t(
                "Loads {0} drives pay here once they are completed. Recurring pay and deductions join on the next run.",
                name,
              )}
            </span>
          </div>
        )}
      </CardSection>

      {cases.length > 0 && (
        <CardSection
          title={cases.length === 1 ? t("Dispute") : t("Disputes")}
          aside={
            cases.some((item) => item.open) ? (
              <span>{t("{0} open", cases.filter((item) => item.open).length)}</span>
            ) : undefined
          }
        >
          <SettlementDisputes cases={cases} />
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
        busy={busy}
        onAction={act}
        hint={
          done
            ? t("See the statement and its journal on its page")
            : t("Add pay by hand and read every line on its page")
        }
        pageTo={recordPath("driver_settlement", id)}
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
        settlementId={id}
        onOpenChange={setPaying}
        onChanged={changed}
      />
    </RecordCard>
  );
}
