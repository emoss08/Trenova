import { InputField } from "@/components/fields/input-field";
import { MultiCheckboxField } from "@/components/fields/multi-checkbox-field";
import { NumberField } from "@/components/fields/number-field";
import { SelectField } from "@/components/fields/select-field";
import { SwitchField } from "@/components/fields/switch-field";
import { TextareaField } from "@/components/fields/textarea-field";
import { capturePixelTypeLabel, captureSeparatorLabel } from "@/lib/capture";
import { FormControl, FormGroup, FormSection } from "@trenova/shared/components/ui/form";
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
    <div className="flex flex-col gap-6">
      <FormSection title={t("Profile details")}>
        <FormGroup cols={2}>
          <FormControl>
            <InputField
              control={control}
              name="name"
              label={t("Name")}
              placeholder={t("Paperwork")}
              rules={{ required: true }}
              maxLength={100}
            />
          </FormControl>
          <FormControl>
            <SelectField
              control={control}
              name="status"
              label={t("Status")}
              options={PROFILE_STATUSES.map((value) => ({
                value,
                label: value === "Active" ? t("Offered") : t("Not offered"),
              }))}
            />
          </FormControl>
          <FormControl cols="full">
            <TextareaField
              control={control}
              name="description"
              label={t("Description")}
              placeholder={t("What it is for, so people pick the right one")}
              maxLength={500}
            />
          </FormControl>
          <FormControl cols="full">
            <SwitchField
              control={control}
              name="isDefault"
              label={t("Use when nobody picks one")}
              description={t("Making this the default takes it off the one that had it.")}
              position="left"
            />
          </FormControl>
        </FormGroup>
      </FormSection>

      <FormSection
        title={t("Scan settings")}
        description={t(
          "Settings people pick from when they start a scan. The scanner is asked for each; one it cannot do is noted on the stack, not refused.",
        )}
      >
        <FormGroup cols={2}>
          <FormControl>
            <SelectField
              control={control}
              name="dpi"
              label={t("Resolution")}
              options={PROFILE_RESOLUTIONS.map((value) => ({
                value,
                label: t("{0} DPI", value),
              }))}
              description={t("300 reads well and keeps a page small.")}
            />
          </FormControl>
          <FormControl>
            <SelectField
              control={control}
              name="pixelType"
              label={t("Color mode")}
              options={PIXEL_TYPES.map((value) => ({
                value,
                label: capturePixelTypeLabel(t, value),
              }))}
            />
          </FormControl>
          {pixelType !== "BlackWhite" && (
            <FormControl>
              <NumberField
                control={control}
                name="jpegQuality"
                label={t("Image quality")}
                description={t("Between 30 and 95. Higher is sharper and larger.")}
                min={30}
                max={95}
              />
            </FormControl>
          )}
          <FormControl cols="full">
            <SwitchField
              control={control}
              name="duplex"
              label={t("Both sides")}
              description={t("Scan the back of every page as well as the front.")}
              position="left"
            />
          </FormControl>
          <FormControl cols="full">
            <SwitchField
              control={control}
              name="useFeeder"
              label={t("Use the document feeder")}
              description={t("Feed a stack rather than scanning one page from the glass.")}
              position="left"
            />
          </FormControl>
          <FormControl cols="full">
            <SwitchField
              control={control}
              name="discardBlankPages"
              label={t("Drop blank pages")}
              description={t(
                "Ask the scanner to leave out empty backs. Trenova sets aside any it still sends.",
              )}
              position="left"
            />
          </FormControl>
          <FormControl cols="full">
            <SwitchField
              control={control}
              name="showDriverUi"
              label={t("Show the scanner's own window")}
              description={t(
                "For scanners that need their settings chosen on the computer, such as Kofax VRS. The person scanning confirms each scan there.",
              )}
              position="left"
            />
          </FormControl>
          <FormControl cols="full">
            <MultiCheckboxField
              control={control}
              name="separatorStrategies"
              label={t("Split the stack on")}
              description={t(
                "How Trenova tells where one document ends and the next begins. Pages it splits on are set aside.",
              )}
              options={SEPARATOR_STRATEGIES.map((value) => ({
                value,
                label: captureSeparatorLabel(t, value),
              }))}
            />
          </FormControl>
          {(separators ?? []).includes("FixedPageCount") && (
            <FormControl>
              <NumberField
                control={control}
                name="fixedPageCount"
                label={t("Pages per document")}
                min={1}
                max={500}
              />
            </FormControl>
          )}
        </FormGroup>
      </FormSection>
    </div>
  );
}
