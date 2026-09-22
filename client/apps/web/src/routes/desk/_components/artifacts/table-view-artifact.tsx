import { ResultGrid } from "@/routes/reports/_components/result-grid";
import type { AssistantArtifact } from "@/types/assistant";
import { useT } from "@trenova/shared/i18n/use-t";
import { useMemo } from "react";
import { tableViewFrom } from "./artifact-payloads";

/**
 * What a list or search turn found, as the table it already was.
 *
 * It is the same grid the report builder and a report preview use, so a
 * column sorts and a number lines up the way it does everywhere else — the
 * pane does not get a second table with its own manners.
 *
 * The footer says what was asked, not just how many came back. "12 shipments"
 * on its own invites reading a filtered page as the whole fleet; "12 rows ·
 * status is InTransit" cannot be read that way.
 */
export function TableViewArtifact({ artifact }: { artifact: AssistantArtifact }) {
  const t = useT();
  const view = useMemo(() => tableViewFrom(artifact), [artifact]);

  return (
    <div className="flex min-h-0 flex-1 flex-col">
      <ResultGrid
        columns={view.columns}
        rows={view.rows}
        density="compact"
        emptyMessage={t("Nothing matched.")}
        className="min-h-0 flex-1"
      />
      <p className="text-muted-foreground border-border flex h-8 shrink-0 items-center gap-2 border-t px-3 text-xs">
        <span className="truncate">
          {t("{0, plural, one {# row} other {# rows}}", view.rowCount)}
          {view.searchedFor.length > 0 ? ` · ${view.searchedFor.join(", ")}` : ""}
        </span>
        {view.truncated && (
          <span className="shrink-0">· {t("Showing the first {0}.", view.rows.length)}</span>
        )}
      </p>
    </div>
  );
}
