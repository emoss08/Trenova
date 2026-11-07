import { SectionPanel } from "@/components/section-panel";
import type { DispatchMovePlanPrice } from "@/lib/graphql/dispatch-console";
import { formatTimeAgo } from "@/lib/time-utils";
import { AssistMark } from "@trenova/shared/components/ui/assist-mark";
import { DescriptionItem, DescriptionList } from "@trenova/shared/components/ui/description-list";
import { useT, type TranslateFn } from "@trenova/shared/i18n/use-t";
import { formatCentsWhole } from "@trenova/shared/lib/utils";
import { UNCOVERED_REASON_LABELS } from "./cost-vocabulary";

/**
 * Why the plan left a move uncovered, in one sentence with the driver and the money it is
 * about. A reason that names a driver has a sentence without one, for a plan that priced
 * the move before any driver was considered.
 */
export function uncoveredSentence(price: DispatchMovePlanPrice, t: TranslateFn): string {
  const name = price.workerName.trim();
  const cost = formatCentsWhole(price.costCents);
  const coverage = formatCentsWhole(price.coveragePriceCents);

  switch (price.uncoveredReason) {
    case "NotWorthIt":
      return name
        ? t(
            "The cheapest driver, {0}, costs {1}, more than the {2} it is worth to cover.",
            name,
            cost,
            coverage,
          )
        : t("Every driver costs more than the {0} it is worth to cover.", coverage);
    case "Outbid":
      return name
        ? t(
            "Its drivers are worth more on other moves. The cheapest, {0}, would cost {1} against the {2} it is worth to cover.",
            name,
            cost,
            coverage,
          )
        : t(
            "Its drivers are worth more on other moves than the {0} it is worth to cover.",
            coverage,
          );
    case "Disqualified":
      return name
        ? t("The nearest driver, {0}, can't take it.", name)
        : t("The nearest drivers can't take it.");
    case "BeyondDeadhead":
      return name
        ? t("The nearest driver, {0}, is past the empty-mile limit.", name)
        : t("The nearest drivers are past the empty-mile limit.");
    case "LocationUnknown":
      return t("No driver near the pickup has a known location.");
    case "NoCandidate":
      return t("No driver could take it.");
    default:
      return t("The last plan left it uncovered.");
  }
}

function Alternatives({ alternatives }: { alternatives: DispatchMovePlanPrice["alternatives"] }) {
  const t = useT();
  if (alternatives.length === 0) return null;

  return (
    <ul className="flex flex-col gap-1" aria-label={t("Other drivers")}>
      {alternatives.map((alternative) => (
        <li key={alternative.workerId} className="text-xs text-muted-foreground tabular-nums">
          {t(
            "Choosing {0} instead adds {1}",
            alternative.workerName,
            formatCentsWhole(alternative.extraCents),
          )}
        </li>
      ))}
    </ul>
  );
}

/**
 * What the last plan said this uncovered move is worth and who should take it: the price of
 * leaving it, the driver the plan chose and what that driver is worth to the whole plan, and
 * the exact extra cost of choosing someone else. When the plan left it uncovered, it says why.
 * The plan is a suggestion and can be old, so it wears the assist mark and says when it priced.
 */
export function PlanPricePanel({
  price,
  nowSeconds,
}: {
  price: DispatchMovePlanPrice;
  nowSeconds: number;
}) {
  const t = useT();
  const priced = t("Priced {0}", formatTimeAgo(price.generatedAt * 1000, nowSeconds * 1000));

  return (
    <SectionPanel
      title={t("Plan price")}
      icon={<AssistMark aria-hidden />}
      hint={priced}
      className="rounded-md"
    >
      <div className="flex flex-col gap-2 px-3 py-2.5 text-xs">
        {price.workerId ? (
          <>
            <p className="text-sm font-semibold tabular-nums">
              {t("Worth {0} to cover", formatCentsWhole(price.coveragePriceCents))}
            </p>
            <DescriptionList layout="inline" className="text-xs">
              <DescriptionItem label={t("Planned driver")}>{price.workerName}</DescriptionItem>
              <DescriptionItem label={t("Cost to cover")} numeric>
                {formatCentsWhole(price.costCents)}
              </DescriptionItem>
            </DescriptionList>
            {price.driverValueCents > 0 ? (
              <p className="tabular-nums">
                {t(
                  "{0} is worth {1} to the plan",
                  price.workerName,
                  formatCentsWhole(price.driverValueCents),
                )}
              </p>
            ) : null}
          </>
        ) : (
          <>
            <p className="text-sm font-semibold text-warning-foreground">
              {price.uncoveredReason
                ? UNCOVERED_REASON_LABELS[price.uncoveredReason]
                : t("Left uncovered")}
            </p>
            <p className="text-muted-foreground tabular-nums">{uncoveredSentence(price, t)}</p>
          </>
        )}
        <Alternatives alternatives={price.alternatives} />
      </div>
    </SectionPanel>
  );
}
