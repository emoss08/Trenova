import type { AccountingDriftLabels } from "@/hooks/use-accounting-drift-labels";
import type { AccountingSyncLabels } from "@/hooks/use-accounting-sync-labels";
import {
  ACCOUNTING_DRIFT_KINDS,
  ACCOUNTING_DRIFT_STATUSES,
  accountingDriftPhase,
  accountingDriftWithinTolerance,
  formatAccountingMinor,
} from "@/lib/accounting-sync";
import type { AccountingDriftRow } from "@/lib/graphql/accounting-drift-table";
import { Badge } from "@trenova/shared/components/ui/badge";
import type { TranslateFn } from "@trenova/shared/i18n/use-t";
import { formatUnixDateTimeShort } from "@trenova/shared/lib/date";
import { phaseTone } from "@trenova/shared/lib/status-phase";
import type { ColumnDef } from "@trenova/shared/types/data-table";

function minorOrDash(minor: number | null | undefined, currency: string): string {
  return minor == null ? "—" : formatAccountingMinor(minor, currency);
}

export function getDriftColumns(
  t: TranslateFn,
  labels: AccountingDriftLabels,
  objectLabels: AccountingSyncLabels["objectType"],
  toleranceMinor: number,
): ColumnDef<AccountingDriftRow>[] {
  return [
    {
      accessorKey: "status",
      header: t("Status"),
      cell: ({ row }) => (
        <Badge variant={phaseTone(accountingDriftPhase(row.original.status))}>
          {row.original.pushed ? t("Sent, waiting to match") : labels.status[row.original.status]}
        </Badge>
      ),
      size: 170,
      minSize: 130,
      maxSize: 200,
      meta: {
        label: t("Status"),
        apiField: "status",
        filterable: true,
        sortable: true,
        filterType: "select",
        filterOptions: ACCOUNTING_DRIFT_STATUSES.map((value) => ({
          value,
          label: labels.status[value],
        })),
        defaultFilterOperator: "eq",
      },
    },
    {
      accessorKey: "kind",
      header: t("Difference"),
      cell: ({ row }) => labels.kind[row.original.kind],
      size: 200,
      minSize: 150,
      maxSize: 240,
      meta: {
        label: t("Difference"),
        apiField: "kind",
        filterable: true,
        sortable: true,
        filterType: "select",
        filterOptions: ACCOUNTING_DRIFT_KINDS.map((value) => ({
          value,
          label: labels.kind[value],
        })),
        defaultFilterOperator: "eq",
      },
    },
    {
      accessorKey: "objectNumber",
      header: t("Document"),
      cell: ({ row }) => (
        <span className="truncate">
          {objectLabels[row.original.objectType]}{" "}
          <span className="font-mono text-xs">{row.original.objectNumber}</span>
        </span>
      ),
      size: 200,
      minSize: 150,
      maxSize: 260,
      meta: {
        label: t("Document"),
        apiField: "objectNumber",
        filterable: true,
        sortable: true,
        filterType: "text",
        defaultFilterOperator: "contains",
      },
    },
    {
      accessorKey: "partyName",
      header: t("Customer, carrier or driver"),
      cell: ({ row }) => row.original.partyName,
      size: 200,
      minSize: 150,
      maxSize: 280,
      meta: {
        label: t("Customer, carrier or driver"),
        apiField: "partyName",
        filterable: true,
        sortable: true,
        filterType: "text",
        defaultFilterOperator: "contains",
      },
    },
    {
      accessorKey: "trenovaMinor",
      header: t("In Trenova"),
      cell: ({ row }) => (
        <span className="tabular-nums">
          {row.original.trenovaMinor == null
            ? row.original.trenovaState
            : formatAccountingMinor(row.original.trenovaMinor, row.original.currencyCode)}
        </span>
      ),
      size: 140,
      minSize: 110,
      maxSize: 170,
      meta: { label: t("In Trenova"), apiField: "trenovaMinor", sortable: true },
    },
    {
      accessorKey: "providerMinor",
      header: t("In the books"),
      cell: ({ row }) => (
        <span className="tabular-nums">
          {row.original.providerMinor == null
            ? row.original.providerState
            : formatAccountingMinor(row.original.providerMinor, row.original.currencyCode)}
        </span>
      ),
      size: 140,
      minSize: 110,
      maxSize: 170,
      meta: { label: t("In the books"), apiField: "providerMinor", sortable: true },
    },
    {
      accessorKey: "differenceMinor",
      header: t("Difference"),
      cell: ({ row }) => (
        <span className="flex items-center gap-2 tabular-nums">
          {minorOrDash(row.original.differenceMinor, row.original.currencyCode)}
          {accountingDriftWithinTolerance(row.original, toleranceMinor) ? (
            <Badge variant="neutral" appearance="outline">
              {t("Within tolerance")}
            </Badge>
          ) : null}
        </span>
      ),
      size: 200,
      minSize: 140,
      maxSize: 240,
      meta: { label: t("Amount difference"), apiField: "differenceMinor", sortable: true },
    },
    {
      accessorKey: "providerModifiedAt",
      header: t("Changed in the books"),
      cell: ({ row }) =>
        row.original.providerModifiedAt
          ? row.original.providerModifiedBy
            ? t(
                "{0} by {1}",
                formatUnixDateTimeShort(row.original.providerModifiedAt),
                row.original.providerModifiedBy,
              )
            : formatUnixDateTimeShort(row.original.providerModifiedAt)
          : "",
      size: 220,
      minSize: 160,
      maxSize: 280,
      meta: {
        label: t("Changed in the books"),
        apiField: "providerModifiedAt",
        filterable: true,
        sortable: true,
        filterType: "date",
      },
    },
  ];
}
