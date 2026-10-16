import { recordPath } from "@/config/record-links";
import { useApiMutation } from "@/hooks/use-api-mutation";
import { useNowSeconds } from "@/hooks/use-now-seconds";
import { usePermission } from "@/hooks/use-permission";
import type { DeskDetention } from "@/lib/graphql/detention-card";
import { detention } from "@/lib/queries/detention";
import {
  DisputeDialog,
  WaiveDialog,
} from "@/routes/detention-desk/_components/occurrence-action-dialogs";
import {
  useInvalidateDetention,
  useSendDetentionNotice,
} from "@/routes/detention-desk/_components/use-detention-actions";
import { apiService } from "@/services/api";
import { useQuery } from "@tanstack/react-query";
import { useT } from "@trenova/shared/i18n/use-t";
import { formatDetentionMinutes } from "@trenova/shared/lib/detention";
import { cn } from "@trenova/shared/lib/utils";
import type { DetentionOccurrence } from "@trenova/shared/types/detention";
import { Operation, Resource } from "@trenova/shared/types/permission";
import { useMemo, useState } from "react";
import { Link } from "react-router";
import { CardActionButton } from "../../record-card/card-action";
import { useCardAccess } from "../../record-card/card-access";
import { SettlementStub } from "../../record-card/card-settlement";
import { CardShipments } from "../../record-card/card-shipments";
import { CardTrack } from "../../record-card/card-track";
import {
  CardBar,
  CardBarHint,
  CardFacts,
  CardHead,
  CardIcon,
  CardOpenLink,
  CardSection,
  CardStatus,
  CardStepper,
  CardUnreadable,
  RecordCard,
} from "../../record-card/record-card";
import { cardMoment, money, neighborsIn } from "../../record-card/record-card-values";
import { useRecordShipments } from "../../record-card/record-shipments";
import { useListedRecords, type RecordArtifactProps } from "../registry";
import {
  clockModel,
  type DetentionAction,
  detentionActions,
  detentionFacts,
  detentionFigure,
  detentionKicker,
  detentionStanding,
  detentionSubtitle,
  detentionTrack,
  detentionTrackNote,
  isOffTrack,
  noticeRows,
  noticeRule,
  noticeSlip,
  paySplit,
  placeLine,
  placeName,
  proofReading,
  receiptRows,
  settledSlip,
} from "./detention-occurrence-card";
import {
  DetentionClock,
  DetentionNotices,
  DetentionPaySplit,
  DetentionProof,
  DetentionReceipt,
  DetentionSettledSlip,
  DetentionStopRefs,
  type StopRef,
} from "./detention-occurrence-parts";
import { DETENTION_OCCURRENCE_SECTIONS } from "./detention-occurrence-sections";
import { DeskDetentionOccurrenceSkeleton } from "./detention-occurrence-skeleton";

/**
 * What the card is given: the detention charge, and when it was opened from a
 * table or from a card that lists it, the rows in the order shown, how to step
 * to another, and the way back.
 */
export type DetentionOccurrenceCardProps = {
  recordId: string;
  order?: readonly string[];
  onSelect?: (recordId: string) => void;
  onBack?: () => void;
  /** What the way back says; the table, unless the charge was opened from another card. */
  backLabel?: string;
};

const NO_ROWS: readonly string[] = [];

/**
 * One stop's detention charge as the biller who approves it reads it, opened
 * from a table's row or looked up by the agent: the facility and what the
 * charge comes to, or has come to so far while the truck is still there;
 * where it stands on its way from the dock to the invoice; the stop's clock
 * from the appointment to the departure; the calculation as the server
 * recorded it; what the customer is billed against what the worker is paid;
 * the notice the policy asks for; whether the charge would hold up to a
 * dispute; and the shipment, facility and customer it belongs to. A waived or
 * disputed charge says why.
 *
 * Its buttons are the detention desk's own, taken by the person, never by the
 * agent, under the permission the server checks: approve a charge waiting on
 * approval, send the notice while one is owed, record the customer's dispute,
 * and waive the charge. The policy and every other stop stay on the desk.
 *
 * The card steps through the rows it was opened from in the order shown. A
 * step draws the next charge's skeleton until it is read, never the charge it
 * left with its buttons still live.
 */
export function DeskDetentionOccurrence({
  recordId,
  order: rowOrder = NO_ROWS,
  onSelect,
  onBack,
  backLabel,
}: DetentionOccurrenceCardProps) {
  const t = useT();
  const query = useQuery({ ...detention.card(recordId), retry: false });
  const detail = query.data;

  if (!detail) {
    return query.isError ? (
      <CardUnreadable
        kind={t("Detention charge")}
        title={t("This detention charge could not be read")}
        error={query.error}
        onBack={onBack}
        backLabel={backLabel}
      />
    ) : (
      <DeskDetentionOccurrenceSkeleton stepper={onSelect !== undefined && rowOrder.length > 1} />
    );
  }

  return (
    <DetentionOccurrenceCard
      key={detail.occurrence.id}
      detail={detail}
      rowOrder={rowOrder}
      onSelect={onSelect}
      onBack={onBack}
      backLabel={backLabel}
    />
  );
}

/**
 * The charge as an artifact of its own, when the agent looked it up: the same
 * card, with nothing to step through and nowhere to go back to.
 */
export function DeskDetentionOccurrenceArtifact({ recordId }: RecordArtifactProps) {
  return <DeskDetentionOccurrence recordId={recordId} />;
}

function DetentionOccurrenceCard({
  detail,
  rowOrder,
  onSelect,
  onBack,
  backLabel,
}: Omit<DetentionOccurrenceCardProps, "order" | "recordId"> & {
  detail: DeskDetention;
  rowOrder: readonly string[];
}) {
  const t = useT();
  const now = useNowSeconds();
  const access = useCardAccess(DETENTION_OCCURRENCE_SECTIONS);
  const { allowed: mayAct } = usePermission(Resource.Detention, Operation.Update);
  const invalidate = useInvalidateDetention();
  const { occurrence } = detail;
  const id = occurrence.id;
  const [waiving, setWaiving] = useState(false);
  const [disputing, setDisputing] = useState(false);

  const approve = useApiMutation<DetentionOccurrence, undefined>({
    mutationFn: () => apiService.detentionService.approve(id),
    onSuccess: () => {
      invalidate();
    },
    resourceName: "Detention Occurrence",
  });
  const sendNotice = useSendDetentionNotice(id);

  const shipments = useRecordShipments({
    scope: { kind: "shipment", id: occurrence.shipmentId },
    pageSize: 1,
    enabled: access.shipments && occurrence.shipmentId !== "",
  });

  const place = placeName(occurrence, t);
  const backToCharge = t("Back to the charge at {0}", place);
  const listedShipments = useListedRecords({
    entity: "shipment",
    order: shipments.ids,
    backLabel: backToCharge,
  });
  const locationIds = useMemo(
    () => (access.location && occurrence.locationId ? [occurrence.locationId] : NO_ROWS),
    [access.location, occurrence.locationId],
  );
  const listedLocation = useListedRecords({
    entity: "location",
    order: locationIds,
    backLabel: backToCharge,
  });
  const customerIds = useMemo(
    () => (access.customer && occurrence.customerId ? [occurrence.customerId] : NO_ROWS),
    [access.customer, occurrence.customerId],
  );
  const listedCustomer = useListedRecords({
    entity: "customer",
    order: customerIds,
    backLabel: backToCharge,
  });
  const opened = listedShipments.opened ?? listedLocation.opened ?? listedCustomer.opened;
  if (opened) {
    return opened;
  }

  const standing = detentionStanding(occurrence.status, t);
  const figure = detentionFigure(detail, now, t);
  const off = isOffTrack(occurrence.status);
  const clock = clockModel(detail, now, t);
  const receipt = clock.live ? null : receiptRows(occurrence);
  const split = paySplit(occurrence, t);
  const rule = noticeRule(occurrence, t);
  const slip = noticeSlip(occurrence, now, t);
  const notices = noticeRows(detail.notices, t);
  const proof = proofReading(detail, t);
  const settled = settledSlip(occurrence, t);
  const actions = detentionActions(occurrence);
  const requirement = occurrence.snapshot?.notificationRequirement ?? "None";
  const amount = money(occurrence.billableAmount, occurrence.currency);
  const busy = approve.isPending || sendNotice.isPending;

  const labelOf = (action: DetentionAction): string => {
    switch (action) {
      case "approve":
        return t("Approve {0}", amount);
      case "notice":
        return t("Send notice");
      case "dispute":
        return t("Record dispute");
      case "waive":
        return t("Waive");
    }
  };
  const act = (action: DetentionAction) => {
    switch (action) {
      case "approve":
        approve.mutate(undefined);
        return;
      case "notice":
        sendNotice.mutate(undefined);
        return;
      case "dispute":
        setDisputing(true);
        return;
      case "waive":
        setWaiving(true);
    }
  };
  const leading = mayAct ? actions.primary : null;

  const refs: StopRef[] = [];
  if (occurrence.locationId) {
    refs.push({
      key: "location",
      kind: t("Location"),
      name: place,
      line: placeLine(occurrence),
      to: access.location ? recordPath("location", occurrence.locationId) : null,
      onOpen: listedLocation.open && (() => listedLocation.open?.(occurrence.locationId)),
    });
  }
  if (occurrence.customerId) {
    refs.push({
      key: "customer",
      kind: t("Customer"),
      name: occurrence.customerName.trim() || t("Unknown customer"),
      line: "",
      to: access.customer ? recordPath("customer", occurrence.customerId) : null,
      onOpen: listedCustomer.open && (() => listedCustomer.open?.(occurrence.customerId)),
    });
  }

  return (
    <RecordCard className="dk-dtn">
      <CardHead
        onBack={onBack}
        backLabel={backLabel ?? t("Back to the table")}
        kicker={detentionKicker(occurrence, t)}
        stepper={
          onSelect &&
          rowOrder.length > 1 && (
            <CardStepper
              neighbors={neighborsIn(rowOrder, id)}
              onStep={onSelect}
              previousLabel={t("Previous charge")}
              nextLabel={t("Next charge")}
            />
          )
        }
        lead={<SettlementStub mark={t("DET")} />}
        title={place}
        subtitle={detentionSubtitle(occurrence, t)}
        figure={
          <>
            <b className={cn(figure.quiet && "dk-dtn-quiet")}>
              {figure.struck ? <s>{figure.amount}</s> : figure.amount}
            </b>
            <span>{figure.caption}</span>
          </>
        }
        meta={
          <>
            <CardStatus key={standing.label} tone={standing.tone} mark={standing.mark}>
              {standing.label}
            </CardStatus>
            <CardFacts>{detentionFacts(detail, now, t)}</CardFacts>
          </>
        }
      />

      {settled && (
        <CardSection title={settled.title}>
          <DetentionSettledSlip note={settled.note} facts={settled.facts} />
        </CardSection>
      )}

      <CardSection
        title={occurrence.status === "Waived" ? t("Where it stood") : t("Where it stands")}
        aside={
          access.shipments && occurrence.shipmentId ? (
            <CardOpenLink
              to={recordPath("shipment", occurrence.shipmentId)}
              onOpen={listedShipments.open && (() => listedShipments.open?.(occurrence.shipmentId))}
            >
              {t("Open shipment")}
            </CardOpenLink>
          ) : undefined
        }
      >
        <CardTrack
          steps={detentionTrack(occurrence.status, t)}
          label={t("Detention lifecycle")}
          off={off}
          note={detentionTrackNote(occurrence, now, t)}
        />
      </CardSection>

      <CardSection
        title={t("The clock")}
        aside={
          <span>
            {clock.live
              ? t("{0} at the dock and counting", formatDetentionMinutes(clock.dwellMinutes))
              : t("{0} at the dock", formatDetentionMinutes(clock.dwellMinutes))}
          </span>
        }
      >
        <DetentionClock model={clock} />
      </CardSection>

      {receipt && (
        <CardSection
          title={t("How it was worked out")}
          aside={
            <span>{t("Engine {0} · {1}", receipt.engine, cardMoment(receipt.computedAt))}</span>
          }
        >
          <DetentionReceipt rows={receipt.rows} total={receipt.total ?? amount} />
        </CardSection>
      )}

      {!clock.live && split.bill > 0 && (
        <CardSection title={t("Who pays, who's paid")}>
          <DetentionPaySplit {...split} currency={occurrence.currency} />
        </CardSection>
      )}

      {(requirement !== "None" || notices.length > 0) && (
        <CardSection
          title={t("Notice to the customer")}
          aside={
            requirement === "Required" ? (
              <span>{t("Required")}</span>
            ) : requirement === "Advisory" ? (
              <span>{t("Advisory")}</span>
            ) : undefined
          }
        >
          <DetentionNotices slip={slip} rule={rule} rows={notices} />
        </CardSection>
      )}

      <CardSection title={t("Will it hold up")} aside={<span>{proof.aside}</span>}>
        <DetentionProof
          score={proof.score}
          band={proof.band}
          weak={proof.weak}
          factors={proof.factors}
          events={proof.events}
          chain={proof.chain}
          chainValid={proof.chainValid}
        />
      </CardSection>

      {access.shipments && occurrence.shipmentId && (
        <CardShipments
          title={t("The shipment")}
          shipments={shipments}
          onOpen={listedShipments.open}
          emptyText={t("The shipment could not be found.")}
          layout="route"
        />
      )}

      {refs.length > 0 && (
        <CardSection title={t("The stop")}>
          <DetentionStopRefs refs={refs} />
        </CardSection>
      )}

      <CardBar>
        {mayAct &&
          actions.more.map((action) => (
            <CardActionButton
              key={action}
              variant="quiet"
              className="dk-rec-step"
              blocker={null}
              busy={busy}
              onClick={() => act(action)}
            >
              {action === "waive" && <CardIcon name="x" size={12} stroke={2.4} />}
              {labelOf(action)}
            </CardActionButton>
          ))}
        <CardBarHint>
          {occurrence.status === "Waived"
            ? t("See the waiver against the facility's other stops on the detention desk")
            : t("The policy and every stop are on the detention desk")}
        </CardBarHint>
        <Link
          className={cn("dk-ax-btn", leading === null && "dk-ink")}
          to={recordPath("detention_occurrence", id)}
        >
          <CardIcon name="ext" size={13} />
          {t("Open in detention desk")}
        </Link>
        {leading !== null && (
          <CardActionButton blocker={null} busy={busy} onClick={() => act(leading)}>
            {labelOf(leading)}
          </CardActionButton>
        )}
      </CardBar>

      <WaiveDialog occurrenceId={id} open={waiving} onOpenChange={setWaiving} onDone={invalidate} />
      <DisputeDialog
        occurrenceId={id}
        open={disputing}
        onOpenChange={setDisputing}
        onDone={invalidate}
      />
    </RecordCard>
  );
}
