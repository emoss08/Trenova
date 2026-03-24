import React, { useMemo } from "react";
import { Label } from "../ui/label";
import { cn } from "@/lib/utils";

export function ErrorMessage({ formError }: { formError?: string }) {
  return (
    <span
      role="alert"
      className="mt-1 inline-block rounded-md bg-red-50 px-2 py-1 text-left text-xs leading-tight text-destructive dark:bg-destructive/40 dark:text-red-50"
    >
      {formError ? formError : "An Error has occurred. Please try again."}
    </span>
  );
}

export function FieldDescription({
  description,
}: {
  description: string | React.ReactNode;
}) {
  if (!description) {
    return null;
  }

  if (React.isValidElement(description)) {
    return description;
  }

  return <p className="text-left text-2xs text-foreground/70">{description}</p>;
}

type FieldWrapperProps = {
  label?: string;
  description?: string;
  required?: boolean;
  className?: string;
  children: React.ReactNode;
  error?: string;
};

export function FieldLabel({
  label,
  required,
}: {
  label?: string;
  required?: boolean;
}) {
  return label ? (
    <Label className={cn("block text-xs font-medium", required && "required")}>
      {label}
    </Label>
  ) : null;
}

function FieldWrapperInner({ children }: { children: React.ReactNode }) {
  return <div className="mb-0.5 flex items-center">{children}</div>;
}

function FieldWrapperDescriptionInner({
  children,
}: {
  children: React.ReactNode;
}) {
  return <div className="flex justify-start">{children}</div>;
}

export function FieldWrapper({
  label,
  description,
  required,
  className,
  children,
  error,
}: FieldWrapperProps) {
  const descriptionElement = useMemo(() => {
    return description && !error ? (
      <FieldDescription description={description} />
    ) : null;
  }, [description, error]);

  const errorElement = useMemo(() => {
    return error ? <ErrorMessage formError={error} /> : null;
  }, [error]);

  return (
    <div className={cn("flex flex-col gap-0.5", className)}>
      {label && (
        <FieldWrapperInner>
          <FieldLabel label={label} required={required} />
        </FieldWrapperInner>
      )}
      {children}
      <FieldWrapperDescriptionInner>
        {descriptionElement}
        {errorElement}
      </FieldWrapperDescriptionInner>
    </div>
  );
}
