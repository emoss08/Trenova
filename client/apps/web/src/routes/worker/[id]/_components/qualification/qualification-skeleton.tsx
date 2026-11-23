import type { WorkerViewOf } from "@/routes/worker/use-worker-state";
import { useT } from "@trenova/shared/i18n/use-t";
import { ShimmerLine, ShimmerPanel } from "../tab-skeleton-parts";

type View = WorkerViewOf<"qualification">;

/**
 * The Qualification tab while its view's code or data loads: the view's own layout in shimmer,
 * so nothing moves when the answer lands. The tab's lazy fallback and each view's pending read
 * draw the same thing.
 */
export function QualificationSkeleton({ view = "credentials" }: { view?: View | null }) {
  return <QualificationSkeletonBody view={view ?? "credentials"} />;
}

export function QualificationSkeletonBody({ view }: { view: View }) {
  const t = useT();
  const label =
    view === "credentials"
      ? t("Loading the credentials")
      : view === "dq-file"
        ? t("Loading the qualification file")
        : t("Loading the drug and alcohol program");
  return (
    <div aria-busy aria-label={label} className="flex flex-col gap-4">
      <div aria-hidden className="contents">
        {view === "credentials" ? <CredentialsShape /> : null}
        {view === "dq-file" ? <DQFileShape /> : null}
        {view === "drug-and-alcohol" ? <DrugAlcoholShape /> : null}
      </div>
    </div>
  );
}

function CredentialsShape() {
  return (
    <>
      <div className="rounded-lg border bg-card">
        <div className="flex min-h-13 items-center gap-4 py-2.5 pr-3 pl-4">
          <ShimmerLine className="h-6 w-16" />
          <ShimmerLine className="w-24" />
          <ShimmerLine className="w-56" />
          <ShimmerLine className="h-5 w-24 rounded-badge" />
          <ShimmerLine className="ml-auto h-7 w-32 rounded-md" />
        </div>
        <div className="border-t border-border-subtle px-4 py-3">
          <span className="block h-16 ui-shimmer rounded-md" />
        </div>
      </div>
      <ShimmerPanel rows={9} />
    </>
  );
}

function DQFileShape() {
  return (
    <>
      <div className="flex items-center gap-2.5">
        <ShimmerLine className="h-4 w-48" />
        <ShimmerLine className="h-5 w-20 rounded-badge" />
        <ShimmerLine className="w-20" />
        <ShimmerLine className="ml-auto h-7 w-40 rounded-md" />
      </div>
      <div className="grid grid-cols-2 overflow-hidden rounded-lg border bg-card lg:grid-cols-4">
        {Array.from({ length: 4 }, (_, index) => (
          <div key={index} className="flex flex-col gap-2 px-3.5 py-3 not-first:border-l">
            <div className="flex justify-between">
              <ShimmerLine className="w-24" />
              <ShimmerLine className="w-10" />
            </div>
            <span className="block h-1.5 ui-shimmer rounded-xs" />
            <ShimmerLine className="w-20" />
          </div>
        ))}
      </div>
      <div className="grid items-start gap-4 lg:grid-cols-[minmax(0,1fr)_24rem]">
        <div className="flex min-w-0 flex-col gap-4">
          <ShimmerPanel rows={6} />
          <ShimmerPanel rows={4} />
          <ShimmerPanel rows={2} />
        </div>
        <div className="flex min-w-0 flex-col gap-4">
          <ShimmerPanel rows={3} />
          <ShimmerPanel rows={2} />
        </div>
      </div>
    </>
  );
}

function DrugAlcoholShape() {
  return (
    <>
      <div className="grid overflow-hidden rounded-lg border bg-card lg:grid-cols-[15rem_minmax(0,1fr)_auto]">
        <div className="flex flex-col gap-2 px-4 py-3">
          <ShimmerLine className="h-5 w-16 rounded-badge" />
          <ShimmerLine className="w-44" />
        </div>
        <div className="grid grid-cols-2 border-t lg:grid-cols-4 lg:border-t-0 lg:border-l">
          {Array.from({ length: 4 }, (_, index) => (
            <div key={index} className="flex flex-col gap-2 px-3.5 py-3">
              <ShimmerLine className="w-24" />
              <ShimmerLine className="w-20" />
            </div>
          ))}
        </div>
        <div className="flex items-center gap-2 border-t p-3 lg:border-t-0 lg:border-l">
          <ShimmerLine className="h-7 w-36 rounded-md" />
          <ShimmerLine className="h-7 w-24 rounded-md" />
        </div>
      </div>
      <div className="grid items-start gap-4 xl:grid-cols-2">
        <ShimmerPanel rows={3} />
        <ShimmerPanel rows={4} />
      </div>
    </>
  );
}
