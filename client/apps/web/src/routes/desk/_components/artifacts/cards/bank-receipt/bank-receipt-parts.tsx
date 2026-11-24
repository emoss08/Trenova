import { useT } from "@trenova/shared/i18n/use-t";
import { cardDay, minorMoney } from "../../record-card/record-card-values";

/**
 * The receipt as the bank statement wrote it: the day the money came in, the
 * words the bank put beside it, and the amount. This is what a cash clerk
 * holds the posted payments against.
 */
export function BankLine({
  receiptDate,
  reference,
  memo,
  amountMinor,
}: {
  receiptDate: number;
  reference: string;
  memo: string | undefined;
  amountMinor: number;
}) {
  const t = useT();
  const words =
    [reference, memo].filter(Boolean).join(" · ") || t("The bank gave no reference or memo");
  return (
    <div
      className="grid grid-cols-[auto_1fr_auto] items-baseline gap-3 rounded-md border px-3 py-2"
      aria-label={t("The bank's line")}
    >
      <span className="font-mono text-xs text-muted-foreground tabular-nums">
        {cardDay(receiptDate)}
      </span>
      <span className="min-w-0 truncate font-mono text-xs" title={words}>
        {words}
      </span>
      <span className="font-mono text-sm tabular-nums">{minorMoney(amountMinor)}</span>
    </div>
  );
}
