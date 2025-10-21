import { Badge, BadgeProps } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { formatRange, inclusiveDays } from "@/lib/date";
import { queries } from "@/lib/queries";
import { WorkerPTOSchema } from "@/lib/schemas/worker-schema";
import { api } from "@/services/api";
import { APIError } from "@/types/errors";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { CalendarRange, EllipsisIcon } from "lucide-react";
import { useState } from "react";
import { toast } from "sonner";
import { usePTOTypeMeta } from ".";
import { PTORejectionDialog } from "../pto-rejection-dialog";

function UpcomingContentOuter({ children }: { children: React.ReactNode }) {
  return <div className="min-w-0 flex-1">{children}</div>;
}

function UpcomingContentInner({ children }: { children: React.ReactNode }) {
  return (
    <div className="flex items-center justify-between gap-2">{children}</div>
  );
}

export function UpcomingPTOContent({ pto }: { pto: WorkerPTOSchema }) {
  const queryClient = useQueryClient();
  const [rejectPTODialogOpen, setRejectPTODialogOpen] = useState(false);

  const { mutateAsync: approvePTO } = useMutation({
    mutationFn: () => api.worker.approvePTO(pto.id),
    onSuccess: () => {
      toast.success("PTO approved");
      queryClient.invalidateQueries({
        queryKey: [...queries.worker.listUpcomingPTO._def] as string[],
      });
    },
    onError: (error: APIError) => {
      if (error.isValidationError()) {
        toast.error("Failed to approve PTO", {
          description: error.message,
        });
      }

      if (error.isRateLimitError()) {
        toast.error("Rate limit exceeded", {
          description:
            "You have exceeded the rate limit. Please try again later.",
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
            <DropdownMenuTrigger asChild>
              <Button size="sm" variant="ghostInvert" className="size-6">
                <EllipsisIcon />
              </Button>
            </DropdownMenuTrigger>
            <DropdownMenuContent side="bottom" align="end">
              <DropdownMenuLabel>Actions</DropdownMenuLabel>
              <DropdownMenuSeparator />
              <DropdownMenuItem
                title="Approve"
                description="Approve this PTO request"
                onClick={() => {
                  approvePTO();
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

function PTOHeader({ pto }: { pto: WorkerPTOSchema }) {
  const { worker, type } = pto;
  const { label, badgeVariant } = usePTOTypeMeta(type);

  return (
    <div className="flex items-center gap-2 min-w-0">
      <span className="truncate font-medium">
        {worker?.firstName} {worker?.lastName}
      </span>
      <Badge
        withDot={false}
        variant={badgeVariant as BadgeProps["variant"]}
        className="shrink-0 gap-1 px-2 py-0.5 text-[11px] leading-4"
      >
        {label}
      </Badge>
    </div>
  );
}

function PTODateRange({ pto }: { pto: WorkerPTOSchema }) {
  const { startDate, endDate } = pto;
  const range = formatRange(startDate, endDate);
  const days = inclusiveDays(startDate, endDate);

  return (
    <PTODateRangeInner>
      <CalendarRange className="size-3.5" aria-hidden />
      <span className="tabular-nums">{range}</span>
      <span aria-hidden>•</span>
      <span className="tabular-nums">{days}d</span>
    </PTODateRangeInner>
  );
}

function PTODateRangeInner({ children }: { children: React.ReactNode }) {
  return (
    <div className="mt-0.5 flex items-center gap-1 text-xs text-muted-foreground shrink-0">
      {children}
    </div>
  );
}
