import { TextareaField } from "@/components/fields/textarea-field";
import { Button } from "@/components/ui/button";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { FormControl, FormGroup } from "@/components/ui/form";
import { Icon } from "@/components/ui/icons";
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from "@/components/ui/tooltip";
import { faChevronDown } from "@fortawesome/pro-regular-svg-icons";
import { useFormContext } from "react-hook-form";

// Define preset cancellation reasons
const CANCELLATION_PRESETS = [
  {
    id: "shipper-request",
    label: "Shipper Request",
    description: "Shipment canceled at shipper's request",
  },
  {
    id: "booking-change",
    label: "Booking Change",
    description: "Booking date or details need to be modified",
  },
  {
    id: "duplicate-entry",
    label: "Duplicate Entry",
    description: "Shipment was entered multiple times",
  },
  {
    id: "rate-issue",
    label: "Rate Issue",
    description: "Pricing or rate-related cancellation",
  },
  {
    id: "service-unavailable",
    label: "Service Unavailable",
    description: "Required service is not available",
  },
] as const;

export function ShipmentCancellationForm() {
  const { control, setValue } = useFormContext();

  const handlePresetSelect = (description: string) => {
    setValue("cancelReason", description, {
      shouldValidate: true,
      shouldDirty: true,
    });
  };

  return (
    <FormGroup cols={1}>
      <FormControl cols="full">
        <div className="relative">
          <TextareaField
            control={control}
            rules={{ required: true }}
            name="cancelReason"
            label="Cancel Reason"
            placeholder="Cancel Reason"
            description="Provide a reason for cancelling the shipment."
          />
          <div className="absolute top-5 right-1">
            <DropdownMenu>
              <DropdownMenuTrigger className="outline-none">
                <TooltipProvider>
                  <Tooltip>
                    <TooltipTrigger>
                      <Button
                        variant="ghost"
                        className="text-2xs gap-1 h-5 w-16 hover:bg-background"
                      >
                        Presets <Icon icon={faChevronDown} />
                      </Button>
                    </TooltipTrigger>
                    <TooltipContent>
                      <p>Select a preset reason</p>
                    </TooltipContent>
                  </Tooltip>
                </TooltipProvider>
              </DropdownMenuTrigger>
              <DropdownMenuContent align="end" className="w-[240px]">
                {CANCELLATION_PRESETS.map((preset) => (
                  <DropdownMenuItem
                    key={preset.id}
                    onClick={() => handlePresetSelect(preset.description)}
                    className="flex flex-col items-start py-2 gap-1"
                    title={preset.label}
                    description={preset.description}
                  />
                ))}
              </DropdownMenuContent>
            </DropdownMenu>
          </div>
        </div>
      </FormControl>
    </FormGroup>
  );
}
