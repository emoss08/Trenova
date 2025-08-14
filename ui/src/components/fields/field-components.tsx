/*
 * Copyright 2023-2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

import { Label } from "@/components/ui/label";
import { cn } from "@/lib/utils";
import React, { memo, useMemo } from "react";
import { Button } from "../ui/button";

const ErrorMessage = memo(
  function ErrorMessage({ formError }: { formError?: string }) {
    return (
      <span
        role="alert"
        className="mt-1 inline-block rounded bg-red-50 px-2 py-1 text-left text-2xs leading-tight text-red-500 dark:bg-red-500/40 dark:text-red-50"
      >
        {formError ? formError : "An Error has occurred. Please try again."}
      </span>
    );
  },
  (prevProps, nextProps) => prevProps.formError === nextProps.formError,
);

export const FieldDescription = memo(
  function FieldDescription({
    description,
  }: {
    description: string | React.ReactNode;
  }) {
    if (!description) {
      return null;
    }

    // Check if description is a React component
    if (React.isValidElement(description)) {
      // Render the component directly
      return description;
    }

    // Otherwise, render as text in a paragraph
    return (
      <p className="text-left text-2xs text-foreground/70">{description}</p>
    );
  },
  (prevProps, nextProps) => prevProps.description === nextProps.description,
);

type FieldWrapperProps = {
  label?: string;
  description?: string;
  required?: boolean;
  className?: string;
  children: React.ReactNode;
  error?: string;
};

const FieldLabel = memo(
  function FieldLabel({
    label,
    required,
  }: {
    label?: string;
    required?: boolean;
  }) {
    return label ? (
      <Label
        className={cn("block text-xs font-medium", required && "required")}
      >
        {label}
      </Label>
    ) : null;
  },
  (prevProps, nextProps) =>
    prevProps.label === nextProps.label &&
    prevProps.required === nextProps.required,
);

// Memoize the FieldWrapper component
export const FieldWrapper = memo(function FieldWrapper({
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
    <div className={className}>
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
});

function FieldWrapperInner({ children }: { children: React.ReactNode }) {
  return <div className="flex items-center mb-0.5">{children}</div>;
}

function FieldWrapperDescriptionInner({
  children,
}: {
  children: React.ReactNode;
}) {
  return <div className="flex justify-start">{children}</div>;
}

type PasswordFieldWrapperProps = FieldWrapperProps & {
  onPasswordReset?: () => void;
};

// Also memoize PasswordFieldWrapper since it's derived from FieldWrapper
export const PasswordFieldWrapper = memo(
  function PasswordFieldWrapper({
    label,
    description,
    required,
    className,
    children,
    error,
    onPasswordReset,
  }: PasswordFieldWrapperProps) {
    // Use useMemo for the description and error components to avoid unnecessary re-renders
    const descriptionElement = useMemo(() => {
      return description && !error ? (
        <FieldDescription description={description} />
      ) : null;
    }, [description, error]);

    const errorElement = useMemo(() => {
      return error ? <ErrorMessage formError={error} /> : null;
    }, [error]);

    return (
      <div className={className}>
        {label && (
          <div className="mb-1 flex items-center">
            <Label
              className={cn(
                "block text-sm font-medium",
                required && "required",
              )}
            >
              {label}
            </Label>
            {onPasswordReset && (
              <Button
                variant="link"
                type="button"
                size="noSize"
                onClick={onPasswordReset}
                className="ml-auto inline-block text-xs underline"
              >
                Forgot your password?
              </Button>
            )}
          </div>
        )}
        {children}
        <div className="flex justify-start">
          {descriptionElement}
          {errorElement}
        </div>
      </div>
    );
  },
  (prevProps, nextProps) => {
    // Custom comparison function to optimize re-renders
    return (
      prevProps.label === nextProps.label &&
      prevProps.description === nextProps.description &&
      prevProps.required === nextProps.required &&
      prevProps.className === nextProps.className &&
      prevProps.error === nextProps.error &&
      prevProps.children === nextProps.children &&
      prevProps.onPasswordReset === nextProps.onPasswordReset
    );
  },
);
