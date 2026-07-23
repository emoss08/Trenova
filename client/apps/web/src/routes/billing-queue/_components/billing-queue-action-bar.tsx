import { Button } from "@trenova/shared/components/ui/button";
import { Tooltip, TooltipContent, TooltipTrigger } from "@trenova/shared/components/ui/tooltip";
import { useApiMutation } from "@/hooks/use-api-mutation";
import {
  assignBillingQueueBillerGraphQL,
  updateBillingQueueStatusGraphQL,
} from "@/lib/graphql/billing-queue";
import { queries } from "@/lib/queries";
import { useAuthStore } from "@trenova/shared/stores/auth-store";
import type {
  BillingQueueItem,
  BillingQueueUpdateStatusInput,
} from "@trenova/shared/types/billing-queue";
import { useQuery } from "@tanstack/react-query";
import {
  AlertTriangleIcon,
  CheckIcon,
  PauseIcon,
  PlayIcon,
  SendIcon,
  UndoIcon,
  UserPlusIcon,
} from "lucide-react";
import { toast } from "sonner";
import { BillingQueueExceptionPopover } from "./billing-queue-exception-popover";
import { useInvalidateBillingQueue } from "./use-billing-queue-invalidate";

export function BillingQueueActionBar({
  item,
  onAssignBiller,
  onAutoAdvance,
}: {
  item: BillingQueueItem;
  onAssignBiller: () => void;
  onAutoAdvance?: () => void;
}) {
  const currentUser = useAuthStore((s) => s.user);
  const invalidate = useInvalidateBillingQueue();

  const { data: billingReadiness } = useQuery({
    ...queries.shipment.billingReadiness(item.shipmentId),
    enabled: item.status === "InReview",
  });

  const canApprove = billingReadiness?.canMarkReadyToInvoice !== false;
  const missingCount = billingReadiness?.missingRequirements?.length ?? 0;

  const { mutate: updateStatus, isPending: isStatusPending } = useApiMutation({
    mutationFn: (input: BillingQueueUpdateStatusInput) =>
      updateBillingQueueStatusGraphQL(item.id, input),
    resourceName: "BillingQueueItem",
    onSuccess: (_, input) => {
      invalidate();
      toast.success(`Status updated to ${input.status}`);
      if (input.status === "Approved" && onAutoAdvance) {
        onAutoAdvance();
      }
    },
  });

  const { mutate: assignAndReview, isPending: isAssignPending } = useApiMutation({
    mutationFn: (billerId: string) =>
      assignBillingQueueBillerGraphQL(item.id, { billerId }),
    resourceName: "BillingQueueItem",
    onSuccess: () => {
      invalidate();
      toast.success("Review started");
    },
  });

  const isPending = isStatusPending || isAssignPending;

  switch (item.status) {
    case "ReadyForReview":
      return (
        <div className="flex items-center gap-2 border-b px-4 py-2">
          <Button
            size="sm"
            onClick={() => currentUser?.id && assignAndReview(currentUser.id)}
            disabled={isPending || !currentUser?.id}
          >
            <PlayIcon className="size-3.5" />
            Start Review
          </Button>
          <Button size="sm" variant="outline" onClick={onAssignBiller} disabled={isPending}>
            <UserPlusIcon className="size-3.5" />
            Assign Biller
          </Button>
          <Button
            size="sm"
            variant="outline"
            onClick={() => updateStatus({ status: "OnHold" })}
            disabled={isPending}
          >
            <PauseIcon className="size-3.5" />
            Hold
          </Button>
        </div>
      );

    case "InReview":
      return (
        <div className="flex items-center gap-2 border-b px-4 py-2">
          <Tooltip>
            <TooltipTrigger
              render={
                <Button
                  size="sm"
                  className="bg-green-600 text-white hover:bg-green-700 disabled:opacity-50"
                  onClick={() => updateStatus({ status: "Approved" })}
                  disabled={isPending || !canApprove}
                >
                  <CheckIcon className="size-3.5" />
                  Approve
                </Button>
              }
            />
            {!canApprove && (
              <TooltipContent side="bottom" sideOffset={8}>
                {missingCount > 0
                  ? `${missingCount} required document${missingCount > 1 ? "s" : ""} missing`
                  : "Billing requirements not met"}
              </TooltipContent>
            )}
          </Tooltip>
          <BillingQueueExceptionPopover
            itemId={item.id}
            targetStatus="Exception"
            label="Exception"
            icon={<AlertTriangleIcon className="size-3.5" />}
            variant="destructive"
            disabled={isPending}
            successMessage="Marked as exception"
            onSuccess={invalidate}
          />
          <BillingQueueExceptionPopover
            itemId={item.id}
            targetStatus="SentBackToOps"
            label="Send Back"
            icon={<SendIcon className="size-3.5" />}
            variant="outline"
            disabled={isPending}
            successMessage="Sent back to ops"
            onSuccess={invalidate}
          />
          <Button
            size="sm"
            variant="outline"
            onClick={() => updateStatus({ status: "OnHold" })}
            disabled={isPending}
          >
            <PauseIcon className="size-3.5" />
            Hold
          </Button>
        </div>
      );

    case "OnHold":
      return (
        <div className="flex items-center gap-2 border-b px-4 py-2">
          <Button
            size="sm"
            onClick={() => updateStatus({ status: "ReadyForReview" })}
            disabled={isPending}
          >
            <PlayIcon className="size-3.5" />
            Resume
          </Button>
        </div>
      );

    case "SentBackToOps":
    case "Exception":
      return (
        <div className="flex items-center gap-2 border-b px-4 py-2">
          <Button
            size="sm"
            onClick={() => updateStatus({ status: "ReadyForReview" })}
            disabled={isPending}
          >
            <UndoIcon className="size-3.5" />
            Resolve
          </Button>
        </div>
      );

    case "Approved":
      return (
        <div className="flex items-center gap-2 border-b px-4 py-2">
          <Button
            size="sm"
            variant="outline"
            onClick={() => updateStatus({ status: "InReview" })}
            disabled={isPending}
          >
            <UndoIcon className="size-3.5" />
            Revert to Review
          </Button>
        </div>
      );

    default:
      return null;
  }
}
