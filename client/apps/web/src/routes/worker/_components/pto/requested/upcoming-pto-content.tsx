import { Badge, type BadgeVariant } from "@trenova/shared/components/ui/badge";
import { formatRange } from "@trenova/shared/lib/date";
import type { WorkerPTO } from "@trenova/shared/types/worker";
import { CalendarRange } from "lucide-react";
import { PTOActionsMenu } from "../pto-actions-menu";
import { usePTOTypeMeta } from "./meta";

function UpcomingContentOuter({ children }: { children: React.ReactNode }) {
  return <div className="min-w-0 flex-1">{children}</div>;
}

function UpcomingContentInner({ children }: { children: React.ReactNode }) {
  return <div className="flex items-center justify-between gap-2">{children}</div>;
}

export function UpcomingPTOContent({ pto }: { pto: WorkerPTO }) {
  return (
    <UpcomingContentOuter>
      <UpcomingContentInner>
        <PTOHeader pto={pto} />
        <PTOActionsMenu pto={pto} />
      </UpcomingContentInner>
      <PTODateRange pto={pto} />
    </UpcomingContentOuter>
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
