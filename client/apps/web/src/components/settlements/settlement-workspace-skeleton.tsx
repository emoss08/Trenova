import { KpiStripSkeleton } from "@/components/kpi/kpi-strip-skeleton";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { useT } from "@trenova/shared/i18n/use-t";
import { cn } from "@trenova/shared/lib/utils";

/**
 * The driver and carrier settlement workspaces share one layout: the pay period and its
 * actions over a strip of figures, then the queue, the open settlement and the payee's context
 * in three columns. These are its loading shapes, drawn while the workspace's code or the
 * period's summary is read, and the queue's rows while the period's settlements are.
 */

const QUEUE_ROWS = 6;
const QUEUE_NAME_WIDTHS = ["w-28", "w-24", "w-32", "w-20", "w-28", "w-24"] as const;
const CHIP_WIDTHS = ["w-8", "w-16", "w-12", "w-14", "w-16"] as const;
const LINE_GROUPS = [
  { label: "w-16", rows: ["w-40", "w-32", "w-36"] },
  { label: "w-20", rows: ["w-28", "w-36"] },
] as const;
const RAIL_SECTIONS = [
  { title: "w-24", hint: "w-40", block: "h-16" },
  { title: "w-20", hint: "w-36", block: "h-12" },
  { title: "w-24", hint: "w-32", block: "h-12" },
  { title: "w-16", hint: "w-36", block: "h-10" },
] as const;

/** The queue's settlements while the period's are read: a payee and net pay, a number and a status. */
export function SettlementQueueRowsSkeleton() {
  const t = useT();

  return (
    <ul aria-busy aria-label={t("Loading settlements")}>
      {Array.from({ length: QUEUE_ROWS }, (_, index) => (
        <li key={index} className="border-b last:border-b-0" aria-hidden>
          <div className="flex items-start gap-2 px-2 py-2">
            <Skeleton className="mt-0.5 size-4 shrink-0 rounded-sm" />
            <div className="min-w-0 flex-1">
              <div className="flex items-center gap-1.5 text-xs">
                <Skeleton
                  className={cn("inline-block h-2.5 align-middle", QUEUE_NAME_WIDTHS[index])}
                />
                <Skeleton className="ml-auto inline-block h-2.5 w-14 align-middle" />
              </div>
              <div className="mt-0.5 flex items-center gap-1.5">
                <Skeleton className="h-2 w-16" />
                <Skeleton className="h-5 w-16 rounded-full" />
              </div>
            </div>
          </div>
        </li>
      ))}
    </ul>
  );
}

function QueueSkeleton() {
  return (
    <div className="bg-card flex min-h-0 flex-col overflow-hidden rounded-lg border">
      <div className="flex flex-col gap-2 border-b p-2">
        <Skeleton className="h-8 w-full" />
        <div className="flex flex-wrap gap-1">
          {CHIP_WIDTHS.map((width, index) => (
            <Skeleton key={index} className={cn("h-5 rounded-full", width)} />
          ))}
        </div>
      </div>
      <div className="flex items-center gap-2 border-b px-3 py-1.5 text-xs">
        <Skeleton className="size-4 rounded-sm" />
        <Skeleton className="inline-block h-2.5 w-48 align-middle" />
      </div>
      <div className="min-h-0 flex-1 overflow-hidden">
        <SettlementQueueRowsSkeleton />
      </div>
    </div>
  );
}

function DetailSkeleton() {
  return (
    <div className="flex flex-col gap-4 p-3">
      <div className="flex flex-col gap-3">
        <div className="flex flex-wrap items-center gap-2">
          <Skeleton className="h-5 w-20 rounded-full" />
          <Skeleton className="h-5 w-24 rounded-full" />
          <Skeleton className="ml-auto h-2.5 w-40" />
        </div>
        <KpiStripSkeleton count={4} minItemWidth="8rem" sub={false} />
      </div>
      <div className="flex flex-wrap items-center gap-2">
        <Skeleton className="h-7 w-20" />
        <Skeleton className="h-7 w-24" />
        <Skeleton className="h-7 w-20" />
      </div>
      <div className="flex flex-col gap-4">
        {LINE_GROUPS.map((group, index) => (
          <div key={index}>
            <div className="mb-1 flex items-baseline justify-between text-xs">
              <Skeleton className={cn("inline-block h-2.5 align-middle", group.label)} />
              <Skeleton className="inline-block h-2.5 w-16 align-middle" />
            </div>
            <div className="overflow-hidden rounded-lg border">
              {group.rows.map((width, row) => (
                <div
                  key={row}
                  className="flex items-center gap-3 border-b px-3 py-2 text-xs last:border-b-0"
                >
                  <Skeleton className={cn("inline-block h-2.5 align-middle", width)} />
                  <Skeleton className="ml-auto inline-block h-2.5 w-12 align-middle" />
                  <Skeleton className="inline-block h-2.5 w-16 align-middle" />
                </div>
              ))}
            </div>
          </div>
        ))}
      </div>
    </div>
  );
}

function RailSkeleton() {
  return (
    <div className="bg-card hidden min-h-0 flex-col overflow-hidden rounded-lg border lg:flex">
      <div className="flex flex-col gap-3 p-3">
        <div>
          <div className="text-sm">
            <Skeleton className="inline-block h-3 w-32 align-middle" />
          </div>
          <div className="text-xs">
            <Skeleton className="inline-block h-2.5 w-44 align-middle" />
          </div>
        </div>
        {RAIL_SECTIONS.map((section, index) => (
          <div key={index} className="border-t pt-3 first:border-t-0 first:pt-0">
            <div className="mb-1.5">
              <div className="text-xs">
                <Skeleton className={cn("inline-block h-2.5 align-middle", section.title)} />
              </div>
              <div className="text-2xs">
                <Skeleton className={cn("inline-block h-2 align-middle", section.hint)} />
              </div>
            </div>
            <Skeleton className={cn("w-full", section.block)} />
          </div>
        ))}
      </div>
    </div>
  );
}

type SettlementWorkspaceSkeletonProps = {
  /** How many figures the period's strip shows. */
  figures: number;
  /** The widths of the buttons beside the pay period, in order. */
  actions: readonly string[];
};

/** The whole workspace: the pay period and its actions, the figures, and the three columns. */
export function SettlementWorkspaceSkeleton({
  figures,
  actions,
}: SettlementWorkspaceSkeletonProps) {
  const t = useT();

  return (
    <div
      className="flex min-h-135 flex-1 flex-col gap-3"
      aria-busy
      aria-label={t("Loading the settlement workspace")}
    >
      <div className="contents" aria-hidden>
        <div className="flex flex-col gap-2">
          <div className="flex flex-wrap items-center justify-between gap-2">
            <div className="text-xs">
              <Skeleton className="inline-block h-2.5 w-64 align-middle" />
            </div>
            <div className="flex items-center gap-2">
              {actions.map((width, index) => (
                <Skeleton key={index} className={cn("h-7", width)} />
              ))}
            </div>
          </div>
          <KpiStripSkeleton count={figures} />
        </div>
        <div className="grid min-h-0 flex-1 grid-cols-1 gap-3 lg:grid-cols-[300px_minmax(0,1fr)_300px]">
          <QueueSkeleton />
          <div className="bg-card min-h-0 overflow-hidden rounded-lg border">
            <DetailSkeleton />
          </div>
          <RailSkeleton />
        </div>
      </div>
    </div>
  );
}
