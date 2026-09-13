import type { TranslateFn } from "@trenova/shared/i18n/use-t";
import { EntityRefCell } from "@/components/data-table/_components/entity-ref-link";
import { jurisdictionLabel } from "@/lib/ifta-jurisdiction-options";
import { HoverCardTimestamp } from "@/components/hover-card-timestamp";
import { fuelPurchaseSourceChoices, iftaFuelTypeChoices } from "@/lib/choices";
import type { TractorRow } from "@/lib/graphql/equipment-table";
import type { FuelPurchaseRow } from "@/lib/graphql/fuel-purchase";
import { Badge } from "@trenova/shared/components/ui/badge";
import { formatCurrency } from "@trenova/shared/lib/utils";
import type { ColumnDef } from "@trenova/shared/types/data-table";
import { formatDecimalString } from "@trenova/shared/types/decimal";
import type { GenericSelectOption } from "@trenova/shared/types/fields";
import {
  FUEL_PURCHASE_SOURCE_LABELS,
  IFTA_FUEL_TYPE_LABELS,
} from "@trenova/shared/types/fuel-ifta-enums";
import { FUEL_QUANTITY_SCALE } from "@trenova/shared/types/fuel-purchase";

const TAX_PAID_OPTIONS = [
  { value: true, label: "Tax paid", color: "#15803d" },
  { value: false, label: "Untaxed", color: "#b45309" },
] satisfies ReadonlyArray<GenericSelectOption<boolean>>;

export function getColumns(
  jurisdictionOptions: readonly GenericSelectOption<string>[],
  t: TranslateFn,
): ColumnDef<FuelPurchaseRow>[] {
  return [
    {
      accessorKey: "purchasedAt",
      header: t("Purchased"),
      cell: ({ row }) => (
        <HoverCardTimestamp
          className="font-table tracking-tight"
          timestamp={row.original.purchasedAt}
        />
      ),
      size: 170,
      meta: {
        apiField: "purchasedAt",
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
          <EntityRefCell<NonNullable<FuelPurchaseRow["tractor"]>, TractorRow>
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
        exportValue: (row: FuelPurchaseRow) => row.tractor?.code ?? "",
      },
    },
    {
      id: "worker",
      header: t("Worker"),
      cell: ({ row }) => {
        const { worker } = row.original;

        if (!worker) {
          return <p className="text-muted-foreground">-</p>;
        }

        return (
          <EntityRefCell<NonNullable<FuelPurchaseRow["worker"]>, FuelPurchaseRow>
            entity={worker}
            config={{
              basePath: "/workers",
              getId: (worker) => worker.id,
              getDisplayText: (worker) =>
                [worker.firstName, worker.lastName].filter(Boolean).join(" "),
              getHeaderText: "Worker",
            }}
            parent={row.original}
          />
        );
      },
      enableSorting: false,
      meta: {
        apiField: "workerId",
        filterable: false,
        sortable: false,
        exportValue: (row: FuelPurchaseRow) => row.worker?.wholeName ?? "",
      },
    },
    {
      accessorKey: "jurisdictionId",
      header: t("Jurisdiction"),
      cell: ({ row }) => (
        <span className="font-table">{jurisdictionLabel(row.original.jurisdiction)}</span>
      ),
      enableSorting: false,
      size: 150,
      meta: {
        apiField: "jurisdictionId",
        filterable: true,
        sortable: false,
        filterType: "select",
        filterOptions: [...jurisdictionOptions],
        defaultFilterOperator: "eq",
        exportValue: (row: FuelPurchaseRow) => jurisdictionLabel(row.jurisdiction),
      },
    },
    {
      accessorKey: "vendor",
      header: t("Vendor"),
      cell: ({ row }) =>
        row.original.vendor ? (
          <span className="flex flex-col">
            <span>{row.original.vendor}</span>
            {row.original.vendorCity ? (
              <span className="text-muted-foreground text-2xs">{row.original.vendorCity}</span>
            ) : null}
          </span>
        ) : (
          "—"
        ),
      meta: {
        apiField: "vendor",
        filterable: true,
        sortable: true,
        filterType: "text",
        defaultFilterOperator: "contains",
      },
    },
    {
      accessorKey: "fuelType",
      header: t("Fuel"),
      cell: ({ row }) => IFTA_FUEL_TYPE_LABELS[row.original.fuelType],
      size: 110,
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
      accessorKey: "gallons",
      header: t("Gallons"),
      cell: ({ row }) => (
        <span className="font-table block text-right tabular-nums">
          {formatDecimalString(row.original.gallons, FUEL_QUANTITY_SCALE)}
        </span>
      ),
      size: 100,
      meta: {
        apiField: "gallons",
        filterable: true,
        sortable: true,
        filterType: "number",
        defaultFilterOperator: "gte",
      },
    },
    {
      accessorKey: "totalAmount",
      header: t("Amount"),
      cell: ({ row }) => (
        <span className="font-table block text-right tabular-nums">
          {formatCurrency(Number(row.original.totalAmount), row.original.currencyCode)}
        </span>
      ),
      size: 110,
      meta: {
        apiField: "totalAmount",
        filterable: true,
        sortable: true,
        filterType: "number",
        defaultFilterOperator: "gte",
      },
    },
    {
      accessorKey: "taxPaid",
      header: t("Tax"),
      cell: ({ row }) => (
        <Badge
          variant={row.original.taxPaid ? "active" : "warning"}
          className="px-1.5 py-0 text-[10px]"
        >
          {row.original.taxPaid ? t("Paid") : t("Untaxed")}
        </Badge>
      ),
      size: 90,
      meta: {
        apiField: "taxPaid",
        filterable: true,
        sortable: true,
        filterType: "boolean",
        filterOptions: TAX_PAID_OPTIONS,
        defaultFilterOperator: "eq",
        exportValue: (row: FuelPurchaseRow) => (row.taxPaid ? "Paid" : "Untaxed"),
      },
    },
    {
      accessorKey: "source",
      header: t("Source"),
      cell: ({ row }) => FUEL_PURCHASE_SOURCE_LABELS[row.original.source],
      size: 110,
      meta: {
        apiField: "source",
        filterable: true,
        sortable: true,
        filterType: "select",
        filterOptions: fuelPurchaseSourceChoices,
        defaultFilterOperator: "eq",
      },
    },
    {
      accessorKey: "transactionReference",
      header: t("Reference"),
      cell: ({ row }) => (
        <span className="font-table text-xs">{row.original.transactionReference ?? "—"}</span>
      ),
      meta: {
        apiField: "transactionReference",
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
