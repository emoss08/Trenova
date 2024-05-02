import * as CheckboxPrimitive from "@radix-ui/react-checkbox";
import * as React from "react";

import { cn } from "@/lib/utils";
import { faCheck } from "@fortawesome/pro-solid-svg-icons";
import { FontAwesomeIcon } from "@fortawesome/react-fontawesome";
import {
  Controller,
  FieldValues,
  useController,
  UseControllerProps,
} from "react-hook-form";
import { ErrorMessage } from "./error-message";

const Checkbox = React.forwardRef<
  React.ElementRef<typeof CheckboxPrimitive.Root>,
  React.ComponentPropsWithoutRef<typeof CheckboxPrimitive.Root>
>(({ className, ...props }, ref) => (
  <CheckboxPrimitive.Root
    ref={ref}
    className={cn(
      "peer h-4 w-4 shrink-0 rounded-sm border border-primary shadow focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-ring disabled:cursor-not-allowed disabled:opacity-50 data-[state=checked]:bg-primary data-[state=checked]:text-primary-foreground",
      className,
    )}
    {...props}
  >
    <CheckboxPrimitive.Indicator
      className={cn("flex items-center justify-center text-current")}
    >
      <FontAwesomeIcon icon={faCheck} className="size-3 font-bold" />
    </CheckboxPrimitive.Indicator>
  </CheckboxPrimitive.Root>
));
Checkbox.displayName = CheckboxPrimitive.Root.displayName;

export { Checkbox };

type CheckboxInputProps = CheckboxPrimitive.CheckboxProps &
  React.ComponentPropsWithoutRef<typeof CheckboxPrimitive.Root> & {
    formError?: string;
    description?: string;
    label?: string;
  };

export function CheckboxInput<T extends FieldValues>({
  ...props
}: CheckboxInputProps & UseControllerProps<T>) {
  const { fieldState } = useController(props);

  const { label, description, id, className } = props;

  return (
    <label
      htmlFor={id}
      className={cn("items-top flex cursor-pointer space-x-2", className)}
    >
      <Controller
        name={props.name}
        control={props.control}
        render={({ field }) => (
          <Checkbox
            {...field}
            onCheckedChange={(e) => {
              field.onChange(e);
            }}
            checked={field.value as boolean}
            id={id} // Ensure the checkbox has the same id as the label's htmlFor
          />
        )}
      />
      <div className="grid gap-1.5 leading-none">
        {label && (
          <span className="select-none text-sm font-medium leading-none peer-disabled:cursor-not-allowed peer-disabled:opacity-70">
            {label}
          </span>
        )}
        {description && (
          <p className="text-muted-foreground select-none text-wrap text-sm">
            {description}
          </p>
        )}
        {fieldState.invalid && (
          <ErrorMessage formError={fieldState.error?.message} />
        )}
      </div>
    </label>
  );
}
