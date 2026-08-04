import { HoverCardTimestamp } from "@/components/hover-card-timestamp";
import { Badge } from "@trenova/shared/components/ui/badge";
import { formatFeetInches, formatPounds } from "@trenova/shared/lib/permit";
import type { JurisdictionRule, JurisdictionVerificationState } from "@/types/jurisdiction-rule";
import { type ColumnDef } from "@tanstack/react-table";

const VERIFICATION_VARIANT: Record<
  JurisdictionVerificationState,
  "active" | "inactive" | "warning"
> = {
  Verified: "active",
  Unverified: "inactive",
  Disputed: "warning",
};

export function getColumns(): ColumnDef<JurisdictionRule>[] {
  return [
    {
      accessorKey: "state",
      header: "State",
      cell: ({ row }) => (
        <span className="text-sm font-medium">
          {row.original.state?.abbreviation ?? "—"}
          <span className="ml-2 text-muted-foreground">{row.original.state?.name}</span>
        </span>
      ),
      size: 200,
      meta: { label: "State", apiField: "stateId", sortable: true },
    },
    {
      accessorKey: "verificationState",
      header: "Verification",
      // The first column an operator should read. A limit nobody has confirmed
      // is a research baseline, and requirements derived from it say so.
      cell: ({ row }) => (
        <Badge variant={VERIFICATION_VARIANT[row.original.verificationState]}>
          {row.original.verificationState}
        </Badge>
      ),
      size: 130,
      meta: {
        label: "Verification",
        apiField: "verificationState",
        filterable: true,
        sortable: true,
        filterType: "select",
        defaultFilterOperator: "eq",
      },
    },
    {
      accessorKey: "status",
      header: "Status",
      cell: ({ row }) => (
        <Badge variant={row.original.status === "Active" ? "active" : "inactive"}>
          {row.original.status}
        </Badge>
      ),
      size: 110,
      meta: {
        label: "Status",
        apiField: "status",
        filterable: true,
        sortable: true,
        filterType: "select",
        defaultFilterOperator: "eq",
      },
    },
    {
      accessorKey: "maxWidthFeet",
      header: "Max Width",
      cell: ({ row }) => formatFeetInches(row.original.maxWidthFeet),
      size: 120,
      meta: { label: "Max Width", apiField: "maxWidthFeet", sortable: true },
    },
    {
      accessorKey: "maxHeightFeet",
      header: "Max Height",
      cell: ({ row }) => formatFeetInches(row.original.maxHeightFeet),
      size: 120,
      meta: { label: "Max Height", apiField: "maxHeightFeet", sortable: true },
    },
    {
      accessorKey: "maxLengthFeet",
      header: "Max Length",
      cell: ({ row }) => formatFeetInches(row.original.maxLengthFeet),
      size: 120,
      meta: { label: "Max Length", apiField: "maxLengthFeet", sortable: true },
    },
    {
      accessorKey: "maxWeightPounds",
      header: "Max Weight",
      cell: ({ row }) => formatPounds(row.original.maxWeightPounds),
      size: 140,
      meta: { label: "Max Weight", apiField: "maxWeightPounds", sortable: true },
    },
    {
      accessorKey: "permitLeadTimeDays",
      header: "Lead Time",
      cell: ({ row }) => {
        const days = row.original.permitLeadTimeDays;
        return `${days} day${days === 1 ? "" : "s"}`;
      },
      size: 110,
      meta: { label: "Lead Time", apiField: "permitLeadTimeDays", sortable: true },
    },
    {
      accessorKey: "verifiedAt",
      header: "Verified",
      cell: ({ row }) =>
        row.original.verifiedAt ? (
          <HoverCardTimestamp timestamp={row.original.verifiedAt} />
        ) : (
          <span className="text-muted-foreground">Never</span>
        ),
      size: 150,
      meta: { label: "Verified", apiField: "verifiedAt", sortable: true },
    },
  ];
}
