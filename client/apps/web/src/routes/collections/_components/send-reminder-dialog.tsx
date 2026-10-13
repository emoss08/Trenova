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
import { apiService } from "@/services/api";
import { useQueryClient } from "@tanstack/react-query";
import { useState } from "react";
import { toast } from "sonner";

const MAX_NOTE = 2000;

/**
 * Emails the customer a reminder of every overdue invoice now, outside the
 * schedule, with a note. The reminder is logged to the account's history.
 */
export function SendReminderDialog({
  customerId,
  customerName,
  open,
  onOpenChange,
}: {
  customerId: string;
  customerName: string;
  open: boolean;
  onOpenChange: (open: boolean) => void;
}) {
  const t = useT();
  const queryClient = useQueryClient();
  const [note, setNote] = useState("");

  const { mutate, isPending } = useApiMutation({
    mutationFn: () => apiService.collectionService.sendReminder(customerId, note.trim()),
    resourceName: "PaymentReminder",
    onSuccess: (result) => {
      void queryClient.invalidateQueries({ queryKey: ["collection"] });
      const recipients = [...new Set(result.reminders.flatMap((reminder) => reminder.sentTo))];
      toast.success(t("Reminder sent to {0}", recipients.join(", ")), {
        description: result.reminders.map((reminder) => reminder.totalOverdue).join(" · "),
      });
      setNote("");
      onOpenChange(false);
    },
  });

  return (
    <Dialog open={open} onOpenChange={(next) => !isPending && onOpenChange(next)}>
      <DialogContent size="md">
        <DialogHeader>
          <DialogTitle>{t("Send {0} a payment reminder?", customerName)}</DialogTitle>
          <DialogDescription>
            {t(
              "Emails every overdue invoice not in dispute to the invoice recipients on the customer's email profile, whatever the reminder schedule says.",
            )}
          </DialogDescription>
        </DialogHeader>
        <div className="flex flex-col gap-1.5">
          <Label htmlFor="reminder-note">{t("Note to the customer")}</Label>
          <Textarea
            id="reminder-note"
            rows={4}
            maxLength={MAX_NOTE}
            value={note}
            placeholder={t("e.g., Following up on our call this morning")}
            onChange={(event) => setNote(event.target.value)}
          />
        </div>
        <DialogFooter>
          <Button variant="outline" disabled={isPending} onClick={() => onOpenChange(false)}>
            {t("Cancel")}
          </Button>
          <Button isLoading={isPending} loadingText={t("Sending...")} onClick={() => mutate()}>
            {t("Send reminder")}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
