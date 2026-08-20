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
 */
export function MultiCheckboxField<T extends FieldValues, TValue extends string>({
  name,
  control,
  rules,
  label,
  description,
  options,
}: MultiCheckboxFieldProps<T, TValue>) {
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
            <div
              id={groupId}
              className="grid grid-cols-2 gap-1.5"
              role="group"
              aria-label={label}
            >
              {options.map((option) => {
                const checked = selected.has(option.value);
                const optionId = `${groupId}-${option.value}`;

                return (
                  <div
                    key={option.value}
                    className={cn(
                      "border-muted-foreground/20 bg-muted flex items-center gap-2 rounded-md border px-2.5 py-2",
                      "transition-[border-color,box-shadow] duration-200 ease-in-out",
                      checked && "border-foreground ring-foreground/10 ring-2",
                    )}
                  >
                    <Checkbox
                      id={optionId}
                      checked={checked}
                      disabled={disabled}
                      onCheckedChange={(state) => toggle(option.value, state === true)}
                    />
                    <Label htmlFor={optionId} className="cursor-pointer text-xs font-normal">
                      {option.label}
                    </Label>
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
