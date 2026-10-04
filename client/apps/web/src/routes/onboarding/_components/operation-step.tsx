import { ChoiceButton } from "@/components/choice-button";
import { SectionPanel } from "@/components/section-panel";
import { OPERATION_TYPES } from "@/lib/onboarding-form";
import type { OnboardingFormValues } from "@/types/onboarding";
import { useT } from "@trenova/shared/i18n/use-t";
import { Controller, type Control } from "react-hook-form";

export function OperationStep({ control }: { control: Control<OnboardingFormValues> }) {
  const t = useT();

  return (
    <SectionPanel
      title={t("Operation type")}
      help={t(
        "Decides which halves of Trenova are turned on. You can change it later in Organization settings.",
      )}
    >
      <Controller
        name="operationType"
        control={control}
        render={({ field, fieldState }) => (
          <div className="flex flex-col gap-2 p-4">
            <p className="text-muted-foreground m-0 text-sm">
              {t("How does your company move freight?")}
            </p>
            <div
              role="radiogroup"
              aria-label={t("Operation type")}
              aria-invalid={fieldState.invalid}
              className="flex flex-col gap-2"
            >
              {OPERATION_TYPES.map((type) => (
                <ChoiceButton
                  key={type.value}
                  selected={field.value === type.value}
                  onClick={() => field.onChange(type.value)}
                  className="px-3 py-2.5"
                >
                  <span className="font-medium">{t(type.label)}</span>
                  <span className="text-muted-foreground text-xs">{t(type.description)}</span>
                </ChoiceButton>
              ))}
            </div>
            {fieldState.error?.message ? (
              <p role="alert" className="text-danger m-0 text-xs">
                {fieldState.error.message}
              </p>
            ) : null}
          </div>
        )}
      />
    </SectionPanel>
  );
}
