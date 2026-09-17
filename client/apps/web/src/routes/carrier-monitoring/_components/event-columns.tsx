import type { TranslateFn } from "@trenova/shared/i18n/use-t";
import { SeverityBadge } from "@/components/carrier-intelligence/severity-badge";
import { EventStatusBadge } from "@/components/carrier-intelligence/event-status-badge";
import type { CarrierIntelLabels } from "@/components/carrier-intelligence/use-carrier-intel-labels";
import { HoverCardTimestamp } from "@/components/hover-card-timestamp";
import { carrierPanelPath } from "@/lib/carrier-links";
import type { CarrierIntelEventRow } from "@/lib/graphql/carrier-monitoring-table";
import type { CarrierIntelEventSource } from "@trenova/graphql/generated/graphql";
import type { ColumnDef } from "@trenova/shared/types/data-table";
import { ArrowRightIcon } from "lucide-react";
import { Link } from "react-router";

const EVENT_SOURCES: readonly CarrierIntelEventSource[] = [
  "NativeChangeFeed",
  "SnapshotDiff",
  "RuleEvaluation",
  "EquipmentVerification",
  "Enrollment",
  "ProviderError",
  "Override",
];

function EventSubjectCell({ row, t }: { row: CarrierIntelEventRow; t: TranslateFn }) {
  const name = row.subjectName || row.dotNumber;
  return (
    <div className="flex min-w-0 flex-col">
      {row.carrierId ? (
        <Link
          to={carrierPanelPath(row.carrierId, "intelligence")}
          className="truncate font-medium hover:underline"
          onClick={(event) => event.stopPropagation()}
        >
          {name}
        </Link>
      ) : (
        <span className="truncate font-medium">{name}</span>
      )}
      <span className="text-muted-foreground text-2xs">{t("USDOT {0}", row.dotNumber)}</span>
    </div>
  );
}

function EventSummaryCell({ row, t }: { row: CarrierIntelEventRow; t: TranslateFn }) {
  const hasChange = row.priorValue !== null || row.currentValue !== null;
  return (
    <div className="flex min-w-0 flex-col gap-0.5">
      <span className="truncate" title={row.summary}>
        {row.summary}
      </span>
      {hasChange ? (
        <span className="text-muted-foreground flex min-w-0 items-center gap-1 text-2xs">
          <span className="truncate line-through">{row.priorValue ?? t("empty")}</span>
          <ArrowRightIcon className="size-3 shrink-0" aria-hidden />
          <span className="text-foreground truncate">{row.currentValue ?? t("empty")}</span>
        </span>
      ) : null}
    </div>
  );
}

export function getEventColumns(
  t: TranslateFn,
  labels: CarrierIntelLabels,
): ColumnDef<CarrierIntelEventRow>[] {
  return [
    {
      accessorKey: "severity",
      header: t("Severity"),
      cell: ({ row }) => <SeverityBadge severity={row.original.severity} />,
      size: 110,
      minSize: 90,
      maxSize: 140,
      meta: {
        apiField: "severity",
        filterable: false,
        sortable: false,
        exportValue: (row: CarrierIntelEventRow) => labels.severity[row.severity],
      },
    },
    {
      accessorKey: "status",
      header: t("Status"),
      cell: ({ row }) => <EventStatusBadge status={row.original.status} />,
      size: 130,
      minSize: 100,
      maxSize: 160,
      meta: {
        apiField: "status",
        filterable: false,
        sortable: true,
        exportValue: (row: CarrierIntelEventRow) => labels.eventStatus[row.status],
      },
    },
    {
      accessorKey: "subjectName",
      header: t("Carrier"),
      cell: ({ row }) => <EventSubjectCell row={row.original} t={t} />,
      size: 220,
      minSize: 160,
      maxSize: 320,
      meta: {
        apiField: "subjectName",
        filterable: true,
        sortable: true,
        filterType: "text",
        defaultFilterOperator: "contains",
      },
    },
    {
      accessorKey: "summary",
      header: t("Change"),
      cell: ({ row }) => <EventSummaryCell row={row.original} t={t} />,
      size: 360,
      minSize: 220,
      maxSize: 560,
      meta: {
        apiField: "summary",
        filterable: true,
        sortable: false,
        filterType: "text",
        defaultFilterOperator: "contains",
      },
    },
    {
      accessorKey: "category",
      header: t("Category"),
      cell: ({ row }) => <span>{labels.section[row.original.category]}</span>,
      size: 140,
      minSize: 110,
      maxSize: 180,
      meta: {
        apiField: "category",
        filterable: false,
        sortable: true,
        exportValue: (row: CarrierIntelEventRow) => labels.section[row.category],
      },
    },
    {
      accessorKey: "source",
      header: t("Source"),
      cell: ({ row }) => (
        <span className="text-muted-foreground">{labels.eventSource[row.original.source]}</span>
      ),
      size: 170,
      minSize: 130,
      maxSize: 220,
      meta: {
        apiField: "source",
        filterable: true,
        sortable: true,
        filterType: "select",
        filterOptions: EVENT_SOURCES.map((source) => ({
          value: source,
          label: labels.eventSource[source],
        })),
        defaultFilterOperator: "eq",
        exportValue: (row: CarrierIntelEventRow) => labels.eventSource[row.source],
      },
    },
    {
      accessorKey: "ruleCode",
      header: t("Rule"),
      cell: ({ row }) =>
        row.original.ruleCode ? (
          <span className="font-mono text-2xs">{row.original.ruleCode}</span>
        ) : (
          <span className="text-muted-foreground">-</span>
        ),
      size: 180,
      minSize: 120,
      maxSize: 260,
      meta: {
        apiField: "ruleCode",
        filterable: true,
        sortable: true,
        filterType: "text",
        defaultFilterOperator: "contains",
      },
    },
    {
      accessorKey: "detectedAt",
      header: t("Detected"),
      cell: ({ row }) => <HoverCardTimestamp timestamp={row.original.detectedAt} />,
      size: 200,
      minSize: 180,
      maxSize: 240,
      meta: {
        apiField: "detectedAt",
        filterable: true,
        sortable: true,
        filterType: "date",
        defaultFilterOperator: "daterange",
      },
    },
  ];
}
