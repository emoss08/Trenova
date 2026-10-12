import { useT } from "@trenova/shared/i18n/use-t";
import { cn } from "@trenova/shared/lib/utils";
import type { WarningProps } from "@trenova/shared/types/fields";
import React, { useMemo } from "react";
import { Label } from "@trenova/shared/components/ui/label";
import {
  FieldNameProvider,
  useFieldRegistration,
  type FieldValueFormat,
} from "@trenova/shared/lib/form-field-registry";

export function ErrorMessage({ formError, id }: { formError?: string; id?: string }) {
  const t = useT();

  return (
    <span
      id={id}
      role="alert"
      className="text-destructive dark:bg-destructive/40 mt-1 inline-block rounded-md bg-danger-subtle px-2 py-1 text-left text-xs leading-tight dark:text-danger-foreground"
    >
      {formError ? formError : t("An Error has occurred. Please try again.")}
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
      <p id={id} className="text-2xs text-left text-warning-foreground">
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
    <p id={id} className="text-2xs text-foreground/70 text-left">
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
  /**
   * Where the label and description sit: above the control, or in a column
   * beside it, for a settings form read down a list of rows.
   */
  layout?: FieldLayout;
  /** The form field's name; inside a form it registers the field and its label. */
  name?: string;
  /**
   * How the field's values read where the stored value is not readable, such as an
   * option's key; a change review shows values through it.
   */
  formatValue?: FieldValueFormat;
};

export type FieldLayout = "stacked" | "inline";

export function FieldLabel({
  label,
  required,
  className,
}: {
  label?: React.ReactNode;
  required?: boolean;
  className?: string;
}) {
  if (!label) {
    return null;
  }

  const classes = cn("block text-xs font-medium", required && "required", className);
  if (React.isValidElement(label)) {
    return <div className={classes}>{label}</div>;
  }

  return <Label className={classes}>{label}</Label>;
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
  layout = "stacked",
  name,
  formatValue,
}: FieldWrapperProps) {
  useFieldRegistration(name, label, formatValue);
  const control = name ? <FieldNameProvider name={name}>{children}</FieldNameProvider> : children;
  const descriptionElement = useMemo(() => {
    return !error && (description || warning?.show) ? (
      <FieldDescription description={description} warning={warning} id={descriptionId} />
    ) : null;
  }, [description, descriptionId, error, warning]);

  const errorElement = useMemo(() => {
    return error ? <ErrorMessage formError={error} id={errorId} /> : null;
  }, [error, errorId]);

  if (layout === "inline") {
    return (
      <div
        className={cn(
          "grid grid-cols-[minmax(0,2fr)_minmax(0,3fr)] items-start gap-x-4",
          className,
        )}
      >
        <div className="flex min-w-0 flex-col gap-1 pt-2">
          <FieldLabel label={label} required={required} className="text-sm" />
          {(description || warning?.show) && (
            <FieldDescription description={description} warning={warning} id={descriptionId} />
          )}
        </div>
        <div className="flex min-w-0 flex-col gap-0.5">
          {control}
          {errorElement}
        </div>
      </div>
    );
  }

  return (
    <div className={cn("flex flex-col gap-0.5", className)}>
      {label && (
        <FieldWrapperInner>
          <FieldLabel label={label} required={required} />
        </FieldWrapperInner>
      )}
      {control}
      <FieldWrapperDescriptionInner>
        {descriptionElement}
        {errorElement}
      </FieldWrapperDescriptionInner>
    </div>
  );
}
