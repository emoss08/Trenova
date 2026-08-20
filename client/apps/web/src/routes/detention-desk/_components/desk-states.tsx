import { Button } from "@trenova/shared/components/ui/button";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { SearchXIcon, TimerIcon, TriangleAlertIcon } from "lucide-react";
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
  return (
    <DeskNotice
      icon={TriangleAlertIcon}
      title="The detention desk could not be loaded"
      body="The clocks are still running on the server and nothing has been lost — this screen just cannot read them right now."
      action={
        <Button variant="outline" size="sm" onClick={onRetry}>
          Try again
        </Button>
      }
    />
  );
}

export function DeskEmpty() {
  return (
    <DeskNotice
      icon={TimerIcon}
      title="No drivers are sitting on a dock"
      body="Stops appear here the moment an arrival is recorded, with the free-time clock and the notice deadline already running."
    />
  );
}

export function DeskNoMatches({ onReset }: { onReset: () => void }) {
  return (
    <DeskNotice
      icon={SearchXIcon}
      title="No stops match this view"
      body="Nothing on the floor matches the lane and search you are in. Clearing them brings the whole desk back."
      action={
        <Button variant="outline" size="sm" onClick={onReset}>
          Clear filters
        </Button>
      }
    />
  );
}
