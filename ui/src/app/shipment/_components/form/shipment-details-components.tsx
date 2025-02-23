import { AutocompleteField } from "@/components/fields/autocomplete";
import { DoubleClickInput } from "@/components/fields/input-field";
import { ColorOptionValue } from "@/components/fields/select-components";
import { ShipmentStatusBadge } from "@/components/status-badge";
import { FormControl, FormGroup } from "@/components/ui/form";
import { Icon } from "@/components/ui/icons";
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from "@/components/ui/tooltip";
import { EquipmentTypeSchema } from "@/lib/schemas/equipment-type-schema";
import { ServiceTypeSchema } from "@/lib/schemas/service-type-schema";
import { ShipmentSchema } from "@/lib/schemas/shipment-schema";
import { ShipmentTypeSchema } from "@/lib/schemas/shipment-type-schema";
import { cn } from "@/lib/utils";
import { faCheck, faCopy } from "@fortawesome/pro-solid-svg-icons";
import { useState } from "react";
import { useFormContext } from "react-hook-form";

export function ShipmentDetailsHeader() {
  const { control, getValues } = useFormContext<ShipmentSchema>();

  const { proNumber, status } = getValues();

  return (
    <div className="flex flex-col gap-0.5 px-4 pb-2 border-b border-bg-sidebar-border">
      <div className="flex items-center gap-2 justify-between">
        <h2 className="text-xl">{proNumber || "New Shipment"}</h2>
        <ShipmentStatusBadge status={status} />
      </div>
      <div className="flex items-center gap-1 text-sm">
        <span className="text-muted-foreground">Tracking ID:</span>
        <DoubleClickInput control={control} name="bol" />
      </div>
    </div>
  );
}

export function ShipmentServiceDetails() {
  const { control } = useFormContext<ShipmentSchema>();

  return (
    <div className="flex flex-col gap-2">
      <h3 className="text-sm font-medium">Service Information</h3>
      <FormGroup cols={2}>
        <FormControl>
          <AutocompleteField<ShipmentTypeSchema, ShipmentSchema>
            name="shipmentTypeId"
            control={control}
            link="/shipment-types/"
            label="Shipment Type"
            rules={{ required: true }}
            placeholder="Select Shipment Type"
            description="Select the shipment type for the shipment."
            getOptionValue={(option) => option.id || ""}
            getDisplayValue={(option) => (
              <ColorOptionValue color={option.color} value={option.code} />
            )}
            renderOption={(option) => (
              <div className="flex flex-col gap-0.5 items-start size-full">
                <ColorOptionValue color={option.color} value={option.code} />
                {option?.description && (
                  <span className="text-2xs text-muted-foreground truncate w-full">
                    {option?.description}
                  </span>
                )}
              </div>
            )}
          />
        </FormControl>
        <FormControl>
          <AutocompleteField<ServiceTypeSchema, ShipmentSchema>
            name="serviceTypeId"
            control={control}
            link="/service-types/"
            label="Service Type"
            rules={{ required: true }}
            placeholder="Select Service Type"
            description="Select the service type for the shipment."
            getOptionValue={(option) => option.id || ""}
            getDisplayValue={(option) => (
              <ColorOptionValue color={option.color} value={option.code} />
            )}
            renderOption={(option) => (
              <div className="flex flex-col gap-0.5 items-start size-full">
                <ColorOptionValue color={option.color} value={option.code} />
                {option?.description && (
                  <span className="text-2xs text-muted-foreground truncate w-full">
                    {option?.description}
                  </span>
                )}
              </div>
            )}
          />
        </FormControl>
        <FormControl>
          <AutocompleteField<EquipmentTypeSchema, ShipmentSchema>
            name="tractorTypeId"
            control={control}
            label="Tractor Type"
            link="/equipment-types/"
            placeholder="Select Tractor Type"
            description="Select the type of tractor used, considering any special requirements (e.g., refrigeration)."
            getOptionValue={(option) => option.id || ""}
            getDisplayValue={(option) => (
              <ColorOptionValue color={option.color} value={option.code} />
            )}
            renderOption={(option) => (
              <div className="flex flex-col gap-0.5 items-start size-full">
                <ColorOptionValue color={option.color} value={option.code} />
                {option?.description && (
                  <span className="text-2xs text-muted-foreground truncate w-full">
                    {option?.description}
                  </span>
                )}
              </div>
            )}
          />
        </FormControl>
        <FormControl>
          <AutocompleteField<EquipmentTypeSchema, ShipmentSchema>
            name="trailerTypeId"
            control={control}
            label="Trailer Type"
            link="/equipment-types/"
            placeholder="Select Trailer Type"
            description="Select the type of trailer used, considering any special requirements (e.g., refrigeration)."
            getOptionValue={(option) => option.id || ""}
            getDisplayValue={(option) => (
              <ColorOptionValue color={option.color} value={option.code} />
            )}
            renderOption={(option) => (
              <div className="flex flex-col gap-0.5 items-start size-full">
                <ColorOptionValue color={option.color} value={option.code} />
                {option?.description && (
                  <span className="text-2xs text-muted-foreground truncate w-full">
                    {option?.description}
                  </span>
                )}
              </div>
            )}
          />
        </FormControl>
      </FormGroup>
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
