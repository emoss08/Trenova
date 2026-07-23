import { cn } from "@/lib/utils";
import type { FormControlProps } from "@/types/fields";
import { Controller, type FieldValues } from "react-hook-form";
import { Checkbox } from "../animate-ui/components/base/checkbox";
import type { CheckboxProps } from "../animate-ui/primitives/base/checkbox";
import { Label } from "../ui/label";

type BaseCheckboxFieldProps = Omit<CheckboxProps, "name"> & {
  label: string;
  outlined?: boolean;
  description?: string;
};

export type CheckboxFieldProps<T extends FieldValues> = BaseCheckboxFieldProps &
  FormControlProps<T>;

export function CheckboxField<T extends FieldValues>({
  label,
  description,
  name,
  control,
  rules,
  outlined,
  "aria-describedby": ariaDescribedBy,
  ...props
}: CheckboxFieldProps<T>) {
  const inputId = `checkbox-${name}`;
  const descriptionId = `${inputId}-description`;
  const errorId = `${inputId}-error`;

  return (
    <Controller
      name={name}
      control={control}
      rules={rules}
      render={({
        field: { value, onChange, disabled, onBlur, name, ref },
        fieldState,
      }) => (
        <div
          className={cn(
            "relative flex w-full items-start gap-2 rounded-md p-2.5",
            outlined &&
              "border border-muted-foreground/20 bg-muted transition-[border-color,box-shadow] duration-200 ease-in-out has-data-[state=checked]:border-foreground has-data-[state=checked]:ring-4 has-data-[state=checked]:ring-foreground/20",
          )}
        >
          <Checkbox
            id={inputId}
            aria-describedby={cn(
              description && descriptionId,
              fieldState.error && errorId,
              ariaDescribedBy,
            )}
            ref={ref}
            name={name}
            onBlur={onBlur}
            checked={value}
            onCheckedChange={onChange}
            disabled={disabled}
            className="order-1 after:absolute after:inset-0"
            onClick={(e) => e.stopPropagation()}
            {...props}
          />
          <div className="grid grow gap-2">
            <Label htmlFor={inputId}>{label}</Label>
            <p
              id={`${inputId}-description`}
              className="text-2xs text-muted-foreground"
            >
              {description}
            </p>
          </div>
        </div>
      )}
    />
  );
}
