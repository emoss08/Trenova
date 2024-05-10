import { Input } from "@/components/common/fields/input";
import { Label } from "@/components/common/fields/label";
import { Button } from "@/components/ui/button";
import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from "@/components/ui/popover";
import { cn } from "@/lib/utils";
import { faPaintBrush } from "@fortawesome/pro-duotone-svg-icons";
import { FontAwesomeIcon } from "@fortawesome/react-fontawesome";
import {
  FieldValues,
  UseControllerProps,
  useController,
} from "react-hook-form";
import { FieldDescription } from "./components";
import { FieldErrorMessage } from "./error-message";

export function GradientPicker<TFieldValues extends FieldValues>({
  className,
  ...props
}: {
  className?: string;
  label?: string;
  description?: string;
} & UseControllerProps<TFieldValues>) {
  const { field, fieldState } = useController(props);

  // Define the solid colors array
  const solids = [
    "#E2E2E2", // Light gray
    "#ff75c3", // Bright pink
    "#ffa647", // Vibrant orange
    "#ffe83f", // Bright yellow
    "#9fff5b", // Vibrant green
    "#70e2ff", // Light blue
    "#cd93ff", // Soft purple
    "#09203f", // Dark blue-grey
    "#ff7575", // Soft red
    "#b284be", // Periwinkle
    "#ff7f50", // Coral
    "#deb887", // Burlywood
    "#5f9ea0", // Cadet blue
    "#ffd700", // Gold
    "#6a5acd", // Slate blue
    "#ff4500", // Orange red
    "#2e8b57", // Sea green
    "#4682b4", // Steel blue
    "#d2691e", // Chocolate (new)
    "#6495ed", // Cornflower blue (new)
    "#dc143c", // Crimson (new)
    "#008b8b", // Dark cyan (new)
    "#b8860b", // Dark goldenrod (new)
    "#006400", // Dark green (new)
    "#8b008b", // Dark magenta (new)
    "#556b2f", // Dark olive green (new)
    "#ff8c00", // Dark orange (new)
    "#9932cc", // Dark orchid (new)
    "#8b0000", // Dark red (new)
    "#e9967a", // Dark salmon (new)
  ];

  // Handler to update the field value
  const handleChange = (newColor: string) => {
    field.onChange(newColor);
  };

  return (
    <Popover>
      <PopoverTrigger asChild>
        <div className="relative">
          {props.label && (
            <Label
              className={cn(
                "text-sm font-medium",
                props.rules?.required && "required",
              )}
            >
              {props.label}
            </Label>
          )}
          <Button
            variant={"outline"}
            type="button"
            className={cn(
              "w-full justify-start text-left font-normal truncate",
              !field.value && "text-muted-foreground",
              className,
            )}
          >
            <div className="flex w-full items-center gap-2">
              {field.value ? (
                <div
                  className="size-4 rounded !bg-cover !bg-center transition-all"
                  style={{ background: field.value }}
                ></div>
              ) : (
                <FontAwesomeIcon icon={faPaintBrush} className="size-4" />
              )}
              <div className="flex-1 truncate">
                {field.value ? field.value : "Pick a color"}
              </div>
            </div>
            {fieldState.invalid && (
              <FieldErrorMessage formError={fieldState.error?.message} />
            )}
            {props.description && !fieldState.invalid && (
              <FieldDescription description={props.description} />
            )}
          </Button>
        </div>
      </PopoverTrigger>
      <PopoverContent className="w-64">
        <div className="mt-0 flex flex-wrap gap-1">
          {solids.map((color) => (
            <div
              key={color}
              style={{ background: color }}
              className="size-6 cursor-pointer rounded-md active:scale-105"
              onClick={() => handleChange(color)}
            />
          ))}
        </div>
        <Input
          id="custom"
          value={field.value || ""}
          className="col-span-2 mt-4 h-8"
          onChange={(e) => field.onChange(e.target.value)}
        />
      </PopoverContent>
    </Popover>
  );
}
