import type { TranslateFn } from "@trenova/shared/i18n/use-t";
import { ColorOptionValue } from "@/components/fields/select-components";
import { HoverCardTimestamp } from "@/components/hover-card-timestamp";
import { formulaTemplateStatusChoices } from "@/lib/choices";
import { Badge } from "@trenova/shared/components/ui/badge";
import type { ColumnDef } from "@trenova/shared/types/data-table";
import type { FormulaTemplateRow } from "@/lib/graphql/formula-template-table";

export function getColumns(t: TranslateFn): ColumnDef<FormulaTemplateRow>[] {
  return [
    {
      accessorKey: "name",
      header: t("Name"),
      cell: ({ row }) => (
        <div className="flex items-center gap-3">
          <div className="min-w-0">
            <span className="text-sm font-medium">{row.original.name}</span>
            {row.original.description && (
              <p className="text-2xs text-muted-foreground line-clamp-1">
                {t(row.original.description)}
              </p>
            )}
          </div>
        </div>
      ),
      meta: {
        label: t("Name"),
        apiField: "name",
        filterable: true,
        sortable: true,
        filterType: "text",
        defaultFilterOperator: "contains",
      },
      size: 280,
      minSize: 220,
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
      id: "imports",
      header: t("Libraries"),
      cell: ({ row }) =>
        row.original.imports.length > 0 ? (
          <span className="flex flex-wrap gap-1">
            {row.original.imports.map((library) => (
              <Badge key={library.path} variant="neutral" appearance="outline" className="font-mono">
                {library.path}
              </Badge>
            ))}
          </span>
        ) : (
          <span className="text-muted-foreground text-xs">{t("None")}</span>
        ),
      meta: {
        label: t("Libraries"),
        apiField: "imports",
        filterable: false,
        sortable: false,
      },
      size: 180,
      minSize: 120,
      maxSize: 280,
    },
    {
      accessorKey: "currentVersionNumber",
      header: t("Version"),
      cell: ({ row }) => (
        <Badge variant="neutral" appearance="outline" className="font-mono">
          {row.original.currentVersionNumber ? t("v{0}", row.original.currentVersionNumber) : "—"}
        </Badge>
      ),
      meta: {
        label: t("Version"),
        apiField: "currentVersionNumber",
        filterable: false,
        sortable: true,
        filterType: "number",
      },
      size: 80,
      minSize: 60,
      maxSize: 100,
    },
    {
      accessorKey: "usageCount",
      header: t("In use"),
      cell: ({ row }) => {
        const count = row.original.usageCount ?? 0;
        return count > 0 ? (
          <Badge variant="info" className="font-mono">
            {count}
          </Badge>
        ) : (
          <span className="text-muted-foreground text-xs">{t("Not in use")}</span>
        );
      },
      meta: {
        label: t("In use"),
        apiField: "usageCount",
        filterable: false,
        sortable: false,
        filterType: "number",
      },
      size: 100,
      minSize: 90,
      maxSize: 120,
    },
    {
      accessorKey: "testCount",
      header: t("Tests"),
      cell: ({ row }) => {
        const count = row.original.testCount;
        return count > 0 ? (
          <Badge variant="neutral" appearance="outline" className="font-mono">
            {count}
          </Badge>
        ) : (
          <span className="text-muted-foreground text-xs">{t("None")}</span>
        );
      },
      meta: {
        label: t("Tests"),
        apiField: "testCount",
        filterable: false,
        sortable: false,
        filterType: "number",
      },
      size: 100,
      minSize: 90,
      maxSize: 120,
    },
    {
      accessorKey: "approvedAt",
      header: t("Approved"),
      cell: ({ row }) =>
        row.original.approvedAt ? (
          <HoverCardTimestamp timestamp={row.original.approvedAt} />
        ) : (
          <span className="text-muted-foreground text-xs">{t("Never")}</span>
        ),
      meta: {
        label: t("Approved"),
        apiField: "approvedAt",
        filterable: true,
        sortable: true,
        filterType: "date",
        defaultFilterOperator: "gt",
      },
      size: 160,
      minSize: 140,
      maxSize: 200,
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
    {
      accessorKey: "createdAt",
      header: t("Created At"),
      cell: ({ row }) => <HoverCardTimestamp timestamp={row.original.createdAt} />,
      meta: {
        label: t("Created At"),
        apiField: "createdAt",
        filterable: true,
        sortable: true,
        filterType: "date",
        defaultFilterOperator: "gt",
      },
      size: 180,
      minSize: 160,
      maxSize: 220,
    },
  ];
}
