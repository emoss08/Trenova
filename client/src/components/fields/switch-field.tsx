import { cn } from "@/lib/utils";
import type { FormControlProps } from "@/types/fields";
import { Controller, type FieldValues } from "react-hook-form";
import type { SwitchProps } from "../animate-ui/components/base/switch";
import { RecommendedBadge } from "../recommended-badge";
import { Label } from "../ui/label";
import { Skeleton } from "../ui/skeleton";
import { Switch } from "../ui/switch";

type BaseSwitchFieldProps = Omit<SwitchProps, "name"> & {
  label: string;
  description?: string | React.ReactNode;
  outlined?: boolean;
  position?: "left" | "right";
  switchInputClassName?: string;
  recommended?: boolean;
  readOnly?: boolean;
  warning?: {
    show: boolean;
    message: string;
  };
  tooltip?: React.ReactNode;
};

export type SwitchFieldProps<T extends FieldValues> = BaseSwitchFieldProps & FormControlProps<T>;

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
  const inputId = `switch-${name}`;
  const descriptionId = `${inputId}-description`;
  const errorId = `${inputId}-error`;

  return (
    <Controller
      name={name}
      control={control}
      rules={rules}
      render={({ field: { value, onChange, disabled, onBlur, name, ref }, fieldState }) => (
        <div
          className={cn(
            "group relative flex w-full items-start gap-2 rounded-md border border-transparent p-3",
            outlined &&
              "border-input bg-muted transition-[border-color,box-shadow,background-color] duration-200 ease-in-out has-data-checked:border-blue-600 has-data-checked:bg-blue-600/10 has-data-checked:text-blue-500 has-data-checked:ring-4 has-data-checked:ring-blue-600/20 dark:has-data-checked:text-blue-400",
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

          <div className={cn("grid grow gap-1", position === "left" ? "order-1" : "order-0")}>
            <div className="flex items-center gap-2">
              <Label htmlFor={inputId}>{label}</Label>
              {recommended && <RecommendedBadge size="sm" variant="warning" tooltip={tooltip} />}
            </div>
            {description && (
              <p
                id={descriptionId}
                className={cn(
                  "text-2xs text-muted-foreground",
                  outlined &&
                    "group-has-data-checked:text-blue-500 dark:group-has-data-checked:text-blue-400",
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

export function SwitchFieldSkeleton() {
  return (
    <div className="group relative flex w-full items-start gap-2 rounded-md border border-transparent p-3">
      <Skeleton className="size-5" />
      <div className="grid grow gap-1">
        <Skeleton className="h-4 w-[150px]" />
        <Skeleton className="h-4 w-[200px]" />
      </div>
    </div>
  );
}
