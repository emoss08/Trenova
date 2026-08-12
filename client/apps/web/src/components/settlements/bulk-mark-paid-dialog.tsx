import { Button } from "@trenova/shared/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@trenova/shared/components/ui/dialog";
import { Input } from "@trenova/shared/components/ui/input";
import { useEffect, useState } from "react";

export type BulkMarkPaidMethod = {
  value: string;
  label: string;
};

/** Driver payroll disbursement methods — the dialog's historical default. */
export const DRIVER_MARK_PAID_METHODS: ReadonlyArray<BulkMarkPaidMethod> = [
  { value: "ACH", label: "ACH" },
  { value: "Check", label: "Check" },
  { value: "InstantPay", label: "Instant Pay" },
  { value: "Other", label: "Other" },
];

/** Carrier AP disbursement methods — carriers are paid by check or manual ACH, never payroll rails. */
export const CARRIER_MARK_PAID_METHODS: ReadonlyArray<BulkMarkPaidMethod> = [
  { value: "Check", label: "Check" },
  { value: "ACHManual", label: "ACH (Manual)" },
  { value: "Other", label: "Other" },
];

export function BulkMarkPaidDialog({
  open,
  count,
  pending,
  methods = DRIVER_MARK_PAID_METHODS,
  onOpenChange,
  onConfirm,
}: {
  open: boolean;
  count: number;
  pending: boolean;
  methods?: ReadonlyArray<BulkMarkPaidMethod>;
  onOpenChange: (open: boolean) => void;
  onConfirm: (paymentMethod: string, paymentReference: string) => void;
}) {
  const [paymentMethod, setPaymentMethod] = useState(methods[0]?.value ?? "");
  const [paymentReference, setPaymentReference] = useState("");

  // If the offered methods change (or no longer include the current pick),
  // fall back to the first offered method rather than submitting a stale one.
  useEffect(() => {
    if (!methods.some((method) => method.value === paymentMethod)) {
      setPaymentMethod(methods[0]?.value ?? "");
    }
  }, [methods, paymentMethod]);

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>
            Mark {count} settlement{count === 1 ? "" : "s"} paid
          </DialogTitle>
          <DialogDescription>
            Records the disbursement on every selected posted settlement. Use a batch reference
            (e.g. the ACH file ID) so the whole run reconciles against one bank entry.
          </DialogDescription>
        </DialogHeader>
        <div className="flex flex-col gap-3">
          <div>
            <p className="mb-1 text-xs font-medium">Payment method</p>
            <div className="flex gap-2">
              {methods.map((method) => (
                <Button
                  key={method.value}
                  size="sm"
                  variant={paymentMethod === method.value ? "default" : "outline"}
                  onClick={() => setPaymentMethod(method.value)}
                >
                  {method.label}
                </Button>
              ))}
            </div>
          </div>
          <div>
            <p className="mb-1 text-xs font-medium">Batch reference</p>
            <Input
              value={paymentReference}
              onChange={(event) => setPaymentReference(event.target.value)}
              placeholder="ACH file / batch ID (optional)"
            />
          </div>
        </div>
        <DialogFooter>
          <Button variant="outline" onClick={() => onOpenChange(false)}>
            Cancel
          </Button>
          <Button disabled={pending} onClick={() => onConfirm(paymentMethod, paymentReference)}>
            Mark Paid
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
