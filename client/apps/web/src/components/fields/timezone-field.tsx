import { timezoneGroupedChoices } from "@/lib/choices";
import { useT } from "@trenova/shared/i18n/use-t";
import { formatTimezoneLabel } from "@trenova/shared/lib/timezones";
import type { FormControlProps, SelectOptionGroup } from "@trenova/shared/types/fields";
import { useMemo } from "react";
import { useWatch, type FieldValues } from "react-hook-form";
import { SelectField } from "./select-field";

type TimezoneFieldProps<T extends FieldValues> = FormControlProps<T> & {
  label?: string;
  description?: string;
  placeholder?: string;
  isClearable?: boolean;
  className?: string;
  triggerClassName?: string;
};

const KNOWN_ZONES = new Set(
  timezoneGroupedChoices.flatMap((group) => group.options.map((option) => String(option.value))),
);

/**
 * A time zone picked from the grouped list every settings form offers, each
 * with its UTC offset. A zone saved before it was offered, or from a list that
 * named every zone, is kept and shown under its own heading rather than blank.
 */
export function TimezoneField<T extends FieldValues>({
  name,
  control,
  rules,
  label,
  description,
  placeholder,
  isClearable,
  className,
  triggerClassName,
}: TimezoneFieldProps<T>) {
  const t = useT();
  const value: unknown = useWatch({ control, name });
  const saved = typeof value === "string" && value !== "" && !KNOWN_ZONES.has(value) ? value : null;
  const groups = useMemo<SelectOptionGroup[]>(
    () =>
      saved
        ? [
            ...timezoneGroupedChoices,
            {
              label: t("Saved"),
              options: [{ value: saved, label: formatTimezoneLabel(saved) }],
            },
          ]
        : timezoneGroupedChoices,
    [saved, t],
  );

  return (
    <SelectField
      control={control}
      rules={rules}
      name={name}
      label={label}
      description={description}
      placeholder={placeholder ?? t("Select timezone")}
      groups={groups}
      isClearable={isClearable}
      className={className}
      triggerClassName={triggerClassName}
      renderOption={(option) => (
        <span className="flex w-full items-center justify-between gap-3">
          <span>{t(option.label)}</span>
          {option.description && (
            <span className="text-muted-foreground text-xs">{t(option.description)}</span>
          )}
        </span>
      )}
    />
  );
}
