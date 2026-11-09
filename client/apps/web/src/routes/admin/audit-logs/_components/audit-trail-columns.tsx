import { HoverCardTimestamp } from "@/components/hover-card-timestamp";
import type { AuditEntryRow } from "@/lib/graphql/audit-log-table";
import {
  ArrowNarrowRightIcon,
  BotIcon,
  ChevronRightIcon,
  ClockIcon,
  Code02Icon,
  Monitor01Icon,
  Moon01Icon,
  type IconComponent,
} from "@trenova/shared/components/icons";
import { Badge } from "@trenova/shared/components/ui/badge";
import { defineLabels } from "@trenova/shared/i18n/labels";
import type { TranslateFn } from "@trenova/shared/i18n/use-t";
import { cn } from "@trenova/shared/lib/utils";
import type { ColumnDef } from "@trenova/shared/types/data-table";
import { Resource } from "@trenova/shared/types/permission";
import {
  actorKindOfChannel,
  actorKindOfPrincipal,
  changeChips,
  entrySentence,
  type AuditActorKind,
} from "./audit-entry-sentence";
import {
  AUDIT_CHANNEL_LABELS,
  auditOperationFilterOptions,
  channelLabel,
  resourceLabel,
  resourceSourceLabel,
} from "./audit-log-formatters";
import { isOffHour } from "./audit-overview-model";
import { ActorMark } from "./audit-top-actors";

const CHIP_LIMIT = 3;

const auditResourceFilterOptions = Object.values(Resource).map((value) => ({
  value,
  label: resourceSourceLabel(value),
}));

const AUDIT_CATEGORY_LABELS = defineLabels({
  User: "User",
  System: "System",
  Security: "Security",
} as const);

/** A channel's short name for the Via column; the inspector names it in full. */
const CHANNEL_SHORT_LABELS = defineLabels({
  web: "Web app",
  api: "API key",
  portal: "Portal",
  agent_conversation: "Conversation",
  agent_run: "Agent run",
  support: "Support",
  scim: "SCIM",
  job: "Job",
  cli: "Command line",
  system: "System",
} as const);

const CHANNEL_ICONS: Record<string, IconComponent> = {
  agent_conversation: BotIcon,
  agent_run: BotIcon,
  api: Code02Icon,
  scim: Code02Icon,
  job: ClockIcon,
  cli: ClockIcon,
  system: ClockIcon,
};

const TAG_LABELS = defineLabels({
  delete: "Delete",
  export: "Export",
  offHours: "Off hours",
} as const);

function optionsOf<T extends Record<string, string>>(labels: T) {
  return Object.entries(labels).map(([value, label]) => ({ value, label }));
}

/** The flags an entry carries in the trail: security, a delete, an export, written off hours. */
/** `offHoursZone` is the organization's time zone, where off hours are counted; null while unknown. */
export function EntryTags({
  entry,
  offHoursZone,
}: {
  entry: AuditEntryRow;
  offHoursZone: string | null;
}) {
  const operation = entry.operation?.toLowerCase();
  return (
    <span className="inline-flex flex-wrap gap-1 align-[1px]">
      {entry.category === "Security" ? (
        <Badge variant="warning" className="h-4.5 px-1.5 text-xs">
          {AUDIT_CATEGORY_LABELS.Security}
        </Badge>
      ) : null}
      {operation === "delete" ? (
        <Badge variant="danger" className="h-4.5 px-1.5 text-xs">
          {TAG_LABELS.delete}
        </Badge>
      ) : null}
      {operation === "export" ? (
        <Badge variant="warning" className="h-4.5 px-1.5 text-xs">
          {TAG_LABELS.export}
        </Badge>
      ) : null}
      {actorKindOfChannel(entry.channel) === "people" &&
      offHoursZone !== null &&
      isOffHour(entry.timestamp, offHoursZone) ? (
        <Badge variant="neutral" className="h-4.5 gap-1 px-1.5 text-xs">
          <Moon01Icon aria-hidden className="size-3" />
          {TAG_LABELS.offHours}
        </Badge>
      ) : null}
    </span>
  );
}

function WhatCell({
  entry,
  offHoursZone,
  expanded,
  onToggleRequest,
  t,
}: {
  entry: AuditEntryRow;
  offHoursZone: string | null;
  expanded: boolean;
  onToggleRequest: (id: string) => void;
  t: TranslateFn;
}) {
  const { chips, more } = changeChips(entry.changes, CHIP_LIMIT);
  const others = Math.max(entry.requestEntryCount - 1, 0);
  return (
    <div className="flex min-w-0 flex-col gap-1.5 py-0.5">
      <p className="leading-5">
        <span className="mr-1.5">{entrySentence(entry)}</span>
        <EntryTags entry={entry} offHoursZone={offHoursZone} />
      </p>
      {chips.length > 0 ? (
        <div className="flex flex-wrap gap-1">
          {chips.map((chip) => (
            <span
              key={chip.path}
              className="inline-flex h-5.5 max-w-full items-center gap-1.5 rounded-sm border bg-muted/60 px-1.5 text-xs"
            >
              <span className="shrink-0 text-foreground-subtle">{chip.field}</span>
              {chip.from !== null ? (
                <span
                  className={cn(
                    "truncate text-muted-foreground",
                    chip.to !== null && "line-through decoration-muted-foreground/45",
                  )}
                >
                  {chip.from}
                </span>
              ) : null}
              {chip.from !== null && chip.to !== null ? (
                <ArrowNarrowRightIcon
                  aria-hidden
                  className="size-3 shrink-0 text-foreground-subtle"
                />
              ) : null}
              {chip.to !== null ? <span className="truncate">{chip.to}</span> : null}
            </span>
          ))}
          {more > 0 ? (
            <span className="inline-flex h-5.5 items-center px-1 text-xs text-muted-foreground">
              {t("{0, plural, one {+# field} other {+# fields}}", more)}
            </span>
          ) : null}
        </div>
      ) : null}
      {others > 0 ? (
        <button
          type="button"
          aria-expanded={expanded}
          onClick={(event) => {
            event.stopPropagation();
            onToggleRequest(entry.id);
          }}
          className="inline-flex h-5.5 items-center gap-1 self-start rounded-sm ui-focus-ring px-1 text-xs text-muted-foreground hover:text-foreground"
        >
          <ChevronRightIcon
            aria-hidden
            className={cn("size-3 transition-transform duration-200", expanded && "rotate-90")}
          />
          {t("{0, plural, one {+# more in this request} other {+# more in this request}}", others)}
        </button>
      ) : null}
    </div>
  );
}

/**
 * What a row says under the actor's name: a person's email, or how an agent, key or job acted;
 * an agent's system user has an address, but it names no one.
 */
export function actorSubtitle(
  t: TranslateFn,
  kind: AuditActorKind,
  channel: string,
  email: string | null | undefined,
): string {
  if (channel === "agent_conversation") return t("through an agent");
  if (kind === "people" && email) return email;
  return (CHANNEL_SHORT_LABELS as Record<string, string>)[channel] ?? channelLabel(channel);
}

export type TrailColumnContext = {
  offHoursZone: string | null;
  expandedId: string | null;
  onToggleRequest: (id: string) => void;
};

export function getTrailColumns(
  t: TranslateFn,
  { offHoursZone, expandedId, onToggleRequest }: TrailColumnContext,
): ColumnDef<AuditEntryRow>[] {
  return [
    {
      accessorKey: "timestamp",
      header: t("Time"),
      cell: ({ row }) => (
        <HoverCardTimestamp
          timestamp={row.original.timestamp}
          showDate={false}
          className="pt-0.5 text-sm text-muted-foreground"
        />
      ),
      size: 120,
      minSize: 100,
      maxSize: 160,
      meta: {
        label: t("Time"),
        apiField: "timestamp",
        filterable: true,
        sortable: true,
        filterType: "date",
        defaultFilterOperator: "daterange",
      },
    },
    {
      id: "actor",
      header: t("Who"),
      cell: ({ row }) => {
        const { user, actorName, principalType, channel } = row.original;
        const name = actorName || user?.name || t("Unknown");
        const kind = actorKindOfPrincipal(principalType);
        return (
          <div className="flex min-w-0 items-start gap-2 py-0.5">
            <ActorMark kind={kind} name={name} user={user} />
            <div className="flex min-w-0 flex-col">
              <span className="truncate font-medium">{name}</span>
              <span className="truncate text-xs text-muted-foreground">
                {actorSubtitle(t, kind, channel, user?.emailAddress)}
              </span>
            </div>
          </div>
        );
      },
      size: 210,
      minSize: 170,
      maxSize: 280,
      meta: {
        label: t("Who"),
        apiField: "principalId",
        filterable: true,
        sortable: false,
        filterType: "record",
        filterRecord: "USER",
      },
    },
    {
      id: "what",
      accessorKey: "comment",
      header: t("What happened"),
      cell: ({ row }) => (
        <WhatCell
          entry={row.original}
          offHoursZone={offHoursZone}
          expanded={expandedId === row.original.id}
          onToggleRequest={onToggleRequest}
          t={t}
        />
      ),
      size: 460,
      minSize: 320,
      maxSize: 720,
      meta: {
        label: t("Description"),
        apiField: "comment",
        filterable: true,
        sortable: false,
        filterType: "text",
        defaultFilterOperator: "contains",
      },
    },
    {
      accessorKey: "entityRef",
      header: t("Record"),
      cell: ({ row }) => (
        <div className="flex min-w-0 flex-col py-0.5">
          <span className="truncate text-xs text-foreground-subtle">
            {resourceLabel(row.original.resource)}
          </span>
          {row.original.entityRef ? (
            <span className="truncate">{row.original.entityRef}</span>
          ) : (
            <span className="truncate font-mono text-xs text-muted-foreground">
              {row.original.resourceId}
            </span>
          )}
        </div>
      ),
      size: 200,
      minSize: 150,
      maxSize: 280,
      meta: {
        label: t("Record"),
        apiField: "entityRef",
        filterable: true,
        sortable: false,
        filterType: "text",
        defaultFilterOperator: "contains",
      },
    },
    {
      accessorKey: "channel",
      header: t("Via"),
      cell: ({ row }) => {
        const Icon = CHANNEL_ICONS[row.original.channel] ?? Monitor01Icon;
        const label =
          (CHANNEL_SHORT_LABELS as Record<string, string>)[row.original.channel] ??
          channelLabel(row.original.channel);
        return (
          <span className="flex min-w-0 items-center gap-1.5 py-0.5 text-muted-foreground">
            <Icon aria-hidden className="size-3.5 shrink-0" />
            <span className="truncate">{label}</span>
          </span>
        );
      },
      size: 140,
      minSize: 110,
      maxSize: 180,
      meta: {
        label: t("Via"),
        apiField: "channel",
        filterable: true,
        sortable: true,
        filterType: "select",
        filterOptions: optionsOf(AUDIT_CHANNEL_LABELS),
        defaultFilterOperator: "eq",
      },
    },
    {
      accessorKey: "resource",
      header: t("Kind of record"),
      cell: ({ row }) => resourceLabel(row.original.resource),
      size: 170,
      meta: {
        label: t("Kind of record"),
        apiField: "resource",
        filterable: true,
        sortable: true,
        filterType: "select",
        filterOptions: auditResourceFilterOptions,
        defaultFilterOperator: "eq",
      },
    },
    {
      accessorKey: "operation",
      header: t("Action"),
      size: 150,
      meta: {
        label: t("Action"),
        apiField: "operation",
        filterable: true,
        sortable: true,
        filterType: "select",
        filterOptions: auditOperationFilterOptions,
        defaultFilterOperator: "eq",
      },
    },
    {
      accessorKey: "category",
      header: t("Category"),
      cell: ({ row }) => AUDIT_CATEGORY_LABELS[row.original.category],
      size: 120,
      meta: {
        label: t("Category"),
        apiField: "category",
        filterable: true,
        sortable: true,
        filterType: "select",
        filterOptions: optionsOf(AUDIT_CATEGORY_LABELS),
        defaultFilterOperator: "eq",
      },
    },
    {
      accessorKey: "resourceId",
      header: t("Record ID"),
      size: 220,
      meta: {
        label: t("Record ID"),
        apiField: "resourceId",
        filterable: true,
        sortable: true,
        filterType: "text",
        defaultFilterOperator: "contains",
      },
    },
    {
      accessorKey: "correlationId",
      header: t("Request ID"),
      cell: ({ row }) => row.original.correlationId || "—",
      size: 200,
      meta: {
        label: t("Request ID"),
        apiField: "correlationId",
        filterable: true,
        sortable: false,
        filterType: "text",
        defaultFilterOperator: "eq",
      },
    },
  ];
}

/** The columns the trail keeps for filtering and the display menu, hidden until asked for. */
export const TRAIL_HIDDEN_COLUMNS = {
  resource: false,
  operation: false,
  category: false,
  resourceId: false,
  correlationId: false,
};
