import { Avatar, AvatarFallback, AvatarImage } from "@/components/ui/avatar";

import {
  SidebarMenu,
  SidebarMenuButton,
  SidebarMenuItem,
} from "@/components/ui/sidebar";
import { useUser } from "@/stores/user-store";

import { CLIENT_VERSION } from "@/constants/env";
import { useLogout } from "@/hooks/use-auth";
import { User } from "@/types/user";
import {
  faBell,
  faDisplay,
  faGear,
  faMoon,
  faSignOut,
  faSunBright,
} from "@fortawesome/pro-regular-svg-icons";
import { CaretSortIcon } from "@radix-ui/react-icons";
import { useState } from "react";
import { toast } from "sonner";
import { Theme, useTheme } from "./theme-provider";
import {
  DropdownMenu,
  DropdownMenuCheckboxItem,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuPortal,
  DropdownMenuSeparator,
  DropdownMenuSub,
  DropdownMenuSubContent,
  DropdownMenuSubTrigger,
  DropdownMenuTrigger,
} from "./ui/dropdown-menu";
import { Icon } from "./ui/icons";
import { ExternalLink } from "./ui/link";

export function UserAvatar({ user }: { user: User | null }) {
  if (!user) {
    return null;
  }

  return (
    <div className="flex items-center gap-2 px-1 py-1.5 text-left text-sm">
      <Avatar className="size-8 rounded-lg">
        <AvatarImage src={user?.profilePicUrl} alt={user?.name} />
        <AvatarFallback>{user?.name?.charAt(0)}</AvatarFallback>
      </Avatar>
      <div className="grid w-full flex-1 text-left leading-tight">
        <span className="truncate text-sm font-semibold">{user?.name}</span>
        <span className="text-xs">{user?.emailAddress}</span>
      </div>
    </div>
  );
}

export function NavUser() {
  const { theme, setTheme } = useTheme();
  const [currentTheme, setCurrentTheme] = useState(theme);
  const logout = useLogout();
  const user = useUser();

  const handleLogout = async () => {
    toast.promise(logout, {
      loading: "Logging out...",
      success: "Logged out successfully",
      error: "Failed to log out",
    });
  };

  const switchTheme = (selectedTheme: Theme) => {
    // If the selected theme is the same as the current one, just return
    if (theme === selectedTheme) {
      return;
    }

    // Now, set the current theme to the selected theme
    setCurrentTheme(selectedTheme);

    // Then, make necessary changes like showing toast and so on
    setTheme(selectedTheme);
  };

  const themeIcon =
    currentTheme === "system" ? (
      <Icon icon={faDisplay} />
    ) : currentTheme === "light" ? (
      <Icon icon={faSunBright} />
    ) : (
      <Icon icon={faMoon} />
    );

  return (
    <SidebarMenu>
      <SidebarMenuItem>
        <DropdownMenu>
          <DropdownMenuTrigger asChild>
            <SidebarMenuButton
              className="border border-input bg-background [&>svg]:size-5"
              size="lg"
            >
              <UserAvatar user={user} />
              <CaretSortIcon className="ml-auto size-5" />
            </SidebarMenuButton>
          </DropdownMenuTrigger>
          <DropdownMenuContent
            side="right"
            align="end"
            className="w-[270px] max-w-[300px]"
          >
            <DropdownMenuLabel className="p-0 font-normal">
              <UserAvatar user={user} />
            </DropdownMenuLabel>
            <DropdownMenuSeparator />
            <DropdownMenuItem
              startContent={<Icon icon={faGear} />}
              title="Account Settings"
              description="Manage your account settings"
            />
            <DropdownMenuItem
              startContent={<Icon icon={faBell} />}
              title="Notifications"
              description="Manage your notifications"
            />
            <DropdownMenuSub>
              <DropdownMenuSubTrigger
                startContent={themeIcon}
                description="Switch application themes"
              >
                Switch Theme
              </DropdownMenuSubTrigger>
              <DropdownMenuPortal>
                <DropdownMenuSubContent sideOffset={10}>
                  <DropdownMenuCheckboxItem
                    checked={currentTheme === "light"}
                    onCheckedChange={() => switchTheme("light")}
                    className="cursor-pointer"
                  >
                    Light
                  </DropdownMenuCheckboxItem>
                  <DropdownMenuCheckboxItem
                    className="cursor-pointer"
                    checked={currentTheme === "dark"}
                    onCheckedChange={() => switchTheme("dark")}
                  >
                    Dark
                  </DropdownMenuCheckboxItem>
                  <DropdownMenuCheckboxItem
                    checked={currentTheme === "system"}
                    onCheckedChange={() => switchTheme("system")}
                    className="cursor-pointer"
                  >
                    System
                  </DropdownMenuCheckboxItem>
                </DropdownMenuSubContent>
              </DropdownMenuPortal>
            </DropdownMenuSub>
            <DropdownMenuItem
              startContent={<Icon icon={faSignOut} />}
              title="Sign out"
              description="Sign out of your account"
              className="pb-2"
              onClick={handleLogout}
            />
            <div className="flex flex-col w-full select-none items-center justify-center gap-1 text-2xs text-muted-foreground py-2 border-t border-input/50">
              <p>Client Build: v{CLIENT_VERSION}</p>
              <div className="flex items-center gap-2">
                <ExternalLink href="#">Terms & Conditions</ExternalLink>
                <div className="size-1 rounded-full bg-muted-foreground" />
                <ExternalLink href="https://github.com/emoss08/Trenova/blob/master/LICENSE">
                  License
                </ExternalLink>
              </div>
            </div>
          </DropdownMenuContent>
        </DropdownMenu>
      </SidebarMenuItem>
    </SidebarMenu>
  );
}
