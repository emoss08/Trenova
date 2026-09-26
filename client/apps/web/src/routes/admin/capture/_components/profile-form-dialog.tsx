import { InputField } from "@/components/fields/input-field";
import { MultiCheckboxField } from "@/components/fields/multi-checkbox-field";
import { NumberField } from "@/components/fields/number-field";
import { SelectField } from "@/components/fields/select-field";
import { SwitchField } from "@/components/fields/switch-field";
import { TextareaField } from "@/components/fields/textarea-field";
import { useApiMutation } from "@/hooks/use-api-mutation";
import { capturePixelTypeLabel, captureSeparatorLabel } from "@/lib/capture";
import {
  createCaptureProfile,
  updateCaptureProfile,
  type CaptureProfile,
} from "@/lib/graphql/capture";
import { zodResolver } from "@hookform/resolvers/zod";
import { Button } from "@trenova/shared/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@trenova/shared/components/ui/dialog";
import { FormControl, FormGroup } from "@trenova/shared/components/ui/form";
import { useT } from "@trenova/shared/i18n/use-t";
import { useEffect } from "react";
import { useForm, useWatch } from "react-hook-form";
import { toast } from "sonner";
import {
  PIXEL_TYPES,
  PROFILE_RESOLUTIONS,
  PROFILE_STATUSES,
  SEPARATOR_STRATEGIES,
  newProfileDefaults,
  profileFormSchema,
  profileFormValues,
  profileInput,
  type ProfileFormInput,
  type ProfileFormValues,
} from "./profile-schema";

/**
 * Creates or edits a scanning profile. A change reaches the next scan started
 * with it; a stack already captured keeps the settings it was captured with.
 */
export function ProfileFormDialog({
  open,
  profile,
  onClose,
  onSaved,
}: {
  open: boolean;
  profile: CaptureProfile | null;
  onClose: () => void;
  onSaved: () => Promise<void> | void;
}) {
  const t = useT();
  const editing = profile !== null;

  const form = useForm<ProfileFormInput, unknown, ProfileFormValues>({
    resolver: zodResolver(profileFormSchema),
    defaultValues: profile === null ? newProfileDefaults() : profileFormValues(profile),
  });

  useEffect(() => {
    if (open) {
      form.reset(profile === null ? newProfileDefaults() : profileFormValues(profile));
    }
  }, [open, profile, form]);

  const separators = useWatch({ control: form.control, name: "separatorStrategies" });
  const pixelType = useWatch({ control: form.control, name: "pixelType" });

  const save = useApiMutation({
    mutationFn: (values: ProfileFormValues) =>
      profile === null
        ? createCaptureProfile(profileInput(values))
        : updateCaptureProfile(profile.id, profile.version, profileInput(values)),
    onSuccess: async () => {
      toast.success(editing ? t("Profile saved") : t("Profile created"));
      onClose();
      await onSaved();
    },
    form,
    resourceName: "Scan profile",
  });

  const submit = form.handleSubmit((values) => save.mutate(values));

  return (
    <Dialog open={open} onOpenChange={(next) => !next && onClose()}>
      <DialogContent size="lg">
        <form onSubmit={submit}>
          <DialogHeader>
            <DialogTitle>
              {editing ? t("Edit {0}", profile.name) : t("New scan profile")}
            </DialogTitle>
            <DialogDescription>
              {t(
                "Settings people pick from when they start a scan. The scanner is asked for each; one it cannot do is noted on the stack, not refused.",
              )}
            </DialogDescription>
          </DialogHeader>

          <FormGroup cols={2} className="py-4">
            <FormControl>
              <InputField
                control={form.control}
                name="name"
                label={t("Name")}
                placeholder={t("Paperwork")}
                rules={{ required: true }}
              />
            </FormControl>
            <FormControl>
              <SelectField
                control={form.control}
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
                control={form.control}
                name="description"
                label={t("Description")}
                placeholder={t("What it is for, so people pick the right one")}
              />
            </FormControl>
            <FormControl>
              <SelectField
                control={form.control}
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
                control={form.control}
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
                  control={form.control}
                  name="jpegQuality"
                  label={t("Image quality")}
                  description={t("Between 30 and 95. Higher is sharper and larger.")}
                />
              </FormControl>
            )}
            <FormControl cols="full">
              <MultiCheckboxField
                control={form.control}
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
                  control={form.control}
                  name="fixedPageCount"
                  label={t("Pages per document")}
                />
              </FormControl>
            )}
            <FormControl cols="full">
              <SwitchField
                control={form.control}
                name="duplex"
                label={t("Both sides")}
                description={t("Scan the back of every page as well as the front.")}
                position="left"
              />
            </FormControl>
            <FormControl cols="full">
              <SwitchField
                control={form.control}
                name="useFeeder"
                label={t("Use the document feeder")}
                description={t("Feed a stack rather than scanning one page from the glass.")}
                position="left"
              />
            </FormControl>
            <FormControl cols="full">
              <SwitchField
                control={form.control}
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
                control={form.control}
                name="showDriverUi"
                label={t("Show the scanner's own window")}
                description={t(
                  "For scanners that need their settings chosen on the computer, such as Kofax VRS. The person scanning confirms each scan there.",
                )}
                position="left"
              />
            </FormControl>
            <FormControl cols="full">
              <SwitchField
                control={form.control}
                name="isDefault"
                label={t("Use when nobody picks one")}
                description={t("Making this the default takes it off the one that had it.")}
                position="left"
              />
            </FormControl>
          </FormGroup>

          <DialogFooter>
            <Button type="button" variant="outline" onClick={onClose}>
              {t("Cancel")}
            </Button>
            <Button type="submit" isLoading={save.isPending} loadingText={t("Saving…")}>
              {editing ? t("Save changes") : t("Create profile")}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  );
}
