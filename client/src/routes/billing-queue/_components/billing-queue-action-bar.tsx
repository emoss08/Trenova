import { Button } from "@/components/ui/button";
import { Popover, PopoverContent, PopoverTrigger } from "@/components/ui/popover";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { Textarea } from "@/components/ui/textarea";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import { exceptionReasonLabels } from "@/lib/choices";
import { queries } from "@/lib/queries";
import { apiService } from "@/services/api";
import { useAuthStore } from "@/stores/auth-store";
import {
  exceptionReasonCodeSchema,
  type BillingQueueItem,
  type BillingQueueStatus,
  type ExceptionReasonCode,
} from "@/types/billing-queue";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import {
  AlertTriangleIcon,
  CheckIcon,
  PauseIcon,
  PlayIcon,
  SendIcon,
  UndoIcon,
  UserPlusIcon,
} from "lucide-react";
import { useState } from "react";
import { toast } from "sonner";


export function BillingQueueActionBar({
  item,
  onAssignBiller,
  onAutoAdvance,
}: {
  item: BillingQueueItem;
  onAssignBiller: () => void;
  onAutoAdvance?: () => void;
}) {
  const queryClient = useQueryClient();
  const currentUser = useAuthStore((s) => s.user);

  const { data: billingReadiness } = useQuery({
    ...queries.shipment.billingReadiness(item.shipmentId),
    enabled: item.status === "InReview",
  });

  const canApprove = billingReadiness?.canMarkReadyToInvoice !== false;
  const missingCount = billingReadiness?.missingRequirements?.length ?? 0;

  const invalidateAll = () => {
    void queryClient.invalidateQueries({ queryKey: ["billing-queue-list"] });
    void queryClient.invalidateQueries({ queryKey: ["billingQueue"] });
  };

  const { mutate: updateStatus, isPending: isStatusPending } = useMutation({
    mutationFn: ({
      status,
      exceptionReasonCode,
      exceptionNotes,
      reviewNotes,
      cancelReason,
    }: {
      status: BillingQueueStatus;
      exceptionReasonCode?: ExceptionReasonCode;
      exceptionNotes?: string;
      reviewNotes?: string;
      cancelReason?: string;
    }) =>
      apiService.billingQueueService.updateStatus(item.id, {
        status,
        exceptionReasonCode,
        exceptionNotes,
        reviewNotes,
        cancelReason,
      }),
    onSuccess: (_, variables) => {
      invalidateAll();
      toast.success(`Status updated to ${variables.status}`);
      if (variables.status === "Approved" && onAutoAdvance) {
        onAutoAdvance();
      }
    },
    onError: () => {
      toast.error("Failed to update status");
    },
  });

  const { mutate: assignAndReview, isPending: isAssignPending } = useMutation({
    mutationFn: () =>
      apiService.billingQueueService.assign(item.id, {
        billerId: currentUser?.id ?? "",
      }),
    onSuccess: () => {
      invalidateAll();
      toast.success("Review started");
    },
    onError: () => {
      toast.error("Failed to start review");
    },
  });

  const isPending = isStatusPending || isAssignPending;

  switch (item.status) {
    case "ReadyForReview":
      return (
        <div className="flex items-center gap-2 border-b px-4 py-2">
          <Button
            size="sm"
            onClick={() => assignAndReview()}
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
          <ExceptionPopover
            onSubmit={(reasonCode, notes) =>
              updateStatus({
                status: "Exception",
                exceptionReasonCode: reasonCode,
                exceptionNotes: notes,
              })
            }
            disabled={isPending}
            label="Exception"
            icon={<AlertTriangleIcon className="size-3.5" />}
            variant="destructive"
          />
          <ExceptionPopover
            onSubmit={(reasonCode, notes) =>
              updateStatus({
                status: "SentBackToOps",
                exceptionReasonCode: reasonCode,
                exceptionNotes: notes,
              })
            }
            disabled={isPending}
            label="Send Back"
            icon={<SendIcon className="size-3.5" />}
            variant="outline"
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

function ExceptionPopover({
  onSubmit,
  disabled,
  label,
  icon,
  variant = "outline",
}: {
  onSubmit: (reasonCode: ExceptionReasonCode, notes: string) => void;
  disabled: boolean;
  label: string;
  icon: React.ReactNode;
  variant?: "outline" | "destructive";
}) {
  const [open, setOpen] = useState(false);
  const [reasonCode, setReasonCode] = useState<ExceptionReasonCode | "">("");
  const [notes, setNotes] = useState("");

  const handleSubmit = () => {
    if (!reasonCode) return;
    onSubmit(reasonCode, notes);
    setOpen(false);
    setReasonCode("");
    setNotes("");
  };

  return (
    <Popover open={open} onOpenChange={setOpen}>
      <PopoverTrigger
        render={
          <Button size="sm" variant={variant} disabled={disabled}>
            {icon}
            {label}
          </Button>
        }
      />
      <PopoverContent className="w-80" align="start">
        <div className="flex flex-col gap-3">
          <p className="text-sm font-medium">{label}</p>
          <Select
            value={reasonCode}
            onValueChange={(v) => setReasonCode(v as ExceptionReasonCode)}
            items={exceptionReasonCodeSchema.options.map((code) => ({
              label: exceptionReasonLabels[code],
              value: code,
            }))}
          >
            <SelectTrigger className="w-full">
              <SelectValue placeholder="Select reason..." />
            </SelectTrigger>
            <SelectContent>
              {exceptionReasonCodeSchema.options.map((code) => (
                <SelectItem key={code} value={code}>
                  {exceptionReasonLabels[code]}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
          <Textarea
            placeholder="Notes (required for Exception status)..."
            value={notes}
            onChange={(e) => setNotes(e.target.value)}
            minRows={3}
          />
          <Button size="sm" onClick={handleSubmit} disabled={!reasonCode}>
            Submit
          </Button>
        </div>
      </PopoverContent>
    </Popover>
  );
}
