import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { useT } from "@trenova/shared/i18n/use-t";

export function AuditChainStatusSkeleton() {
  const t = useT();

  return (
    <div aria-busy aria-label={t("Loading the audit log's seal status")} className="w-full">
      <div aria-hidden className="contents">
        <Skeleton className="h-10 w-full rounded-md" />
      </div>
    </div>
  );
}
