import { LazyImage } from "@/components/image";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuGroup,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import {
  SidebarMenu,
  SidebarMenuButton,
  SidebarMenuItem,
  useSidebar,
} from "@/components/ui/sidebar";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import { useSwitchOrganization } from "@/hooks/use-organization-switch";
import { queries } from "@/lib/queries";
import { cn } from "@/lib/utils";
import type { UserOrganization } from "@/types/organization";
import { useQuery } from "@tanstack/react-query";
import { Check, ChevronsUpDown, Loader2 } from "lucide-react";
import * as React from "react";

type OrganizationButtonContentProps = {
  currentOrg?: UserOrganization;
  orgInitials: string;
  logoUrl?: string;
  isSwitching?: boolean;
};

function OrganizationButtonContent({
  currentOrg,
  orgInitials,
  logoUrl,
  isSwitching = false,
}: OrganizationButtonContentProps) {
  const [imageLoadFailed, setImageLoadFailed] = React.useState(false);
  const hasLogo = Boolean(logoUrl) && !imageLoadFailed;

  React.useEffect(() => {
    setImageLoadFailed(false);
  }, [logoUrl]);

  return (
    <>
      <div className="relative flex size-8 shrink-0 items-center justify-center overflow-hidden rounded-md select-none after:absolute after:inset-0 after:rounded-md">
        {isSwitching ? (
          <div className="flex size-full items-center justify-center rounded-md bg-linear-to-br from-primary to-primary/80 text-primary-foreground">
            <Loader2 className="size-3 animate-spin" />
          </div>
        ) : hasLogo ? (
          <LazyImage
            src={logoUrl}
            alt={`${currentOrg?.name ?? "Organization"} logo`}
            className="aspect-square size-full rounded-md object-cover"
            onError={() => setImageLoadFailed(true)}
          />
        ) : (
          <div className="flex size-full items-center justify-center rounded-md bg-linear-to-br from-primary to-primary/80 text-xs font-bold text-primary-foreground">
            {orgInitials}
          </div>
        )}
      </div>
      <div className="grid flex-1 text-left text-sm leading-tight">
        <span className="truncate font-semibold">{currentOrg?.name ?? "Organization"}</span>
        <span className="text-2xs text-muted-foreground">
          {currentOrg?.city}, {currentOrg?.state}
        </span>
      </div>
    </>
  );
}

export function OrganizationSwitcher() {
  const { isMobile } = useSidebar();
  const { data: organizations, isLoading: isLoadingOrgs } = useQuery(
    queries.userOrganization.all(),
  );

  const switchMutation = useSwitchOrganization();

  const currentOrg = organizations?.find((org) => org.isCurrent);
  const orgInitials = currentOrg?.name
    ? currentOrg.name
        .split(" ")
        .map((word) => word[0])
        .join("")
        .toUpperCase()
        .slice(0, 2)
    : "T";

  const handleSwitch = (org: UserOrganization) => {
    if (org.isCurrent || switchMutation.isPending) return;
    switchMutation.mutate({ organizationId: org.id });
  };

  const currentOrgLogoUrl = currentOrg?.logoUrl ?? "";
  const hasAbsoluteLogoURL =
    currentOrgLogoUrl.startsWith("http://") || currentOrgLogoUrl.startsWith("https://");
  const shouldResolveLogo =
    Boolean(currentOrg?.id) && Boolean(currentOrgLogoUrl) && !hasAbsoluteLogoURL;

  const { data: resolvedLogoURL } = useQuery({
    ...queries.organization.logo(currentOrg?.id ?? ""),
    enabled: shouldResolveLogo,
    retry: false,
  });

  const logoURL = hasAbsoluteLogoURL ? currentOrgLogoUrl : resolvedLogoURL;

  if (isLoadingOrgs) {
    return (
      <SidebarMenu>
        <SidebarMenuItem>
          <SidebarMenuButton size="lg" disabled>
            <div className="flex size-8 items-center justify-center rounded-lg bg-gradient-to-br from-primary to-primary/80 text-primary-foreground">
              <Loader2 className="size-4 animate-spin" />
            </div>
            <div className="grid flex-1 text-left text-sm leading-tight">
              <span className="truncate font-semibold">Loading...</span>
            </div>
          </SidebarMenuButton>
        </SidebarMenuItem>
      </SidebarMenu>
    );
  }

  if (!organizations || organizations.length <= 1) {
    return (
      <SidebarMenu>
        <SidebarMenuItem>
          <SidebarMenuButton size="lg" className="cursor-default">
            <OrganizationButtonContent
              currentOrg={currentOrg}
              orgInitials={orgInitials}
              logoUrl={logoURL}
            />
          </SidebarMenuButton>
        </SidebarMenuItem>
      </SidebarMenu>
    );
  }

  return (
    <SidebarMenu>
      <SidebarMenuItem>
        <DropdownMenu>
          <div className="flex items-center gap-1">
            <SidebarMenuButton
              size="lg"
              className="cursor-default hover:bg-transparent bg-transparent data-[state=open]:bg-transparent"
              disabled={switchMutation.isPending}
            >
              <OrganizationButtonContent
                currentOrg={currentOrg}
                orgInitials={orgInitials}
                logoUrl={logoURL}
                isSwitching={switchMutation.isPending}
              />
            </SidebarMenuButton>
            <Tooltip>
              <TooltipTrigger
                render={
                  <DropdownMenuTrigger
                    render={
                      <SidebarMenuButton
                        size="lg"
                        className="w-10 flex-none justify-center p-0 data-[state=open]:bg-sidebar-accent data-[state=open]:text-sidebar-accent-foreground"
                        disabled={switchMutation.isPending}
                        aria-label="Switch organizations"
                      >
                        <ChevronsUpDown className="size-4" />
                      </SidebarMenuButton>
                    }
                  />
                }
              />
              <TooltipContent side="top">Switch organizations</TooltipContent>
            </Tooltip>
          </div>
          <DropdownMenuContent
            className="w-(--anchor-width) min-w-56 rounded-lg"
            side={isMobile ? "bottom" : "right"}
            align="start"
            sideOffset={4}
          >
            <DropdownMenuGroup className="space-y-1">
              <DropdownMenuLabel className="flex items-center gap-2 text-xs text-muted-foreground">
                <span>Switch Organization</span>
              </DropdownMenuLabel>
              <DropdownMenuSeparator />
              {organizations.map((org) => (
                <DropdownMenuItem
                  key={org.id}
                  onClick={() => handleSwitch(org)}
                  disabled={switchMutation.isPending || org.isCurrent}
                  className={cn("cursor-pointer", org.isCurrent && "bg-accent")}
                  title={org.name}
                  description={org.isDefault ? "Default" : undefined}
                  endContent={org.isCurrent ? <Check className="size-4 text-primary" /> : undefined}
                />
              ))}
            </DropdownMenuGroup>
          </DropdownMenuContent>
        </DropdownMenu>
      </SidebarMenuItem>
    </SidebarMenu>
  );
}
