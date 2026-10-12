import { useT } from "@trenova/shared/i18n/use-t";
import type React from "react";
import { cn } from "@trenova/shared/lib/utils";
import { fieldInvalidClass } from "@trenova/shared/lib/variants/field";
import type { FormControlProps } from "@trenova/shared/types/fields";
import { ChevronDownIcon } from "@trenova/shared/components/icons";
import { Controller, type FieldValues } from "react-hook-form";
import { Button } from "@trenova/shared/components/ui/button";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@trenova/shared/components/ui/dropdown-menu";
import { Textarea, type TextareaProps } from "@trenova/shared/components/ui/textarea";
import { FieldWrapper } from "./field-components";

export type TextareaPreset = {
  id: string;
  label: string;
  description: string;
};

type BaseTextareaFieldProps = Omit<TextareaProps, "name" | "bare"> & {
  label: string;
  description?: string;
  presets?: TextareaPreset[];
  /**
   * Drawn inside the field's frame under the text, such as suggestions and a send
   * button; the frame then carries the field's border, fill, focus and invalid state.
   */
  footer?: React.ReactNode;
  /** "lg" for a field that is the page's main input: larger text and more room. */
  size?: "default" | "lg";
};

const LARGE_TEXT = "px-5 pt-4.5 pb-2 text-lg leading-relaxed md:text-lg";
export type TextareaFieldProps<T extends FieldValues> = BaseTextareaFieldProps &
  FormControlProps<T>;

export function TextareaField<T extends FieldValues>({
  label,
  description,
  name,
  control,
  rules,
  className,
  disabled,
  autoComplete,
  placeholder,
  presets,
  footer,
  size = "default",
  "aria-label": ariaLabel,
  "aria-describedby": ariaDescribedBy,
  ...props
}: TextareaFieldProps<T>) {
  const t = useT();

  const inputId = `textarea-${name}`;
  const descriptionId = `${inputId}-description`;
  const errorId = `${inputId}-error`;
  const hasPresets = presets && presets.length > 0;

  return (
    <Controller<T>
      name={name}
      control={control}
      rules={rules}
      render={({ field, fieldState }) => {
        const textarea = (
          <Textarea
            {...field}
            {...props}
            id={inputId}
            className={cn(hasPresets && "pb-5", size === "lg" && LARGE_TEXT, className)}
            bare={footer !== undefined}
            disabled={disabled}
            minRows={3}
            autoComplete={autoComplete}
            placeholder={placeholder}
            aria-label={ariaLabel || label}
            isInvalid={fieldState.invalid}
            aria-describedby={cn(
              description && descriptionId,
              fieldState.error && errorId,
              ariaDescribedBy,
            )}
          />
        );

        return (
          <FieldWrapper
            name={name}
            label={label}
            description={description}
            required={!!rules?.required}
            error={fieldState.error?.message}
          >
            {footer !== undefined ? (
              <div
                className={cn(
                  "ui-field ui-container-focus-ring flex flex-col",
                  size === "lg" && "rounded-xl",
                  fieldState.invalid && fieldInvalidClass,
                  disabled && "opacity-60",
                )}
              >
                {textarea}
                <div
                  className={cn(
                    "flex items-center gap-2.5",
                    size === "lg" ? "px-3.5 pt-1 pb-2.5" : "px-2 pb-1.5",
                  )}
                >
                  {footer}
                </div>
              </div>
            ) : hasPresets ? (
              <div className="relative">
                {textarea}
                <div className="absolute right-2.5 bottom-0.5">
                  <DropdownMenu>
                    <DropdownMenuTrigger
                      render={
                        <Button
                          title={t("Select a preset")}
                          variant="ghost"
                          className="text-2xs hover:bg-background h-5 w-16 gap-1"
                        >
                          {t("Preset")} <ChevronDownIcon />
                        </Button>
                      }
                      className="outline-none"
                    />
                    <DropdownMenuContent align="end" className="w-60">
                      {presets.map((preset) => (
                        <DropdownMenuItem
                          key={preset.id}
                          onClick={() => field.onChange(t(preset.description))}
                          className="flex flex-col items-start gap-1 py-2"
                          title={t(preset.label)}
                          description={t(preset.description)}
                        />
                      ))}
                    </DropdownMenuContent>
                  </DropdownMenu>
                </div>
              </div>
            ) : (
              textarea
            )}
          </FieldWrapper>
        );
      }}
    />
  );
}
