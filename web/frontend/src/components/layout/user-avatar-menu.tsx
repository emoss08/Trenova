/**
 * COPYRIGHT(c) 2024 Trenova
 *
 * This file is part of Trenova.
 *
 * The Trenova software is licensed under the Business Source License 1.1. You are granted the right
 * to copy, modify, and redistribute the software, but only for non-production use or with a total
 * of less than three server instances. Starting from the Change Date (November 16, 2026), the
 * software will be made available under version 2 or later of the GNU General Public License.
 * If you use the software in violation of this license, your rights under the license will be
 * terminated automatically. The software is provided "as is," and the Licensor disclaims all
 * warranties and conditions. If you use this license's text or the "Business Source License" name
 * and trademark, you must comply with the Licensor's covenants, which include specifying the
 * Change License as the GPL Version 2.0 or a compatible license, specifying an Additional Use
 * Grant, and not modifying the license in any other way.
 */

import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from "@/components/ui/alert-dialog";
import { Avatar, AvatarFallback } from "@/components/ui/avatar";
import {
  DropdownMenu,
  DropdownMenuCheckboxItem,
  DropdownMenuContent,
  DropdownMenuGroup,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuPortal,
  DropdownMenuSeparator,
  DropdownMenuShortcut,
  DropdownMenuSub,
  DropdownMenuSubContent,
  DropdownMenuSubTrigger,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { KeyCombo, Keys, ShortcutsProvider } from "@/components/ui/keyboard";
import { useTheme } from "@/components/ui/theme-provider";
import { useLogout } from "@/hooks/useLogout";
import { useNotifications } from "@/hooks/useQueries";
import { useHeaderStore } from "@/stores/HeaderStore";
import { type ThemeOptions } from "@/types";
import { User } from "@/types/accounts";
import { AvatarImage } from "@radix-ui/react-avatar";
import React, { useState } from "react";
import { useNavigate } from "react-router-dom";
import { UserSettingsDialog } from "../user-settings-dialog";
import { truncateText } from "@/lib/utils";

type UserAvatarProps = React.ComponentPropsWithoutRef<typeof Avatar> & {
  user: User;
};

export function SignOutDialog({
  open,
  onClose,
}: {
  open: boolean;
  onClose: () => void;
}) {
  const logout = useLogout();

  return (
    <AlertDialog open={open} onOpenChange={onClose}>
      <AlertDialogContent>
        <AlertDialogHeader>
          <AlertDialogTitle>Sign out of Trenova</AlertDialogTitle>
          <AlertDialogDescription>
            Are you sure you want to sign out?
          </AlertDialogDescription>
        </AlertDialogHeader>
        <AlertDialogFooter>
          <AlertDialogCancel>Cancel</AlertDialogCancel>
          <AlertDialogAction onClick={() => logout()}>
            Sign Out
          </AlertDialogAction>
        </AlertDialogFooter>
      </AlertDialogContent>
    </AlertDialog>
  );
}

export const UserAvatar = React.forwardRef<HTMLDivElement, UserAvatarProps>(
  ({ user, ...props }, ref) => {
    // Determine the initials for the fallback avatar
    const initials = user.name ? user.name[0].toUpperCase() : user.username[0];

    // Determine the avatar image source
    const avatarSrc =
      user.profilePicUrl || `https://avatar.vercel.sh/${user.email}`;

    return (
      <div className="flex select-none items-center" ref={ref} {...props}>
        <Avatar className="m-auto inline-block">
          <AvatarImage
            src={avatarSrc}
            alt={user.username}
            className="size-9 rounded-full"
          />
          <AvatarFallback delayMs={600}>{initials}</AvatarFallback>
        </Avatar>
      </div>
    );
  },
);

export function UserAvatarMenuContent({
  user,
  hasNotifications,
}: {
  user: User;
  hasNotifications: boolean;
}) {
  const { theme, setTheme } = useTheme();
  const [currentTheme, setCurrentTheme] = useState(theme);
  const [signOutDialogOpen, setSignOutDialogOpen] =
    useHeaderStore.use("signOutMenuOpen");
  const [settingsOpen, setSettingsOpen] = useState<boolean>(false);

  const navigate = useNavigate();

  React.useEffect(() => {
    const down = (e: KeyboardEvent) => {
      if (e.key === "q" && (e.metaKey || e.ctrlKey)) {
        e.preventDefault();
        setSignOutDialogOpen(true);
      }

      if (e.key === "b" && (e.metaKey || e.ctrlKey)) {
        e.preventDefault();
        setSettingsOpen(true);
      }
    };
    document.addEventListener("keydown", down);
    return () => document.removeEventListener("keydown", down);
  }, [setSignOutDialogOpen, navigate]);

  const switchTheme = (selectedTheme: ThemeOptions) => {
    // If the selected theme is the same as the current one, just return
    if (currentTheme === selectedTheme) {
      return;
    }

    // Now, set the current theme to the selected theme
    setCurrentTheme(selectedTheme);

    // Then, make necessary changes like showing toast and so on
    setTheme(selectedTheme);
  };

  return (
    <>
      <DropdownMenuContent className="w-56 bg-card">
        <DropdownMenuLabel className="font-normal">
          <div className="flex flex-col space-y-1">
            <div className="flex items-center space-x-2">
              <UserAvatar user={user} />
              <div>
                <p className="truncate text-sm font-medium leading-none">
                  {truncateText(user.name || user.username, 20)}
                </p>
                <p className="text-xs leading-none text-muted-foreground">
                  {user.email}
                </p>
              </div>
            </div>
          </div>
        </DropdownMenuLabel>
        <DropdownMenuSeparator />
        <DropdownMenuGroup>
          <DropdownMenuItem onClick={() => setSettingsOpen(true)}>
            Account Settings
            <DropdownMenuShortcut>
              <ShortcutsProvider os="mac">
                <KeyCombo keyNames={[Keys.Command, "B"]} />
              </ShortcutsProvider>
            </DropdownMenuShortcut>
          </DropdownMenuItem>
          <DropdownMenuItem onClick={() => {}}>Tasks</DropdownMenuItem>
          <DropdownMenuItem onClick={() => {}}>
            Notifications
            <span className="ml-auto flex gap-x-0.5 tracking-widest">
              {hasNotifications && (
                <span className="size-1.5 rounded-full bg-green-600 ring-background motion-safe:animate-pulse"></span>
              )}
            </span>
          </DropdownMenuItem>
        </DropdownMenuGroup>
        <DropdownMenuSub>
          <DropdownMenuSubTrigger>Switch Theme</DropdownMenuSubTrigger>
          <DropdownMenuPortal>
            <DropdownMenuSubContent sideOffset={10}>
              <DropdownMenuCheckboxItem
                checked={currentTheme === "light"}
                onCheckedChange={() => switchTheme("light")}
              >
                Light
              </DropdownMenuCheckboxItem>
              <DropdownMenuCheckboxItem
                checked={currentTheme === "dark"}
                onCheckedChange={() => switchTheme("dark")}
              >
                Dark
              </DropdownMenuCheckboxItem>
              <DropdownMenuCheckboxItem
                checked={currentTheme === "system"}
                onCheckedChange={() => switchTheme("system")}
              >
                System
              </DropdownMenuCheckboxItem>
            </DropdownMenuSubContent>
          </DropdownMenuPortal>
        </DropdownMenuSub>
        <DropdownMenuSeparator />
        <DropdownMenuItem onClick={() => setSignOutDialogOpen(true)}>
          Log out
          <DropdownMenuShortcut>
            <ShortcutsProvider os="mac">
              <KeyCombo keyNames={[Keys.Command, "Q"]} />
            </ShortcutsProvider>
          </DropdownMenuShortcut>
        </DropdownMenuItem>
      </DropdownMenuContent>
      {signOutDialogOpen && (
        <SignOutDialog
          open={signOutDialogOpen}
          onClose={() => setSignOutDialogOpen(false)}
        />
      )}
      {settingsOpen && (
        <UserSettingsDialog
          open={settingsOpen}
          onOpenChange={() => setSettingsOpen(false)}
          user={user}
        />
      )}
    </>
  );
}

export function UserAvatarMenu({ user }: { user: User }) {
  const { notificationsData } = useNotifications(user.id);
  const userHasNotifications =
    (notificationsData && notificationsData?.unreadCount > 0) || false;

  return (
    <DropdownMenu>
      <DropdownMenuTrigger asChild>
        <span className="relative inline-block">
          <UserAvatar user={user} />
          <span className="absolute bottom-1 right-1 block size-2 rounded-full bg-green-600 ring-2 ring-background" />
        </span>
      </DropdownMenuTrigger>
      <UserAvatarMenuContent
        user={user}
        hasNotifications={userHasNotifications}
      />
    </DropdownMenu>
  );
}
