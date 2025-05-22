import { Avatar, AvatarFallback, AvatarImage } from "@/components/ui/avatar";
import {
  SidebarMenu,
  SidebarMenuButton,
  SidebarMenuItem,
} from "@/components/ui/sidebar";
import { useLogout } from "@/hooks/use-auth";
import { useUser } from "@/stores/user-store";
import { User } from "@/types/user";
import { faUpRightFromSquare } from "@fortawesome/pro-regular-svg-icons";
import { CaretSortIcon } from "@radix-ui/react-icons";
import { useState } from "react";
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

  const [licenseDialogOpen, setLicenseDialogOpen] = useState(false);

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
              side="top"
              align="start"
              className="w-[270px] max-w-[300px]"
            >
              <DropdownMenuLabel className="p-0 font-normal">
                <UserAvatar user={user} />
              </DropdownMenuLabel>
              <DropdownMenuSeparator />
              <DropdownMenuItem title="Account Settings" />
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
              <DropdownMenuSub>
                <DropdownMenuSubTrigger>Learn More</DropdownMenuSubTrigger>
                <DropdownMenuPortal>
                  <DropdownMenuSubContent sideOffset={5}>
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
    </>
  );
}
