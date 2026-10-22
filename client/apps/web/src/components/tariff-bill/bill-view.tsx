import { Badge } from "@trenova/shared/components/ui/badge";
import { DescriptionItem, DescriptionList } from "@trenova/shared/components/ui/description-list";
import { AnnotationIcon } from "@trenova/shared/components/icons";
import type { TranslateFn } from "@trenova/shared/i18n/runtime";
import { useT } from "@trenova/shared/i18n/use-t";
import { formatTariffMoney, formatTariffValue } from "@trenova/shared/lib/tariff-value";
import { cn } from "@trenova/shared/lib/utils";
import type {
  TariffBill,
  TariffBillLine,
  TariffLineKind,
  TariffRange,
} from "@trenova/shared/types/tariff";

export type BillViewProps = {
  bill: TariffBill;
  /** The rating currency's ISO code, when known; money then shows its symbol. */
  currency?: string | null;
  /**
   * Called with a line's or a note's source range when it is picked, to reveal it in an
   * editor. Without it the bill is read-only text.
   */
  onSelectRange?: (range: TariffRange) => void;
  /** Leaves out the freight and accessorial subtotals, for a tight surface such as a card. */
  totalOnly?: boolean;
  className?: string;
};

/** What the reader calls a line a total clause added; the tariff gives adjustments no label. */
export function tariffLineLabel(line: TariffBillLine, t: TranslateFn): string {
  return adjustmentLabel(line.kind, t) ?? line.label ?? "";
}

function adjustmentLabel(kind: TariffLineKind, t: TranslateFn): string | null {
  switch (kind) {
    case "minimum":
      return t("Minimum charge adjustment");
    case "maximum":
      return t("Maximum charge adjustment");
    case "rounding":
      return t("Rounding");
    case "charge":
      return null;
  }
}

type LineRowProps = {
  line: TariffBillLine;
  converted: string | null;
  currency?: string | null;
  onSelectRange?: (range: TariffRange) => void;
};

function LineRow({ line, converted, currency, onSelectRange }: LineRowProps) {
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
  const range = line.range;
  const body = (
    <>
      <span className="flex min-w-0 flex-col gap-0.5">
        <span className="flex min-w-0 items-center gap-1.5">
          <span
            className={cn("truncate text-sm", adjustment && "text-foreground-muted italic")}
            title={label}
          >
            {label}
          </span>
          {line.code && (
            <Badge variant="neutral" appearance="outline" className="font-mono">
              {line.code}
            </Badge>
          )}
        </span>
        {basis && (
          <span className="text-foreground-subtle font-mono text-xs tabular-nums">{basis}</span>
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
    </>
  );
  const rowClass = "flex w-full items-start justify-between gap-3 px-(--cell-px) py-1.5 text-left";
  return (
    <li data-kind={line.kind} data-freight={line.freight || undefined}>
      {onSelectRange && range ? (
        <button
          type="button"
          className={cn(rowClass, "hover:bg-surface-hover ui-focus-ring rounded-md")}
          onClick={() => onSelectRange(range)}
        >
          {body}
        </button>
      ) : (
        <div className={rowClass}>{body}</div>
      )}
    </li>
  );
}

/**
 * A tariff's bill as the customer will be billed: each charge with its code and its quantity
 * times rate, the adjustments the total clauses made (labelled in the reader's language), the
 * notes, the freight, accessorial and grand totals, and the totals in the billing currency when
 * the bill was converted. Charge labels are the customer's text and are shown as written.
 */
export function BillView({
  bill,
  currency,
  onSelectRange,
  totalOnly = false,
  className,
}: BillViewProps) {
  const t = useT();
  const converted = bill.converted;
  const convertedLine = (index: number) => {
    const amount = converted?.lines[index];
    return amount ? formatTariffMoney(amount.$q, { currency: converted.currency }) : null;
  };

  return (
    <section aria-label={t("Bill")} className={cn("flex flex-col gap-3", className)}>
      {bill.lines.length === 0 ? (
        <p className="text-foreground-muted text-sm">{t("This rating billed no charges.")}</p>
      ) : (
        <ul className="divide-border -mx-(--cell-px) flex flex-col divide-y">
          {bill.lines.map((line, index) => (
            <LineRow
              // oxlint-disable-next-line react/no-array-index-key -- a bill's lines have no identity but their order
              key={index}
              line={line}
              converted={convertedLine(index)}
              currency={currency}
              onSelectRange={onSelectRange}
            />
          ))}
        </ul>
      )}

      {bill.notes.length > 0 && (
        <ul aria-label={t("Notes")} className="flex flex-col gap-1">
          {bill.notes.map((note, index) => {
            const range = note.range;
            const text = (
              <>
                <AnnotationIcon className="text-foreground-subtle mt-0.5 size-3.5 shrink-0" />
                <span>{note.text}</span>
              </>
            );
            return (
              // oxlint-disable-next-line react/no-array-index-key -- notes have no identity but their order
              <li key={index} className="text-foreground-muted text-xs">
                {onSelectRange && range ? (
                  <button
                    type="button"
                    className="hover:text-foreground ui-focus-ring flex items-start gap-1.5 rounded-sm text-left"
                    onClick={() => onSelectRange(range)}
                  >
                    {text}
                  </button>
                ) : (
                  <span className="flex items-start gap-1.5">{text}</span>
                )}
              </li>
            );
          })}
        </ul>
      )}

      <DescriptionList layout="split" className="border-t pt-1">
        {!totalOnly && (
          <>
            <DescriptionItem label={t("Freight")} numeric valueClassName="font-mono">
              {formatTariffValue(bill.freightTotal, t, { currency })}
            </DescriptionItem>
            <DescriptionItem label={t("Accessorials")} numeric valueClassName="font-mono">
              {formatTariffValue(bill.accessorialTotal, t, { currency })}
            </DescriptionItem>
          </>
        )}
        <DescriptionItem
          label={t("Total")}
          numeric
          valueClassName="font-mono font-semibold text-foreground"
        >
          {formatTariffValue(bill.total, t, { currency })}
        </DescriptionItem>
        {converted && (
          <>
            <DescriptionItem
              label={t("Total in {0}", converted.currency)}
              numeric
              valueClassName="font-mono"
            >
              {formatTariffMoney(converted.total.$q, { currency: converted.currency })}
            </DescriptionItem>
            <DescriptionItem label={t("Exchange rate")} numeric valueClassName="font-mono">
              {converted.rate}
            </DescriptionItem>
          </>
        )}
      </DescriptionList>
    </section>
  );
}
