import { Badge, type BadgeVariant } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuGroup,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import type { ApiRequestError } from "@/lib/api";
import { formatRange } from "@/lib/date";
import { queries } from "@/lib/queries";
import { apiService } from "@/services/api";
import type { WorkerPTO } from "@/types/worker";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { CalendarRange, EllipsisIcon } from "lucide-react";
import { useState } from "react";
import { toast } from "sonner";
import { PTORejectionDialog } from "../pto-rejection-dialog";
import { usePTOTypeMeta } from "./meta";

function UpcomingContentOuter({ children }: { children: React.ReactNode }) {
  return <div className="min-w-0 flex-1">{children}</div>;
}

function UpcomingContentInner({ children }: { children: React.ReactNode }) {
  return <div className="flex items-center justify-between gap-2">{children}</div>;
}

export function UpcomingPTOContent({ pto }: { pto: WorkerPTO }) {
  const queryClient = useQueryClient();
  const [rejectPTODialogOpen, setRejectPTODialogOpen] = useState(false);

  const { mutateAsync: approvePTO } = useMutation({
    mutationFn: () => apiService.workerService.approvePTO(pto.id),
    onSuccess: () => {
      toast.success("PTO approved");
      void queryClient.invalidateQueries({
        queryKey: [...queries.worker.listUpcomingPTO._def] as string[],
      });
    },
    onError: (error: ApiRequestError) => {
      if (error.isValidationError()) {
        toast.error("Failed to approve PTO", {
          description: error.message,
        });
      }

      if (error.isRateLimitError()) {
        toast.error("Rate limit exceeded", {
          description: "You have exceeded the rate limit. Please try again later.",
        });
      }
    },
  });
  return (
    <>
      <UpcomingContentOuter>
        <UpcomingContentInner>
          <PTOHeader pto={pto} />
          <DropdownMenu>
            <DropdownMenuTrigger
              render={
                <Button size="sm" variant="ghostInvert" className="size-6">
                  <EllipsisIcon />
                </Button>
              }
            />
            <DropdownMenuContent side="bottom" align="end">
              <DropdownMenuGroup>
                <DropdownMenuLabel>Actions</DropdownMenuLabel>
                <DropdownMenuSeparator />
                <DropdownMenuItem
                  title="Approve"
                  description="Approve this PTO request"
                  onClick={() => {
                    void approvePTO();
                  }}
                  color="success"
                />
                <DropdownMenuItem
                  title="Reject"
                  description="Reject this PTO request"
                  onClick={() => {
                    setRejectPTODialogOpen(true);
                  }}
                  color="danger"
                />
              </DropdownMenuGroup>
            </DropdownMenuContent>
          </DropdownMenu>
        </UpcomingContentInner>
        <PTODateRange pto={pto} />
      </UpcomingContentOuter>
      {rejectPTODialogOpen && (
        <PTORejectionDialog
          open={rejectPTODialogOpen}
          onOpenChange={setRejectPTODialogOpen}
          ptoId={pto.id ?? ""}
        />
      )}
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
    <div className="mt-0.5 flex shrink-0 items-center gap-1 text-xs text-muted-foreground">
      {children}
    </div>
  );
}
