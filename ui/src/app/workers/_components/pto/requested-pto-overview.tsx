/*
 * Copyright 2023-2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

import { Avatar, AvatarFallback, AvatarImage } from "@/components/ui/avatar";
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
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from "@/components/ui/popover";
import { ScrollArea, ScrollAreaShadow } from "@/components/ui/scroll-area";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { broadcastQueryInvalidation } from "@/hooks/use-invalidate-query";
import { dateToUnixTimestamp, formatRange, inclusiveDays } from "@/lib/date";
import { queries } from "@/lib/queries";
import { WorkerPTOSchema } from "@/lib/schemas/worker-schema";
import { cn } from "@/lib/utils";
import { api } from "@/services/api";
import { APIError } from "@/types/errors";
import { PTOStatus } from "@/types/worker";
import { useMutation, useQuery } from "@tanstack/react-query";
import { CalendarRange, EllipsisIcon, FilterIcon } from "lucide-react";
import { memo, useMemo, useState } from "react";
import { toast } from "sonner";
import { PTORejectionDialog } from "./pto-rejection-dialog";

export default function RequestedPTOOverview() {
  const defaultStart = dateToUnixTimestamp(new Date());

  const [type, setType] = useState<WorkerPTOSchema["type"] | undefined>(
    undefined,
  );

  const [startDate, setStartDate] = useState<number | undefined>(defaultStart);
  const [endDate, setEndDate] = useState<number | undefined>(undefined);

  const hasActiveFilters = Boolean(type || endDate);
  const toInput = (unix?: number) => {
    if (!unix) return "";
    const d = new Date(unix * 1000);
    const yyyy = d.getFullYear();
    const mm = String(d.getMonth() + 1).padStart(2, "0");
    const dd = String(d.getDate()).padStart(2, "0");
    return `${yyyy}-${mm}-${dd}`;
  };

  const query = useQuery({
    ...queries.worker.listUpcomingPTO({
      filter: { limit: 20, offset: 0 },
      type,
      status: PTOStatus.Requested,
      startDate,
      endDate,
    }),
  });

  return (
    <div className="flex flex-col gap-1 h-fit col-span-4 w-full">
      <div className="flex items-center justify-between">
        <h3 className="text-lg font-medium font-table">Requested PTO</h3>
        <div className="flex items-center gap-1">
          <Popover>
            <PopoverTrigger asChild>
              <Button variant="outline" className="relative size-6">
                <FilterIcon className="size-4" />
                {hasActiveFilters && (
                  <span className="absolute -right-1 -top-1 h-2 w-2 rounded-full bg-primary" />
                )}
              </Button>
            </PopoverTrigger>
            <PopoverContent align="end" className="w-72 p-3">
              <div className="grid gap-3">
                <div className="grid gap-1.5">
                  <Label htmlFor="pto-type">Type</Label>
                  <Select
                    value={(type as string) ?? ""}
                    onValueChange={(v) =>
                      setType(
                        (v || undefined) as WorkerPTOSchema["type"] | undefined,
                      )
                    }
                  >
                    <SelectTrigger id="pto-type">
                      <SelectValue placeholder="All types" />
                    </SelectTrigger>
                    <SelectContent>
                      <SelectItem value="all">All</SelectItem>
                      <SelectItem value="Vacation">Vacation</SelectItem>
                      <SelectItem value="Sick">Sick</SelectItem>
                      <SelectItem value="Holiday">Holiday</SelectItem>
                      <SelectItem value="Bereavement">Bereavement</SelectItem>
                      <SelectItem value="Maternity">Maternity</SelectItem>
                      <SelectItem value="Paternity">Paternity</SelectItem>
                    </SelectContent>
                  </Select>
                </div>

                <div className="grid grid-cols-2 gap-2">
                  <div className="grid gap-1.5">
                    <Label htmlFor="start-date">Start</Label>
                    <Input
                      id="start-date"
                      type="date"
                      value={toInput(startDate)}
                      onChange={(e) =>
                        setStartDate(
                          e.target.value
                            ? dateToUnixTimestamp(
                                new Date(`${e.target.value}T00:00:00`),
                              )
                            : undefined,
                        )
                      }
                    />
                  </div>
                  <div className="grid gap-1.5">
                    <Label htmlFor="end-date">End</Label>
                    <Input
                      id="end-date"
                      type="date"
                      value={toInput(endDate)}
                      onChange={(e) =>
                        setEndDate(
                          e.target.value
                            ? dateToUnixTimestamp(
                                new Date(`${e.target.value}T23:59:59`),
                              ) // inclusive
                            : undefined,
                        )
                      }
                    />
                  </div>
                </div>

                <div className="flex justify-end gap-2 pt-1 border-t border-border/60">
                  <Button
                    variant="ghost"
                    size="sm"
                    onClick={() => {
                      setType(undefined);
                      setEndDate(undefined);
                      setStartDate(defaultStart);
                    }}
                  >
                    Reset
                  </Button>
                </div>
              </div>
            </PopoverContent>
          </Popover>
        </div>
      </div>

      <ScrollArea className="border border-border rounded-md p-3 h-[300px]">
        <div className="flex flex-col gap-2">
          {query.data?.results.map((workerPTO) => (
            <UpcomingPTOCard key={workerPTO.id} workerPTO={workerPTO} />
          ))}
          {query.data?.count === 0 && (
            <div className="flex flex-col text-center items-center justify-center h-[250px]">
              <p className="text-sm font-medium">No PTOs found</p>
              <p className="text-2xs text-muted-foreground">
                Try adjusting your filters or search query.
              </p>
            </div>
          )}
        </div>
        <ScrollAreaShadow />
      </ScrollArea>
    </div>
  );
}

const initials = (first?: string, last?: string) =>
  `${(first?.[0] ?? "").toUpperCase()}${(last?.[0] ?? "").toUpperCase()}`.trim() ||
  "•";

function usePTOTypeMeta(type: WorkerPTOSchema["type"]) {
  return useMemo(() => {
    switch (type) {
      case "Vacation":
        return {
          label: "Vacation",
          badgeVariant: "purple",
          accentClass: "from-purple-600 to-purple-600/5",
        };
      case "Sick":
        return {
          label: "Sick",
          badgeVariant: "red",
          accentClass: "from-red-600 to-red-600/5",
        };
      case "Holiday":
        return {
          label: "Holiday",
          badgeVariant: "info",
          accentClass: "from-blue-600 to-blue-600/5",
        };
      case "Bereavement":
        return {
          label: "Bereavement",
          badgeVariant: "active",
          accentClass: "from-green-600 to-green-600/5",
        };
      case "Maternity":
        return {
          label: "Maternity",
          badgeVariant: "pink",
          accentClass: "from-pink-600 to-pink-600/5",
        };
      case "Paternity":
        return {
          label: "Paternity",
          badgeVariant: "teal",
          accentClass: "from-teal-600 to-teal-600/5",
        };
      default:
        return {
          label: String(type),
          accentClass: "from-muted-foreground/30 to-transparent",
        };
    }
  }, [type]);
}

function usePTOStatusMeta(status: WorkerPTOSchema["status"]) {
  return useMemo(() => {
    switch (status) {
      case "Approved":
        return { label: "Approved", dot: "bg-emerald-500/90" };
      case "Rejected":
        return { label: "Rejected", dot: "bg-rose-500/90" };
      case "Cancelled":
        return { label: "Cancelled", dot: "bg-zinc-500/70" };
      default:
        return { label: "Requested", dot: "bg-muted-foreground/60" };
    }
  }, [status]);
}

export const UpcomingPTOCard = memo(function UpcomingPTOCard({
  workerPTO,
}: {
  workerPTO: WorkerPTOSchema;
}) {
  const { worker, startDate, endDate, type, status } = workerPTO;
  const [rejectPTODialogOpen, setRejectPTODialogOpen] = useState(false);

  const days = inclusiveDays(startDate, endDate);
  const range = formatRange(startDate, endDate);
  const { label, accentClass, badgeVariant } = usePTOTypeMeta(type);
  const statusMeta = usePTOStatusMeta(status);

  const { mutateAsync: approvePTO } = useMutation({
    mutationFn: () => api.worker.approvePTO(workerPTO.id),
    onSuccess: () => {
      toast.success("PTO approved");
      broadcastQueryInvalidation({
        queryKey: [...queries.worker.listUpcomingPTO._def] as string[],
        options: {
          correlationId: `approve-pto-${Date.now()}`,
        },
        config: {
          predicate: true,
          refetchType: "all",
        },
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
      <div
        className="group relative overflow-hidden rounded-xl border border-border p-3 transition-colors"
        role="article"
        aria-label={`${worker?.firstName} ${worker?.lastName} • ${label} • ${range} • ${statusMeta.label}`}
      >
        <div
          className={cn(
            "pointer-events-none absolute inset-y-0 left-0 w-[3px] bg-gradient-to-b",
            accentClass,
          )}
          aria-hidden
        />
        <div className="flex items-center gap-3">
          <Avatar className="h-9 w-9 ring-1 ring-border">
            <AvatarImage
              src={worker?.profilePictureUrl ?? undefined}
              alt={`${worker?.firstName ?? ""} ${worker?.lastName ?? ""}`}
            />
            <AvatarFallback className="text-xs">
              {initials(worker?.firstName, worker?.lastName)}
            </AvatarFallback>
          </Avatar>
          <div className="min-w-0 flex-1">
            <div className="flex items-center justify-between gap-2">
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
            </div>
            <div className="mt-0.5 flex items-center gap-1 text-xs text-muted-foreground shrink-0">
              <CalendarRange className="size-3.5" aria-hidden />
              <span className="tabular-nums">{range}</span>
              <span aria-hidden>•</span>
              <span className="tabular-nums">{days}d</span>
            </div>
          </div>
        </div>
      </div>
      {rejectPTODialogOpen && (
        <PTORejectionDialog
          open={rejectPTODialogOpen}
          onOpenChange={setRejectPTODialogOpen}
          ptoId={workerPTO.id ?? ""}
        />
      )}
    </>
  );
});
