import { useT } from "@trenova/shared/i18n/use-t";
import { SidebarLayoutSubmenu } from "@/components/navigation/sidebar-variant-menu";
import { ResolvedUserAvatar } from "@/components/resolved-user-avatar";
import { useTheme } from "@trenova/shared/components/theme-provider";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from "@trenova/shared/components/ui/dialog";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
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
} from "@trenova/shared/components/ui/dropdown-menu";
import { useAuthStore } from "@trenova/shared/stores/auth-store";
import { ChevronsUpDown, LogOut, Palette, Settings, User } from "lucide-react";
import { lazy, Suspense, useState } from "react";
import { useNavigate } from "react-router";

// The settings dialog carries the timezone/time-format choice tables, the
// avatar cropper and the react-hook-form field set — none of which belong in
// the chunk that renders the sidebar on every page.
const UserSettingsDialog = lazy(() =>
  import("@/components/navigation/user-settings-dialog").then((module) => ({
    default: module.UserSettingsDialog,
  })),
);

function UserSettingsDialogSkeleton({
  open,
  onOpenChange,
}: {
  open: boolean;
  onOpenChange: (open: boolean) => void;
}) {
  const t = useT();

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-[520px]">
        <DialogHeader>
          <DialogTitle>{t("Settings")}</DialogTitle>
          <DialogDescription>{t("Manage your preferences and security.")}</DialogDescription>
        </DialogHeader>
        <div className="bg-sidebar flex items-center gap-4 rounded-md border p-4">
          <Skeleton className="size-14 shrink-0 rounded-md" />
          <div className="flex min-w-0 flex-1 flex-col gap-1.5">
            <Skeleton className="h-3.5 w-40 rounded" />
            <Skeleton className="h-3 w-28 rounded" />
            <Skeleton className="h-3 w-48 rounded" />
          </div>
        </div>
        <div className="space-y-5">
          {Array.from({ length: 3 }, (_, section) => (
            <div key={section} className="space-y-3">
              <div className="flex items-center gap-3">
                <Skeleton className="size-8 shrink-0 rounded-lg" />
                <div className="flex flex-col gap-1.5">
                  <Skeleton className="h-3.5 w-32 rounded" />
                  <Skeleton className="h-3 w-56 rounded" />
                </div>
              </div>
              <Skeleton className="h-9 w-full rounded-md" />
            </div>
          ))}
        </div>
      </DialogContent>
    </Dialog>
  );
}

/**
 * `compact` renders the avatar alone as the trigger, for a rail with no room
 * for a name; the full menu is unchanged.
 */
export function UserMenu({ compact = false }: { compact?: boolean }) {
  const t = useT();

  const user = useAuthStore((s) => s.user);
  const logout = useAuthStore((s) => s.logout);
  const navigate = useNavigate();
  const { theme, setTheme } = useTheme();
  const [settingsOpen, setSettingsOpen] = useState(false);
  const [settingsMounted, setSettingsMounted] = useState(false);

  const displayName = user?.name ?? user?.username ?? "User";

  const handleLogout = async () => {
    await logout();
    void navigate("/login");
  };

  return (
    <>
      {settingsMounted && (
        <Suspense
          fallback={
            <UserSettingsDialogSkeleton open={settingsOpen} onOpenChange={setSettingsOpen} />
          }
        >
          <UserSettingsDialog open={settingsOpen} onOpenChange={setSettingsOpen} />
        </Suspense>
      )}
      <DropdownMenu>
        {compact ? (
          <DropdownMenuTrigger
            render={
              <button
                type="button"
                aria-label={`Account menu for ${displayName}`}
                className="hover:ring-ring/40 inline-flex rounded-full transition-shadow hover:ring-2 data-popup-open:ring-2"
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
          </DropdownMenuTrigger>
        ) : (
          <DropdownMenuTrigger
            render={
              <button
                type="button"
                className="hover:bg-accent/50 flex w-full items-center gap-2 rounded-md p-1.5 transition-colors"
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
              <span className="text-2xs text-muted-foreground truncate">{user?.emailAddress}</span>
            </span>
            <ChevronsUpDown className="text-muted-foreground size-3.5 shrink-0" />
          </DropdownMenuTrigger>
        )}
        <DropdownMenuContent
          side={compact ? "bottom" : "right"}
          align="end"
          sideOffset={8}
          className="min-w-56 rounded-lg"
        >
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
                  <span className="text-foreground truncate font-semibold">{user?.name}</span>
                  <span className="text-muted-foreground truncate text-xs">
                    {user?.emailAddress}
                  </span>
                </div>
              </div>
            </DropdownMenuLabel>
            <DropdownMenuSeparator />
            <DropdownMenuItem
              title={t("Profile")}
              startContent={<User className="size-4" />}
              onClick={() => void navigate("/profile")}
            />
            <DropdownMenuItem
              title={t("Settings")}
              startContent={<Settings className="size-4" />}
              onClick={() => {
                setSettingsMounted(true);
                setSettingsOpen(true);
              }}
            />
            <SidebarLayoutSubmenu />
            <DropdownMenuSub>
              <DropdownMenuSubTrigger>
                <Palette className="mr-2 size-4" />
                <span>{t("Switch Theme")}</span>
              </DropdownMenuSubTrigger>
              <DropdownMenuPortal>
                <DropdownMenuSubContent sideOffset={5}>
                  <DropdownMenuCheckboxItem
                    checked={theme === "light"}
                    onCheckedChange={() => setTheme("light")}
                    className="cursor-pointer"
                  >
                    {t("Light")}
                  </DropdownMenuCheckboxItem>
                  <DropdownMenuCheckboxItem
                    checked={theme === "dark"}
                    onCheckedChange={() => setTheme("dark")}
                    className="cursor-pointer"
                  >
                    {t("Dark")}
                  </DropdownMenuCheckboxItem>
                  <DropdownMenuCheckboxItem
                    checked={theme === "system"}
                    onCheckedChange={() => setTheme("system")}
                    className="cursor-pointer"
                  >
                    {t("System")}
                  </DropdownMenuCheckboxItem>
                </DropdownMenuSubContent>
              </DropdownMenuPortal>
            </DropdownMenuSub>
            <DropdownMenuSeparator />
            <DropdownMenuItem
              title={t("Log out")}
              startContent={<LogOut className="size-4" />}
              onClick={() => void handleLogout()}
            />
          </DropdownMenuGroup>
        </DropdownMenuContent>
      </DropdownMenu>
    </>
  );
}
