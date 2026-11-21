import { SettingRow, settingRowDescriptionClass } from "@/components/fields/field-components";
import { InputField } from "@/components/fields/input-field";
import { NumberField } from "@/components/fields/number-field";
import { SegmentedField } from "@/components/fields/segmented-field";
import { SelectField } from "@/components/fields/select-field";
import { ToggleChipsField } from "@/components/fields/toggle-chips-field";
import { SettingsSection } from "@/components/settings/settings-section";
import { useErrorSurface } from "@/hooks/use-error-surface";
import {
  locationCodeComponents,
  MAX_LOCATION_CODE_LENGTH,
  type SequenceConfigDocument,
} from "@/types/sequence-config";
import { useT } from "@trenova/shared/i18n/use-t";
import { useFormContext, useFormState } from "react-hook-form";
import {
  casingLabels,
  locationComponentLabels,
  locationSeparatorChoices,
  sequenceDescriptions,
  sequenceTitles,
} from "./sequence-config-constants";
import { SequencePreview } from "./sequence-preview";

/**
 * How a new location's code is built: parts of its name and address cut to a width, then a
 * counter. The code under the title is the one the sample location would get.
 */
export function LocationCodeSection({ index, now }: { index: number; now: Date }) {
  const t = useT();
  const { control } = useFormContext<SequenceConfigDocument>();
  const strategy = `configs.${index}.locationCodeStrategy` as const;

  // The server refuses a strategy as a whole; that refusal is said here, above its settings.
  useErrorSurface(control, strategy);
  const { errors } = useFormState({ control, name: strategy });
  const refusal = errors.configs?.[index]?.locationCodeStrategy?.root?.message;

  return (
    <SettingsSection
      title={sequenceTitles.location_code}
      description={sequenceDescriptions.location_code}
    >
      <SequencePreview index={index} now={now} />
      {refusal ? (
        <p role="alert" className="px-3.5 py-2.5 text-xs text-danger-foreground">
          {refusal}
        </p>
      ) : null}
      <SettingRow
        label={t("Parts")}
        controlSize="auto"
        description={
          <p className={settingRowDescriptionClass}>
            {t("What the code is built from, in the order you choose them.")}
          </p>
        }
      >
        <ToggleChipsField
          control={control}
          name={`${strategy}.components`}
          label={t("Parts")}
          options={locationCodeComponents.map((component) => ({
            value: component,
            label: locationComponentLabels[component],
          }))}
          required
        />
      </SettingRow>
      <NumberField
        control={control}
        name={`${strategy}.componentWidth`}
        label={t("Characters per part")}
        description={t(
          "How much of each part is kept. Spaces and punctuation are left out first.",
        )}
        min={1}
        max={10}
        required
      />
      <NumberField
        control={control}
        name={`${strategy}.sequenceDigits`}
        label={t("Counter digits")}
        description={t(
          "How many digits the counter is padded to. The whole code can be at most {0} characters.",
          MAX_LOCATION_CODE_LENGTH,
        )}
        sideText={t("digits")}
        min={1}
        max={10}
        required
      />
      <SelectField
        control={control}
        name={`${strategy}.separator`}
        label={t("Separator")}
        description={t("The character between the parts and before the counter.")}
        options={locationSeparatorChoices}
        clearable={false}
      />
      <SegmentedField
        control={control}
        name={`${strategy}.casing`}
        label={t("Letter case")}
        options={[
          { value: "upper", label: casingLabels.upper },
          { value: "lower", label: casingLabels.lower },
        ]}
        fullWidth={false}
        required
      />
      <InputField
        control={control}
        name={`${strategy}.fallbackPrefix`}
        label={t("Fallback")}
        description={t("Used in place of a part the location does not have, such as a ZIP code.")}
        placeholder={t("e.g. LOC")}
        maxLength={MAX_LOCATION_CODE_LENGTH}
        required
      />
    </SettingsSection>
  );
}
