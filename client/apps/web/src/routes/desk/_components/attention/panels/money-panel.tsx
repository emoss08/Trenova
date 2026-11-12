import type { WatchtowerPanel } from "@/lib/graphql/watchtower";
import { Badge } from "@trenova/shared/components/ui/badge";
import { useT, type TranslateFn } from "@trenova/shared/i18n/use-t";
import { cn } from "@trenova/shared/lib/utils";
import { formatFactTime } from "../../watchtower/lanes";
import { PanelBody, PanelCard } from "./panel-parts";
import { agoOrIn } from "./panel-time";

type MoneyPanelData = NonNullable<WatchtowerPanel["money"]>;

function stakeWords(t: TranslateFn, money: MoneyPanelData): { eyebrow: string; label: string } {
  switch (money.stake) {
    case "Advance":
      return { eyebrow: t("Factoring"), label: t("Advances not yet funded") };
    case "Chargeback":
      return { eyebrow: t("Factoring"), label: t("Owed back if it is charged back") };
    default:
      return { eyebrow: t("Billing"), label: t("Cannot be invoiced as it stands") };
  }
}

/**
 * Money that waits on someone: the amount first and large, who it waits on,
 * and the day it turns into a loss. An invoice that cannot go out says why in
 * place of an amount it does not have.
 */
export function MoneyPanel({
  money,
  now,
  timezone,
}: {
  money: MoneyPanelData;
  now: number;
  timezone: string;
}) {
  const t = useT();
  const words = stakeWords(t, money);
  const past = money.dueAt > 0 && money.dueAt <= now;

  return (
    <PanelCard
      eyebrow={words.eyebrow}
      title={money.party || words.label}
      badge={
        money.dueAt > 0 ? (
          <Badge variant={past ? "danger" : "warning"} dot>
            {past ? t("Past due") : t("Due {0}", agoOrIn(t, money.dueAt, now))}
          </Badge>
        ) : undefined
      }
    >
      <PanelBody className="flex flex-col gap-2">
        {money.amount !== "" && (
          <div className="flex items-baseline gap-2.5">
            <span
              className={cn(
                "text-2xl/none font-semibold tabular-nums",
                past ? "text-dsk-error" : "text-dsk-fg",
              )}
            >
              {money.amount}
            </span>
            <span className="text-dsk-sm text-dsk-subtle">
              {money.count > 1
                ? t(
                    "{0} · {1, plural, one {# invoice} other {# invoices}}",
                    words.label,
                    money.count,
                  )
                : words.label}
            </span>
          </div>
        )}
        {money.reason !== "" && <p className="text-dsk-base text-dsk-fg2">{money.reason}</p>}
        {money.dueAt > 0 && (
          <p className="text-xs text-dsk-subtle">
            {past
              ? t("Was due {0}.", formatFactTime(t, money.dueAt, now, timezone))
              : t("Due {0}.", formatFactTime(t, money.dueAt, now, timezone))}
          </p>
        )}
      </PanelBody>
    </PanelCard>
  );
}
