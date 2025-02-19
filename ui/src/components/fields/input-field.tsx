import { cn } from "@/lib/utils";
import { InputFieldProps } from "@/types/fields";
import { faPencil } from "@fortawesome/pro-regular-svg-icons";
import { useEffect, useRef, useState } from "react";
import {
  Control,
  Controller,
  FieldValues,
  Path,
  useController,
} from "react-hook-form";
import { Icon } from "../ui/icons";
import { Input } from "../ui/input";
import { FieldWrapper } from "./field-components";

export function InputField<T extends FieldValues>({
  label,
  description,
  icon,
  name,
  control,
  rules,
  className,
  type = "text",
  disabled,
  autoComplete,
  placeholder,
  inputClassProps,
  "aria-label": ariaLabel,
  "aria-describedby": ariaDescribedBy,
  readOnly,
  ...props
}: InputFieldProps<T>) {
  const inputId = `input-${name}`;
  const descriptionId = `${inputId}-description`;
  const errorId = `${inputId}-error`;

  return (
    <Controller<T>
      name={name}
      control={control}
      rules={rules}
      render={({ field, fieldState }) => (
        <FieldWrapper
          label={label}
          description={description}
          required={!!rules?.required}
          error={fieldState.error?.message}
          className={className}
        >
          <Input
            {...field}
            {...props}
            id={inputId}
            type={type}
            readOnly={readOnly}
            disabled={disabled}
            autoComplete={autoComplete}
            placeholder={placeholder}
            aria-label={ariaLabel || label}
            className={inputClassProps}
            isInvalid={fieldState.invalid}
            icon={icon}
            aria-describedby={cn(
              description && descriptionId,
              fieldState.error && errorId,
              ariaDescribedBy,
            )}
          />
        </FieldWrapper>
      )}
    />
  );
}

interface DoubleClickInputProps<T extends Record<string, any>> {
  control: Control<T>;
  name: Path<T>;
  className?: string;
  inputClassName?: string;
  displayClassName?: string;
}

export function DoubleClickInput<T extends Record<string, any>>({
  control,
  name,
  className,
  inputClassName,
  displayClassName,
}: DoubleClickInputProps<T>) {
  const { field } = useController({ control, name });
  const [isEditing, setIsEditing] = useState(false);
  const [isHovered, setIsHovered] = useState(false);
  const inputRef = useRef<HTMLInputElement>(null);

  useEffect(() => {
    if (isEditing && inputRef.current) {
      inputRef.current.focus();
      inputRef.current.select();
    }
  }, [isEditing]);

  const handleEditClick = () => {
    setIsEditing(true);
  };

  const handleBlur = () => {
    setIsEditing(false);
  };

  const handleKeyDown = (e: React.KeyboardEvent<HTMLInputElement>) => {
    if (e.key === "Enter" || e.key === "Escape") {
      setIsEditing(false);
    }
  };

  return (
    <div
      className={cn(
        "relative flex items-center max-h-[inherit] group",
        className,
      )}
      onMouseEnter={() => setIsHovered(true)}
      onMouseLeave={() => setIsHovered(false)}
    >
      {isEditing ? (
        <input
          {...field}
          ref={inputRef}
          className={cn(
            "outline-none h-5 border-muted-foreground/20 flex w-full rounded-md border px-2 py-1 text-sm",
            "placeholder:text-muted-foreground",
            "disabled:cursor-not-allowed disabled:opacity-50",
            "focus-visible:border-blue-600 focus-visible:outline-hidden focus-visible:ring-4 focus-visible:ring-blue-600/20",
            "transition-[border-color,box-shadow] duration-200 ease-in-out",
            inputClassName,
          )}
          onBlur={handleBlur}
          onKeyDown={handleKeyDown}
        />
      ) : (
        <div className="flex items-center w-full gap-x-1.5">
          <span
            className={cn(
              "text-sm flex-grow text-blue-500 underline",
              displayClassName,
            )}
            role="textbox"
            tabIndex={0}
            onKeyDown={(e) => {
              if (e.key === "Enter") {
                handleEditClick();
              }
            }}
          >
            {field.value || "Click to edit"}
          </span>
          <button
            onClick={handleEditClick}
            className={cn(
              "transition-opacity duration-200 cursor-pointer",
              isHovered ? "opacity-100" : "opacity-0",
            )}
            aria-label="Edit"
          >
            <Icon icon={faPencil} className="size-3 text-muted-foreground" />
          </button>
        </div>
      )}
    </div>
  );
}
