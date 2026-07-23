import { AmountDisplay } from "@trenova/shared/components/accounting/amount-display";
import { EditableStatusBadge } from "@/components/editable-status-badge";
import { statusChoices } from "@/lib/choices";
import { updatePayCode, type PayCodeRow } from "@/lib/graphql/driver-settlement";
import { cn } from "@trenova/shared/lib/utils";
import { useQueryClient } from "@tanstack/react-query";
import { type ColumnDef } from "@tanstack/react-table";
import { toast } from "sonner";

export function payCodeStatusInput(row: PayCodeRow, status: "Active" | "Inactive") {
  return {
    id: row.id,
    version: row.version,
    status,
    code: row.code,
    name: row.name,
    description: row.description || undefined,
    taxable: row.taxable,
    countsTowardGuarantee: row.countsTowardGuarantee,
    glAccountId: row.glAccountId ?? undefined,
    defaultAmountMinor: row.defaultAmountMinor ?? undefined,
  };
}

function StatusCell({ row }: { row: PayCodeRow }) {
  const queryClient = useQueryClient();

  return (
    <EditableStatusBadge<"Active" | "Inactive">
      status={row.status as "Active" | "Inactive"}
      options={statusChoices}
      onStatusChange={async (status) => {
        await updatePayCode(payCodeStatusInput(row, status));
        await queryClient.invalidateQueries({ queryKey: ["pay-code-list"] });
        toast.success(
          status === "Active"
            ? "Pay code activated — it appears in dropdowns again"
            : "Pay code deactivated — it stays on historical records only",
        );
      }}
    />
  );
}

export function getColumns(): ColumnDef<PayCodeRow>[] {
  return [
    {
      accessorKey: "status",
      header: "Status",
      cell: ({ row }) => <StatusCell row={row.original} />,
      size: 120,
      meta: { apiField: "status" },
    },
    {
      accessorKey: "direction",
      header: "Direction",
      cell: ({ row }) => (
        <span
          className={cn(
            "inline-flex items-center rounded-full px-2 py-0.5 text-[11px] font-medium",
            row.original.direction === "Earning"
              ? "bg-green-50 text-green-700 dark:bg-green-950 dark:text-green-300"
              : "bg-red-50 text-red-700 dark:bg-red-950 dark:text-red-300",
          )}
        >
          {row.original.direction}
        </span>
      ),
      size: 100,
      meta: { apiField: "direction" },
    },
    {
      accessorKey: "code",
      header: "Code",
      cell: ({ row }) => <span className="font-mono text-xs font-medium">{row.original.code}</span>,
      size: 110,
      meta: { apiField: "code" },
    },
    {
      accessorKey: "name",
      header: "Name",
      cell: ({ row }) => (
        <span className="text-xs">
          {row.original.name}
          {row.original.isSystem && (
            <span className="ml-1.5 rounded bg-muted px-1 py-0.5 text-[10px] text-muted-foreground">
              System
            </span>
          )}
        </span>
      ),
      size: 200,
      meta: { apiField: "name" },
    },
    {
      id: "behavior",
      header: "Behavior",
      cell: ({ row }) => {
        if (row.original.direction !== "Earning") return null;
        return (
          <span className="text-[11px] text-muted-foreground">
            {row.original.taxable ? "Taxable" : "Reimbursement"}
            {!row.original.countsTowardGuarantee && " · excl. guarantee"}
          </span>
        );
      },
      size: 150,
    },
    {
      id: "glAccount",
      header: "GL Account",
      cell: ({ row }) =>
        row.original.glAccount ? (
          <span className="text-xs">
            <span className="font-mono">{row.original.glAccount.accountCode}</span>{" "}
            <span className="text-muted-foreground">{row.original.glAccount.name}</span>
          </span>
        ) : (
          <span className="text-[11px] text-muted-foreground">Default</span>
        ),
      size: 200,
    },
    {
      accessorKey: "defaultAmountMinor",
      header: () => <div className="text-right">Default Amount</div>,
      cell: ({ row }) =>
        row.original.defaultAmountMinor != null ? (
          <div className="text-right">
            <AmountDisplay value={row.original.defaultAmountMinor} currency="USD" />
          </div>
        ) : null,
      size: 120,
      meta: { apiField: "defaultAmountMinor" },
    },
  ];
}
