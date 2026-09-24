import { DataTable } from "@/components/data-table/data-table";
import { DataTablePanelContainer } from "@/components/data-table/data-table-panel";
import {
  AI_RETRIEVAL_FAILED_LIST_KEY,
  createAIRetrievalFailedEntryTableGraphQLConfig,
  type AIRetrievalFailedEntryRow,
} from "@/lib/graphql/ai-retrieval";
import { Badge } from "@trenova/shared/components/ui/badge";
import { Button } from "@trenova/shared/components/ui/button";
import {
  DescriptionEmpty,
  DescriptionItem,
  DescriptionList,
} from "@trenova/shared/components/ui/description-list";
import { useT } from "@trenova/shared/i18n/use-t";
import { formatUnixInUserTimezone } from "@trenova/shared/lib/date";
import { phaseTone } from "@trenova/shared/lib/status-phase";
import type { DataTablePanelProps } from "@trenova/shared/types/data-table";
import { Resource } from "@trenova/shared/types/permission";
import { XIcon } from "lucide-react";
import { useQueryState } from "nuqs";
import { useMemo } from "react";
import { RETRIEVAL_SOURCE_PARAM, retrievalSourceParser } from "../../ai-control-tabs";
import { useAIControlNavigation } from "../../use-ai-control-navigation";
import { getFailedEntryColumns } from "./retrieval-columns";
import { SOURCE_LABEL, failedEntryStatus } from "./retrieval-model";

const ATTEMPT_FORMAT = {
  year: "numeric",
  month: "short",
  day: "numeric",
  hour: "numeric",
  minute: "2-digit",
} as const;

function FailedEntryPanel({
  open,
  onOpenChange,
  row,
}: DataTablePanelProps<AIRetrievalFailedEntryRow>) {
  const t = useT();
  const status = row ? failedEntryStatus(row) : null;

  return (
    <DataTablePanelContainer
      open={open}
      onOpenChange={onOpenChange}
      title={row ? t(SOURCE_LABEL[row.sourceType].label) : t("Failed item")}
      description={row?.sourceId}
      size="lg"
    >
      {row && status ? (
        <DescriptionList layout="split">
          <DescriptionItem label={t("Status")}>
            <Badge variant={phaseTone(status.phase)}>{t(status.text)}</Badge>
          </DescriptionItem>
          <DescriptionItem label={t("Item")}>
            <span className="font-mono text-xs break-all select-all">{row.sourceId}</span>
          </DescriptionItem>
          <DescriptionItem label={t("Model")}>
            <span className="font-mono text-xs break-all">{row.modelKey}</span>
          </DescriptionItem>
          <DescriptionItem label={t("Attempts")} numeric>
            {row.attempts}
          </DescriptionItem>
          <DescriptionItem label={t("Last attempt")}>
            {row.lastAttemptAt ? (
              formatUnixInUserTimezone(row.lastAttemptAt, ATTEMPT_FORMAT)
            ) : (
              <DescriptionEmpty />
            )}
          </DescriptionItem>
          <DescriptionItem label={t("Next attempt")}>
            {row.nextAttemptAt ? (
              formatUnixInUserTimezone(row.nextAttemptAt, ATTEMPT_FORMAT)
            ) : (
              <DescriptionEmpty />
            )}
          </DescriptionItem>
          <DescriptionItem label={t("Error")}>
            <span className="whitespace-pre-wrap">{row.error}</span>
          </DescriptionItem>
        </DescriptionList>
      ) : null}
    </DataTablePanelContainer>
  );
}

/**
 * The items the indexer could not embed, for every source or the one a
 * source's row narrowed it to. An item waiting on a retry is listed too,
 * because its error is the one the next attempt is likely to hit.
 */
export default function FailedEntriesTable() {
  const t = useT();
  const navigate = useAIControlNavigation();
  const [sourceType] = useQueryState(RETRIEVAL_SOURCE_PARAM, retrievalSourceParser);
  const columns = useMemo(() => getFailedEntryColumns(t), [t]);
  const graphql = useMemo(
    () => createAIRetrievalFailedEntryTableGraphQLConfig(sourceType),
    [sourceType],
  );

  return (
    <div className="flex min-w-0 flex-col gap-2">
      {sourceType ? (
        <div className="flex items-center justify-between gap-2">
          <p className="text-muted-foreground min-w-0 truncate text-sm">
            {t("Showing failures in {0} only", t(SOURCE_LABEL[sourceType].label))}
          </p>
          <Button variant="ghost" size="xs" onClick={() => navigate({ tab: "retrieval" })}>
            <XIcon className="size-3.5" />
            {t("Show every source")}
          </Button>
        </div>
      ) : null}
      <DataTable<AIRetrievalFailedEntryRow>
        name="Failed Item"
        queryKey={AI_RETRIEVAL_FAILED_LIST_KEY}
        graphql={graphql}
        resource={Resource.AIProvider}
        columns={columns}
        TablePanel={FailedEntryPanel}
        enableCreateAction={false}
        enableReadOnlyPanel
        initialColumnVisibility={{ nextAttemptAt: false, modelKey: false }}
      />
    </div>
  );
}
