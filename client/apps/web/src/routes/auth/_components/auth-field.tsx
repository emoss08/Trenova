import { Button } from "@trenova/shared/components/ui/button";
import { cn } from "@trenova/shared/lib/utils";
import { useId, useState, type ReactNode } from "react";
import { Controller, type Control, type FieldValues, type Path } from "react-hook-form";

/**
 * The sign-in screen runs its own field geometry — a 9px control on the --field fill
 * with a translucent focus ring — rather than the dense in-app InputField, whose 28px
 * row and boxed red error belong to data entry, not to a 400px auth card. The form
 * plumbing is unchanged: still react-hook-form Controllers over the real zod schema.
 */
export function AuthTextField<T extends FieldValues>({
  name,
  control,
  label,
  type = "text",
  placeholder,
  autoComplete,
  required,
  disabled,
  trailing,
  revealable,
}: {
  name: Path<T>;
  control: Control<T>;
  label: string;
  type?: "text" | "email" | "password";
  placeholder?: string;
  autoComplete?: string;
  required?: boolean;
  disabled?: boolean;
  /** Rendered opposite the label — the "Forgot?" link on the password row. */
  trailing?: ReactNode;
  revealable?: boolean;
}) {
  const inputId = useId();
  const errorId = `${inputId}-error`;
  const [revealed, setRevealed] = useState(false);
  const resolvedType = revealable && revealed ? "text" : type;

  return (
    <Controller<T>
      name={name}
      control={control}
      render={({ field, fieldState }) => (
        <div className="flex flex-col gap-1.5">
          <div className="flex items-baseline justify-between gap-3">
            <label
              htmlFor={inputId}
              className="text-muted-foreground text-[11.5px] font-medium whitespace-nowrap"
            >
              {label} {required && <span className="text-auth-danger">*</span>}
            </label>
            {trailing}
          </div>
          {/* overflow-hidden is what keeps the control round: the input paints its own
              square background on focus and, in Chrome, an opaque autofill fill that
              would otherwise square off the corners inside the rounded border. */}
          <div
            data-invalid={fieldState.invalid}
            className="auth-control bg-field border-border flex items-center overflow-hidden rounded-[9px] border"
          >
            <input
              {...field}
              id={inputId}
              type={resolvedType}
              value={field.value ?? ""}
              placeholder={placeholder}
              autoComplete={autoComplete}
              disabled={disabled}
              aria-invalid={fieldState.invalid}
              aria-describedby={fieldState.error ? errorId : undefined}
              className="text-foreground placeholder:text-subtle-foreground min-w-0 flex-1 rounded-[8px] border-0 bg-transparent px-3 py-2.5 text-[13px] outline-none disabled:cursor-not-allowed disabled:opacity-60"
            />
            {revealable && (
              <button
                type="button"
                onClick={() => setRevealed((current) => !current)}
                className="text-subtle-foreground hover:text-foreground cursor-pointer bg-transparent px-3 text-[11.5px] font-medium transition-colors duration-150"
              >
                {revealed ? "hide" : "show"}
              </button>
            )}
          </div>
          {fieldState.error?.message && (
            <AuthErrorText id={errorId}>{fieldState.error.message}</AuthErrorText>
          )}
        </div>
      )}
    />
  );
}

export function AuthErrorText({ id, children }: { id?: string; children: ReactNode }) {
  return (
    <span id={id} role="alert" className="text-auth-danger auth-step-enter text-[11.5px]">
      {children}
    </span>
  );
}

/**
 * Full-width primary action. The default Button variant is brand blue; the auth screen
 * is monochrome by design, so the fill is the foreground and the label the background.
 */
export function AuthSubmit({
  children,
  isLoading,
  loadingText,
  disabled,
  type = "button",
  onClick,
}: {
  children: ReactNode;
  isLoading?: boolean;
  loadingText?: string;
  disabled?: boolean;
  type?: "button" | "submit";
  onClick?: () => void;
}) {
  return (
    <Button
      type={type}
      onClick={onClick}
      disabled={disabled}
      isLoading={isLoading}
      loadingText={loadingText}
      className={cn(
        "bg-foreground text-background border-foreground h-10 w-full rounded-[9px] border text-[13px] font-[550]",
        "hover:bg-foreground hover:opacity-90 active:scale-[0.988] disabled:opacity-40",
      )}
    >
      {children}
    </Button>
  );
}
