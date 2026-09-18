import { useT } from "@trenova/shared/i18n/use-t";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { TriangleAlert } from "lucide-react";

export function RequestedPTOOverviewSkeleton() {
  return (
    <div className="border-border flex size-full flex-col gap-1 overflow-y-hidden rounded-md border p-3">
      {Array.from({ length: 10 }).map((_, index) => (
        <div
          key={index}
          className="border-border flex items-center justify-center rounded-md border"
        >
          <Skeleton className="h-[70px] w-full" />
        </div>
      ))}
    </div>
  );
}
export function RequestedPTOEmptyState() {
  const t = useT();

  return (
    <div className="border-border flex size-full flex-col items-center justify-center overflow-hidden rounded-md border">
      <div className="relative size-full">
        <div className="pointer-events-none absolute inset-0 flex flex-col items-center justify-center gap-1">
          <p className="font-table bg-warning px-1 py-0.5 text-center text-sm/none font-medium text-warning-foreground uppercase select-none dark:bg-warning dark:text-foreground">
            {t("No data available")}
          </p>
          <p className="font-table bg-invert px-1 py-0.5 text-center text-sm/none font-medium uppercase select-none text-invert-foreground">
            {t("Try adjusting your filters or search query")}
          </p>
        </div>
      </div>
    </div>
  );
}

export function RequestedPTOErrorState() {
  const t = useT();

  return (
    <div className="border-border flex size-full flex-col items-center justify-center gap-1 overflow-hidden rounded-md border p-3">
      <TriangleAlert className="mt-0.5 size-5 text-danger-foreground" />
      <div className="flex flex-col items-center text-center">
        <p className="font-medium text-danger-foreground">{t("Error loading PTO requests")}</p>
        <p className="text-muted-foreground mt-1 text-xs">
          {t("Looks like we hit a snag. Please try again later.")}
        </p>
      </div>
    </div>
  );
}
