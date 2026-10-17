import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { useT } from "@trenova/shared/i18n/use-t";

/** The portal sites section while it loads: its heading and three site rows. */
export function PortalSitesSkeleton() {
  const t = useT();

  return (
    <div aria-busy aria-label={t("Loading portal sites")} className="flex flex-col gap-3">
      <div aria-hidden className="contents">
        <div className="flex items-center justify-between gap-2">
          <Skeleton className="h-4 w-28" />
          <Skeleton className="h-8 w-36" />
        </div>
        <div className="flex flex-col divide-y rounded-md border">
          {[0, 1, 2].map((row) => (
            <div key={row} className="flex flex-col gap-1.5 px-3 py-2.5">
              <Skeleton className="h-4 w-48" />
              <Skeleton className="h-3 w-64" />
            </div>
          ))}
        </div>
      </div>
    </div>
  );
}
