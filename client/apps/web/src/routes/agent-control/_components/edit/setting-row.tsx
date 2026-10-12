import { Switch } from "@/components/animate-ui/components/base/switch";
import { useFieldRegistration } from "@trenova/shared/lib/form-field-registry";
import { cn } from "@trenova/shared/lib/utils";
import type { ReactNode } from "react";
import { Controller, type Control, type FieldPath, type FieldValues } from "react-hook-form";

type SettingRowProps = {
  label: string;
  note: ReactNode;
  /** Shown under the note, in the warning tone, such as why the setting is off. */
  why?: ReactNode;
  disabled?: boolean;
  labelFor?: string;
  children: ReactNode;
};

/** One setting: what it is and a line on what it does, with its control on the right. */
export function SettingRow({
  label,
  note,
  why,
  disabled = false,
  labelFor,
  children,
}: SettingRowProps) {
  return (
    <div className="border-border-subtle flex items-start justify-between gap-4 border-b py-3">
      <div className="flex flex-col gap-0.5">
        <label
          htmlFor={labelFor}
          className={cn("text-base font-medium", disabled && "text-muted-foreground")}
        >
          {label}
        </label>
        <span className="text-muted-foreground text-sm">{note}</span>
        {why && <span className="text-warning-foreground text-sm">{why}</span>}
      </div>
      {children}
    </div>
  );
}

type SwitchRowProps<T extends FieldValues> = {
  control: Control<T>;
  name: FieldPath<T>;
  label: string;
  note: ReactNode;
  disabled?: boolean;
  /** Why it is disabled, shown in place of nothing. */
  why?: ReactNode;
};

/** A setting that is a switch, bound to a form field; a field error shows in the row. */
export function SwitchRow<T extends FieldValues>({
  control,
  name,
  label,
  note,
  disabled = false,
  why,
}: SwitchRowProps<T>) {
  useFieldRegistration(name, label);
  const id = `setting-${name}`;

  return (
    <Controller
      control={control}
      name={name}
      render={({ field: { value, onChange, onBlur, ref }, fieldState }) => (
        <SettingRow
          label={label}
          labelFor={id}
          note={note}
          disabled={disabled}
          why={fieldState.error?.message ?? (disabled ? why : undefined)}
        >
          <Switch
            id={id}
            ref={ref}
            checked={Boolean(value) && !disabled}
            disabled={disabled}
            aria-invalid={fieldState.invalid || undefined}
            onCheckedChange={(checked) => onChange(checked)}
            onBlur={onBlur}
          />
        </SettingRow>
      )}
    />
  );
}
