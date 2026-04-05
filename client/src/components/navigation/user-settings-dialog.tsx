import { ImageCropUploadDialog } from "@/components/image-crop-upload-dialog";
import { ResolvedUserAvatar } from "@/components/resolved-user-avatar";
import { SelectField } from "@/components/fields/select-field";
import { SensitiveField } from "@/components/fields/sensitive-field";
import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Form, FormControl, FormGroup } from "@/components/ui/form";
import { Separator } from "@/components/ui/separator";
import { useApiMutation } from "@/hooks/use-api-mutation";
import { timeFormatChoices, timezoneChoices } from "@/lib/choices";
import { validateCroppableImage } from "@/lib/images/crop-image";
import {
  IMAGE_UPLOAD_ACCEPT,
  profilePictureCropConfig,
} from "@/lib/images/upload-config";
import { queries } from "@/lib/queries";
import { apiService } from "@/services/api";
import { useAuthStore } from "@/stores/auth-store";
import type { ChangeMyPassword, UpdateMySettings, User } from "@/types/user";
import { useQueryClient } from "@tanstack/react-query";
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
      <div className="flex size-8 shrink-0 items-center justify-center rounded-lg bg-muted">
        <Icon className="size-4 text-muted-foreground" />
      </div>
      <div>
        <h3 className="text-sm leading-none font-medium">{title}</h3>
        <p className="mt-1 text-xs text-muted-foreground">{description}</p>
      </div>
    </div>
  );
}

export function UserSettingsDialog({ open, onOpenChange }: UserSettingsDialogProps) {
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
    setFormError: settingsForm.setError,
    onSuccess: (updatedUser) => {
      useAuthStore.getState().setUser(updatedUser);
      toast.success("Settings updated", {
        description: "Your profile settings have been saved.",
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
    setFormError: passwordForm.setError,
    onSuccess: () => {
      toast.success("Password changed", {
        description: "Your password has been updated successfully.",
      });
    },
  });

  const handleClose = useCallback(() => {
    onOpenChange(false);
    passwordForm.reset();
    setPendingFile(null);
    setIsCropOpen(false);
  }, [onOpenChange, passwordForm]);

  const syncUserSettings = useCallback(async (updatedUser: User) => {
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
  }, [queryClient]);

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
      toast.error("Unsupported profile picture", {
        description: error instanceof Error ? error.message : "Please choose a JPG, PNG, or WEBP file.",
      });
    }
  }, []);

  const handleProfilePictureUpload = useCallback(async (file: File) => {
    const updatedUser = await apiService.userService.uploadMyProfilePicture(file);
    await syncUserSettings(updatedUser);
    toast.success("Profile picture updated");
  }, [syncUserSettings]);

  const handleRemoveProfilePicture = useCallback(async () => {
    if (isRemovingProfilePicture) {
      return;
    }

    setIsRemovingProfilePicture(true);
    try {
      const updatedUser = await apiService.userService.deleteMyProfilePicture();
      await syncUserSettings(updatedUser);
      toast.success("Profile picture removed");
    } catch (error) {
      toast.error("Failed to remove profile picture", {
        description: error instanceof Error ? error.message : "Please try again.",
      });
    }

    setIsRemovingProfilePicture(false);
  }, [isRemovingProfilePicture, syncUserSettings]);

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
          <DialogTitle>Settings</DialogTitle>
          <DialogDescription>Manage your preferences and security.</DialogDescription>
        </DialogHeader>

        <div className="flex items-center gap-4 rounded-md border bg-sidebar p-4">
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
            <p className="truncate text-xs text-muted-foreground">@{user?.username}</p>
            <div className="mt-1 flex items-center gap-1.5 text-xs text-muted-foreground">
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
                aria-label="Remove profile picture"
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
              title="Preferences"
              description="Configure your regional and display settings."
            />
            <Form onSubmit={(e) => e.preventDefault()}>
              <FormGroup cols={2}>
                <FormControl>
                  <SelectField
                    control={settingsForm.control}
                    name="timezone"
                    label="Timezone"
                    options={timezoneChoices}
                    rules={{ required: "Timezone is required" }}
                  />
                </FormControl>
                <FormControl>
                  <SelectField
                    control={settingsForm.control}
                    name="timeFormat"
                    label="Time Format"
                    options={timeFormatChoices}
                    rules={{ required: "Time format is required" }}
                  />
                </FormControl>
              </FormGroup>
            </Form>
          </div>

          <Separator />

          <div className="space-y-3">
            <SectionHeader
              icon={KeyRound}
              title="Change Password"
              description="Leave blank to keep your current password."
            />
            <Form onSubmit={(e) => e.preventDefault()}>
              <FormGroup cols={1}>
                <FormControl>
                  <SensitiveField
                    control={passwordForm.control}
                    name="currentPassword"
                    label="Current Password"
                    placeholder="Enter current password"
                  />
                </FormControl>
              </FormGroup>
              <FormGroup cols={2} className="mt-2">
                <FormControl>
                  <SensitiveField
                    control={passwordForm.control}
                    name="newPassword"
                    label="New Password"
                    placeholder="Enter new password"
                  />
                </FormControl>
                <FormControl>
                  <SensitiveField
                    control={passwordForm.control}
                    name="confirmPassword"
                    label="Confirm Password"
                    placeholder="Confirm new password"
                  />
                </FormControl>
              </FormGroup>
            </Form>
          </div>
        </div>

        <Separator />

        <DialogFooter>
          <Button type="button" variant="outline" onClick={handleClose}>
            Cancel
          </Button>
          <Button type="button" onClick={onSubmit} isLoading={isSubmitting} loadingText="Saving...">
            Save Changes
          </Button>
        </DialogFooter>
      </DialogContent>

      <ImageCropUploadDialog
        open={isCropOpen}
        file={pendingFile}
        title="Crop Profile Picture"
        description="Adjust your image before uploading. Profile pictures are cropped to a square."
        {...profilePictureCropConfig}
        confirmLabel="Upload Picture"
        onClose={() => {
          setIsCropOpen(false);
          setPendingFile(null);
        }}
        onConfirm={handleProfilePictureUpload}
      />
    </Dialog>
  );
}
