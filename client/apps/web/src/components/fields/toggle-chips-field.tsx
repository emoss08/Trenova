import { CheckIcon, PlusIcon } from "@trenova/shared/components/icons";
import { cn } from "@trenova/shared/lib/utils";
import type { FormControlProps } from "@trenova/shared/types/fields";
import type React from "react";
import { Controller, type FieldValues } from "react-hook-form";
import { ErrorMessage } from "./field-components";
import { useFieldRegistration } from "@trenova/shared/lib/form-field-registry";

export type ToggleChipOption<TValue extends string> = {
  value: TValue;
  label: string;
  /** A mark after the label, such as a shield on a choice that needs trust. */
  mark?: React.ReactNode;
};

type ToggleChipsFieldProps<T extends FieldValues, TValue extends string> = FormControlProps<T> & {
  /** Names the group for assistive technology; the section heading shows it on screen. */
  label: string;
  options: readonly ToggleChipOption<TValue>[];
  /** How a chosen chip reads: the ink of the page, or the success tone. */
  checkedTone?: "neutral" | "success";
  className?: string;
};

const CHECKED_TONE = {
  neutral: "border-border-strong bg-card text-foreground border-solid",
  success: "border-success-border bg-success-subtle text-success-subtle-foreground border-solid",
} as const;

/**
 * Several choices from a short set, each a chip that toggles, bound to a list
 * field. An empty list means none chosen; every choice stays in view.
 */
export function ToggleChipsField<T extends FieldValues, TValue extends string>({
  name,
  control,
  rules,
  label,
  options,
  checkedTone = "neutral",
  className,
}: ToggleChipsFieldProps<T, TValue>) {
  useFieldRegistration(name, label);
  return (
    <Controller
      name={name}
      control={control}
      rules={rules}
      render={({ field: { value, onChange, disabled }, fieldState }) => {
        const chosen = (value as TValue[] | null | undefined) ?? [];

        return (
          <div className={cn("flex flex-col gap-1", className)}>
            <div className="flex flex-wrap gap-1" role="group" aria-label={label}>
              {options.map((option) => {
                const on = chosen.includes(option.value);
                return (
                  <button
                    key={option.value}
                    type="button"
                    aria-pressed={on}
                    disabled={disabled}
                    className={cn(
                      "ui-focus-ring inline-flex h-6 items-center gap-1.5 rounded-full border pr-2.5 pl-2 text-sm transition-colors disabled:cursor-not-allowed disabled:opacity-60",
                      on
                        ? CHECKED_TONE[checkedTone]
                        : "border-border-strong text-muted-foreground hover:border-foreground hover:text-foreground border-dashed",
                    )}
                    onClick={() =>
                      onChange(
                        on
                          ? chosen.filter((entry) => entry !== option.value)
                          : [...chosen, option.value],
                      )
                    }
                  >
                    {on ? <CheckIcon className="size-3" /> : <PlusIcon className="size-3" />}
                    {option.label}
                    {option.mark}
                  </button>
                );
              })}
            </div>
            {fieldState.error?.message && <ErrorMessage formError={fieldState.error.message} />}
          </div>
        );
      }}
    />
  );
}
