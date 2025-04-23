import { Button } from "@/components/ui/button";
import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from "@/components/ui/popover";
import { cn } from "@/lib/utils";
import { ColorFieldProps } from "@/types/fields";
import { faPaintBrush } from "@fortawesome/pro-solid-svg-icons";
import { memo, useCallback, useState } from "react";
import {
  Controller,
  ControllerFieldState,
  ControllerRenderProps,
  FieldValues,
  Path,
} from "react-hook-form";
import { Icon } from "../ui/icons";
import { Input } from "../ui/input";
import { FieldWrapper } from "./field-components";

// Define the solid colors array as objects
const solids = [
  { color: "#E2E2E2", name: "Light gray" },
  { color: "#ff75c3", name: "Bright pink" },
  { color: "#ffa647", name: "Vibrant orange" },
  { color: "#ffe83f", name: "Bright yellow" },
  { color: "#9fff5b", name: "Vibrant green" },
  { color: "#70e2ff", name: "Light blue" },
  { color: "#cd93ff", name: "Soft purple" },
  { color: "#09203f", name: "Dark blue-grey" },
  { color: "#ff7575", name: "Soft red" },
  { color: "#b284be", name: "Periwinkle" },
  { color: "#ff7f50", name: "Coral" },
  { color: "#deb887", name: "Burlywood" },
  { color: "#5f9ea0", name: "Cadet blue" },
  { color: "#ffd700", name: "Gold" },
  { color: "#6a5acd", name: "Slate blue" },
  { color: "#ff4500", name: "Orange red" },
  { color: "#2e8b57", name: "Sea green" },
  { color: "#4682b4", name: "Steel blue" },
  { color: "#d2691e", name: "Chocolate" },
  { color: "#6495ed", name: "Cornflower blue" },
  { color: "#dc143c", name: "Crimson" },
  { color: "#008b8b", name: "Dark cyan" },
  { color: "#b8860b", name: "Dark goldenrod" },
  { color: "#006400", name: "Dark green" },
  { color: "#8b008b", name: "Dark magenta" },
  { color: "#556b2f", name: "Dark olive green" },
  { color: "#ff8c00", name: "Dark orange" },
  { color: "#9932cc", name: "Dark orchid" },
  { color: "#8b0000", name: "Dark red" },
  { color: "#e9967a", name: "Dark salmon" },
  { color: "#20b2aa", name: "Light sea green" },
  { color: "#ff6347", name: "Tomato" },
  { color: "#ffa07a", name: "Light salmon" },
  { color: "#da70d6", name: "Orchid" },
  { color: "#f0e68c", name: "Khaki" },
  { color: "#40e0d0", name: "Turquoise" },
  { color: "#ee82ee", name: "Violet" },
  { color: "#696969", name: "Dim gray" },
  { color: "#ffdab9", name: "Peach puff" },
  { color: "#87cefa", name: "Light sky blue" },
];

const ColorGrid = memo(function ColorGrid({
  handleChange,
}: {
  handleChange: (color: string) => void;
}) {
  return (
    <div className="grid grid-cols-[repeat(auto-fill,minmax(20px,1fr))] gap-1.5">
      {solids.map(({ color }) => (
        <div
          key={color}
          style={{ background: color }}
          className="size-6 cursor-pointer rounded-md active:scale-105"
          onClick={() => handleChange(color)}
        />
      ))}
    </div>
  );
});

// Controlled input component
const ColorFieldInput = memo(function ColorFieldInput<T extends FieldValues>({
  field,
  fieldState,
  className,
  disabled,
}: {
  field: ControllerRenderProps<T, Path<T>>;
  fieldState: ControllerFieldState;
  className?: string;
  disabled?: boolean;
}) {
  const [isOpen, setIsOpen] = useState(false);
  const { onChange, value, ...restField } = field;

  // Memoize the handleChange function
  const handleChange = useCallback(
    (color: string) => {
      setIsOpen(false);
      onChange(color);
    },
    [onChange],
  );

  return (
    <Popover open={isOpen} onOpenChange={setIsOpen}>
      <PopoverTrigger asChild>
        <Button
          {...restField}
          variant="outline"
          type="button"
          className={cn(
            "w-full font-normal gap-2 justify-start text-left items-center rounded border-muted-foreground/20 bg-muted px-1.5 data-[state=open]:border-blue-600 data-[state=open]:outline-hidden data-[state=open]:ring-4 data-[state=open]:ring-blue-600/20",
            "[&_svg]:size-3 justify-between",
            "transition-[border-color,box-shadow] duration-200 ease-in-out",
            disabled && "opacity-50 cursor-not-allowed",
            fieldState.invalid &&
              "border-red-500 bg-red-500/20 ring-0 ring-red-500 placeholder:text-red-500 focus:outline-hidden focus-visible:border-red-600 focus-visible:ring-4 focus-visible:ring-red-400/20 hover:border-red-500 hover:bg-red-500/20 data-[state=open]:border-red-500 data-[state=open]:bg-red-500/20 data-[state=open]:ring-red-500/20",
            !value && "text-muted-foreground",
            className,
          )}
        >
          <div className="flex w-full items-center gap-2">
            {value ? (
              <div
                className="size-4 rounded bg-cover! bg-center! transition-all"
                style={{ background: value }}
              />
            ) : (
              <Icon icon={faPaintBrush} className="size-4" />
            )}
            <div className="flex-1 truncate">
              {value ? value : "Pick a color"}
            </div>
          </div>
        </Button>
      </PopoverTrigger>
      <PopoverContent className="w-(--radix-popover-trigger-width) p-2">
        <div className="flex flex-col gap-1">
          <div className="mb-2 flex items-center justify-between border-b border-border">
            <p className="text-left text-2xs font-normal">Predefined Colors</p>
            <p className="text-2xs text-muted-foreground">
              Click to select a color
            </p>
          </div>
          <ColorGrid handleChange={handleChange} />
        </div>
        <Input
          id="custom"
          value={value || ""}
          className="col-span-2 mt-4 h-7"
          placeholder="Enter a custom color (e.g. #000000)"
          onChange={(e) => onChange(e.target.value)}
        />
      </PopoverContent>
    </Popover>
  );
});

export function ColorField<T extends FieldValues>({
  className,
  rules,
  description,
  label,
  name,
  control,
  disabled,
  ...props
}: ColorFieldProps<T>) {
  return (
    <Controller<T>
      name={name}
      control={control}
      rules={rules}
      render={({ field, fieldState }) => (
        <FieldWrapper
          label={label}
          description={description}
          required={!!rules?.required}
          error={fieldState.error?.message}
          className={className}
        >
          <ColorFieldInput
            field={field as ControllerRenderProps<FieldValues, string>}
            fieldState={fieldState}
            className={className}
            disabled={disabled}
            {...props}
          />
        </FieldWrapper>
      )}
    />
  );
}
