import {
  DropdownMenu,
  DropdownMenuCheckboxItem,
  DropdownMenuContent,
  DropdownMenuGroup,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuPortal,
  DropdownMenuSeparator,
  DropdownMenuSub,
  DropdownMenuSubContent,
  DropdownMenuSubTrigger,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import {
  SidebarMenu,
  SidebarMenuButton,
  SidebarMenuItem,
  useSidebar,
} from "@/components/ui/sidebar";
import { useAuthStore } from "@/stores/auth-store";
import { useHotkey } from "@tanstack/react-hotkeys";
import { ChevronsUpDown, LogOut, Palette, Settings, User } from "lucide-react";
import { useState } from "react";
import { useNavigate } from "react-router";
import { useTheme } from "../theme-provider";
import { UserSettingsDialog } from "./user-settings-dialog";

export function NavUser() {
  const navigate = useNavigate();
  const { isMobile } = useSidebar();
  const { theme, setTheme } = useTheme();
  const { user, logout } = useAuthStore();
  const [settingsOpen, setSettingsOpen] = useState(false);

  useHotkey(
    "Mod+Shift+S",
    () => {
      setSettingsOpen((prev) => !prev);
    },
    {
      ignoreInputs: true,
      preventDefault: true,
    },
  );

  const initials = user?.name
    ? user.name
        .split(" ")
        .map((n) => n[0])
        .join("")
        .toUpperCase()
        .slice(0, 2)
    : "U";

  const handleLogout = async () => {
    await logout();
    void navigate("/login");
  };

  return (
    <>
      <UserSettingsDialog open={settingsOpen} onOpenChange={setSettingsOpen} />
      <SidebarMenu className="border-t border-border">
        <SidebarMenuItem>
          <DropdownMenu>
            <DropdownMenuTrigger
              render={
                <SidebarMenuButton
                  size="lg"
                  className="bg-sidebar data-[state=open]:bg-sidebar-accent data-[state=open]:text-sidebar-accent-foreground rounded-none h-14"
                />
              }
            >
              <div className="flex size-8 items-center justify-center rounded-lg bg-gradient-to-br from-sidebar-accent to-sidebar-accent/80 text-sidebar-accent-foreground">
                <span className="text-xs font-semibold">{initials}</span>
              </div>
              <div className="grid flex-1 text-left text-sm leading-tight">
                <span className="truncate font-semibold">{user?.name ?? "User"}</span>
                <span className="truncate text-xs text-muted-foreground">{user?.emailAddress}</span>
              </div>
              <ChevronsUpDown className="ml-auto size-4" />
            </DropdownMenuTrigger>
            <DropdownMenuContent
              className="w-(--anchor-width) min-w-56 rounded-lg"
              side={isMobile ? "bottom" : "right"}
              align="end"
              sideOffset={4}
            >
              <DropdownMenuGroup>
                <DropdownMenuLabel className="p-0 font-normal">
                  <div className="flex items-center gap-2 px-1 py-1.5 text-left text-sm">
                    <div className="flex size-8 items-center justify-center rounded-lg bg-gradient-to-br from-sidebar-accent to-sidebar-accent/80 text-sidebar-accent-foreground">
                      <span className="text-xs font-semibold">{initials}</span>
                    </div>
                    <div className="grid flex-1 text-left text-sm leading-tight">
                      <span className="truncate font-semibold">{user?.name}</span>
                      <span className="truncate text-xs text-muted-foreground">
                        {user?.emailAddress}
                      </span>
                    </div>
                  </div>
                </DropdownMenuLabel>
                <DropdownMenuSeparator />
                <DropdownMenuItem
                  title="Profile"
                  startContent={<User className="size-4" />}
                  onClick={() => void navigate("/profile")}
                />
                <DropdownMenuItem
                  title="Settings"
                  startContent={<Settings className="size-4" />}
                  onClick={() => setSettingsOpen(true)}
                />
                <DropdownMenuSub>
                  <DropdownMenuSubTrigger>
                    <Palette className="mr-2 size-4" />
                    <span>Switch Theme</span>
                  </DropdownMenuSubTrigger>
                  <DropdownMenuPortal>
                    <DropdownMenuSubContent sideOffset={5}>
                      <DropdownMenuCheckboxItem
                        checked={theme === "light"}
                        onCheckedChange={() => setTheme("light")}
                        className="cursor-pointer"
                      >
                        Light
                      </DropdownMenuCheckboxItem>
                      <DropdownMenuCheckboxItem
                        checked={theme === "dark"}
                        onCheckedChange={() => setTheme("dark")}
                        className="cursor-pointer"
                      >
                        Dark
                      </DropdownMenuCheckboxItem>
                      <DropdownMenuCheckboxItem
                        checked={theme === "system"}
                        onCheckedChange={() => setTheme("system")}
                        className="cursor-pointer"
                      >
                        System
                      </DropdownMenuCheckboxItem>
                    </DropdownMenuSubContent>
                  </DropdownMenuPortal>
                </DropdownMenuSub>
                <DropdownMenuSeparator />
                <DropdownMenuItem
                  title="Log out"
                  startContent={<LogOut className="size-4" />}
                  onClick={() => void handleLogout()}
                />
              </DropdownMenuGroup>
            </DropdownMenuContent>
          </DropdownMenu>
        </SidebarMenuItem>
      </SidebarMenu>
    </>
  );
}
