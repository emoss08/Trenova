import { cn } from "@/lib/utils";
import type { WarningProps } from "@/types/fields";
import React, { useMemo } from "react";
import { Label } from "../ui/label";

export function ErrorMessage({ formError, id }: { formError?: string; id?: string }) {
  return (
    <span
      id={id}
      role="alert"
      className="mt-1 inline-block rounded-md bg-red-50 px-2 py-1 text-left text-xs leading-tight text-destructive dark:bg-destructive/40 dark:text-red-50"
    >
      {formError ? formError : "An Error has occurred. Please try again."}
    </span>
  );
}

export function FieldDescription({
  description,
  warning,
  id,
}: {
  description?: string | React.ReactNode;
  warning?: WarningProps;
  id?: string;
}) {
  if (warning?.show) {
    return (
      <p id={id} className="text-left text-2xs text-amber-600">
        {warning.message}
      </p>
    );
  }

  if (!description) {
    return null;
  }

  if (React.isValidElement(description)) {
    return description;
  }

  return (
    <p id={id} className="text-left text-2xs text-foreground/70">
      {description}
    </p>
  );
}

type FieldWrapperProps = {
  label?: React.ReactNode;
  description?: string | React.ReactNode;
  warning?: WarningProps;
  required?: boolean;
  className?: string;
  children: React.ReactNode;
  error?: string;
  descriptionId?: string;
  errorId?: string;
};

export function FieldLabel({ label, required }: { label?: React.ReactNode; required?: boolean }) {
  if (!label) {
    return null;
  }

  if (React.isValidElement(label)) {
    return (
      <div className={cn("block text-xs font-medium", required && "required")}>{label}</div>
    );
  }

  return <Label className={cn("block text-xs font-medium", required && "required")}>{label}</Label>;
}

function FieldWrapperInner({ children }: { children: React.ReactNode }) {
  return <div className="mb-0.5 flex items-center">{children}</div>;
}

function FieldWrapperDescriptionInner({ children }: { children: React.ReactNode }) {
  return <div className="flex justify-start">{children}</div>;
}

export function FieldWrapper({
  label,
  description,
  warning,
  required,
  className,
  children,
  error,
  descriptionId,
  errorId,
}: FieldWrapperProps) {
  const descriptionElement = useMemo(() => {
    return !error && (description || warning?.show) ? (
      <FieldDescription description={description} warning={warning} id={descriptionId} />
    ) : null;
  }, [description, descriptionId, error, warning]);

  const errorElement = useMemo(() => {
    return error ? <ErrorMessage formError={error} id={errorId} /> : null;
  }, [error, errorId]);

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
