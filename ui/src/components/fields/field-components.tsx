import { Label } from "@/components/ui/label";
import { cn } from "@/lib/utils";
import { Button } from "../ui/button";

export function ErrorMessage({ formError }: { formError?: string }) {
  return (
    <div className="mt-1 inline-block rounded bg-red-50 px-2 py-1 text-left text-2xs leading-tight text-red-500 dark:bg-red-500/40 dark:text-red-50">
      {formError ? formError : "An Error has occurred. Please try again."}
    </div>
  );
}

export function FieldDescription({ description }: { description: string }) {
  return description ? (
    <p className="text-left text-2xs text-foreground/70">{description}</p>
  ) : null;
}

type FieldWrapperProps = {
  label?: string;
  description?: string;
  required?: boolean;
  className?: string;
  children: React.ReactNode;
  error?: string;
};

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
          <Label
            className={cn("block text-xs font-medium", required && "required")}
          >
            {label}
          </Label>
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
