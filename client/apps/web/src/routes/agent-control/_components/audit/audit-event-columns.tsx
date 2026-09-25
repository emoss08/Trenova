import { DataTableLink } from "@/components/data-table/_components/data-table-components";
import { HoverCardTimestamp } from "@/components/hover-card-timestamp";
import { formatUsd } from "@/lib/ai-usage-format";
import type { AIAuditEventRow } from "@/lib/graphql/ai-audit";
import { Badge } from "@trenova/shared/components/ui/badge";
import type { TranslateFn } from "@trenova/shared/i18n/use-t";
import type { ColumnDef } from "@trenova/shared/types/data-table";
import { traceColumn } from "../trace-link";
import { AuditOutcomeBadge, AuditTierCell } from "./audit-badges";
import {
  AUDIT_EVENT_KINDS,
  AUDIT_EVENT_OUTCOMES,
  auditEventRecordPath,
  auditKindLabel,
  auditOutcomeAttrs,
  auditTierLabel,
} from "./audit-model";

const TIERS = ["Propose", "ActWithApproval", "AutoExecute"] as const;

function PersonCell({ row, t }: { row: AIAuditEventRow; t: TranslateFn }) {
  const forName = row.onBehalfOfUserName;
  const decidedName = row.decidedByUserName;
  if (!forName && !decidedName) {
    return (
      <span className="text-foreground-subtle">
        {row.principalType === "System" ? t("System") : "—"}
      </span>
    );
  }

  return (
    <span className="flex min-w-0 flex-col leading-tight">
      {forName ? <span className="truncate">{t("For {0}", forName)}</span> : null}
      {decidedName ? (
        <span className="text-foreground-muted truncate text-xs">
          {t("Decided by {0}", decidedName)}
        </span>
      ) : null}
    </span>
  );
}

function RecordCell({ row }: { row: AIAuditEventRow }) {
  if (!row.entityType || !row.entityId) {
    return <span className="text-foreground-subtle">—</span>;
  }

  const href = auditEventRecordPath(row.entityType, row.entityId);
  return (
    <span className="flex min-w-0 flex-col leading-tight">
      <span className="truncate">{row.entityType}</span>
      {href === null ? (
        <span className="text-foreground-muted truncate font-mono text-xs">{row.entityId}</span>
      ) : (
        <DataTableLink text={row.entityId} href={href} className="truncate font-mono text-xs" />
      )}
    </span>
  );
}

/**
 * The trail's columns. When, the agent and whose work it was are narrowed
 * from beside the table; what happened, how it turned out, the record, the
 * tier, the model and the trace are the table's own filters.
 */
export function getAuditEventColumns(t: TranslateFn): ColumnDef<AIAuditEventRow>[] {
  const outcomes = auditOutcomeAttrs(t);

  return [
    {
      accessorKey: "occurredAt",
      header: t("When"),
      cell: ({ row }) => <HoverCardTimestamp timestamp={row.original.occurredAt} />,
      size: 160,
      meta: {
        label: t("When"),
        apiField: "occurredAt",
        filterable: false,
        sortable: true,
      },
    },
    {
      accessorKey: "agentName",
      header: t("Agent"),
      cell: ({ row }) =>
        row.original.agentName ? (
          <span className="flex min-w-0 items-baseline gap-1.5">
            <span className="truncate">{row.original.agentName}</span>
            {row.original.agentDefinitionVersion != null ? (
              <span className="text-foreground-muted shrink-0 text-xs tabular-nums">
                {t("v{0}", row.original.agentDefinitionVersion)}
              </span>
            ) : null}
          </span>
        ) : (
          <span className="text-foreground-subtle">—</span>
        ),
      size: 180,
      meta: {
        label: t("Agent"),
        apiField: "agentName",
        filterable: true,
        sortable: true,
        filterType: "text",
      },
    },
    {
      accessorKey: "kind",
      header: t("What"),
      cell: ({ row }) => (
        <span className="flex min-w-0 flex-col leading-tight">
          <span className="truncate">{auditKindLabel(t, row.original.kind)}</span>
          {row.original.toolName ? (
            <span className="text-foreground-muted truncate font-mono text-xs">
              {row.original.toolName}
            </span>
          ) : null}
        </span>
      ),
      size: 200,
      meta: {
        label: t("What"),
        apiField: "kind",
        filterable: true,
        sortable: true,
        filterType: "select",
        filterOptions: AUDIT_EVENT_KINDS.map((kind) => ({
          value: kind,
          label: auditKindLabel(t, kind),
        })),
        defaultFilterOperator: "eq",
      },
    },
    {
      accessorKey: "toolName",
      header: t("Tool"),
      cell: ({ row }) => (
        <span className="text-foreground-muted font-mono text-xs">
          {row.original.toolName ?? "—"}
        </span>
      ),
      size: 180,
      meta: {
        label: t("Tool"),
        apiField: "toolName",
        filterable: true,
        sortable: true,
        filterType: "text",
      },
    },
    {
      accessorKey: "outcome",
      header: t("Outcome"),
      cell: ({ row }) => <AuditOutcomeBadge outcome={row.original.outcome} />,
      size: 150,
      meta: {
        label: t("Outcome"),
        apiField: "outcome",
        filterable: true,
        sortable: true,
        filterType: "select",
        filterOptions: AUDIT_EVENT_OUTCOMES.map((outcome) => ({
          value: outcome,
          label: outcomes[outcome].text,
        })),
        defaultFilterOperator: "eq",
      },
    },
    {
      id: "person",
      accessorKey: "onBehalfOfUserName",
      header: t("Person"),
      cell: ({ row }) => <PersonCell row={row.original} t={t} />,
      size: 190,
      meta: {
        label: t("Person"),
        apiField: "onBehalfOfUserName",
        filterable: false,
        sortable: true,
      },
    },
    {
      accessorKey: "entityId",
      header: t("Record"),
      cell: ({ row }) => <RecordCell row={row.original} />,
      size: 200,
      meta: {
        label: t("Record"),
        apiField: "entityId",
        filterable: true,
        sortable: false,
        filterType: "text",
        defaultFilterOperator: "eq",
      },
    },
    {
      accessorKey: "entityType",
      header: t("Record type"),
      cell: ({ row }) => <span>{row.original.entityType ?? "—"}</span>,
      size: 150,
      meta: {
        label: t("Record type"),
        apiField: "entityType",
        filterable: true,
        sortable: true,
        filterType: "text",
      },
    },
    {
      accessorKey: "tier",
      header: t("Tier"),
      cell: ({ row }) => <AuditTierCell tier={row.original.tier} heldBy={row.original.heldBy} />,
      size: 190,
      meta: {
        label: t("Tier"),
        apiField: "tier",
        filterable: true,
        sortable: true,
        filterType: "select",
        filterOptions: TIERS.map((tier) => ({ value: tier, label: auditTierLabel(t, tier) })),
        defaultFilterOperator: "eq",
      },
    },
    {
      accessorKey: "model",
      header: t("Model"),
      cell: ({ row }) => (
        <span className="text-foreground-muted font-mono text-xs">{row.original.model ?? "—"}</span>
      ),
      size: 170,
      meta: {
        label: t("Model"),
        apiField: "model",
        filterable: true,
        sortable: true,
        filterType: "text",
      },
    },
    {
      accessorKey: "costUsd",
      header: t("Cost"),
      cell: ({ row }) => (
        <span className="tabular-nums">{formatUsd(row.original.costUsd) ?? "—"}</span>
      ),
      size: 100,
      meta: {
        label: t("Cost"),
        apiField: "costUsd",
        filterable: true,
        sortable: true,
        filterType: "number",
      },
    },
    {
      accessorKey: "tainted",
      header: t("Tainted"),
      cell: ({ row }) =>
        row.original.tainted ? (
          <Badge variant="warning">{t("Read outside content")}</Badge>
        ) : (
          <span className="text-foreground-subtle">—</span>
        ),
      size: 170,
      meta: {
        label: t("Tainted"),
        apiField: "tainted",
        filterable: true,
        sortable: true,
        filterType: "boolean",
      },
    },
    traceColumn<AIAuditEventRow>(t),
    {
      accessorKey: "seq",
      header: t("Seq"),
      cell: ({ row }) => <span className="tabular-nums">#{row.original.seq}</span>,
      size: 90,
      meta: {
        label: t("Seq"),
        apiField: "seq",
        filterable: true,
        sortable: true,
        filterType: "number",
      },
    },
  ];
}
