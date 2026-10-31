import { useT, type TranslateFn } from "@trenova/shared/i18n/use-t";
import { Button } from "@trenova/shared/components/ui/button";
import { Kbd } from "@trenova/shared/components/ui/kbd";
import { useHotkey } from "@tanstack/react-hotkeys";
import { Tooltip, TooltipContent, TooltipTrigger } from "@trenova/shared/components/ui/tooltip";
import { useApiMutation } from "@/hooks/use-api-mutation";
import {
  assignBillingQueueBillerGraphQL,
  updateBillingQueueStatusGraphQL,
} from "@/lib/graphql/billing-queue";
import { shipment } from "@/lib/queries/shipment";
import { useAuthStore } from "@trenova/shared/stores/auth-store";
import type {
  BillingQueueItem,
  BillingQueueUpdateStatusInput,
} from "@trenova/shared/types/billing-queue";
import { useQuery } from "@tanstack/react-query";
import {
  AlertTriangleIcon,
  CheckIcon,
  FlipBackwardIcon,
  PauseIcon,
  PlayIcon,
  Send01Icon,
  UserPlus01Icon,
} from "@trenova/shared/components/icons";
import { approveBlocker, type ApproveBlocker } from "@/lib/billing-queue-approve";
import { BillingQueueExceptionPopover } from "./billing-queue-exception-popover";
import { useInvalidateBillingQueue } from "./use-billing-queue-invalidate";

/** Why Approve is closed, in the words the action bar and its tooltip use. */
function approveBlockerText(blocker: ApproveBlocker, t: TranslateFn): string {
  switch (blocker.kind) {
    case "detention":
      return t(
        "{0, plural, one {# detention charge needs approval} other {# detention charges need approval}} before you can approve",
        blocker.count,
      );
    case "rating":
      return t("The tariff held this rating for review; approve the rating first");
    case "review":
      return t(
        "{0, plural, one {# review item needs you} other {# review items need you}} before you can approve",
        blocker.count,
      );
    case "documents":
      return t(
        "{0, plural, one {# required document is missing} other {# required documents are missing}}; add them on the Documents tab",
        blocker.count,
      );
    case "requirement":
      return blocker.message ?? t("The shipment doesn't meet your billing requirements yet");
  }
}

export function BillingQueueActionBar({
  item,
  onAssignBiller,
  onAutoAdvance,
  onShowReview,
}: {
  item: BillingQueueItem;
  onAssignBiller: () => void;
  onAutoAdvance?: () => void;
  /** Sends the biller to the review checks that close Approve. */
  onShowReview?: () => void;
}) {
  const t = useT();

  const currentUser = useAuthStore((s) => s.user);
  const invalidate = useInvalidateBillingQueue();

  const { data: billingReadiness } = useQuery({
    ...shipment.billingReadiness(item.shipmentId ?? ""),
    enabled: item.status === "InReview" && !!item.shipmentId,
  });

  const blocker = approveBlocker(item, billingReadiness);
  const canApprove = blocker === null;
  const blockedReason = blocker ? approveBlockerText(blocker, t) : null;

  const { mutate: updateStatus, isPending: isStatusPending } = useApiMutation({
    mutationFn: (input: BillingQueueUpdateStatusInput) =>
      updateBillingQueueStatusGraphQL(item.id, input),
    resourceName: "BillingQueueItem",
    onSuccess: (_, input) => {
      invalidate();
      if (input.status === "Approved" && onAutoAdvance) {
        onAutoAdvance();
      }
    },
  });

  const { mutate: assignAndReview, isPending: isAssignPending } = useApiMutation({
    mutationFn: (billerId: string) => assignBillingQueueBillerGraphQL(item.id, { billerId }),
    resourceName: "BillingQueueItem",
    onSuccess: () => {
      invalidate();
    },
  });

  const isPending = isStatusPending || isAssignPending;

  const approve = () => updateStatus({ status: "Approved" });
  const hold = () => updateStatus({ status: "OnHold" });
  const hotkeys = { ignoreInputs: true, preventDefault: true };
  useHotkey(
    "A",
    () => {
      if (item.status === "InReview" && canApprove && !isPending) approve();
    },
    hotkeys,
  );
  useHotkey(
    "H",
    () => {
      if ((item.status === "InReview" || item.status === "ReadyForReview") && !isPending) hold();
    },
    hotkeys,
  );

  const actions = (() => {
    switch (item.status) {
      case "ReadyForReview":
        return (
          <>
            <Button size="sm" variant="outline" onClick={onAssignBiller} disabled={isPending}>
              <UserPlus01Icon className="size-3.5" />
              {t("Assign biller")}
            </Button>
            <Button size="sm" variant="outline" onClick={hold} disabled={isPending}>
              <PauseIcon className="size-3.5" />
              {t("Hold")}
            </Button>
            <Button
              onClick={() => currentUser?.id && assignAndReview(currentUser.id)}
              disabled={isPending || !currentUser?.id}
            >
              <PlayIcon className="size-3.5" />
              {t("Start review")}
            </Button>
          </>
        );
      case "InReview":
        return (
          <>
            <BillingQueueExceptionPopover
              itemId={item.id}
              targetStatus="Exception"
              label={t("Exception")}
              icon={<AlertTriangleIcon className="size-3.5" />}
              variant="ghost"
              disabled={isPending}
              onSuccess={invalidate}
            />
            <BillingQueueExceptionPopover
              itemId={item.id}
              targetStatus="SentBackToOps"
              label={t("Send back to ops")}
              icon={<Send01Icon className="size-3.5" />}
              variant="outline"
              disabled={isPending}
              onSuccess={invalidate}
            />
            <Button size="sm" variant="outline" onClick={hold} disabled={isPending}>
              <PauseIcon className="size-3.5" />
              {t("Hold")}
            </Button>
            {/* A disabled button takes no pointer events, so the tooltip hangs on a
                wrapper that does. */}
            <Tooltip>
              <TooltipTrigger render={<span className="inline-flex" />}>
                <Button size="sm" onClick={approve} disabled={isPending || !canApprove}>
                  <CheckIcon className="size-3.5" />
                  {t("Approve")}
                </Button>
              </TooltipTrigger>
              {blockedReason ? (
                <TooltipContent side="top" sideOffset={8}>
                  {blockedReason}
                </TooltipContent>
              ) : null}
            </Tooltip>
          </>
        );
      case "OnHold":
        return (
          <Button
            size="sm"
            onClick={() => updateStatus({ status: "ReadyForReview" })}
            disabled={isPending}
          >
            <PlayIcon className="size-3.5" />
            {t("Resume")}
          </Button>
        );
      case "SentBackToOps":
      case "Exception":
        return (
          <Button
            size="sm"
            onClick={() => updateStatus({ status: "ReadyForReview" })}
            disabled={isPending}
          >
            <FlipBackwardIcon className="size-3.5" />
            {t("Resolve")}
          </Button>
        );
      case "Approved":
        return (
          <Tooltip>
            <TooltipTrigger
              render={
                <Button
                  size="sm"
                  variant="outline"
                  onClick={() => updateStatus({ status: "InReview" })}
                  disabled={isPending}
                >
                  <FlipBackwardIcon className="size-3.5" />
                  {t("Revert to review")}
                </Button>
              }
            />
            <TooltipContent side="top" sideOffset={8}>
              {t(
                "The draft invoice is voided and a new one is built from the corrected charges when you approve again",
              )}
            </TooltipContent>
          </Tooltip>
        );
      default:
        return null;
    }
  })();

  return (
    <footer className="border-border bg-card flex shrink-0 flex-wrap items-center gap-2 border-t px-5 py-2.5">
      {item.status === "InReview" && blocker && blockedReason ? (
        <span
          role="status"
          className="text-warning-foreground flex min-w-0 flex-1 items-center gap-1.5 text-xs"
          data-testid="billing-queue-approve-blocked"
        >
          <AlertTriangleIcon className="size-3.5 shrink-0" />
          <span className="min-w-0">{blockedReason}</span>
          {blocker.kind === "review" && onShowReview ? (
            <Button
              type="button"
              size="xs"
              variant="link"
              className="h-auto shrink-0 p-0 text-xs font-medium"
              onClick={onShowReview}
            >
              {t("Show me")}
            </Button>
          ) : null}
        </span>
      ) : (
        <span className="text-foreground-muted flex flex-1 items-center gap-2.5 text-xs">
          <span className="inline-flex items-center gap-1">
            <Kbd>J</Kbd>
            <Kbd>K</Kbd>
            {t("move")}
          </span>
          {item.status === "InReview" ? (
            <span className="inline-flex items-center gap-1">
              <Kbd>A</Kbd>
              {t("approve")}
            </span>
          ) : null}
          {item.status === "InReview" || item.status === "ReadyForReview" ? (
            <span className="inline-flex items-center gap-1">
              <Kbd>H</Kbd>
              {t("hold")}
            </span>
          ) : null}
        </span>
      )}
      {actions}
    </footer>
  );
}
