import { GitCompareIcon } from "@trenova/shared/components/icons";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { useT } from "@trenova/shared/i18n/use-t";
import type { ReviewDiffResponse } from "@trenova/shared/types/formula-template";
import { useMemo } from "react";
import { describeChangedFields } from "./review-diff";
import { SourceDiff } from "./source-diff";

/**
 * What a review would put into production that is not there today: the source as a line diff
 * against the last approved version, the pinned libraries that changed, and every other changed
 * field as one line. Reviewers approve a change, not a file.
 */
export function ReviewDiffView({
  data,
  isLoading,
  isError,
}: {
  data: ReviewDiffResponse | undefined;
  isLoading: boolean;
  isError: boolean;
}) {
  const t = useT();
  const rows = useMemo(() => (data ? describeChangedFields(data.changes) : []), [data]);

  if (isLoading) {
    return (
      <div
        className="space-y-1.5 rounded-md border p-3"
        aria-busy
        aria-label={t("Loading the changes")}
      >
        <Skeleton className="h-3.5 w-44" />
        <Skeleton className="h-16 w-full" />
      </div>
    );
  }

  if (isError || !data) {
    return (
      <div className="text-muted-foreground rounded-md border px-3 py-2 text-xs">
        {t("The change summary could not be loaded; open Version History to compare by hand.")}
      </div>
    );
  }

  const sourceChanged = data.baseSource !== data.currentSource;

  return (
    <div className="overflow-hidden rounded-md border">
      <div className="flex items-center justify-between gap-2 border-b px-3 py-2">
        <span className="flex items-center gap-1.5 text-xs font-semibold">
          <GitCompareIcon className="size-3.5" />
          {data.hasApprovedBase
            ? t("Changes since approved v{0}", data.baseVersion)
            : t("First approval; everything below is new")}
        </span>
        <span className="text-2xs text-muted-foreground">
          {data.changeCount === 0
            ? t("identical to production")
            : t("{0, plural, one {# change} other {# changes}}", data.changeCount)}
        </span>
      </div>

      {data.changeCount === 0 ? (
        <p className="text-muted-foreground px-3 py-2 text-xs">
          {t(
            "The content matches what is already approved; approving records a fresh review without changing any rate.",
          )}
        </p>
      ) : (
        <div className="space-y-2 p-3">
          {sourceChanged && (
            <div className="space-y-1">
              <span className="text-muted-foreground text-xs font-medium">{t("Source")}</span>
              <SourceDiff before={data.baseSource} after={data.currentSource} />
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
