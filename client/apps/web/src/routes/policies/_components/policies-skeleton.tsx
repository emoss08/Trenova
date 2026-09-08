import { KpiCard } from "@/components/kpi/kpi-card";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { cn } from "@trenova/shared/lib/utils";

/**
 * Stands in twice: the page skeleton is the Suspense fallback while the
 * console's chunk arrives, and the card grid inside it is the console's own
 * loading state while the policies are read. Sharing the grid between the two
 * keeps the page from reflowing between "loading the code" and "loading the
 * policies".
 *
 * Every measurement mirrors the loaded layout: three compact KPI cards, the
 * scope toolbar, and a grid of policy cards each with the document mark, the
 * title and summary, the version and audience pills, and the signed-by line
 * along the bottom. Widths are fixed so the shapes never reshuffle.
 */

const KPI_LABEL_WIDTHS = ["w-12", "w-24", "w-14"] as const;
const KPI_SUB_WIDTHS = ["w-3/4", "w-2/3", "w-full"] as const;
const POLICY_CARDS = [
  { title: "w-3/4", summary: "w-full" },
  { title: "w-1/2", summary: "w-4/5" },
  { title: "w-2/3", summary: "w-3/5" },
  { title: "w-3/5", summary: "w-11/12" },
  { title: "w-4/5", summary: "w-2/3" },
  { title: "w-1/2", summary: "w-3/4" },
] as const;

function OverviewSkeleton() {
  return (
    <div className="grid grid-cols-6 gap-3">
      {KPI_LABEL_WIDTHS.map((width, index) => (
        <KpiCard key={index} span={2} density="compact">
          <div className="flex min-h-[14px] items-center gap-1.5">
            <Skeleton className="size-1.5 rounded-full" />
            <Skeleton className="size-[11px] rounded-sm" />
            <Skeleton className={cn("h-2.5", width)} />
          </div>
          <Skeleton className="h-6.5 w-8" />
          <Skeleton className={cn("mt-auto h-2.5", KPI_SUB_WIDTHS[index])} />
        </KpiCard>
      ))}
    </div>
  );
}

function ToolbarSkeleton() {
  return (
    <div className="flex flex-wrap items-center justify-between gap-3">
      <div className="flex items-center gap-2">
        <Skeleton className="h-7 w-44 rounded-md" />
        <Skeleton className="size-3.5 rounded-full" />
      </div>
      <Skeleton className="h-8 w-32 rounded-md" />
    </div>
  );
}

/**
 * The card grid on its own, for the console to draw while the policies are
 * read; the loaded cards land in the same cells their outlines were.
 */
export function PolicyCardsSkeleton() {
  return (
    <ul
      aria-busy
      aria-label="Loading policies"
      className="grid gap-3 sm:grid-cols-2 xl:grid-cols-3"
    >
      {POLICY_CARDS.map((card, index) => (
        <li
          key={index}
          aria-hidden
          className="border-border/80 flex flex-col gap-3 rounded-lg border p-3"
        >
          <div className="flex items-start gap-3">
            <Skeleton className="size-7 shrink-0 rounded-md" />
            <div className="flex min-w-0 flex-1 flex-col gap-1.5 pt-0.5">
              <Skeleton className={cn("h-3.5", card.title)} />
              <Skeleton className={cn("h-3", card.summary)} />
            </div>
          </div>
          <div className="flex flex-wrap items-center gap-1.5">
            <Skeleton className="h-5 w-10 rounded-full" />
            <Skeleton className="h-5 w-16 rounded-full" />
            <Skeleton className="h-5 w-18 rounded-full" />
          </div>
          <div className="mt-auto flex items-center justify-between gap-2 border-t pt-3">
            <Skeleton className="h-6 w-28 rounded-md" />
          </div>
        </li>
      ))}
    </ul>
  );
}

/**
 * The root announces itself once; everything inside is shape, not content,
 * so it is hidden from assistive technology.
 */
export function PoliciesSkeleton() {
  return (
    <div className="flex flex-col gap-4" aria-busy aria-label="Loading policies">
      <div className="contents" aria-hidden>
        <OverviewSkeleton />
        <ToolbarSkeleton />
        <PolicyCardsSkeleton />
      </div>
    </div>
  );
}
