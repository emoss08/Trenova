import { DataTable } from "@/components/data-table/data-table";
import {
  DESK_CONVERSATION_LIST_KEY,
  DESK_TRANSCRIPT_EXPORT_LIST_KEY,
  deskConversationTableGraphQLConfig,
  requestDeskTranscriptExport,
  type DeskConversationRow,
} from "@/lib/graphql/desk-transcripts";
import { MAX_SELECTED_CONVERSATIONS } from "@/lib/desk-transcript-notices";
import { downloadDeskConversation } from "@/lib/desk-transcripts";
import { useQueryClient } from "@tanstack/react-query";
import { EmptyTable } from "@trenova/shared/components/ui/empty-table";
import { Button } from "@trenova/shared/components/ui/button";
import { toast } from "@trenova/shared/components/ui/toast";
import { Download01Icon, EyeIcon } from "@trenova/shared/components/icons";
import { useT } from "@trenova/shared/i18n/use-t";
import { graphQLErrorMessage } from "@trenova/shared/lib/graphql";
import type { DockAction, RowAction } from "@trenova/shared/types/data-table";
import { Resource } from "@trenova/shared/types/permission";
import { useMemo, useRef, useState } from "react";
import { getConversationColumns } from "./conversation-columns";
import { selectionLabel } from "./export-coverage";
import {
  DEFAULT_CONVERSATION_SCOPE,
  scopeFilter,
  type ConversationScope,
} from "./conversation-scope";
import { ConversationSheet } from "./conversation-sheet";

const EMPTY_COLUMNS = [
  { label: "Person" },
  { label: "Agent" },
  { label: "Title" },
  { label: "Where it began" },
  { label: "Messages", numeric: true },
  { label: "Last message" },
] as const;

type ConversationTableProps = {
  scope: ConversationScope;
  canExport: boolean;
  onScopeChange: (scope: ConversationScope) => void;
  onExportStarted: () => void;
};

/**
 * Everyone's Desk conversations, newest last message first. A row opens the
 * conversation read-only. A holder of Export picks rows to download: one
 * downloads at once, several are written as one zip in the background.
 */
export default function ConversationTable({
  scope,
  canExport,
  onScopeChange,
  onExportStarted,
}: ConversationTableProps) {
  const t = useT();
  const queryClient = useQueryClient();
  const [opened, setOpened] = useState<DeskConversationRow | null>(null);
  const [downloadingId, setDownloadingId] = useState<string | null>(null);
  const sending = useRef(false);

  const columns = useMemo(() => getConversationColumns(t), [t]);
  const filter = useMemo(() => scopeFilter(scope), [scope]);
  const graphql = useMemo(() => deskConversationTableGraphQLConfig(filter), [filter]);

  const downloadOne = async (row: DeskConversationRow) => {
    setDownloadingId(row.id);
    try {
      await downloadDeskConversation(row.id);
    } catch (error) {
      toast.error(t("The transcript could not be downloaded"), {
        description: graphQLErrorMessage(error, t("Try again shortly.")),
      });
    } finally {
      setDownloadingId(null);
    }
  };

  const downloadSelected = async (rows: DeskConversationRow[]) => {
    if (sending.current || rows.length === 0) {
      return;
    }
    if (rows.length > MAX_SELECTED_CONVERSATIONS) {
      toast.error(t("Too many conversations selected"), {
        description: t(
          "Select at most {0} conversations, or export a timeframe instead.",
          MAX_SELECTED_CONVERSATIONS,
        ),
      });
      return;
    }
    sending.current = true;
    try {
      if (rows.length === 1) {
        await downloadDeskConversation(rows[0].id);
        return;
      }
      await requestDeskTranscriptExport({ conversationIds: rows.map((row) => row.id) });
      await queryClient.invalidateQueries({ queryKey: [DESK_TRANSCRIPT_EXPORT_LIST_KEY] });
      toast.success(t("Export started"), {
        description: t(
          "{0, plural, one {# transcript is} other {# transcripts are}} being written to one file. You will be told when it is ready to download.",
          rows.length,
        ),
        action: { label: t("View exports"), onClick: onExportStarted },
      });
    } finally {
      sending.current = false;
    }
  };

  const dockActions: DockAction<DeskConversationRow>[] = canExport
    ? [
        {
          id: "download-transcripts",
          label: t("Download transcripts"),
          labelFor: (count) => selectionLabel(t, count),
          icon: Download01Icon,
          clearSelectionOnSuccess: true,
          onClick: downloadSelected,
        },
      ]
    : [];

  const contextMenuActions: RowAction<DeskConversationRow>[] = [
    {
      id: "open",
      label: t("Open"),
      icon: EyeIcon,
      onClick: (row) => setOpened(row.original),
    },
    {
      id: "download",
      label: t("Download transcript"),
      icon: Download01Icon,
      hidden: () => !canExport,
      isPending: (row) => downloadingId === row.original.id,
      onClick: (row) => downloadOne(row.original),
    },
  ];

  return (
    <>
      <DataTable<DeskConversationRow>
        name="Desk conversation"
        queryKey={DESK_CONVERSATION_LIST_KEY}
        graphql={graphql}
        resource={Resource.DeskTranscript}
        columns={columns}
        contextMenuActions={contextMenuActions}
        dockActions={dockActions}
        enableRowSelection={canExport}
        enableSelectionExport={false}
        enableExport={false}
        enableCreateAction={false}
        onRowClick={(row) => setOpened(row.original)}
        renderEmptyState={() => (
          <EmptyTable
            className="py-10"
            title={t("No conversation matches")}
            description={t(
              "Nobody had a Desk conversation in this timeframe with these filters. Choose a longer timeframe, or another person, agent or place it began.",
            )}
            columns={EMPTY_COLUMNS}
            action={
              <Button
                type="button"
                variant="outline"
                size="sm"
                onClick={() => onScopeChange(DEFAULT_CONVERSATION_SCOPE)}
              >
                {t("Reset filters")}
              </Button>
            }
          />
        )}
      />
      <ConversationSheet row={opened} canExport={canExport} onClose={() => setOpened(null)} />
    </>
  );
}
