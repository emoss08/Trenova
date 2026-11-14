import { useT } from "@trenova/shared/i18n/use-t";
import { formatUnixDate } from "@trenova/shared/lib/date";
import { formatMinor } from "@trenova/shared/lib/money";
import { cn } from "@trenova/shared/lib/utils";
import { useState, type ReactNode, type Ref } from "react";

import { SAMPLE_INVOICE, SAMPLE_INVOICE_TOTAL_MINOR } from "./sample-invoice-data";

const SECONDS_PER_DAY = 86_400;
const NET_DAYS = 30;

/** A section's label, as the Ledger template prints it. */
export function InvoiceTag({ children, className }: { children: ReactNode; className?: string }) {
  return (
    <div
      className={cn(
        "mb-1.5 font-mono text-3xs tracking-widest text-doc-ink-2 uppercase",
        className,
      )}
    >
      [ {children} ]
    </div>
  );
}

export type SampleInvoiceIssuer = {
  name: string;
  /** The mailing address, a line each. */
  lines: readonly string[];
  /** The logo as the masthead draws it, sized with `INVOICE_LOGO_CLASS`; nothing without one. */
  logo: ReactNode;
};

/**
 * A sample invoice laid out as the Ledger invoice template prints it: the organization's
 * masthead, a sample shipment and its charges. It is the one drawing of that template in the
 * app, so a screen that previews what an invoice will carry (a factor's remit-to, the
 * organization's logo) shows the same sheet. The remit-to block, the notice and the payment
 * box are the parts a screen fills in; without them the invoice is paid to the organization.
 */
export function SampleInvoice({
  issuer,
  remitTo,
  notice,
  howToPay,
  className,
  ref,
}: {
  issuer: SampleInvoiceIssuer;
  /** Who is paid, under "Remit To". The organization itself when left out. */
  remitTo?: ReactNode;
  /** A boxed notice under the parties, such as a notice of assignment. */
  notice?: ReactNode;
  /** The payment box beside the totals. */
  howToPay?: ReactNode;
  className?: string;
  ref?: Ref<HTMLElement>;
}) {
  const t = useT();
  const [issued] = useState(() => Math.floor(Date.now() / 1000));
  const due = issued + NET_DAYS * SECONDS_PER_DAY;

  return (
    <article
      ref={ref}
      aria-label={t("Sample invoice")}
      className={cn(
        "flex w-full flex-col gap-4 border border-doc-rule bg-doc-paper px-6 py-7 font-sans text-xs text-doc-ink tabular-nums sm:px-9 sm:py-9",
        className,
      )}
    >
      <header className="flex items-start justify-between gap-6 border-b border-doc-ink pb-4">
        <div className="flex min-w-0 items-center gap-3">
          {issuer.logo}
          <div className="min-w-0">
            <div className="text-base font-semibold tracking-tight">{issuer.name}</div>
            <div className="text-2xs text-doc-ink-2">{issuer.lines.join(" · ")}</div>
          </div>
        </div>
        <div className="grid shrink-0 justify-items-end gap-0.5 text-right">
          <InvoiceTag className="mb-0">{t("Invoice")}</InvoiceTag>
          <div className="text-lg font-semibold tracking-tight">{SAMPLE_INVOICE.invoiceNumber}</div>
          <dl className="mt-1 grid grid-cols-[auto_auto] gap-x-3 text-2xs">
            <dt className="text-doc-ink-2">{t("Issued")}</dt>
            <dd className="m-0 font-medium">{formatUnixDate(issued)}</dd>
            <dt className="text-doc-ink-2">{t("Due")}</dt>
            <dd className="m-0 font-medium">{formatUnixDate(due)}</dd>
          </dl>
          <div className="mt-1.5 flex items-baseline gap-2.5 border-t border-doc-rule pt-1.5">
            <InvoiceTag className="mb-0">{t("Amount due")}</InvoiceTag>
            <span className="text-sm font-semibold">{formatMinor(SAMPLE_INVOICE_TOTAL_MINOR)}</span>
          </div>
        </div>
      </header>

      <section className="grid grid-cols-2 gap-6">
        <div className="min-w-0">
          <InvoiceTag>{t("Bill To")}</InvoiceTag>
          <div className="font-semibold">{SAMPLE_INVOICE.billTo.name}</div>
          {SAMPLE_INVOICE.billTo.lines.map((line) => (
            <div key={line} className="text-doc-ink-2">
              {line}
            </div>
          ))}
        </div>
        <div className="min-w-0">
          <InvoiceTag>{t("Remit To")}</InvoiceTag>
          {remitTo ?? (
            <>
              <div className="font-semibold">{issuer.name}</div>
              {issuer.lines.map((line) => (
                <div key={line} className="text-doc-ink-2">
                  {line}
                </div>
              ))}
            </>
          )}
        </div>
      </section>

      {notice}

      <section className="grid grid-cols-2 gap-6 border border-doc-rule bg-doc-paper-warm px-4 py-3">
        {[
          { tag: t("Shipper"), party: SAMPLE_INVOICE.shipper },
          { tag: t("Consignee"), party: SAMPLE_INVOICE.consignee },
        ].map(({ tag, party }) => (
          <div key={party.name} className="min-w-0">
            <InvoiceTag>{tag}</InvoiceTag>
            <div className="font-semibold">{party.name}</div>
            {party.lines.map((line) => (
              <div key={line} className="text-doc-ink-2">
                {line}
              </div>
            ))}
          </div>
        ))}
      </section>

      <div className="-mt-1.5 flex flex-wrap gap-x-4 gap-y-1 font-mono text-2xs text-doc-ink-2">
        {SAMPLE_INVOICE.refs.map(([label, value]) => (
          <span key={label}>
            {label} <span className="font-medium text-doc-ink">{value}</span>
          </span>
        ))}
      </div>

      <section>
        <InvoiceTag>{t("Charges")}</InvoiceTag>
        <table className="w-full border-collapse text-2xs">
          <thead>
            <tr className="border-b border-doc-ink font-mono text-3xs tracking-widest text-doc-ink-2 uppercase">
              <th className="w-5 py-1.5 pr-2 text-left font-medium">#</th>
              <th className="px-2 py-1.5 text-left font-medium">{t("Description")}</th>
              <th className="px-2 py-1.5 text-right font-medium">{t("Quantity")}</th>
              <th className="px-2 py-1.5 text-right font-medium">{t("Rate")}</th>
              <th className="py-1.5 pl-2 text-right font-medium">{t("Amount")}</th>
            </tr>
          </thead>
          <tbody>
            {SAMPLE_INVOICE.charges.map((charge, index) => (
              <tr key={charge.description} className="border-b border-doc-rule-soft">
                <td className="py-1.5 pr-2 align-top text-doc-ink-3">{index + 1}</td>
                <td className="px-2 py-1.5 align-top">{charge.description}</td>
                <td className="px-2 py-1.5 text-right align-top whitespace-nowrap">
                  {charge.quantity}
                </td>
                <td className="px-2 py-1.5 text-right align-top">{charge.rate}</td>
                <td className="py-1.5 pl-2 text-right align-top whitespace-nowrap">
                  {formatMinor(charge.amountMinor)}
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </section>

      <section className="grid grid-cols-[minmax(0,1fr)_11rem] items-start gap-6">
        <div className="min-w-0">{howToPay}</div>
        <dl className="m-0 grid grid-cols-[1fr_auto] gap-x-4 gap-y-1">
          <dt className="text-doc-ink-2">{t("Subtotal")}</dt>
          <dd className="m-0 text-right">{formatMinor(SAMPLE_INVOICE_TOTAL_MINOR)}</dd>
          <dt className="mt-1 border-t border-doc-ink pt-2 text-sm font-semibold">{t("Total")}</dt>
          <dd className="m-0 mt-1 border-t border-doc-ink pt-2 text-right text-sm font-semibold">
            {formatMinor(SAMPLE_INVOICE_TOTAL_MINOR)}
          </dd>
          <dt className="font-semibold">{t("Balance due")}</dt>
          <dd className="m-0 text-right font-semibold">
            {formatMinor(SAMPLE_INVOICE_TOTAL_MINOR)}
          </dd>
        </dl>
      </section>

      <footer className="flex justify-between gap-4 border-t border-doc-rule-soft pt-2 font-mono text-3xs tracking-wide text-doc-ink-3">
        <span className="truncate">{issuer.name}</span>
        <span>{SAMPLE_INVOICE.invoiceNumber}</span>
      </footer>
    </article>
  );
}
