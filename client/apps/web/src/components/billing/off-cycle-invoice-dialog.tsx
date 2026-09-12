import { Button } from "@trenova/shared/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@trenova/shared/components/ui/dialog";
import { Textarea } from "@trenova/shared/components/ui/textarea";
import { TriangleAlertIcon } from "lucide-react";
import { useState } from "react";

/** The field the API keys an off-cycle refusal on. */
export const OFF_CYCLE_FIELD = "offCycleReason";

/**
 * Reads an API error and reports whether it is the cadence guard asking for a
 * reason, rather than something that actually went wrong.
 *
 * It matches on the field name rather than the message, because the message is
 * written for a person and names the customer and their cycle — which is exactly
 * what we want to show them, and exactly the wrong thing to match on.
 *
 * Both the REST and the GraphQL error classes expose getFieldErrors, so this
 * works whichever path the mutation took without knowing which one it was.
 */
export function offCycleWarningFrom(error: unknown): string | null {
  if (!error || typeof error !== "object") return null;

  const getFieldErrors = (error as { getFieldErrors?: unknown }).getFieldErrors;
  if (typeof getFieldErrors !== "function") return null;

  const fieldErrors = (
    getFieldErrors as (field?: string) => { field: string; message: string }[]
  ).call(error, OFF_CYCLE_FIELD);

  const message = fieldErrors[0]?.message;
  if (!message) return null;

  return message;
}

/**
 * Asks a biller to justify taking freight off a customer's statement.
 *
 * It is a prompt, not a barrier: the reason is what turns an accident into a
 * decision, and it is stamped on the invoice so the next person to wonder why a
 * statement customer has a one-off invoice can read the answer there.
 */
export function OffCycleInvoiceDialog({
  open,
  warning,
  pending,
  onOpenChange,
  onConfirm,
}: {
  open: boolean;
  warning: string | null;
  pending: boolean;
  onOpenChange: (next: boolean) => void;
  onConfirm: (reason: string) => void;
}) {
  const [reason, setReason] = useState("");

  return (
    <Dialog
      open={open}
      onOpenChange={(next) => {
        if (!next) setReason("");
        onOpenChange(next);
      }}
    >
      <DialogContent className="sm:max-w-md">
        <DialogHeader>
          <DialogTitle>Invoice this outside the statement?</DialogTitle>
          <DialogDescription>{warning}</DialogDescription>
        </DialogHeader>

        <div className="flex flex-col gap-2">
          <div className="flex items-start gap-2 rounded-lg border border-amber-300 bg-amber-50/60 p-3 dark:border-amber-900 dark:bg-amber-950/30">
            <TriangleAlertIcon className="mt-0.5 size-3.5 shrink-0 text-amber-600 dark:text-amber-400" />
            <p className="text-xs text-amber-800 dark:text-amber-200">
              This freight will not appear on the customer&apos;s next statement. Their cycle does
              not change — everything else still bills on schedule.
            </p>
          </div>
          <Textarea
            value={reason}
            onChange={(event) => setReason(event.target.value)}
            placeholder="Why does this bill on its own? (e.g. billing to a different party)"
            rows={2}
            className="text-xs"
            aria-label="Reason for invoicing outside the statement"
          />
        </div>

        <DialogFooter>
          <Button variant="outline" onClick={() => onOpenChange(false)}>
            Cancel
          </Button>
          <Button
            disabled={reason.trim().length === 0}
            isLoading={pending}
            loadingText="Creating..."
            onClick={() => onConfirm(reason.trim())}
            title={reason.trim().length === 0 ? "Say why this bills on its own" : undefined}
          >
            Invoice anyway
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
