/*
 * COPYRIGHT(c) 2023 MONTA
 *
 * This file is part of Monta.
 *
 * The Monta software is licensed under the Business Source License 1.1. You are granted the right
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

import { useLogout } from "@/hooks/useLogout";
import { ThemeOptions } from "@/types";
import { User } from "@/types/accounts";
import { AvatarImage } from "@radix-ui/react-avatar";
import React, { useState } from "react";
import { useTheme } from "./theme-provider";
import { Avatar, AvatarFallback } from "./ui/avatar";
import {
  DropdownMenu,
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
} from "./ui/dropdown-menu";
import { ToastAction } from "./ui/toast";
import { useToast } from "./ui/use-toast";

type UserAvatarProps = React.ComponentPropsWithoutRef<typeof Avatar> & {
  user: User;
};

const UserAvatar = React.forwardRef<HTMLDivElement, UserAvatarProps>(
  ({ user, ...props }, ref) => (
    <Avatar ref={ref} {...props}>
      <AvatarImage
        className="w-full h-full rounded-full"
        src={user.profile?.profilePicture}
        alt={user.username}
      />
      <AvatarFallback>
        {user.profile?.firstName.charAt(0)}
        {user.profile?.lastName.charAt(0)}
      </AvatarFallback>
    </Avatar>
  ),
);

UserAvatar.displayName = "UserAvatar";

function UserAvatarMenuContent() {
  const logout = useLogout();
  const { theme, setTheme, isRainbowAnimationActive, toggleRainbowAnimation } =
    useTheme();
  const { toast } = useToast();
  // 1. Adjust the state to store both previous and current theme
  // Separate state variables for current and previous themes
  const [currentTheme, setCurrentTheme] = useState(theme);
  const [previousTheme, setPreviousTheme] = useState(theme);

  const switchTheme = (selectedTheme: ThemeOptions) => {
    // If the selected theme is the same as the current one, just return
    if (currentTheme === selectedTheme) {
      return;
    }

    // First, set the previous theme to the current theme
    setPreviousTheme(currentTheme);

    // Now, set the current theme to the selected theme
    setCurrentTheme(selectedTheme);

    // Then, make necessary changes like showing toast and so on
    setTheme(selectedTheme);
    const today = new Date();

    const formattedDate = `${today.toLocaleString("en-US", {
      weekday: "long",
    })}, ${today.toLocaleString("en-US", {
      month: "long",
    })} ${today.getDate()}, ${today.getFullYear()} at ${today.toLocaleString(
      "en-US",
      { hour: "numeric", minute: "2-digit", hour12: true },
    )}`;

    toast({
      title: `Theme changed to ${selectedTheme}`,
      description: formattedDate,
      action: (
        <ToastAction altText="Goto schedule to undo" onClick={undoThemeChange}>
          Undo
        </ToastAction>
      ),
    });
  };

  const undoThemeChange = () => {
    console.log("Previous theme", previousTheme);

    // Set the current theme back to the previous theme
    setCurrentTheme(previousTheme);

    // Update the actual theme
    setTheme(previousTheme);

    toast({
      title: `Theme reverted to ${previousTheme}`,
      description: "Your theme change was undone.",
    });
  };
  return (
    <DropdownMenuContent className="w-56">
      <DropdownMenuLabel>My Account</DropdownMenuLabel>
      <DropdownMenuSeparator />
      <DropdownMenuGroup>
        <DropdownMenuItem>
          Profile
          <DropdownMenuShortcut>⇧⌘P</DropdownMenuShortcut>
        </DropdownMenuItem>
        <DropdownMenuItem>
          Preferences
          <DropdownMenuShortcut>⌘B</DropdownMenuShortcut>
        </DropdownMenuItem>
      </DropdownMenuGroup>
      <DropdownMenuSeparator />
      <DropdownMenuSub>
        <DropdownMenuSubTrigger>Switch Theme</DropdownMenuSubTrigger>
        <DropdownMenuPortal>
          <DropdownMenuSubContent>
            <DropdownMenuItem onClick={() => switchTheme("light")}>
              Light
            </DropdownMenuItem>
            <DropdownMenuItem onClick={() => switchTheme("dark")}>
              Dark
            </DropdownMenuItem>
            <DropdownMenuSeparator />
            <DropdownMenuItem onClick={() => switchTheme("system")}>
              System
            </DropdownMenuItem>
          </DropdownMenuSubContent>
        </DropdownMenuPortal>
      </DropdownMenuSub>
      <DropdownMenuItem onClick={toggleRainbowAnimation}>
        {isRainbowAnimationActive ? "Turn off" : "Turn on"} rainbow
      </DropdownMenuItem>
      <DropdownMenuItem>Support</DropdownMenuItem>
      <DropdownMenuSeparator />
      <DropdownMenuItem onClick={() => logout()}>
        Log out
        <DropdownMenuShortcut>⇧⌘Q</DropdownMenuShortcut>
      </DropdownMenuItem>
    </DropdownMenuContent>
  );
}

export function UserAvatarMenu({ user }: { user: User }) {
  return (
    <div className="hidden md:flex flex-1 items-center justify-between space-x-2 md:justify-end">
      <DropdownMenu>
        <DropdownMenuTrigger>
          <UserAvatar user={user} />
        </DropdownMenuTrigger>
        <UserAvatarMenuContent />
      </DropdownMenu>
    </div>
  );
}
