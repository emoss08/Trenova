import { useT } from "@trenova/shared/i18n/use-t";

function CardShimmer({ fields }: { fields: number }) {
  return (
    <div className="flex flex-col rounded-lg border bg-card">
      <div className="flex flex-col gap-1.5 border-b px-4 py-3">
        <span className="block h-3.5 w-28 ui-shimmer rounded-sm" />
        <span className="block h-3 w-48 ui-shimmer rounded-sm" />
      </div>
      <div className="grid grid-cols-2 gap-x-4 gap-y-4 p-4">
        {Array.from({ length: fields }, (_, i) => (
          <div key={i} className="flex flex-col gap-1.5">
            <span className="block h-3 w-20 ui-shimmer rounded-sm" />
            <span className="block h-7 ui-shimmer rounded-md" />
          </div>
        ))}
      </div>
    </div>
  );
}

/** The Profile tab's two columns of cards in shimmer, while its code or the record loads. */
export function ProfileSkeleton() {
  const t = useT();
  return (
    <div
      aria-busy
      aria-label={t("Loading the profile")}
      className="grid items-start gap-4 xl:grid-cols-2"
    >
      <div aria-hidden className="flex flex-col gap-4">
        <span className="block h-3 w-20 ui-shimmer rounded-sm" />
        <CardShimmer fields={8} />
        <CardShimmer fields={6} />
        <CardShimmer fields={4} />
      </div>
      <div aria-hidden className="flex flex-col gap-4">
        <span className="block h-3 w-32 ui-shimmer rounded-sm" />
        <CardShimmer fields={2} />
        <CardShimmer fields={6} />
        <CardShimmer fields={4} />
        <CardShimmer fields={2} />
      </div>
    </div>
  );
}
