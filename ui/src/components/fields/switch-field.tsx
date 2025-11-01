import { cn } from "@/lib/utils";
import { type SwitchFieldProps } from "@/types/fields";
import { Controller, FieldValues } from "react-hook-form";
import { Label } from "../ui/label";
import RecommendedBadge from "../ui/recommended-badge";
import { Switch } from "../ui/switch";

export function SwitchField<T extends FieldValues>({
  label,
  description,
  name,
  control,
  rules,
  outlined,
  recommended,
  readOnly,
  position = "right",
  "aria-describedby": ariaDescribedBy,
  ...props
}: SwitchFieldProps<T>) {
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
            "relative flex w-full items-start gap-2 rounded-md p-3",
            outlined &&
              "border border-muted-foreground/20 has-data-[state=checked]:border-foreground has-data-[state=checked]:ring-4 has-data-[state=checked]:ring-foreground/20 bg-primary/5 transition-[border-color,box-shadow] duration-200 ease-in-out",
          )}
        >
          {position === "left" && (
            <Switch
              readOnly={readOnly}
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
              className="after:absolute after:inset-0"
              onClick={(e) => e.stopPropagation()}
              {...props}
            />
          )}

          <div
            className={cn(
              "grid grow gap-2",
              position === "left" ? "order-1" : "order-0",
            )}
          >
            <div className="flex items-center gap-2">
              <Label htmlFor={inputId}>{label}</Label>
              {recommended && <RecommendedBadge size="sm" variant="warning" />}
            </div>
            {description && (
              <p id={descriptionId} className="text-2xs text-muted-foreground">
                {description}
              </p>
            )}
          </div>

          {position === "right" && (
            <Switch
              readOnly={readOnly}
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
              className="after:absolute after:inset-0"
              onClick={(e) => e.stopPropagation()}
              {...props}
            />
          )}

          {fieldState.error && (
            <p
              id={errorId}
              className="text-2xs text-destructive absolute -bottom-5 left-0"
            >
              {fieldState.error.message}
            </p>
          )}
        </div>
      )}
    />
  );
}
