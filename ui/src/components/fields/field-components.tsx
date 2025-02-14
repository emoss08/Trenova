import { Label } from "@/components/ui/label";
import { cn } from "@/lib/utils";
import { memo } from "react";
import { Button } from "../ui/button";

const ErrorMessage = memo(
  function ErrorMessage({ formError }: { formError?: string }) {
    return (
      <div className="mt-1 inline-block rounded bg-red-50 px-2 py-1 text-left text-2xs leading-tight text-red-500 dark:bg-red-500/40 dark:text-red-50">
        {formError ? formError : "An Error has occurred. Please try again."}
      </div>
    );
  },
  (prevProps, nextProps) => prevProps.formError === nextProps.formError,
);

const FieldDescription = memo(
  function FieldDescription({ description }: { description: string }) {
    return description ? (
      <p className="text-left text-2xs text-foreground/70">{description}</p>
    ) : null;
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
  (prevProps, nextProps) => prevProps.label === nextProps.label,
);

export function FieldWrapper({
  label,
  description,
  required,
  className,
  children,
  error,
}: FieldWrapperProps) {
  return (
    <div className={className}>
      {label && (
        <div className="mb-0.5 flex items-center">
          <FieldLabel label={label} required={required} />
        </div>
      )}
      {children}
      <div className="flex justify-start">
        {description && !error && (
          <FieldDescription description={description} />
        )}
        {error && <ErrorMessage formError={error} />}
      </div>
    </div>
  );
}

type PasswordFieldWrapperProps = FieldWrapperProps & {
  onPasswordReset: () => void;
};

export function PasswordFieldWrapper({
  label,
  description,
  required,
  className,
  children,
  error,
  onPasswordReset,
}: PasswordFieldWrapperProps) {
  return (
    <div className={className}>
      {label && (
        <div className="mb-1 flex items-center">
          <Label
            className={cn("block text-sm font-medium", required && "required")}
          >
            {label}
          </Label>
          <Button
            variant="link"
            size="noSize"
            onClick={onPasswordReset}
            className="ml-auto inline-block text-sm underline"
          >
            Forgot your password?
          </Button>
        </div>
      )}
      {children}
      <div className="flex justify-start">
        {description && !error && (
          <FieldDescription description={description} />
        )}
        {error && <ErrorMessage formError={error} />}
      </div>
    </div>
  );
}
