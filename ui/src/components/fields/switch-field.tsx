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
  warning,
  tooltip,
  className,
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
            "relative flex w-full items-start gap-2 rounded-md p-3 group border border-transparent",
            outlined &&
              "border-muted-foreground/20 bg-primary/5 has-data-[state=checked]:border-blue-600 has-data-[state=checked]:ring-4 has-data-[state=checked]:bg-blue-600/10 has-data-[state=checked]:text-blue-500 dark:has-data-[state=checked]:text-blue-400 has-data-[state=checked]:ring-blue-600/20 transition-[border-color,box-shadow,background-color] duration-200 ease-in-out",
            fieldState.error &&
              "border-red-500 bg-red-500/20 ring-0 ring-red-500 placeholder:text-red-500 focus:outline-hidden focus-visible:border-red-600 focus-visible:ring-4 focus-visible:ring-red-400/20",
            warning?.show &&
              "border-amber-500 bg-amber-500/10 ring-0 ring-amber-500 placeholder:text-amber-600 focus:outline-hidden focus-visible:border-amber-600 focus-visible:ring-4 focus-visible:ring-amber-400/20",
            className,
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
              "grid grow gap-1",
              position === "left" ? "order-1" : "order-0",
            )}
          >
            <div className="flex items-center gap-2">
              <Label htmlFor={inputId}>{label}</Label>
              {recommended && (
                <RecommendedBadge
                  size="sm"
                  variant="warning"
                  tooltip={tooltip}
                />
              )}
            </div>
            {description && (
              <p
                id={descriptionId}
                className={cn(
                  "text-2xs text-muted-foreground",
                  outlined &&
                    "group-has-data-[state=checked]:text-blue-500 dark:group-has-data-[state=checked]:text-blue-400",
                  fieldState.error && "text-red-500",
                  warning?.show && "text-amber-600",
                )}
              >
                {fieldState.error
                  ? fieldState.error.message
                  : warning?.show
                    ? warning.message
                    : description}
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
        </div>
      )}
    />
  );
}
