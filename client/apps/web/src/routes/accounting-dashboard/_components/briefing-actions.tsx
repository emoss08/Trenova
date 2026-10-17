import { ActionPill } from "@/components/action-pill";
import type { AccountingBriefingAction } from "@/lib/graphql/accounts-receivable";
import { useT, type TranslateFn } from "@trenova/shared/i18n/use-t";
import { formatCurrencyWhole } from "@trenova/shared/lib/utils";
import { BRIEFING_LINK_LABELS, BRIEFING_LINK_TONES, briefingLinkPath } from "./briefing-links";

function amountOf(action: AccountingBriefingAction): string {
  return formatCurrencyWhole(action.amountMinor / 100);
}

export function briefingActionLabel(action: AccountingBriefingAction, t: TranslateFn): string {
  const amount = amountOf(action);

  switch (action.link) {
    case "BrokenPromise":
      return action.customerName
        ? t("Call {0} about the missed {1} promise", action.customerName, amount)
        : t(
            "{0, plural, one {Chase # missed promise} other {Chase # missed promises}} · {1}",
            action.count,
            amount,
          );
    case "CrossingNinety":
      return t(
        "{0, plural, one {Call about # invoice} other {Call about # invoices}} before 90 days · {1}",
        action.count,
        amount,
      );
    case "Uncontacted":
      return t(
        "{0, plural, one {Contact # customer} other {Contact # customers}} not reached in 14 days · {1}",
        action.count,
        amount,
      );
    case "FollowUpsDue":
      return t(
        "{0, plural, one {Work # follow-up that is due} other {Work # follow-ups that are due}}",
        action.count,
      );
    case "PromisesDue":
      return t(
        "{0, plural, one {Confirm # promise due in 2 days} other {Confirm # promises due in 2 days}} · {1}",
        action.count,
        amount,
      );
    case "DueSoon":
      return t(
        "{0, plural, one {Remind on # invoice due this week} other {Remind on # invoices due this week}} · {1}",
        action.count,
        amount,
      );
    case "ShortPayRemainders":
      return t(
        "{0, plural, one {Clear # short-pay balance} other {Clear # short-pay balances}} · {1}",
        action.count,
        amount,
      );
    case "UnappliedCash":
      return t("Apply {0} of unapplied cash", amount);
    case "Drafts":
      return t(
        "{0, plural, one {Post # draft invoice} other {Post # draft invoices}} · {1}",
        action.count,
        amount,
      );
    case "NoCollections":
      return t("Check payments are being recorded · {0} billed, none collected", amount);
    case "DueThisMonth":
      return t(
        "{0, plural, one {Send a statement ahead of # invoice due this month} other {Send statements ahead of # invoices due this month}} · {1}",
        action.count,
        amount,
      );
    default:
      return BRIEFING_LINK_LABELS[action.link];
  }
}

/**
 * What to do about the brief, most pressing first: each one opens the page
 * where it gets done. A brief with nothing to do draws no row at all.
 */
export function BriefingActions({ actions }: { actions: readonly AccountingBriefingAction[] }) {
  const t = useT();

  const items = actions.flatMap((action) => {
    const path = briefingLinkPath(action.link, action.customerId);
    return path ? [{ action, path }] : [];
  });
  if (items.length === 0) return null;

  return (
    <ul aria-label={t("Next steps")} className="flex flex-wrap gap-1.5">
      {items.map(({ action, path }) => (
        <li key={`${action.link}:${action.customerId ?? ""}`}>
          <ActionPill to={path} tone={BRIEFING_LINK_TONES[action.link]}>
            <span className="text-foreground">{briefingActionLabel(action, t)}</span>
          </ActionPill>
        </li>
      ))}
    </ul>
  );
}
