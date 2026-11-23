import { useT } from "@trenova/shared/i18n/use-t";
import { ShimmerLine } from "../tab-skeleton-parts";

function PanelHead({ title = "w-28" }: { title?: string }) {
  return (
    <div className="flex h-9 items-center justify-between gap-2 border-b px-3">
      <ShimmerLine className={title} />
      <ShimmerLine className="w-14" />
    </div>
  );
}

/** The Overview's own layout in shimmer: its loading state and the tab's lazy fallback. */
export function OverviewSkeleton() {
  const t = useT();
  return (
    <div
      aria-busy
      aria-label={t("Loading the overview")}
      className="@container/overview flex min-w-0 flex-col gap-4"
    >
      <div
        aria-hidden
        className="grid rounded-lg border @[1000px]/overview:grid-cols-[19rem_minmax(0,1fr)]"
      >
        <div className="flex flex-col gap-2 border-b border-border-subtle p-4 @[1000px]/overview:border-r @[1000px]/overview:border-b-0">
          <ShimmerLine className="w-28" />
          <ShimmerLine className="mt-2 h-7 w-24" />
          <ShimmerLine className="w-36" />
          <ShimmerLine className="mt-2 w-56" />
          <ShimmerLine className="w-44" />
          <div className="mt-4 flex gap-2">
            <ShimmerLine className="h-7 w-36 rounded-md" />
            <ShimmerLine className="h-7 w-28 rounded-md" />
          </div>
        </div>
        <div className="flex min-w-0 flex-col gap-3 px-4 py-3">
          <div className="flex items-center gap-4">
            <ShimmerLine className="w-24" />
            <ShimmerLine className="w-52" />
            <ShimmerLine className="ml-auto h-7 w-60 rounded-md" />
          </div>
          <div className="grid grid-cols-[7.5rem_minmax(0,1fr)] gap-y-0">
            {Array.from({ length: 4 }, (_, lane) => (
              <div key={lane} className="contents">
                <div className="flex h-11 items-end pb-2.5">
                  <ShimmerLine className="w-20" />
                </div>
                <div className="relative h-11">
                  <span className="absolute inset-x-0 top-[26px] h-px bg-border-subtle" />
                </div>
              </div>
            ))}
          </div>
          <ShimmerLine className="ml-[7.5rem] h-2 w-[calc(100%-7.5rem)]" />
        </div>
      </div>
      <div
        aria-hidden
        className="grid items-start gap-4 @[900px]/overview:grid-cols-[minmax(0,1fr)_24rem]"
      >
        <div className="overflow-hidden rounded-lg border">
          <PanelHead title="w-32" />
          <div className="flex flex-col gap-1 p-1.5">
            {Array.from({ length: 5 }, (_, row) => (
              <div
                key={row}
                className="grid grid-cols-[1.75rem_minmax(0,1fr)_6rem] gap-3 px-2 py-2.5"
              >
                <ShimmerLine className="size-7 rounded-md" />
                <div className="flex flex-col gap-1.5">
                  <ShimmerLine className="w-3/5" />
                  <ShimmerLine className="w-2/5" />
                </div>
                <ShimmerLine className="h-7 rounded-md" />
              </div>
            ))}
          </div>
        </div>
        <div className="flex min-w-0 flex-col gap-4">
          <div className="overflow-hidden rounded-lg border">
            <PanelHead title="w-20" />
            <div className="flex flex-col gap-2 border-b border-border-subtle p-3">
              <ShimmerLine className="h-5 w-32 rounded-sm" />
              <ShimmerLine className="w-40" />
            </div>
            <div className="flex justify-between px-3 py-2.5">
              <ShimmerLine className="w-16" />
              <ShimmerLine className="w-24" />
            </div>
          </div>
          <div className="overflow-hidden rounded-lg border">
            <PanelHead title="w-24" />
            <div className="flex flex-col divide-y divide-border-subtle px-3">
              {Array.from({ length: 5 }, (_, row) => (
                <div key={row} className="flex justify-between py-2.5">
                  <ShimmerLine className="w-24" />
                  <ShimmerLine className="w-16" />
                </div>
              ))}
            </div>
          </div>
        </div>
      </div>
      <div aria-hidden className="overflow-hidden rounded-lg border">
        <PanelHead title="w-20" />
        <div className="grid grid-cols-1 p-1.5 @[480px]/overview:grid-cols-2 @[760px]/overview:grid-cols-3">
          {Array.from({ length: 12 }, (_, cell) => (
            <div key={cell} className="flex flex-col gap-1.5 px-3 py-3">
              <ShimmerLine className="w-20" />
              <ShimmerLine className="w-32" />
              <ShimmerLine className="w-24" />
            </div>
          ))}
        </div>
      </div>
    </div>
  );
}
