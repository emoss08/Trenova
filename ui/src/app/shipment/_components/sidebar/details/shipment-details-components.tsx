import { ShipmentStatusBadge } from "@/components/status-badge";
import { Icon } from "@/components/ui/icons";
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from "@/components/ui/tooltip";
import { useShipment } from "@/hooks/use-shipment";
import { cn } from "@/lib/utils";
import { mapToRatingMethod } from "@/types/shipment";
import { faCheck, faCopy } from "@fortawesome/pro-solid-svg-icons";
import { useState } from "react";
import { DetailsRow, ShipmentDetailColumn } from "./shipment-detail-column";

export function ShipmentDetailsHeader() {
  const { shipment } = useShipment();

  if (!shipment) {
    return null;
  }

  return (
    <div className="flex flex-col gap-0.5 px-4 pb-2 border-b border-bg-sidebar-border">
      <div className="flex items-center gap-2 justify-between">
        <h2 className="text-xl">{shipment.proNumber}</h2>
        <ShipmentStatusBadge status={shipment.status} />
      </div>
      <ShipmentDetailsBOL label="Tracking ID:" bol={shipment.bol} />
    </div>
  );
}

export function ShipmentServiceDetails() {
  const { shipment } = useShipment();

  if (!shipment) {
    return null;
  }

  return (
    <div className="flex flex-col gap-2 pb-4">
      <h3 className="text-sm font-medium">Service Information</h3>
      <div className="grid grid-cols-2 gap-y-4">
        <div className="space-y-4">
          <div className="space-y-1">
            <p className="text-sm text-muted-foreground">Service Type</p>
            <ShipmentDetailColumn
              color={shipment.serviceType.color}
              text={shipment.serviceType.code}
            />
          </div>
          <div className="space-y-1">
            <p className="text-sm text-muted-foreground">Shipment Type</p>
            <ShipmentDetailColumn
              color={shipment.shipmentType.color}
              text={shipment.shipmentType.code}
            />
          </div>
        </div>
        <div className="space-y-4 text-right">
          <div className="space-y-1">
            <p className="text-sm text-muted-foreground">Tractor Type</p>
            <ShipmentDetailColumn
              color={shipment.tractorType?.color}
              text={shipment.tractorType?.code ?? "-"}
              className="justify-end"
            />
          </div>
          <div className="space-y-1">
            <p className="text-sm text-muted-foreground">Trailer Type</p>
            <ShipmentDetailColumn
              color={shipment.trailerType?.color}
              text={shipment.trailerType?.code ?? "-"}
              className="justify-end"
            />
          </div>
        </div>
      </div>
    </div>
  );
}

export function ShipmentDetailsBOL({
  bol,
  className,
  label,
}: {
  bol: string;
  className?: string;
  label?: string;
}) {
  const [copied, setCopied] = useState(false);

  const handleCopy = async () => {
    try {
      await navigator.clipboard.writeText(bol);
      setCopied(true);
      setTimeout(() => setCopied(false), 1500);
    } catch (error) {
      console.error("Failed to copy BOL:", error);
    }
  };
  return (
    <div className={cn("flex items-center gap-2 text-sm", className)}>
      <span className="text-muted-foreground">{label}</span>
      <div className="flex items-center gap-1">
        <div className="relative inline-block">
          <span
            className={cn(
              "font-medium underline transition-colors duration-300",
              copied ? "text-green-600" : "text-blue-500",
            )}
          >
            {!copied ? bol : "Copied to clipboard"}
          </span>
        </div>
        <TooltipProvider delayDuration={0}>
          <Tooltip>
            <TooltipTrigger asChild>
              <button
                onClick={handleCopy}
                className="inline-flex items-center justify-center h-5 cursor-pointer"
                disabled={copied}
                aria-label={copied ? "Copied" : "Copy BOL number"}
              >
                <div className="relative flex items-center justify-center w-3 h-3">
                  <div
                    className={cn(
                      "absolute inset-0 flex items-center justify-center transition-all duration-300",
                      copied ? "opacity-100 scale-100" : "opacity-0 scale-0",
                    )}
                  >
                    <Icon icon={faCheck} className="text-green-600 size-3" />
                  </div>
                  <div
                    className={cn(
                      "absolute inset-0 flex items-center justify-center transition-all duration-300",
                      copied ? "opacity-0 scale-0" : "opacity-100 scale-100",
                    )}
                  >
                    <Icon icon={faCopy} className="text-blue-500 size-3" />
                  </div>
                </div>
              </button>
            </TooltipTrigger>
            <TooltipContent className="px-2 py-1 text-xs">
              {copied ? "Copied!" : "Copy to clipboard"}
            </TooltipContent>
          </Tooltip>
        </TooltipProvider>
      </div>
    </div>
  );
}

export function ShipmentBillingDetails() {
  const { shipment } = useShipment();

  if (!shipment) {
    return null;
  }

  return (
    <div className="flex flex-col gap-2 border-t border-bg-sidebar-border py-4">
      <h3 className="text-sm font-medium">Billing Information</h3>
      <div className="space-y-4">
        <DetailsRow
          label="Rating Method"
          value={mapToRatingMethod(shipment.ratingMethod)}
        />
        <DetailsRow
          label="Rating Unit"
          value={shipment.ratingUnit.toString()}
        />

        <DetailsRow
          label="Other Charge Amount"
          value={shipment.otherChargeAmount.toString()}
        />
        <DetailsRow
          label="Freight Charge Amount"
          value={shipment.freightChargeAmount.toString()}
        />
        <DetailsRow
          label="Total Charge Amount"
          value={shipment.totalChargeAmount.toString()}
        />
      </div>
    </div>
  );
}
