import { recordPath } from "@/config/record-links";
import type { DeskInvoice } from "@/lib/graphql/invoice-card";
import {
  chargeFoot,
  chargeRows,
  disputeCases,
  headFacts,
  invoiceFigure,
  invoiceKind,
  invoiceStanding,
  invoiceSubtitle,
  owedSegments,
  termsRuler,
  type OwedSegment,
} from "@/routes/desk/_components/artifacts/cards/invoice/invoice-card";
import { useNow } from "@trenova/shared/hooks/use-now";
import { useT } from "@trenova/shared/i18n/use-t";
import { cn } from "@trenova/shared/lib/utils";
import { useQuery } from "@tanstack/react-query";
import { PALETTE_ENTITY_ICONS } from "../../palette-entities";
import { StatusDot } from "../../palette-parts";
import {
  PreviewError,
  PreviewFields,
  PreviewHead,
  PreviewSection,
  PreviewSkeleton,
  PreviewSource,
  RelatedRow,
  StatePill,
} from "./preview-frame";
import { invoicePreviewQuery } from "./preview-queries";
import { TONE_STATE } from "./preview-tones";
import type { RecordPreviewProps } from "./record-preview-props";

const CHARGES_SHOWN = 4;

/** Each part of what is owed in its own fill: open is the ink, a dispute the warning. */
const SEGMENT_FILL: Record<OwedSegment["key"], string> = {
  paid: "bg-success",
  credited: "bg-foreground-faint",
  disputed: "bg-warning",
  open: "bg-foreground-muted",
};

/**
 * What is owed, as the Desk's invoice card draws it: one strip, each part as
 * wide as its share of the total, and the parts with something in them keyed
 * underneath.
 */
export function OwedStrip({ segments }: { segments: readonly OwedSegment[] }) {
  const shown = segments.filter((segment) => !segment.zero);
  if (shown.length === 0) {
    return null;
  }
  return (
    <div className="flex flex-col gap-2">
      <div className="bg-sunken flex h-2 overflow-hidden rounded-full" aria-hidden>
        {shown.map((segment) => (
          <span
            key={segment.key}
            className={cn(
              "cp-grow-x h-full first:rounded-l-full last:rounded-r-full",
              SEGMENT_FILL[segment.key],
            )}
            style={{ flexGrow: segment.share }}
          />
        ))}
      </div>
      <ul className="flex flex-wrap gap-x-4 gap-y-1">
        {shown.map((segment) => (
          <li key={segment.key} className="flex items-center gap-1.5 text-sm">
            <span
              aria-hidden
              className={cn("size-1.5 shrink-0 rounded-full", SEGMENT_FILL[segment.key])}
            />
            <span className="text-foreground-subtle">{segment.label}</span>
            <span className="text-foreground font-mono">{segment.amount}</span>
          </li>
        ))}
      </ul>
    </div>
  );
}

function InvoiceBody({
  invoice,
  onNavigate,
}: {
  invoice: DeskInvoice;
  onNavigate: (href: string) => void;
}) {
  const t = useT();
  const now = useNow();
  const kind = invoiceKind(invoice);
  const standing = invoiceStanding(invoice, t);
  const figure = invoiceFigure(invoice, t);
  const terms = termsRuler(invoice, now, t);
  const charges = chargeRows(invoice, t);
  const due = chargeFoot(invoice, t).find((row) => row.kind === "due");
  const { open } = disputeCases(invoice, now, t);
  const late = terms?.kind === "late";
  const customer = invoice.customer;
  const figureInk =
    figure.tone === "struck"
      ? "text-foreground-subtle line-through"
      : figure.tone === "owed" && late
        ? "text-warning-subtle-foreground"
        : "text-foreground";

  return (
    <>
      <PreviewSource kind={kind.mark} facts={headFacts(invoice, now, t)} />
      <PreviewHead
        title={invoice.number}
        mono
        subtitle={invoiceSubtitle(invoice, t)}
        status={<StatePill label={standing.label} state={TONE_STATE[standing.tone]} />}
      />
      <div className="bg-sunken rounded-surface flex flex-col gap-3 px-4 py-3.5">
        <div className="flex items-baseline gap-2">
          <span className={cn("text-2xl font-medium tabular-nums", figureInk)}>
            {figure.amount}
          </span>
          <span className="text-foreground-subtle text-sm">{figure.caption}</span>
        </div>
        <OwedStrip segments={owedSegments(invoice, t)} />
      </div>
      {open && (
        <p className="text-foreground-muted flex items-start gap-2 text-base">
          <StatusDot state="warn" className="mt-2 size-1.5" />
          <span>{t("In dispute: {0}", open.reason)}</span>
        </p>
      )}
      {charges.length > 0 && (
        <PreviewSection label={t("Charges")} aside={due?.value}>
          <PreviewFields
            rows={charges.slice(0, CHARGES_SHOWN).map((row) => ({
              label: row.name,
              value: <span className="font-mono">{row.amount}</span>,
            }))}
          />
          {charges.length > CHARGES_SHOWN && (
            <p className="text-foreground-subtle text-sm">
              {t(
                "{0, plural, one {# more charge} other {# more charges}}",
                charges.length - CHARGES_SHOWN,
              )}
            </p>
          )}
        </PreviewSection>
      )}
      {customer && (
        <PreviewSection label={t("Bills")}>
          <RelatedRow
            icon={PALETTE_ENTITY_ICONS.customer}
            title={customer.name}
            sub={customer.code}
            onOpen={() => onNavigate(recordPath("customer", customer.id))}
          />
        </PreviewSection>
      )}
      <PreviewFields
        rows={[
          terms ? { label: terms.startLabel, value: terms.startDay } : null,
          invoice.billToName && invoice.billToName !== customer?.name
            ? { label: t("Bill to"), value: invoice.billToName }
            : null,
          invoice.shipmentProNumber
            ? {
                label: t("Shipment"),
                value: <span className="font-mono">{invoice.shipmentProNumber}</span>,
              }
            : null,
        ]}
      />
    </>
  );
}

export function InvoicePreview({ record, onNavigate }: RecordPreviewProps) {
  const { data, isLoading, isError } = useQuery(invoicePreviewQuery(record.id));

  if (isLoading) {
    return <PreviewSkeleton />;
  }
  if (isError || !data) {
    return <PreviewError />;
  }
  return <InvoiceBody invoice={data} onNavigate={onNavigate} />;
}
