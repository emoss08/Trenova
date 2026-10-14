import { SkeletonRegion } from "@/components/settings-form-skeleton";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { useT } from "@trenova/shared/i18n/use-t";

const ruleSetRows = 5;

/** The rule set list while it loads: each row a name over its description, as `RuleSetCard`. */
export function RuleSetListSkeleton() {
  return (
    <div className="space-y-1 p-2">
      {Array.from({ length: ruleSetRows }, (_, row) => (
        <div key={row} className="rounded-md border border-transparent p-3">
          <div className="flex h-5 items-center">
            <Skeleton className="h-3.5 w-3/4" />
          </div>
          <div className="mt-0.5 flex h-4 items-center">
            <Skeleton className="h-3 w-full" />
          </div>
        </div>
      ))}
    </div>
  );
}

/**
 * The parsing rules workspace: the rule set rail with its search and count, beside the empty
 * detail pane that asks for a rule set to be chosen.
 */
export function DocumentParsingRulesSkeleton() {
  const t = useT();

  return (
    <SkeletonRegion
      label={t("Loading rule sets")}
      className="grid h-full grid-cols-[280px_1fr] overflow-hidden"
    >
      <div className="flex flex-col border-r">
        <div className="space-y-2 border-b p-3">
          <Skeleton className="h-7 w-full" />
          <div className="flex items-center justify-between">
            <Skeleton className="h-3 w-20" />
            <Skeleton className="h-6 w-14" />
          </div>
        </div>
        <RuleSetListSkeleton />
      </div>
      <div className="flex items-center justify-center">
        <div className="flex w-full max-w-155 flex-col items-center gap-1 rounded-lg border border-dashed px-10 py-12">
          <Skeleton className="h-9 w-27" />
          <Skeleton className="mt-4 h-4 w-36" />
          <Skeleton className="mt-1 h-3.5 w-80 max-w-full" />
          <Skeleton className="h-3.5 w-64 max-w-full" />
        </div>
      </div>
    </SkeletonRegion>
  );
}
