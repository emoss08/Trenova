import { tariffLimitText, tariffLineLabel } from "@/components/tariff-bill/bill-view";
import { AnnotationIcon, Flag01Icon } from "@trenova/shared/components/icons";
import { useT } from "@trenova/shared/i18n/use-t";
import { formatTariffMoney, formatTariffValue } from "@trenova/shared/lib/tariff-value";
import { cn } from "@trenova/shared/lib/utils";
import type { TariffBill, TariffBillLine, TariffNote } from "@trenova/shared/types/tariff";
import type { ReactNode } from "react";

type ReceiptLine = { line: TariffBillLine; index: number };

function ReceiptRow({
  line,
  converted,
  currency,
}: {
  line: TariffBillLine;
  converted: string | null;
  currency?: string | null;
}) {
  const t = useT();
  const adjustment = line.kind !== "charge";
  const label = tariffLineLabel(line, t);
  const basis =
    line.quantity !== null && line.rate !== null
      ? t(
          "{0} × {1}",
          formatTariffValue(line.quantity, t, { currency }),
          formatTariffValue(line.rate, t, { currency }),
        )
      : null;

  return (
    <li data-kind={line.kind} className="flex items-start justify-between gap-4 py-1.5">
      <span className="flex min-w-0 flex-col gap-0.5">
        <span className="flex min-w-0 items-baseline gap-1.5">
          <span
            className={cn("truncate text-sm", adjustment && "text-foreground-muted italic")}
            title={label}
          >
            {label}
          </span>
          {line.code && (
            <span className="text-foreground-subtle shrink-0 font-mono text-2xs">{line.code}</span>
          )}
        </span>
        {basis && (
          <span className="text-foreground-subtle font-mono text-xs tabular-nums">{basis}</span>
        )}
        {line.limit && (
          <span data-limit={line.limit.kind} className="text-foreground-muted text-xs tabular-nums">
            {tariffLimitText(line.limit, t, currency)}
          </span>
        )}
      </span>
      <span className="flex shrink-0 flex-col items-end gap-0.5">
        <span className="font-mono text-sm tabular-nums">
          {formatTariffValue(line.amount, t, { currency })}
        </span>
        {converted && (
          <span className="text-foreground-subtle font-mono text-xs tabular-nums">{converted}</span>
        )}
      </span>
    </li>
  );
}

function ReceiptGroup({
  label,
  total,
  lines,
  convertedLine,
  currency,
}: {
  label: string;
  total: TariffBill["freightTotal"];
  lines: ReceiptLine[];
  convertedLine: (index: number) => string | null;
  currency?: string | null;
}) {
  const t = useT();
  if (lines.length === 0) return null;

  return (
    <section aria-label={label} className="flex flex-col">
      <div className="text-foreground-muted flex items-baseline justify-between gap-4 text-xs">
        <span className="font-medium">{label}</span>
        <span className="font-mono tabular-nums">{formatTariffValue(total, t, { currency })}</span>
      </div>
      <ul className="flex flex-col">
        {lines.map(({ line, index }) => (
          <ReceiptRow
            // oxlint-disable-next-line react/no-array-index-key -- a bill's lines have no identity but their order
            key={index}
            line={line}
            converted={convertedLine(index)}
            currency={currency}
          />
        ))}
      </ul>
    </section>
  );
}

function ReceiptRemarks({
  label,
  remarks,
  icon,
  className,
}: {
  label: string;
  remarks: TariffNote[];
  icon: ReactNode;
  className: string;
}) {
  if (remarks.length === 0) return null;

  return (
    <ul aria-label={label} className="flex flex-col gap-1">
      {remarks.map((remark, index) => (
        // oxlint-disable-next-line react/no-array-index-key -- remarks have no identity but their order
        <li key={index} className={cn("flex items-start gap-1.5 text-xs", className)}>
          {icon}
          <span>{remark.text}</span>
        </li>
      ))}
    </ul>
  );
}

/**
 * A bill read as a receipt, for a compact surface such as a popover: the freight charges and
 * the accessorials each under their subtotal, every charge with its code and its quantity times
 * rate beneath it, the minimum or maximum that set its amount, then one rule and the total, the
 * total in the billing currency when the bill was converted, and what the tariff asks a person
 * to review and noted. It reads; `BillView` is the bill beside a source it can reveal.
 */
export function BillReceipt({
  bill,
  currency,
  className,
}: {
  bill: TariffBill;
  /** The rating currency's ISO code, when known; money then shows its symbol. */
  currency?: string | null;
  className?: string;
}) {
  const t = useT();
  const converted = bill.converted;
  const convertedLine = (index: number) => {
    const amount = converted?.lines[index];
    return amount ? formatTariffMoney(amount.$q, { currency: converted.currency }) : null;
  };
  const lines = bill.lines.map((line, index) => ({ line, index }));
  const freight = lines.filter(({ line }) => line.freight);
  const accessorials = lines.filter(({ line }) => !line.freight);

  return (
    <section aria-label={t("Bill")} className={cn("flex flex-col gap-4", className)}>
      {bill.lines.length === 0 ? (
        <p className="text-foreground-muted text-sm">{t("This rating billed no charges.")}</p>
      ) : (
        <>
          <ReceiptGroup
            label={t("Freight")}
            total={bill.freightTotal}
            lines={freight}
            convertedLine={convertedLine}
            currency={currency}
          />
          <ReceiptGroup
            label={t("Accessorials")}
            total={bill.accessorialTotal}
            lines={accessorials}
            convertedLine={convertedLine}
            currency={currency}
          />
        </>
      )}

      <div className="border-border-strong flex flex-col gap-1 border-t pt-3">
        <div className="flex items-baseline justify-between gap-4">
          <span className="text-sm font-medium">{t("Total")}</span>
          <span className="font-mono text-base font-semibold tabular-nums">
            {formatTariffValue(bill.total, t, { currency })}
          </span>
        </div>
        {converted && (
          <div className="text-foreground-muted flex items-baseline justify-between gap-4 text-xs">
            <span>{t("In {0} at {1}", converted.currency, converted.rate)}</span>
            <span className="font-mono tabular-nums">
              {formatTariffMoney(converted.total.$q, { currency: converted.currency })}
            </span>
          </div>
        )}
      </div>

      {(bill.reviews.length > 0 || bill.notes.length > 0) && (
        <div className="flex flex-col gap-1.5">
          <ReceiptRemarks
            label={t("Reviews")}
            remarks={bill.reviews}
            icon={<Flag01Icon className="text-warning mt-0.5 size-3.5 shrink-0" />}
            className="text-foreground"
          />
          <ReceiptRemarks
            label={t("Notes")}
            remarks={bill.notes}
            icon={<AnnotationIcon className="text-foreground-subtle mt-0.5 size-3.5 shrink-0" />}
            className="text-foreground-muted"
          />
        </div>
      )}
    </section>
  );
}
