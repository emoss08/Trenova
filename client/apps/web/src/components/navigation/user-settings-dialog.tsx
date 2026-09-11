import { useT } from "@trenova/shared/i18n/use-t";
import { DEFAULT_LOCALE } from "@trenova/shared/i18n/generated/locales";
import { SelectField } from "@/components/fields/select-field";
import { SensitiveField } from "@/components/fields/sensitive-field";
import { ImageCropUploadDialog } from "@/components/image-crop-upload-dialog";
import { ResolvedUserAvatar } from "@/components/resolved-user-avatar";
import { useApiMutation } from "@/hooks/use-api-mutation";
import { localeChoices, timeFormatChoices, timezoneGroupedChoices } from "@/lib/choices";
import { validateCroppableImage } from "@/lib/images/crop-image";
import { IMAGE_UPLOAD_ACCEPT, profilePictureCropConfig } from "@/lib/images/upload-config";
import { queries } from "@/lib/queries";
import { apiService } from "@/services/api";
import { useQueryClient } from "@tanstack/react-query";
import { Button } from "@trenova/shared/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@trenova/shared/components/ui/dialog";
import { Form, FormControl, FormGroup } from "@trenova/shared/components/ui/form";
import { Separator } from "@trenova/shared/components/ui/separator";
import { useAuthStore } from "@trenova/shared/stores/auth-store";
import type { ChangeMyPassword, UpdateMySettings, User } from "@trenova/shared/types/user";
import { Camera, Globe, KeyRound, Mail, Trash2 } from "lucide-react";
import type { ChangeEvent, ComponentType } from "react";
import { useCallback, useEffect, useRef, useState } from "react";
import { useForm } from "react-hook-form";
import { toast } from "sonner";

type UserSettingsDialogProps = {
  open: boolean;
  onOpenChange: (open: boolean) => void;
};

function SectionHeader({
  icon: Icon,
  title,
  description,
}: {
  icon: ComponentType<{ className?: string }>;
  title: string;
  description: string;
}) {
  return (
    <div className="flex items-center gap-3">
      <div className="bg-muted flex size-8 shrink-0 items-center justify-center rounded-lg">
        <Icon className="text-muted-foreground size-4" />
      </div>
      <div>
        <h3 className="text-sm leading-none font-medium">{title}</h3>
        <p className="text-muted-foreground mt-1 text-xs">{description}</p>
      </div>
    </div>
  );
}

export function UserSettingsDialog({ open, onOpenChange }: UserSettingsDialogProps) {
  const t = useT();

  const user = useAuthStore((s) => s.user);
  const queryClient = useQueryClient();
  const fileInputRef = useRef<HTMLInputElement | null>(null);
  const [pendingFile, setPendingFile] = useState<File | null>(null);
  const [isCropOpen, setIsCropOpen] = useState(false);
  const [isRemovingProfilePicture, setIsRemovingProfilePicture] = useState(false);

  const settingsForm = useForm<UpdateMySettings>({
    defaultValues: {
      timezone: user?.timezone ?? "",
      timeFormat: user?.timeFormat ?? "12-hour",
      locale: user?.locale ?? DEFAULT_LOCALE,
    },
  });

  const passwordForm = useForm<ChangeMyPassword>({
    defaultValues: {
      currentPassword: "",
      newPassword: "",
      confirmPassword: "",
    },
  });

  useEffect(() => {
    settingsForm.reset({
      timezone: user?.timezone ?? "",
      timeFormat: user?.timeFormat ?? "12-hour",
      locale: user?.locale ?? DEFAULT_LOCALE,
    });
  }, [settingsForm, user]);

  const { mutateAsync: updateSettings } = useApiMutation<
    User,
    UpdateMySettings,
    unknown,
    UpdateMySettings
  >({
    mutationFn: (values) => apiService.userService.updateMySettings(values),
    resourceName: "User Settings",
    form: settingsForm,
    onSuccess: (updatedUser) => {
      useAuthStore.getState().setUser(updatedUser);
      toast.success(t("Settings updated"), {
        description: t("Your profile settings have been saved."),
      });
    },
  });

  const { mutateAsync: changePassword } = useApiMutation<
    User,
    ChangeMyPassword,
    unknown,
    ChangeMyPassword
  >({
    mutationFn: (values) => apiService.userService.changeMyPassword(values),
    resourceName: "Change Password",
    form: passwordForm,
    onSuccess: () => {
      toast.success(t("Password changed"), {
        description: t("Your password has been updated successfully."),
      });
    },
  });

  const handleClose = useCallback(() => {
    onOpenChange(false);
    passwordForm.reset();
    setPendingFile(null);
    setIsCropOpen(false);
  }, [onOpenChange, passwordForm]);

  const syncUserSettings = useCallback(
    async (updatedUser: User) => {
      useAuthStore.getState().setUser(updatedUser);
      if (updatedUser.id) {
        await Promise.all([
          queryClient.invalidateQueries({
            queryKey: queries.user.profilePicture(updatedUser.id, "thumbnail").queryKey,
          }),
          queryClient.invalidateQueries({
            queryKey: queries.user.profilePicture(updatedUser.id, "full").queryKey,
          }),
        ]);
      }
    },
    [queryClient],
  );

  const handleFileSelection = useCallback((event: ChangeEvent<HTMLInputElement>) => {
    const selectedFile = event.target.files?.[0];
    event.target.value = "";
    if (!selectedFile) {
      return;
    }

    try {
      validateCroppableImage(selectedFile, "profile pictures");
      setPendingFile(selectedFile);
      setIsCropOpen(true);
    } catch (error) {
      toast.error(t("Unsupported profile picture"), {
        description:
          error instanceof Error ? error.message : "Please choose a JPG, PNG, or WEBP file.",
      });
    }
  }, [t]);

  const handleProfilePictureUpload = useCallback(
    async (file: File) => {
      const updatedUser = await apiService.userService.uploadMyProfilePicture(file);
      await syncUserSettings(updatedUser);
      toast.success(t("Profile picture updated"));
    },
    [syncUserSettings, t],
  );

  const handleRemoveProfilePicture = useCallback(async () => {
    if (isRemovingProfilePicture) {
      return;
    }

    setIsRemovingProfilePicture(true);
    try {
      const updatedUser = await apiService.userService.deleteMyProfilePicture();
      await syncUserSettings(updatedUser);
      toast.success(t("Profile picture removed"));
    } catch (error) {
      toast.error(t("Failed to remove profile picture"), {
        description: error instanceof Error ? error.message : "Please try again.",
      });
    }

    setIsRemovingProfilePicture(false);
  }, [isRemovingProfilePicture, syncUserSettings, t]);

  const onSubmit = useCallback(async () => {
    const settingsValid = await settingsForm.trigger();
    const hasPasswordInput =
      passwordForm.getValues("currentPassword") ||
      passwordForm.getValues("newPassword") ||
      passwordForm.getValues("confirmPassword");

    const passwordValid = hasPasswordInput ? await passwordForm.trigger() : true;

    if (!settingsValid || !passwordValid) return;

    await updateSettings(settingsForm.getValues());

    if (hasPasswordInput) {
      await changePassword(passwordForm.getValues());
      passwordForm.reset();
    }

    handleClose();
  }, [settingsForm, passwordForm, updateSettings, changePassword, handleClose]);

  const isSubmitting = settingsForm.formState.isSubmitting || passwordForm.formState.isSubmitting;

  return (
    <Dialog open={open} onOpenChange={(nextOpen) => !nextOpen && handleClose()}>
      <DialogContent className="sm:max-w-[520px]">
        <DialogHeader>
          <DialogTitle>{t("Settings")}</DialogTitle>
          <DialogDescription>{t("Manage your preferences and security.")}</DialogDescription>
        </DialogHeader>

        <div className="bg-sidebar flex items-center gap-4 rounded-md border p-4">
          <ResolvedUserAvatar
            size="lg"
            userId={user?.id}
            name={user?.name}
            profilePicUrl={user?.profilePicUrl}
            thumbnailUrl={user?.thumbnailUrl}
            fallbackClassName="rounded-md bg-linear-to-br from-sidebar-accent to-sidebar-accent/80 text-sm font-semibold text-sidebar-accent-foreground"
          />
          <div className="min-w-0 flex-1">
            <p className="truncate text-sm font-semibold">{user?.name}</p>
            <p className="text-muted-foreground truncate text-xs">@{user?.username}</p>
            <div className="text-muted-foreground mt-1 flex items-center gap-1.5 text-xs">
              <Mail className="size-3 shrink-0" />
              <span className="truncate">{user?.emailAddress}</span>
            </div>
          </div>
          <div className="flex shrink-0 gap-2 self-start">
            <Button
              type="button"
              variant="outline"
              size="sm"
              onClick={() => fileInputRef.current?.click()}
            >
              <Camera className="size-4" />
              {user?.profilePicUrl ? "Change" : "Upload"}
            </Button>
            {user?.profilePicUrl ? (
              <Button
                type="button"
                variant="ghost"
                size="icon"
                onClick={() => void handleRemoveProfilePicture()}
                disabled={isRemovingProfilePicture}
                aria-label={t("Remove profile picture")}
              >
                <Trash2 className="size-4" />
              </Button>
            ) : null}
          </div>
        </div>
        <input
          ref={fileInputRef}
          type="file"
          accept={IMAGE_UPLOAD_ACCEPT}
          className="hidden"
          onChange={handleFileSelection}
        />

        <Separator />

        <div className="space-y-5">
          <div className="space-y-3">
            <SectionHeader
              icon={Globe}
              title={t("Preferences")}
              description={t("Configure your regional and display settings.")}
            />
            <Form onSubmit={(e) => e.preventDefault()}>
              <FormGroup cols={2}>
                <FormControl>
                  <SelectField
                    control={settingsForm.control}
                    rules={{ required: true }}
                    name="timezone"
                    label={t("Timezone")}
                    placeholder={t("Select timezone")}
                    groups={timezoneGroupedChoices}
                    // isReadOnly={isDisabled}
                    renderOption={(option) => (
                      <span className="flex w-full items-center justify-between gap-3">
                        <span>{option.label}</span>
                        {option.description && (
                          <span className="text-muted-foreground text-xs">
                            {option.description}
                          </span>
                        )}
                      </span>
                    )}
                  />
                </FormControl>
                <FormControl>
                  <SelectField
                    control={settingsForm.control}
                    name="timeFormat"
                    label={t("Time Format")}
                    options={timeFormatChoices}
                    rules={{ required: "Time format is required" }}
                  />
                </FormControl>
                <FormControl>
                  <SelectField
                    control={settingsForm.control}
                    name="locale"
                    label={t("Language")}
                    description={t("Applies to the interface, and to the emails and documents sent to you.")}
                    options={localeChoices}
                    rules={{ required: "Language is required" }}
                  />
                </FormControl>
              </FormGroup>
            </Form>
          </div>

          <Separator />

          <div className="space-y-3">
            <SectionHeader
              icon={KeyRound}
              title={t("Change Password")}
              description={t("Leave blank to keep your current password.")}
            />
            <Form onSubmit={(e) => e.preventDefault()}>
              <FormGroup cols={1}>
                <FormControl>
                  <SensitiveField
                    control={passwordForm.control}
                    name="currentPassword"
                    label={t("Current Password")}
                    placeholder={t("Enter current password")}
                  />
                </FormControl>
              </FormGroup>
              <FormGroup cols={2} className="mt-2">
                <FormControl>
                  <SensitiveField
                    control={passwordForm.control}
                    name="newPassword"
                    label={t("New Password")}
                    placeholder={t("Enter new password")}
                  />
                </FormControl>
                <FormControl>
                  <SensitiveField
                    control={passwordForm.control}
                    name="confirmPassword"
                    label={t("Confirm Password")}
                    placeholder={t("Confirm new password")}
                  />
                </FormControl>
              </FormGroup>
            </Form>
          </div>
        </div>

        <Separator />

        <DialogFooter>
          <Button type="button" variant="outline" onClick={handleClose}>
            {t("Cancel")}
          </Button>
          <Button type="button" onClick={onSubmit} isLoading={isSubmitting} loadingText={t("Saving...")}>
            {t("Save Changes")}
          </Button>
        </DialogFooter>
      </DialogContent>

      <ImageCropUploadDialog
        open={isCropOpen}
        file={pendingFile}
        title={t("Crop Profile Picture")}
        description={t("Adjust your image before uploading. Profile pictures are cropped to a square.")}
        {...profilePictureCropConfig}
        confirmLabel={t("Upload Picture")}
        onClose={() => {
          setIsCropOpen(false);
          setPendingFile(null);
        }}
        onConfirm={handleProfilePictureUpload}
      />
    </Dialog>
  );
}
