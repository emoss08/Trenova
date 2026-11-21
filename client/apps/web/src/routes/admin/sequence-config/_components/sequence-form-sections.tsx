import { InputField } from "@/components/fields/input-field";
import { NumberField } from "@/components/fields/number-field";
import { SegmentedField } from "@/components/fields/segmented-field";
import { SelectField } from "@/components/fields/select-field";
import { SwitchField } from "@/components/fields/switch-field";
import {
  SettingsNested,
  SettingsReadout,
  SettingsSection,
} from "@/components/settings/settings-section";
import { SettingsReveal } from "@/components/settings/settings-reveal";
import type { SequenceConfigDocument, SequenceType } from "@/types/sequence-config";
import { useT } from "@trenova/shared/i18n/use-t";
import { useFormContext, useWatch } from "react-hook-form";
import {
  drawsRecordCodes,
  patternTokenLabels,
  patternTokens,
  recordCodeTokens,
  separatorChoices,
  sequenceDescriptions,
  sequenceTitles,
  yearDigitOptions,
} from "./sequence-config-constants";
import { SequencePreview } from "./sequence-preview";

type SectionProps = { index: number; type: SequenceType; now: Date };

/**
 * One numbering sequence: its next number at the top, then its settings in the order the
 * pieces appear in the number. A sequence written as a pattern shows the pattern and the
 * settings its placeholders read instead of the pieces it replaces.
 */
export function SequenceSection({ index, type, now }: SectionProps) {
  const t = useT();
  const { control } = useFormContext<SequenceConfigDocument>();
  const path = `configs.${index}` as const;
  const [custom, year, week, separators, random] = useWatch({
    control,
    name: [
      `${path}.allowCustomFormat`,
      `${path}.includeYear`,
      `${path}.includeWeekNumber`,
      `${path}.useSeparators`,
      `${path}.includeRandomDigits`,
    ],
  });
  const codes = drawsRecordCodes(type);

  const yearDigits = (
    <SegmentedField
      control={control}
      name={`${path}.yearDigits`}
      label={t("Year digits")}
      description={t("Two digits (26) or the whole year (2026).")}
      options={yearDigitOptions}
      fullWidth={false}
      required
    />
  );

  return (
    <SettingsSection title={sequenceTitles[type]} description={sequenceDescriptions[type]}>
      <SequencePreview index={index} now={now} />
      <InputField
        control={control}
        name={`${path}.prefix`}
        label={t("Prefix")}
        description={t("The letters every number starts with.")}
        placeholder={t("e.g. INV")}
        maxLength={20}
        required
      />
      <SwitchField
        control={control}
        name={`${path}.allowCustomFormat`}
        label={t("Write your own pattern")}
        description={t("Lay the number out yourself with placeholders instead of the pieces below.")}
      />
      <SettingsNested show={custom}>
        <InputField
          control={control}
          name={`${path}.customFormat`}
          label={t("Pattern")}
          description={t("Anything outside the placeholders is written as it is.")}
          placeholder={t("{P}-{Y}{M}-{S}")}
          required={custom}
        />
        <PatternTokens index={index} type={type} />
        {yearDigits}
      </SettingsNested>
      <SettingsReveal show={!custom} className="divide-y divide-border">
        {codes ? (
          <SwitchField
            control={control}
            name={`${path}.includeBusinessUnitCode`}
            label={t("Business unit code")}
            description={t("Your business unit's code, after the prefix.")}
          />
        ) : null}
        <SwitchField
          control={control}
          name={`${path}.includeYear`}
          label={t("Year")}
          description={t("The year the number is issued.")}
        />
        <SettingsNested show={year}>{yearDigits}</SettingsNested>
        <SwitchField
          control={control}
          name={`${path}.includeMonth`}
          label={t("Month")}
          description={
            week
              ? t("The week takes the month's place while it is on.")
              : t("The month the number is issued, as two digits.")
          }
          disabled={week}
        />
        <SwitchField
          control={control}
          name={`${path}.includeWeekNumber`}
          label={t("Week")}
          description={t("The week of the year, in place of the month.")}
        />
        <SwitchField
          control={control}
          name={`${path}.includeDay`}
          label={t("Day")}
          description={t("The day of the month, as two digits.")}
        />
        {codes ? (
          <SwitchField
            control={control}
            name={`${path}.includeLocationCode`}
            label={t("Location code")}
            description={t("The code of the shipment's first stop, before the counter.")}
          />
        ) : null}
        <SwitchField
          control={control}
          name={`${path}.useSeparators`}
          label={t("Separators")}
          description={t("A character between the pieces, so the number reads in parts.")}
        />
        <SettingsNested show={separators}>
          <SelectField
            control={control}
            name={`${path}.separatorChar`}
            label={t("Separator")}
            options={separatorChoices}
            required={separators}
          />
        </SettingsNested>
      </SettingsReveal>
      <NumberField
        control={control}
        name={`${path}.sequenceDigits`}
        label={t("Counter digits")}
        description={t("How many digits the counter is padded to with zeros.")}
        sideText={t("digits")}
        min={1}
        max={10}
        required
      />
      <SwitchField
        control={control}
        name={`${path}.includeRandomDigits`}
        label={t("Random digits")}
        description={t("Digits after the counter that make a number hard to guess.")}
      />
      <SettingsNested show={random}>
        <NumberField
          control={control}
          name={`${path}.randomDigitsCount`}
          label={t("How many")}
          sideText={t("digits")}
          min={1}
          max={10}
          required={random}
        />
      </SettingsNested>
      <SwitchField
        control={control}
        name={`${path}.includeCheckDigit`}
        label={t("Check digit")}
        description={
          custom
            ? t("Fills {C} in the pattern with a digit that catches a mistyped number.")
            : t("A last digit that catches a mistyped number.")
        }
      />
    </SettingsSection>
  );
}

/** The placeholders a pattern can use; pressing one adds it to the end of the pattern. */
function PatternTokens({ index, type }: { index: number; type: SequenceType }) {
  const t = useT();
  const { getValues, setValue } = useFormContext<SequenceConfigDocument>();
  const name = `configs.${index}.customFormat` as const;
  const tokens = drawsRecordCodes(type) ? [...patternTokens, ...recordCodeTokens] : patternTokens;

  return (
    <SettingsReadout>
      <ul aria-label={t("Placeholders")} className="flex flex-wrap gap-1">
        {tokens.map((letter) => (
          <li key={letter}>
            <button
              type="button"
              aria-label={t("Add {0} to the pattern", patternTokenLabels[letter])}
              onClick={() =>
                setValue(name, `${getValues(name)}{${letter}}`, {
                  shouldDirty: true,
                  shouldValidate: true,
                })
              }
              className="ui-focus-ring inline-flex h-6 items-center gap-1.5 rounded-badge border border-border pr-2.5 pl-1.5 text-xs text-muted-foreground transition-colors hover:border-border-strong hover:text-foreground"
            >
              <code className="font-mono text-foreground">{`{${letter}}`}</code>
              {patternTokenLabels[letter]}
            </button>
          </li>
        ))}
      </ul>
    </SettingsReadout>
  );
}
