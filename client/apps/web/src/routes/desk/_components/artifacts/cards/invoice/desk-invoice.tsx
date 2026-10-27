import { recordPath } from "@/config/record-links";
import { useApiMutation } from "@/hooks/use-api-mutation";
import { useNowSeconds } from "@/hooks/use-now-seconds";
import { usePermission } from "@/hooks/use-permission";
import { sendInvoiceEdi } from "@/lib/graphql/invoice";
import type { DeskInvoice as DeskInvoiceRecord } from "@/lib/graphql/invoice-card";
import { invoiceEdiSendBlocker, invoiceEmailSendBlocker } from "@/lib/invoice-actions";
import { accountingSync } from "@/lib/queries/accounting-sync";
import { invalidateInvoiceQueries, invoice as invoiceQueries } from "@/lib/queries/invoice";
import { InvoicePostDialog } from "@/routes/invoice/_components/invoice-post-dialog";
import { InvoiceVoidDialog } from "@/routes/invoice/_components/invoice-void-dialog";
import { apiService } from "@/services/api";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { Button } from "@trenova/shared/components/ui/button";
import { useT, type TranslateFn } from "@trenova/shared/i18n/use-t";
import { cn } from "@trenova/shared/lib/utils";
import { Operation, Resource } from "@trenova/shared/types/permission";
import { useMemo, useState } from "react";
import { Link } from "react-router";
import { toast } from "@trenova/shared/components/ui/toast";
import { CardActionButton } from "../../record-card/card-action";
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
  voidBlocker,
  voidNote,
} from "./invoice-card";
import {
  type ChannelAct,
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
 * what it settled.
 *
 * Its buttons are the invoice's own actions, taken by the person, never by the
 * agent, each under the permission the server checks: post a draft (after the
 * same preview of what posting books that the invoice's page shows), email it
 * or make its PDF first, send its EDI 210, and void it. One that cannot be
 * taken now stays in place and says why. Applying cash, adjusting and
 * disputing stay on the invoice's page, one click away.
 *
 * The card steps through the rows it was opened from in the order shown. A
 * step draws the next invoice's skeleton until it is read, never the invoice
 * it left with its buttons still live.
 */
export function DeskInvoice({
  recordId,
  order: rowOrder = NO_ROWS,
  onSelect,
  onBack,
  backLabel,
}: InvoiceCardProps) {
  const t = useT();
  const query = useQuery({ ...invoiceQueries.card(recordId), retry: false });
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

/** What the bar says of where the rest of the invoice's work is done. */
function barHint(record: DeskInvoiceRecord, t: TranslateFn): string {
  if (record.status === "Voided") {
    return t("A voided invoice is kept for the record");
  }
  if (record.status === "Draft") {
    return t("Change the lines and terms on its page");
  }
  return invoiceKind(record).kind === "credit"
    ? t("Apply or take back the credit on its page")
    : t("Apply payments and adjust on its page");
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

  const { allowed: canEdit } = usePermission(Resource.Invoice, Operation.Update);
  const { allowed: canSend } = usePermission(Resource.Invoice, Operation.Submit);
  const { allowed: canVoid } = usePermission(Resource.Invoice, Operation.Cancel);
  const { allowed: canPost } = usePermission(Resource.Invoice, Operation.Post);
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
  const sendPlanQuery = useQuery({
    ...invoiceQueries.sendPlan(id),
    retry: false,
    enabled: !voided && (record.sendStatus === "NotSent" || canSend),
  });
  const sendPlan = sendPlanQuery.data;
  const syncState = useQuery({
    ...accountingSync.syncObjectStates([id]),
    retry: false,
    enabled: access.books && !draft,
  }).data?.find((state) => state.objectId === id);

  const queryClient = useQueryClient();
  const [dialog, setDialog] = useState<"post" | "void" | null>(null);
  const settle = () => invalidateInvoiceQueries(queryClient);
  const sendEmail = useApiMutation({
    resourceName: "invoice",
    mutationFn: () => apiService.invoiceService.send(id),
    onSuccess: () => settle(),
  });
  const makePdf = useApiMutation({
    resourceName: "invoice PDF",
    mutationFn: () => apiService.invoiceService.generatePdf(id),
    onSuccess: () => {
      settle();
      toast.success(t("{0} PDF generation started", record.number));
    },
  });
  const sendEdi = useApiMutation({
    resourceName: "invoice EDI",
    mutationFn: () => sendInvoiceEdi(id, record.ediSendPlan.status === "Sent"),
    onSuccess: () => settle(),
  });
  const busy = sendEmail.isPending || makePdf.isPending || sendEdi.isPending;

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
  const live = !draft && !voided;
  const emailActs: ChannelAct[] = [];
  if (live && !record.pdfDocumentId && canEdit) {
    emailActs.push({
      kind: "do",
      label: t("Generate PDF"),
      onClick: () => makePdf.mutate(undefined),
      blocker: null,
      busy,
    });
  } else if (live && canSend) {
    emailActs.push({
      kind: "do",
      label: record.sendStatus === "Sent" ? t("Resend") : t("Send"),
      onClick: () => sendEmail.mutate(undefined),
      blocker: invoiceEmailSendBlocker(record, sendPlan, sendPlanQuery.isLoading, t),
      busy,
    });
  }
  const ediActs: ChannelAct[] = [];
  if (live && canEdit && record.ediSendPlan.enabled) {
    ediActs.push({
      kind: "do",
      label: record.ediSendPlan.status === "Sent" ? t("Resend 210") : t("Send 210"),
      onClick: () => sendEdi.mutate(undefined),
      blocker: invoiceEdiSendBlocker(record.status, record.ediSendPlan, t),
      busy,
    });
  }
  if (edi.messageId && listedMessages.open) {
    ediActs.push({
      kind: "open",
      label: t("Open message"),
      onClick: () => listedMessages.open?.(edi.messageId ?? ""),
    });
  }
  const slips: ChannelSlip[] = [
    { key: "email", label: t("Email"), ...email, actions: emailActs },
    { key: "edi", label: t("EDI 210"), ...edi, actions: ediActs },
  ];
  if (books) {
    slips.push({
      key: "books",
      label: t("Accounting"),
      ...books,
      actions: books.externalUrl
        ? [
            {
              kind: "outside",
              label: t("Open in {0}", books.provider),
              href: books.externalUrl,
            },
          ]
        : books.needsAction
          ? [
              {
                kind: "page",
                label: t("Resolve in the sync ledger"),
                to: recordPath("accounting_sync_record", books.recordId),
              },
            ]
          : undefined,
    });
  }

  const showsOwed = !draft && !voided;
  const postable = draft && canPost;
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
                kind="customer"
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
            tariffNotes={record.ratingNotes}
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
        {!voided && canVoid && (
          <CardActionButton
            variant="quiet"
            blocker={voidBlocker(record, t)}
            busy={busy}
            onClick={() => setDialog("void")}
          >
            <CardIcon name="x" size={12} stroke={2.4} />
            {credit ? t("Void memo") : t("Void")}
          </CardActionButton>
        )}
        <CardBarHint>{barHint(record, t)}</CardBarHint>
        <Link className={cn("dk-ax-btn", !postable && "dk-ink")} to={recordPath("invoice", id)}>
          <CardIcon name="ext" size={13} />
          {credit ? t("Open credit memo") : t("Open invoice")}
        </Link>
        {postable && (
          <Button disabled={busy} onClick={() => setDialog("post")}>
            {t("Post {0}", figure.amount)}
          </Button>
        )}
      </CardBar>

      {dialog === "post" && (
        <InvoicePostDialog invoiceId={id} open onOpenChange={(open) => !open && setDialog(null)} />
      )}
      <InvoiceVoidDialog
        invoice={record}
        open={dialog === "void"}
        onOpenChange={(open) => setDialog(open ? "void" : null)}
      />
    </RecordCard>
  );
}
