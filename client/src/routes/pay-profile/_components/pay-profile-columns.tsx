import { AmountDisplay } from "@/components/accounting/amount-display";
import { EditableStatusBadge } from "@/components/editable-status-badge";
import { PayeeClassificationBadge } from "@/components/status-badge";
import { updatePayProfile, type PayProfileRow } from "@/lib/graphql/driver-settlement";
import { payCalcMethodChoices, payComponentKindChoices, statusChoices } from "@/lib/choices";
import type { PayeeClassification } from "@/types/driver-pay";
import { useQueryClient } from "@tanstack/react-query";
import { type ColumnDef } from "@tanstack/react-table";
import { toast } from "sonner";

export function payProfileStatusInput(row: PayProfileRow, status: "Active" | "Inactive") {
  return {
    id: row.id,
    version: row.version,
    status,
    name: row.name,
    description: row.description || undefined,
    classification: row.classification,
    currencyCode: row.currencyCode,
    guaranteedPeriodMinimumMinor: row.guaranteedPeriodMinimumMinor,
    perDiemRatePerMile: row.perDiemRatePerMile,
    perDiemDailyCapMinor: row.perDiemDailyCapMinor,
    components: (row.components ?? []).map((component) => ({
      kind: component.kind,
      method: component.method,
      description: component.description || undefined,
      rate: component.rate,
      revenueBasis: component.revenueBasis ?? undefined,
      bands:
        component.bands && component.bands.length > 0
          ? component.bands.map((band) => ({
              minMiles: band.minMiles,
              maxMiles: band.maxMiles,
              rate: band.rate,
            }))
          : undefined,
      freeTimeMinutes: component.freeTimeMinutes,
      minAmountMinor: component.minAmountMinor ?? undefined,
      maxAmountMinor: component.maxAmountMinor ?? undefined,
      isActive: component.isActive,
    })),
  };
}

function StatusCell({ row }: { row: PayProfileRow }) {
  const queryClient = useQueryClient();

  return (
    <EditableStatusBadge<"Active" | "Inactive">
      status={row.status as "Active" | "Inactive"}
      options={statusChoices}
      onStatusChange={async (status) => {
        await updatePayProfile(payProfileStatusInput(row, status));
        await queryClient.invalidateQueries({ queryKey: ["pay-profile-list"] });
        toast.success(
          status === "Active"
            ? "Pay profile activated"
            : "Pay profile deactivated — existing assignments keep paying until reassigned",
        );
      }}
    />
  );
}

function componentSummary(row: PayProfileRow): string {
  const components = (row.components ?? []).filter((component) => component.isActive);
  if (components.length === 0) return "—";
  return components
    .slice(0, 3)
    .map((component) => {
      const kind =
        payComponentKindChoices.find((choice) => choice.value === component.kind)?.label ??
        component.kind;
      const method =
        payCalcMethodChoices.find((choice) => choice.value === component.method)?.label ??
        component.method;
      if (component.method === "PercentOfRevenue") {
        return `${kind} ${Number(component.rate)}%`;
      }
      if (component.bands && component.bands.length > 0) {
        return `${kind} (${component.bands.length} bands)`;
      }
      return `${kind} $${Number(component.rate).toFixed(2)} ${method.toLowerCase()}`;
    })
    .join(" · ")
    .concat(components.length > 3 ? ` +${components.length - 3} more` : "");
}

export function getColumns(): ColumnDef<PayProfileRow>[] {
  return [
    {
      accessorKey: "status",
      header: "Status",
      cell: ({ row }) => <StatusCell row={row.original} />,
      size: 120,
      meta: { apiField: "status" },
    },
    {
      accessorKey: "name",
      header: "Name",
      cell: ({ row }) => <span className="text-xs font-medium">{row.original.name}</span>,
      size: 180,
      meta: { apiField: "name" },
    },
    {
      accessorKey: "classification",
      header: "Type",
      cell: ({ row }) => (
        <PayeeClassificationBadge
          classification={row.original.classification as PayeeClassification}
        />
      ),
      size: 130,
      meta: { apiField: "classification" },
    },
    {
      id: "components",
      header: "Pay Components",
      cell: ({ row }) => (
        <span className="text-xs text-muted-foreground">{componentSummary(row.original)}</span>
      ),
      size: 340,
    },
    {
      accessorKey: "guaranteedPeriodMinimumMinor",
      header: () => <div className="text-right">Guarantee</div>,
      cell: ({ row }) =>
        row.original.guaranteedPeriodMinimumMinor > 0 ? (
          <div className="text-right">
            <AmountDisplay
              value={row.original.guaranteedPeriodMinimumMinor}
              currency={row.original.currencyCode}
            />
          </div>
        ) : (
          <div className="text-right text-xs text-muted-foreground">—</div>
        ),
      size: 110,
      meta: { apiField: "guaranteedPeriodMinimumMinor" },
    },
    {
      accessorKey: "activeAssignmentCount",
      header: () => <div className="text-right">Drivers</div>,
      cell: ({ row }) => (
        <div className="text-right text-xs tabular-nums">{row.original.activeAssignmentCount}</div>
      ),
      size: 80,
    },
  ];
}
