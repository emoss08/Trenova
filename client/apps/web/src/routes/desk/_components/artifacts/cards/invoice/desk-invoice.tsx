import { recordPath } from "@/config/record-links";
import { useNowSeconds } from "@/hooks/use-now-seconds";
import type { DeskInvoice as DeskInvoiceRecord } from "@/lib/graphql/invoice-card";
import { accountingSync } from "@/lib/queries/accounting-sync";
import { invoice as invoiceQueries } from "@/lib/queries/invoice";
import { keepPreviousData, useQuery } from "@tanstack/react-query";
import { useT, type TranslateFn } from "@trenova/shared/i18n/use-t";
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
import { neighborsIn } from "../../record-card/record-card-values";
import { useRecordActivity } from "../../record-card/record-activity";
import { useRecordShipments } from "../../record-card/record-shipments";
import { useListedRecords, type RecordArtifactProps } from "../registry";
import {
  booksChannel,
  chargeFoot,
  chargeRows,
  creditUses,
  disputeCases,
  ediChannel,
  emailChannel,
  headFacts,
  invoiceFigure,
  invoiceKicker,
  invoiceKind,
  invoiceStanding,
  invoiceSubtitle,
  owedSegments,
  neighborRow,
  remittance,
  waitingNote,
  type RemitRow,
  settledSummary,
  termsRuler,
  voidNote,
} from "./invoice-card";
import {
  ChannelSlips,
  type ChannelSlip,
  ChargeBill,
  DisputeCases,
  InvoiceSlip,
  OwedStrip,
  RemitList,
  TermsRulerView,
  VoidSlip,
} from "./invoice-parts";
import { INVOICE_SECTIONS } from "./invoice-sections";
import {
  DeskInvoiceSkeleton,
  INVOICE_ACTIVITY_ROWS,
  INVOICE_SHIPMENT_ROWS,
} from "./invoice-skeleton";

/**
 * What the card is given: the invoice, and when it was opened from a table or
 * from a card that lists it, the rows in the order shown, how to step to
 * another, and the way back.
 */
export type InvoiceCardProps = {
  recordId: string;
  order?: readonly string[];
  onSelect?: (recordId: string) => void;
  onBack?: () => void;
  /** What the way back says; the table, unless the invoice was opened from another card. */
  backLabel?: string;
};

const NO_ROWS: readonly string[] = [];

/**
 * One invoice as a biller or a collector reads it, opened from a table's row
 * or looked up by the agent: who it bills and what is still owed, the terms
 * and how far past them it is (or, for a draft, how long its terms have run
 * while it waits to be posted), its charges summed by how each was billed with
 * the formula template that rated the freight, what settled it, its disputes,
 * where it went by email, EDI and to the accounting system, and the shipments
 * it bills. A voided invoice says why; a credit memo, what it has left and
 * what it settled. It reads; posting, sending, applying cash and voiding stay
 * on the invoice's page, one click away.
 *
 * The card steps through the rows it was opened from in the order shown.
 */
export function DeskInvoice({
  recordId,
  order: rowOrder = NO_ROWS,
  onSelect,
  onBack,
  backLabel,
}: InvoiceCardProps) {
  const t = useT();
  const query = useQuery({
    ...invoiceQueries.card(recordId),
    retry: false,
    placeholderData: keepPreviousData,
  });
  const record = query.data;

  if (!record) {
    return query.isError ? (
      <CardUnreadable
        kind={t("Invoice")}
        title={t("This invoice could not be read")}
        error={query.error}
        onBack={onBack}
        backLabel={backLabel}
      />
    ) : (
      <DeskInvoiceSkeleton stepper={onSelect !== undefined && rowOrder.length > 1} />
    );
  }

  return (
    <InvoiceCard
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
 * The invoice as an artifact of its own, when the agent looked it up: the same
 * card, with nothing to step through and nowhere to go back to.
 */
export function DeskInvoiceArtifact({ recordId }: RecordArtifactProps) {
  return <DeskInvoice recordId={recordId} />;
}

/** What the bar says of where the invoice is worked. */
function barHint(record: DeskInvoiceRecord, t: TranslateFn): string {
  if (record.status === "Voided") {
    return t("A voided invoice is kept for the record");
  }
  if (record.status === "Draft") {
    return t("Post, send and change the invoice on its page");
  }
  return invoiceKind(record).kind === "credit"
    ? t("Apply or take back the credit on its page")
    : t("Send, apply payments and void on its page");
}

/** The ids of the listed records, held still while the list is. */
function useStableIds(ids: readonly string[]): readonly string[] {
  const key = ids.join(" ");
  return useMemo(() => (key ? key.split(" ") : NO_ROWS), [key]);
}

function InvoiceCard({
  record,
  rowOrder,
  onSelect,
  onBack,
  backLabel,
}: Omit<InvoiceCardProps, "order" | "recordId"> & {
  record: DeskInvoiceRecord;
  rowOrder: readonly string[];
}) {
  const t = useT();
  const now = useNowSeconds();
  const access = useCardAccess(INVOICE_SECTIONS);
  const id = record.id;
  const kind = invoiceKind(record);
  const credit = kind.kind === "credit";
  const draft = record.status === "Draft";
  const voided = record.status === "Voided";

  const shipments = useRecordShipments({
    scope: { kind: "invoice", id },
    pageSize: INVOICE_SHIPMENT_ROWS,
    enabled: access.shipments && record.shipmentCount > 0,
  });
  const activity = useRecordActivity({
    resourceId: id,
    pageSize: INVOICE_ACTIVITY_ROWS,
    enabled: access.activity,
  });
  const lastAttempt = useQuery({
    ...invoiceQueries.lastEmailAttempt(id),
    retry: false,
    enabled: record.sendStatus !== "NotSent",
  }).data;
  const sendPlan = useQuery({
    ...invoiceQueries.sendPlan(id),
    retry: false,
    enabled: record.sendStatus === "NotSent" && !voided,
  }).data;
  const syncState = useQuery({
    ...accountingSync.syncObjectStates([id]),
    retry: false,
    enabled: access.books && !draft,
  }).data?.find((state) => state.objectId === id);

  const standing = invoiceStanding(record, t);
  const figure = invoiceFigure(record, t);
  const neighbors = neighborsIn(rowOrder, id);
  const ruler = termsRuler(record, now, t);
  const rows = chargeRows(record, t);
  const foot = chargeFoot(record, t);
  const settledBy = credit ? creditUses(record, t) : remittance(record, t);
  const cases = disputeCases(record, now, t);
  const voiding = voidNote(record, t);
  const edi = ediChannel(record, t);
  const books = syncState ? booksChannel(syncState, t) : null;

  const title = record.customer?.name ?? record.billToName;
  const backToInvoice = t("Back to {0}", record.number);
  const templateIds = useStableIds(
    record.chargeSummary.flatMap((group) =>
      group.formulaTemplateId ? [group.formulaTemplateId] : [],
    ),
  );
  const messageIds = useStableIds(edi.messageId ? [edi.messageId] : []);
  const invoiceIds = useStableIds([
    ...(record.referenceInvoice ? [record.referenceInvoice.id] : []),
    ...record.relatedInvoices.map((related) => related.id),
    ...settledBy.flatMap((row) => (row.open ? [row.open.id] : [])),
  ]);
  const customerIds = useStableIds(record.customer ? [record.customer.id] : []);
  const listedShipments = useListedRecords({
    entity: "shipment",
    order: shipments.ids,
    backLabel: backToInvoice,
  });
  const listedTemplates = useListedRecords({
    entity: "formula_template",
    order: templateIds,
    backLabel: backToInvoice,
  });
  const listedMessages = useListedRecords({
    entity: "edi_message",
    order: messageIds,
    backLabel: backToInvoice,
  });
  const listedInvoices = useListedRecords({
    entity: "invoice",
    order: invoiceIds,
    backLabel: backToInvoice,
  });
  const listedCustomer = useListedRecords({
    entity: "customer",
    order: customerIds,
    backLabel: backToInvoice,
  });

  // A listed record opened from the card takes the card's place, and its way
  // back returns here.
  const opened =
    listedShipments.opened ??
    listedTemplates.opened ??
    listedMessages.opened ??
    listedInvoices.opened ??
    listedCustomer.opened;
  if (opened) {
    return opened;
  }

  const openInvoice = (row: RemitRow) =>
    row.open && listedInvoices.open ? () => listedInvoices.open?.(row.open?.id ?? "") : undefined;

  const email = emailChannel(
    record,
    {
      attempt: lastAttempt
        ? {
            toRecipients: lastAttempt.toRecipients,
            sentAt: lastAttempt.sentAt,
            error: lastAttempt.error,
          }
        : null,
      plan: sendPlan ? { to: sendPlan.recipients.to, errors: sendPlan.errors } : null,
    },
    t,
  );
  const slips: ChannelSlip[] = [
    { key: "email", label: t("Email"), ...email },
    {
      key: "edi",
      label: t("EDI 210"),
      ...edi,
      action:
        edi.messageId && listedMessages.open
          ? {
              kind: "open",
              label: t("Open message"),
              onClick: () => listedMessages.open?.(edi.messageId ?? ""),
            }
          : undefined,
    },
  ];
  if (books) {
    slips.push({
      key: "books",
      label: t("Accounting"),
      ...books,
      action: books.externalUrl
        ? {
            kind: "outside",
            label: t("Open in {0}", books.provider),
            href: books.externalUrl,
          }
        : books.needsAction
          ? {
              kind: "page",
              label: t("Resolve in the sync ledger"),
              to: recordPath("accounting_sync_record", books.recordId),
            }
          : undefined,
    });
  }

  const showsOwed = !draft && !voided;
  const related = record.relatedInvoices;

  return (
    <RecordCard className={cn("dk-inv", `dk-inv-${kind.kind}`, voided && "dk-inv-voided")}>
      <CardHead
        onBack={onBack}
        backLabel={backLabel ?? t("Back to the table")}
        kicker={invoiceKicker(record, t)}
        stepper={
          onSelect &&
          rowOrder.length > 1 && (
            <CardStepper
              neighbors={neighbors}
              onStep={onSelect}
              previousLabel={t("Previous invoice")}
              nextLabel={t("Next invoice")}
            />
          )
        }
        lead={<InvoiceSlip mark={kind.mark} />}
        title={title}
        subtitle={invoiceSubtitle(record, t)}
        figure={
          <>
            <b className={cn(figure.tone === "owed" && "dk-inv-owing")}>
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
            <CardFacts>{headFacts(record, now, t)}</CardFacts>
          </>
        }
      />

      {voiding && (
        <CardSection title={t("Why it was voided")}>
          <VoidSlip note={voiding} />
        </CardSection>
      )}

      {draft && (
        <CardSection
          title={t("Waiting to post")}
          aside={<span className="dk-inv-aside">{t("Not sent to {0} yet", title)}</span>}
        >
          {ruler && <TermsRulerView ruler={ruler} />}
          <p className="dk-inv-pending">
            {ruler ? waitingNote(ruler, t) : t("Nothing is owed until it is posted.")}
          </p>
        </CardSection>
      )}

      {showsOwed && (
        <CardSection
          title={credit ? t("Credit") : t("What is owed")}
          aside={
            record.customer ? (
              <CardOpenLink
                to={recordPath("customer", record.customer.id)}
                onOpen={
                  listedCustomer.open && (() => listedCustomer.open?.(record.customer?.id ?? ""))
                }
              >
                {t("Open customer")}
              </CardOpenLink>
            ) : undefined
          }
        >
          <OwedStrip segments={owedSegments(record, t)} />
          {ruler && <TermsRulerView ruler={ruler} />}
          {credit && record.memoReason.trim() && (
            <p className="dk-inv-reason">
              <b>{t("Why it was issued")}</b>
              {record.memoReason.trim()}
            </p>
          )}
        </CardSection>
      )}

      {record.referenceInvoice && (
        <CardSection title={t("Corrects")}>
          <RemitList rows={[neighborRow(record.referenceInvoice, t)]} onOpen={openInvoice} />
        </CardSection>
      )}

      <CardSection
        title={t("Charges")}
        aside={
          <span className="dk-inv-aside">
            {t("{0, plural, one {# charge} other {# charges}}", rows.length)}
          </span>
        }
      >
        {rows.length > 0 ? (
          <ChargeBill
            rows={rows}
            foot={foot}
            note={
              record.shipmentCount > 1
                ? t(
                    "Each row totals every line billed the same way. The invoice lists each line by shipment.",
                  )
                : undefined
            }
            onOpenTemplate={listedTemplates.open}
          />
        ) : (
          <p className="dk-rec-note">{t("No charge is on this invoice.")}</p>
        )}
      </CardSection>

      {showsOwed && (
        <CardSection
          title={credit ? t("Applied to") : t("Payments and credits")}
          aside={
            settledBy.length > 0 ? (
              <span className="dk-inv-aside">{settledSummary(settledBy, credit, t)}</span>
            ) : undefined
          }
        >
          {settledBy.length > 0 ? (
            <RemitList rows={settledBy} onOpen={openInvoice} />
          ) : (
            <p className="dk-rec-note">
              {credit
                ? t("This credit has not been applied to an invoice yet.")
                : t("Nothing has been paid or credited against this invoice yet.")}
            </p>
          )}
        </CardSection>
      )}

      {record.disputes.length > 0 && (
        <CardSection
          title={t("Dispute")}
          aside={cases.open ? <span className="dk-inv-aside">{cases.open.age}</span> : undefined}
        >
          <DisputeCases open={cases.open} closed={cases.closed} />
        </CardSection>
      )}

      {related.length > 0 && (
        <CardSection title={record.isSplitBill ? t("Split with") : t("Related invoices")}>
          <RemitList rows={related.map((item) => neighborRow(item, t))} onOpen={openInvoice} />
        </CardSection>
      )}

      <CardSection title={draft ? t("Where it will go") : t("Where it went")}>
        <ChannelSlips slips={slips} />
      </CardSection>

      {access.shipments && record.shipmentCount > 0 && (
        <CardShipments
          title={record.shipmentCount === 1 ? t("Shipment it bills") : t("Shipments it bills")}
          shipments={shipments}
          onOpen={listedShipments.open}
          emptyText={t("No shipment is billed on this invoice.")}
          layout="route"
        />
      )}

      {access.activity && (
        <CardActivity
          activity={activity}
          emptyText={t("Nothing has been done to this invoice yet.")}
          showChanges
        />
      )}

      <CardBar>
        <CardBarHint>{barHint(record, t)}</CardBarHint>
        <Link className="dk-ax-btn dk-ink" to={recordPath("invoice", id)}>
          <CardIcon name="ext" size={13} />
          {credit ? t("Open credit memo") : t("Open invoice")}
        </Link>
      </CardBar>
    </RecordCard>
  );
}
