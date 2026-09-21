import type { TranslateFn } from "@trenova/shared/i18n/use-t";
import { AmountDisplay } from "@trenova/shared/components/accounting/amount-display";
import { EditableStatusBadge } from "@/components/editable-status-badge";
import { statusChoices } from "@/lib/choices";
import { updatePayCode, type PayCodeRow } from "@/lib/graphql/driver-settlement";
import { cn } from "@trenova/shared/lib/utils";
import { useQueryClient } from "@tanstack/react-query";
import type { ColumnDef } from "@trenova/shared/types/data-table";
import { selectOptionsQueryFilter } from "@/lib/select-options-cache";
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
        await queryClient.invalidateQueries(selectOptionsQueryFilter("PAY_CODE"));
        toast.success(
          status === "Active"
            ? "Pay code activated — it appears in dropdowns again"
            : "Pay code deactivated — it stays on historical records only",
        );
      }}
    />
  );
}

export function getColumns(t: TranslateFn): ColumnDef<PayCodeRow>[] {
  return [
    {
      accessorKey: "status",
      header: t("Status"),
      cell: ({ row }) => <StatusCell row={row.original} />,
      size: 120,
      meta: { apiField: "status" },
    },
    {
      accessorKey: "direction",
      header: t("Direction"),
      cell: ({ row }) => (
        <span
          className={cn(
            "inline-flex items-center rounded-full px-2 py-0.5 text-xs font-medium",
            row.original.direction === "Earning"
              ? "bg-success-subtle text-success-foreground dark:bg-success-subtle dark:text-success-foreground"
              : "bg-danger-subtle text-danger-foreground dark:bg-danger-subtle dark:text-danger-foreground",
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
      header: t("Code"),
      cell: ({ row }) => <span className="font-mono font-medium">{row.original.code}</span>,
      size: 110,
      meta: { apiField: "code" },
    },
    {
      accessorKey: "name",
      header: t("Name"),
      cell: ({ row }) => (
        <span>
          {row.original.name}
          {row.original.isSystem && (
            <span className="bg-muted text-muted-foreground ml-1.5 rounded px-1 py-0.5 text-2xs">
              {t("System")}
            </span>
          )}
        </span>
      ),
      size: 200,
      meta: { apiField: "name" },
    },
    {
      id: "behavior",
      header: t("Behavior"),
      cell: ({ row }) => {
        if (row.original.direction !== "Earning") return null;
        return (
          <span className="text-muted-foreground text-xs">
            {row.original.taxable ? t("Taxable") : t("Reimbursement")}
            {!row.original.countsTowardGuarantee && ` ${t("· excl. guarantee")}`}
          </span>
        );
      },
      size: 150,
    },
    {
      id: "glAccount",
      header: t("GL Account"),
      cell: ({ row }) =>
        row.original.glAccount ? (
          <span>
            <span className="font-mono">{row.original.glAccount.accountCode}</span>{" "}
            <span className="text-muted-foreground">{row.original.glAccount.name}</span>
          </span>
        ) : (
          <span className="text-muted-foreground text-xs">{t("Default")}</span>
        ),
      size: 200,
    },
    {
      accessorKey: "defaultAmountMinor",
      header: () => <div className="text-right">{t("Default Amount")}</div>,
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
