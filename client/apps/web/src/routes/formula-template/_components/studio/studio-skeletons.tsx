import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { cn } from "@trenova/shared/lib/utils";

/*
 * The studio's loading shapes. Each lazy piece has the outline of what
 * replaces it, at its size, so nothing moves when the piece lands: the form
 * column keeps its rhythm, the rail keeps its height, the editor keeps its
 * lines. Every one is drawn from `Skeleton`, which spends `ui-shimmer`.
 */

const EDITOR_LINES = [58, 72, 44, 36] as const;
const REFERENCE_ROWS = [64, 52, 70, 46, 58, 40, 66, 50] as const;
const THREAD_LINES = [92, 84, 60] as const;

function Rows({ widths, height = "h-3.5" }: { widths: readonly number[]; height?: string }) {
  return (
    <>
      {widths.map((width, index) => (
        // oxlint-disable-next-line react/no-array-index-key -- a fixed outline that never reorders
        <Skeleton key={index} className={height} style={{ width: `${width}%` }} />
      ))}
    </>
  );
}

/** A section's title row: icon, title and its line of purpose, with an action on the right. */
function SectionHeadSkeleton({ action = true }: { action?: boolean }) {
  return (
    <div className="flex items-end justify-between gap-3">
      <div className="flex flex-col gap-1.5">
        <Skeleton className="h-5 w-32" />
        <Skeleton className="h-3 w-72" />
      </div>
      {action && <Skeleton className="h-6 w-20" />}
    </div>
  );
}

/** The expression section: its head, the editor at its resting height, and the status bar. */
export function ExpressionSectionSkeleton() {
  return (
    <section className="flex flex-col gap-3" aria-busy>
      <SectionHeadSkeleton />
      <div className="bg-card overflow-hidden rounded-lg border">
        <div className="flex min-h-30 flex-col gap-2.5 px-4 py-3">
          <Rows widths={EDITOR_LINES} />
        </div>
        <div className="bg-sunken flex h-7 items-center gap-3 border-t px-3">
          <Skeleton className="h-3 w-14" />
          <Skeleton className="h-3 w-40" />
        </div>
      </div>
    </section>
  );
}

/** The breakdown section: its head and two lines. */
export function BreakdownSectionSkeleton() {
  return (
    <section className="flex flex-col gap-3" aria-busy>
      <SectionHeadSkeleton />
      <div className="bg-card flex flex-col gap-2 rounded-lg border p-3">
        <Skeleton className="h-8 w-full" />
        <Skeleton className="h-8 w-full" />
      </div>
    </section>
  );
}

/** The preview pane: the load picker, the billed card, and the values it reads. */
export function PreviewPaneSkeleton() {
  return (
    <div className="flex flex-col gap-3 p-3" aria-busy>
      <Skeleton className="h-7 w-48" />
      <div className="bg-card flex flex-col gap-3 rounded-lg border p-3.5">
        <Skeleton className="h-3 w-12" />
        <Skeleton className="h-9 w-36" />
        <Skeleton className="h-3 w-44" />
      </div>
      <Skeleton className="mt-1 h-4 w-40" />
      <div className="flex flex-col gap-2">
        <Rows widths={[100, 100, 100, 100]} height="h-7" />
      </div>
    </div>
  );
}

/** The scenarios pane: the tally, its actions, and three rows. */
export function ScenariosPaneSkeleton() {
  return (
    <div className="flex flex-col gap-3 p-3" aria-busy>
      <div className="flex items-center gap-2">
        <Skeleton className="h-7 w-16" />
        <span className="flex-1" />
        <Skeleton className="h-6 w-28" />
        <Skeleton className="h-6 w-14" />
      </div>
      <div className="bg-card flex flex-col gap-px overflow-hidden rounded-lg border">
        <Rows widths={[100, 100, 100]} height="h-12" />
      </div>
    </div>
  );
}

/** The reference pane: the search, the switch, and a column of names. */
export function ReferencePaneSkeleton() {
  return (
    <div className="flex flex-col" aria-busy>
      <div className="flex flex-col gap-2 border-b px-3 pt-3 pb-2.5">
        <Skeleton className="h-8 w-full" />
        <Skeleton className="h-7 w-full" />
      </div>
      <div className="flex flex-col gap-2.5 p-3">
        <Skeleton className="h-3 w-20" />
        <Rows widths={REFERENCE_ROWS} height="h-5" />
      </div>
    </div>
  );
}

/** Nova's pane: the conversation's outline over the composer. */
export function NovaPaneSkeleton() {
  return (
    <div className="flex h-full min-h-0 flex-col" aria-busy>
      <div className="flex flex-1 flex-col gap-2.5 px-4 pt-6">
        <Skeleton className="h-5 w-3/5" />
        <Skeleton className="mt-2 h-3 w-16" />
        <Rows widths={THREAD_LINES} />
      </div>
      <div className="p-3">
        <div className="bg-card flex flex-col gap-2 rounded-xl border p-3">
          <Skeleton className="h-4 w-2/5" />
          <div className="flex items-center gap-2">
            <Skeleton className="h-7 w-40 rounded-full" />
            <span className="flex-1" />
            <Skeleton className="h-7 w-7 rounded-full" />
          </div>
        </div>
      </div>
    </div>
  );
}

/** The whole studio before its template arrives: the top bar, the trace, the form and the rail. */
export function StudioSkeleton({ className }: { className?: string }) {
  return (
    <div className={cn("bg-canvas flex h-full flex-col overflow-hidden", className)} aria-busy>
      <div className="flex h-13 items-center gap-3 border-b px-3">
        <Skeleton className="size-7 rounded-md" />
        <Skeleton className="h-5 w-48" />
        <Skeleton className="h-5 w-64 rounded-full max-[760px]:hidden" />
        <span className="flex-1" />
        <Skeleton className="h-7 w-32" />
        <Skeleton className="h-7 w-20" />
        <Skeleton className="size-7 rounded-md" />
      </div>
      <div className="grid min-h-0 flex-1 grid-cols-[minmax(0,1fr)] min-[900px]:grid-cols-[minmax(0,1fr)_minmax(360px,420px)] min-[1180px]:grid-cols-[248px_minmax(0,1fr)_minmax(360px,420px)]">
        <div className="hidden flex-col gap-4 border-r p-4 min-[1180px]:flex">
          <Skeleton className="h-4 w-28" />
          <Skeleton className="h-3.5 w-40" />
          <Skeleton className="h-24 w-full" />
          <Skeleton className="h-16 w-full" />
          <Skeleton className="h-20 w-full rounded-lg" />
        </div>
        <div className="mx-auto flex w-full max-w-200 flex-col gap-7 px-8 pt-5 max-[900px]:px-4">
          <div className="flex flex-col gap-2">
            <Skeleton className="h-8 w-2/3" />
            <Skeleton className="h-4 w-full" />
            <Skeleton className="h-6 w-56" />
          </div>
          <ExpressionSectionSkeleton />
          <BreakdownSectionSkeleton />
        </div>
        <div className="bg-card border-border-strong my-2 mr-2 hidden flex-col overflow-hidden rounded-lg border min-[900px]:flex">
          <div className="flex h-10 items-center gap-3 border-b px-3">
            <Skeleton className="h-4 w-14" />
            <Skeleton className="h-4 w-16" />
            <Skeleton className="h-4 w-16" />
          </div>
          <PreviewPaneSkeleton />
        </div>
      </div>
    </div>
  );
}
