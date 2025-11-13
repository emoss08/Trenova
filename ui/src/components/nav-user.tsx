import { Avatar, AvatarFallback, AvatarImage } from "@/components/ui/avatar";
import {
  SidebarMenu,
  SidebarMenuButton,
  SidebarMenuItem,
} from "@/components/ui/sidebar";
import { useLogout } from "@/hooks/use-auth";
import type { UserSchema } from "@/lib/schemas/user-schema";
import { useUser } from "@/stores/user-store";
import { faUpRightFromSquare } from "@fortawesome/pro-regular-svg-icons";
import { CaretSortIcon } from "@radix-ui/react-icons";
import { useEffect, useState } from "react";
import { toast } from "sonner";
import { LicenseInformation } from "./license-information";
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
import { UserSettingsDialog } from "./user-settings-dialog";

export function UserAvatar({ user }: { user: UserSchema | null }) {
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

  const [licenseDialogOpen, setLicenseDialogOpen] = useState(false);
  const [settingsDialogOpen, setSettingsDialogOpen] = useState(false);

  // Keybind support for opening settings (Ctrl+Shift+S or Cmd+Shift+S)
  useEffect(() => {
    const handleKeyDown = (event: KeyboardEvent) => {
      if (
        event.key === "S" &&
        event.shiftKey &&
        (event.ctrlKey || event.metaKey)
      ) {
        event.preventDefault();
        setSettingsDialogOpen(true);
      }
    };

    window.addEventListener("keydown", handleKeyDown);
    return () => window.removeEventListener("keydown", handleKeyDown);
  }, []);

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

  return (
    <>
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
              side="bottom"
              align="end"
              className="w-[270px] max-w-[300px]"
            >
              <DropdownMenuLabel className="p-0 font-normal">
                <UserAvatar user={user} />
              </DropdownMenuLabel>
              <DropdownMenuSeparator />
              <DropdownMenuItem
                title="Account Settings"
                onClick={() => setSettingsDialogOpen(true)}
                className="cursor-pointer"
              />
              <DropdownMenuItem title="Notifications" />
              <DropdownMenuSub>
                <DropdownMenuSubTrigger>Switch Theme</DropdownMenuSubTrigger>
                <DropdownMenuPortal>
                  <DropdownMenuSubContent sideOffset={5}>
                    <DropdownMenuCheckboxItem
                      checked={currentTheme === "light"}
                      onCheckedChange={() => switchTheme("light")}
                      className="cursor-pointer"
                    >
                      Light
                    </DropdownMenuCheckboxItem>
                    <DropdownMenuCheckboxItem
                      checked={currentTheme === "dark"}
                      onCheckedChange={() => switchTheme("dark")}
                      className="cursor-pointer"
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
              <DropdownMenuSub>
                <DropdownMenuSubTrigger>Learn More</DropdownMenuSubTrigger>
                <DropdownMenuPortal>
                  <DropdownMenuSubContent alignOffset={-30} sideOffset={5}>
                    <DropdownMenuItem
                      endContent={<Icon icon={faUpRightFromSquare} />}
                      title="About Trenova"
                      className="cursor-pointer"
                    />
                    <DropdownMenuItem
                      endContent={<Icon icon={faUpRightFromSquare} />}
                      title="User Guide & Documentation"
                      className="cursor-pointer"
                    />
                    <DropdownMenuSeparator />
                    <DropdownMenuItem
                      onClick={() => setLicenseDialogOpen(true)}
                      title="License Agreement"
                      className="cursor-pointer"
                    />
                    <DropdownMenuItem
                      endContent={<Icon icon={faUpRightFromSquare} />}
                      title="Terms & Conditions"
                      className="cursor-pointer"
                    />
                  </DropdownMenuSubContent>
                </DropdownMenuPortal>
              </DropdownMenuSub>
              <DropdownMenuSeparator />
              <DropdownMenuItem title="Log out" onClick={handleLogout} />
            </DropdownMenuContent>
          </DropdownMenu>
        </SidebarMenuItem>
      </SidebarMenu>
      {licenseDialogOpen && (
        <LicenseInformation
          open={licenseDialogOpen}
          onOpenChange={setLicenseDialogOpen}
        />
      )}
      {user && settingsDialogOpen && (
        <UserSettingsDialog
          open={settingsDialogOpen}
          onOpenChange={setSettingsDialogOpen}
          user={user}
        />
      )}
    </>
  );
}
