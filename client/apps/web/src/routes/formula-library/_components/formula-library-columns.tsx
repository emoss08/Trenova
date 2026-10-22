import { ColorOptionValue } from "@/components/fields/select-components";
import { HoverCardTimestamp } from "@/components/hover-card-timestamp";
import { formulaTemplateStatusChoices } from "@/lib/choices";
import type { FormulaLibraryRow } from "@/lib/graphql/formula-library";
import { Badge } from "@trenova/shared/components/ui/badge";
import type { TranslateFn } from "@trenova/shared/i18n/use-t";
import type { ColumnDef } from "@trenova/shared/types/data-table";

export function getColumns(t: TranslateFn): ColumnDef<FormulaLibraryRow>[] {
  return [
    {
      accessorKey: "name",
      header: t("Module"),
      cell: ({ row }) => (
        <div className="min-w-0">
          <span className="font-mono text-sm">org.{row.original.name}</span>
          {row.original.description && (
            <p className="text-2xs text-muted-foreground line-clamp-1">
              {row.original.description}
            </p>
          )}
        </div>
      ),
      meta: {
        label: t("Module"),
        apiField: "name",
        filterable: true,
        sortable: true,
        filterType: "text",
        defaultFilterOperator: "contains",
      },
      size: 280,
      minSize: 200,
      maxSize: 400,
    },
    {
      accessorKey: "status",
      header: t("Status"),
      cell: ({ row }) => {
        const choice = formulaTemplateStatusChoices.find(
          (option) => option.value === row.original.status,
        );
        return choice ? (
          <ColorOptionValue color={choice.color} value={choice.label} />
        ) : (
          row.original.status
        );
      },
      meta: {
        label: t("Status"),
        apiField: "status",
        filterable: true,
        sortable: true,
        filterType: "select",
        filterOptions: formulaTemplateStatusChoices,
        defaultFilterOperator: "eq",
      },
      size: 120,
      minSize: 100,
      maxSize: 150,
    },
    {
      accessorKey: "approvedVersionNumber",
      header: t("Approved"),
      cell: ({ row }) =>
        row.original.approvedVersionNumber ? (
          <Badge variant="neutral" appearance="outline" className="font-mono">
            {t("v{0}", row.original.approvedVersionNumber)}
          </Badge>
        ) : (
          <span className="text-muted-foreground text-xs">{t("Never")}</span>
        ),
      meta: {
        label: t("Approved"),
        apiField: "approvedVersionNumber",
        filterable: false,
        sortable: true,
        filterType: "number",
      },
      size: 100,
      minSize: 80,
      maxSize: 120,
    },
    {
      accessorKey: "testCount",
      header: t("Tests"),
      cell: ({ row }) =>
        row.original.testCount > 0 ? (
          <Badge variant="neutral" appearance="outline" className="font-mono">
            {row.original.testCount}
          </Badge>
        ) : (
          <span className="text-muted-foreground text-xs">{t("None")}</span>
        ),
      meta: {
        label: t("Tests"),
        apiField: "testCount",
        filterable: false,
        sortable: false,
        filterType: "number",
      },
      size: 90,
      minSize: 80,
      maxSize: 110,
    },
    {
      accessorKey: "usageCount",
      header: t("Imported by"),
      cell: ({ row }) =>
        row.original.usageCount > 0 ? (
          <Badge variant="info" className="font-mono">
            {t("{0, plural, one {# template version} other {# template versions}}", row.original.usageCount)}
          </Badge>
        ) : (
          <span className="text-muted-foreground text-xs">{t("Not imported")}</span>
        ),
      meta: {
        label: t("Imported by"),
        apiField: "usageCount",
        filterable: false,
        sortable: false,
        filterType: "number",
      },
      size: 170,
      minSize: 120,
      maxSize: 220,
    },
    {
      accessorKey: "updatedAt",
      header: t("Updated"),
      cell: ({ row }) => <HoverCardTimestamp timestamp={row.original.updatedAt} />,
      meta: {
        label: t("Updated"),
        apiField: "updatedAt",
        filterable: true,
        sortable: true,
        filterType: "date",
        defaultFilterOperator: "gt",
      },
      size: 160,
      minSize: 140,
      maxSize: 200,
    },
  ];
}
