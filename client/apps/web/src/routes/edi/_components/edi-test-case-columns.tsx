import type { TranslateFn } from "@trenova/shared/i18n/use-t";
import { DataTablePlaceholder } from "@/components/data-table/_components/data-table-components";
import { HoverCardTimestamp } from "@/components/hover-card-timestamp";
import { Badge } from "@trenova/shared/components/ui/badge";
import type { EDITestCaseTableRow } from "@/lib/graphql/edi-table";
import type { ColumnDef } from "@trenova/shared/types/data-table";

export function getTestCaseColumns(t: TranslateFn): ColumnDef<EDITestCaseTableRow>[] {
  return [
    {
      accessorKey: "name",
      header: t("Name"),
      cell: ({ row }) => (
        <div className="min-w-0">
          <div className="truncate font-medium">{row.original.name}</div>
          {row.original.description ? (
            <div className="text-muted-foreground truncate text-xs">
              {t(row.original.description)}
            </div>
          ) : null}
        </div>
      ),
      size: 260,
      meta: {
        label: t("Name"),
        apiField: "name",
        filterable: true,
        sortable: true,
        filterType: "text",
        defaultFilterOperator: "contains",
      },
    },
    {
      id: "partner",
      header: t("Partner"),
      cell: ({ row }) =>
        row.original.documentProfile?.partner ? (
          <div className="min-w-0">
            <div className="truncate font-medium">{row.original.documentProfile.partner.name}</div>
            <div className="text-muted-foreground truncate text-xs">
              {row.original.documentProfile.partner.code}
            </div>
          </div>
        ) : (
          <DataTablePlaceholder />
        ),
      size: 220,
      meta: {
        label: t("Partner"),
        apiField: "partnerDocumentProfileId",
        filterable: false,
        sortable: false,
      },
    },
    {
      id: "transaction",
      header: t("Transaction"),
      cell: ({ row }) =>
        row.original.documentProfile ? (
          <div className="flex items-center gap-2">
            <Badge variant="secondary">{row.original.documentProfile.transactionSet}</Badge>
            <Badge variant="outline">{row.original.documentProfile.direction}</Badge>
          </div>
        ) : (
          <DataTablePlaceholder />
        ),
      size: 170,
      meta: {
        label: t("Transaction"),
        apiField: "partnerDocumentProfileId",
        filterable: false,
        sortable: false,
      },
    },
    {
      id: "documentProfile",
      header: t("Document Profile"),
      cell: ({ row }) =>
        row.original.documentProfile?.name ? (
          <span className="truncate">{row.original.documentProfile.name}</span>
        ) : (
          <DataTablePlaceholder />
        ),
      size: 220,
      meta: {
        label: t("Document Profile"),
        apiField: "partnerDocumentProfileId",
        filterable: false,
        sortable: false,
      },
    },
    {
      id: "expectations",
      header: t("Expected Outcome"),
      cell: ({ row }) => {
        const { expectedWarnings, expectedErrors } = row.original;
        if (expectedWarnings === 0 && expectedErrors === 0) {
          return <Badge variant="outline">{t("Clean")}</Badge>;
        }
        return (
          <div className="flex items-center gap-1.5">
            {expectedWarnings > 0 && (
              <Badge variant="secondary">
                {t("{0, plural, one {# warning} other {# warnings}}", expectedWarnings)}
              </Badge>
            )}
            {expectedErrors > 0 && (
              <Badge variant="warning">
                {t("{0, plural, one {# error} other {# errors}}", expectedErrors)}
              </Badge>
            )}
          </div>
        );
      },
      size: 180,
      meta: {
        label: t("Expected Outcome"),
        apiField: "expectedWarnings",
        filterable: false,
        sortable: false,
      },
    },
    {
      accessorKey: "updatedAt",
      header: t("Updated"),
      cell: ({ row }) =>
        row.original.updatedAt ? (
          <HoverCardTimestamp timestamp={row.original.updatedAt} />
        ) : (
          <DataTablePlaceholder />
        ),
      size: 180,
      meta: {
        label: t("Updated"),
        apiField: "updatedAt",
        filterable: false,
        sortable: true,
        filterType: "date",
      },
    },
  ];
}
