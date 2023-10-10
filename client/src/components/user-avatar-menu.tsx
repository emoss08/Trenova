import { useLogout } from "@/hooks/useLogout";
import { User } from "@/types/accounts";
import { AvatarImage } from "@radix-ui/react-avatar";
import { DropdownMenuSeparator } from "@radix-ui/react-dropdown-menu";
import React from "react";
import { useTheme } from "./theme-provider";
import { Avatar, AvatarFallback } from "./ui/avatar";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuGroup,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuPortal,
  DropdownMenuShortcut,
  DropdownMenuSub,
  DropdownMenuSubContent,
  DropdownMenuSubTrigger,
  DropdownMenuTrigger,
} from "./ui/dropdown-menu";

type UserAvatarProps = React.ComponentPropsWithoutRef<typeof Avatar> & {
  user: User;
};

const UserAvatar = React.forwardRef<HTMLDivElement, UserAvatarProps>(
  ({ user, ...props }, ref) => (
    <Avatar ref={ref} {...props} className="focus-visible: outline-none">
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

export function UserAvatarMenu({ user }: { user: User }) {
  const logout = useLogout();
  const { setTheme } = useTheme();

  return (
    <div className="hidden md:flex flex-1 items-center justify-between space-x-2 md:justify-end">
      <DropdownMenu>
        <DropdownMenuTrigger>
          <UserAvatar user={user} />
        </DropdownMenuTrigger>
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
                <DropdownMenuItem onClick={() => setTheme("light")}>
                  Light
                </DropdownMenuItem>
                <DropdownMenuItem onClick={() => setTheme("dark")}>
                  Dark
                </DropdownMenuItem>
                <DropdownMenuSeparator />
                <DropdownMenuItem onClick={() => setTheme("system")}>
                  System
                </DropdownMenuItem>
              </DropdownMenuSubContent>
            </DropdownMenuPortal>
          </DropdownMenuSub>
          <DropdownMenuSeparator />
          <DropdownMenuItem>Support</DropdownMenuItem>
          <DropdownMenuSeparator />
          <DropdownMenuItem onClick={() => logout()}>
            Log out
            <DropdownMenuShortcut>⇧⌘Q</DropdownMenuShortcut>
          </DropdownMenuItem>
        </DropdownMenuContent>
      </DropdownMenu>
    </div>
  );
}
