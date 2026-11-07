import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { useT } from "@trenova/shared/i18n/use-t";

export function AuditAlertPoliciesSkeleton() {
  const t = useT();

  return (
    <div aria-busy aria-label={t("Loading security alerts")} className="flex flex-col gap-2">
      <div aria-hidden className="contents">
        {Array.from({ length: 6 }, (_, index) => (
          <Skeleton key={index} className="h-14 w-full rounded-md" />
        ))}
      </div>
    </div>
  );
}
