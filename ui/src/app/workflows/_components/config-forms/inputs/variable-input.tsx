import { FieldWrapper } from "@/components/fields/field-components";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from "@/components/ui/popover";
import { ScrollArea } from "@/components/ui/scroll-area";
import { Separator } from "@/components/ui/separator";
import { cn } from "@/lib/utils";
import { VARIABLE_CATEGORIES } from "@/lib/workflow";
import { InputFieldProps } from "@/types/fields";
import { Braces } from "lucide-react";
import { Controller, FieldValues } from "react-hook-form";
import {
  CustomPathNotice,
  WorkflowVariableHeader,
} from "../config-help-components";
import { VariableCategoryItem } from "../variable-category-item";

export function VariableInput<T extends FieldValues>({
  name,
  control,
  rules,
  label,
  description,
  className,
  type = "text",
  disabled,
  autoComplete,
  placeholder,
  "aria-label": ariaLabel,
  "aria-describedby": ariaDescribedBy,
}: InputFieldProps<T>) {
  const inputId = `input-${name}`;
  const descriptionId = `${inputId}-description`;
  const errorId = `${inputId}-error`;

  return (
    <Controller
      name={name}
      control={control}
      rules={rules}
      render={({ field, fieldState }) => (
        <FieldWrapper
          label={label}
          required={!!rules?.required}
          error={fieldState.error?.message}
          description={description}
          className={className}
        >
          <div className="flex gap-2">
            <Input
              type={type}
              value={field.value || ""}
              isInvalid={fieldState.invalid}
              onChange={field.onChange}
              placeholder={placeholder}
              className="flex-1 font-mono text-sm"
              disabled={disabled}
              autoComplete={autoComplete}
              aria-label={ariaLabel || label}
              aria-describedby={cn(
                description && descriptionId,
                fieldState.error && errorId,
                ariaDescribedBy,
              )}
            />
            <Popover>
              <PopoverTrigger asChild>
                <Button
                  type="button"
                  variant="outline"
                  className="flex size-7 items-center justify-center [&_svg]:size-3.5"
                  size="icon"
                  title="Insert variable"
                >
                  <Braces className="size-3.5" />
                </Button>
              </PopoverTrigger>
              <PopoverContent
                className="w-96 p-0"
                side="right"
                sideOffset={20}
                align="center"
              >
                <div>
                  <WorkflowVariableHeader />
                  <Separator />
                  <ScrollArea className="h-80 pr-2">
                    {VARIABLE_CATEGORIES.map((category) => (
                      <VariableCategoryItem
                        key={category.label}
                        category={category}
                        onChange={field.onChange}
                      />
                    ))}
                  </ScrollArea>
                  <Separator />
                  <CustomPathNotice />
                </div>
              </PopoverContent>
            </Popover>
          </div>
        </FieldWrapper>
      )}
    />
  );
}
