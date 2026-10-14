import { useT } from "@trenova/shared/i18n/use-t";
import { Button } from "@trenova/shared/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@trenova/shared/components/ui/dialog";
import { Label } from "@trenova/shared/components/ui/label";
import { Textarea } from "@trenova/shared/components/ui/textarea";
import { useApiMutation } from "@/hooks/use-api-mutation";
import { customerStatement } from "@/lib/queries/customer-statement";
import { apiService } from "@/services/api";
import type { StatementRange } from "@/types/customer-statement";
import { useQueryClient } from "@tanstack/react-query";
import { useState } from "react";
import { toast } from "sonner";

const MAX_NOTE = 2000;

/**
 * Emails the statement on screen, as a PDF, to the invoice recipients on the
 * customer's email profile. It is logged to the account's collections history
 * and to the statements sent below.
 */
export function SendStatementDialog({
  customerId,
  customerName,
  range,
  rangeLabel,
  open,
  onOpenChange,
}: {
  customerId: string;
  customerName: string;
  range: StatementRange;
  rangeLabel: string;
  open: boolean;
  onOpenChange: (open: boolean) => void;
}) {
  const t = useT();
  const queryClient = useQueryClient();
  const [note, setNote] = useState("");

  const { mutate, isPending } = useApiMutation({
    mutationFn: () => apiService.customerStatementService.send(customerId, range, note.trim()),
    resourceName: "CustomerStatement",
    onSuccess: (result) => {
      void queryClient.invalidateQueries({
        queryKey: customerStatement.sent(customerId).queryKey,
      });
      void queryClient.invalidateQueries({ queryKey: ["collection"] });
      toast.success(t("Statement sent to {0}", result.sentTo.join(", ")), {
        description: t("{0} due", result.endingBalance),
      });
      setNote("");
      onOpenChange(false);
    },
  });

  return (
    <Dialog open={open} onOpenChange={(next) => !isPending && onOpenChange(next)}>
      <DialogContent size="md">
        <DialogHeader>
          <DialogTitle>{t("Email {0} this statement?", customerName)}</DialogTitle>
          <DialogDescription>
            {t(
              "Sends the statement {0} as a PDF to the invoice recipients on the customer's email profile.",
              rangeLabel,
            )}
          </DialogDescription>
        </DialogHeader>
        <div className="flex flex-col gap-1.5">
          <Label htmlFor="statement-note">{t("Note to the customer")}</Label>
          <Textarea
            id="statement-note"
            rows={4}
            maxLength={MAX_NOTE}
            value={note}
            placeholder={t("e.g., Here is the statement you asked for")}
            onChange={(event) => setNote(event.target.value)}
          />
        </div>
        <DialogFooter>
          <Button variant="outline" disabled={isPending} onClick={() => onOpenChange(false)}>
            {t("Cancel")}
          </Button>
          <Button isLoading={isPending} loadingText={t("Sending...")} onClick={() => mutate()}>
            {t("Send statement")}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
