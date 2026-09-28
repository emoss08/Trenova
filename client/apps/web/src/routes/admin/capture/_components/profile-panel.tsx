import { FormCreatePanel } from "@/components/form-create-panel";
import { FormEditPanel } from "@/components/form-edit-panel";
import {
  createCaptureProfile,
  updateCaptureProfile,
  type CaptureProfile,
} from "@/lib/graphql/capture";
import { zodResolver } from "@hookform/resolvers/zod";
import { useT } from "@trenova/shared/i18n/use-t";
import { useMemo } from "react";
import { useForm } from "react-hook-form";
import { ProfileForm } from "./profile-form";
import {
  newProfileDefaults,
  profileFormSchema,
  profileInput,
  profileRecord,
  type ProfileFormInput,
  type ProfileFormValues,
  type ProfileRecord,
} from "./profile-schema";

export type ProfilePanelMode = "create" | "edit";

type ProfilePanelProps = {
  mode: ProfilePanelMode;
  open: boolean;
  onOpenChange: (open: boolean) => void;
  /**
   * The profile being edited. The owner keeps it after the panel closes, so the
   * title does not change while the panel slides away.
   */
  profile: CaptureProfile | null;
  onSaved: (profile: CaptureProfile) => Promise<void>;
};

const PANEL_QUERY_KEY = "capture-profile-panel";

/**
 * Creates or edits a scanning profile. A change reaches the next scan started
 * with it; a stack already captured keeps the settings it was captured with.
 */
export function ProfilePanel({ mode, open, onOpenChange, profile, onSaved }: ProfilePanelProps) {
  const t = useT();

  const form = useForm<ProfileFormInput, unknown, ProfileFormValues>({
    resolver: zodResolver(profileFormSchema),
    defaultValues: newProfileDefaults(),
  });

  const row = useMemo(() => (profile === null ? null : profileRecord(profile)), [profile]);

  if (mode === "edit") {
    return (
      <FormEditPanel<ProfileFormInput, ProfileRecord, ProfileFormValues, CaptureProfile>
        open={open}
        onOpenChange={onOpenChange}
        row={row}
        form={form}
        queryKey={PANEL_QUERY_KEY}
        title={t("Scan profile")}
        fieldKey="name"
        size="lg"
        formComponent={<ProfileForm />}
        mutationFn={async (values, current) => {
          const saved = await updateCaptureProfile(
            current.id,
            current.version,
            profileInput(values),
          );
          await onSaved(saved);
          return saved;
        }}
      />
    );
  }

  return (
    <FormCreatePanel<ProfileFormInput, CaptureProfile, ProfileFormValues, CaptureProfile>
      open={open}
      onOpenChange={onOpenChange}
      form={form}
      queryKey={PANEL_QUERY_KEY}
      title={t("Scan profile")}
      description={t(
        "What people pick from when they start a scan. The default is used when nobody picks one.",
      )}
      size="lg"
      formComponent={<ProfileForm />}
      mutationFn={async (values) => {
        const saved = await createCaptureProfile(profileInput(values));
        await onSaved(saved);
        return saved;
      }}
    />
  );
}
