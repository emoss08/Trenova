import {
  SidebarMenu,
  SidebarMenuButton,
  SidebarMenuItem,
} from "@/components/ui/sidebar";
import { queries } from "@/lib/queries";
import type { OrganizationSchema } from "@/lib/schemas/organization-schema";
import { useIsAuthenticated, useUser } from "@/stores/user-store";
import { faCheckCircle } from "@fortawesome/pro-solid-svg-icons";
import { CaretSortIcon, DragHandleDots2Icon } from "@radix-ui/react-icons";
import { useQuery } from "@tanstack/react-query";
import { useState } from "react";
import { Avatar, AvatarFallback, AvatarImage } from "./ui/avatar";
import { Icon } from "./ui/icons";
import { Popover, PopoverContent, PopoverTrigger } from "./ui/popover";
import { Separator } from "./ui/separator";
import { Skeleton } from "./ui/skeleton";

function OrganizationSwitcherButtonSkeleton() {
  return (
    <div className="flex w-full items-center justify-between gap-2 p-1">
      <Skeleton className="size-10" />
      <div className="flex w-[150px] flex-col gap-0.5">
        <Skeleton className="h-4" />
        <Skeleton className="h-2 w-[120px]" />
      </div>
      <CaretSortIcon className="ml-auto size-5" />
    </div>
  );
}

type OrganizationSwitcherButtonProps = {
  org: OrganizationSchema | undefined;
  setOpen: (open: boolean) => void;
  isLoading: boolean;
};

function OrganizationSwitcherButton({
  org,
  setOpen,
  isLoading,
}: OrganizationSwitcherButtonProps) {
  return isLoading ? (
    <OrganizationSwitcherButtonSkeleton />
  ) : (
    <SidebarMenuButton
      size="lg"
      onClick={() => setOpen(true)}
      className="bg-sidebar data-[state=open]:bg-sidebar-accent data-[state=open]:text-sidebar-accent-foreground [&>svg]:size-5"
    >
      <Avatar className="size-8 items-center rounded-lg">
        <AvatarImage
          alt={org?.name}
          className="size-8 object-contain"
          src={org?.logoUrl}
        />
        <AvatarFallback>{org?.name.slice(0, 2).toUpperCase()}</AvatarFallback>
      </Avatar>
      <div className="flex flex-col gap-0.5 truncate leading-none">
        <span className="max-w-[200px] truncate font-semibold">
          {org?.name}
        </span>
        <span className="w-[150px] truncate text-2xs text-muted-foreground">
          {org?.city}, {org?.state?.abbreviation}
        </span>
      </div>
      <CaretSortIcon className="ml-auto" />
    </SidebarMenuButton>
  );
}

export function OrganizationSwitcher() {
  const [open, setOpen] = useState(false);
  const user = useUser();
  const isAuthenticated = useIsAuthenticated();
  const userOrganization = useQuery({
    ...queries.organization.getOrgById(
      user?.currentOrganizationId ?? "",
      true,
      false,
    ),
    enabled: !!user?.currentOrganizationId && isAuthenticated,
  });

  const userOrganizations = useQuery({
    ...queries.organization.getUserOrganizations(),
    enabled: isAuthenticated,
  });

  const org = userOrganization.data;
  const organizations = userOrganizations.data;

  const isLoading = userOrganization.isLoading || userOrganizations.isLoading;

  return (
    <SidebarMenu>
      <SidebarMenuItem>
        <Popover open={open} onOpenChange={setOpen}>
          <PopoverTrigger asChild>
            <OrganizationSwitcherButton
              org={org}
              setOpen={() => setOpen(!open)}
              isLoading={isLoading}
            />
          </PopoverTrigger>
          <PopoverContent align="start" className="w-[300px] p-1">
            <span className="ml-1.5 select-none text-xs text-foreground">
              Switch Organization
            </span>
            <Separator className="my-1" />
            <div className="flex flex-col gap-2">
              {organizations?.results.map((org) => (
                <OrganizationContent
                  key={org.id}
                  org={org}
                  currentOrgId={user?.currentOrganizationId ?? ""}
                />
              ))}
            </div>
          </PopoverContent>
        </Popover>
      </SidebarMenuItem>
    </SidebarMenu>
  );
}

function OrganizationContent({
  org,
  currentOrgId,
}: {
  org: OrganizationSchema;
  currentOrgId: string;
}) {
  return (
    <div className="flex items-center justify-between">
      <button className="flex w-full items-center gap-1 rounded-md p-1 hover:bg-muted">
        <DragHandleDots2Icon className="size-4" />
        <Avatar className="size-8 rounded-lg">
          <AvatarImage src={org.logoUrl} />
          <AvatarFallback className="text-xs">
            {org.name.slice(0, 2).toUpperCase()}
          </AvatarFallback>
        </Avatar>
        <div className="grid flex-1 text-left leading-tight">
          <span className="truncate text-xs">{org.name}</span>
          <span className="text-2xs text-muted-foreground">
            {org.addressLine1}, {org.city}, {org.state?.abbreviation}{" "}
            {org.postalCode}
          </span>
        </div>
        {currentOrgId === org.id && (
          <Icon
            icon={faCheckCircle}
            className="ml-auto size-4 pr-2 text-primary"
          />
        )}
      </button>
    </div>
  );
}
