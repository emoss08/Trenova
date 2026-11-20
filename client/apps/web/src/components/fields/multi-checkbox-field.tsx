import { isRequired, withRequired } from "@trenova/shared/lib/field-rules";
import { Label } from "@trenova/shared/components/ui/label";
import { useT } from "@trenova/shared/i18n/use-t";
import { useFieldRegistration } from "@trenova/shared/lib/form-field-registry";
import { cn } from "@trenova/shared/lib/utils";
import type { FormControlProps, GenericSelectOption } from "@trenova/shared/types/fields";
import { useErrorSurface } from "@/hooks/use-error-surface";
import { Controller, get, useFormState, type FieldValues } from "react-hook-form";
import { Checkbox } from "../animate-ui/components/base/checkbox";
import { ErrorMessage, settingRowDescriptionClass, useFieldLayout } from "./field-components";

/** Options shown together under a heading of their own, such as the events of one ledger. */
export type MultiCheckboxGroup<TValue extends string> = {
  label: string;
  options: ReadonlyArray<GenericSelectOption<TValue>>;
};

type MultiCheckboxFieldProps<T extends FieldValues, TValue extends string> = FormControlProps<T> & {
  label: string;
  description?: string;
  emptyAs?: "null" | "array";
} & (
    | { options: ReadonlyArray<GenericSelectOption<TValue>>; groups?: never }
    | { groups: ReadonlyArray<MultiCheckboxGroup<TValue>>; options?: never }
  );

/**
 * The first message under an error, its own or one on an entry of the list (a server that
 * refuses the fourth entry says so at `name.3`), so the field always says why.
 */
function firstErrorMessage(error: unknown): string | undefined {
  if (!error || typeof error !== "object") return undefined;
  const own = (error as { message?: unknown }).message;
  if (typeof own === "string" && own !== "") return own;
  for (const [key, value] of Object.entries(error)) {
    if (key === "ref" || key === "types") continue;
    const nested = firstErrorMessage(value);
    if (nested) return nested;
  }
  return undefined;
}

/**
 * A set of choices any number of which may be on, one checkbox to a choice. In a settings
 * list it is a full-width row: its name and meaning, then the choices under them, by group
 * when it has groups. It shows every error under its name, an entry's included.
 */
export function MultiCheckboxField<T extends FieldValues, TValue extends string>({
  name,
  control,
  rules,
  required,
  label,
  description,
  options,
  groups,
  emptyAs = "null",
}: MultiCheckboxFieldProps<T, TValue>) {
  useFieldRegistration(name, label);
  useErrorSurface(control, name);
  const t = useT();
  const layout = useFieldLayout();

  const groupId = `multi-checkbox-${name}`;
  const marked = isRequired(rules, required);
  // The controller follows errors on its own name only; an error on one entry of the list
  // (`name.3`) has to be followed here, or the field would not say it until something else
  // drew the form again.
  const { errors } = useFormState({ control, name });
  const errorMessage = firstErrorMessage(get(errors, name));

  return (
    <Controller
      name={name}
      control={control}
      rules={withRequired(rules, required)}
      render={({ field: { value, onChange, disabled } }) => {
        const selected = new Set<string>((value as string[] | null | undefined) ?? []);

        const toggle = (optionValue: string, checked: boolean) => {
          const next = new Set(selected);
          if (checked) {
            next.add(optionValue);
          } else {
            next.delete(optionValue);
          }
          onChange(next.size > 0 || emptyAs === "array" ? Array.from(next) : null);
        };

        const optionGrid = (items: ReadonlyArray<GenericSelectOption<TValue>>) => (
          <div
            className={cn(
              "grid gap-1.5",
              layout === "row" ? "grid-cols-1 sm:grid-cols-2 md:grid-cols-3" : "grid-cols-2",
            )}
          >
            {items.map((option) => {
              const checked = selected.has(option.value);
              const optionId = `${groupId}-${option.value}`;
              // An unavailable option can still be cleared, so a choice that
              // became invalid is never stuck on; it just cannot be made.
              const unavailable = option.disabled === true && !checked;

              return (
                <div
                  key={option.value}
                  className={cn(
                    "flex items-center gap-2 rounded-md border border-input bg-field px-2.5 py-2",
                    "transition-[border-color,box-shadow] duration-200 ease-in-out",
                    checked &&
                      "border-info bg-info-subtle text-info-foreground ring-2 ring-info/20 dark:text-info-foreground",
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
                      <span className="inline-flex items-center gap-1">
                        {t(option.label)}
                        {option.icon}
                      </span>
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
        );

        const choices = groups ? (
          <div id={groupId} className="flex flex-col gap-3" role="group" aria-label={label}>
            {groups.map((group) => (
              <div
                key={group.label}
                className="flex flex-col gap-1.5"
                role="group"
                aria-label={t(group.label)}
              >
                <p className="text-xs font-medium text-muted-foreground">{t(group.label)}</p>
                {optionGrid(group.options)}
              </div>
            ))}
          </div>
        ) : (
          <div id={groupId} role="group" aria-label={label}>
            {optionGrid(options ?? [])}
          </div>
        );

        // One row of a settings list, as wide as the list: a set of choices has no single
        // control to stand at the row's end, so they sit under its name and meaning.
        if (layout === "row") {
          return (
            <div data-slot="setting-row" className="flex flex-col gap-3 px-3.5 py-3">
              <div className="flex min-w-0 flex-col gap-px">
                <Label
                  htmlFor={groupId}
                  className={cn("text-sm font-medium", marked && "required")}
                >
                  {label}
                </Label>
                {description ? <p className={settingRowDescriptionClass}>{description}</p> : null}
                {errorMessage ? <ErrorMessage formError={errorMessage} /> : null}
              </div>
              {choices}
            </div>
          );
        }

        return (
          <div className="flex w-full flex-col gap-1.5">
            <Label htmlFor={groupId} className={cn(marked && "required")}>
              {label}
            </Label>
            {choices}
            {errorMessage ? (
              <p className="text-2xs text-destructive">{errorMessage}</p>
            ) : description ? (
              <p className="text-2xs text-muted-foreground">{description}</p>
            ) : null}
          </div>
        );
      }}
    />
  );
}
