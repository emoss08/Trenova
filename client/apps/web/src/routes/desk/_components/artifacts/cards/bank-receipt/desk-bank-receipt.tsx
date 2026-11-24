import { Link } from "@trenova/shared/components/link";
import { recordPath } from "@/config/record-links";
import { bankReceipt } from "@/lib/queries/bank-receipt";
import type { BankReceiptCard } from "@/types/bank-receipt";
import { useQuery } from "@tanstack/react-query";
import { useT } from "@trenova/shared/i18n/use-t";
import { useMemo } from "react";
import {
  CardBar,
  CardBarHint,
  CardCheck,
  CardChecks,
  CardFacts,
  CardHead,
  CardIcon,
  CardRecordRow,
  CardSection,
  CardStats,
  CardStatus,
  CardStepper,
  CardUnreadable,
  RecordCard,
} from "../../record-card/record-card";
import { minorMoney, neighborsIn } from "../../record-card/record-card-values";
import { useListedRecords, type RecordArtifactProps } from "../registry";
import {
  applicationLine,
  candidateEvidence,
  candidateName,
  candidateOpen,
  candidateStanding,
  candidatesWithheldNote,
  paymentLine,
  paymentPayer,
  paymentStats,
  receiptCheck,
  receiptFacts,
  receiptMemo,
  receiptStanding,
  receiptTitle,
  workItemHolder,
  workItemStanding,
} from "./bank-receipt-card";
import { BankLine } from "./bank-receipt-parts";
import { DeskBankReceiptSkeleton } from "./bank-receipt-skeleton";

export type BankReceiptCardProps = {
  recordId: string;
  /** The rows the receipt was opened from, in the order shown, to step through them. */
  order?: readonly string[];
  onSelect?: (recordId: string) => void;
  onBack?: () => void;
  /** What the way back says; the table, unless the receipt was opened from another card. */
  backLabel?: string;
};

const NO_ROWS: readonly string[] = [];

/**
 * One bank receipt as a cash clerk reads it, opened from a board's row or
 * looked up by the agent: what came into the bank and what the bank wrote
 * beside it, whether it is applied, and if so through which customer's payment
 * and to which invoices, or why not, and who holds it in the reconciliation
 * queue. It reads; matching and resolving stay on the reconciliation page, one
 * click away.
 */
export function DeskBankReceipt({
  recordId,
  order: rowOrder = NO_ROWS,
  onSelect,
  onBack,
  backLabel,
}: BankReceiptCardProps) {
  const t = useT();
  const query = useQuery({ ...bankReceipt.card(recordId), retry: false });
  const card = query.data;

  if (!card) {
    return query.isError ? (
      <CardUnreadable
        kind={t("Bank receipt")}
        title={t("This bank receipt could not be read")}
        error={query.error}
        onBack={onBack}
        backLabel={backLabel}
      />
    ) : (
      <DeskBankReceiptSkeleton stepper={onSelect !== undefined && rowOrder.length > 1} />
    );
  }

  return (
    <BankReceiptCardView
      key={card.receipt.id ?? recordId}
      recordId={recordId}
      card={card}
      order={rowOrder}
      onSelect={onSelect}
      onBack={onBack}
      backLabel={backLabel}
    />
  );
}

/** The receipt as an artifact of its own, when the agent looked it up. */
export function DeskBankReceiptArtifact({ recordId }: RecordArtifactProps) {
  return <DeskBankReceipt recordId={recordId} />;
}

function BankReceiptCardView({
  recordId,
  card,
  order: rowOrder = NO_ROWS,
  onSelect,
  onBack,
  backLabel,
}: BankReceiptCardProps & { card: BankReceiptCard }) {
  const t = useT();
  const { receipt, payment, workItem } = card;
  const title = receiptTitle(card, t);
  const memo = receiptMemo(card);
  const standing = receiptStanding(receipt.status, t);
  const check = receiptCheck(card, t);
  const currency = payment?.currencyCode || "USD";

  const backToReceipt = t("Back to {0}", title);
  const customerIds = useMemo(
    () => (payment?.customerId ? [payment.customerId] : NO_ROWS),
    [payment],
  );
  const invoiceIds = useMemo(
    () => [
      ...card.candidates
        .filter((candidate) => candidate.kind === "Invoice" && candidate.invoiceNumber.trim())
        .map((candidate) => candidate.invoiceId),
      ...(payment?.applications
        .filter((application) => application.invoiceNumber.trim() !== "")
        .map((application) => application.invoiceId) ?? NO_ROWS),
    ],
    [card.candidates, payment],
  );
  const listedCustomer = useListedRecords({
    entity: "customer",
    order: customerIds,
    backLabel: backToReceipt,
  });
  const listedInvoices = useListedRecords({
    entity: "invoice",
    order: invoiceIds,
    backLabel: backToReceipt,
  });

  const opened = listedCustomer.opened ?? listedInvoices.opened;
  if (opened) {
    return opened;
  }

  return (
    <RecordCard>
      <CardHead
        onBack={onBack}
        backLabel={backLabel ?? t("Back to the table")}
        kicker={[t("Bank receipt"), receipt.status === "Matched" ? t("Applied") : t("Not applied")]}
        stepper={
          onSelect &&
          rowOrder.length > 1 && (
            <CardStepper
              neighbors={neighborsIn(rowOrder, recordId)}
              onStep={onSelect}
              previousLabel={t("Previous receipt")}
              nextLabel={t("Next receipt")}
            />
          )
        }
        title={title}
        subtitle={memo}
        figure={
          <>
            <b>{minorMoney(receipt.amountMinor)}</b>
            <span>{t("received")}</span>
          </>
        }
        meta={
          <>
            <CardStatus key={receipt.status} tone={standing.tone}>
              {standing.label}
            </CardStatus>
            <CardFacts>{receiptFacts(card, t)}</CardFacts>
          </>
        }
      />

      <CardSection title={t("The bank's line")}>
        <BankLine
          receiptDate={receipt.receiptDate}
          reference={receipt.referenceNumber.trim()}
          memo={memo}
          amountMinor={receipt.amountMinor}
        />
      </CardSection>

      <CardSection title={t("Whether it is applied")}>
        <CardChecks>
          <CardCheck state={check.state} title={check.title} detail={check.detail} />
        </CardChecks>
      </CardSection>

      {receipt.status !== "Matched" && (
        <CardSection title={t("What it could pay")}>
          {card.candidates.length > 0 && (
            <div className="dk-rec-list">
              {card.candidates.map((candidate) => {
                const standing = candidateStanding(candidate, t);
                const body = (
                  <span className="dk-rec-main">
                    <b>{candidateName(candidate, t)}</b>
                    <span className="dk-rec-sub">
                      {[candidateOpen(candidate, t), ...candidateEvidence(candidate, t)].join(
                        " · ",
                      )}
                    </span>
                  </span>
                );
                const status = <CardStatus tone={standing.tone}>{standing.label}</CardStatus>;
                return candidate.kind === "Invoice" && candidate.invoiceNumber.trim() ? (
                  <CardRecordRow
                    key={candidate.rank}
                    kind="invoice"
                    to={recordPath("invoice", candidate.invoiceId)}
                    onOpen={
                      listedInvoices.open && (() => listedInvoices.open?.(candidate.invoiceId))
                    }
                  >
                    {body}
                    {status}
                  </CardRecordRow>
                ) : (
                  <div key={candidate.rank} className="dk-rec-row">
                    {body}
                    {status}
                  </div>
                );
              })}
            </div>
          )}
          {card.candidates.length === 0 && card.candidatesWithheld.length === 0 && (
            <p className="dk-rec-note">
              {t("Nothing the receipt carries names an open invoice or a posted payment.")}
            </p>
          )}
          {card.candidatesWithheld.length > 0 && (
            <p className="dk-rec-note">{candidatesWithheldNote(card, t)}</p>
          )}
        </CardSection>
      )}

      {payment && (
        <CardSection title={t("The payment it was matched to")}>
          <CardStats stats={paymentStats(payment, t)} />
          <div className="dk-rec-list">
            <CardRecordRow
              kind="customer"
              to={recordPath("customer", payment.customerId)}
              onOpen={listedCustomer.open && (() => listedCustomer.open?.(payment.customerId))}
            >
              <span className="dk-rec-main">
                <b>{paymentPayer(payment, t)}</b>
                <span className="dk-rec-sub">{paymentLine(payment, t)}</span>
              </span>
            </CardRecordRow>
            {payment.applications.map((application) => {
              const line = applicationLine(application, currency, t);
              const readable = application.invoiceNumber.trim() !== "";
              const body = (
                <span className="dk-rec-main">
                  <b>{line.name}</b>
                  <span className="dk-rec-sub">
                    {line.note
                      ? t("{0} applied, {1}", line.amount, line.note)
                      : t("{0} applied", line.amount)}
                  </span>
                </span>
              );
              return readable ? (
                <CardRecordRow
                  key={application.invoiceId}
                  kind="invoice"
                  to={recordPath("invoice", application.invoiceId)}
                  onOpen={
                    listedInvoices.open && (() => listedInvoices.open?.(application.invoiceId))
                  }
                >
                  {body}
                </CardRecordRow>
              ) : (
                <div key={application.invoiceId} className="dk-rec-row">
                  {body}
                </div>
              );
            })}
          </div>
          {payment.applications.length === 0 && (
            <p className="dk-rec-note">{t("The payment is not applied to any invoice yet.")}</p>
          )}
        </CardSection>
      )}

      {card.workItemShown && (
        <CardSection title={t("Reconciliation queue")}>
          {workItem ? (
            <div className="dk-rec-list">
              <div className="dk-rec-row">
                <span className="dk-rec-main">
                  <b>{workItemHolder(workItem, t)}</b>
                  {workItem.resolutionNote.trim() && (
                    <span className="dk-rec-sub">{workItem.resolutionNote.trim()}</span>
                  )}
                </span>
                <CardStatus tone={workItemStanding(workItem, t).tone}>
                  {workItemStanding(workItem, t).label}
                </CardStatus>
              </div>
            </div>
          ) : (
            <p className="dk-rec-note">{t("It has never been in the reconciliation queue.")}</p>
          )}
        </CardSection>
      )}

      <CardBar>
        <CardBarHint>{t("Match or resolve it on the reconciliation page")}</CardBarHint>
        <Link className="dk-ax-btn dk-ink" to={recordPath("bank_receipt", receipt.id ?? recordId)}>
          <CardIcon name="ext" size={13} />
          {t("Open bank receipt")}
        </Link>
      </CardBar>
    </RecordCard>
  );
}
