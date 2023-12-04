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

import { Avatar, AvatarFallback } from "@/components/ui/avatar";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuGroup,
  DropdownMenuItem,
  DropdownMenuPortal,
  DropdownMenuSeparator,
  DropdownMenuShortcut,
  DropdownMenuSub,
  DropdownMenuSubContent,
  DropdownMenuSubTrigger,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { useTheme } from "@/components/ui/theme-provider";
import { ToastAction } from "@/components/ui/toast";
import { useToast } from "@/components/ui/use-toast";
import { useLogout } from "@/hooks/useLogout";
import { ThemeOptions } from "@/types";
import { User } from "@/types/accounts";
import { AvatarImage } from "@radix-ui/react-avatar";
import { ChevronDownIcon } from "@radix-ui/react-icons";
import React, { useState } from "react";
import { useNavigate } from "react-router-dom";

type UserAvatarProps = React.ComponentPropsWithoutRef<typeof Avatar> & {
  user: User;
};

const UserAvatar = React.forwardRef<HTMLDivElement, UserAvatarProps>(
  ({ user, ...props }, ref) => {
    // Determine the initials for the fallback avatar
    const initials = user.profile
      ? user.profile.firstName.charAt(0) + user.profile.lastName.charAt(0)
      : "";

    // Determine the avatar image source
    const avatarSrc = user.profile?.thumbnail
      ? user.profile.thumbnail
      : `https://avatar.vercel.sh/${user.email}`;

    return (
      <div
        className="group flex items-center hover:cursor-pointer"
        ref={ref}
        {...props}
      >
        <div className="border-l border-accent h-7 mr-2 pl-2" />
        <Avatar className="inline-block ml-4 m-auto">
          <AvatarImage
            src={avatarSrc}
            alt={user.username}
            className="h-9 w-9 rounded-full"
          />
          <AvatarFallback delayMs={600}>{initials}</AvatarFallback>
        </Avatar>
        <div className="flex items-center ml-2 mb-1">
          <ChevronDownIcon
            className="h-4 w-4 transition duration-200 group-data-[state=open]:rotate-180"
            aria-hidden="true"
          />
        </div>
      </div>
    );
  },
);

function UserAvatarMenuContent({ user }: { user: User }) {
  const logout = useLogout();
  const { theme, setTheme, isRainbowAnimationActive, toggleRainbowAnimation } =
    useTheme();
  const { toast } = useToast();
  const [currentTheme, setCurrentTheme] = useState(theme);
  const [previousTheme, setPreviousTheme] = useState(theme);
  const navigate = useNavigate();

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
    console.info("Previous theme", previousTheme);

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
      <div className="px-2">
        <p className="text-sm">Signed in as</p>
        <p className="truncate text-sm font-medium text-accent-foreground">
          {user.email}
        </p>
      </div>
      <DropdownMenuSeparator />
      <DropdownMenuGroup>
        <DropdownMenuItem>Changelog</DropdownMenuItem>
        <DropdownMenuItem>Support</DropdownMenuItem>
        <DropdownMenuItem>License</DropdownMenuItem>
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
            <DropdownMenuItem onClick={() => switchTheme("slate-dark")}>
              Slate Dark
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
      <DropdownMenuSeparator />
      <DropdownMenuItem onClick={() => navigate("/account/settings/")}>
        Account Settings
        <DropdownMenuShortcut>⇧⌘P</DropdownMenuShortcut>
      </DropdownMenuItem>
      <DropdownMenuItem onClick={() => logout()}>
        Log out
        <DropdownMenuShortcut>⇧⌘Q</DropdownMenuShortcut>
      </DropdownMenuItem>
    </DropdownMenuContent>
  );
}

export function UserAvatarMenu({ user }: { user: User }) {
  return (
    <div className="hidden md:flex flex-1 items-center justify-between space-x-2 md:justify-end focus-visible:outline-none">
      <DropdownMenu>
        <DropdownMenuTrigger asChild>
          <UserAvatar user={user} />
        </DropdownMenuTrigger>
        <UserAvatarMenuContent user={user} />
      </DropdownMenu>
    </div>
  );
}
