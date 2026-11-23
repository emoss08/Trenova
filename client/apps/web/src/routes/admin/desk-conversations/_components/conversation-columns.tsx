import { HoverCardTimestamp } from "@/components/hover-card-timestamp";
import type { DeskConversationRow } from "@/lib/graphql/desk-transcripts";
import { Badge } from "@trenova/shared/components/ui/badge";
import type { TranslateFn } from "@trenova/shared/i18n/use-t";
import type { ColumnDef } from "@trenova/shared/types/data-table";
import { originLabel } from "./conversation-scope";

/**
 * Who, with which agent, what it is called, where it began and when: what an
 * administrator needs to find a conversation, never what was said in it. The
 * list is filtered and ordered by the controls above it, so no column sorts
 * or filters on its own.
 */
export function getConversationColumns(t: TranslateFn): ColumnDef<DeskConversationRow>[] {
  return [
    {
      accessorKey: "ownerName",
      header: t("Person"),
      cell: ({ row }) => <span className="font-medium">{row.original.ownerName}</span>,
      size: 180,
      enableSorting: false,
      meta: { label: t("Person"), filterable: false, sortable: false },
    },
    {
      accessorKey: "agentName",
      header: t("Agent"),
      cell: ({ row }) => <span>{row.original.agentName}</span>,
      size: 170,
      enableSorting: false,
      meta: { label: t("Agent"), filterable: false, sortable: false },
    },
    {
      accessorKey: "title",
      header: t("Title"),
      cell: ({ row }) => (
        <span className="flex min-w-0 items-center gap-2">
          <span className="truncate" title={row.original.title || undefined}>
            {row.original.title || (
              <span className="text-foreground-subtle">{t("Untitled conversation")}</span>
            )}
          </span>
          {row.original.archived ? <Badge variant="neutral">{t("Archived")}</Badge> : null}
        </span>
      ),
      size: 320,
      enableSorting: false,
      meta: { label: t("Title"), filterable: false, sortable: false },
    },
    {
      accessorKey: "origin",
      header: t("Where it began"),
      cell: ({ row }) => <span>{originLabel(t, row.original.origin)}</span>,
      size: 150,
      enableSorting: false,
      meta: { label: t("Where it began"), filterable: false, sortable: false },
    },
    {
      accessorKey: "messageCount",
      header: t("Messages"),
      cell: ({ row }) => (
        <span className="tabular-nums">{row.original.messageCount.toLocaleString()}</span>
      ),
      size: 100,
      enableSorting: false,
      meta: { label: t("Messages"), filterable: false, sortable: false },
    },
    {
      accessorKey: "startedAt",
      header: t("Started"),
      cell: ({ row }) => <HoverCardTimestamp timestamp={row.original.startedAt} />,
      size: 160,
      enableSorting: false,
      meta: { label: t("Started"), filterable: false, sortable: false },
    },
    {
      accessorKey: "lastMessageAt",
      header: t("Last message"),
      cell: ({ row }) =>
        row.original.lastMessageAt ? (
          <HoverCardTimestamp timestamp={row.original.lastMessageAt} />
        ) : (
          <span className="text-foreground-subtle">—</span>
        ),
      size: 160,
      enableSorting: false,
      meta: { label: t("Last message"), filterable: false, sortable: false },
    },
  ];
}
