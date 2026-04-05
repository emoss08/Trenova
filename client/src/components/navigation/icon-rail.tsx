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
  Tooltip,
  TooltipContent,
  TooltipTrigger,
} from "@/components/ui/tooltip";
import { LazyImage } from "@/components/image";
import type { ModuleId, NavModule } from "@/config/navigation.types";
import { useSwitchOrganization } from "@/hooks/use-organization-switch";
import { queries } from "@/lib/queries";
import { cn } from "@/lib/utils";
import type { UserOrganization } from "@/types/organization";
import { useAuthStore } from "@/stores/auth-store";
import { useCommandPaletteStore } from "@/stores/command-palette-store";
import { useQuery } from "@tanstack/react-query";
import { Check, Loader2, LogOut, Palette, Search, Settings, Star, User } from "lucide-react";
import { useEffect, useState } from "react";
import { useNavigate } from "react-router";
import { useTheme } from "@/components/theme-provider";
import { UserSettingsDialog } from "@/components/navigation/user-settings-dialog";

type IconRailProps = {
  modules: NavModule[];
  activeModuleId: ModuleId | null;
  onModuleSelect: (id: ModuleId) => void;
  isFavoritesActive: boolean;
  onFavoritesSelect: () => void;
};

function OrgAvatar() {
  const { data: organizations, isLoading } = useQuery(
    queries.userOrganization.all(),
  );
  const switchMutation = useSwitchOrganization();

  const currentOrg = organizations?.find((org) => org.isCurrent);
  const hasMultipleOrgs = (organizations?.length ?? 0) > 1;

  const orgInitials = currentOrg?.name
    ? currentOrg.name
        .split(" ")
        .map((w: string) => w[0])
        .join("")
        .toUpperCase()
        .slice(0, 2)
    : "T";

  const currentOrgLogoUrl = currentOrg?.logoUrl ?? "";
  const hasAbsoluteLogoURL =
    currentOrgLogoUrl.startsWith("http://") ||
    currentOrgLogoUrl.startsWith("https://");
  const shouldResolveLogo =
    Boolean(currentOrg?.id) &&
    Boolean(currentOrgLogoUrl) &&
    !hasAbsoluteLogoURL;

  const { data: resolvedLogoURL } = useQuery({
    ...queries.organization.logo(currentOrg?.id ?? ""),
    enabled: shouldResolveLogo,
    retry: false,
  });

  const logoURL = hasAbsoluteLogoURL ? currentOrgLogoUrl : resolvedLogoURL;

  const handleSwitch = (org: UserOrganization) => {
    if (org.isCurrent || switchMutation.isPending) return;
    switchMutation.mutate({ organizationId: org.id });
  };

  if (isLoading) {
    return (
      <div className="flex size-8 items-center justify-center rounded-md bg-primary text-primary-foreground">
        <Loader2 className="size-3 animate-spin" />
      </div>
    );
  }

  const avatarContent = (
    <OrgAvatarContent
      logoURL={logoURL}
      orgName={currentOrg?.name}
      initials={orgInitials}
      isSwitching={switchMutation.isPending}
    />
  );

  if (!hasMultipleOrgs) {
    return (
      <Tooltip>
        <TooltipTrigger
          render={
            <a href="/" className="flex size-8 shrink-0 items-center justify-center rounded-md overflow-hidden select-none" />
          }
        >
          {avatarContent}
        </TooltipTrigger>
        <TooltipContent side="right" sideOffset={6}>
          {currentOrg?.name ?? "Home"}
        </TooltipContent>
      </Tooltip>
    );
  }

  return (
    <DropdownMenu>
      <Tooltip>
        <TooltipTrigger
          render={
            <DropdownMenuTrigger
              render={
                <button
                  type="button"
                  className="flex size-8 shrink-0 items-center justify-center rounded-md overflow-hidden select-none"
                  disabled={switchMutation.isPending}
                />
              }
            />
          }
        >
          {avatarContent}
        </TooltipTrigger>
        <TooltipContent side="right" sideOffset={6}>
          {currentOrg?.name ?? "Switch organization"}
        </TooltipContent>
      </Tooltip>
      <DropdownMenuContent
        side="right"
        align="start"
        sideOffset={8}
        className="w-56"
      >
        <DropdownMenuGroup>
          <DropdownMenuLabel>Switch Organization</DropdownMenuLabel>
          {organizations?.map((org) => (
            <DropdownMenuItem
              key={org.id}
              title={org.name}
              description={org.isDefault ? "Default" : undefined}
              onClick={() => handleSwitch(org)}
              disabled={switchMutation.isPending || org.isCurrent}
              className={cn(org.isCurrent && "bg-accent")}
              endContent={
                org.isCurrent ? (
                  <Check className="size-4 text-primary" />
                ) : undefined
              }
            />
          ))}
        </DropdownMenuGroup>
      </DropdownMenuContent>
    </DropdownMenu>
  );
}

function OrgAvatarContent({
  logoURL,
  orgName,
  initials,
  isSwitching,
}: {
  logoURL?: string;
  orgName?: string;
  initials: string;
  isSwitching: boolean;
}) {
  const [imageLoadFailed, setImageLoadFailed] = useState(false);
  const hasLogo = Boolean(logoURL) && !imageLoadFailed;

  useEffect(() => {
    setImageLoadFailed(false);
  }, [logoURL]);

  if (isSwitching) {
    return (
      <div className="flex size-full items-center justify-center rounded-md bg-primary text-primary-foreground">
        <Loader2 className="size-3 animate-spin" />
      </div>
    );
  }

  if (hasLogo) {
    return (
      <LazyImage
        src={logoURL}
        alt={`${orgName ?? "Organization"} logo`}
        className="size-full rounded-md object-cover"
        onError={() => setImageLoadFailed(true)}
      />
    );
  }

  return (
    <div className="flex size-full items-center justify-center rounded-md bg-primary text-[10px] font-bold text-primary-foreground">
      {initials}
    </div>
  );
}

function ModuleButton({
  module,
  isActive,
  onSelect,
}: {
  module: NavModule;
  isActive: boolean;
  onSelect: () => void;
}) {
  const Icon = module.icon;

  return (
    <Tooltip>
      <TooltipTrigger
        render={
          <button
            type="button"
            onClick={onSelect}
            className={cn(
              "flex size-8 items-center justify-center rounded-md transition-colors",
              isActive
                ? "bg-accent text-foreground"
                : "text-muted-foreground/60 hover:text-muted-foreground hover:bg-accent/50",
            )}
          />
        }
      >
        <Icon className="size-[18px]" strokeWidth={isActive ? 2 : 1.5} />
      </TooltipTrigger>
      <TooltipContent side="right" sideOffset={6}>
        {module.label}
      </TooltipContent>
    </Tooltip>
  );
}

function FavoritesButton({ isActive, onSelect }: { isActive: boolean; onSelect: () => void }) {
  return (
    <Tooltip>
      <TooltipTrigger
        render={
          <button
            type="button"
            onClick={onSelect}
            className={cn(
              "flex size-8 items-center justify-center rounded-md transition-colors",
              isActive
                ? "bg-accent text-foreground"
                : "text-muted-foreground/60 hover:text-muted-foreground hover:bg-accent/50",
            )}
          />
        }
      >
        <Star className={cn("size-4", isActive && "fill-amber-400 text-amber-400")} strokeWidth={isActive ? 2 : 1.5} />
      </TooltipTrigger>
      <TooltipContent side="right" sideOffset={6}>
        Favorites
      </TooltipContent>
    </Tooltip>
  );
}

function SearchButton() {
  const setOpen = useCommandPaletteStore((s) => s.setOpen);

  return (
    <Tooltip>
      <TooltipTrigger
        render={
          <button
            type="button"
            onClick={() => setOpen(true)}
            className="flex size-8 items-center justify-center rounded-md text-muted-foreground/60 transition-colors hover:bg-accent/50 hover:text-muted-foreground"
          />
        }
      >
        <Search className="size-4" strokeWidth={1.5} />
      </TooltipTrigger>
      <TooltipContent side="right" sideOffset={6}>
        Search
      </TooltipContent>
    </Tooltip>
  );
}

function UserMenu() {
  const user = useAuthStore((s) => s.user);
  const logout = useAuthStore((s) => s.logout);
  const navigate = useNavigate();
  const { theme, setTheme } = useTheme();
  const [settingsOpen, setSettingsOpen] = useState(false);

  const displayName = user?.name ?? user?.username ?? "User";
  const initials = displayName
    .split(" ")
    .map((w: string) => w[0])
    .join("")
    .toUpperCase()
    .slice(0, 2);

  const handleLogout = async () => {
    await logout();
    void navigate("/login");
  };

  return (
    <>
      <UserSettingsDialog open={settingsOpen} onOpenChange={setSettingsOpen} />
      <DropdownMenu>
        <Tooltip>
          <TooltipTrigger
            render={
              <DropdownMenuTrigger
                render={
                  <button
                    type="button"
                    className="flex size-8 items-center justify-center rounded-full bg-muted text-[10px] font-medium text-muted-foreground transition-colors hover:bg-accent"
                  />
                }
              />
            }
          >
            {initials}
          </TooltipTrigger>
          <TooltipContent side="right" sideOffset={6}>
            {displayName}
          </TooltipContent>
        </Tooltip>
        <DropdownMenuContent
          side="right"
          align="end"
          sideOffset={8}
          className="min-w-56 rounded-lg"
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
    </>
  );
}

export function IconRail({
  modules,
  activeModuleId,
  onModuleSelect,
  isFavoritesActive,
  onFavoritesSelect,
}: IconRailProps) {
  return (
    <div className="flex h-screen w-12 flex-col items-center border-r border-border bg-sidebar">
      {/* Organization logo */}
      <div className="flex h-12 items-center justify-center">
        <OrgAvatar />
      </div>

      {/* Module icons */}
      <nav className="flex flex-1 flex-col items-center gap-0.5 px-1.5 pt-1">
        {modules.map((mod) => (
          <ModuleButton
            key={mod.id}
            module={mod}
            isActive={activeModuleId === mod.id}
            onSelect={() => onModuleSelect(mod.id)}
          />
        ))}
      </nav>

      {/* Bottom utilities */}
      <div className="flex flex-col items-center gap-1 px-1.5 pb-3">
        <FavoritesButton isActive={isFavoritesActive} onSelect={onFavoritesSelect} />
        <SearchButton />
        <div className="my-1 h-px w-5 bg-border" />
        <UserMenu />
      </div>
    </div>
  );
}
