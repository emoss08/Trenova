import { translate } from "@trenova/shared/i18n/runtime";
import type { TranslateFn } from "@trenova/shared/i18n/use-t";
import { DataTableDescription } from "@/components/data-table/_components/data-table-components";
import { ColorOptionValue } from "@/components/fields/select-components";
import { HoverCardTimestamp } from "@/components/hover-card-timestamp";
import {
  rateAgreementStatusChoices,
  rateAgreementTypeChoices,
  ratePartyTypeChoices,
} from "@/lib/choices";
import { Badge } from "@trenova/shared/components/ui/badge";
import type { ColumnDef } from "@trenova/shared/types/data-table";
import type { RateAgreementRow } from "@/lib/graphql/rate-tables";

/** An agreement with no end date runs until somebody ends it. */
function effectiveWindow(row: RateAgreementRow) {
  if (!row.effectiveTo) {
    return <span className="text-muted-foreground">{translate("No end date")}</span>;
  }

  return <HoverCardTimestamp timestamp={row.effectiveTo} />;
}

export function getColumns(t: TranslateFn): ColumnDef<RateAgreementRow>[] {
  return [
    {
      accessorKey: "status",
      header: t("Status"),
      cell: ({ row }) => {
        const choice = rateAgreementStatusChoices.find(
          (option) => option.value === row.original.status,
        );

        return choice ? (
          <ColorOptionValue color={choice.color} value={choice.label} />
        ) : (
          row.original.status
        );
      },
      size: 120,
      minSize: 110,
      maxSize: 140,
      meta: {
        label: t("Status"),
        apiField: "status",
        filterable: true,
        sortable: true,
        filterType: "select",
        filterOptions: rateAgreementStatusChoices,
        defaultFilterOperator: "eq",
      },
    },
    {
      accessorKey: "code",
      header: t("Code"),
      cell: ({ row }) => <span className="font-mono text-xs">{row.original.code}</span>,
      size: 140,
      minSize: 120,
      maxSize: 180,
      meta: {
        label: t("Code"),
        apiField: "code",
        filterable: true,
        sortable: true,
      },
    },
    {
      accessorKey: "name",
      header: t("Name"),
      cell: ({ row }) => (
        <div className="flex items-center gap-1.5">
          <span className="font-medium">{row.original.name}</span>
          {row.original.currentVersionNumber > 1 && (
            <Badge variant="outline" className="text-[10px]">
              {t("v{0}", row.original.currentVersionNumber)}
            </Badge>
          )}
        </div>
      ),
      size: 260,
      minSize: 200,
      maxSize: 340,
      meta: {
        label: t("Name"),
        apiField: "name",
        filterable: true,
        sortable: true,
      },
    },
    {
      accessorKey: "partyType",
      header: t("Side"),
      cell: ({ row }) => {
        const choice = ratePartyTypeChoices.find(
          (option) => option.value === row.original.partyType,
        );

        return choice ? (
          <ColorOptionValue color={choice.color} value={choice.label} />
        ) : (
          row.original.partyType
        );
      },
      size: 110,
      minSize: 100,
      maxSize: 130,
      meta: {
        label: t("Side"),
        apiField: "partyType",
        filterable: true,
        sortable: true,
        filterType: "select",
        filterOptions: ratePartyTypeChoices,
        defaultFilterOperator: "eq",
      },
    },
    {
      accessorKey: "agreementType",
      header: t("Type"),
      cell: ({ row }) =>
        rateAgreementTypeChoices.find((option) => option.value === row.original.agreementType)
          ?.label ?? row.original.agreementType,
      size: 120,
      minSize: 110,
      maxSize: 150,
      meta: {
        label: t("Type"),
        apiField: "agreementType",
        filterable: true,
        sortable: true,
        filterType: "select",
        filterOptions: rateAgreementTypeChoices,
        defaultFilterOperator: "eq",
      },
    },
    {
      accessorKey: "effectiveFrom",
      header: t("In force from"),
      cell: ({ row }) => <HoverCardTimestamp timestamp={row.original.effectiveFrom} />,
      size: 150,
      minSize: 130,
      maxSize: 190,
      meta: {
        label: t("In force from"),
        apiField: "effectiveFrom",
        filterable: true,
        sortable: true,
        filterType: "date",
      },
    },
    {
      accessorKey: "effectiveTo",
      header: t("Until"),
      cell: ({ row }) => effectiveWindow(row.original),
      size: 150,
      minSize: 130,
      maxSize: 190,
      meta: {
        label: t("Until"),
        apiField: "effectiveTo",
        filterable: true,
        sortable: true,
        filterType: "date",
      },
    },
    {
      accessorKey: "priority",
      header: t("Priority"),
      cell: ({ row }) => <span className="tabular-nums">{row.original.priority}</span>,
      size: 100,
      minSize: 90,
      maxSize: 120,
      meta: {
        label: t("Priority"),
        apiField: "priority",
        filterable: true,
        sortable: true,
        filterType: "number",
      },
    },
    {
      accessorKey: "currency",
      header: t("Currency"),
      size: 100,
      minSize: 90,
      maxSize: 120,
      meta: {
        label: t("Currency"),
        apiField: "currency",
        filterable: true,
        sortable: true,
      },
    },
    {
      accessorKey: "description",
      header: t("Description"),
      cell: ({ row }) => <DataTableDescription description={t(row.original.description)} />,
      size: 280,
      minSize: 200,
      maxSize: 400,
      meta: {
        label: t("Description"),
        apiField: "description",
        filterable: true,
        sortable: false,
      },
    },
  ];
}
