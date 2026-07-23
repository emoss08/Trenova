import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import { useState } from "react";

export function BulkMarkPaidDialog({
  open,
  count,
  pending,
  onOpenChange,
  onConfirm,
}: {
  open: boolean;
  count: number;
  pending: boolean;
  onOpenChange: (open: boolean) => void;
  onConfirm: (paymentMethod: string, paymentReference: string) => void;
}) {
  const [paymentMethod, setPaymentMethod] = useState("ACH");
  const [paymentReference, setPaymentReference] = useState("");

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
              {["ACH", "Check", "InstantPay", "Other"].map((method) => (
                <Button
                  key={method}
                  size="sm"
                  variant={paymentMethod === method ? "default" : "outline"}
                  onClick={() => setPaymentMethod(method)}
                >
                  {method === "InstantPay" ? "Instant Pay" : method}
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
