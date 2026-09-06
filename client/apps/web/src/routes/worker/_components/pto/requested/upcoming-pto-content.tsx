import { usePermission } from "@/hooks/use-permission";
import { approveWorkerPTO } from "@/lib/graphql/worker-mutations";
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogMedia,
  AlertDialogTitle,
} from "@trenova/shared/components/ui/alert-dialog";
import { Badge, type BadgeVariant } from "@trenova/shared/components/ui/badge";
import { Button } from "@trenova/shared/components/ui/button";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuGroup,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@trenova/shared/components/ui/dropdown-menu";
import type { ApiRequestError } from "@trenova/shared/lib/api";
import { formatRange } from "@trenova/shared/lib/date";
import { Operation, Resource } from "@trenova/shared/types/permission";
import type { WorkerPTO } from "@trenova/shared/types/worker";
import { useMutation } from "@tanstack/react-query";
import { CalendarRange, CircleCheckIcon, EllipsisIcon } from "lucide-react";
import { useState } from "react";
import { toast } from "sonner";
import { canApplyPTOAction } from "../pto-actions";
import { PTOReasonDialog, type PTOReasonDialogMode } from "../pto-reason-dialog";
import { usePTOInvalidation } from "../use-pto-invalidation";
import { usePTOTypeMeta } from "./meta";

function UpcomingContentOuter({ children }: { children: React.ReactNode }) {
  return <div className="min-w-0 flex-1">{children}</div>;
}

function UpcomingContentInner({ children }: { children: React.ReactNode }) {
  return <div className="flex items-center justify-between gap-2">{children}</div>;
}

export function UpcomingPTOContent({ pto }: { pto: WorkerPTO }) {
  const invalidate = usePTOInvalidation();
  const { allowed: canApprove } = usePermission(Resource.WorkerPTO, Operation.Approve);
  const { allowed: canReject } = usePermission(Resource.WorkerPTO, Operation.Reject);
  const { allowed: canCancel } = usePermission(Resource.WorkerPTO, Operation.Cancel);
  const [approveDialogOpen, setApproveDialogOpen] = useState(false);
  const [reasonMode, setReasonMode] = useState<PTOReasonDialogMode | null>(null);

  const showApprove = canApprove && canApplyPTOAction(pto.status, "Approve");
  const showReject = canReject && canApplyPTOAction(pto.status, "Reject");
  const showCancel = canCancel && canApplyPTOAction(pto.status, "Cancel");
  const hasActions = showApprove || showReject || showCancel;

  const { mutateAsync: approvePTO, isPending: approving } = useMutation({
    mutationFn: () => approveWorkerPTO(pto.id ?? ""),
    onSuccess: () => {
      toast.success("PTO approved", { description: "The worker has been notified." });
      setApproveDialogOpen(false);
      void invalidate();
    },
    onError: (error: ApiRequestError) => {
      if (error.isRateLimitError()) {
        toast.error("Rate limit exceeded", {
          description: "You have exceeded the rate limit. Please try again later.",
        });
        return;
      }
      toast.error("Failed to approve PTO", {
        description: error.message,
      });
    },
  });

  return (
    <>
      <UpcomingContentOuter>
        <UpcomingContentInner>
          <PTOHeader pto={pto} />
          {hasActions ? (
            <DropdownMenu>
              <DropdownMenuTrigger
                render={
                  <Button
                    size="sm"
                    variant="ghostInvert"
                    className="size-6"
                    aria-label="PTO actions"
                  >
                    <EllipsisIcon />
                  </Button>
                }
              />
              <DropdownMenuContent side="bottom" align="end">
                <DropdownMenuGroup>
                  <DropdownMenuLabel>Actions</DropdownMenuLabel>
                  <DropdownMenuSeparator />
                  {showApprove ? (
                    <DropdownMenuItem
                      title="Approve"
                      description="Approve this PTO request"
                      onClick={() => setApproveDialogOpen(true)}
                      color="success"
                    />
                  ) : null}
                  {showReject ? (
                    <DropdownMenuItem
                      title="Reject"
                      description="Reject this PTO request"
                      onClick={() => setReasonMode("reject")}
                      color="danger"
                    />
                  ) : null}
                  {showCancel ? (
                    <DropdownMenuItem
                      title="Cancel"
                      description="Withdraw this PTO request"
                      onClick={() => setReasonMode("cancel")}
                      color="warning"
                    />
                  ) : null}
                </DropdownMenuGroup>
              </DropdownMenuContent>
            </DropdownMenu>
          ) : null}
        </UpcomingContentInner>
        <PTODateRange pto={pto} />
      </UpcomingContentOuter>
      <AlertDialog open={approveDialogOpen} onOpenChange={setApproveDialogOpen}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogMedia>
              <CircleCheckIcon />
            </AlertDialogMedia>
            <AlertDialogTitle>Approve PTO request</AlertDialogTitle>
            <AlertDialogDescription>
              {pto.worker?.firstName} {pto.worker?.lastName} will be notified in Dash and by SMS
              that their time off ({formatRange(pto.startDate, pto.endDate)}) is approved.
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel disabled={approving}>Back</AlertDialogCancel>
            <AlertDialogAction
              disabled={approving}
              onClick={(event) => {
                event.preventDefault();
                void approvePTO();
              }}
            >
              {approving ? "Approving..." : "Approve"}
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
      {reasonMode ? (
        <PTOReasonDialog
          open
          onOpenChange={(open) => {
            if (!open) setReasonMode(null);
          }}
          ptoIds={[pto.id ?? ""]}
          mode={reasonMode}
        />
      ) : null}
    </>
  );
}

function PTOHeader({ pto }: { pto: WorkerPTO }) {
  const { worker, type } = pto;
  const { label, badgeVariant } = usePTOTypeMeta(type);

  return (
    <div className="flex min-w-0 items-center gap-2">
      <span className="truncate font-medium">
        {worker?.firstName} {worker?.lastName}
      </span>
      <Badge
        variant={badgeVariant as BadgeVariant}
        className="shrink-0 gap-1 px-2 py-0.5 text-[11px] leading-4"
      >
        {label}
      </Badge>
    </div>
  );
}

function PTODateRange({ pto }: { pto: WorkerPTO }) {
  const { startDate, endDate } = pto;
  const range = formatRange(startDate, endDate);

  return (
    <PTODateRangeInner>
      <CalendarRange className="size-3.5" aria-hidden />
      <span className="tabular-nums">{range}</span>
    </PTODateRangeInner>
  );
}

function PTODateRangeInner({ children }: { children: React.ReactNode }) {
  return (
    <div className="text-muted-foreground mt-0.5 flex shrink-0 items-center gap-1 text-xs">
      {children}
    </div>
  );
}
