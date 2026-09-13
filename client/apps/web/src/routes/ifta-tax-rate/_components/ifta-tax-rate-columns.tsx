import type { TranslateFn } from "@trenova/shared/i18n/use-t";
import { HoverCardTimestamp } from "@/components/hover-card-timestamp";
import { jurisdictionLabel } from "@/lib/ifta-jurisdiction-options";
import { iftaFuelTypeChoices, iftaQuarterChoices } from "@/lib/choices";
import type { IftaTaxRateRow } from "@/lib/graphql/ifta-tax-rate";
import { Badge } from "@trenova/shared/components/ui/badge";
import type { ColumnDef } from "@trenova/shared/types/data-table";
import { formatDecimalString } from "@trenova/shared/types/decimal";
import type { GenericSelectOption } from "@trenova/shared/types/fields";
import { IFTA_FUEL_TYPE_LABELS } from "@trenova/shared/types/fuel-ifta-enums";
import { IFTA_RATE_SCALE } from "@trenova/shared/types/ifta-tax-rate";

export function periodLabel(year: number, quarter: number): string {
  return `Q${quarter} ${year}`;
}

function rate(value: string | null | undefined): string {
  return value ? formatDecimalString(value, IFTA_RATE_SCALE) : "—";
}

export function getColumns(
  jurisdictionOptions: readonly GenericSelectOption<string>[],
  t: TranslateFn,
): ColumnDef<IftaTaxRateRow>[] {
  return [
    {
      accessorKey: "year",
      header: t("Year"),
      cell: ({ row }) => <span className="font-table tabular-nums">{row.original.year}</span>,
      size: 90,
      meta: {
        apiField: "year",
        filterable: true,
        sortable: true,
        filterType: "number",
        defaultFilterOperator: "eq",
      },
    },
    {
      accessorKey: "quarter",
      header: t("Quarter"),
      cell: ({ row }) => (
        <span className="font-table tabular-nums">{t("Q{0}", row.original.quarter)}</span>
      ),
      size: 100,
      meta: {
        apiField: "quarter",
        filterable: true,
        sortable: true,
        filterType: "select",
        filterOptions: iftaQuarterChoices,
        defaultFilterOperator: "eq",
        exportValue: (row: IftaTaxRateRow) => `Q${row.quarter}`,
      },
    },
    {
      accessorKey: "jurisdictionId",
      header: t("Jurisdiction"),
      cell: ({ row }) => (
        <span className="flex items-center gap-2 font-medium">
          {jurisdictionLabel(row.original.jurisdiction)}
          {row.original.jurisdiction.hasSurcharge ? (
            <Badge variant="orange" className="px-1.5 py-0 text-[10px]">
              {t("Surcharge")}
            </Badge>
          ) : null}
          {row.original.jurisdiction.isIftaMember ? null : (
            <Badge variant="outline" className="px-1.5 py-0 text-[10px]">
              {t("Non-member")}
            </Badge>
          )}
        </span>
      ),
      enableSorting: false,
      meta: {
        apiField: "jurisdictionId",
        filterable: true,
        sortable: false,
        filterType: "select",
        filterOptions: [...jurisdictionOptions],
        defaultFilterOperator: "eq",
        exportValue: (row: IftaTaxRateRow) => jurisdictionLabel(row.jurisdiction),
      },
    },
    {
      accessorKey: "fuelType",
      header: t("Fuel"),
      cell: ({ row }) => IFTA_FUEL_TYPE_LABELS[row.original.fuelType],
      size: 140,
      meta: {
        apiField: "fuelType",
        filterable: true,
        sortable: true,
        filterType: "select",
        filterOptions: iftaFuelTypeChoices,
        defaultFilterOperator: "eq",
      },
    },
    {
      accessorKey: "ratePerGallon",
      header: t("Rate / gal"),
      cell: ({ row }) => (
        <span className="font-table block text-right tabular-nums">
          {rate(row.original.ratePerGallon)}
        </span>
      ),
      size: 110,
      meta: {
        apiField: "ratePerGallon",
        filterable: false,
        sortable: true,
      },
    },
    {
      accessorKey: "surchargeRatePerGallon",
      header: t("Surcharge / gal"),
      cell: ({ row }) => (
        <span className="font-table block text-right tabular-nums">
          {rate(row.original.surchargeRatePerGallon)}
        </span>
      ),
      size: 130,
      meta: {
        apiField: "surchargeRatePerGallon",
        filterable: false,
        sortable: true,
      },
    },
    {
      accessorKey: "sourceNote",
      header: t("Source"),
      cell: ({ row }) =>
        row.original.sourceUrl ? (
          <a
            href={row.original.sourceUrl}
            target="_blank"
            rel="noreferrer"
            className="text-xs underline-offset-2 hover:underline"
          >
            {row.original.sourceNote || row.original.sourceUrl}
          </a>
        ) : (
          <span className="text-xs">{row.original.sourceNote || "—"}</span>
        ),
      enableSorting: false,
      meta: {
        apiField: "sourceNote",
        filterable: true,
        sortable: false,
        filterType: "text",
        defaultFilterOperator: "contains",
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
