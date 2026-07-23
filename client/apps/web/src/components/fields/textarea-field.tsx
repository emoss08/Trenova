import { cn } from "@/lib/utils";
import type { FormControlProps } from "@/types/fields";
import { ChevronDownIcon } from "lucide-react";
import { Controller, type FieldValues } from "react-hook-form";
import { Button } from "../ui/button";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "../ui/dropdown-menu";
import { Textarea, type TextareaProps } from "../ui/textarea";
import { FieldWrapper } from "./field-components";

export type TextareaPreset = {
  id: string;
  label: string;
  description: string;
};

type BaseTextareaFieldProps = Omit<TextareaProps, "name"> & {
  label: string;
  description?: string;
  presets?: TextareaPreset[];
};
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
  "aria-label": ariaLabel,
  "aria-describedby": ariaDescribedBy,
  ...props
}: TextareaFieldProps<T>) {
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
            className={cn(hasPresets && "pb-5", className)}
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
            label={label}
            description={description}
            required={!!rules?.required}
            error={fieldState.error?.message}
          >
            {hasPresets ? (
              <div className="relative">
                {textarea}
                <div className="absolute right-2.5 bottom-0.5">
                  <DropdownMenu>
                    <DropdownMenuTrigger
                      render={
                        <Button
                          title="Select a preset"
                          variant="ghost"
                          className="h-5 w-16 gap-1 text-2xs hover:bg-background"
                        >
                          Preset <ChevronDownIcon />
                        </Button>
                      }
                      className="outline-none"
                    />
                    <DropdownMenuContent align="end" className="w-60">
                      {presets.map((preset) => (
                        <DropdownMenuItem
                          key={preset.id}
                          onClick={() => field.onChange(preset.description)}
                          className="flex flex-col items-start gap-1 py-2"
                          title={preset.label}
                          description={preset.description}
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
