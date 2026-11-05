import { useT, type TranslateFn } from "@trenova/shared/i18n/use-t";
import { billingQueue } from "@/lib/queries/billing-queue";
import { useAuthStore } from "@trenova/shared/stores/auth-store";
import { CheckDone01Icon, Lightning01Icon, XCloseIcon } from "@trenova/shared/components/icons";
import { Button } from "@trenova/shared/components/ui/button";
import { NovaBadge } from "@trenova/shared/components/ui/nova-badge";
import { formatCurrency } from "@trenova/shared/lib/utils";
import type {
  BillingQueueApprovalRun,
  BillingQueueRecommendation,
} from "@trenova/shared/types/billing-queue";
import { useQuery } from "@tanstack/react-query";
import { useState } from "react";
import type { StartBulkRun } from "./use-bulk-run";

type Suggestion = {
  title: string;
  detail: string;
  action: string;
  /** Starting a batch returns its run; opening an item or the hold dialog returns nothing. */
  run: (() => Promise<BillingQueueApprovalRun>) | (() => void);
};

/** Batches whose press clears items outright; the first of them is the primary. */
const CLEARING: readonly BillingQueueRecommendation["kind"][] = [
  "ApproveAgentAgreed",
  "ApproveReady",
];

/**
 * Batches the queue can clear in one press, each naming exactly the items it covers.
 * Rules suggest them for every organization; with Nova reviewing, the batch it
 * agreed with leads and replaces the plain "approve every ready item". A suggestion
 * the biller hides stays hidden until its batch covers different items.
 *
 * Pressing a suggestion that starts a batch shows the button working until the run
 * is accepted, then takes the suggestion away: its items are now that run's, and
 * offering them again would start a second run over the same freight. It comes back
 * if the run is undone or fails.
 */
export function BillingQueueSuggestions({
  run,
  onOpenItem,
  onHoldItems,
  isStarting,
  revertedRunIds,
}: {
  run: StartBulkRun;
  onOpenItem: (itemId: string) => void;
  onHoldItems: (itemIds: string[], note: string) => void;
  isStarting: boolean;
  revertedRunIds: ReadonlySet<string>;
}) {
  const t = useT();
  const userId = useAuthStore((state) => state.user?.id ?? null);
  const { data: recommendations = [] } = useQuery(billingQueue.recommendations());
  const [hidden, setHidden] = useState<ReadonlySet<string>>(() => new Set());
  const [taken, setTaken] = useState<ReadonlyMap<string, string>>(() => new Map());
  const [startingKey, setStartingKey] = useState<string | null>(null);

  const isTaken = (key: string) => {
    const runId = taken.get(key);
    return runId !== undefined && !revertedRunIds.has(runId);
  };

  const press = (key: string, suggestion: Suggestion) => {
    const started = suggestion.run();
    if (!started) return;

    setStartingKey(key);
    void started
      .then((accepted) => setTaken((current) => new Map(current).set(key, accepted.id)))
      .catch(() => {
        // useApiMutation has already reported why the run did not start; the
        // suggestion stays so the biller can try again.
      })
      .finally(() => setStartingKey(null));
  };

  const agentAgreed = recommendations.some((r) => r.kind === "ApproveAgentAgreed");
  const suggestions = recommendations
    .filter((r) => r.count > 0 && !(agentAgreed && r.kind === "ApproveReady"))
    .filter((r) => !hidden.has(hideKey(r)) && !isTaken(hideKey(r)))
    .map((r) => ({
      recommendation: r,
      suggestion: suggestionFor(r, t, { run, onOpenItem, onHoldItems, userId }),
    }))
    .filter(
      (entry): entry is { recommendation: BillingQueueRecommendation; suggestion: Suggestion } =>
        entry.suggestion !== null,
    );

  if (suggestions.length === 0) {
    return null;
  }

  const primaryKind = suggestions.find(({ recommendation }) =>
    CLEARING.includes(recommendation.kind),
  )?.recommendation.kind;

  return (
    <section aria-label={t("Suggested")} className="px-2.5 pt-2.5 pb-1">
      <h2 className="text-foreground-muted flex items-center gap-1.5 px-1 pb-1.5 text-xs font-medium">
        <Lightning01Icon className="size-3" />
        {t("Suggested")}
      </h2>
      <ul className="border-border divide-border flex flex-col divide-y overflow-hidden rounded-md border">
        {suggestions.map(({ recommendation, suggestion }) => (
          <li
            key={recommendation.kind}
            className="hover:bg-surface-hover flex items-center gap-2.5 px-2.5 py-2 text-sm"
          >
            {recommendation.source === "Agent" ? (
              <NovaBadge size="sm" className="shrink-0" />
            ) : (
              <CheckDone01Icon className="text-foreground-subtle size-3.5 shrink-0" />
            )}
            <div className="flex min-w-0 flex-1 flex-col gap-px">
              <p className="text-foreground truncate font-medium">{suggestion.title}</p>
              <p className="text-foreground-muted truncate text-xs">{suggestion.detail}</p>
            </div>
            <Button
              size="xs"
              variant={recommendation.kind === primaryKind ? "default" : "outline"}
              disabled={isStarting || startingKey !== null}
              isLoading={startingKey === hideKey(recommendation)}
              loadingText={suggestion.action}
              onClick={() => press(hideKey(recommendation), suggestion)}
            >
              {suggestion.action}
            </Button>
            <Button
              size="icon-xs"
              variant="ghost"
              aria-label={t("Hide this suggestion")}
              onClick={() => setHidden((current) => new Set([...current, hideKey(recommendation)]))}
            >
              <XCloseIcon className="size-3" />
            </Button>
          </li>
        ))}
      </ul>
    </section>
  );
}

function hideKey(r: BillingQueueRecommendation): string {
  return `${r.kind}:${r.itemIds.join(",")}`;
}

function suggestionFor(
  r: BillingQueueRecommendation,
  t: TranslateFn,
  {
    run,
    onOpenItem,
    onHoldItems,
    userId,
  }: {
    run: StartBulkRun;
    onOpenItem: (itemId: string) => void;
    onHoldItems: (itemIds: string[], note: string) => void;
    userId: string | null;
  },
): Suggestion | null {
  const total = formatCurrency(Number(r.total));
  const first = r.itemIds[0];

  switch (r.kind) {
    case "ApproveAgentAgreed":
      return {
        title: t("Approve the {0} Nova checked", r.count),
        detail: t("All five checks clear; each matches its contract · {0}", total),
        action: t("Approve {0}", r.count),
        run: () => run("Approve", r.itemIds, { assignApprover: true }),
      };
    case "ApproveReady":
      return {
        title: t("{0, plural, one {Approve # ready item} other {Approve # ready items}}", r.count),
        detail: t("All five checks clear on each · {0}", total),
        action: t("Approve {0}", r.count),
        run: () => run("Approve", r.itemIds, { assignApprover: true }),
      };
    case "ReviewAgentConcerns":
      return {
        title: t(
          "{0, plural, one {# ready item Nova questions} other {# ready items Nova questions}}",
          r.count,
        ),
        detail: t("The checks pass, but something looks off · {0}", total),
        action: t("Review {0}", r.count),
        run: () => {
          if (first) onOpenItem(first);
        },
      };
    case "RequestPaperwork":
      return {
        title: t(
          "{0, plural, one {Ask a driver for # signed POD} other {Ask drivers for # signed PODs}}",
          r.count,
        ),
        detail: t("The same request the POD check offers, sent to each driver"),
        action: t("Ask {0}", r.count),
        run: () => run("RequestPaperwork", r.itemIds),
      };
    case "ReviewUnbilled":
      return {
        title: t(
          "{0, plural, one {# charge was never billed} other {# charges were never billed}}",
          r.count,
        ),
        detail: t("Receipts or extra stops with no matching line · {0}", total),
        action: t("Review {0}", r.count),
        run: () => {
          if (first) onOpenItem(first);
        },
      };
    case "HoldDuplicates":
      return {
        title: t(
          "{0, plural, one {Hold # possible duplicate} other {Hold # possible duplicates}}",
          r.count,
        ),
        detail: t("Another invoice already bills the shipment · {0}", total),
        action: t("Hold {0}", r.count),
        run: () => onHoldItems(r.itemIds, t("Possible duplicate invoice")),
      };
    case "AssignToMe":
      if (!userId) return null;
      return {
        title: t(
          "{0, plural, one {Assign # unassigned item to you} other {Assign # unassigned items to you}}",
          r.count,
        ),
        detail: t("Nobody has picked these up · {0}", total),
        action: t("Assign {0}", r.count),
        run: () => run("Assign", r.itemIds, { billerId: userId }),
      };
    case "PostApproved":
      return {
        title: t(
          "{0, plural, one {Post # approved draft} other {Post # approved drafts}}",
          r.count,
        ),
        detail: t("Approved and waiting to post · {0}", total),
        action: t("Post {0}", r.count),
        run: () => run("Post", r.itemIds),
      };
  }
}
