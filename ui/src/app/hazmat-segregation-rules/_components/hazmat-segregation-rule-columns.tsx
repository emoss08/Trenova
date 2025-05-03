import { createCommonColumns } from "@/components/data-table/_components/data-table-column-helpers";
import { type HazmatSegregationRuleSchema } from "@/lib/schemas/hazmat-segregation-rule-schema";
import {
  HazardousClassChoiceProps,
  mapToHazardousClassChoice,
} from "@/types/hazardous-material";
import { createColumnHelper, type ColumnDef } from "@tanstack/react-table";

export function getColumns(): ColumnDef<HazmatSegregationRuleSchema>[] {
  const columnHelper = createColumnHelper<HazmatSegregationRuleSchema>();
  const commonColumns = createCommonColumns(columnHelper);

  return [
    commonColumns.status,
    columnHelper.display({
      id: "name",
      header: "Name",
      enableResizing: true,
      cell: ({ row }) => <p>{row.original.name}</p>,
    }),
    columnHelper.display({
      id: "classA",
      header: "Class A",
      enableResizing: true,
      cell: ({ row }) =>
        mapToHazardousClassChoice(
          row.original.classA as HazardousClassChoiceProps,
        ),
    }),
    columnHelper.display({
      id: "classB",
      header: "Class B",
      enableResizing: true,
      cell: ({ row }) =>
        mapToHazardousClassChoice(
          row.original.classB as HazardousClassChoiceProps,
        ),
    }),
    commonColumns.description,
    commonColumns.createdAt,
  ];
}
