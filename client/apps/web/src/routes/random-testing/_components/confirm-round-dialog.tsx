import type { RandomDrawListRow } from "@/lib/graphql/worker-drug-alcohol";
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from "@trenova/shared/components/ui/alert-dialog";

export type RoundAction = {
  kind: "finalise" | "void";
  draw: RandomDrawListRow;
};

type ConfirmRoundDialogProps = {
  action: RoundAction | null;
  pending: boolean;
  onOpenChange: (open: boolean) => void;
  onConfirm: (action: RoundAction) => void;
};

/**
 * Both moves are one way. Finalising makes the names the record; voiding
 * throws the round away and reopens the period. Neither should be a slip of
 * the mouse on a table row.
 */
export function ConfirmRoundDialog({
  action,
  pending,
  onOpenChange,
  onConfirm,
}: ConfirmRoundDialogProps) {
  const finalise = action?.kind === "finalise";
  return (
    <AlertDialog open={action !== null} onOpenChange={onOpenChange}>
      <AlertDialogContent>
        <AlertDialogHeader>
          <AlertDialogTitle>
            {finalise
              ? `Finalise round ${action.draw.periodKey}?`
              : `Void round ${action?.draw.periodKey}?`}
          </AlertDialogTitle>
          <AlertDialogDescription>
            {finalise
              ? `${action.draw.drugSelected} drivers for drug testing and ${action.draw.alcoholSelected} for alcohol become the record for this period. Correcting a name afterwards means voiding the whole round.`
              : "The selections are discarded and the period can be drawn again. The voided round stays on file with its seed, so the audit trail shows it happened."}
          </AlertDialogDescription>
        </AlertDialogHeader>
        <AlertDialogFooter>
          <AlertDialogCancel disabled={pending}>Keep as draft</AlertDialogCancel>
          <AlertDialogAction
            variant={finalise ? "default" : "destructive"}
            disabled={pending || action === null}
            onClick={() => action && onConfirm(action)}
          >
            {finalise ? "Finalise" : "Void the round"}
          </AlertDialogAction>
        </AlertDialogFooter>
      </AlertDialogContent>
    </AlertDialog>
  );
}
