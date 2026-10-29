import { useApiMutation } from "@/hooks/use-api-mutation";
import { usePermission } from "@/hooks/use-permission";
import { checkDetail, checkTitle, issueOf } from "@/lib/billing-queue-checks";
import { billingQueue } from "@/lib/queries/billing-queue";
import { apiService } from "@/services/api";
import { CheckIcon, Flag01Icon } from "@trenova/shared/components/icons";
import { Alert, AlertDescription, AlertTitle } from "@trenova/shared/components/ui/alert";
import { Button } from "@trenova/shared/components/ui/button";
import { useT } from "@trenova/shared/i18n/use-t";
import { formatUnixInUserTimezone } from "@trenova/shared/lib/date";
import type {
  BillingCheck,
  BillingIssue,
  BillingQueueItem,
} from "@trenova/shared/types/billing-queue";
import { Operation, Resource } from "@trenova/shared/types/permission";
import { useQueryClient } from "@tanstack/react-query";
import { useInvalidateBillingQueue } from "./use-billing-queue-invalidate";

/**
 * The checks an item in review must clear before it can be approved, with each flagged one's
 * choices where the biller already is, and an undo for what was settled. Settling answers with
 * the item, which replaces the cached one, so the charges, these checks and the approve action
 * follow at once.
 */
export function BillingQueueReviewChecks({ item }: { item: BillingQueueItem }) {
  const t = useT();
  const queryClient = useQueryClient();
  const invalidate = useInvalidateBillingQueue();
  const { allowed: canUpdate } = usePermission(Resource.BillingQueue, Operation.Update);

  const settle = (next: BillingQueueItem) => {
    if (next.review) {
      queryClient.setQueryData(billingQueue.get(item.id).queryKey, next);
    }
    invalidate();
  };
  const resolve = useApiMutation({
    mutationFn: ({ issueId, optionKey }: { issueId: string; optionKey: string }) =>
      apiService.billingQueueService.resolveIssue(item.id, issueId, optionKey),
    resourceName: "BillingQueueItem",
    onSuccess: settle,
  });
  const undo = useApiMutation({
    mutationFn: (issueId: string) => apiService.billingQueueService.undoIssue(item.id, issueId),
    resourceName: "BillingQueueItem",
    onSuccess: settle,
  });

  const review = item.review;
  if (item.status !== "InReview" || !review) {
    return null;
  }

  const busy = resolve.isPending || undo.isPending;
  const flagged = review.checks.filter((check) => check.state !== "ok");
  const settled = review.issues.filter((issue) => issue.resolvedAt && issue.undoable);

  if (flagged.length === 0) {
    return (
      <div className="shrink-0 px-4 pt-2" data-testid="billing-queue-review-checks">
        <Alert size="sm" variant="success">
          <CheckIcon />
          <AlertTitle>{t("All five checks clear")}</AlertTitle>
          {settled.length > 0 && (
            <AlertDescription>
              <SettledIssues
                issues={settled}
                canUndo={canUpdate}
                busy={busy}
                onUndo={(issueId) => undo.mutate(issueId)}
              />
            </AlertDescription>
          )}
        </Alert>
      </div>
    );
  }

  return (
    <div className="shrink-0 px-4 pt-2" data-testid="billing-queue-review-checks">
      <Alert size="sm" variant="warning">
        <Flag01Icon />
        <AlertTitle>
          {t(
            "{0, plural, one {# check needs you before this can be approved} other {# checks need you before this can be approved}}",
            flagged.length,
          )}
        </AlertTitle>
        <AlertDescription>
          <ul className="grid w-full gap-2.5">
            {flagged.map((check) => (
              <FlaggedCheck
                key={check.key}
                item={item}
                check={check}
                canAct={canUpdate}
                busy={busy}
                onResolve={(issueId, optionKey) => resolve.mutate({ issueId, optionKey })}
              />
            ))}
          </ul>
          {settled.length > 0 && (
            <SettledIssues
              issues={settled}
              canUndo={canUpdate}
              busy={busy}
              onUndo={(issueId) => undo.mutate(issueId)}
            />
          )}
        </AlertDescription>
      </Alert>
    </div>
  );
}

function FlaggedCheck({
  item,
  check,
  canAct,
  busy,
  onResolve,
}: {
  item: BillingQueueItem;
  check: BillingCheck;
  canAct: boolean;
  busy: boolean;
  onResolve: (issueId: string, optionKey: string) => void;
}) {
  const t = useT();
  const issue = issueOf(item, check);

  return (
    <li className="grid gap-1" data-testid={`billing-queue-check-${check.key}`}>
      <p>
        <span className="text-foreground font-medium">{checkTitle(check.key, t)}</span>
        {" · "}
        {issue ? issue.summary : checkDetail(check, t)}
      </p>
      {issue?.reasoning && <p>{issue.reasoning}</p>}
      {check.key === "biller" && !issue && (
        <p>{t("Start review or assign a biller to clear it.")}</p>
      )}
      {issue && canAct && issue.options.length > 0 && (
        <div className="flex flex-wrap gap-1.5 pt-0.5">
          {issue.options.map((option, index) => {
            const asked = option.effect.kind === "request" && issue.requestedAt;
            return (
              <Button
                key={option.key}
                size="xs"
                variant={index === 0 && !asked ? "default" : "outline"}
                disabled={busy || Boolean(asked)}
                onClick={() => onResolve(issue.id, option.key)}
              >
                {asked
                  ? t(
                      "Asked {0}",
                      formatUnixInUserTimezone(issue.requestedAt, {
                        month: "short",
                        day: "numeric",
                        hour: "numeric",
                        minute: "2-digit",
                      }),
                    )
                  : option.label}
              </Button>
            );
          })}
        </div>
      )}
    </li>
  );
}

function SettledIssues({
  issues,
  canUndo,
  busy,
  onUndo,
}: {
  issues: BillingIssue[];
  canUndo: boolean;
  busy: boolean;
  onUndo: (issueId: string) => void;
}) {
  const t = useT();

  return (
    <ul className="grid w-full gap-1 pt-1">
      {issues.map((issue) => (
        <li key={issue.id} className="flex flex-wrap items-center justify-between gap-x-3">
          <span>{t("Settled · {0}", issue.resolutionText || issue.summary)}</span>
          {canUndo && (
            <Button size="xs" variant="ghost" disabled={busy} onClick={() => onUndo(issue.id)}>
              {t("Undo")}
            </Button>
          )}
        </li>
      ))}
    </ul>
  );
}
