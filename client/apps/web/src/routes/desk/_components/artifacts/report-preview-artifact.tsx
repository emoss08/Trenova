import { useT } from "@trenova/shared/i18n/use-t";
import { ResultGrid } from "@/routes/reports/_components/result-grid";
import type { AssistantArtifact } from "@/types/assistant";
import { useMemo } from "react";
import { reportPreviewFrom } from "./artifact-payloads";

/**
 * A preview as a table: the same grid the report builder uses, so a column
 * sorts and a number lines up the way it would on the Reports page.
 */
export function ReportPreviewArtifact({ artifact }: { artifact: AssistantArtifact }) {
  const t = useT();
  const preview = useMemo(() => reportPreviewFrom(artifact), [artifact]);

  return (
    <div className="flex min-h-0 flex-1 flex-col">
      <ResultGrid
        columns={preview.columns}
        rows={preview.rows}
        totals={preview.totals}
        density="compact"
        emptyMessage={t("The preview returned no rows.")}
        className="min-h-0 flex-1"
      />
      <p className="text-foreground-subtle border-border-subtle bg-sunken flex h-8 shrink-0 items-center gap-2 border-t px-3 text-xs tabular-nums">
        <span className="truncate">
          {t("{0, plural, one {# row} other {# rows}}", preview.rowCount)}
          {preview.dataset !== "" ? ` · ${preview.dataset}` : ""}
        </span>
        {preview.truncated && (
          <span className="shrink-0">· {t("Cut short; the full set needs a run.")}</span>
        )}
      </p>
    </div>
  );
}
