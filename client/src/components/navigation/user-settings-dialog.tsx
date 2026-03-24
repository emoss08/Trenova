import { SelectField } from "@/components/fields/select-field";
import { SensitiveField } from "@/components/fields/sensitive-field";
import { Avatar, AvatarFallback } from "@/components/ui/avatar";
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
import { apiService } from "@/services/api";
import { useAuthStore } from "@/stores/auth-store";
import type { ChangeMyPassword, UpdateMySettings, User } from "@/types/user";
import { Globe, KeyRound, Mail } from "lucide-react";
import { useCallback } from "react";
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
  icon: React.ComponentType<{ className?: string }>;
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

  const initials = user?.name
    ? user.name
        .split(" ")
        .map((n) => n[0])
        .join("")
        .toUpperCase()
        .slice(0, 2)
    : "U";

  const settingsForm = useForm<UpdateMySettings>({
    defaultValues: {
      timezone: user?.timezone ?? "",
      timeFormat: user?.timeFormat ?? "12-hour",
      profilePicUrl: user?.profilePicUrl ?? undefined,
      thumbnailUrl: user?.thumbnailUrl ?? undefined,
    },
  });

  const passwordForm = useForm<ChangeMyPassword>({
    defaultValues: {
      currentPassword: "",
      newPassword: "",
      confirmPassword: "",
    },
  });

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
  }, [onOpenChange, passwordForm]);

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
          <Avatar size="lg">
            <AvatarFallback className="rounded-md bg-linear-to-br from-sidebar-accent to-sidebar-accent/80 text-sm font-semibold text-sidebar-accent-foreground">
              {initials}
            </AvatarFallback>
          </Avatar>
          <div className="min-w-0 flex-1">
            <p className="truncate text-sm font-semibold">{user?.name}</p>
            <p className="truncate text-xs text-muted-foreground">@{user?.username}</p>
            <div className="mt-1 flex items-center gap-1.5 text-xs text-muted-foreground">
              <Mail className="size-3 shrink-0" />
              <span className="truncate">{user?.emailAddress}</span>
            </div>
          </div>
        </div>

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
    </Dialog>
  );
}
