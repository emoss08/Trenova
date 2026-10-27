import { useT } from "@trenova/shared/i18n/use-t";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";

const TREE_ROWS = ["w-28", "w-36", "w-32", "w-40", "w-24", "w-32"] as const;
const CODE_ROWS = ["w-64", "w-48", "w-80", "w-56", "w-72", "w-40", "w-60", "w-52"] as const;

/**
 * The studio while its module, project and version load: the toolbar, the file tree, the
 * editor's lines and the panel tabs below, where the loaded studio draws them.
 */
export function LadingStudioSkeleton() {
  const t = useT();
  return (
    <div
      className="flex h-full min-h-0 flex-col gap-3"
      aria-busy
      aria-label={t("Loading the Lading studio")}
    >
      <div className="flex items-center gap-2" aria-hidden>
        <Skeleton className="h-8 w-48" />
        <Skeleton className="h-6 w-20" />
        <div className="ml-auto flex gap-2">
          <Skeleton className="h-8 w-24" />
          <Skeleton className="h-8 w-24" />
        </div>
      </div>
      <div className="grid min-h-0 flex-1 grid-cols-[240px_1fr] gap-3" aria-hidden>
        <div className="bg-card flex flex-col gap-2 rounded-md border p-3">
          {TREE_ROWS.map((width, index) => (
            // oxlint-disable-next-line react/no-array-index-key -- fixed placeholder rows
            <Skeleton key={index} className={`h-4 ${width}`} />
          ))}
        </div>
        <div className="flex min-h-0 flex-col gap-3">
          <div className="bg-card flex flex-1 flex-col gap-2 rounded-md border p-3">
            {CODE_ROWS.map((width, index) => (
              // oxlint-disable-next-line react/no-array-index-key -- fixed placeholder rows
              <Skeleton key={index} className={`h-4 ${width}`} />
            ))}
          </div>
          <div className="bg-card flex h-48 flex-col gap-2 rounded-md border p-3">
            <div className="flex gap-3">
              <Skeleton className="h-5 w-20" />
              <Skeleton className="h-5 w-16" />
              <Skeleton className="h-5 w-12" />
            </div>
            <Skeleton className="h-4 w-72" />
            <Skeleton className="h-4 w-56" />
          </div>
        </div>
      </div>
    </div>
  );
}
