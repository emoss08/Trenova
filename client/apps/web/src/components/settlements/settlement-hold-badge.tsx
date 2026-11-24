import type { DriverSettlementHoldReason } from "@trenova/graphql/generated/graphql";
import { Badge } from "@trenova/shared/components/ui/badge";
import { useT } from "@trenova/shared/i18n/use-t";
import { cn } from "@trenova/shared/lib/utils";
import { holdReasonLabel } from "@/lib/settlement-hold";

/**
 * A settlement a person keeps out of the pay run. The status beside it stays what it was; the
 * badge says it is held, and its title says why when the reader may see the reason.
 */
export function SettlementHoldBadge({
  reason,
  note,
  className,
}: {
  reason?: DriverSettlementHoldReason | null;
  note?: string | null;
  className?: string;
}) {
  const t = useT();
  const label = holdReasonLabel(reason);
  const words = note?.trim() ?? "";
  const title = label
    ? words
      ? t("On hold: {0}, {1}", label, words)
      : t("On hold: {0}", label)
    : t("On hold until a person releases it");

  return (
    <Badge variant="warning" dot className={cn("max-h-5", className)} title={title}>
      {t("On hold")}
    </Badge>
  );
}
