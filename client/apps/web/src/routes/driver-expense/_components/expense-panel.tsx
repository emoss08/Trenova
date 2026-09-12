import { useT } from "@trenova/shared/i18n/use-t";
import { AmountDisplay } from "@trenova/shared/components/accounting/amount-display";
import { DataTablePanelContainer } from "@/components/data-table/data-table-panel";
import { Button } from "@trenova/shared/components/ui/button";
import { Label } from "@trenova/shared/components/ui/label";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { Textarea } from "@trenova/shared/components/ui/textarea";
import { formatUnixDate } from "@trenova/shared/lib/date";
import {
  fetchDriverExpenseDetail,
  reviewDriverExpense,
  type DriverExpenseRow,
} from "@trenova/shared/lib/graphql/driver-portal";
import { apiService } from "@/services/api";
import type { DataTablePanelProps } from "@trenova/shared/types/data-table";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { ExternalLinkIcon } from "lucide-react";
import { useState } from "react";
import { toast } from "sonner";
import { DriverExpenseStatusBadge } from "./expense-columns";

export function ExpensePanel({
  open,
  onOpenChange,
  mode,
  row,
}: DataTablePanelProps<DriverExpenseRow>) {
  const t = useT();

  if (mode !== "edit" || !row) {
    return null;
  }

  return (
    <DataTablePanelContainer
      open={open}
      onOpenChange={onOpenChange}
      title={t("Driver Expense")}
      description={row.worker ? `${row.worker.firstName} ${row.worker.lastName}`.trim() : undefined}
      size="lg"
    >
      <ExpenseDetail expenseId={row.id} onClose={() => onOpenChange(false)} />
    </DataTablePanelContainer>
  );
}

function ExpenseDetail({ expenseId, onClose }: { expenseId: string; onClose: () => void }) {
  const t = useT();

  const queryClient = useQueryClient();
  const detail = useQuery({
    queryKey: ["driver-expense-detail", expenseId],
    queryFn: ({ signal }) => fetchDriverExpenseDetail(expenseId, { signal }),
  });

  const invalidate = async () => {
    await queryClient.invalidateQueries({ queryKey: ["driver-expense-detail", expenseId] });
    await queryClient.invalidateQueries({ queryKey: ["driver-expense-list"] });
    await queryClient.invalidateQueries({ queryKey: ["pending-driver-expense-count"] });
  };

  const handleViewReceipt = async (documentId: string) => {
    const url = await apiService.documentService.getViewUrl(documentId);
    window.open(url, "_blank", "noopener,noreferrer");
  };

  if (detail.isPending) {
    return (
      <div className="flex flex-col gap-3 p-4">
        <Skeleton className="h-24 w-full rounded-lg" />
        <Skeleton className="h-40 w-full rounded-lg" />
      </div>
    );
  }

  const expense = detail.data;
  if (!expense) {
    return (
      <p className="text-muted-foreground p-6 text-center text-sm">
        {t("This expense could not be loaded.")}
      </p>
    );
  }

  const isTerminal = expense.status !== "Pending";

  return (
    <div className="flex flex-col gap-4">
      <div className="border-border bg-muted/40 rounded-lg border p-3">
        <div className="flex items-center justify-between gap-2">
          <p className="text-sm font-semibold tabular-nums">
            <AmountDisplay value={expense.amountMinor} currency={expense.currencyCode} />
          </p>
          <DriverExpenseStatusBadge status={expense.status} />
        </div>
        <p className="text-muted-foreground mt-1 text-xs">
          {t(
            "Incurred {0} · Submitted {1}{2}",
            formatUnixDate(expense.incurredDate),
            formatUnixDate(expense.createdAt),
            expense.worker
              ? ` ${t("by {0}", `${expense.worker.firstName} ${expense.worker.lastName}`.trim())}`
              : "",
          )}
        </p>
        <p className="mt-3 text-sm whitespace-pre-wrap">{t(expense.description)}</p>
        {expense.payCode ? (
          <p className="text-muted-foreground mt-2 text-xs">
            {t("Pay code:")} <span className="font-mono">{expense.payCode.code}</span>
            {expense.payCode.description ? ` — ${expense.payCode.description}` : ""}
          </p>
        ) : null}
      </div>

      {expense.receiptDocumentId ? (
        <Button
          variant="outline"
          onClick={() => void handleViewReceipt(expense.receiptDocumentId!)}
        >
          <ExternalLinkIcon className="size-4" />
          {t("View receipt")}
        </Button>
      ) : (
        <p className="border-border text-muted-foreground rounded-lg border border-dashed p-3 text-center text-xs">
          {t("No receipt attached — consider asking the driver for one before approving.")}
        </p>
      )}

      {isTerminal ? (
        <div className="border-border rounded-lg border p-3">
          <p className="text-muted-foreground text-xs font-medium uppercase">{t("Review")}</p>
          <p className="mt-1 text-sm whitespace-pre-wrap">{expense.reviewNote || "—"}</p>
          <p className="text-muted-foreground mt-2 text-xs">
            {formatUnixDate(expense.reviewedAt ?? 0) || "—"}
            {expense.reviewedBy ? ` ${t("by {0}", expense.reviewedBy.name)}` : ""}
            {expense.settlementLineId ? ` ${t("· reimbursement applied to open settlement")}` : ""}
          </p>
        </div>
      ) : (
        <ReviewForm expenseId={expenseId} onDone={invalidate} onClose={onClose} />
      )}
    </div>
  );
}

function ReviewForm({
  expenseId,
  onDone,
  onClose,
}: {
  expenseId: string;
  onDone: () => Promise<void>;
  onClose: () => void;
}) {
  const t = useT();

  const [approve, setApprove] = useState(true);
  const [note, setNote] = useState("");

  const valid = approve || note.trim() !== "";

  const mutation = useMutation({
    mutationFn: () =>
      reviewDriverExpense({
        expenseId,
        approve,
        note: note.trim() || undefined,
      }),
    onSuccess: async () => {
      toast.success(approve ? "Expense approved — reimbursement applied" : "Expense rejected");
      await onDone();
      onClose();
    },
    onError: (error: Error) => toast.error(error.message || "Failed to review expense"),
  });

  return (
    <div className="border-border flex flex-col gap-4 rounded-lg border p-3">
      <p className="text-sm font-semibold">{t("Review this expense")}</p>

      <div className="grid grid-cols-2 gap-2">
        <Button
          type="button"
          variant={approve ? "default" : "outline"}
          onClick={() => setApprove(true)}
        >
          {t("Approve & reimburse")}
        </Button>
        <Button
          type="button"
          variant={approve ? "outline" : "default"}
          onClick={() => setApprove(false)}
        >
          {t("Reject")}
        </Button>
      </div>

      <div className="flex flex-col gap-1.5">
        <Label htmlFor="expense-review-note">
          {approve ? t("Note (optional)") : t("Rejection reason")}
        </Label>
        <Textarea
          id="expense-review-note"
          value={note}
          onChange={(event) => setNote(event.target.value)}
          placeholder={
            approve
              ? "Anything worth recording about this reimbursement."
              : "Explain why — the driver sees this in Dash."
          }
          rows={3}
        />
        <p className="text-muted-foreground text-[11px]">
          {approve
            ? t(
                "Approval immediately adds a reimbursement line to the driver's open settlement (an off-cycle draft is created if none exists).",
              )
            : t("Required — shown to the driver verbatim.")}
        </p>
      </div>

      <Button onClick={() => mutation.mutate()} disabled={!valid || mutation.isPending}>
        {mutation.isPending ? t("Saving...") : approve ? t("Approve expense") : t("Reject expense")}
      </Button>
    </div>
  );
}
