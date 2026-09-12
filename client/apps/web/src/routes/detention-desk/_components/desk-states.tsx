import { useT } from "@trenova/shared/i18n/use-t";
import { Button } from "@trenova/shared/components/ui/button";
import { EmptySheet, GhostBar, GhostLine } from "@trenova/shared/components/ui/empty-sheet";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { cn } from "@trenova/shared/lib/utils";
import { TriangleAlertIcon, XIcon } from "lucide-react";
import type { LucideIcon } from "lucide-react";
import type { ReactNode } from "react";

function DeskNotice({
  icon: Icon,
  title,
  body,
  action,
}: {
  icon: LucideIcon;
  title: string;
  body: string;
  action?: ReactNode;
}) {
  return (
    <div className="flex flex-col items-center gap-2 rounded-lg border px-6 py-20 text-center">
      <Icon className="text-muted-foreground size-4" />
      <p className="text-sm">{title}</p>
      <p className="text-muted-foreground max-w-sm text-xs">{body}</p>
      {action ? <div className="mt-2">{action}</div> : null}
    </div>
  );
}

/** Mirrors the real layout so the page does not resize when the data lands. */
export function DeskSkeleton() {
  return (
    <div className="flex flex-col gap-4">
      <div className="grid grid-cols-2 gap-6 border-b pb-5 sm:grid-cols-4">
        {Array.from({ length: 4 }).map((_, index) => (
          <div key={index} className="flex flex-col gap-2">
            <Skeleton className="h-2.5 w-20" />
            <Skeleton className="h-6 w-28" />
            <Skeleton className="h-3 w-32" />
          </div>
        ))}
      </div>
      <div className="flex items-center justify-between border-b pb-2">
        <Skeleton className="h-4 w-72" />
        <Skeleton className="h-6 w-56" />
      </div>
      <div className="flex flex-col gap-px">
        {Array.from({ length: 8 }).map((_, index) => (
          <Skeleton key={index} className="h-13 w-full rounded-none" />
        ))}
      </div>
    </div>
  );
}

export function DeskError({ onRetry }: { onRetry: () => void }) {
  const t = useT();

  return (
    <DeskNotice
      icon={TriangleAlertIcon}
      title={t("The detention desk could not be loaded")}
      body={t(
        "The clocks are still running on the server and nothing has been lost — this screen just cannot read them right now.",
      )}
      action={
        <Button variant="outline" size="sm" onClick={onRetry}>
          {t("Try again")}
        </Button>
      }
    />
  );
}

/**
 * The desk as it will look with drivers on docks: the four figures along
 * the top, then a row per stop with its facility, the free-time clock as a
 * track, the notice column and the money at the right, in the order the
 * real row sets them.
 */
const GHOST_STOPS: readonly { facility: string; context: string; clock: number; money: string }[] =
  [
    { facility: "w-2/5", context: "w-3/5", clock: 85, money: "w-14" },
    { facility: "w-1/2", context: "w-2/5", clock: 60, money: "w-12" },
    { facility: "w-1/3", context: "w-1/2", clock: 35, money: "w-10" },
    { facility: "w-2/5", context: "w-3/5", clock: 15, money: "w-10" },
  ];

const GHOST_METRICS = 4;

function DeskRailSketch() {
  return (
    <div className="grid grid-cols-4 gap-6 border-b px-3 pb-4">
      {Array.from({ length: GHOST_METRICS }, (_, index) => (
        <div key={index} className="flex flex-col gap-2">
          <GhostLine className="w-16" />
          <GhostLine className="h-3.5 w-20" />
          <GhostLine className="w-24" />
        </div>
      ))}
    </div>
  );
}

function DeskRowsSketch() {
  return (
    <div className="divide-border/60 flex flex-col divide-y divide-dashed">
      {GHOST_STOPS.map((stop, index) => (
        <div key={index} className="flex items-center gap-3 px-3 py-2.5">
          <span className="bg-muted size-1.5 shrink-0 rounded-full" />
          <div className="flex min-w-0 flex-1 flex-col gap-1.5">
            <GhostLine className={`h-2 ${stop.facility}`} />
            <GhostLine className={stop.context} />
          </div>
          <div className="flex w-36 shrink-0 flex-col gap-1.5">
            <GhostBar share={stop.clock} className="w-full" />
            <GhostLine className="w-24" />
          </div>
          <GhostLine className="w-16 shrink-0" />
          <div className="flex w-20 shrink-0 flex-col items-end gap-1.5">
            <GhostLine className={`h-2 ${stop.money}`} />
            <GhostLine className="w-10" />
          </div>
        </div>
      ))}
    </div>
  );
}

export function DeskEmpty() {
  const t = useT();

  return (
    <EmptySheet
      className="py-10"
      sketchClassName="max-w-2xl"
      title={t("No drivers are sitting on a dock")}
      description={t(
        "Stops appear here the moment an arrival is recorded, with the free-time clock and the notice deadline already running.",
      )}
      sketch={
        <div className="border-border/70 bg-card flex flex-col gap-2 rounded-lg border pt-4 text-left">
          <DeskRailSketch />
          <DeskRowsSketch />
        </div>
      }
    />
  );
}

/**
 * The rail and the toolbar stay on screen above this, so the sketch is only
 * the rows that the lane and search have hidden.
 */
export function DeskNoMatches({ onReset, className }: { onReset: () => void; className?: string }) {
  const t = useT();

  return (
    <EmptySheet
      className={cn("py-8", className)}
      sketchClassName="max-w-2xl"
      title={t("No stops match this view")}
      description={t(
        "Nothing on the floor matches the lane and search you are in. Clearing them brings the whole desk back.",
      )}
      action={
        <Button variant="outline" size="sm" onClick={onReset}>
          <XIcon className="size-3.5" />
          {t("Clear filters")}
        </Button>
      }
      sketch={
        <div className="border-border/70 bg-card rounded-lg border text-left">
          <DeskRowsSketch />
        </div>
      }
    />
  );
}
