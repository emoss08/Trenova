import { useT } from "@trenova/shared/i18n/use-t";

/** The page in shimmer before its factor answers: the tab bar, then the overview's layout. */
export function FactoringCompanyPageSkeleton() {
  const t = useT();

  return (
    <div aria-busy aria-label={t("Loading the factoring company")} className="flex flex-col gap-4">
      <div aria-hidden className="flex gap-5 border-b pb-2">
        {Array.from({ length: 7 }, (_, i) => (
          <span key={i} className="block h-4 w-16 ui-shimmer rounded-sm" />
        ))}
      </div>
      <FactoringCompanyOverviewSkeleton />
    </div>
  );
}

/** The overview's own layout in shimmer: Nova's summary, the money path and the two columns. */
export function FactoringCompanyOverviewSkeleton() {
  const t = useT();

  return (
    <div aria-busy aria-label={t("Loading the overview")} className="flex flex-col gap-4">
      <div aria-hidden className="flex flex-col gap-2">
        <span className="block h-3 w-40 ui-shimmer rounded-sm" />
        <span className="block h-5 w-11/12 ui-shimmer rounded-sm" />
        <span className="block h-5 w-3/5 ui-shimmer rounded-sm" />
      </div>
      <div aria-hidden className="grid grid-cols-2 gap-4 rounded-lg border p-3 lg:grid-cols-4">
        {Array.from({ length: 4 }, (_, i) => (
          <div key={i} className="flex flex-col gap-2">
            <span className="block h-3 w-24 ui-shimmer rounded-sm" />
            <span className="block h-6 w-28 ui-shimmer rounded-sm" />
            <span className="block h-3 w-36 ui-shimmer rounded-sm" />
          </div>
        ))}
      </div>
      <div aria-hidden className="grid gap-4 lg:grid-cols-[minmax(0,1fr)_24rem]">
        <span className="block h-64 ui-shimmer rounded-lg" />
        <div className="flex flex-col gap-4">
          <span className="block h-32 ui-shimmer rounded-lg" />
          <span className="block h-40 ui-shimmer rounded-lg" />
        </div>
      </div>
    </div>
  );
}

/** The details cards beside the invoice they print, in shimmer. */
export function FactoringCompanyDetailsSkeleton() {
  const t = useT();

  return (
    <div
      aria-busy
      aria-label={t("Loading the factor's details")}
      className="grid items-start gap-6 xl:grid-cols-[minmax(0,40rem)_minmax(0,1fr)]"
    >
      <div aria-hidden className="flex flex-col gap-4">
        {Array.from({ length: 4 }, (_, i) => (
          <div key={i} className="flex flex-col gap-3 rounded-lg border p-4">
            <span className="block h-4 w-36 ui-shimmer rounded-sm" />
            <span className="block h-3 w-64 ui-shimmer rounded-sm" />
            <div className="grid grid-cols-2 gap-3">
              <span className="block h-8 ui-shimmer rounded-md" />
              <span className="block h-8 ui-shimmer rounded-md" />
            </div>
          </div>
        ))}
      </div>
      <div aria-hidden className="rounded-lg border bg-sunken p-5">
        <span className="mx-auto block aspect-[8.5/11] w-full max-w-176 ui-shimmer rounded-sm" />
      </div>
    </div>
  );
}

/** The agreement's versions beside its terms and calculator, in shimmer. */
export function FactoringAgreementTabSkeleton() {
  const t = useT();

  return (
    <div
      aria-busy
      aria-label={t("Loading the agreements")}
      className="grid items-start gap-4 lg:flex-1 lg:grid-cols-[15rem_minmax(0,1fr)] lg:items-stretch"
    >
      <div aria-hidden className="flex flex-col gap-2 rounded-lg border p-2">
        {Array.from({ length: 3 }, (_, i) => (
          <span key={i} className="block h-12 ui-shimmer rounded-md" />
        ))}
      </div>
      <div aria-hidden className="flex flex-col gap-4">
        <span className="block h-6 w-48 ui-shimmer rounded-sm" />
        <span className="block h-136 ui-shimmer rounded-lg" />
        <span className="block h-36 ui-shimmer rounded-lg" />
        <span className="block h-28 ui-shimmer rounded-lg" />
        <span className="block h-56 ui-shimmer rounded-lg" />
      </div>
    </div>
  );
}

/** A tab made of stacked panels (customers, delivery, the ledger), in shimmer. */
export function FactoringCompanyPanelsSkeleton({ label }: { label: string }) {
  return (
    <div aria-busy aria-label={label} className="flex flex-col gap-4">
      <span aria-hidden className="block h-8 w-80 ui-shimmer rounded-md" />
      <div aria-hidden className="flex flex-col gap-1 rounded-lg border p-2">
        {Array.from({ length: 6 }, (_, i) => (
          <span key={i} className="block h-10 ui-shimmer rounded-md" />
        ))}
      </div>
    </div>
  );
}
