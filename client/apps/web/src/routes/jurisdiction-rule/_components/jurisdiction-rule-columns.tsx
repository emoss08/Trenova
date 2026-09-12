import type { TranslateFn } from "@trenova/shared/i18n/use-t";
import { HoverCardTimestamp } from "@/components/hover-card-timestamp";
import { Badge } from "@trenova/shared/components/ui/badge";
import { formatFeetInches, formatPounds } from "@trenova/shared/lib/permit";
import type { JurisdictionRuleRow } from "@/lib/graphql/jurisdiction-rule-table";
import type { JurisdictionVerificationState } from "@/types/jurisdiction-rule";
import type { ColumnDef } from "@trenova/shared/types/data-table";

const VERIFICATION_VARIANT: Record<
  JurisdictionVerificationState,
  "active" | "inactive" | "warning"
> = {
  Verified: "active",
  Unverified: "inactive",
  Disputed: "warning",
};

export function getColumns(t: TranslateFn): ColumnDef<JurisdictionRuleRow>[] {
  return [
    {
      accessorKey: "state",
      header: t("State"),
      cell: ({ row }) => (
        <span className="text-sm font-medium">
          {row.original.state?.abbreviation ?? "—"}
          <span className="text-muted-foreground ml-2">{row.original.state?.name}</span>
        </span>
      ),
      size: 200,
      meta: { label: t("State"), apiField: "stateId", sortable: true },
    },
    {
      accessorKey: "verificationState",
      header: t("Verification"),
      // The first column an operator should read. A limit nobody has confirmed
      // is a research baseline, and requirements derived from it say so.
      cell: ({ row }) => (
        <Badge variant={VERIFICATION_VARIANT[row.original.verificationState]}>
          {row.original.verificationState}
        </Badge>
      ),
      size: 130,
      meta: {
        label: t("Verification"),
        apiField: "verificationState",
        filterable: true,
        sortable: true,
        filterType: "select",
        defaultFilterOperator: "eq",
      },
    },
    {
      accessorKey: "status",
      header: t("Status"),
      cell: ({ row }) => (
        <Badge variant={row.original.status === "Active" ? "active" : "inactive"}>
          {row.original.status}
        </Badge>
      ),
      size: 110,
      meta: {
        label: t("Status"),
        apiField: "status",
        filterable: true,
        sortable: true,
        filterType: "select",
        defaultFilterOperator: "eq",
      },
    },
    {
      accessorKey: "maxWidthFeet",
      header: t("Max Width"),
      cell: ({ row }) => formatFeetInches(row.original.maxWidthFeet),
      size: 120,
      meta: { label: t("Max Width"), apiField: "maxWidthFeet", sortable: true },
    },
    {
      accessorKey: "maxHeightFeet",
      header: t("Max Height"),
      cell: ({ row }) => formatFeetInches(row.original.maxHeightFeet),
      size: 120,
      meta: { label: t("Max Height"), apiField: "maxHeightFeet", sortable: true },
    },
    {
      accessorKey: "maxLengthFeet",
      header: t("Max Length"),
      cell: ({ row }) => formatFeetInches(row.original.maxLengthFeet),
      size: 120,
      meta: { label: t("Max Length"), apiField: "maxLengthFeet", sortable: true },
    },
    {
      accessorKey: "maxWeightPounds",
      header: t("Max Weight"),
      cell: ({ row }) => formatPounds(row.original.maxWeightPounds),
      size: 140,
      meta: { label: t("Max Weight"), apiField: "maxWeightPounds", sortable: true },
    },
    {
      accessorKey: "permitLeadTimeDays",
      header: t("Lead Time"),
      cell: ({ row }) => {
        const days = row.original.permitLeadTimeDays;
        return `${days} day${days === 1 ? "" : "s"}`;
      },
      size: 110,
      meta: { label: t("Lead Time"), apiField: "permitLeadTimeDays", sortable: true },
    },
    {
      accessorKey: "verifiedAt",
      header: t("Verified"),
      cell: ({ row }) =>
        row.original.verifiedAt ? (
          <HoverCardTimestamp timestamp={row.original.verifiedAt} />
        ) : (
          <span className="text-muted-foreground">{t("Never")}</span>
        ),
      size: 150,
      meta: { label: t("Verified"), apiField: "verifiedAt", sortable: true },
    },
  ];
}
