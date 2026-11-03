import { checkAssignment, type DriverLane, type TimelineLoad } from "@/lib/shipment-board/timeline";
import { AlertTriangleIcon, CheckIcon, XCloseIcon } from "@trenova/shared/components/icons";
import { Button } from "@trenova/shared/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@trenova/shared/components/ui/dialog";
import { useNow } from "@trenova/shared/hooks/use-now";
import { useT } from "@trenova/shared/i18n/use-t";
import {
  formatClockDurationMs,
  formatMinutesSpan,
  formatUnixDateTimeShort,
} from "@trenova/shared/lib/date";
import { cn } from "@trenova/shared/lib/utils";
import type { ReactNode } from "react";
import { useBoardActions } from "../../use-board-actions";
import { useDriverHos } from "../../use-driver-hos";

export type PendingAssignment = { load: TimelineLoad; lane: DriverLane };

/**
 * Confirms a load dropped on a driver: who, which tractor, the pickup, and
 * whatever should give the dispatcher pause first. A driver with no tractor
 * cannot be dispatched, so that refuses the assignment; an overlap or short
 * drive time only asks for "Assign anyway".
 */
export function TimelineAssignDialog({
  pending,
  onClose,
}: {
  pending: PendingAssignment | null;
  onClose: () => void;
}) {
  return (
    <Dialog open={!!pending} onOpenChange={(open) => !open && onClose()}>
      {pending ? <AssignBody pending={pending} onClose={onClose} /> : null}
    </Dialog>
  );
}

function AssignBody({ pending, onClose }: { pending: PendingAssignment; onClose: () => void }) {
  const t = useT();
  const actions = useBoardActions();
  const { load, lane } = pending;
  const readHos = useDriverHos(lane.driveRemainingMs == null ? lane.id : null);
  const remaining = lane.driveRemainingMs ?? readHos;
  const now = useNow("minute");
  const check = checkAssignment(load, lane, remaining, now);
  const pickup = load.stops[0];
  const warned = check.overlaps.length > 0 || !!check.driveShort;

  const assign = async () => {
    if (!load.moveId || !lane.tractorId) return;
    await actions.assign.mutateAsync([
      { moveId: load.moveId, workerId: lane.id, tractorId: lane.tractorId },
    ]);
    onClose();
  };

  return (
    <DialogContent size="sm">
      <DialogHeader>
        <DialogTitle>
          {t("Assign {0} to {1}", load.shipment.proNumber ?? "—", lane.name)}
        </DialogTitle>
        <DialogDescription>
          {t(
            "{0} · pickup {1} in {2}",
            lane.tractorCode ?? t("No tractor"),
            formatUnixDateTimeShort(pickup.windowStart),
            pickup.place,
          )}
        </DialogDescription>
      </DialogHeader>
      <ul className="flex flex-col gap-1.5 text-sm">
        {check.blocked ? (
          <CheckLine tone="danger">
            {t("{0} has no tractor to dispatch with.", lane.name)}
          </CheckLine>
        ) : null}
        {check.overlaps.length > 0 ? (
          check.overlaps.map((other) => (
            <CheckLine key={other.id} tone="warning">
              {t(
                "Overlaps {0}, {1} to {2}.",
                other.shipment.proNumber ?? "—",
                formatUnixDateTimeShort(other.start),
                formatUnixDateTimeShort(other.end),
              )}
            </CheckLine>
          ))
        ) : (
          <CheckLine tone="success">{t("Free for the whole load.")}</CheckLine>
        )}
        {check.driveShort ? (
          <CheckLine tone="warning">
            {t(
              "Needs about {0} of driving; {1} left today.",
              formatMinutesSpan(check.driveShort.neededMinutes),
              formatClockDurationMs(check.driveShort.remainingMs),
            )}
          </CheckLine>
        ) : remaining != null && load.driveMinutes != null ? (
          <CheckLine tone="success">
            {t(
              "{0} of drive time left covers about {1}.",
              formatClockDurationMs(remaining),
              formatMinutesSpan(load.driveMinutes),
            )}
          </CheckLine>
        ) : null}
      </ul>
      <DialogFooter>
        <Button variant="outline" onClick={onClose}>
          {t("Cancel")}
        </Button>
        <Button
          disabled={check.blocked || !load.moveId}
          isLoading={actions.assign.isPending}
          loadingText={t("Assigning")}
          onClick={() => void assign()}
        >
          {warned ? t("Assign anyway") : t("Assign")}
        </Button>
      </DialogFooter>
    </DialogContent>
  );
}

function CheckLine({
  tone,
  children,
}: {
  tone: "success" | "warning" | "danger";
  children: ReactNode;
}) {
  const Icon = tone === "success" ? CheckIcon : tone === "warning" ? AlertTriangleIcon : XCloseIcon;
  return (
    <li className="flex gap-2">
      <Icon
        aria-hidden
        className={cn(
          "mt-0.5 size-4 shrink-0",
          tone === "success" && "text-success",
          tone === "warning" && "text-warning",
          tone === "danger" && "text-danger",
        )}
      />
      <span>{children}</span>
    </li>
  );
}
