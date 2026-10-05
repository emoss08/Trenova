import type { OnboardingFormValues } from "@/types/onboarding";
import { ArrowRightIcon } from "@trenova/shared/components/icons";
import { useT } from "@trenova/shared/i18n/use-t";
import { useEffect, useRef, type KeyboardEvent } from "react";
import { Controller, useFormContext, useWatch } from "react-hook-form";
import { Kbd } from "./keyboard-hint";
import { useShake } from "./use-shake";

/** The carrier IDs turn: SCAC and USDOT, both optional and skippable together. */
export function IdsAsk({ onSubmit }: { onSubmit: () => Promise<boolean> }) {
  const t = useT();
  const { control, setValue, clearErrors } = useFormContext<OnboardingFormValues>();
  const [scacCode, dotNumber] = useWatch({
    control,
    name: ["organization.scacCode", "organization.dotNumber"],
  });
  const [shaking, shake] = useShake();
  const scacRef = useRef<HTMLInputElement>(null);
  const empty = !scacCode && !dotNumber;

  useEffect(() => {
    scacRef.current?.focus({ preventScroll: true });
  }, []);

  const submit = async () => {
    if (!(await onSubmit())) {
      shake();
    }
  };

  const skip = () => {
    setValue("organization.scacCode", "", { shouldDirty: true });
    setValue("organization.dotNumber", "", { shouldDirty: true });
    clearErrors(["organization.scacCode", "organization.dotNumber"]);
    void submit();
  };

  const onEnter = (event: KeyboardEvent<HTMLInputElement>) => {
    if (event.key === "Enter" && !event.nativeEvent.isComposing) {
      event.preventDefault();
      void submit();
    }
  };

  return (
    <div className="nv-fc">
      <div className="nv-fg" data-cols="2">
        <Controller
          control={control}
          name="organization.scacCode"
          render={({ field, fieldState }) => (
            <div className="nv-fl">
              <label htmlFor="organization.scacCode">
                {t("SCAC code")} <i>{t("optional")}</i>
              </label>
              <input
                ref={(node) => {
                  field.ref(node);
                  scacRef.current = node;
                }}
                id="organization.scacCode"
                name={field.name}
                className="nv-in nv-mono"
                value={field.value ?? ""}
                maxLength={4}
                placeholder={t("RVFL")}
                onChange={(event) =>
                  field.onChange(event.target.value.toUpperCase().replace(/[^A-Z]/g, ""))
                }
                onBlur={field.onBlur}
                onKeyDown={onEnter}
                aria-invalid={fieldState.invalid || undefined}
                aria-describedby="nv-scac-help"
                data-shake={shaking}
                autoComplete="off"
              />
              <span className="nv-help" id="nv-scac-help">
                {fieldState.error ? (
                  <span className="nv-err" role="alert">
                    {fieldState.error.type === "validation"
                      ? fieldState.error.message
                      : t("2–4 letters.")}
                  </span>
                ) : (
                  t("Your Standard Carrier Alpha Code.")
                )}
              </span>
            </div>
          )}
        />
        <Controller
          control={control}
          name="organization.dotNumber"
          render={({ field, fieldState }) => (
            <div className="nv-fl">
              <label htmlFor="organization.dotNumber">
                {t("USDOT number")} <i>{t("optional")}</i>
              </label>
              <input
                ref={field.ref}
                id="organization.dotNumber"
                name={field.name}
                className="nv-in nv-mono"
                value={field.value ?? ""}
                inputMode="numeric"
                placeholder={t("3812045")}
                onChange={(event) =>
                  field.onChange(event.target.value.replace(/\D/g, "").slice(0, 8))
                }
                onBlur={field.onBlur}
                onKeyDown={onEnter}
                aria-invalid={fieldState.invalid || undefined}
                aria-describedby="nv-dot-help"
                data-shake={shaking}
                autoComplete="off"
              />
              <span className="nv-help" id="nv-dot-help">
                {fieldState.error ? (
                  <span className="nv-err" role="alert">
                    {fieldState.error.message}
                  </span>
                ) : (
                  t("Issued by FMCSA.")
                )}
              </span>
            </div>
          )}
        />
      </div>
      <div className="nv-fc-f">
        <span className="nv-hint">
          <Kbd>{"↵"}</Kbd> {empty ? t("to skip") : t("to continue")}
        </span>
        <span className="nv-sp" />
        {empty ? null : (
          <button type="button" className="nv-bt" onClick={skip}>
            {t("Skip")}
          </button>
        )}
        <button type="button" className="nv-bt" data-ink="true" onClick={() => void submit()}>
          {empty ? t("Skip for now") : t("Continue")}
          <ArrowRightIcon size={14} strokeWidth={1.7} aria-hidden="true" />
        </button>
      </div>
    </div>
  );
}
