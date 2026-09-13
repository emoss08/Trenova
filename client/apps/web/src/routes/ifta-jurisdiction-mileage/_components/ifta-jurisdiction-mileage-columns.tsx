import type { TranslateFn } from "@trenova/shared/i18n/use-t";
import { EntityRefCell } from "@/components/data-table/_components/entity-ref-link";
import { jurisdictionLabel } from "@/lib/ifta-jurisdiction-options";
import { HoverCardTimestamp } from "@/components/hover-card-timestamp";
import { iftaMileageSourceChoices } from "@/lib/choices";
import type { TractorRow } from "@/lib/graphql/equipment-table";
import type { IftaMileageEntryRow } from "@/lib/graphql/ifta-jurisdiction-mileage";
import { BooleanBadge } from "@trenova/shared/components/status-badge";
import { Badge, type BadgeVariant } from "@trenova/shared/components/ui/badge";
import { formatUnixDate } from "@trenova/shared/lib/date";
import type { ColumnDef } from "@trenova/shared/types/data-table";
import { formatDecimalString } from "@trenova/shared/types/decimal";
import type { GenericSelectOption } from "@trenova/shared/types/fields";
import {
  IFTA_MILEAGE_SOURCE_LABELS,
  type IftaMileageSource,
} from "@trenova/shared/types/fuel-ifta-enums";
import { IFTA_MILES_SCALE } from "@trenova/shared/types/ifta-jurisdiction-mileage";

const LOADED_OPTIONS = [
  { value: true, label: "Loaded", color: "#15803d" },
  { value: false, label: "Empty", color: "#6b7280" },
] satisfies ReadonlyArray<GenericSelectOption<boolean>>;

const SOURCE_VARIANTS: Record<IftaMileageSource, BadgeVariant> = {
  Manual: "purple",
  RouteCalculation: "info",
  Telematics: "secondary",
};

export function getColumns(
  jurisdictionOptions: readonly GenericSelectOption<string>[],
  t: TranslateFn,
): ColumnDef<IftaMileageEntryRow>[] {
  return [
    {
      accessorKey: "traveledAt",
      header: t("Travelled"),
      cell: ({ row }) => (
        <span className="font-table tabular-nums">{formatUnixDate(row.original.traveledAt)}</span>
      ),
      size: 130,
      meta: {
        apiField: "traveledAt",
        filterable: true,
        sortable: true,
        filterType: "date",
        defaultFilterOperator: "daterange",
      },
    },
    {
      id: "tractor",
      header: t("Tractor"),
      cell: ({ row }) => {
        const { tractor } = row.original;

        if (!tractor) {
          return <p className="text-muted-foreground">-</p>;
        }

        return (
          <EntityRefCell<NonNullable<IftaMileageEntryRow["tractor"]>, TractorRow>
            entity={tractor}
            parent={tractor as TractorRow}
            config={{
              basePath: "/equipment/tractors",
              getId: (t) => t.id,
              getDisplayText: (t) => t.code,
            }}
          />
        );
      },
      enableSorting: false,
      size: 110,
      meta: {
        apiField: "tractorId",
        filterable: false,
        sortable: false,
        exportValue: (row: IftaMileageEntryRow) => row.tractor?.code ?? "",
      },
    },
    {
      accessorKey: "jurisdictionId",
      header: t("Jurisdiction"),
      cell: ({ row }) => (
        <span className="font-table">{jurisdictionLabel(row.original.jurisdiction)}</span>
      ),
      enableSorting: false,
      size: 160,
      meta: {
        apiField: "jurisdictionId",
        filterable: true,
        sortable: false,
        filterType: "select",
        filterOptions: [...jurisdictionOptions],
        defaultFilterOperator: "eq",
        exportValue: (row: IftaMileageEntryRow) => jurisdictionLabel(row.jurisdiction),
      },
    },
    {
      accessorKey: "miles",
      header: t("Miles"),
      cell: ({ row }) => (
        <span className="font-table block text-left tabular-nums">
          {formatDecimalString(row.original.miles, IFTA_MILES_SCALE)}
        </span>
      ),
      size: 100,
      meta: {
        apiField: "miles",
        filterable: true,
        sortable: true,
        filterType: "number",
        defaultFilterOperator: "gte",
      },
    },
    {
      accessorKey: "loaded",
      header: t("Loaded"),
      cell: ({ row }) => <BooleanBadge value={row.original.loaded} />,
      size: 90,
      meta: {
        apiField: "loaded",
        filterable: true,
        sortable: true,
        filterType: "boolean",
        filterOptions: LOADED_OPTIONS,
        defaultFilterOperator: "eq",
        exportValue: (row: IftaMileageEntryRow) => (row.loaded ? "Loaded" : "Empty"),
      },
    },
    {
      accessorKey: "source",
      header: t("Source"),
      cell: ({ row }) => (
        <Badge variant={SOURCE_VARIANTS[row.original.source]} className="px-1.5 py-0 text-[10px]">
          {IFTA_MILEAGE_SOURCE_LABELS[row.original.source]}
        </Badge>
      ),
      size: 130,
      meta: {
        apiField: "source",
        filterable: true,
        sortable: true,
        filterType: "select",
        filterOptions: iftaMileageSourceChoices,
        defaultFilterOperator: "eq",
        exportValue: (row: IftaMileageEntryRow) => IFTA_MILEAGE_SOURCE_LABELS[row.source],
      },
    },
    {
      accessorKey: "notes",
      header: t("Notes"),
      cell: ({ row }) => (
        <span className="block max-w-[28ch] truncate text-xs" title={row.original.notes ?? ""}>
          {row.original.notes ?? "—"}
        </span>
      ),
      enableSorting: false,
      meta: {
        apiField: "notes",
        filterable: true,
        sortable: false,
        filterType: "text",
        defaultFilterOperator: "contains",
      },
    },
    {
      accessorKey: "createdAt",
      header: t("Created"),
      cell: ({ row }) => (
        <HoverCardTimestamp
          className="font-table tracking-tight"
          timestamp={row.original.createdAt}
        />
      ),
      meta: {
        apiField: "createdAt",
        filterable: false,
        sortable: true,
        filterType: "date",
        defaultFilterOperator: "daterange",
      },
    },
    {
      accessorKey: "updatedAt",
      header: t("Updated"),
      cell: ({ row }) => (
        <HoverCardTimestamp
          className="font-table tracking-tight"
          timestamp={row.original.updatedAt}
        />
      ),
      meta: {
        apiField: "updatedAt",
        filterable: false,
        sortable: true,
        filterType: "date",
        defaultFilterOperator: "daterange",
      },
    },
  ];
}
