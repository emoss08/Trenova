import { LazyImage } from "@/components/image";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuGroup,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuTrigger,
} from "@trenova/shared/components/ui/dropdown-menu";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { useSwitchOrganization } from "@/hooks/use-organization-switch";
import { queries } from "@/lib/queries";
import { cn } from "@trenova/shared/lib/utils";
import type { UserOrganization } from "@trenova/shared/types/organization";
import { useQuery } from "@tanstack/react-query";
import { Check, ChevronsUpDown, Loader2 } from "lucide-react";
import { useState } from "react";

function OrgLogo({
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

  if (isSwitching) {
    return (
      <div className="flex size-7 shrink-0 items-center justify-center rounded-md bg-primary text-primary-foreground">
        <Loader2 className="size-3 animate-spin" />
      </div>
    );
  }

  if (hasLogo) {
    return (
      <LazyImage
        src={logoURL}
        alt={`${orgName ?? "Organization"} logo`}
        className="size-7 shrink-0 rounded-md object-cover"
        onError={() => setImageLoadFailed(true)}
      />
    );
  }

  return (
    <div className="flex size-7 shrink-0 items-center justify-center rounded-md bg-primary text-[10px] font-bold text-primary-foreground">
      {initials}
    </div>
  );
}

export function OrgSwitcher() {
  const { data: organizations, isLoading } = useQuery(queries.userOrganization.all());
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
    currentOrgLogoUrl.startsWith("http://") || currentOrgLogoUrl.startsWith("https://");
  const shouldResolveLogo =
    Boolean(currentOrg?.id) && Boolean(currentOrgLogoUrl) && !hasAbsoluteLogoURL;

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
      <div className="flex items-center gap-2 px-1 py-1">
        <Skeleton className="size-7 rounded-md" />
        <Skeleton className="h-4 flex-1 rounded-md" />
      </div>
    );
  }

  const rowContent = (
    <>
      <OrgLogo
        key={logoURL ?? "no-logo"}
        logoURL={logoURL}
        orgName={currentOrg?.name}
        initials={orgInitials}
        isSwitching={switchMutation.isPending}
      />
      <span className="min-w-0 flex-1 truncate text-left text-sm font-semibold">
        {currentOrg?.name ?? "Trenova"}
      </span>
    </>
  );

  if (!hasMultipleOrgs) {
    return <div className="flex w-full items-center gap-2 px-1 py-1 select-none">{rowContent}</div>;
  }

  return (
    <DropdownMenu>
      <DropdownMenuTrigger
        render={
          <button
            type="button"
            disabled={switchMutation.isPending}
            className="flex w-full items-center gap-2 rounded-md px-1 py-1 transition-colors select-none hover:bg-accent/50"
          />
        }
      >
        {rowContent}
        <ChevronsUpDown className="size-3.5 shrink-0 text-muted-foreground" />
      </DropdownMenuTrigger>
      <DropdownMenuContent side="bottom" align="start" sideOffset={6} className="w-60">
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
              endContent={org.isCurrent ? <Check className="size-4 text-primary" /> : undefined}
            />
          ))}
        </DropdownMenuGroup>
      </DropdownMenuContent>
    </DropdownMenu>
  );
}
