/*
 * Copyright 2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

import ChangePasswordForm from "@/app/auth/_components/change-password-form";
import {
  Dialog,
  DialogBody,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { VisuallyHidden } from "@/components/ui/visually-hidden";
import type { UserSchema } from "@/lib/schemas/user-schema";
import { faLock, faUser } from "@fortawesome/pro-regular-svg-icons";
import { Icon } from "./ui/icons";
import { UserProfileForm } from "./user-profile-form";

interface UserSettingsDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  user: UserSchema;
}

export function UserSettingsDialog({
  open,
  onOpenChange,
  user,
}: UserSettingsDialogProps) {
  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-2xl">
        <DialogHeader>
          <DialogTitle>User Settings</DialogTitle>
          <DialogDescription>
            Manage your profile information and security settings
          </DialogDescription>
        </DialogHeader>
        <DialogBody>
          <Tabs defaultValue="profile" className="w-full">
            <TabsList className="grid w-full grid-cols-2">
              <TabsTrigger value="profile">
                <Icon icon={faUser} />
                Profile
              </TabsTrigger>
              <TabsTrigger value="password">
                <Icon icon={faLock} />
                Password
              </TabsTrigger>
            </TabsList>
            <TabsContent value="profile" className="mt-4">
              <UserProfileForm
                user={user}
                onSuccess={() => onOpenChange(false)}
              />
            </TabsContent>
            <TabsContent value="password" className="mt-4">
              <PasswordTabContent onOpenChange={onOpenChange} />
            </TabsContent>
          </Tabs>
        </DialogBody>
      </DialogContent>
    </Dialog>
  );
}

function PasswordTabContent({
  onOpenChange,
}: {
  onOpenChange: (open: boolean) => void;
}) {
  return (
    <div className="space-y-4">
      <div className="text-sm text-muted-foreground">
        <p>Change your password to keep your account secure.</p>
        <p className="mt-2">
          Your password must be at least 8 characters long and contain:
        </p>
        <ul className="mt-2 list-inside list-disc space-y-1 text-xs">
          <li>At least one uppercase letter</li>
          <li>At least one lowercase letter</li>
          <li>At least one number</li>
          <li>At least one special character</li>
        </ul>
      </div>
      <ChangePasswordForm onOpenChange={onOpenChange} />
    </div>
  );
}
