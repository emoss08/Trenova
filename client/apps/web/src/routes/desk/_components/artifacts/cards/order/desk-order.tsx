import { recordPath } from "@/config/record-links";
import type { DeskOrder as DeskOrderRecord } from "@/lib/graphql/order-card";
import { order } from "@/lib/queries/order";
import { useQuery } from "@tanstack/react-query";
import { useT } from "@trenova/shared/i18n/use-t";
import { cn } from "@trenova/shared/lib/utils";
import { useMemo } from "react";
import { Link } from "react-router";
import { CardActivity } from "../../record-card/card-activity";
import { useCardAccess } from "../../record-card/card-access";
import { CardShipments } from "../../record-card/card-shipments";
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
import { CardTrack } from "../../record-card/card-track";
import { cardMoment, money, neighborsIn } from "../../record-card/record-card-values";
import { useRecordActivity } from "../../record-card/record-activity";
import { useRecordShipments } from "../../record-card/record-shipments";
import { useListedRecords, type RecordArtifactProps } from "../registry";
import { lifecycleSteps, orderFacts, orderStanding, orderWorth } from "./order-card";
import { ChargeLedger, OrderReferences } from "./order-parts";
import { ORDER_SECTIONS } from "./order-sections";
import { DeskOrderSkeleton } from "./order-skeleton";

/** The legs the card reads at a time; the next page is read on request. */
const SHIPMENT_ROWS = 5;
/** The activity entries the card reads at a time. */
const ACTIVITY_ROWS = 5;

/**
 * What the card is given: the order, and when it was opened from a table or
 * from a card that lists it, the rows in the order shown, how to step to
 * another, and the way back.
 */
export type OrderCardProps = {
  recordId: string;
  order?: readonly string[];
  onSelect?: (recordId: string) => void;
  onBack?: () => void;
  /** What the way back says; the table, unless the order was opened from another card. */
  backLabel?: string;
};

const NO_ROWS: readonly string[] = [];

/**
 * One order as a customer service rep or a biller reads it, opened from a
 * table's row or looked up by the agent: who it is for and what it is worth,
 * how far along its way it has come, the shipments that carry it, its charges
 * against what has been invoiced, and what has been done to it. It reads;
 * changing, splitting and billing the order stay on its page, one click away.
 *
 * The card steps through the rows it was opened from in the order shown.
 * A step draws the next order's skeleton until it is read, never the one it left.
 */
export function DeskOrder({
  recordId,
  order: rowOrder = NO_ROWS,
  onSelect,
  onBack,
  backLabel,
}: OrderCardProps) {
  const t = useT();
  const query = useQuery({ ...order.card(recordId), retry: false });
  const record = query.data;

  if (!record) {
    return query.isError ? (
      <CardUnreadable
        kind={t("Order")}
        title={t("This order could not be read")}
        error={query.error}
        onBack={onBack}
        backLabel={backLabel}
      />
    ) : (
      <DeskOrderSkeleton stepper={onSelect !== undefined && rowOrder.length > 1} />
    );
  }

  return (
    <OrderCard
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
 * The order as an artifact of its own, when the agent looked it up: the same
 * card, with nothing to step through and nowhere to go back to.
 */
export function DeskOrderArtifact({ recordId }: RecordArtifactProps) {
  return <DeskOrder recordId={recordId} />;
}

function OrderCard({
  record,
  rowOrder,
  onSelect,
  onBack,
  backLabel,
}: Omit<OrderCardProps, "order" | "recordId"> & {
  record: DeskOrderRecord;
  rowOrder: readonly string[];
}) {
  const t = useT();
  const access = useCardAccess(ORDER_SECTIONS);
  const id = record.id;

  const shipments = useRecordShipments({
    scope: { kind: "order", id },
    pageSize: SHIPMENT_ROWS,
    enabled: access.shipments,
  });
  const activity = useRecordActivity({
    resourceId: id,
    pageSize: ACTIVITY_ROWS,
    enabled: access.activity,
  });

  const standing = orderStanding(record.status);
  const worth = orderWorth(record, t);
  const neighbors = neighborsIn(rowOrder, id);
  const customer = record.customer ?? null;
  const steps = lifecycleSteps(record.status);

  const backToOrder = t("Back to {0}", record.orderNumber);
  const listedShipments = useListedRecords({
    entity: "shipment",
    order: shipments.ids,
    backLabel: backToOrder,
  });
  const customerIds = useMemo(() => (customer ? [customer.id] : NO_ROWS), [customer]);
  const listedCustomer = useListedRecords({
    entity: "customer",
    order: customerIds,
    backLabel: backToOrder,
  });

  // A listed record opened from the card takes the card's place, and its way
  // back returns here.
  const opened = listedShipments.opened ?? listedCustomer.opened;
  if (opened) {
    return opened;
  }

  return (
    <RecordCard className={cn("dk-or", `dk-or-${standing.tone}`)}>
      <CardHead
        onBack={onBack}
        backLabel={backLabel ?? t("Back to the table")}
        kicker={[
          record.orderNumber,
          ...(record.poNumber?.trim() ? [t("PO {0}", record.poNumber.trim())] : []),
          ...(record.bol?.trim() ? [t("BOL {0}", record.bol.trim())] : []),
        ]}
        stepper={
          onSelect &&
          rowOrder.length > 1 && (
            <CardStepper
              neighbors={neighbors}
              onStep={onSelect}
              previousLabel={t("Previous order")}
              nextLabel={t("Next order")}
            />
          )
        }
        title={customer?.name ?? t("No customer")}
        subtitle={t("Order {0}", record.orderNumber)}
        figure={
          <>
            <b className={cn(!worth.amount && "dk-or-wait")}>{worth.amount ?? "—"}</b>
            <span>{worth.caption}</span>
          </>
        }
        meta={
          <>
            <CardStatus key={standing.label} tone={standing.tone}>
              {standing.label}
            </CardStatus>
            <CardFacts>
              {[
                record.quotedAmount &&
                record.totalAmount &&
                record.quotedAmount !== record.totalAmount
                  ? t("Quoted {0}", money(record.quotedAmount, record.currencyCode))
                  : "",
              ]}
            </CardFacts>
          </>
        }
      />

      <CardSection
        title={t("Where it stands")}
        aside={
          customer ? (
            <CardOpenLink
              kind="customer" to={recordPath("customer", customer.id)}
              onOpen={listedCustomer.open && (() => listedCustomer.open?.(customer.id))}
            >
              {t("Open customer")}
            </CardOpenLink>
          ) : undefined
        }
      >
        {record.status === "Canceled" && (
          <p className="dk-rec-note dk-or-canceled">{t("This order was canceled.")}</p>
        )}
        <CardTrack
          steps={steps}
          label={t("Order lifecycle")}
          off={record.status === "Canceled"}
          note={
            record.status === "Canceled"
              ? t("Canceled · last changed {0}", cardMoment(record.updatedAt))
              : t(
                  "Step {0} of {1} · last changed {2}",
                  steps.findIndex((step) => step.state === "now") + 1,
                  steps.length,
                  cardMoment(record.updatedAt),
                )
          }
        />
      </CardSection>

      {access.shipments && (
        <CardShipments
          title={t("Legs")}
          shipments={shipments}
          onOpen={listedShipments.open}
          emptyText={t("No shipment carries this order yet.")}
          layout="route"
        />
      )}

      <CardSection title={t("Charges")}>
        {record.charges.length > 0 ? (
          <ChargeLedger charges={record.charges} currency={record.currencyCode} />
        ) : (
          <p className="dk-rec-note">{t("No charge has been added to this order.")}</p>
        )}
      </CardSection>

      {access.activity && (
        <CardActivity
          activity={activity}
          emptyText={t("Nothing has been done to this order yet.")}
          showChanges
        />
      )}

      <CardSection title={t("References")}>
        <OrderReferences facts={orderFacts(record, t)} />
      </CardSection>

      <CardBar>
        <CardBarHint>{t("Change, split and bill the order on its page")}</CardBarHint>
        <Link className="dk-ax-btn dk-ink" to={recordPath("order", id)}>
          <CardIcon name="ext" size={13} />
          {t("Open order")}
        </Link>
      </CardBar>
    </RecordCard>
  );
}
