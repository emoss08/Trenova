import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { useT } from "@trenova/shared/i18n/use-t";
import { cn } from "@trenova/shared/lib/utils";

const ROWS = [
  { pro: "w-20", payer: "w-24", amount: "w-14", note: "w-40" },
  { pro: "w-24", payer: "w-20", amount: "w-16", note: "w-32" },
  { pro: "w-20", payer: "w-28", amount: "w-12", note: "w-44" },
] as const;

/** A lane's rows while they load: the PRO, payer and amount line over the note line. */
export function BillingQueueLaneSkeleton({ rows = ROWS.length }: { rows?: number }) {
  const t = useT();

  return (
    <div aria-busy aria-label={t("Loading the lane")}>
      <div className="contents" aria-hidden>
        {ROWS.slice(0, rows).map((row, index) => (
          <div key={index} className="flex flex-col gap-1.5 px-3 py-2.5 pl-9">
            <div className="flex items-center gap-2">
              <Skeleton className={cn("h-3.5", row.pro)} />
              <Skeleton className={cn("h-3", row.payer)} />
              <Skeleton className={cn("ml-auto h-3.5", row.amount)} />
            </div>
            <Skeleton className={cn("h-3", row.note)} />
          </div>
        ))}
      </div>
    </div>
  );
}

/** The whole list while it loads: three lane headings, each over its rows. */
export function BillingQueueListSkeleton() {
  const t = useT();

  return (
    <div aria-busy aria-label={t("Loading the billing queue")} className="flex flex-col">
      <div className="contents" aria-hidden>
        {[3, 2, 2].map((rows, index) => (
          <div key={index} className="border-b">
            <div className="flex items-center gap-2 px-3 py-2">
              <Skeleton className="size-1.5 rounded-full" />
              <Skeleton className="h-3 w-24" />
              <Skeleton className="ml-auto h-3 w-16" />
            </div>
            <BillingQueueLaneSkeleton rows={rows} />
          </div>
        ))}
      </div>
    </div>
  );
}
