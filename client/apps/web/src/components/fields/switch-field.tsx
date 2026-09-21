import { cn } from "@trenova/shared/lib/utils";
import type { FormControlProps, WarningProps } from "@trenova/shared/types/fields";
import { Controller, type FieldValues } from "react-hook-form";
import type { SwitchProps } from "../animate-ui/components/base/switch";
import { RecommendedBadge } from "../recommended-badge";
import { Label } from "@trenova/shared/components/ui/label";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { Switch } from "@trenova/shared/components/ui/switch";
import { Tooltip, TooltipContent, TooltipTrigger } from "@trenova/shared/components/ui/tooltip";
import { InfoIcon } from "lucide-react";
import { fieldInvalidClass } from "@trenova/shared/lib/variants/field";

type BaseSwitchFieldProps = Omit<SwitchProps, "name"> & {
  label: string;
  description?: string | React.ReactNode;
  outlined?: boolean;
  position?: "left" | "right";
  switchInputClassName?: string;
  recommended?: boolean;
  readOnly?: boolean;
  warning?: WarningProps;
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
            "group relative flex w-full items-start gap-2 rounded-md border border-transparent p-2.5 transition-all duration-300 ease-in-out",
            outlined &&
              "border-input bg-field transition-[border-color,box-shadow,background-color] duration-200 ease-in-out has-data-checked:border-info has-data-checked:bg-info-subtle has-data-checked:text-info-foreground has-data-checked:ring-4 has-data-checked:ring-info/20 dark:has-data-checked:text-info-foreground",
            fieldState.error && fieldInvalidClass,
            warning?.show &&
              "ui-focus-ring border-warning bg-warning-subtle ring-0 ring-warning placeholder:text-warning-foreground focus:outline-hidden",
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
              {!recommended && tooltip && (
                <Tooltip>
                  <TooltipTrigger
                    render={
                      <button
                        type="button"
                        aria-label={`About ${label}`}
                        className="ui-focus-ring text-muted-foreground/70 hover:bg-muted hover:text-foreground inline-flex size-4 shrink-0 items-center justify-center rounded-sm transition-colors"
                      >
                        <InfoIcon className="size-3" />
                      </button>
                    }
                  />
                  <TooltipContent className="max-w-xs">{tooltip}</TooltipContent>
                </Tooltip>
              )}
            </div>
            {description && (
              <p
                id={descriptionId}
                className={cn(
                  "text-2xs text-muted-foreground",
                  outlined &&
                    "group-has-data-checked:text-info-foreground dark:group-has-data-checked:text-info-foreground",
                  fieldState.error && "text-danger-foreground",
                  warning?.show && "text-warning-foreground",
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
    <div className="group relative flex w-full items-start gap-2 rounded-md border border-transparent p-2.5">
      <Skeleton className="size-5" />
      <div className="grid grow gap-1">
        <Skeleton className="h-4 w-37.5" />
        <Skeleton className="h-4 w-50" />
      </div>
    </div>
  );
}
