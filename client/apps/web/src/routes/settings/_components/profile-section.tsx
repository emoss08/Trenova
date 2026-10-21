import { UserAutocompleteField } from "@/components/autocomplete-fields";
import { DatePickerField } from "@/components/fields/date-picker-field";
import { FieldWrapper } from "@/components/fields/field-components";
import { InputField } from "@/components/fields/input-field";
import { PhoneNumberField } from "@/components/fields/phone-number-field";
import { SegmentedField } from "@/components/fields/segmented-field";
import { ResolvedUserAvatar } from "@/components/resolved-user-avatar";
import { FOCUS_CHOICES, focusChoiceFor, focusUntil, type FocusChoice } from "@/lib/user-settings";
import { apiService } from "@/services/api";
import { Badge } from "@trenova/shared/components/ui/badge";
import { Button } from "@trenova/shared/components/ui/button";
import {
  Camera01Icon,
  Copy01Icon,
  InfoCircleIcon,
  Lock01Icon,
  Trash01Icon,
} from "@trenova/shared/components/icons";
import { useNow } from "@trenova/shared/hooks/use-now";
import { useT } from "@trenova/shared/i18n/use-t";
import { resolveUserTimezone } from "@trenova/shared/lib/date";
import { startOfDay } from "date-fns";
import { useAuthStore } from "@trenova/shared/stores/auth-store";
import type { AvailabilityStatus, User } from "@trenova/shared/types/user";
import { useForm, useWatch } from "react-hook-form";
import { AvatarPicker } from "./avatar-picker";
import { useSettingsFeedback } from "./settings-feedback";
import { SettingsCard, SettingsRow } from "./settings-layout";
import { useAutosaveForm } from "./use-autosave-form";
import type { ProfilePictureControls } from "./use-profile-picture";

type AboutValues = {
  name: string;
  username: string;
  emailAddress: string;
  jobTitle: string;
  phoneNumber: string;
  avatarStyle: string;
};

type AvailabilityValues = {
  status: AvailabilityStatus;
  focusFor: FocusChoice;
  backOn: number | null;
  delegateUserId: string | null;
  awayNote: string;
};

const ABOUT_TEXT_FIELDS = ["jobTitle", "phoneNumber"] as const;
const AVAILABILITY_TEXT_FIELDS = ["awayNote"] as const;

function aboutValues(user: User | null): AboutValues {
  return {
    name: user?.name ?? "",
    username: user?.username ?? "",
    emailAddress: user?.emailAddress ?? "",
    jobTitle: user?.jobTitle ?? "",
    phoneNumber: user?.phoneNumber ?? "",
    avatarStyle: user?.avatarStyle ?? "auto",
  };
}

function availabilityValues(user: User | null): AvailabilityValues {
  return {
    status: user?.availabilityStatus ?? "available",
    focusFor: focusChoiceFor(user?.focusUntil, new Date()),
    backOn: user?.awayBackOn ?? null,
    delegateUserId: user?.awayDelegateUserId ?? null,
    awayNote: user?.awayNote ?? "",
  };
}

function AboutCard({ user, picture }: { user: User | null; picture: ProfilePictureControls }) {
  const t = useT();
  const feedback = useSettingsFeedback();
  const form = useForm<AboutValues>({ defaultValues: aboutValues(user) });
  const avatarStyle = useWatch({ control: form.control, name: "avatarStyle" });
  const hasPhoto = Boolean(user?.profilePicUrl || user?.thumbnailUrl);

  const { commit } = useAutosaveForm({
    form,
    resourceName: "Profile",
    textFields: ABOUT_TEXT_FIELDS,
    save: async (next, previous) => {
      const updated = await apiService.userService.updateMyProfile({
        ...(next.jobTitle !== previous.jobTitle ? { jobTitle: next.jobTitle } : {}),
        ...(next.phoneNumber !== previous.phoneNumber ? { phoneNumber: next.phoneNumber } : {}),
        ...(next.avatarStyle !== previous.avatarStyle ? { avatarStyle: next.avatarStyle } : {}),
      });
      useAuthStore.getState().setUser(updated);
    },
  });

  const copyUsername = async () => {
    try {
      await navigator.clipboard.writeText(`@${user?.username ?? ""}`);
      feedback.message(t("Copied @{0}", user?.username ?? ""));
    } catch {
      feedback.message(t("Could not copy the username"));
    }
  };

  return (
    <SettingsCard
      title={t("About you")}
      description={t("Shown to your team on comments, approvals and assignments.")}
    >
      <SettingsRow id="photo">
        <FieldWrapper
          layout="inline"
          label={t("Photo")}
          description={t("Square crop. JPG, PNG or WEBP.")}
        >
          <div className="flex flex-wrap items-center gap-2">
            <ResolvedUserAvatar
              userId={user?.id}
              name={user?.name}
              profilePicUrl={user?.profilePicUrl}
              thumbnailUrl={user?.thumbnailUrl}
              className="size-9"
            />
            <Button type="button" variant="outline" size="sm" onClick={picture.openPicker}>
              <Camera01Icon className="size-3.5" />
              {hasPhoto ? t("Change") : t("Upload")}
            </Button>
            {hasPhoto ? (
              <Button
                type="button"
                variant="ghost"
                size="icon-sm"
                aria-label={t("Remove photo")}
                disabled={picture.removing}
                onClick={() => void picture.remove()}
              >
                <Trash01Icon className="size-3.5" />
              </Button>
            ) : null}
          </div>
        </FieldWrapper>
      </SettingsRow>
      {!hasPhoto ? (
        <SettingsRow id="avatar" sub>
          <FieldWrapper
            layout="inline"
            label={t("Avatar")}
            description={t(
              "No photo? Pick a gradient. Auto is made from your username, so it never changes.",
            )}
          >
            <AvatarPicker
              seed={user?.username || user?.id || ""}
              value={avatarStyle}
              onChange={(style) => {
                form.setValue("avatarStyle", style, { shouldDirty: true });
                commit();
              }}
            />
          </FieldWrapper>
        </SettingsRow>
      ) : null}
      <SettingsRow id="name">
        <InputField
          control={form.control}
          name="name"
          layout="inline"
          label={t("Display name")}
          description={t("Managed by your organization. Ask an admin to change it.")}
          readOnly
        />
      </SettingsRow>
      <SettingsRow id="job-title">
        <InputField
          control={form.control}
          name="jobTitle"
          layout="inline"
          label={t("Job title")}
          placeholder={t("e.g. Dispatcher")}
          maxLength={100}
          onBlur={commit}
        />
      </SettingsRow>
      <SettingsRow id="username">
        <InputField
          control={form.control}
          name="username"
          layout="inline"
          label={t("Username")}
          readOnly
          rightElement={
            <Button
              type="button"
              variant="ghost"
              size="icon-xs"
              aria-label={t("Copy username")}
              onClick={() => void copyUsername()}
            >
              <Copy01Icon className="size-3" />
            </Button>
          }
        />
      </SettingsRow>
      <SettingsRow id="email">
        <InputField
          control={form.control}
          name="emailAddress"
          layout="inline"
          label={t("Email")}
          description={t("Managed by your organization. Ask an admin to change it.")}
          readOnly
          rightElement={
            <Badge variant="neutral" className="mr-1">
              <Lock01Icon />
              {t("Managed")}
            </Badge>
          }
        />
      </SettingsRow>
      <SettingsRow id="phone">
        <PhoneNumberField
          control={form.control}
          name="phoneNumber"
          layout="inline"
          label={t("Phone")}
          description={t("Optional. Only your teammates see it.")}
          placeholder="(312) 555-0142"
          onBlur={commit}
        />
      </SettingsRow>
    </SettingsCard>
  );
}

function AvailabilityCard({ user }: { user: User | null }) {
  const t = useT();
  const form = useForm<AvailabilityValues>({ defaultValues: availabilityValues(user) });
  const status = useWatch({ control: form.control, name: "status" });
  const timezone = resolveUserTimezone(user?.timezone);
  const now = useNow();

  const { commit } = useAutosaveForm({
    form,
    resourceName: "Availability",
    textFields: AVAILABILITY_TEXT_FIELDS,
    save: async (next) => {
      const updated = await apiService.userService.updateMyAvailability(
        next.status === "focus"
          ? { status: "focus", focusUntil: focusUntil(next.focusFor, new Date(), timezone) }
          : next.status === "away"
            ? {
                status: "away",
                backOn: next.backOn,
                delegateUserId: next.delegateUserId || null,
                awayNote: next.awayNote,
              }
            : { status: "available" },
      );
      useAuthStore.getState().setUser(updated);
    },
  });

  return (
    <SettingsCard
      id="availability"
      title={t("Availability")}
      description={t("Let your team know where you are. Notifications follow it.")}
    >
      <SettingsRow id="status">
        <SegmentedField
          control={form.control}
          name="status"
          layout="inline"
          label={t("Status")}
          options={[
            { value: "available", label: t("Available") },
            { value: "focus", label: t("Focus") },
            { value: "away", label: t("Away") },
          ]}
        />
      </SettingsRow>
      {status === "focus" ? (
        <SettingsRow sub>
          <SegmentedField
            control={form.control}
            name="focusFor"
            layout="inline"
            label={t("Focus for")}
            description={t("Only urgent notifications and direct @mentions reach you.")}
            options={FOCUS_CHOICES.map((choice) => ({
              value: choice,
              label:
                choice === "1h" ? t("1 hour") : choice === "2h" ? t("2 hours") : t("Rest of day"),
            }))}
          />
        </SettingsRow>
      ) : null}
      {status === "away" ? (
        <>
          <SettingsRow sub>
            <DatePickerField
              control={form.control}
              name="backOn"
              layout="inline"
              label={t("Back on")}
              description={t("You show as available again from this day.")}
              fromDate={startOfDay(now * 1000)}
            />
          </SettingsRow>
          <SettingsRow sub id="hand-off">
            <UserAutocompleteField
              control={form.control}
              name="delegateUserId"
              layout="inline"
              label={t("Hand off to")}
              description={t(
                "Gets your approvals, tender reviews and @mentions while you're away.",
              )}
              placeholder={t("No one, hold them for me")}
              clearable
              filterOption={(option) => option.id !== user?.id}
            />
          </SettingsRow>
          <SettingsRow sub>
            <InputField
              control={form.control}
              name="awayNote"
              layout="inline"
              label={t("Away note")}
              description={t("Shown when someone @mentions you.")}
              placeholder={t("Out until Monday. Jordan has my loads.")}
              maxLength={280}
              onBlur={commit}
            />
          </SettingsRow>
        </>
      ) : null}
      {status !== "available" ? (
        <SettingsRow sub className="text-foreground-subtle flex items-start gap-2 text-xs">
          <InfoCircleIcon className="text-brand mt-0.5 size-3.5 shrink-0" />
          <span>
            {status === "focus"
              ? t("Everything else waits in the bell until your focus ends.")
              : t("Everything still lands in your bell; nothing is lost while you're away.")}
          </span>
        </SettingsRow>
      ) : null}
    </SettingsCard>
  );
}

export function ProfileSection({ picture }: { picture: ProfilePictureControls }) {
  const user = useAuthStore((state) => state.user);

  return (
    <>
      <AboutCard user={user} picture={picture} />
      <AvailabilityCard user={user} />
    </>
  );
}
