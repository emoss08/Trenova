import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Icon } from "@/components/ui/icons";
import type { ShipmentSchema } from "@/lib/schemas/shipment-schema";
import { cn } from "@/lib/utils";
import type { DocumentCategory } from "@/types/document";
import { ShipmentStatus } from "@/types/shipment";
import { faArrowRight } from "@fortawesome/pro-solid-svg-icons";
import { useMemo } from "react";

export function BillingReadinessBadge({
  documentCategories,
  shipmentStatus,
}: {
  documentCategories: DocumentCategory[];
  shipmentStatus: ShipmentSchema["status"];
}) {
  const billingReadiness = useMemo(() => {
    const requiredCategories = documentCategories;
    const completedRequired = documentCategories.filter((cat) => cat.complete);
    const isShipmentCompleted = shipmentStatus === ShipmentStatus.Completed;

    return {
      total: requiredCategories.length,
      completed: completedRequired.length,
      ready:
        requiredCategories.length > 0 &&
        requiredCategories.length === completedRequired.length &&
        isShipmentCompleted,
      isShipmentCompleted,
      isDocumentsComplete:
        requiredCategories.length === completedRequired.length &&
        requiredCategories.length > 0,
    };
  }, [documentCategories, shipmentStatus]);

  return (
    billingReadiness.total > 0 && (
      <div className="mt-3 p-3 rounded-lg bg-background border border-border">
        <div className="flex items-center justify-between mb-2">
          <div className="flex items-center">
            <div
              className={cn(
                "w-1.5 h-5 rounded-sm mr-2",
                billingReadiness.ready
                  ? "bg-green-700 border border-green-500/60"
                  : billingReadiness.isDocumentsComplete &&
                      !billingReadiness.isShipmentCompleted
                    ? "bg-blue-700 border border-blue-500/60"
                    : "bg-primary",
              )}
            />
            <span className="text-xs font-medium">Billing Readiness</span>
          </div>
          {billingReadiness.ready ? (
            <Button
              title="Mark Ready to Bill"
              aria-label="Mark Ready to Bill"
              variant="green"
              size="xs"
            >
              Mark Ready to Bill
              <Icon icon={faArrowRight} className="size-4 ml-1" />
            </Button>
          ) : (
            <Badge
              withDot={false}
              className={cn(
                "text-2xs",
                billingReadiness.isDocumentsComplete &&
                  !billingReadiness.isShipmentCompleted &&
                  "bg-blue-700/20 text-blue-500 border border-blue-500/60",
              )}
            >
              {billingReadiness.completed}/{billingReadiness.total} Documents
            </Badge>
          )}
        </div>

        <div className="flex w-full h-1.5 bg-background rounded-full overflow-hidden">
          {Array.from({ length: billingReadiness.total }).map((_, index) => (
            <div
              key={index}
              className={cn(
                "h-full flex-1 mx-px first:ml-0 last:mr-0 transition-colors",
                index < billingReadiness.completed
                  ? billingReadiness.ready
                    ? "bg-green-700 border border-green-500/60"
                    : billingReadiness.isDocumentsComplete &&
                        !billingReadiness.isShipmentCompleted
                      ? "bg-blue-700 border border-blue-500/60"
                      : "bg-primary"
                  : "bg-transparent",
              )}
            />
          ))}
        </div>

        {billingReadiness.isDocumentsComplete &&
          !billingReadiness.isShipmentCompleted && (
            <div className="flex items-center mt-2 px-2 py-1 bg-blue-600/20 border border-blue-500 rounded text-xs text-blue-500">
              Waiting for shipment completion
            </div>
          )}
      </div>
    )
  );
}
