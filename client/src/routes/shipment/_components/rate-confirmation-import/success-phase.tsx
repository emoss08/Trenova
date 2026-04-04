import { Button } from "@/components/ui/button";
import { m } from "motion/react";
import { CheckIcon, ExternalLinkIcon } from "lucide-react";
import { Link } from "react-router";

type SuccessPhaseProps = {
  shipmentId: string;
  attachError: string | null;
  onDone: () => void;
};

export function SuccessPhase({ shipmentId, attachError, onDone }: SuccessPhaseProps) {
  return (
    <div className="flex flex-1 flex-col items-center justify-center p-8">
      <m.div
        initial={{ opacity: 0, scale: 0.95 }}
        animate={{ opacity: 1, scale: 1 }}
        transition={{ duration: 0.3 }}
        className="flex flex-col items-center gap-6 max-w-xs text-center"
      >
        <div className="flex size-10 items-center justify-center rounded-full bg-foreground text-background">
          <CheckIcon className="size-5" />
        </div>

        <div className="space-y-1">
          <h3 className="text-base font-medium">Shipment created</h3>
          <p className="text-xs text-muted-foreground tabular-nums">{shipmentId}</p>
        </div>

        {attachError && (
          <p className="text-xs text-muted-foreground">
            Document could not be attached: {attachError}
          </p>
        )}

        {!attachError && (
          <p className="text-xs text-muted-foreground">
            Source document attached successfully.
          </p>
        )}

        <div className="flex gap-2">
          <Button
            variant="outline"
            size="sm"
            render={<Link to="/shipment-management/shipments" />}
          >
            <ExternalLinkIcon className="size-3" />
            Open shipments
          </Button>
          <Button size="sm" onClick={onDone}>
            Done
          </Button>
        </div>
      </m.div>
    </div>
  );
}
