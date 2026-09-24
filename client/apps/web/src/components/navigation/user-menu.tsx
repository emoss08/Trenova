import { useT } from "@trenova/shared/i18n/use-t";
import { LanguageSubmenu } from "@/components/navigation/language-submenu";
import { SidebarLayoutSubmenu } from "@/components/navigation/sidebar-variant-menu";
import { ResolvedUserAvatar } from "@/components/resolved-user-avatar";
import { useTheme } from "@trenova/shared/components/theme-provider";
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
import { useSignOut } from "@/hooks/use-sign-out";
import { useAppDialogsStore } from "@/stores/app-dialogs-store";
import { useAuthStore } from "@trenova/shared/stores/auth-store";
import { ChevronsUpDown, LogOut, Palette, Settings, User } from "lucide-react";
import { useNavigate } from "react-router";

/**
 * `compact` renders the avatar alone as the trigger, for a rail with no room
 * for a name; the full menu is unchanged.
 */
export function UserMenu({ compact = false }: { compact?: boolean }) {
  const t = useT();

  const user = useAuthStore((s) => s.user);
  const navigate = useNavigate();
  const signOut = useSignOut();
  const openDialog = useAppDialogsStore((state) => state.openDialog);
  const { theme, setTheme } = useTheme();

  const displayName = user?.name ?? user?.username ?? "User";

  return (
    <>
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
              fallbackClassName="bg-muted text-2xs font-medium text-muted-foreground"
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
              fallbackClassName="bg-muted text-2xs font-medium text-muted-foreground"
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
                  fallbackClassName="rounded-lg bg-sidebar-accent text-xs font-semibold text-sidebar-accent-foreground"
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
              onClick={() => openDialog("settings")}
            />
            <SidebarLayoutSubmenu />
            <LanguageSubmenu />
            <DropdownMenuSub>
              <DropdownMenuSubTrigger>
                <Palette className="mr-2 size-4" />
                <span>{t("Switch theme")}</span>
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
              onClick={() => void signOut()}
            />
          </DropdownMenuGroup>
        </DropdownMenuContent>
      </DropdownMenu>
    </>
  );
}
