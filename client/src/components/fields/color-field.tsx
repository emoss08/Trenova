import { Button } from "@/components/ui/button";
import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from "@/components/ui/popover";
import { cn } from "@/lib/utils";
import type { FormControlProps } from "@/types/fields";
import { Paintbrush } from "lucide-react";
import { useCallback, useState } from "react";
import {
  Controller,
  type ControllerFieldState,
  type ControllerRenderProps,
  type FieldValues,
  type Path,
} from "react-hook-form";
import { Input } from "../ui/input";
import { FieldWrapper } from "./field-components";

export type ColorFieldProps<TFieldValues extends FieldValues> = {
  hideHeader?: boolean;
  description?: string;
  label?: string;
  className?: string;
  disabled?: boolean;
  autoWidth?: boolean;
} & FormControlProps<TFieldValues>;

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

function ColorGrid({
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
}

function ColorFieldInput<T extends FieldValues>({
  field,
  fieldState,
  className,
  disabled,
  autoWidth = false,
  hideHeader = false,
}: {
  field: ControllerRenderProps<T, Path<T>>;
  fieldState: ControllerFieldState;
  autoWidth?: boolean;
  className?: string;
  disabled?: boolean;
  hideHeader?: boolean;
}) {
  const [isOpen, setIsOpen] = useState(false);
  const { onChange, value, ...restField } = field;

  const handleChange = useCallback(
    (color: string) => {
      setIsOpen(false);
      onChange(color);
    },
    [onChange],
  );

  return (
    <Popover open={isOpen} onOpenChange={setIsOpen}>
      <PopoverTrigger
        render={
          <Button
            size="sm"
            {...restField}
            variant="outline"
            type="button"
            className={cn(
              "w-full items-center justify-start gap-2 rounded border-input bg-muted px-1.5 text-left font-normal [&_svg]:size-3 [&_svg]:shrink-0",
              "data-pressed:border-brand data-pressed:ring-4 data-pressed:ring-brand/20 data-pressed:outline-hidden",
              "transition-[border-color,box-shadow] duration-200 ease-in-out",
              disabled && "cursor-not-allowed opacity-50",
              fieldState.invalid &&
                "border-destructive bg-destructive/20 text-destructive ring-0 ring-destructive hover:border-destructive hover:bg-destructive/20 focus:outline-hidden focus-visible:border-destructive focus-visible:ring-4 focus-visible:ring-destructive/20 data-pressed:border-destructive data-pressed:bg-destructive/20 data-pressed:ring-destructive/20",
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
                <Paintbrush className="size-4" />
              )}
              <div className="flex-1 truncate">
                {value ? value : "Pick a color"}
              </div>
            </div>
          </Button>
        }
      />
      <PopoverContent
        className={cn("p-2", autoWidth ? "w-52" : "w-(--anchor-width)")}
      >
        <div className="flex flex-col gap-1">
          {!hideHeader && (
            <div className="mb-2 flex items-center justify-between border-b border-border">
              <p className="text-left text-2xs font-normal">
                Predefined Colors
              </p>
              <p className="text-2xs text-muted-foreground">
                Click to select a color
              </p>
            </div>
          )}
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
}

export function ColorField<T extends FieldValues>({
  hideHeader = false,
  className,
  rules,
  description,
  label,
  name,
  control,
  disabled,
  autoWidth = false,
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
            autoWidth={autoWidth}
            hideHeader={hideHeader}
            {...props}
          />
        </FieldWrapper>
      )}
    />
  );
}
