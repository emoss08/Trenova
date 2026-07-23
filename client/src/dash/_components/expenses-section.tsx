import { AmountDisplay } from "@/components/accounting/amount-display";
import { Button } from "@/components/ui/button";
import {
  Drawer,
  DrawerContent,
  DrawerDescription,
  DrawerFooter,
  DrawerHeader,
  DrawerTitle,
} from "@/components/ui/drawer";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Skeleton } from "@/components/ui/skeleton";
import { Textarea } from "@/components/ui/textarea";
import { dateToUnixTimestamp, formatRange, generateDateOnly } from "@/lib/date";
import { cancelMyExpense, fetchMyExpenses, submitMyExpense } from "@/lib/graphql/driver-portal";
import { uploadMyExpenseReceipt } from "@/lib/portal";
import { cn } from "@/lib/utils";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { CameraIcon, PlusIcon, ReceiptIcon, XIcon } from "lucide-react";
import { useRef, useState } from "react";
import { toast } from "sonner";
import { ExpenseStatusBadge } from "./portal-badges";
import { useDashFeatures } from "./use-dash-features";
import { stopPlace, originStop, destinationStop, useMyLoads } from "./use-loads";

export function ExpensesSection() {
  const queryClient = useQueryClient();
  const [submitOpen, setSubmitOpen] = useState(false);
  const expenses = useQuery({ queryKey: ["dash-expenses"], queryFn: fetchMyExpenses });

  const cancel = useMutation({
    mutationFn: (id: string) => cancelMyExpense(id),
    onSuccess: async () => {
      toast.success("Expense cancelled.");
      await queryClient.invalidateQueries({ queryKey: ["dash-expenses"] });
    },
    onError: (error: Error) => toast.error(error.message || "We couldn't cancel that expense."),
  });

  return (
    <section className="flex flex-col gap-3">
      <div className="flex items-center justify-between gap-2">
        <h2 className="text-sm font-semibold">Expenses</h2>
        <Button variant="outline" size="sm" className="h-8" onClick={() => setSubmitOpen(true)}>
          <PlusIcon className="size-3.5" />
          Submit
        </Button>
      </div>

      {expenses.isPending ? (
        <Skeleton className="h-20 w-full rounded-2xl" />
      ) : expenses.data && expenses.data.length > 0 ? (
        <ul className="divide-y divide-border rounded-2xl border border-border bg-card">
          {expenses.data.map((expense) => (
            <li key={expense.id} className="px-4 py-3">
              <div className="flex items-center justify-between gap-2">
                <p className="min-w-0 truncate text-sm font-medium">{expense.description}</p>
                <span className="shrink-0 text-sm font-semibold tabular-nums">
                  <AmountDisplay value={expense.amountMinor} currency={expense.currencyCode} />
                </span>
              </div>
              <div className="mt-1 flex items-center justify-between gap-2">
                <p className="text-xs text-muted-foreground">
                  {formatRange(expense.incurredDate, expense.incurredDate)}
                  {expense.receiptDocumentId ? " · Receipt attached" : ""}
                </p>
                <ExpenseStatusBadge status={expense.status} />
              </div>
              {expense.status === "Rejected" && expense.reviewNote ? (
                <p className="mt-1 text-xs text-muted-foreground">
                  <span className="font-medium text-foreground">Carrier note:</span>{" "}
                  {expense.reviewNote}
                </p>
              ) : null}
              {expense.status === "Pending" ? (
                <Button
                  variant="ghost"
                  size="sm"
                  className="mt-1 h-7 px-2 text-xs text-muted-foreground"
                  disabled={cancel.isPending}
                  onClick={() => cancel.mutate(expense.id)}
                >
                  Cancel
                </Button>
              ) : null}
            </li>
          ))}
        </ul>
      ) : (
        <div className="flex flex-col items-center gap-2 rounded-2xl border border-dashed border-border p-8 text-center">
          <ReceiptIcon className="size-6 text-muted-foreground" />
          <p className="text-sm text-muted-foreground">
            Paid a lumper, tolls, or a scale out of pocket? Submit it with a photo of the receipt
            and get it back on your next settlement.
          </p>
        </div>
      )}

      <ExpenseSubmitDrawer open={submitOpen} onOpenChange={setSubmitOpen} />
    </section>
  );
}

type ExpenseSubmitDrawerProps = {
  open: boolean;
  onOpenChange: (open: boolean) => void;
};

function ExpenseSubmitDrawer({ open, onOpenChange }: ExpenseSubmitDrawerProps) {
  const queryClient = useQueryClient();
  const features = useDashFeatures();
  const fileInputRef = useRef<HTMLInputElement>(null);
  const loads = useMyLoads("Active");
  const [amount, setAmount] = useState("");
  const [description, setDescription] = useState("");
  const [incurredDate, setIncurredDate] = useState("");
  const [shipmentId, setShipmentId] = useState<string | null>(null);
  const [receipt, setReceipt] = useState<File | null>(null);

  const reset = () => {
    setAmount("");
    setDescription("");
    setIncurredDate("");
    setShipmentId(null);
    setReceipt(null);
  };

  const submit = useMutation({
    mutationFn: async () => {
      const amountMinor = Math.round(Number.parseFloat(amount) * 100);
      if (!Number.isFinite(amountMinor) || amountMinor <= 0) {
        throw new Error("Enter the amount you paid.");
      }
      const incurred = incurredDate ? generateDateOnly(incurredDate) : null;
      const expense = await submitMyExpense({
        amountMinor,
        description: description.trim(),
        shipmentId: shipmentId ?? undefined,
        incurredDate: incurred ? dateToUnixTimestamp(incurred) : undefined,
      });
      if (receipt) {
        await uploadMyExpenseReceipt(expense.id, receipt);
      }
      return expense;
    },
    onSuccess: async () => {
      toast.success("Expense sent — payroll will review it.");
      await queryClient.invalidateQueries({ queryKey: ["dash-expenses"] });
      reset();
      onOpenChange(false);
    },
    onError: (error: Error) => toast.error(error.message || "We couldn't submit your expense."),
  });

  const handleFileChange = (event: React.ChangeEvent<HTMLInputElement>) => {
    const file = event.target.files?.[0];
    event.target.value = "";
    if (file) {
      setReceipt(file);
    }
  };

  const canSubmit =
    amount.trim().length > 0 &&
    description.trim().length > 0 &&
    (!features.requireExpenseReceipt || receipt != null);

  return (
    <Drawer
      open={open}
      onOpenChange={(next) => {
        if (!next) reset();
        onOpenChange(next);
      }}
    >
      <DrawerContent>
        <DrawerHeader>
          <DrawerTitle>Submit an expense</DrawerTitle>
          <DrawerDescription>
            Approved expenses land as a reimbursement on your next settlement.
          </DrawerDescription>
        </DrawerHeader>

        <div className="flex max-h-[55vh] flex-col gap-4 overflow-y-auto px-4">
          <div className="grid grid-cols-2 gap-3">
            <div className="flex flex-col gap-1.5">
              <Label className="text-xs text-muted-foreground">Amount (USD)</Label>
              <Input
                inputMode="decimal"
                placeholder="0.00"
                value={amount}
                onChange={(event) => setAmount(event.target.value)}
              />
            </div>
            <div className="flex flex-col gap-1.5">
              <Label className="text-xs text-muted-foreground">Date paid</Label>
              <Input
                type="date"
                value={incurredDate}
                onChange={(event) => setIncurredDate(event.target.value)}
              />
            </div>
          </div>

          <Textarea
            value={description}
            onChange={(event) => setDescription(event.target.value)}
            placeholder="What was it for? Lumper at the receiver, tolls, scale ticket..."
            rows={3}
            maxLength={255}
          />

          {loads.data && loads.data.length > 0 ? (
            <div className="flex flex-col gap-1.5">
              <Label className="text-xs text-muted-foreground">Load (optional)</Label>
              <div className="flex flex-wrap gap-1.5">
                {loads.data.map((load) => (
                  <button
                    key={load.assignmentId}
                    type="button"
                    onClick={() =>
                      setShipmentId((current) =>
                        current === load.shipmentId ? null : load.shipmentId,
                      )
                    }
                    className={cn(
                      "rounded-full border border-border px-2.5 py-1 text-xs font-medium text-muted-foreground transition-colors",
                      shipmentId === load.shipmentId &&
                        "border-primary bg-primary text-primary-foreground",
                    )}
                  >
                    {load.proNumber ||
                      `${stopPlace(originStop(load))} → ${stopPlace(destinationStop(load))}`}
                  </button>
                ))}
              </div>
            </div>
          ) : null}

          <input
            ref={fileInputRef}
            type="file"
            accept="image/*,application/pdf"
            capture="environment"
            className="hidden"
            onChange={handleFileChange}
          />
          {receipt ? (
            <div className="flex items-center justify-between gap-2 rounded-xl border border-border bg-muted/40 px-3 py-2">
              <p className="min-w-0 truncate text-sm font-medium">{receipt.name}</p>
              <button
                type="button"
                aria-label="Remove receipt"
                className="text-muted-foreground hover:text-foreground"
                onClick={() => setReceipt(null)}
              >
                <XIcon className="size-4" />
              </button>
            </div>
          ) : (
            <Button
              variant="outline"
              className="h-10"
              onClick={() => fileInputRef.current?.click()}
            >
              <CameraIcon className="size-4" />
              {features.requireExpenseReceipt
                ? "Add receipt photo (required)"
                : "Add receipt photo"}
            </Button>
          )}
        </div>

        <DrawerFooter>
          <Button
            className="h-11"
            disabled={!canSubmit || submit.isPending}
            onClick={() => submit.mutate()}
          >
            {submit.isPending ? "Submitting..." : "Submit expense"}
          </Button>
        </DrawerFooter>
      </DrawerContent>
    </Drawer>
  );
}
