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
import { Label } from "@trenova/shared/components/ui/label";
import type { StopActualAction } from "@trenova/shared/types/shipment";
import { useEffect, useId, useState } from "react";

const actionCopy: Record<StopActualAction, { title: string; description: string }> = {
  Arrive: {
    title: "Record Arrival",
    description:
      "Records the carrier's arrival at this stop and advances the move. Leave the event time empty to record it as of now, or backdate it to when the check call reported the arrival.",
  },
  Depart: {
    title: "Record Departure",
    description:
      "Records the carrier's departure from this stop and advances the move. Leave the event time empty to record it as of now, or backdate it to when the check call reported the departure.",
  },
};

export function RecordStopActualDialog({
  open,
  onOpenChange,
  action,
  stopDescription,
  isSubmitting,
  onConfirm,
}: {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  action: StopActualAction;
  stopDescription?: string | null;
  isSubmitting: boolean;
  onConfirm: (occurredAt?: number) => void;
}) {
  const inputId = useId();
  const [eventTime, setEventTime] = useState("");
  const [error, setError] = useState<string | null>(null);
  const copy = actionCopy[action];

  useEffect(() => {
    if (open) {
      setEventTime("");
      setError(null);
    }
  }, [open]);

  const handleConfirm = () => {
    if (!eventTime) {
      onConfirm(undefined);
      return;
    }
    const parsed = new Date(eventTime);
    if (Number.isNaN(parsed.getTime())) {
      setError("Enter a valid date and time");
      return;
    }
    const occurredAt = Math.floor(parsed.getTime() / 1000);
    if (occurredAt > Math.floor(Date.now() / 1000)) {
      setError("The event time cannot be in the future");
      return;
    }
    onConfirm(occurredAt);
  };

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-[440px]">
        <DialogHeader>
          <DialogTitle>{copy.title}</DialogTitle>
          <DialogDescription>
            {stopDescription ? `${stopDescription} — ${copy.description}` : copy.description}
          </DialogDescription>
        </DialogHeader>
        <div className="flex flex-col gap-1.5 pb-4">
          <Label htmlFor={inputId}>Event time (optional)</Label>
          <Input
            id={inputId}
            type="datetime-local"
            value={eventTime}
            onChange={(event) => {
              setEventTime(event.target.value);
              setError(null);
            }}
          />
          {error ? (
            <p className="text-xs text-destructive">{error}</p>
          ) : (
            <p className="text-xs text-muted-foreground">Leave empty to record the time as now.</p>
          )}
        </div>
        <DialogFooter>
          <Button type="button" variant="outline" onClick={() => onOpenChange(false)}>
            Cancel
          </Button>
          <Button
            type="button"
            isLoading={isSubmitting}
            loadingText="Recording..."
            onClick={handleConfirm}
          >
            {copy.title}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
