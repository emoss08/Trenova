import { useT } from "@trenova/shared/i18n/use-t";
import { Label } from "@trenova/shared/components/ui/label";
import { cn } from "@trenova/shared/lib/utils";
import type { FormControlProps, GenericSelectOption } from "@trenova/shared/types/fields";
import { Controller, type FieldValues } from "react-hook-form";
import { Checkbox } from "../animate-ui/components/base/checkbox";

type MultiCheckboxFieldProps<T extends FieldValues, TValue extends string> = FormControlProps<T> & {
  label: string;
  description?: string;
  options: ReadonlyArray<GenericSelectOption<TValue>>;
};

/**
 * A checkbox group bound to an array field. For small, fixed option sets where
 * an async multi-select would be overkill and every choice should stay visible.
 * An empty selection is stored as null so the server treats it as a wildcard.
 * A disabled option shows its description as the reason it cannot be chosen.
 */
export function MultiCheckboxField<T extends FieldValues, TValue extends string>({
  name,
  control,
  rules,
  label,
  description,
  options,
}: MultiCheckboxFieldProps<T, TValue>) {
  const t = useT();

  const groupId = `multi-checkbox-${name}`;

  return (
    <Controller
      name={name}
      control={control}
      rules={rules}
      render={({ field: { value, onChange, disabled }, fieldState }) => {
        const selected = new Set<string>((value as string[] | null | undefined) ?? []);

        const toggle = (optionValue: string, checked: boolean) => {
          const next = new Set(selected);
          if (checked) {
            next.add(optionValue);
          } else {
            next.delete(optionValue);
          }
          onChange(next.size > 0 ? Array.from(next) : null);
        };

        return (
          <div className="flex w-full flex-col gap-1.5">
            <Label htmlFor={groupId}>{label}</Label>
            <div id={groupId} className="grid grid-cols-2 gap-1.5" role="group" aria-label={label}>
              {options.map((option) => {
                const checked = selected.has(option.value);
                const optionId = `${groupId}-${option.value}`;
                // An unavailable option can still be cleared, so a choice that
                // became invalid is never stuck on; it just cannot be made.
                const unavailable = option.disabled === true && !checked;

                return (
                  <div
                    key={option.value}
                    className={cn(
                      "border-input bg-field flex items-center gap-2 rounded-md border px-2.5 py-2",
                      "transition-[border-color,box-shadow] duration-200 ease-in-out",
                      checked && "border-foreground ring-foreground/10 ring-2",
                      unavailable && "bg-sunken opacity-60",
                    )}
                  >
                    <Checkbox
                      id={optionId}
                      checked={checked}
                      disabled={disabled || unavailable}
                      aria-describedby={
                        option.disabled && option.description ? `${optionId}-reason` : undefined
                      }
                      onCheckedChange={(state) => toggle(option.value, state === true)}
                    />
                    <div className="flex min-w-0 flex-col">
                      <Label
                        htmlFor={optionId}
                        className={cn(
                          "text-xs font-normal",
                          unavailable ? "cursor-not-allowed" : "cursor-pointer",
                        )}
                      >
                        {t(option.label)}
                      </Label>
                      {option.disabled && option.description && (
                        <span id={`${optionId}-reason`} className="text-2xs text-muted-foreground">
                          {t(option.description)}
                        </span>
                      )}
                    </div>
                  </div>
                );
              })}
            </div>
            {fieldState.error ? (
              <p className="text-2xs text-destructive">{fieldState.error.message}</p>
            ) : description ? (
              <p className="text-2xs text-muted-foreground">{description}</p>
            ) : null}
          </div>
        );
      }}
    />
  );
}
