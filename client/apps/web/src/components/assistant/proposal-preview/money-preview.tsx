import type { PreviewMoney } from "@/lib/graphql/agent-preview";
import {
  DescriptionEmpty,
  DescriptionItem,
  DescriptionList,
} from "@trenova/shared/components/ui/description-list";
import { useT } from "@trenova/shared/i18n/use-t";
import { ArrowRightIcon } from "lucide-react";
import { formatPreviewAmount } from "./preview-format";
import { HiddenValue } from "./value-change";

function Amount({ value, currency }: { value: string | null; currency: string }) {
  const text = formatPreviewAmount(value, currency);

  return text === null ? <DescriptionEmpty /> : <span className="font-mono">{text}</span>;
}

/** Before and after, or only the side that exists: a new charge has no before. */
function BeforeAfter({
  before,
  after,
  currency,
}: {
  before: string | null;
  after: string | null;
  currency: string;
}) {
  const t = useT();
  const hasBefore = formatPreviewAmount(before, currency) !== null;

  return (
    <span className="inline-flex items-baseline gap-1.5">
      {hasBefore && (
        <>
          <span className="text-foreground-subtle line-through">
            <Amount value={before} currency={currency} />
          </span>
          <ArrowRightIcon aria-hidden className="text-foreground-subtle size-3 self-center" />
          <span className="sr-only">{t("becomes")}</span>
        </>
      )}
      <Amount value={after} currency={currency} />
    </span>
  );
}

/**
 * The amounts a write would move, line by line, with the totals before and
 * after and the difference. The difference is a fact, not a verdict, so it is
 * signed and drawn in ink: whether more is better depends on whose money it is.
 */
export function MoneyPreview({ money }: { money: PreviewMoney }) {
  const t = useT();

  if (money.withheld) {
    return <HiddenValue />;
  }

  const delta = formatPreviewAmount(money.delta, money.currency, { signed: true });
  const hasTotals = money.totalBefore !== null || money.totalAfter !== null;

  return (
    <DescriptionList layout="split">
      {money.lines.map((line, index) => (
        <DescriptionItem key={`${line.label}-${index}`} label={line.label} numeric>
          <BeforeAfter before={line.before} after={line.after} currency={money.currency} />
        </DescriptionItem>
      ))}
      {hasTotals && (
        <DescriptionItem label={t("Total")} numeric>
          <BeforeAfter
            before={money.totalBefore}
            after={money.totalAfter}
            currency={money.currency}
          />
        </DescriptionItem>
      )}
      {delta !== null && (
        <DescriptionItem label={t("Difference")} numeric>
          <span className="font-mono">{delta}</span>
        </DescriptionItem>
      )}
    </DescriptionList>
  );
}
