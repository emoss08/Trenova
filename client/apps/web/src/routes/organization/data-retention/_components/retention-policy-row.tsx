import { SettingRow, settingRowDescriptionClass } from "@/components/fields/field-components";
import { NumberField } from "@/components/fields/number-field";
import { SettingsNested } from "@/components/settings/settings-section";
import { Switch } from "@trenova/shared/components/ui/switch";
import { useT } from "@trenova/shared/i18n/use-t";
import { formatUnixDateMedium } from "@trenova/shared/lib/date";
import { useState } from "react";
import { useFormContext, useWatch } from "react-hook-form";
import { retentionCutoff, type DataRetentionFormValues } from "../data-retention-form";

type RetentionField = keyof DataRetentionFormValues;

/** What the next purge does to the data older than the cutoff, said with the cutoff's date. */
export type RetentionConsequence = (cutoffDate: string) => string;

/**
 * One period in the policy: what the data is and what is removed on the left, the days it is
 * kept at the row's end. Under the description, the date the next nightly purge reaches back to,
 * so the consequence of a period is read before it is saved. The description keeps the id the
 * field points its input at, so a screen reader hears it with the field.
 */
export function RetentionPeriodField({
  name,
  label,
  description,
  consequence,
  min,
  max,
  now,
  disabled,
}: {
  name: RetentionField;
  label: string;
  description: string;
  consequence: RetentionConsequence;
  min?: number;
  max?: number;
  now: number;
  disabled: boolean;
}) {
  const t = useT();
  const { control } = useFormContext<DataRetentionFormValues>();
  const days = useWatch({ control, name });
  const cutoff = typeof days === "number" ? retentionCutoff(days, now) : null;

  return (
    <NumberField
      control={control}
      name={name}
      label={label}
      description={
        <div id={`input-${name}-description`} className={settingRowDescriptionClass}>
          <p>{description}</p>
          {cutoff !== null && (
            <p className="text-foreground">{consequence(formatUnixDateMedium(cutoff))}</p>
          )}
        </div>
      }
      sideText={t("days")}
      placeholder={t("Days")}
      clearable={false}
      min={min}
      max={max}
      disabled={disabled}
    />
  );
}

/**
 * A kind of data the server keeps whole unless it is given a period (a period of 0 keeps
 * everything). The row's switch turns the purge on, and the period it uses opens under it;
 * turning the purge off and on again brings back the last period it had.
 */
export function RetentionLimit({
  name,
  label,
  description,
  periodLabel,
  periodDescription,
  consequence,
  fallbackDays,
  now,
  disabled,
}: {
  name: RetentionField;
  label: string;
  description: string;
  periodLabel: string;
  periodDescription: string;
  consequence: RetentionConsequence;
  /** The period a purge starts at when it was never given one. */
  fallbackDays: number;
  now: number;
  disabled: boolean;
}) {
  const { control, setValue } = useFormContext<DataRetentionFormValues>();
  const days = useWatch({ control, name });
  const limited = days !== 0;
  const [lastLimit, setLastLimit] = useState(() =>
    typeof days === "number" && days > 0 ? days : fallbackDays,
  );
  const switchId = `switch-${name}`;
  const descriptionId = `${switchId}-description`;

  const setLimited = (on: boolean) => {
    if (!on && typeof days === "number" && days > 0) {
      setLastLimit(days);
    }
    setValue(name, on ? lastLimit : 0, { shouldDirty: true, shouldValidate: true });
  };

  return (
    <>
      <SettingRow
        label={label}
        htmlFor={switchId}
        controlSize="auto"
        description={
          <p id={descriptionId} className={settingRowDescriptionClass}>
            {description}
          </p>
        }
      >
        <Switch
          id={switchId}
          aria-describedby={descriptionId}
          checked={limited}
          disabled={disabled}
          onCheckedChange={setLimited}
        />
      </SettingRow>
      <SettingsNested show={limited}>
        <RetentionPeriodField
          name={name}
          label={periodLabel}
          description={periodDescription}
          consequence={consequence}
          min={1}
          now={now}
          disabled={disabled}
        />
      </SettingsNested>
    </>
  );
}
