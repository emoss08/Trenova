import { WorkerAutocompleteField } from "@/components/autocomplete-fields";
import { AmountDisplay } from "@trenova/shared/components/accounting/amount-display";
import { DataTablePanelContainer } from "@/components/data-table/data-table-panel";
import { FormCreatePanel } from "@/components/form-create-panel";
import { NumberField } from "@/components/fields/number-field";
import { EscrowAccountStatusBadge } from "@trenova/shared/components/status-badge";
import { Button } from "@trenova/shared/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@trenova/shared/components/ui/dialog";
import { FormControl, FormGroup } from "@trenova/shared/components/ui/form";
import { Input } from "@trenova/shared/components/ui/input";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import {
  adjustEscrowAccount,
  closeEscrowAccount,
  fetchEscrowAccountDetail,
  openEscrowAccount,
  type EscrowAccountRow,
} from "@/lib/graphql/driver-settlement";
import type { DataTablePanelProps } from "@trenova/shared/types/data-table";
import {
  openEscrowAccountFormSchema,
  type EscrowAccountStatus,
  type OpenEscrowAccountFormValues,
} from "@trenova/shared/types/driver-pay";
import { zodResolver } from "@hookform/resolvers/zod";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { useState } from "react";
import { useForm, type Resolver } from "react-hook-form";
import { toast } from "sonner";

const transactionTypeLabels: Record<string, string> = {
  Contribution: "Contribution",
  InterestAccrual: "Interest",
  Application: "Applied",
  Refund: "Refund",
  Adjustment: "Adjustment",
};

function formatDate(unix?: number | null): string {
  if (!unix) return "—";
  return new Date(unix * 1000).toLocaleDateString("en-US", {
    month: "short",
    day: "numeric",
    year: "numeric",
  });
}

export function EscrowPanel({
  open,
  onOpenChange,
  mode,
  row,
}: DataTablePanelProps<EscrowAccountRow>) {
  if (mode === "edit" && row) {
    return (
      <DataTablePanelContainer
        open={open}
        onOpenChange={onOpenChange}
        title={
          row.worker
            ? `Escrow — ${row.worker.firstName} ${row.worker.lastName}`.trim()
            : "Escrow Account"
        }
        size="lg"
      >
        <EscrowDetail accountId={row.id} onClose={() => onOpenChange(false)} />
      </DataTablePanelContainer>
    );
  }

  return <OpenEscrowPanel open={open} onOpenChange={onOpenChange} />;
}

function OpenEscrowPanel({
  open,
  onOpenChange,
}: {
  open: boolean;
  onOpenChange: (open: boolean) => void;
}) {
  const form = useForm<OpenEscrowAccountFormValues>({
    resolver: zodResolver(openEscrowAccountFormSchema) as Resolver<OpenEscrowAccountFormValues>,
    defaultValues: { workerId: "", targetAmount: 0, annualInterestRate: null, openedDate: null },
  });
  const { control } = form;

  return (
    <FormCreatePanel<OpenEscrowAccountFormValues, EscrowAccountRow>
      open={open}
      onOpenChange={onOpenChange}
      title="Escrow Account"
      description="One active escrow account per driver; contributions flow in from settlements via a recurring deduction."
      queryKey="escrow-account-list"
      form={form}
      formComponent={
        <FormGroup cols={2}>
          <FormControl className="col-span-2">
            <WorkerAutocompleteField
              control={control}
              name="workerId"
              label="Driver"
              placeholder="Select owner-operator"
              ownerOperatorsOnly
              rules={{ required: true }}
              description="Only owner-operators are listed — contractors and drivers on an owner-operator pay profile."
            />
          </FormControl>
          <FormControl>
            <NumberField
              control={control}
              name="targetAmount"
              label="Funding Target"
              decimalScale={2}
              fixedDecimalScale
              sideText="USD"
              description="Contributions stop automatically once the balance reaches this target."
            />
          </FormControl>
          <FormControl>
            <NumberField
              control={control}
              name="annualInterestRate"
              label="Annual Interest Rate"
              decimalScale={2}
              fixedDecimalScale
              sideText="%"
              description="Defaults to your settlement control rate. Interest accrues at least quarterly."
            />
          </FormControl>
        </FormGroup>
      }
      mutationFn={async (values) => {
        await openEscrowAccount({
          workerId: values.workerId,
          targetAmountMinor: Math.round(values.targetAmount * 100),
          annualInterestRate:
            values.annualInterestRate != null ? String(values.annualInterestRate) : undefined,
          openedDate: values.openedDate ?? undefined,
        });
        return values;
      }}
    />
  );
}

function EscrowDetail({ accountId, onClose }: { accountId: string; onClose: () => void }) {
  const queryClient = useQueryClient();
  const [adjustOpen, setAdjustOpen] = useState(false);
  const [closeOpen, setCloseOpen] = useState(false);
  const [adjustAmount, setAdjustAmount] = useState("");
  const [adjustDescription, setAdjustDescription] = useState("");

  const { data: account, isLoading } = useQuery({
    queryKey: ["escrow-account-detail", accountId],
    queryFn: () => fetchEscrowAccountDetail(accountId),
  });

  const invalidate = () => {
    void queryClient.invalidateQueries({ queryKey: ["escrow-account-detail", accountId] });
    void queryClient.invalidateQueries({ queryKey: ["escrow-account-list"] });
  };

  const adjustMutation = useMutation({
    mutationFn: () =>
      adjustEscrowAccount({
        accountId,
        amountMinor: Math.round(Number(adjustAmount) * 100),
        description: adjustDescription.trim(),
      }),
    onSuccess: () => {
      toast.success("Escrow adjustment recorded");
      setAdjustOpen(false);
      setAdjustAmount("");
      setAdjustDescription("");
      invalidate();
    },
    onError: (error: Error) => toast.error(error.message || "Adjustment failed"),
  });

  const closeMutation = useMutation({
    mutationFn: () => closeEscrowAccount(accountId),
    onSuccess: () => {
      toast.success("Escrow account closed; remaining balance refunded");
      setCloseOpen(false);
      invalidate();
      onClose();
    },
    onError: (error: Error) => toast.error(error.message || "Close failed"),
  });

  if (isLoading || !account) {
    return (
      <div className="flex flex-col gap-3 p-4">
        <Skeleton className="h-16 w-full" />
        <Skeleton className="h-48 w-full" />
      </div>
    );
  }

  const adjustValid =
    adjustDescription.trim() !== "" &&
    adjustAmount !== "" &&
    !Number.isNaN(Number(adjustAmount)) &&
    Number(adjustAmount) !== 0;

  return (
    <div className="flex h-full flex-col gap-4">
      <div className="flex flex-wrap items-center gap-2">
        <EscrowAccountStatusBadge status={account.status as EscrowAccountStatus} />
        <span className="text-xs text-muted-foreground">
          Opened {formatDate(account.openedDate)}
          {account.closedDate ? ` · Closed ${formatDate(account.closedDate)}` : ""}
        </span>
        {account.status === "Active" && (
          <div className="ml-auto flex gap-2">
            <Button size="sm" variant="outline" onClick={() => setAdjustOpen(true)}>
              Record Adjustment
            </Button>
            <Button
              size="sm"
              variant="ghost"
              className="text-red-600 dark:text-red-400"
              onClick={() => setCloseOpen(true)}
            >
              Close Account
            </Button>
          </div>
        )}
      </div>

      <div className="grid grid-cols-3 gap-2">
        <div className="rounded-lg border bg-muted/30 p-3">
          <p className="text-[11px] font-medium text-muted-foreground uppercase">Balance</p>
          <p className="mt-1 text-sm font-semibold">
            <AmountDisplay value={account.balanceMinor} currency={account.currencyCode} />
          </p>
        </div>
        <div className="rounded-lg border bg-muted/30 p-3">
          <p className="text-[11px] font-medium text-muted-foreground uppercase">Target</p>
          <p className="mt-1 text-sm font-semibold">
            {account.targetAmountMinor > 0 ? (
              <AmountDisplay value={account.targetAmountMinor} currency={account.currencyCode} />
            ) : (
              "—"
            )}
          </p>
        </div>
        <div className="rounded-lg border bg-muted/30 p-3">
          <p className="text-[11px] font-medium text-muted-foreground uppercase">Interest Rate</p>
          <p className="mt-1 text-sm font-semibold tabular-nums">
            {Number(account.annualInterestRate) > 0
              ? `${Number(account.annualInterestRate).toFixed(2)}% / yr`
              : "—"}
          </p>
        </div>
      </div>

      <div>
        <h4 className="mb-2 text-xs font-semibold tracking-wide text-muted-foreground uppercase">
          Transaction Ledger
        </h4>
        <div className="overflow-hidden rounded-lg border">
          <table className="w-full text-xs">
            <thead className="bg-muted/50 text-left">
              <tr>
                <th className="px-3 py-2 font-medium">Date</th>
                <th className="px-3 py-2 font-medium">Type</th>
                <th className="px-3 py-2 font-medium">Description</th>
                <th className="px-3 py-2 text-right font-medium">Amount</th>
                <th className="px-3 py-2 text-right font-medium">Balance</th>
              </tr>
            </thead>
            <tbody>
              {(account.transactions ?? []).length === 0 && (
                <tr>
                  <td colSpan={5} className="px-3 py-4 text-center text-muted-foreground">
                    No transactions yet
                  </td>
                </tr>
              )}
              {(account.transactions ?? []).map((tx) => (
                <tr key={tx.id} className="border-t">
                  <td className="px-3 py-2">{formatDate(tx.occurredDate)}</td>
                  <td className="px-3 py-2">{transactionTypeLabels[tx.type] ?? tx.type}</td>
                  <td className="px-3 py-2 text-muted-foreground">{tx.description || "—"}</td>
                  <td className="px-3 py-2 text-right">
                    <AmountDisplay
                      value={tx.amountMinor}
                      variant="auto"
                      currency={account.currencyCode}
                    />
                  </td>
                  <td className="px-3 py-2 text-right tabular-nums">
                    <AmountDisplay value={tx.balanceAfterMinor} currency={account.currencyCode} />
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
        <p className="mt-2 text-[11px] text-muted-foreground">
          This ledger satisfies the transaction-level accounting owed to lessors under 49 CFR
          376.12(k); interest accrues at least quarterly.
        </p>
      </div>

      <Dialog open={adjustOpen} onOpenChange={setAdjustOpen}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Record escrow adjustment</DialogTitle>
            <DialogDescription>
              Positive amounts add to the balance; negative amounts apply funds (e.g. a repair paid
              from escrow).
            </DialogDescription>
          </DialogHeader>
          <div className="flex flex-col gap-3">
            <div>
              <Input
                value={adjustAmount}
                onChange={(e) => setAdjustAmount(e.target.value)}
                placeholder="Amount (e.g. 250.00 or -400.00)"
                inputMode="decimal"
              />
              <p className="mt-1 text-[11px] text-muted-foreground">
                Dollars, not cents; positive deposits into escrow, negative applies funds out.
              </p>
            </div>
            <div>
              <Input
                value={adjustDescription}
                onChange={(e) => setAdjustDescription(e.target.value)}
                placeholder="Description (required)"
              />
              <p className="mt-1 text-[11px] text-muted-foreground">
                Recorded permanently on the ledger — 49 CFR 376.12(k) requires every escrow
                transaction to be accounted for.
              </p>
            </div>
          </div>
          <DialogFooter>
            <Button variant="outline" onClick={() => setAdjustOpen(false)}>
              Cancel
            </Button>
            <Button
              disabled={!adjustValid || adjustMutation.isPending}
              onClick={() => adjustMutation.mutate()}
            >
              Record
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      <Dialog open={closeOpen} onOpenChange={setCloseOpen}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Close escrow account</DialogTitle>
            <DialogDescription>
              The remaining balance of{" "}
              <AmountDisplay value={account.balanceMinor} currency={account.currencyCode} /> will be
              refunded to the driver as a ledger entry. This cannot be undone.
            </DialogDescription>
          </DialogHeader>
          <DialogFooter>
            <Button variant="outline" onClick={() => setCloseOpen(false)}>
              Cancel
            </Button>
            <Button
              variant="destructive"
              disabled={closeMutation.isPending}
              onClick={() => closeMutation.mutate()}
            >
              Close Account
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  );
}
