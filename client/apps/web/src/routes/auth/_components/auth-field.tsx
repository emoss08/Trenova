import { ArrowRightIcon, EyeIcon, EyeOffIcon, type IconComponent } from "@trenova/shared/components/icons";
import { Spinner } from "@trenova/shared/components/ui/spinner";
import { useT } from "@trenova/shared/i18n/use-t";
import {
  useId,
  useState,
  type InputHTMLAttributes,
  type KeyboardEvent,
  type ReactNode,
} from "react";
import { Controller, type Control, type FieldValues, type Path } from "react-hook-form";

type NativeInputProps = Omit<
  InputHTMLAttributes<HTMLInputElement>,
  "id" | "name" | "type" | "value" | "defaultValue" | "onChange" | "onBlur" | "className" | "children"
>;

export type AuthTextFieldProps<T extends FieldValues> = NativeInputProps & {
  name: Path<T>;
  control: Control<T>;
  label: string;
  type?: "text" | "email" | "password";
  /** Opposite the label: the "Forgot?" link, a strength label. */
  trailing?: ReactNode;
  /** Inside the control after the input: the caps-lock pill, the reveal button. */
  adornment?: ReactNode;
};

/**
 * The one input every sign-in screen is built from: a 44px filled control with its
 * label above and its error below, bound to react-hook-form through a Controller so
 * the zod schema stays the single source of what is valid. It knows about fields and
 * nothing else; a screen changes what it shows through the slots.
 */
export function AuthTextField<T extends FieldValues>({
  name,
  control,
  label,
  type = "text",
  trailing,
  adornment,
  ...inputProps
}: AuthTextFieldProps<T>) {
  const inputId = useId();
  const errorId = `${inputId}-error`;

  return (
    <Controller<T>
      name={name}
      control={control}
      render={({ field, fieldState }) => (
        <div className="flex flex-col gap-[7px]">
          <div className="flex items-baseline justify-between gap-3">
            <label htmlFor={inputId} className="text-base font-medium whitespace-nowrap">
              {label}
            </label>
            {trailing}
          </div>
          {/* overflow-hidden keeps the control round: Chrome's autofill paints the input
              square, which would otherwise show at the corners. */}
          <div
            data-invalid={fieldState.invalid}
            className="auth-control ui-container-focus-ring bg-auth-field border-border-strong relative flex h-11 items-center overflow-hidden rounded-[10px] border [--ring-opacity:0.16] [--ring-width:4px]"
          >
            <input
              {...inputProps}
              ref={field.ref}
              name={field.name}
              value={field.value ?? ""}
              onChange={field.onChange}
              onBlur={field.onBlur}
              id={inputId}
              type={type}
              aria-invalid={fieldState.invalid}
              aria-describedby={fieldState.error ? errorId : undefined}
              className="text-foreground text-auth-body placeholder:text-muted-foreground/70 h-full min-w-0 flex-1 border-0 bg-transparent px-3.5 outline-none disabled:cursor-not-allowed disabled:opacity-60"
            />
            {adornment}
          </div>
          {fieldState.error?.message ? (
            <AuthErrorText id={errorId}>{fieldState.error.message}</AuthErrorText>
          ) : null}
        </div>
      )}
    />
  );
}

/**
 * A password input with a reveal button and a caps-lock pill. Caps lock is read from
 * the key events the input already receives, so it shows only while typing here.
 */
export function AuthPasswordField<T extends FieldValues>({
  onKeyDown,
  onKeyUp,
  ...props
}: Omit<AuthTextFieldProps<T>, "type" | "adornment">) {
  const t = useT();
  const [revealed, setRevealed] = useState(false);
  const [capsLock, setCapsLock] = useState(false);

  const readCapsLock = (event: KeyboardEvent<HTMLInputElement>) => {
    setCapsLock(event.getModifierState("CapsLock"));
  };

  return (
    <AuthTextField<T>
      {...props}
      type={revealed ? "text" : "password"}
      onKeyDown={(event) => {
        readCapsLock(event);
        onKeyDown?.(event);
      }}
      onKeyUp={(event) => {
        readCapsLock(event);
        onKeyUp?.(event);
      }}
      adornment={
        <>
          {capsLock ? (
            <span className="auth-enter-quick border-border-strong text-muted-foreground text-2xs rounded-[5px] border px-1.5 py-px font-mono whitespace-nowrap">
              {t("Caps lock")}
            </span>
          ) : null}
          <button
            type="button"
            onClick={() => setRevealed((current) => !current)}
            aria-label={revealed ? t("Hide password") : t("Show password")}
            aria-pressed={revealed}
            className="text-muted-foreground hover:text-foreground grid h-full w-10 shrink-0 cursor-pointer place-items-center bg-transparent transition-colors outline-none"
          >
            {revealed ? (
              <EyeOffIcon className="size-[15px]" />
            ) : (
              <EyeIcon className="size-[15px]" />
            )}
          </button>
        </>
      }
    />
  );
}

export function AuthErrorText({ id, children }: { id?: string; children: ReactNode }) {
  return (
    <p id={id} role="alert" className="auth-enter-quick text-danger m-0 text-base">
      {children}
    </p>
  );
}

// Not merged through cn: tailwind-merge reads text-auth-body as a colour and would drop
// text-ink-foreground beside it.
const SUBMIT_STATE_CLASS = {
  busy: "cursor-progress",
  idle: "cursor-pointer active:scale-[0.985] disabled:cursor-not-allowed disabled:opacity-40",
} as const;

/**
 * The full-width primary action, in ink. While `busy` it is disabled and its label
 * swaps to `busyLabel` beside a spinner; the label re-enters whenever it changes.
 */
export function AuthSubmit({
  children,
  busy = false,
  busyLabel,
  disabled,
  type = "button",
  onClick,
  leadingIcon: LeadingIcon,
  trailingIcon: TrailingIcon = ArrowRightIcon,
}: {
  children: ReactNode;
  busy?: boolean;
  busyLabel?: string;
  disabled?: boolean;
  type?: "button" | "submit";
  onClick?: () => void;
  leadingIcon?: IconComponent;
  /** An arrow by default; null for none. */
  trailingIcon?: IconComponent | null;
}) {
  return (
    <button
      type={type}
      onClick={onClick}
      disabled={disabled || busy}
      aria-busy={busy}
      className={`auth-submit ui-focus-ring bg-ink text-ink-foreground text-auth-body relative flex h-11 w-full items-center justify-center overflow-hidden rounded-[10px] font-medium transition-transform ${SUBMIT_STATE_CLASS[busy ? "busy" : "idle"]}${type === "submit" ? "mt-1.5" : ""}`}
    >
      <span key={busy ? "busy" : "idle"} className="auth-enter-quick inline-flex items-center gap-2">
        {busy ? (
          <>
            <Spinner className="size-3.5" />
            {busyLabel ?? children}
          </>
        ) : (
          <>
            {LeadingIcon ? <LeadingIcon data-auth-icon="leading" className="size-3.5" /> : null}
            {children}
            {TrailingIcon ? <TrailingIcon data-auth-icon="trailing" className="size-3.5" /> : null}
          </>
        )}
      </span>
    </button>
  );
}
