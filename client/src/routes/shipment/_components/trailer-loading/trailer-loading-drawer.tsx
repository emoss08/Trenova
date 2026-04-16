import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { ScrollArea } from "@/components/ui/scroll-area";
import { cn } from "@/lib/utils";
import { apiService } from "@/services/api";
import { ContainerIcon, LoaderIcon, PrinterIcon, SaveIcon } from "lucide-react";
import { useState } from "react";
import { toast } from "sonner";
import { COMMODITY_PALETTE } from "./constants";
import { LinearFeetBar } from "./linear-feet-bar";
import { LoadingRecommendations } from "./loading-recommendations";
import { LoadingWarnings } from "./loading-warnings";
import {
  buildLoadPlanBlob,
  printLoadPlan,
  useShipmentMeta,
} from "./print-load-plan";
import { TrailerTopView } from "./trailer-top-view";
import { useLoadingOptimization } from "./use-loading-optimization";
import { AxleWeightDisplay } from "./weight-distribution-bar";

const gradeColors: Record<string, string> = {
  Excellent: "bg-emerald-500/15 text-emerald-600 dark:text-emerald-400",
  Good: "bg-primary/15 text-primary",
  Fair: "bg-warning/15 text-warning",
  Poor: "bg-destructive/15 text-destructive",
};

export default function LoadPlannerDialog({
  open,
  onOpenChange,
}: {
  open: boolean;
  onOpenChange: (open: boolean | null) => void;
}) {
  const { data, revenue, calculate, isPending, hasCommodities } =
    useLoadingOptimization();
  const shipmentMeta = useShipmentMeta();
  const [saving, setSaving] = useState(false);

  async function handleSaveToShipment() {
    if (!data || !shipmentMeta.shipmentId) return;
    setSaving(true);
    try {
      const blob = buildLoadPlanBlob(data, shipmentMeta);
      const fileName = `load-plan-${shipmentMeta.proNumber || "draft"}.html`;
      const file = new File([blob], fileName, { type: "text/html" });
      await apiService.documentService.upload({
        file,
        resourceId: shipmentMeta.shipmentId,
        resourceType: "shipment",
        description: "Load Plan",
      });
      toast.success("Load plan saved to shipment documents");
    } catch {
      toast.error("Failed to save load plan");
    } finally {
      setSaving(false);
    }
  }

  return (
    <Dialog
      open={open}
      onOpenChange={(isOpen) => {
        if (!isOpen) onOpenChange(null);
      }}
    >
      <DialogContent className="flex max-h-[85vh] flex-col gap-0 overflow-hidden p-0 sm:max-w-3xl">
        <DialogHeader className="px-5 pt-5 pb-3">
          <div className="flex items-center gap-2">
            <div className="flex size-7 items-center justify-center rounded-md bg-primary/10">
              <ContainerIcon className="size-4 text-primary" />
            </div>
            <div className="flex-1">
              <DialogTitle className="text-sm">Load Planner</DialogTitle>
              <DialogDescription className="text-2xs">
                Commodity placement, weight distribution, and compliance
              </DialogDescription>
            </div>
          </div>
        </DialogHeader>

        <ScrollArea
          className="flex-1"
          viewportClassName="max-h-[calc(85vh-140px)]"
        >
          <div className="flex flex-col gap-3 px-5 pb-4">
            {data ? (
              <>
                <TrailerTopView
                  data={data}
                  scoreBadge={
                    <span
                      className={cn(
                        "rounded-full px-2 py-0.5 text-2xs font-semibold tabular-nums",
                        gradeColors[data.utilizationGrade] ?? gradeColors.Poor,
                      )}
                    >
                      {data.utilizationScore}% {data.utilizationGrade}
                    </span>
                  }
                />

                <LinearFeetBar
                  totalLinearFeet={data.totalLinearFeet}
                  trailerLengthFeet={data.trailerLengthFeet}
                  utilization={data.linearFeetUtil}
                  commodities={data.placements.map((p, idx) => ({
                    name: p.commodityName,
                    weight: p.weight,
                    lengthFeet: p.lengthFeet,
                    instructions: p.loadingInstructions,
                    palette: COMMODITY_PALETTE[idx % COMMODITY_PALETTE.length],
                  }))}
                />

                <AxleWeightDisplay
                  axleWeights={data.axleWeights}
                  totalWeight={data.totalWeight}
                  maxWeight={data.maxWeight}
                  revenue={revenue}
                />

                {data.recommendations.length > 0 && (
                  <div>
                    <span className="mb-1.5 block text-2xs font-medium tracking-wider text-muted-foreground uppercase">
                      Recommendations
                    </span>
                    <LoadingRecommendations
                      recommendations={data.recommendations}
                    />
                  </div>
                )}

                {data.warnings.length > 0 && (
                  <div>
                    <span className="mb-1.5 block text-2xs font-medium tracking-wider text-muted-foreground uppercase">
                      Alerts
                    </span>
                    <LoadingWarnings warnings={data.warnings} />
                  </div>
                )}
              </>
            ) : (
              <div className="flex flex-col items-center justify-center gap-4 py-20">
                <div className="flex size-14 items-center justify-center rounded-2xl bg-muted">
                  <ContainerIcon className="size-7 text-muted-foreground/40" />
                </div>
                <div className="text-center">
                  <p className="text-sm font-medium text-foreground">
                    No loading plan yet
                  </p>
                  <p className="mt-1 text-2xs text-muted-foreground">
                    Click below to calculate optimal placement
                  </p>
                </div>
              </div>
            )}
          </div>
        </ScrollArea>

        <DialogFooter className="m-0 flex flex-row justify-between">
          {data && (
            <>
              <Button
                type="button"
                variant="outline"
                onClick={() => printLoadPlan(data, shipmentMeta)}
              >
                <PrinterIcon className="size-3.5" />
                Print
              </Button>
              <Button
                type="button"
                variant="outline"
                onClick={handleSaveToShipment}
                disabled={saving || !shipmentMeta.shipmentId}
              >
                {saving ? (
                  <LoaderIcon className="size-3.5 animate-spin" />
                ) : (
                  <SaveIcon className="size-3.5" />
                )}
                Save
              </Button>
            </>
          )}
          <Button
            type="button"
            onClick={calculate}
            disabled={isPending || !hasCommodities}
          >
            {isPending && <LoaderIcon className="size-3.5 animate-spin" />}
            {data ? "Recalculate" : "Calculate Optimal Loading"}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
