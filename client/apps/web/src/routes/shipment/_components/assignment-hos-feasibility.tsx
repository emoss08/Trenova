import type { DriverFeasibility } from "@/lib/graphql/telematics";
import { queries } from "@/lib/queries";
import { Badge } from "@trenova/shared/components/ui/badge";
import { Button } from "@trenova/shared/components/ui/button";
import {
  Collapsible,
  CollapsibleContent,
  CollapsibleTrigger,
} from "@trenova/shared/components/ui/collapsible";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { formatClockDurationMs } from "@trenova/shared/lib/date";
import { cn } from "@trenova/shared/lib/utils";
import type { BadgeVariant } from "@trenova/shared/types/badge";
import { useQuery } from "@tanstack/react-query";
import { CheckIcon, ChevronDownIcon, TriangleAlertIcon } from "lucide-react";
import { useMemo, useState } from "react";

const FEASIBILITY_STALE_MS = 30_000;

const VERDICT_META: Record<string, { label: string; variant: BadgeVariant }> = {
  feasible: { label: "Feasible", variant: "active" },
  tight: { label: "Tight", variant: "warning" },
  infeasible: { label: "Infeasible", variant: "inactive" },
  unknown: { label: "Unknown", variant: "outline" },
};

const UNKNOWN_VERDICT = { label: "Unknown", variant: "outline" as BadgeVariant };

const DUTY_STATUS_META: Record<string, { label: string; variant: BadgeVariant }> = {
  driving: { label: "Driving", variant: "info" },
  onDuty: { label: "On duty", variant: "warning" },
  offDuty: { label: "Off duty", variant: "outline" },
  sleeperBed: { label: "Sleeper", variant: "purple" },
  yardMove: { label: "Yard move", variant: "teal" },
  personalConveyance: { label: "Personal", variant: "teal" },
};

function verdictMeta(verdict: string): { label: string; variant: BadgeVariant } {
  return VERDICT_META[verdict] ?? UNKNOWN_VERDICT;
}

function FeasibilityRow({
  driver,
  selected,
  onSelect,
}: {
  driver: DriverFeasibility;
  selected: boolean;
  onSelect: ((workerId: string) => void) | null;
}) {
  const verdict = verdictMeta(driver.verdict);
  const duty = driver.dutyStatus ? DUTY_STATUS_META[driver.dutyStatus] : undefined;
  const selectable = onSelect !== null && driver.verdict === "feasible";

  const content = (
    <>
      <div className="flex items-center justify-between gap-2">
        <div className="flex min-w-0 items-center gap-1.5">
          <span
            className={cn(
              "truncate text-xs font-medium",
              selected ? "text-foreground" : "text-foreground/90",
            )}
          >
            {driver.workerName}
          </span>
          {selected && <CheckIcon className="size-3 shrink-0 text-brand" />}
          {duty && (
            <Badge variant={duty.variant} className="h-4 shrink-0 rounded px-1 text-[9px]">
              {duty.label}
            </Badge>
          )}
          {driver.tractorCode && (
            <Badge
              variant="outline"
              className="h-4 shrink-0 rounded border-border px-1 font-mono text-[9px]"
            >
              {driver.tractorCode}
            </Badge>
          )}
        </div>
        <Badge variant={verdict.variant} className="h-4 shrink-0 rounded px-1 text-[9px]">
          {verdict.label}
        </Badge>
      </div>
      <div className="flex flex-wrap items-center gap-x-2 gap-y-0.5 font-table text-[10px] text-muted-foreground tabular-nums">
        <span>Drive {formatClockDurationMs(driver.driveRemainingMs)}</span>
        <span aria-hidden>·</span>
        <span>Shift {formatClockDurationMs(driver.shiftRemainingMs)}</span>
        <span aria-hidden>·</span>
        <span>Cycle {formatClockDurationMs(driver.cycleRemainingMs)}</span>
        {driver.deadheadMiles !== null && (
          <>
            <span aria-hidden>·</span>
            <span>~{Math.round(driver.deadheadMiles)} mi away</span>
          </>
        )}
      </div>
      {driver.verdict !== "feasible" && driver.reasons.length > 0 && (
        <p className="line-clamp-2 text-[10px] leading-snug text-muted-foreground">
          {driver.reasons.join(" · ")}
        </p>
      )}
    </>
  );

  if (selectable) {
    return (
      <button
        type="button"
        onClick={() => onSelect(driver.workerId)}
        className={cn(
          "flex w-full flex-col gap-0.5 rounded-md px-2 py-1.5 text-left transition-colors",
          "hover:bg-muted/60 focus-visible:ring-[3px] focus-visible:ring-ring/50 focus-visible:outline-none",
          selected && "bg-muted/40",
        )}
      >
        {content}
      </button>
    );
  }

  return (
    <div className={cn("flex flex-col gap-0.5 px-2 py-1.5", selected && "rounded-md bg-muted/40")}>
      {content}
    </div>
  );
}

function FeasibilitySkeletonRow() {
  return (
    <div className="flex flex-col gap-1.5 px-2 py-2">
      <div className="flex items-center justify-between gap-2">
        <div className="flex items-center gap-1.5">
          <Skeleton className="h-3 w-28" />
          <Skeleton className="h-3.5 w-12" />
        </div>
        <Skeleton className="h-3.5 w-14" />
      </div>
      <Skeleton className="h-2.5 w-44" />
    </div>
  );
}

export function AssignmentHosFeasibility({
  open,
  shipmentId,
  selectedWorkerId,
  onSelectWorker,
}: {
  open: boolean;
  shipmentId?: string | null;
  selectedWorkerId?: string | null;
  onSelectWorker: (workerId: string) => void;
}) {
  const [expanded, setExpanded] = useState(true);

  const statusQuery = useQuery({
    ...queries.telematics.status(),
    staleTime: 5 * 60 * 1000,
    retry: false,
    refetchOnWindowFocus: false,
    enabled: open,
  });
  const telematicsEnabled = statusQuery.data?.enabled ?? false;

  const feasibilityQuery = useQuery({
    ...queries.telematics.shipmentDriverFeasibility(shipmentId ?? ""),
    staleTime: FEASIBILITY_STALE_MS,
    retry: false,
    refetchOnWindowFocus: false,
    enabled: open && telematicsEnabled && !!shipmentId,
  });

  const drivers = useMemo(() => feasibilityQuery.data ?? [], [feasibilityQuery.data]);
  const feasibleCount = useMemo(
    () => drivers.filter((d) => d.verdict === "feasible").length,
    [drivers],
  );
  const selectedDriver = useMemo(
    () => (selectedWorkerId ? drivers.find((d) => d.workerId === selectedWorkerId) : undefined),
    [drivers, selectedWorkerId],
  );

  if (!open || !shipmentId || !telematicsEnabled) {
    return null;
  }

  let body: React.ReactNode;
  if (feasibilityQuery.isLoading) {
    body = (
      <div className="divide-y divide-border">
        <FeasibilitySkeletonRow />
        <FeasibilitySkeletonRow />
        <FeasibilitySkeletonRow />
      </div>
    );
  } else if (feasibilityQuery.isError) {
    body = (
      <div className="flex flex-col items-center gap-2 px-3 py-4 text-center">
        <p className="text-[10.5px] text-muted-foreground">
          Driver feasibility could not be loaded from Samsara.
        </p>
        <Button
          type="button"
          variant="outline"
          size="xs"
          onClick={() => void feasibilityQuery.refetch()}
        >
          Try again
        </Button>
      </div>
    );
  } else if (drivers.length === 0) {
    body = (
      <p className="px-3 py-4 text-center text-[10.5px] text-muted-foreground">
        No HOS data for any drivers yet.
      </p>
    );
  } else {
    body = (
      <div className="max-h-56 divide-y divide-border overflow-y-auto">
        {drivers.map((driver) => (
          <FeasibilityRow
            key={driver.workerId}
            driver={driver}
            selected={driver.workerId === selectedWorkerId}
            onSelect={driver.workerId === selectedWorkerId ? null : onSelectWorker}
          />
        ))}
      </div>
    );
  }

  return (
    <div className="flex flex-col gap-2 pb-4">
      {selectedDriver?.verdict === "infeasible" && (
        <p className="flex items-start gap-1.5 text-xs text-destructive">
          <TriangleAlertIcon className="mt-0.5 size-3.5 shrink-0" />
          <span>
            Selected driver has insufficient hours
            {selectedDriver.reasons.length > 0 ? ` — ${selectedDriver.reasons.join("; ")}` : ""}
          </span>
        </p>
      )}
      <Collapsible
        open={expanded}
        onOpenChange={setExpanded}
        className="rounded-md border border-border"
      >
        <CollapsibleTrigger className="flex w-full cursor-pointer items-center justify-between gap-2 px-3 py-2 text-left">
          <div className="flex items-center gap-1.5">
            <span className="text-xs font-semibold">HOS feasibility</span>
            {!feasibilityQuery.isLoading && !feasibilityQuery.isError && drivers.length > 0 && (
              <Badge
                variant={feasibleCount > 0 ? "active" : "warning"}
                className="h-4 rounded px-1 text-[9px]"
              >
                {feasibleCount} feasible
              </Badge>
            )}
          </div>
          <ChevronDownIcon
            className={cn(
              "size-3.5 shrink-0 text-muted-foreground transition-transform",
              expanded && "rotate-180",
            )}
          />
        </CollapsibleTrigger>
        <CollapsibleContent>
          <div className="border-t border-border">{body}</div>
        </CollapsibleContent>
      </Collapsible>
    </div>
  );
}
