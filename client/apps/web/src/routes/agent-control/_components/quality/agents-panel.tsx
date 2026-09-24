import { SectionTable } from "@/components/data-table/section-table";
import { SectionPanel } from "@/components/section-panel";
import type { AgentQualityRow } from "@/lib/graphql/agent-quality";
import { queries } from "@/lib/queries";
import { Button } from "@trenova/shared/components/ui/button";
import { useT } from "@trenova/shared/i18n/use-t";
import type { ColumnDef } from "@trenova/shared/types/data-table";
import { ChevronRightIcon } from "lucide-react";
import { useMemo } from "react";
import { agentQualityColumns } from "./quality-columns";
import { useQualityPages } from "./use-quality-pages";

const agentRowId = (row: AgentQualityRow) => row.agentDefinitionId;

/**
 * Every agent a page at a time: how people rated it against the window
 * before, its suite scores over the window, and how its last suite run went.
 * Opening a row shows the agent's runs and what each case scored.
 */
export function AgentsPanel({ onOpen }: { onOpen: (agentDefinitionId: string) => void }) {
  const t = useT();
  const { query, rows, pagination } = useQualityPages("agents", (page) =>
    queries.agentQuality.agents(page),
  );

  const columns = useMemo<ColumnDef<AgentQualityRow>[]>(
    () => [
      ...agentQualityColumns(t),
      {
        id: "open",
        header: () => <span className="sr-only">{t("Open")}</span>,
        cell: ({ row }) => (
          <Button
            variant="ghost"
            size="icon-xs"
            aria-label={t("Open {0}", row.original.name)}
            onClick={() => onOpen(row.original.agentDefinitionId)}
          >
            <ChevronRightIcon className="size-3.5" />
          </Button>
        ),
        meta: { headerClassName: "w-10", cellClassName: "w-10" },
      },
    ],
    [onOpen, t],
  );

  return (
    <SectionPanel
      title={t("Agents")}
      count={pagination.totalCount ?? undefined}
      help={t(
        "Satisfaction is the share of rated answers that were thumbs up, against the window before. The quality line is each scored suite run's score; the last run says whether the nightly sweep ran the agent, skipped it because nothing changed, or stopped at the budget.",
      )}
    >
      <SectionTable
        label={t("Agents")}
        columns={columns}
        rows={rows}
        getRowId={agentRowId}
        isLoading={query.isPending}
        isRefreshing={query.isPlaceholderData}
        error={query.isError ? t("Agents could not be loaded.") : null}
        onRetry={() => void query.refetch()}
        empty={t("No agents yet.")}
        pagination={pagination}
      />
    </SectionPanel>
  );
}
