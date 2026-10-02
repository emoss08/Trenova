import { ArtifactNotice } from "@/components/assistant/voice/artifact-chrome";
import { useCopyToClipboard } from "@/hooks/use-copy-to-clipboard";
import type { AssistantArtifact } from "@/types/assistant";
import { Button } from "@trenova/shared/components/ui/button";
import { Input } from "@trenova/shared/components/ui/input";
import { useT } from "@trenova/shared/i18n/use-t";
import { CheckIcon, CopyIcon, SearchIcon } from "lucide-react";
import { useMemo, useState } from "react";
import { tableViewCsv } from "./artifact-export";
import { filterTableRows, tableViewFrom } from "./artifact-payloads";
import { ArtifactFooter, ArtifactToolbar } from "./artifact-section";
import { ArtifactTable } from "./artifact-table";

/** Rows read at a glance; one more and the table earns a filter. */
export const FILTER_FROM_ROWS = 9;

/**
 * What a list or search turn found, as a table a person reads: the columns
 * that say something about the records, typed and labelled, never the ids and
 * bookkeeping the model worked from. Over it, a filter once there is enough
 * to filter and the way to take the rows out as CSV; under it, what was
 * asked.
 *
 * The footer says what was asked, not just how many came back. "12 shipments"
 * on its own invites reading a filtered page as the whole fleet; "12 rows ·
 * searched for: status is InTransit" cannot be read that way.
 */
export function TableViewArtifact({ artifact }: { artifact: AssistantArtifact }) {
  const t = useT();
  const view = useMemo(() => tableViewFrom(artifact), [artifact]);
  const [query, setQuery] = useState("");
  const shown = useMemo(
    () => filterTableRows(view.rows, view.columns, query, t),
    [view.rows, view.columns, query, t],
  );
  const { copy, isCopied } = useCopyToClipboard();

  if (view.rows.length === 0 || view.columns.length === 0) {
    return (
      <ArtifactNotice kind={artifact.kind}>
        {view.rows.length === 0
          ? t("Nothing matched.")
          : t("Nothing in these results can be shown as a table; the reply has the answer.")}
      </ArtifactNotice>
    );
  }

  const filtering = query.trim() !== "";
  const copyShown = () => copy(tableViewCsv({ ...view, rows: shown }, t), { timeout: 2000 });

  return (
    <div className="flex min-h-0 flex-1 flex-col">
      <ArtifactToolbar>
        {view.rows.length >= FILTER_FROM_ROWS && (
          <Input
            type="search"
            aria-label={t("Filter rows")}
            placeholder={t("Filter rows…")}
            value={query}
            onChange={(event) => setQuery(event.target.value)}
            leftElement={<SearchIcon className="text-foreground-subtle size-3.5" />}
            inputContainerClassName="w-52 max-w-full"
            className="h-6.5 text-xs"
          />
        )}
        <Button
          variant="ghost"
          size="xs"
          className="text-foreground-muted hover:text-foreground ml-auto"
          disabled={shown.length === 0}
          onClick={() => void copyShown()}
        >
          <span
            key={isCopied ? "copied" : "copy"}
            className={isCopied ? "animate-confirm flex" : "flex"}
          >
            {isCopied ? <CheckIcon className="size-3.5" /> : <CopyIcon className="size-3.5" />}
          </span>
          {isCopied ? t("Copied") : t("Copy as CSV")}
        </Button>
      </ArtifactToolbar>

      {shown.length === 0 ? (
        <ArtifactNotice kind={artifact.kind}>{t("No rows match the filter.")}</ArtifactNotice>
      ) : (
        <ArtifactTable columns={view.columns} rows={shown} />
      )}

      <ArtifactFooter>
        <span className="truncate">
          {filtering
            ? t("{0} of {1, plural, one {# row} other {# rows}}", shown.length, view.rowCount)
            : t("{0, plural, one {# row} other {# rows}}", view.rowCount)}
          {view.searchedFor.length > 0
            ? ` · ${t("searched for: {0}", view.searchedFor.join(", "))}`
            : ""}
        </span>
        {view.truncated && (
          <span className="shrink-0">· {t("Showing the first {0}.", view.rows.length)}</span>
        )}
      </ArtifactFooter>
    </div>
  );
}
