import type { OnboardingFormValues } from "@/types/onboarding";
import { ArrowUpIcon } from "@trenova/shared/components/icons";
import { useT } from "@trenova/shared/i18n/use-t";
import { useEffect, useRef } from "react";
import { Controller, useFormContext } from "react-hook-form";
import { Kbd } from "./keyboard-hint";
import { useShake } from "./use-shake";

/** The company name turn: one borderless input in a floating composer. */
export function ComposerAsk({ onSubmit }: { onSubmit: () => Promise<boolean> }) {
  const t = useT();
  const { control } = useFormContext<OnboardingFormValues>();
  const [shaking, shake] = useShake();
  const inputRef = useRef<HTMLInputElement>(null);

  useEffect(() => {
    inputRef.current?.focus();
  }, []);

  const submit = async () => {
    if (!(await onSubmit())) {
      shake();
      inputRef.current?.focus();
    }
  };

  return (
    <Controller
      control={control}
      name="organization.name"
      render={({ field, fieldState }) => {
        const empty = (field.value ?? "").trim() === "";
        const error = fieldState.error
          ? empty
            ? t("Your company name is required.")
            : fieldState.error.message
          : undefined;

        return (
          <div>
            <div className="nv-cmp" data-bad={shaking}>
              <span className="nv-cmp-ring" aria-hidden="true" />
              <input
                ref={(node) => {
                  inputRef.current = node;
                  field.ref(node);
                }}
                name={field.name}
                value={field.value ?? ""}
                onChange={field.onChange}
                onBlur={field.onBlur}
                onKeyDown={(event) => {
                  if (event.key === "Enter" && !event.nativeEvent.isComposing) {
                    event.preventDefault();
                    void submit();
                  }
                }}
                placeholder={t("Company name")}
                aria-label={t("Company name")}
                aria-invalid={error ? true : undefined}
                aria-describedby="nv-name-hint"
                autoComplete="organization"
                maxLength={150}
              />
              <button
                type="button"
                className="nv-send"
                disabled={empty}
                onClick={() => void submit()}
                aria-label={t("Send")}
              >
                <ArrowUpIcon size={14} strokeWidth={1.8} aria-hidden="true" />
              </button>
            </div>
            <div className="nv-hint" id="nv-name-hint">
              {error ? (
                <p className="nv-err" role="alert">
                  {error}
                </p>
              ) : (
                <>
                  <Kbd>{"↵"}</Kbd> {t("to answer")}
                </>
              )}
            </div>
          </div>
        );
      }}
    />
  );
}
