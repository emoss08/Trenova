import { Button } from "@/components/ui/button";
import { cn } from "@/lib/utils";
import { faEye, faEyeSlash } from "@fortawesome/pro-regular-svg-icons";
import { FontAwesomeIcon } from "@fortawesome/react-fontawesome";
import * as React from "react";
import {
  Controller,
  FieldValues,
  UseControllerProps,
  useController,
} from "react-hook-form";
import { FieldDescription } from "./components";
import { FieldErrorMessage } from "./error-message";
import { Label } from "./label";

export interface InputProps
  extends React.InputHTMLAttributes<HTMLInputElement> {}

const Input = React.forwardRef<HTMLInputElement, InputProps>(
  ({ className, type, ...props }, ref) => {
    return (
      <input
        type={type}
        className={cn(
          "flex h-9 w-full rounded-md border border-border bg-background px-3 py-2 text-sm file:border-0 file:pb-5 file:bg-transparent file:text-sm file:font-medium file:text-foreground placeholder:text-muted-foreground focus-visible:outline-none focus:ring-1 focus:ring-inset focus:ring-foreground disabled:cursor-not-allowed disabled:opacity-50 read-only:cursor-not-allowed read-only:opacity-50 sm:text-sm sm:leading-6",
          className,
        )}
        ref={ref}
        {...props}
      />
    );
  },
);
Input.displayName = "Input";

export { Input };

export type ExtendedInputProps = Omit<InputProps, "name"> & {
  description?: string;
  label?: string;
  ref?: React.ForwardedRef<HTMLInputElement>;
  icon?: React.ReactNode;
};

export function InputField<T extends FieldValues>({
  icon,
  ...props
}: ExtendedInputProps & UseControllerProps<T>) {
  const { fieldState } = useController(props);

  const { rules, label, name, control, className, description } = props;

  return (
    <>
      <span className="space-x-1">
        {label && <Label className="text-sm font-medium">{label}</Label>}
        {rules?.required && <span className="text-red-500">*</span>}
      </span>
      <div className="relative">
        {icon && (
          <div className="pointer-events-none absolute inset-y-0 left-0 flex items-center pl-3">
            {icon}
          </div>
        )}
        <Controller
          name={name}
          control={control}
          render={({ field }) => (
            <Input
              {...field}
              className={cn(
                icon && "pl-10",
                fieldState.invalid &&
                  "ring-1 ring-inset ring-red-500 placeholder:text-red-500 focus:ring-red-500 bg-red-500 bg-opacity-20",
                className,
              )}
              {...props}
            />
          )}
        />
        {fieldState.invalid && (
          <FieldErrorMessage formError={fieldState.error?.message} />
        )}
        {description && !fieldState.invalid && (
          <FieldDescription description={description} />
        )}
      </div>
    </>
  );
}

export function FileField<T extends FieldValues>({
  ...props
}: ExtendedInputProps & UseControllerProps<T>) {
  const { field, fieldState } = useController(props);

  return (
    <>
      {props.label && (
        <Label
          className={cn(
            "text-sm font-medium",
            props.rules?.required && "required",
          )}
        >
          {props.label}
        </Label>
      )}
      <div className="relative">
        <Input
          type="file"
          className={cn(
            "pr-10",
            fieldState.invalid &&
              "ring-1 ring-inset ring-red-500 placeholder:text-red-500 focus:ring-red-500 bg-red-500 bg-opacity-20",
            props.className,
          )}
          onChange={(e) => {
            const value = e.target.files;
            if (value) {
              field.onChange(value);
            }
          }}
          {...props}
        />
        {fieldState.invalid && (
          <FieldErrorMessage formError={fieldState.error?.message} />
        )}
        {props.description && !fieldState.invalid && (
          <FieldDescription description={props.description} />
        )}
      </div>
    </>
  );
}

export function TimeField<T extends FieldValues>({
  ...props
}: ExtendedInputProps & UseControllerProps<T>) {
  const { field, fieldState } = useController(props);

  return (
    <>
      {props.label && (
        <Label
          className={cn(
            "text-sm font-medium",
            props.rules?.required && "required",
          )}
        >
          {props.label}
        </Label>
      )}
      <div className="relative">
        <Input
          type="time"
          step="1" // Include this to allow seconds in the format HH:MM:SS
          className={cn(
            fieldState.invalid &&
              "ring-1 ring-inset ring-red-500 placeholder:text-red-500 focus:ring-red-500 bg-red-500 bg-opacity-20",
            props.className,
          )}
          {...field}
          {...props}
        />
        {fieldState.invalid && (
          <FieldErrorMessage formError={fieldState.error?.message} />
        )}
        {props.description && !fieldState.invalid && (
          <FieldDescription description={props.description} />
        )}
      </div>
    </>
  );
}

export function PasswordField<T extends FieldValues>({
  icon,
  ...props
}: ExtendedInputProps & UseControllerProps<T>) {
  const { field, fieldState } = useController(props);
  const [showPassword, setShowPassword] = React.useState(false);

  const togglePasswordVisibility = () => {
    setShowPassword((prevState) => {
      return !prevState;
    });
  };

  return (
    <>
      {props.label && (
        <Label
          className={cn(
            "text-sm font-medium",
            props.rules?.required && "required",
          )}
          htmlFor={props.id}
        >
          {props.label}
        </Label>
      )}
      <div className="relative">
        <div className="relative">
          {icon && (
            <div className="pointer-events-none absolute inset-y-0 left-0 flex items-center pl-3">
              {icon}
            </div>
          )}
          <Input
            {...field}
            className={cn(
              icon && "pl-10",
              fieldState.invalid &&
                "ring-1 ring-inset ring-red-500 placeholder:text-red-500 focus:ring-red-500 bg-red-500 bg-opacity-20",
              props.className,
            )}
            type={showPassword ? "text" : "password"}
            {...props}
          />
          {field.value && !fieldState.invalid && (
            <Button
              type="button"
              size="icon"
              variant="ghost"
              className="absolute right-1 top-1/2 size-6 -translate-y-1/2 rounded-md"
              onClick={togglePasswordVisibility}
            >
              {showPassword ? (
                <FontAwesomeIcon icon={faEyeSlash} className="size-4" />
              ) : (
                <FontAwesomeIcon icon={faEye} className="size-4" />
              )}
            </Button>
          )}
        </div>
        {fieldState.invalid && (
          <FieldErrorMessage formError={fieldState.error?.message} />
        )}
        {props.description && !fieldState.invalid && (
          <FieldDescription description={props.description} />
        )}
      </div>
    </>
  );
}
