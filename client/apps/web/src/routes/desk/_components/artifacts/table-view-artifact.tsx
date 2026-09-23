import { ArtifactNotice } from "@/components/assistant/voice/artifact-chrome";
import type { AssistantArtifact } from "@/types/assistant";
import { useT } from "@trenova/shared/i18n/use-t";
import { useMemo } from "react";
import { ArtifactTable } from "./artifact-table";
import { tableViewFrom } from "./artifact-payloads";

/**
 * What a list or search turn found, as a table a person reads: the columns
 * that say something about the records, typed and labelled, never the ids and
 * bookkeeping the model worked from.
 *
 * The footer says what was asked, not just how many came back. "12 shipments"
 * on its own invites reading a filtered page as the whole fleet; "12 rows ·
 * status is InTransit" cannot be read that way.
 */
export function TableViewArtifact({ artifact }: { artifact: AssistantArtifact }) {
  const t = useT();
  const view = useMemo(() => tableViewFrom(artifact), [artifact]);

  if (view.rows.length === 0 || view.columns.length === 0) {
    return (
      <ArtifactNotice kind={artifact.kind}>
        {view.rows.length === 0
          ? t("Nothing matched.")
          : t("Nothing in these results can be shown as a table; the reply has the answer.")}
      </ArtifactNotice>
    );
  }

  return (
    <div className="flex min-h-0 flex-1 flex-col">
      <ArtifactTable columns={view.columns} rows={view.rows} />
      <p className="text-foreground-subtle border-border-subtle bg-sunken flex h-8 shrink-0 items-center gap-2 border-t px-3 text-xs tabular-nums">
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
