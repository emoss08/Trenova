import { HoverCardTimestamp } from "@/components/hover-card-timestamp";
import type { AICorrectionRow } from "@/lib/graphql/extraction-eval";
import type { TranslateFn } from "@trenova/shared/i18n/use-t";
import type { ColumnDef } from "@trenova/shared/types/data-table";
import { formatShare } from "../quality-model";
import { documentKindLabel, modelLabel } from "./extraction-model";

function count(value: number) {
  return <span className="tabular-nums">{value}</span>;
}

export function getCorrectionColumns(t: TranslateFn): ColumnDef<AICorrectionRow>[] {
  return [
    {
      accessorKey: "capturedAt",
      header: t("Captured"),
      cell: ({ row }) => <HoverCardTimestamp timestamp={row.original.capturedAt} />,
      size: 170,
      meta: {
        label: t("Captured"),
        apiField: "capturedAt",
        filterable: true,
        sortable: true,
        filterType: "date",
      },
    },
    {
      accessorKey: "documentKind",
      header: t("Document"),
      cell: ({ row }) => documentKindLabel(row.original.documentKind, t),
      size: 170,
      meta: {
        label: t("Document"),
        apiField: "documentKind",
        filterable: true,
        sortable: true,
        filterType: "text",
      },
    },
    {
      accessorKey: "documentFingerprint",
      header: t("Issuer"),
      cell: ({ row }) =>
        row.original.documentFingerprint || <span className="text-muted-foreground">—</span>,
      size: 180,
      meta: {
        label: t("Issuer"),
        apiField: "documentFingerprint",
        filterable: true,
        sortable: true,
        filterType: "text",
      },
    },
    {
      accessorKey: "extractionModel",
      header: t("Model"),
      cell: ({ row }) => modelLabel(row.original.extractionModel, t),
      size: 200,
      meta: {
        label: t("Model"),
        apiField: "extractionModel",
        filterable: true,
        sortable: true,
        filterType: "text",
      },
    },
    {
      accessorKey: "accuracy",
      header: t("Accuracy"),
      cell: ({ row }) => (
        <span className="tabular-nums">
          {row.original.scoredCount > 0 ? formatShare(row.original.accuracy) : "—"}
        </span>
      ),
      size: 110,
      meta: { label: t("Accuracy"), apiField: "accuracy", filterable: false, sortable: false },
    },
    {
      accessorKey: "correctedCount",
      header: t("Corrected"),
      cell: ({ row }) => count(row.original.correctedCount),
      size: 110,
      meta: {
        label: t("Corrected"),
        apiField: "correctedCount",
        filterable: true,
        sortable: true,
        filterType: "number",
      },
    },
    {
      accessorKey: "missedCount",
      header: t("Missed"),
      cell: ({ row }) => count(row.original.missedCount),
      size: 100,
      meta: {
        label: t("Missed"),
        apiField: "missedCount",
        filterable: true,
        sortable: true,
        filterType: "number",
      },
    },
    {
      accessorKey: "scoredCount",
      header: t("Fields scored"),
      cell: ({ row }) => count(row.original.scoredCount),
      size: 120,
      meta: {
        label: t("Fields scored"),
        apiField: "scoredCount",
        filterable: true,
        sortable: true,
        filterType: "number",
      },
    },
  ];
}
