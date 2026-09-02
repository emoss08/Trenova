import { ExpressionDiff } from "@/components/formula-editor/expression-diff";
import { queries } from "@/lib/queries";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { useQuery } from "@tanstack/react-query";
import { GitCompareIcon } from "lucide-react";
import { useMemo } from "react";
import { describeChangedFields } from "./review-diff";

/**
 * What this review would put into production that is not there today: the
 * expression as a word diff against the last approved snapshot, and every
 * other changed field as one line. Reviewers approve a change, not a template.
 */
export function ReviewDiffPanel({ templateId }: { templateId: string }) {
  const { data, isLoading, isError } = useQuery({
    ...queries.formulaTemplate.reviewDiff(templateId),
    enabled: !!templateId,
    staleTime: 0,
  });

  const rows = useMemo(() => (data ? describeChangedFields(data.changes) : []), [data]);

  if (isLoading) {
    return (
      <div className="space-y-1.5 rounded-md border p-3">
        <Skeleton className="h-3.5 w-44" />
        <Skeleton className="h-16 w-full" />
      </div>
    );
  }

  if (isError || !data) {
    return (
      <div className="text-muted-foreground rounded-md border px-3 py-2 text-xs">
        The change summary could not be loaded; open Version History to compare by hand.
      </div>
    );
  }

  const expressionChanged = data.baseExpression !== data.currentExpression;

  return (
    <div className="overflow-hidden rounded-md border">
      <div className="flex items-center justify-between gap-2 border-b px-3 py-2">
        <span className="flex items-center gap-1.5 text-xs font-semibold">
          <GitCompareIcon className="size-3.5" />
          {data.hasApprovedBase
            ? `Changes since approved v${data.baseVersion}`
            : "First approval; everything below is new"}
        </span>
        <span className="text-2xs text-muted-foreground">
          {data.changeCount === 0
            ? "identical to production"
            : `${data.changeCount} ${data.changeCount === 1 ? "change" : "changes"}`}
        </span>
      </div>

      {data.changeCount === 0 ? (
        <p className="text-muted-foreground px-3 py-2 text-xs">
          The content matches what is already approved; approving records a fresh review without
          changing any rate.
        </p>
      ) : (
        <div className="space-y-2 p-3">
          {expressionChanged && (
            <div className="space-y-1">
              <span className="text-2xs text-muted-foreground font-medium tracking-wide uppercase">
                Expression
              </span>
              <ExpressionDiff before={data.baseExpression} after={data.currentExpression} />
            </div>
          )}
          {rows.length > 0 && (
            <ul className="divide-y rounded-md border text-xs">
              {rows.map((row) => (
                <li key={row.path} className="flex items-center justify-between gap-3 px-3 py-1.5">
                  <span className="font-medium">{row.label}</span>
                  <span className="text-muted-foreground truncate font-mono">{row.summary}</span>
                </li>
              ))}
            </ul>
          )}
        </div>
      )}
    </div>
  );
}
