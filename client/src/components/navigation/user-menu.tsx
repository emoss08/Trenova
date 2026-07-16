import { UserSettingsDialog } from "@/components/navigation/user-settings-dialog";
import { ResolvedUserAvatar } from "@/components/resolved-user-avatar";
import { useTheme } from "@/components/theme-provider";
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
import { useAuthStore } from "@/stores/auth-store";
import { ChevronsUpDown, LogOut, Palette, Settings, User } from "lucide-react";
import { useState } from "react";
import { useNavigate } from "react-router";

export function UserMenu() {
  const user = useAuthStore((s) => s.user);
  const logout = useAuthStore((s) => s.logout);
  const navigate = useNavigate();
  const { theme, setTheme } = useTheme();
  const [settingsOpen, setSettingsOpen] = useState(false);

  const displayName = user?.name ?? user?.username ?? "User";

  const handleLogout = async () => {
    await logout();
    void navigate("/login");
  };

  return (
    <>
      <UserSettingsDialog open={settingsOpen} onOpenChange={setSettingsOpen} />
      <DropdownMenu>
        <DropdownMenuTrigger
          render={
            <button
              type="button"
              className="flex w-full items-center gap-2 rounded-md p-1.5 transition-colors hover:bg-accent/50"
            />
          }
        >
          <ResolvedUserAvatar
            userId={user?.id}
            name={user?.name}
            profilePicUrl={user?.profilePicUrl}
            thumbnailUrl={user?.thumbnailUrl}
            className="size-7"
            fallbackClassName="bg-muted text-[10px] font-medium text-muted-foreground"
          />
          <span className="grid min-w-0 flex-1 text-left leading-tight">
            <span className="truncate text-sm font-medium">{displayName}</span>
            <span className="truncate text-2xs text-muted-foreground">{user?.emailAddress}</span>
          </span>
          <ChevronsUpDown className="size-3.5 shrink-0 text-muted-foreground" />
        </DropdownMenuTrigger>
        <DropdownMenuContent side="right" align="end" sideOffset={8} className="min-w-56 rounded-lg">
          <DropdownMenuGroup>
            <DropdownMenuLabel className="p-0 font-normal">
              <div className="flex items-center gap-2 px-1 py-1.5 text-left text-sm">
                <ResolvedUserAvatar
                  userId={user?.id}
                  name={user?.name}
                  profilePicUrl={user?.profilePicUrl}
                  thumbnailUrl={user?.thumbnailUrl}
                  className="size-8"
                  fallbackClassName="rounded-lg bg-gradient-to-br from-sidebar-accent to-sidebar-accent/80 text-xs font-semibold text-sidebar-accent-foreground"
                />
                <div className="grid flex-1 text-left text-sm leading-tight">
                  <span className="truncate font-semibold text-foreground">{user?.name}</span>
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
    </>
  );
}
