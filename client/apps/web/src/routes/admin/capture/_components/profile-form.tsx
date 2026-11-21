import { InputField } from "@/components/fields/input-field";
import { MultiCheckboxField } from "@/components/fields/multi-checkbox-field";
import { NumberField } from "@/components/fields/number-field";
import { SelectField } from "@/components/fields/select-field";
import { SwitchField } from "@/components/fields/switch-field";
import { TextareaField } from "@/components/fields/textarea-field";
import { capturePixelTypeLabel, captureSeparatorLabel } from "@/lib/capture";
import {
  SettingsNested,
  SettingsSection,
  SettingsSections,
} from "@/components/settings/settings-section";
import { useT } from "@trenova/shared/i18n/use-t";
import { useFormContext, useWatch } from "react-hook-form";
import {
  PIXEL_TYPES,
  PROFILE_RESOLUTIONS,
  PROFILE_STATUSES,
  SEPARATOR_STRATEGIES,
  type ProfileFormInput,
  type ProfileFormValues,
} from "./profile-schema";

/**
 * The fields of a scanning profile, shared by the create and edit panels. The
 * scanner is asked for each setting; one it cannot do is noted on the stack,
 * not refused.
 */
export function ProfileForm() {
  const t = useT();
  const { control } = useFormContext<ProfileFormInput, unknown, ProfileFormValues>();
  const separators = useWatch({ control, name: "separatorStrategies" });
  const pixelType = useWatch({ control, name: "pixelType" });

  return (
    <SettingsSections>
      <SettingsSection title={t("Profile")}>
        <InputField
          control={control}
          name="name"
          label={t("Name")}
          description={t("What people see when they pick a profile.")}
          placeholder={t("Paperwork")}
          required
          maxLength={100}
        />
        <TextareaField
          control={control}
          name="description"
          label={t("Description")}
          description={t("What it is for, so people pick the right one.")}
          placeholder={t("Proofs of delivery and bills of lading")}
          maxLength={500}
        />
        <SelectField
          control={control}
          name="status"
          label={t("Status")}
          description={t("A profile that is not offered stays here, but nobody can pick it.")}
          options={PROFILE_STATUSES.map((value) => ({
            value,
            label: value === "Active" ? t("Offered") : t("Not offered"),
          }))}
          clearable={false}
        />
        <SwitchField
          control={control}
          name="isDefault"
          label={t("Use when nobody picks one")}
          description={t("Making this the default takes it off the one that had it.")}
        />
      </SettingsSection>

      <SettingsSection
        title={t("Scanning")}
        description={t(
          "The scanner is asked for each setting; one it cannot do is noted on the stack, not refused.",
        )}
      >
        <SelectField
          control={control}
          name="dpi"
          label={t("Resolution")}
          description={t("300 DPI reads well and keeps a page small.")}
          options={PROFILE_RESOLUTIONS.map((value) => ({
            value,
            label: t("{0} DPI", value),
          }))}
          clearable={false}
        />
        <SelectField
          control={control}
          name="pixelType"
          label={t("Color mode")}
          description={t("Black and white keeps a page of text smallest.")}
          options={PIXEL_TYPES.map((value) => ({
            value,
            label: capturePixelTypeLabel(t, value),
          }))}
          clearable={false}
        />
        <SettingsNested show={pixelType !== "BlackWhite"}>
          <NumberField
            control={control}
            name="jpegQuality"
            label={t("Image quality")}
            description={t("From 30 to 95. Higher is sharper and larger.")}
            min={30}
            max={95}
            required
          />
        </SettingsNested>
        <SwitchField
          control={control}
          name="duplex"
          label={t("Both sides")}
          description={t("Scan the back of every page as well as the front.")}
        />
        <SwitchField
          control={control}
          name="useFeeder"
          label={t("Use the document feeder")}
          description={t("Feed a stack rather than scanning one page from the glass.")}
        />
        <SwitchField
          control={control}
          name="discardBlankPages"
          label={t("Drop blank pages")}
          description={t(
            "Ask the scanner to leave out empty backs. Trenova sets aside any it still sends.",
          )}
        />
        <SwitchField
          control={control}
          name="showDriverUi"
          label={t("Show the scanner's own window")}
          description={t(
            "For scanners that need their settings chosen on the computer, such as Kofax VRS. The person scanning confirms each scan there.",
          )}
        />
      </SettingsSection>

      <SettingsSection
        title={t("Splitting")}
        description={t(
          "How Trenova tells where one document ends and the next begins. Pages it splits on are set aside.",
        )}
      >
        <MultiCheckboxField
          control={control}
          name="separatorStrategies"
          label={t("Split the stack on")}
          options={SEPARATOR_STRATEGIES.map((value) => ({
            value,
            label: captureSeparatorLabel(t, value),
          }))}
        />
        <SettingsNested show={(separators ?? []).includes("FixedPageCount")}>
          <NumberField
            control={control}
            name="fixedPageCount"
            label={t("Pages per document")}
            description={t("From 1 to 500. Every so many pages starts a new document.")}
            min={1}
            max={500}
            required
          />
        </SettingsNested>
      </SettingsSection>
    </SettingsSections>
  );
}
