import { useT } from "@trenova/shared/i18n/use-t";
import { cn } from "@trenova/shared/lib/utils";

function Line({ className }: { className?: string }) {
  return <span className={cn("block h-2.5 ui-shimmer rounded-sm", className)} />;
}

const TAB_WIDTHS = ["w-13", "w-10", "w-17", "w-19", "w-19", "w-11", "w-8", "w-6", "w-11", "w-12"];

/**
 * The worker page before the record answers: the header band with the mark, name, badges, meta
 * line and ten tab stubs, then the overview's clearance band and its two columns, all where the
 * real ones land.
 */
export function WorkerPageSkeleton() {
  const t = useT();
  return (
    <div aria-busy aria-label={t("Loading the worker")} className="flex min-w-0 flex-col">
      <div aria-hidden className="flex flex-col gap-2.5 border-b bg-card px-6 pt-3.5">
        <div className="flex items-start gap-3.5">
          <span className="block size-10 shrink-0 ui-shimmer rounded-md" />
          <div className="flex flex-1 flex-col gap-2 pt-1">
            <div className="flex items-center gap-2">
              <Line className="h-3.5 w-36" />
              <Line className="h-4.5 w-14 rounded-sm" />
              <Line className="h-4.5 w-16 rounded-sm" />
              <Line className="h-4.5 w-10 rounded-sm" />
            </div>
            <div className="flex gap-4">
              <Line className="w-18" />
              <Line className="w-24" />
              <Line className="w-17" />
              <Line className="w-26" />
            </div>
          </div>
          <Line className="hidden h-7 w-30 rounded-md sm:block" />
          <Line className="h-7 w-7 rounded-md" />
        </div>
        <div className="flex h-10 items-center gap-5 overflow-hidden">
          {TAB_WIDTHS.map((width, index) => (
            <Line key={index} className={cn("shrink-0", width)} />
          ))}
        </div>
      </div>
      <div aria-hidden className="flex flex-col gap-4 p-4">
        <div className="grid rounded-lg border bg-card lg:h-67 lg:grid-cols-[19rem_minmax(0,1fr)]">
          <div className="flex flex-col gap-2.5 p-4">
            <Line className="h-3 w-28" />
            <Line className="mt-1.5 h-6 w-30" />
            <Line className="h-3 w-36" />
            <Line className="mt-2.5 w-11/12" />
            <Line className="w-2/3" />
          </div>
          <div className="flex flex-col gap-2.5 border-t border-border-subtle p-4 lg:border-t-0 lg:border-l">
            <Line className="w-24" />
            {Array.from({ length: 4 }, (_, index) => (
              <span key={index} className="block h-9 ui-shimmer rounded-sm" />
            ))}
          </div>
        </div>
        <div className="grid items-start gap-4 lg:grid-cols-[minmax(0,1fr)_24rem]">
          <div className="overflow-hidden rounded-lg border bg-card">
            <div className="flex h-9 items-center border-b px-3">
              <Line className="w-28" />
            </div>
            <div className="flex flex-col divide-y divide-border-subtle">
              {["w-2/3", "w-1/2", "w-3/5", "w-1/2"].map((width, index) => (
                <div
                  key={index}
                  className="grid grid-cols-[1.75rem_minmax(0,1fr)_5rem] items-center gap-3 px-3.5 py-3"
                >
                  <span className="block size-7 ui-shimmer rounded-md" />
                  <div className="flex flex-col gap-1.5">
                    <Line className={width} />
                    <Line className="h-2 w-4/5" />
                  </div>
                  <span className="block h-7 ui-shimmer rounded-md" />
                </div>
              ))}
            </div>
          </div>
          <div className="flex flex-col gap-4">
            <div className="flex flex-col gap-3 rounded-lg border bg-card p-3">
              <Line className="w-18" />
              <Line className="h-4.5 w-3/5" />
              <Line className="h-1.5 w-full" />
              <Line className="h-1.5 w-full" />
              <Line className="h-1.5 w-full" />
            </div>
            <div className="flex flex-col gap-3.5 rounded-lg border bg-card p-3">
              <Line className="w-20" />
              <Line className="w-full" />
              <Line className="w-full" />
              <Line className="w-full" />
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}
