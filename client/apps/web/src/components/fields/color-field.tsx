import { isRequired, withRequired } from "@trenova/shared/lib/field-rules";
import { useT } from "@trenova/shared/i18n/use-t";
import { Button } from "@trenova/shared/components/ui/button";
import { Popover, PopoverContent, PopoverTrigger } from "@trenova/shared/components/ui/popover";
import { cn } from "@trenova/shared/lib/utils";
import type { FormControlProps } from "@trenova/shared/types/fields";
import { Brush01Icon } from "@trenova/shared/components/icons";
import { useCallback, useState } from "react";
import {
  Controller,
  type ControllerFieldState,
  type ControllerRenderProps,
  type FieldValues,
  type Path,
} from "react-hook-form";
import { Input } from "@trenova/shared/components/ui/input";
import { ColorSwatchGrid } from "@/components/color-swatches";
import { FieldWrapper } from "./field-components";
import { fieldInvalidClass, fieldTriggerClass } from "@trenova/shared/lib/variants/field";

export type ColorFieldProps<TFieldValues extends FieldValues> = {
  hideHeader?: boolean;
  description?: React.ReactNode;
  label?: string;
  className?: string;
  disabled?: boolean;
  autoWidth?: boolean;
} & FormControlProps<TFieldValues>;

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
  const t = useT();

  const [isOpen, setIsOpen] = useState(false);
  const { onChange, value, ...restField } = field;

  const handleOpenChange = useCallback(
    (open: boolean) => {
      if (!disabled) {
        setIsOpen(open);
      }
    },
    [disabled],
  );

  const handleChange = useCallback(
    (color: string) => {
      setIsOpen(false);
      onChange(color);
    },
    [onChange],
  );

  return (
    <Popover open={isOpen} onOpenChange={handleOpenChange}>
      <PopoverTrigger
        render={
          <Button
            size="sm"
            {...restField}
            variant="outline"
            type="button"
            className={cn(
              fieldTriggerClass,
              "w-full items-center justify-start gap-2 px-1.5 text-left font-normal [&_svg]:size-3 [&_svg]:shrink-0",
              disabled && "cursor-not-allowed opacity-50",
              fieldState.invalid && fieldInvalidClass,
              !value && "text-muted-foreground",
              className,
            )}
          >
            <div className="flex w-full items-center gap-1">
              {value ? (
                <div
                  className="size-4 rounded-md bg-cover! bg-center! transition-all"
                  style={{ background: value }}
                />
              ) : (
                <Brush01Icon className="size-4" />
              )}
              <div className="mt-0.5 flex-1 truncate text-left">
                {value ? value : t("Pick a color")}
              </div>
            </div>
          </Button>
        }
      />
      <PopoverContent className={cn("p-2", autoWidth ? "w-52" : "w-(--anchor-width)")}>
        <div className="flex flex-col gap-1">
          {!hideHeader && (
            <div className="border-border mb-2 flex items-center justify-between border-b">
              <p className="text-2xs text-left font-normal">{t("Predefined colors")}</p>
              <p className="text-2xs text-muted-foreground">{t("Click to select a color")}</p>
            </div>
          )}
          <ColorSwatchGrid value={value} onPick={handleChange} />
        </div>
        <Input
          id="custom"
          value={value || ""}
          className="col-span-2 mt-4 h-6"
          placeholder={t("Enter a custom color (e.g. #000000)")}
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
  required,
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
      rules={withRequired(rules, required)}
      render={({ field, fieldState }) => (
        <FieldWrapper
          name={name}
          label={label}
          description={description}
          required={isRequired(rules, required)}
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
