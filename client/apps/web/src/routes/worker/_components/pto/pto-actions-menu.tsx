import { useT } from "@trenova/shared/i18n/use-t";
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
import { CircleCheckIcon, EllipsisIcon } from "lucide-react";
import { useState } from "react";
import { toast } from "sonner";
import { canApplyPTOAction } from "./pto-actions";
import { PTOReasonDialog, type PTOReasonDialogMode } from "./pto-reason-dialog";
import { usePTOInvalidation } from "./use-pto-invalidation";

export type PTOActionsMenuProps = {
  pto: WorkerPTO;
  className?: string;
};

/**
 * The approve / reject / cancel menu for one PTO row. It renders nothing when
 * the viewer holds none of the three permissions for the row's current status,
 * so callers can drop it into any layout without checking first.
 */
export function PTOActionsMenu({ pto, className }: PTOActionsMenuProps) {
  const t = useT();

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
      toast.success(t("PTO approved"), { description: t("The worker has been notified.") });
      setApproveDialogOpen(false);
      void invalidate();
    },
    onError: (error: ApiRequestError) => {
      if (error.isRateLimitError()) {
        toast.error(t("Rate limit exceeded"), {
          description: t("You have exceeded the rate limit. Please try again later."),
        });
        return;
      }
      toast.error(t("Failed to approve PTO"), {
        description: error.message,
      });
    },
  });

  if (!hasActions) return null;

  return (
    <>
      <DropdownMenu>
        <DropdownMenuTrigger
          render={
            <Button
              size="sm"
              variant="ghostInvert"
              className={className ?? "size-6"}
              aria-label={t("PTO actions")}
            >
              <EllipsisIcon />
            </Button>
          }
        />
        <DropdownMenuContent side="bottom" align="end">
          <DropdownMenuGroup>
            <DropdownMenuLabel>{t("Actions")}</DropdownMenuLabel>
            <DropdownMenuSeparator />
            {showApprove ? (
              <DropdownMenuItem
                title={t("Approve")}
                description={t("Approve this PTO request")}
                onClick={() => setApproveDialogOpen(true)}
                color="success"
              />
            ) : null}
            {showReject ? (
              <DropdownMenuItem
                title={t("Reject")}
                description={t("Reject this PTO request")}
                onClick={() => setReasonMode("reject")}
                color="danger"
              />
            ) : null}
            {showCancel ? (
              <DropdownMenuItem
                title={t("Cancel")}
                description={t("Withdraw this PTO request")}
                onClick={() => setReasonMode("cancel")}
                color="warning"
              />
            ) : null}
          </DropdownMenuGroup>
        </DropdownMenuContent>
      </DropdownMenu>
      <AlertDialog open={approveDialogOpen} onOpenChange={setApproveDialogOpen}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogMedia>
              <CircleCheckIcon />
            </AlertDialogMedia>
            <AlertDialogTitle>{t("Approve PTO request")}</AlertDialogTitle>
            <AlertDialogDescription>
              {t("{0} {1} will be notified in Dash and by SMS that their time off ({2}) is approved.", pto.worker?.firstName, pto.worker?.lastName, formatRange(pto.startDate, pto.endDate))}
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel disabled={approving}>{t("Back")}</AlertDialogCancel>
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
