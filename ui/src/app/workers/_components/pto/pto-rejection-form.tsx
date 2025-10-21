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
import { faChevronDown } from "@fortawesome/pro-regular-svg-icons";
import { useFormContext } from "react-hook-form";

// Define preset cancellation reasons
const CANCELLATION_PRESETS = [
  {
    id: "worker-request",
    label: "Worker Request",
    description: "PTO rejected at worker's request",
  },
  {
    id: "business-request",
    label: "Business Request",
    description: "PTO rejected at business's request",
  },
  {
    id: "other",
    label: "Other",
    description: "Other reason",
  },
] as const;

export function PTORejectionForm() {
  const { control, setValue } = useFormContext();

  const handlePresetSelect = (description: string) => {
    setValue("reason", description, {
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
            name="reason"
            label="Reason"
            placeholder="Reason"
            description="Provide a reason for rejecting the PTO."
            className="pb-5"
          />
          <div className="absolute bottom-5 right-1">
            <DropdownMenu>
              <DropdownMenuTrigger className="outline-none">
                <Button
                  title="Select a preset"
                  variant="ghost"
                  className="text-2xs gap-1 h-5 w-16 hover:bg-background"
                >
                  Preset <Icon icon={faChevronDown} />
                </Button>
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
